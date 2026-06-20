package delivery

import (
	"encoding/json"
)

// SendContext carries workflow/notification metadata into WASM host functions and Send JSON.
type SendContext struct {
	TransactionID  string
	WorkflowID     string
	NotificationID string
	Payload        json.RawMessage
}
