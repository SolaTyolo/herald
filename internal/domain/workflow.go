package domain

import (
	"encoding/json"
	"time"
)

type MessageTemplate struct {
	Subject            string          `json:"subject,omitempty"`
	Title              string          `json:"title,omitempty"`
	Content            string          `json:"content"`
	Variables          json.RawMessage `json:"variables,omitempty"`
	SenderName         string          `json:"senderName,omitempty"`
	ProviderTemplateID string          `json:"providerTemplateId,omitempty"`
}

type StepFilter struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    any    `json:"value"`
}

type StepMetadata struct {
	// Delay
	DelayAmount int    `json:"delayAmount,omitempty"`
	DelayUnit   string `json:"delayUnit,omitempty"` // seconds, minutes, hours

	// Digest
	DigestKey    string `json:"digestKey,omitempty"`
	WindowAmount int    `json:"windowAmount,omitempty"`
	WindowUnit   string `json:"windowUnit,omitempty"`

	// Throttle
	ThrottleKey    string `json:"throttleKey,omitempty"`
	ThrottleLimit  int    `json:"throttleLimit,omitempty"`
	ThrottleWindow int    `json:"throttleWindow,omitempty"` // seconds
	ThrottleAction string `json:"throttleAction,omitempty"` // skip or delay
}

type WorkflowStep struct {
	ID         string           `json:"id"`
	WorkflowID string           `json:"workflowId"`
	StepOrder  int              `json:"stepOrder"`
	Type       StepType         `json:"type"`
	// Provider pins a YAML providerName for this step (e.g. lark_webhook / wecom_webhook).
	// Required when multiple providers share the step's channel; ignored for inapp.
	Provider   string           `json:"provider,omitempty"`
	TemplateID string           `json:"templateId,omitempty"`
	Template   *MessageTemplate `json:"template,omitempty"`
	Filters    []StepFilter     `json:"filters,omitempty"`
	Metadata   StepMetadata     `json:"metadata,omitempty"`
	CreatedAt  time.Time        `json:"createdAt"`
}

type ChannelPrefs map[ChannelType]bool

type Workflow struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name"`
	TriggerID          string         `json:"triggerId"`
	Critical           bool           `json:"critical"`
	PreferenceSettings ChannelPrefs   `json:"preferenceSettings"`
	Active             bool           `json:"active"`
	Steps              []WorkflowStep `json:"steps,omitempty"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
}
