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
	// ProviderName pins a YAML provider (WorkflowStep.Provider). Empty = sole
	// provider for the channel, or error if multiple share the channel.
	ProviderName string
}
