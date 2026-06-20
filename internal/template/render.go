package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/SolaTyolo/herald/internal/domain"
)

type RenderContext struct {
	Subscriber map[string]any
	Payload    map[string]any
	Step       map[string]any
	Actor      map[string]any
}

func ParsePayload(raw json.RawMessage) map[string]any {
	out := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}

func SubscriberMap(sub *domain.Subscriber) map[string]any {
	data := map[string]any{}
	_ = json.Unmarshal(sub.Data, &data)
	return map[string]any{
		"subscriberId": sub.SubscriberID,
		"email":        sub.Email,
		"phone":        sub.Phone,
		"firstName":    sub.FirstName,
		"lastName":     sub.LastName,
		"locale":       sub.Locale,
		"data":         data,
	}
}

func Render(tmpl string, ctx RenderContext) (string, error) {
	if tmpl == "" {
		return "", nil
	}
	t, err := template.New("msg").Option("missingkey=zero").Parse(tmpl)
	if err != nil {
		return "", err
	}
	data := map[string]any{
		"subscriber": ctx.Subscriber,
		"payload":    ctx.Payload,
		"step":       ctx.Step,
		"actor":      ctx.Actor,
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render template: %w", err)
	}
	return buf.String(), nil
}

func RenderMessageTemplate(mt *domain.MessageTemplate, ctx RenderContext) (*domain.MessageTemplate, error) {
	if mt == nil {
		return &domain.MessageTemplate{}, nil
	}
	out := *mt
	var err error
	if out.Subject, err = Render(mt.Subject, ctx); err != nil {
		return nil, err
	}
	if out.Title, err = Render(mt.Title, ctx); err != nil {
		return nil, err
	}
	if out.Content, err = Render(mt.Content, ctx); err != nil {
		return nil, err
	}
	return &out, nil
}
