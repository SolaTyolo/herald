package config_test

import (
	"os"
	"testing"

	"github.com/SolaTyolo/herald/internal/config"
)

func TestRejectsDefaultEncryptionKeyWithoutDevMode(t *testing.T) {
	os.Unsetenv("DEV_MODE")
	os.Unsetenv("ENCRYPTION_KEY")
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for default encryption key")
	}
}

func TestAllowsDefaultEncryptionKeyInDevMode(t *testing.T) {
	os.Setenv("DEV_MODE", "true")
	os.Unsetenv("ENCRYPTION_KEY")
	defer os.Unsetenv("DEV_MODE")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.DevMode {
		t.Fatal("expected dev mode")
	}
}
