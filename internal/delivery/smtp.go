package delivery

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/SolaTyolo/herald/pkg/contract"
)

// SMTPConfig is builtin smtp_email settings from environment variables.
type SMTPConfig struct {
	Host     string
	Port     string
	From     string
	Username string
	Password string
}

func (c SMTPConfig) Validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return fmt.Errorf("SMTP_HOST required for smtp_email")
	}
	if strings.TrimSpace(c.From) == "" {
		return fmt.Errorf("SMTP_FROM required for smtp_email")
	}
	if c.Port != "" {
		if _, err := strconv.Atoi(c.Port); err != nil {
			return fmt.Errorf("SMTP_PORT must be an integer")
		}
	}
	return nil
}

func (c SMTPConfig) Raw() map[string]string {
	port := c.Port
	if port == "" {
		port = "587"
	}
	return map[string]string{
		"host":     c.Host,
		"port":     port,
		"from":     c.From,
		"username": c.Username,
		"password": c.Password,
	}
}

// SMTPSender sends email via SMTP. Override in tests.
type SMTPSender func(ctx context.Context, cfg contract.ProviderConfig, msg contract.Message) contract.DeliveryError

func defaultSMTPSender(ctx context.Context, cfg contract.ProviderConfig, msg contract.Message) contract.DeliveryError {
	host := cfg.Raw["host"]
	if host == "" {
		return contract.Fail("smtp_config", "raw.host required", false)
	}
	port := cfg.Raw["port"]
	if port == "" {
		port = "587"
	}
	from := cfg.Raw["from"]
	if from == "" {
		return contract.Fail("smtp_config", "raw.from required", false)
	}
	to := strings.TrimSpace(msg.To)
	if to == "" {
		return contract.Fail("smtp_invalid_to", "message.to required", false)
	}

	username := cfg.Raw["username"]
	password := cfg.Raw["password"]
	subject := msg.PushTitle
	if subject == "" {
		if cfg.Raw["subject"] != "" {
			subject = cfg.Raw["subject"]
		} else if msg.Metadata != nil {
			subject = msg.Metadata["subject"]
		}
	}
	if subject == "" {
		subject = "(no subject)"
	}

	timeout := time.Duration(cfg.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if deadline, ok := ctx.Deadline(); ok {
		if rem := time.Until(deadline); rem > 0 && rem < timeout {
			timeout = rem
		}
	}

	addr := net.JoinHostPort(host, port)
	body := buildMIME(from, to, subject, msg.Content)
	tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	netDialer := &net.Dialer{Timeout: timeout}

	var conn net.Conn
	var err error
	// Port 465 is SMTPS (implicit TLS). 587/25 use plain dial + optional STARTTLS.
	if port == "465" {
		d := &tls.Dialer{NetDialer: netDialer, Config: tlsCfg}
		conn, err = d.DialContext(ctx, "tcp", addr)
	} else {
		conn, err = netDialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return contract.Fail("smtp_network", err.Error(), true)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return contract.Fail("smtp_network", err.Error(), true)
	}
	defer func() { _ = client.Close() }()

	if port != "465" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsCfg); err != nil {
				return contract.Fail("smtp_tls", err.Error(), true)
			}
		}
	}

	if username != "" {
		auth := smtp.PlainAuth("", username, password, host)
		if err := client.Auth(auth); err != nil {
			return contract.Fail("smtp_auth", err.Error(), false)
		}
	}
	if err := client.Mail(from); err != nil {
		return contract.Fail("smtp_send", err.Error(), false)
	}
	if err := client.Rcpt(to); err != nil {
		return contract.Fail("smtp_send", err.Error(), false)
	}
	w, err := client.Data()
	if err != nil {
		return contract.Fail("smtp_send", err.Error(), true)
	}
	if _, err := w.Write(body); err != nil {
		_ = w.Close()
		return contract.Fail("smtp_send", err.Error(), true)
	}
	if err := w.Close(); err != nil {
		return contract.Fail("smtp_send", err.Error(), true)
	}
	_ = client.Quit()
	return contract.OK(fmt.Sprintf("smtp:%s:%s", host, port))
}

func buildMIME(from, to, subject, content string) []byte {
	var b strings.Builder
	b.WriteString("From: ")
	b.WriteString(from)
	b.WriteString("\r\nTo: ")
	b.WriteString(to)
	b.WriteString("\r\nSubject: ")
	b.WriteString(subject)
	b.WriteString("\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(content)
	return []byte(b.String())
}
