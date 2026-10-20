package domain

import (
	"encoding/json"
	"time"
)

const TemplateGenerationDynamic = "dynamic"

// NotificationTemplate is a reusable dynamic template (SendGrid-style):
// named resource, versions, and one active version used at send time.
type NotificationTemplate struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Generation string            `json:"generation"`
	Versions   []TemplateVersion `json:"versions,omitempty"`
	CreatedAt  time.Time         `json:"createdAt"`
	UpdatedAt  time.Time         `json:"updatedAt"`
}

func (t *NotificationTemplate) ActiveVersion() *TemplateVersion {
	if t == nil {
		return nil
	}
	for i := range t.Versions {
		if t.Versions[i].Active {
			return &t.Versions[i]
		}
	}
	return nil
}

// TemplateVersion is one revision of a dynamic template.
// Subject, HTML, and plain text use the same placeholders as inline step templates.
// ProviderTemplateID is forwarded to remote workers as contract template_id.
type TemplateVersion struct {
	ID                 string          `json:"id"`
	TemplateID         string          `json:"templateId"`
	Name               string          `json:"name"`
	Active             bool            `json:"active"`
	Subject            string          `json:"subject,omitempty"`
	HTMLContent        string          `json:"htmlContent,omitempty"`
	PlainContent       string          `json:"plainContent,omitempty"`
	Editor             string          `json:"editor,omitempty"`
	ProviderTemplateID string          `json:"providerTemplateId,omitempty"`
	TestData           json.RawMessage `json:"testData,omitempty"`
	CreatedAt          time.Time       `json:"createdAt"`
	UpdatedAt          time.Time       `json:"updatedAt"`
}

// AsMessage maps the version onto the channel body used by delivery.
// Email prefers HTML; sms, push, and chat prefer plain text.
func (v *TemplateVersion) AsMessage(ch ChannelType) *MessageTemplate {
	content := v.HTMLContent
	switch ch {
	case ChannelSMS, ChannelChat, ChannelPush:
		if v.PlainContent != "" {
			content = v.PlainContent
		}
	default:
		if content == "" {
			content = v.PlainContent
		}
	}
	return &MessageTemplate{
		Subject:            v.Subject,
		Title:              v.Subject,
		Content:            content,
		ProviderTemplateID: v.ProviderTemplateID,
	}
}
