package bridge

import (
	"context"

	"github.com/SolaTyolo/herald/internal/domain"
)

const DefaultChannel = "herald:worker-api:events"

// Event is the wire format for cross-process Worker → API notification.
type Event struct {
	SubscriberPK string          `json:"subscriberPk"`
	Message      *domain.Message `json:"message"`
}

// EventHandler processes events received on the API process.
type EventHandler func(Event)

// Publisher sends events from the Worker after an inapp message is persisted.
type Publisher interface {
	Publish(subscriberPK string, msg *domain.Message)
}

// Subscriber receives events on the API process (e.g. to POST subscriber webhooks).
type Subscriber interface {
	Run(ctx context.Context, handler EventHandler)
	Close()
}
