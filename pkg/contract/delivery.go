package contract

// ProviderConfig holds channel credentials / settings.
// Vendor-specific fields go in Raw; do not add top-level vendor fields.
type ProviderConfig struct {
	TimeoutSec int               `json:"timeout_sec"`
	Raw        map[string]string `json:"raw"`
}

// Message is the common message payload for all channels.
type Message struct {
	Channel        string            `json:"channel"`
	To             string            `json:"to"`
	Content        string            `json:"content"`
	TemplateID     string            `json:"template_id,omitempty"`
	TemplateParams map[string]string `json:"template_params,omitempty"`
	PushTitle      string            `json:"push_title,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// DeliveryError is the unified send result used for retry / DLQ / audit.
type DeliveryError struct {
	Success       bool   `json:"success"`
	Code          string `json:"code,omitempty"`
	Message       string `json:"message,omitempty"`
	Retryable     bool   `json:"retryable"`
	ProviderBizID string `json:"provider_biz_id,omitempty"`
}

// DeliveryContext carries workflow/notification metadata to remote workers.
type DeliveryContext struct {
	TransactionID  string `json:"transactionId,omitempty"`
	WorkflowID     string `json:"workflowId,omitempty"`
	NotificationID string `json:"notificationId,omitempty"`
}

// DeliveryRequest is the fixed payload Herald posts to remote workers.
type DeliveryRequest struct {
	TaskID         string          `json:"taskId"`
	ProviderName   string          `json:"providerName"`
	TraceID        string          `json:"traceId,omitempty"`
	ProviderConfig ProviderConfig  `json:"providerConfig"`
	Message        Message         `json:"message"`
	Context        DeliveryContext `json:"context,omitempty"`
	// Async asks the worker to return 202 and publish DeliveryResultEvent to the result queue.
	Async bool `json:"async,omitempty"`
}

// DeliveryResultEvent is published by user workers to the shared result queue.
type DeliveryResultEvent struct {
	TaskID         string        `json:"taskId"`
	ProviderName   string        `json:"providerName"`
	TraceID        string        `json:"traceId,omitempty"`
	DeliveryError  DeliveryError `json:"deliveryError"`
	Context        DeliveryContext `json:"context,omitempty"`
}

// OK returns a successful DeliveryError.
func OK(providerBizID string) DeliveryError {
	return DeliveryError{Success: true, ProviderBizID: providerBizID}
}

// Fail returns a failed DeliveryError.
func Fail(code, message string, retryable bool) DeliveryError {
	return DeliveryError{Success: false, Code: code, Message: message, Retryable: retryable}
}
