package delivery

import (
	"errors"
	"testing"
)

func TestRetryableClassification(t *testing.T) {
	t.Parallel()
	// Retryable errors surface to Asynq for the same (pinned) provider — there is
	// no same-channel provider failover loop anymore.
	err := &RetryableError{Code: "net", Message: "timeout"}
	if !IsRetryable(err) {
		t.Fatal("expected retryable")
	}
	if IsRetryable(errors.New("delivery failed (smtp_auth): bad creds")) {
		t.Fatal("permanent must not be retryable")
	}
}
