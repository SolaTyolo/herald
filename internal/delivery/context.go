package delivery

import (
	"encoding/json"
)

// SendContext carries workflow/notification metadata into delivery calls.
type SendContext struct {
	TransactionID  string
	WorkflowID     string
	NotificationID string
	Payload        json.RawMessage
}
