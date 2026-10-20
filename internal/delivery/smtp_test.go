package delivery

import "testing"

func TestSMTPConfigValidate(t *testing.T) {
	t.Parallel()
	if err := (SMTPConfig{Host: "smtp.example.com", From: "a@b.com"}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (SMTPConfig{From: "a@b.com"}).Validate(); err == nil {
		t.Fatal("expected host required")
	}
	if err := (SMTPConfig{Host: "smtp.example.com", From: "a@b.com", Port: "bad"}).Validate(); err == nil {
		t.Fatal("expected port error")
	}
}
