package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/SolaTyolo/herald/internal/realtime/bridge"
	"github.com/SolaTyolo/herald/internal/repository"
)

// InAppWebhookDeliverer POSTs inapp messages to the subscriber's webhook URL.
type InAppWebhookDeliverer struct {
	store  repository.Store
	client *http.Client
}

func NewInAppWebhookDeliverer(store repository.Store) *InAppWebhookDeliverer {
	return &InAppWebhookDeliverer{
		store: store,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (d *InAppWebhookDeliverer) Handle(ev bridge.Event) {
	if ev.Message == nil || ev.SubscriberPK == "" {
		return
	}
	ctx := context.Background()
	sub, err := d.store.GetSubscriberByPK(ctx, ev.SubscriberPK)
	if err != nil {
		slog.Warn("inapp webhook: subscriber lookup failed", "subscriberPk", ev.SubscriberPK, "err", err)
		return
	}
	if sub.WebhookURL == "" {
		slog.Debug("inapp webhook: no webhookUrl on subscriber", "subscriberId", sub.SubscriberID)
		return
	}
	body, err := json.Marshal(ev.Message)
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.WebhookURL, bytes.NewReader(body))
	if err != nil {
		slog.Warn("inapp webhook: build request failed", "url", sub.WebhookURL, "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Herald/1.0")
	resp, err := d.client.Do(req)
	if err != nil {
		slog.Warn("inapp webhook: post failed", "url", sub.WebhookURL, "err", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		slog.Warn("inapp webhook: non-2xx", "url", sub.WebhookURL, "status", resp.StatusCode)
	}
}
