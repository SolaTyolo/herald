package template_test

import (
	"testing"

	"github.com/SolaTyolo/herald/internal/domain"
	"github.com/SolaTyolo/herald/internal/template"
)

func TestRenderMessageTemplate(t *testing.T) {
	mt := &domain.MessageTemplate{
		Subject: "Hello {{.subscriber.firstName}}",
		Content: "Order {{.payload.orderId}} shipped",
	}
	ctx := template.RenderContext{
		Subscriber: map[string]any{"firstName": "Alice"},
		Payload:    map[string]any{"orderId": "123"},
	}
	out, err := template.RenderMessageTemplate(mt, ctx)
	if err != nil {
		t.Fatal(err)
	}
	if out.Subject != "Hello Alice" {
		t.Fatalf("subject: %s", out.Subject)
	}
	if out.Content != "Order 123 shipped" {
		t.Fatalf("content: %s", out.Content)
	}
}
