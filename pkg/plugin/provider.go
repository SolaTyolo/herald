package plugin

import "encoding/json"

// Config holds integration credentials passed to WASM providers.
type Config map[string]any

type Address struct {
	Email        string   `json:"email,omitempty"`
	Phone        string   `json:"phone,omitempty"`
	DeviceTokens []string `json:"deviceTokens,omitempty"`
	ChatTarget   string   `json:"chatTarget,omitempty"`
}

type SendRequest struct {
	To       Address         `json:"to"`
	Subject  string          `json:"subject,omitempty"`
	Title    string          `json:"title,omitempty"`
	Body     string          `json:"body"`
	HTMLBody string          `json:"htmlBody,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type SendResult struct {
	ProviderRef string `json:"providerRef,omitempty"`
	Retryable   bool   `json:"retryable,omitempty"`
}
