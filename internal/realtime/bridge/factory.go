package bridge

import (
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/SolaTyolo/herald/internal/config"
)

// OpenPublisher creates a Publisher for the configured Worker ↔ API backend.
func OpenPublisher(cfg *config.Config, redis *redis.Client) (Publisher, error) {
	switch Backend(cfg.WorkerAPIPubSub) {
	case BackendLocal:
		return sharedLocalBus.publisher(), nil
	case BackendRabbitMQHTTP:
		return NewRabbitMQHTTPPublisher(cfg), nil
	case BackendKafkaHTTP:
		return NewKafkaHTTPPublisher(cfg), nil
	case BackendRedis, "":
		if redis == nil {
			return nil, fmt.Errorf("redis client required for WORKER_API_PUBSUB=redis")
		}
		return NewRedisPublisher(redis), nil
	default:
		return nil, fmt.Errorf("unsupported WORKER_API_PUBSUB: %s", cfg.WorkerAPIPubSub)
	}
}

// OpenSubscriber creates a Subscriber for the API process.
func OpenSubscriber(cfg *config.Config, redis *redis.Client) (Subscriber, error) {
	switch Backend(cfg.WorkerAPIPubSub) {
	case BackendLocal:
		return sharedLocalBus.subscriber(), nil
	case BackendRabbitMQHTTP:
		return NewRabbitMQHTTPSubscriber(cfg), nil
	case BackendKafkaHTTP:
		return NewKafkaHTTPSubscriber(cfg), nil
	case BackendRedis, "":
		if redis == nil {
			return nil, fmt.Errorf("redis client required for WORKER_API_PUBSUB=redis")
		}
		return NewRedisSubscriber(redis), nil
	default:
		return nil, fmt.Errorf("unsupported WORKER_API_PUBSUB: %s", cfg.WorkerAPIPubSub)
	}
}
