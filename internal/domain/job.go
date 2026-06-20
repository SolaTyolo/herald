package domain

import (
	"encoding/json"
	"time"
)

type Integration struct {
	ID                   string      `json:"id"`
	EnvID                string      `json:"envId"`
	Channel              ChannelType `json:"channel"`
	ProviderID           string      `json:"providerId"`
	Name                 string      `json:"name"`
	CredentialsEncrypted []byte      `json:"-"`
	IsPrimary            bool        `json:"primary"`
	Active               bool        `json:"active"`
	CreatedAt            time.Time   `json:"createdAt"`
	UpdatedAt            time.Time   `json:"updatedAt"`
}

type Notification struct {
	ID            string             `json:"id"`
	EnvID         string             `json:"envId"`
	TransactionID string             `json:"transactionId"`
	WorkflowID    string             `json:"workflowId"`
	SubscriberPK  string             `json:"subscriberPk"`
	Payload       json.RawMessage    `json:"payload"`
	Status        NotificationStatus `json:"status"`
	CreatedAt     time.Time          `json:"createdAt"`
}

type Job struct {
	ID             string          `json:"id"`
	NotificationID string          `json:"notificationId"`
	WorkflowStepID string          `json:"workflowStepId"`
	StepOrder      int             `json:"stepOrder"`
	StepType       StepType        `json:"stepType"`
	Status         JobStatus       `json:"status"`
	Payload        json.RawMessage `json:"payload"`
	Error          string          `json:"error,omitempty"`
	ScheduledAt    *time.Time      `json:"scheduledAt,omitempty"`
	StartedAt      *time.Time      `json:"startedAt,omitempty"`
	CompletedAt    *time.Time      `json:"completedAt,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

type Message struct {
	ID           string          `json:"id"`
	EnvID        string          `json:"envId"`
	JobID        *string         `json:"jobId,omitempty"`
	SubscriberPK string          `json:"subscriberPk"`
	Channel      ChannelType     `json:"channel"`
	Subject      string          `json:"subject,omitempty"`
	Title        string          `json:"title,omitempty"`
	Content      string          `json:"content"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	ProviderRef  string          `json:"providerRef,omitempty"`
	Read         bool            `json:"read"`
	Archived     bool            `json:"archived"`
	CreatedAt    time.Time       `json:"createdAt"`
}

type DigestBucket struct {
	ID           string          `json:"id"`
	EnvID        string          `json:"envId"`
	DigestKey    string          `json:"digestKey"`
	SubscriberPK string          `json:"subscriberPk"`
	WorkflowID   string          `json:"workflowId"`
	StepID       string          `json:"stepId"`
	JobID        *string         `json:"jobId,omitempty"`
	Events       json.RawMessage `json:"events"`
	WindowEnd    time.Time       `json:"windowEnd"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

type ExecutionLog struct {
	ID             string          `json:"id"`
	EnvID          string          `json:"envId"`
	TransactionID  string          `json:"transactionId"`
	NotificationID *string         `json:"notificationId,omitempty"`
	JobID          *string         `json:"jobId,omitempty"`
	StepType       string          `json:"stepType,omitempty"`
	Level          string          `json:"level"`
	Message        string          `json:"message"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
}

type TriggerTarget struct {
	SubscriberID string `json:"subscriberId,omitempty"`
	Type         string `json:"type,omitempty"`
	TopicKey     string `json:"topicKey,omitempty"`
}

type TriggerRequest struct {
	Name          string          `json:"name"`
	To            TriggerTarget   `json:"to"`
	Payload       json.RawMessage `json:"payload"`
	TransactionID string          `json:"transactionId,omitempty"`
	Actor         string          `json:"actor,omitempty"`
}

type TriggerResponse struct {
	TransactionID string   `json:"transactionId"`
	NotificationIDs []string `json:"notificationIds,omitempty"`
}
