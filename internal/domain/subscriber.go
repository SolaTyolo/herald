package domain

import (
	"encoding/json"
	"time"
)

type Subscriber struct {
	ID              string          `json:"id"`
	EnvID           string          `json:"envId"`
	SubscriberID    string          `json:"subscriberId"`
	Email           string          `json:"email,omitempty"`
	Phone           string          `json:"phone,omitempty"`
	FirstName       string          `json:"firstName,omitempty"`
	LastName        string          `json:"lastName,omitempty"`
	Locale          string          `json:"locale,omitempty"`
	WebhookURL      string          `json:"webhookUrl,omitempty"`
	Data            json.RawMessage `json:"data,omitempty"`
	DeviceTokens    json.RawMessage `json:"deviceTokens,omitempty"`
	ChatCredentials json.RawMessage `json:"chatCredentials,omitempty"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

type SubscriberPreference struct {
	ID           string      `json:"id"`
	SubscriberPK string      `json:"subscriberPk"`
	WorkflowID   *string     `json:"workflowId,omitempty"`
	Channel      ChannelType `json:"channel"`
	Enabled      bool        `json:"enabled"`
}

type Topic struct {
	ID        string    `json:"id"`
	EnvID     string    `json:"envId"`
	TopicKey  string    `json:"topicKey"`
	Name      string    `json:"name,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type TopicSubscription struct {
	ID           string    `json:"id"`
	TopicID      string    `json:"topicId"`
	SubscriberPK string    `json:"subscriberPk"`
	CreatedAt    time.Time `json:"createdAt"`
}
