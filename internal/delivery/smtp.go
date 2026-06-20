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

	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
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

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
		if err := client.StartTLS(tlsCfg); err != nil {
			return contract.Fail("smtp_tls", err.Error(), true)
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
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(content)
	return []byte(b.String())
}

// ValidateSMTPCredentials checks required fields for builtin smtp_email integrations.
func ValidateSMTPCredentials(raw map[string]any) error {
	host, _ := raw["host"].(string)
	from, _ := raw["from"].(string)
	if strings.TrimSpace(host) == "" {
		return fmt.Errorf("credentials.host required for smtp_email")
	}
	if strings.TrimSpace(from) == "" {
		return fmt.Errorf("credentials.from required for smtp_email")
	}
	if port, ok := raw["port"]; ok {
		switch v := port.(type) {
		case string:
			if _, err := strconv.Atoi(v); err != nil {
				return fmt.Errorf("credentials.port must be an integer")
			}
		case float64:
			// JSON numbers
		case int:
		default:
			return fmt.Errorf("credentials.port must be an integer")
		}
	}
	return nil
}
