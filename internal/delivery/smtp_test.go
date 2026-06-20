package delivery

import (
	"context"
	"testing"

	"github.com/SolaTyolo/herald/pkg/contract"
)

func TestDefaultSMTPSenderMissingConfig(t *testing.T) {
	t.Parallel()
	de := defaultSMTPSender(context.Background(), contract.ProviderConfig{Raw: map[string]string{}}, contract.Message{To: "a@b.com", Content: "x"})
	if de.Success || de.Code != "smtp_config" {
		t.Fatalf("%+v", de)
	}
}

func TestValidateSMTPCredentials(t *testing.T) {
	t.Parallel()
	if err := ValidateSMTPCredentials(map[string]any{"host": "smtp.example.com", "from": "a@b.com"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSMTPCredentials(map[string]any{"from": "a@b.com"}); err == nil {
		t.Fatal("expected host error")
	}
}
