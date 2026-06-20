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

// InAppWebhookDeliverer POSTs in_app messages to the subscriber's webhook URL.
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
		slog.Warn("in_app webhook: subscriber lookup failed", "subscriberPk", ev.SubscriberPK, "err", err)
		return
	}
	if sub.WebhookURL == "" {
		slog.Debug("in_app webhook: no webhookUrl on subscriber", "subscriberId", sub.SubscriberID)
		return
	}
	body, err := json.Marshal(ev.Message)
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.WebhookURL, bytes.NewReader(body))
	if err != nil {
		slog.Warn("in_app webhook: build request failed", "url", sub.WebhookURL, "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Herald/1.0")
	resp, err := d.client.Do(req)
	if err != nil {
		slog.Warn("in_app webhook: post failed", "url", sub.WebhookURL, "err", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		slog.Warn("in_app webhook: non-2xx", "url", sub.WebhookURL, "status", resp.StatusCode)
	}
}
