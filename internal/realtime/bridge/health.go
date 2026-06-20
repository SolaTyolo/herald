package bridge

import (
	"context"

	"github.com/SolaTyolo/herald/internal/config"
)

// CheckBackend probes the configured Worker ↔ API transport when applicable.
func CheckBackend(ctx context.Context, cfg *config.Config) error {
	if cfg == nil {
		return nil
	}
	switch Backend(cfg.WorkerAPIPubSub) {
	case BackendRabbitMQHTTP:
		return checkRabbitMQSidecar(ctx, cfg)
	case BackendKafkaHTTP:
		return checkKafkaHTTP(ctx, cfg)
	default:
		return nil
	}
}
