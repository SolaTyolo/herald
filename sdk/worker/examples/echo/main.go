package main

import (
	"context"
	"log/slog"

	"github.com/SolaTyolo/herald/pkg/contract"
	heraldworker "github.com/SolaTyolo/herald/sdk/worker"
)

type EchoProvider struct{}

func (p *EchoProvider) Send(_ context.Context, cfg contract.ProviderConfig, msg contract.Message) contract.DeliveryError {
	slog.Info("echo send", "to", msg.To, "channel", msg.Channel, "raw_keys", len(cfg.Raw))
	return contract.OK("echo-" + msg.To)
}

func main() {
	// Shares REDIS_ADDR / HERALD_RESULT_REDIS with Herald for async result queue.
	heraldworker.RegisterProvider("echo_sms", &EchoProvider{})
	if err := heraldworker.Run(":8080"); err != nil {
		slog.Error("worker stopped", "err", err)
	}
}
