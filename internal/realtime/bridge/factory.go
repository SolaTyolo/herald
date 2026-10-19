package bridge

import (
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/SolaTyolo/herald/internal/config"
)

// OpenPublisher creates a Publisher for the configured queue mode.
func OpenPublisher(cfg *config.Config, redis *redis.Client) (Publisher, error) {
	switch cfg.QueueMode {
	case config.QueueModeLocal:
		return sharedLocalBus.publisher(), nil
	case config.QueueModeAsynq:
		if redis == nil {
			return nil, fmt.Errorf("redis client required for QUEUE_MODE=asynq")
		}
		return NewRedisPublisher(redis), nil
	default:
		return nil, fmt.Errorf("unsupported QUEUE_MODE: %s", cfg.QueueMode)
	}
}

// OpenSubscriber creates a Subscriber for the API process.
func OpenSubscriber(cfg *config.Config, redis *redis.Client) (Subscriber, error) {
	switch cfg.QueueMode {
	case config.QueueModeLocal:
		return sharedLocalBus.subscriber(), nil
	case config.QueueModeAsynq:
		if redis == nil {
			return nil, fmt.Errorf("redis client required for QUEUE_MODE=asynq")
		}
		return NewRedisSubscriber(redis), nil
	default:
		return nil, fmt.Errorf("unsupported QUEUE_MODE: %s", cfg.QueueMode)
	}
}
