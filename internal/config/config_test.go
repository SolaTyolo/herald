package config_test

import (
	"os"
	"testing"

	"github.com/SolaTyolo/herald/internal/config"
)

func TestLoadSMTPFromEnv(t *testing.T) {
	os.Setenv("API_KEY", "test-key")
	os.Setenv("DEV_MODE", "true")
	os.Setenv("SMTP_HOST", "smtp.example.com")
	os.Setenv("SMTP_FROM", "noreply@example.com")
	os.Setenv("SMTP_PORT", "465")
	os.Setenv("SMTP_USERNAME", "user")
	os.Setenv("SMTP_PASSWORD", "pass")
	defer func() {
		os.Unsetenv("API_KEY")
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
	if cfg.APIKey != "test-key" {
		t.Fatalf("API_KEY not loaded: %+v", cfg)
	}
	if cfg.SMTPHost != "smtp.example.com" || cfg.SMTPFrom != "noreply@example.com" {
		t.Fatalf("smtp env not loaded: %+v", cfg)
	}
	if cfg.SMTPPort != "465" || cfg.SMTPUsername != "user" || cfg.SMTPPassword != "pass" {
		t.Fatalf("smtp auth env not loaded: %+v", cfg)
	}
}

func TestLoadDefaultSMTPPort(t *testing.T) {
	os.Setenv("API_KEY", "test-key")
	os.Setenv("DEV_MODE", "true")
	os.Unsetenv("SMTP_PORT")
	defer func() {
		os.Unsetenv("API_KEY")
		os.Unsetenv("DEV_MODE")
	}()

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SMTPPort != "587" {
		t.Fatalf("expected default SMTP_PORT 587, got %q", cfg.SMTPPort)
	}
}

func TestLoadRequiresAPIKey(t *testing.T) {
	prev, had := os.LookupEnv("API_KEY")
	os.Unsetenv("API_KEY")
	defer func() {
		if had {
			os.Setenv("API_KEY", prev)
		}
	}()

	if _, err := config.Load(); err == nil {
		t.Fatal("expected error when API_KEY is missing")
	}
}

func TestLoadRedisAuth(t *testing.T) {
	os.Setenv("API_KEY", "test-key")
	os.Setenv("QUEUE_MODE", "asynq")
	os.Setenv("REDIS_ADDR", "redis:6379")
	os.Setenv("REDIS_PASSWORD", "s3cret")
	os.Setenv("REDIS_DB", "2")
	defer func() {
		os.Unsetenv("API_KEY")
		os.Unsetenv("QUEUE_MODE")
		os.Unsetenv("REDIS_ADDR")
		os.Unsetenv("REDIS_PASSWORD")
		os.Unsetenv("REDIS_DB")
	}()

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RedisAddr != "redis:6379" || cfg.RedisPassword != "s3cret" || cfg.RedisDB != 2 {
		t.Fatalf("redis auth env not loaded: addr=%q password=%q db=%d", cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	}
	opt := cfg.RedisOptions()
	if opt.Addr != "redis:6379" || opt.Password != "s3cret" || opt.DB != 2 {
		t.Fatalf("RedisOptions mismatch: %+v", opt)
	}
	asynqOpt := cfg.AsynqRedisOpt()
	if asynqOpt.Addr != "redis:6379" || asynqOpt.Password != "s3cret" || asynqOpt.DB != 2 {
		t.Fatalf("AsynqRedisOpt mismatch: %+v", asynqOpt)
	}
}

func TestLoadRedisAuthIgnoredWhenLocal(t *testing.T) {
	os.Setenv("API_KEY", "test-key")
	os.Setenv("QUEUE_MODE", "local")
	os.Setenv("REDIS_PASSWORD", "should-ignore")
	os.Setenv("REDIS_DB", "5")
	defer func() {
		os.Unsetenv("API_KEY")
		os.Unsetenv("QUEUE_MODE")
		os.Unsetenv("REDIS_PASSWORD")
		os.Unsetenv("REDIS_DB")
	}()

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RedisAddr != "" || cfg.RedisPassword != "" || cfg.RedisDB != 0 {
		t.Fatalf("expected empty redis settings in local mode: %+v", cfg)
	}
}
