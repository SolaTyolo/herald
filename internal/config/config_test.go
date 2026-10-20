package config_test

import (
	"os"
	"testing"

	"github.com/SolaTyolo/herald/internal/config"
)

func TestLoadSMTPFromEnv(t *testing.T) {
	os.Setenv("DEV_MODE", "true")
	os.Setenv("SMTP_HOST", "smtp.example.com")
	os.Setenv("SMTP_FROM", "noreply@example.com")
	os.Setenv("SMTP_PORT", "465")
	os.Setenv("SMTP_USERNAME", "user")
	os.Setenv("SMTP_PASSWORD", "pass")
	defer func() {
		os.Unsetenv("DEV_MODE")
		os.Unsetenv("SMTP_HOST")
		os.Unsetenv("SMTP_FROM")
		os.Unsetenv("SMTP_PORT")
		os.Unsetenv("SMTP_USERNAME")
		os.Unsetenv("SMTP_PASSWORD")
	}()

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SMTPHost != "smtp.example.com" || cfg.SMTPFrom != "noreply@example.com" {
		t.Fatalf("smtp env not loaded: %+v", cfg)
	}
	if cfg.SMTPPort != "465" || cfg.SMTPUsername != "user" || cfg.SMTPPassword != "pass" {
		t.Fatalf("smtp auth env not loaded: %+v", cfg)
	}
}

func TestLoadDefaultSMTPPort(t *testing.T) {
	os.Setenv("DEV_MODE", "true")
	os.Unsetenv("SMTP_PORT")
	defer os.Unsetenv("DEV_MODE")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SMTPPort != "587" {
		t.Fatalf("expected default SMTP_PORT 587, got %q", cfg.SMTPPort)
	}
}
