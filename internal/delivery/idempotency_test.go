package delivery_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/SolaTyolo/herald/internal/delivery"
	"github.com/SolaTyolo/herald/internal/domain"
	"github.com/SolaTyolo/herald/internal/repository/filestore"
	"github.com/SolaTyolo/herald/internal/routing"
)

func TestSendChannelIdempotentByJobID(t *testing.T) {
	store, err := filestore.Open(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	jobID := "job-idem-1"
	existing := &domain.Message{
		JobID:        &jobID,
		SubscriberPK: "sub-1",
		Channel:      domain.ChannelEmail,
		Content:      "sent",
		ProviderRef:  "prior",
	}
	if err := store.CreateMessage(context.Background(), existing); err != nil {
		t.Fatal(err)
	}

	snap, err := routing.Parse([]byte(`
providers:
  smtp_email:
    type: builtin
    channel: email
  inapp:
    type: builtin
    channel: inapp
`))
	if err != nil {
		t.Fatal(err)
	}
	svc := delivery.New(store, routing.NewTable("mem", snap), nil, delivery.Options{})
	got, err := svc.SendChannel(context.Background(), domain.ChannelEmail, &domain.Subscriber{ID: "sub-1"}, &domain.MessageTemplate{Content: "new"}, &jobID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.ProviderRef != "prior" {
		t.Fatalf("expected idempotent prior message, got %+v", got)
	}
}
