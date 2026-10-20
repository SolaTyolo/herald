package service_test

import (
	"context"
	"testing"

	"github.com/SolaTyolo/herald/internal/domain"
	"github.com/SolaTyolo/herald/internal/repository/filestore"
	"github.com/SolaTyolo/herald/internal/service"
)

func TestTemplateVersionsAndPreview(t *testing.T) {
	t.Parallel()
	store, err := filestore.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := service.NewTemplateService(store)
	ctx := context.Background()

	tmpl := &domain.NotificationTemplate{Name: "order-shipped"}
	if err := svc.Create(ctx, tmpl); err != nil {
		t.Fatal(err)
	}

	first := &domain.TemplateVersion{
		Name:         "v1",
		Subject:      "Order {{.orderId}}",
		HTMLContent:  "<p>{{.subscriber.firstName}}</p>",
		PlainContent: "{{.subscriber.firstName}} {{.orderId}}",
		TestData:     []byte(`{"orderId":"100"}`),
	}
	if err := svc.CreateVersion(ctx, tmpl.ID, first); err != nil {
		t.Fatal(err)
	}
	if !first.Active {
		t.Fatal("first version should be active")
	}

	second := &domain.TemplateVersion{
		Name:               "v2",
		Active:             true,
		Subject:            "Shipped {{.orderId}}",
		PlainContent:       "shipped",
		ProviderTemplateID: "d-sendgrid",
	}
	if err := svc.CreateVersion(ctx, tmpl.ID, second); err != nil {
		t.Fatal(err)
	}

	got, err := svc.Get(ctx, tmpl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Generation != domain.TemplateGenerationDynamic {
		t.Fatalf("generation %s", got.Generation)
	}
	active := got.ActiveVersion()
	if active == nil || active.ID != second.ID {
		t.Fatalf("active version = %+v", active)
	}
	if got.Versions[0].Active {
		t.Fatal("v1 should be inactive")
	}

	preview, err := svc.Preview(ctx, tmpl.ID, "", map[string]any{"orderId": "55"}, map[string]any{"firstName": "Ada"})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Subject != "Shipped 55" {
		t.Fatalf("subject %q", preview.Subject)
	}

	email := active.AsMessage(domain.ChannelEmail)
	if email.Content != "shipped" {
		t.Fatalf("email content %q", email.Content)
	}
	sms := active.AsMessage(domain.ChannelSMS)
	if sms.Content != "shipped" || sms.ProviderTemplateID != "d-sendgrid" {
		t.Fatalf("sms message %+v", sms)
	}
}
