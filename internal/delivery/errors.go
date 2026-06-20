package delivery

import (
	"errors"
	"fmt"
)

// RetryableError signals Asynq (or other schedulers) that the send may be retried.
type RetryableError struct {
	Code    string
	Message string
}

func (e *RetryableError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func IsRetryable(err error) bool {
	var re *RetryableError
	return errors.As(err, &re)
}
