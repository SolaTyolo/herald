package delivery

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsRetryable(t *testing.T) {
	t.Parallel()
	if IsRetryable(nil) {
		t.Fatal("nil")
	}
	if IsRetryable(fmt.Errorf("permanent")) {
		t.Fatal("plain error")
	}
	re := &RetryableError{Code: "x", Message: "y"}
	if !IsRetryable(re) {
		t.Fatal("direct")
	}
	if !IsRetryable(fmt.Errorf("wrap: %w", re)) {
		t.Fatal("wrapped")
	}
	var target *RetryableError
	if !errors.As(fmt.Errorf("wrap: %w", re), &target) {
		t.Fatal("errors.As")
	}
}
