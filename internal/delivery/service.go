package delivery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SolaTyolo/herald/internal/crypto"
	"github.com/SolaTyolo/herald/internal/domain"
	"github.com/SolaTyolo/herald/internal/platform/plugin/wasm"
	"github.com/SolaTyolo/herald/internal/realtime/bridge"
	"github.com/SolaTyolo/herald/internal/repository"
	"github.com/SolaTyolo/herald/internal/template"
	"github.com/SolaTyolo/herald/pkg/plugin"
)

type Service struct {
	store     repository.Store
	encryptor *crypto.Encryptor
	wasm      *wasm.Runtime
	events    bridge.Publisher
}

func New(store repository.Store, encryptor *crypto.Encryptor, wasm *wasm.Runtime, events bridge.Publisher) *Service {
	return &Service{store: store, encryptor: encryptor, wasm: wasm, events: events}
}

func (s *Service) SendChannel(ctx context.Context, envID string, channel domain.ChannelType, sub *domain.Subscriber, rendered *domain.MessageTemplate, jobID *string, sendCtx *SendContext) (*domain.Message, error) {
	if channel == domain.ChannelInApp {
		return s.sendInApp(ctx, envID, sub, rendered, jobID)
	}

	primaryOnly := channel == domain.ChannelEmail || channel == domain.ChannelSMS
	integrations, err := s.store.ListActiveIntegrations(ctx, envID, channel, primaryOnly)
	if err != nil {
		return nil, err
	}
	if len(integrations) == 0 {
		return nil, fmt.Errorf("no active integration for channel %s", channel)
	}

	var lastErr error
	for _, integration := range integrations {
		msg, err := s.sendViaIntegration(ctx, envID, channel, &integration, sub, rendered, jobID, sendCtx)
		if err == nil {
			return msg, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (s *Service) sendInApp(ctx context.Context, envID string, sub *domain.Subscriber, rendered *domain.MessageTemplate, jobID *string) (*domain.Message, error) {
	msg := &domain.Message{
		EnvID:        envID,
		JobID:        jobID,
		SubscriberPK: sub.ID,
		Channel:      domain.ChannelInApp,
		Subject:      rendered.Subject,
		Title:        rendered.Title,
		Content:      rendered.Content,
		ProviderRef:  "in-app",
		Metadata:     json.RawMessage("{}"),
	}
	if err := s.store.CreateMessage(ctx, msg); err != nil {
		return nil, err
	}
	if s.events != nil {
		s.events.Publish(sub.ID, msg)
	}
	return msg, nil
}

func (s *Service) sendViaIntegration(ctx context.Context, envID string, channel domain.ChannelType, integration *domain.Integration, sub *domain.Subscriber, rendered *domain.MessageTemplate, jobID *string, sendCtx *SendContext) (*domain.Message, error) {
	cfg := plugin.Config{}
	if err := s.encryptor.DecryptJSON(integration.CredentialsEncrypted, &cfg); err != nil {
		return nil, err
	}

	req := buildSendRequest(sub, rendered, channel)
	var result *plugin.SendResult
	var err error

	if s.wasm == nil || !s.wasm.HasProvider(integration.ProviderID) {
		return nil, fmt.Errorf("wasm provider not loaded: %s", integration.ProviderID)
	}
	cc := wasm.CallContext{
		EnvID:        envID,
		SubscriberPK: sub.ID,
	}
	if sendCtx != nil {
		cc.TransactionID = sendCtx.TransactionID
		cc.WorkflowID = sendCtx.WorkflowID
		cc.NotificationID = sendCtx.NotificationID
		cc.Payload = sendCtx.Payload
	}
	result, err = s.wasm.Send(ctx, integration.ProviderID, cfg, req, cc)
	if err != nil {
		return nil, err
	}

	msg := &domain.Message{
		EnvID:        envID,
		JobID:        jobID,
		SubscriberPK: sub.ID,
		Channel:      channel,
		Subject:      rendered.Subject,
		Title:        rendered.Title,
		Content:      rendered.Content,
		ProviderRef:  result.ProviderRef,
		Metadata:     json.RawMessage("{}"),
	}
	if err := s.store.CreateMessage(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func buildSendRequest(sub *domain.Subscriber, rendered *domain.MessageTemplate, channel domain.ChannelType) *plugin.SendRequest {
	addr := plugin.Address{Email: sub.Email, Phone: sub.Phone}
	var tokens []string
	_ = json.Unmarshal(sub.DeviceTokens, &tokens)
	addr.DeviceTokens = tokens

	chatCreds := map[string]string{}
	_ = json.Unmarshal(sub.ChatCredentials, &chatCreds)
	if channel == domain.ChannelChat {
		if v, ok := chatCreds["default"]; ok {
			addr.ChatTarget = v
		}
	}

	return &plugin.SendRequest{
		To:       addr,
		Subject:  rendered.Subject,
		Title:    rendered.Title,
		Body:     rendered.Content,
		HTMLBody: rendered.Content,
	}
}

func BuildRenderContext(sub *domain.Subscriber, payload json.RawMessage, stepMeta map[string]any) template.RenderContext {
	return template.RenderContext{
		Subscriber: template.SubscriberMap(sub),
		Payload:    template.ParsePayload(payload),
		Step:       stepMeta,
	}
}
