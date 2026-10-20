package delivery

import (
	"errors"
	"testing"
)

func TestRetryableStopsFailoverSemantics(t *testing.T) {
	t.Parallel()
	// SendChannel returns immediately on RetryableError so Asynq retries the same
	// provider instead of failing over to another YAML provider for the channel.
	err := &RetryableError{Code: "net", Message: "timeout"}
	if !IsRetryable(err) {
		t.Fatal("expected retryable")
	}
	if IsRetryable(errors.New("delivery failed (smtp_auth): bad creds")) {
		t.Fatal("permanent must not be retryable")
	}
}
