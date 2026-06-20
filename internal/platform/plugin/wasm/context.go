package wasm

import "encoding/json"

// CallContext is injected into each WASM Send/ValidateConfig invocation (JSON + host get_context).
type CallContext struct {
	EnvID          string          `json:"envId"`
	ProviderID     string          `json:"providerId"`
	Channel        string          `json:"channel"`
	SubscriberPK   string          `json:"subscriberPk"`
	WorkflowID     string          `json:"workflowId,omitempty"`
	NotificationID string          `json:"notificationId,omitempty"`
	TransactionID  string          `json:"transactionId,omitempty"`
	Payload        json.RawMessage `json:"payload,omitempty"`
}
