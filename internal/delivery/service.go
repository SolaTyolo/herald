package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SolaTyolo/herald/internal/crypto"
	"github.com/SolaTyolo/herald/internal/delivery/resultqueue"
	"github.com/SolaTyolo/herald/internal/domain"
	"github.com/SolaTyolo/herald/internal/realtime/bridge"
	"github.com/SolaTyolo/herald/internal/repository"
	"github.com/SolaTyolo/herald/internal/routing"
	"github.com/SolaTyolo/herald/internal/template"
	"github.com/SolaTyolo/herald/pkg/contract"
)

type Service struct {
	store     repository.Store
	encryptor *crypto.Encryptor
	routes    *routing.Table
	events    bridge.Publisher
	remote    *remoteClient
	smtpSend  SMTPSender
}

// Options configures delivery.Service.
type Options struct {
	AsyncResults bool
	Results      *resultqueue.Queue
}

func New(store repository.Store, encryptor *crypto.Encryptor, routes *routing.Table, events bridge.Publisher, opts Options) *Service {
	return &Service{
		store:     store,
		encryptor: encryptor,
		routes:    routes,
		events:    events,
		remote:    newRemoteClient(opts.AsyncResults, opts.Results),
		smtpSend:  defaultSMTPSender,
	}
}

func (s *Service) SendChannel(ctx context.Context, envID string, channel domain.ChannelType, sub *domain.Subscriber, rendered *domain.MessageTemplate, jobID *string, sendCtx *SendContext) (*domain.Message, error) {
	if jobID != nil && *jobID != "" {
		existing, err := s.store.GetMessageByJobID(ctx, envID, *jobID)
		if err == nil {
			return existing, nil
		}
		if !repository.IsNotFound(err) {
			return nil, err
		}
	}

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
		// Retryable failures must surface to Asynq — do not failover to another integration.
		if IsRetryable(err) {
			return nil, err
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
		ProviderRef:  routing.BuiltinInApp,
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
	route, ok := s.routes.Lookup(integration.ProviderID)
	if !ok {
		return nil, fmt.Errorf("provider not in routing table: %s", integration.ProviderID)
	}

	if route.Type == routing.TypeRemote && route.Service != "" && !s.routes.IsServiceHealthy(route.Service) {
		return nil, &RetryableError{Code: "service_unhealthy", Message: fmt.Sprintf("service %s is unhealthy", route.Service)}
	}

	creds := map[string]any{}
	if err := s.encryptor.DecryptJSON(integration.CredentialsEncrypted, &creds); err != nil {
		return nil, err
	}
	raw := credentialsToRaw(creds)
	cfg := contract.ProviderConfig{
		TimeoutSec: int(route.Timeout.Seconds()),
		Raw:        raw,
	}
	msgPayload := buildContractMessage(sub, rendered, channel)

	var result contract.DeliveryError
	switch route.Type {
	case routing.TypeBuiltin:
		if route.Name != routing.BuiltinSMTP {
			return nil, fmt.Errorf("builtin provider %s is not sendable via integration", route.Name)
		}
		result = s.smtpSend(ctx, cfg, msgPayload)
	case routing.TypeRemote:
		taskID := ""
		if jobID != nil {
			taskID = *jobID
		}
		delCtx := contract.DeliveryContext{}
		traceID := ""
		if sendCtx != nil {
			traceID = sendCtx.TransactionID
			delCtx.TransactionID = sendCtx.TransactionID
			delCtx.WorkflowID = sendCtx.WorkflowID
			delCtx.NotificationID = sendCtx.NotificationID
		}
		result = s.remote.Send(ctx, route, contract.DeliveryRequest{
			TaskID:         taskID,
			ProviderName:   integration.ProviderID,
			TraceID:        traceID,
			ProviderConfig: cfg,
			Message:        msgPayload,
			Context:        delCtx,
		})
	default:
		return nil, fmt.Errorf("unknown provider type %q", route.Type)
	}

	if !result.Success {
		if result.Retryable {
			return nil, &RetryableError{Code: result.Code, Message: result.Message}
		}
		return nil, fmt.Errorf("delivery failed (%s): %s", result.Code, result.Message)
	}

	ref := result.ProviderBizID
	if ref == "" {
		ref = integration.ProviderID
	}
	msg := &domain.Message{
		EnvID:        envID,
		JobID:        jobID,
		SubscriberPK: sub.ID,
		Channel:      channel,
		Subject:      rendered.Subject,
		Title:        rendered.Title,
		Content:      rendered.Content,
		ProviderRef:  ref,
		Metadata:     json.RawMessage("{}"),
	}
	if err := s.store.CreateMessage(ctx, msg); err != nil {
		// Persist failure after provider success: treat as retryable so Asynq can
		// re-run; GetMessageByJobID will short-circuit if write actually succeeded.
		return nil, &RetryableError{Code: "persist_error", Message: err.Error()}
	}
	return msg, nil
}

func buildContractMessage(sub *domain.Subscriber, rendered *domain.MessageTemplate, channel domain.ChannelType) contract.Message {
	to := ""
	meta := map[string]string{}
	switch channel {
	case domain.ChannelEmail:
		to = sub.Email
		meta["subject"] = rendered.Subject
	case domain.ChannelSMS:
		to = sub.Phone
	case domain.ChannelPush:
		var tokens []string
		_ = json.Unmarshal(sub.DeviceTokens, &tokens)
		if len(tokens) > 0 {
			to = tokens[0]
		}
		if len(tokens) > 1 {
			meta["device_tokens"] = strings.Join(tokens, ",")
		}
	case domain.ChannelChat:
		chatCreds := map[string]string{}
		_ = json.Unmarshal(sub.ChatCredentials, &chatCreds)
		if v, ok := chatCreds["default"]; ok {
			to = v
		}
	}
	return contract.Message{
		Channel:   string(channel),
		To:        to,
		Content:   rendered.Content,
		PushTitle: rendered.Title,
		Metadata:  meta,
	}
}

func BuildRenderContext(sub *domain.Subscriber, payload json.RawMessage, stepMeta map[string]any) template.RenderContext {
	return template.RenderContext{
		Subscriber: template.SubscriberMap(sub),
		Payload:    template.ParsePayload(payload),
		Step:       stepMeta,
	}
}
