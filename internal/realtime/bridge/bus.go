package bridge

import (
	"context"

	"github.com/SolaTyolo/herald/internal/domain"
)

// Backend selects the Worker ↔ API event transport (not client delivery).
type Backend string

const (
	BackendRedis        Backend = "redis"
	BackendLocal        Backend = "local"
	BackendRabbitMQHTTP Backend = "rabbitmq-http"
	BackendKafkaHTTP    Backend = "kafka-http"
)

const DefaultChannel = "herald:worker-api:events"

// Event is the wire format for cross-process Worker → API notification.
type Event struct {
	SubscriberPK string          `json:"subscriberPk"`
	Message      *domain.Message `json:"message"`
}

// EventHandler processes events received on the API process.
type EventHandler func(Event)

// Publisher sends events from the Worker after an in_app message is persisted.
type Publisher interface {
	Publish(subscriberPK string, msg *domain.Message)
}

// Subscriber receives events on the API process (e.g. to POST subscriber webhooks).
type Subscriber interface {
	Run(ctx context.Context, handler EventHandler)
	Close()
}
