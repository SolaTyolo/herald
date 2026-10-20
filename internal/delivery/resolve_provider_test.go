package delivery_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SolaTyolo/herald/internal/delivery"
	"github.com/SolaTyolo/herald/internal/domain"
	"github.com/SolaTyolo/herald/internal/repository/filestore"
	"github.com/SolaTyolo/herald/internal/routing"
)

func TestSendChannelPinnedProvider(t *testing.T) {
	store, err := filestore.Open(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	snap, err := routing.Parse([]byte(`
services:
  w:
    endpoint: "http://127.0.0.1:9/delivery"
providers:
  smtp_email:
    type: builtin
    channel: email
  inapp:
    type: builtin
    channel: inapp
  lark_webhook:
    type: remote
    service: w
    channel: chat
  wecom_webhook:
    type: remote
    service: w
    channel: chat
`))
	if err != nil {
		t.Fatal(err)
	}
	svc := delivery.New(store, routing.NewTable("mem", snap), nil, delivery.Options{})

	_, err = svc.SendChannel(context.Background(), domain.ChannelChat,
		&domain.Subscriber{ID: "s1", ChatCredentials: []byte(`{"default":"https://example.com/hook"}`)},
		&domain.MessageTemplate{Content: "hi"}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "multiple providers") {
		t.Fatalf("expected multiple-providers error, got %v", err)
	}

	_, err = svc.SendChannel(context.Background(), domain.ChannelChat,
		&domain.Subscriber{ID: "s1", ChatCredentials: []byte(`{"default":"https://example.com/hook"}`)},
		&domain.MessageTemplate{Content: "hi"}, nil,
		&delivery.SendContext{ProviderName: "wecom_webhook"})
	// Remote endpoint is unreachable → delivery error, but must pin wecom (not "multiple providers").
	if err == nil {
		t.Fatal("expected remote failure against closed port")
	}
	if strings.Contains(err.Error(), "multiple providers") {
		t.Fatalf("pin ignored: %v", err)
	}
}

func TestSendChannelSoleProviderNoPin(t *testing.T) {
	store, err := filestore.Open(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	snap, err := routing.Parse([]byte(`
providers:
  smtp_email:
    type: builtin
    channel: email
  inapp:
    type: builtin
    channel: inapp
`))
	if err != nil {
		t.Fatal(err)
	}
	svc := delivery.New(store, routing.NewTable("mem", snap), nil, delivery.Options{
		SMTP: delivery.SMTPConfig{Host: "127.0.0.1", Port: "1", From: "a@b.c"},
	})
	_, err = svc.SendChannel(context.Background(), domain.ChannelEmail,
		&domain.Subscriber{ID: "s1", Email: "u@example.com"},
		&domain.MessageTemplate{Content: "hi", Subject: "s"}, nil, nil)
	if err == nil {
		t.Fatal("expected smtp dial failure")
	}
	if strings.Contains(err.Error(), "multiple providers") {
		t.Fatalf("unexpected: %v", err)
	}
}
