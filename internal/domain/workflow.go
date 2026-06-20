package domain

import (
	"encoding/json"
	"time"
)

type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type Environment struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenantId"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"createdAt"`
}

type APIKey struct {
	ID        string    `json:"id"`
	EnvID     string    `json:"envId"`
	Name      string    `json:"name"`
	KeyPrefix string    `json:"keyPrefix"`
	CreatedAt time.Time `json:"createdAt"`
}

type MessageTemplate struct {
	Subject    string          `json:"subject,omitempty"`
	Title      string          `json:"title,omitempty"`
	Content    string          `json:"content"`
	Variables  json.RawMessage `json:"variables,omitempty"`
	SenderName string          `json:"senderName,omitempty"`
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
	DigestKey   string `json:"digestKey,omitempty"`
	WindowAmount int   `json:"windowAmount,omitempty"`
	WindowUnit  string `json:"windowUnit,omitempty"`

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
	Template   *MessageTemplate `json:"template,omitempty"`
	Filters    []StepFilter     `json:"filters,omitempty"`
	Metadata   StepMetadata     `json:"metadata,omitempty"`
	CreatedAt  time.Time        `json:"createdAt"`
}

type ChannelPrefs map[ChannelType]bool

type Workflow struct {
	ID                  string       `json:"id"`
	EnvID               string       `json:"envId"`
	Name                string       `json:"name"`
	TriggerID           string       `json:"triggerId"`
	Critical            bool         `json:"critical"`
	PreferenceSettings  ChannelPrefs `json:"preferenceSettings"`
	Active              bool         `json:"active"`
	Steps               []WorkflowStep `json:"steps,omitempty"`
	CreatedAt           time.Time    `json:"createdAt"`
	UpdatedAt           time.Time    `json:"updatedAt"`
}
