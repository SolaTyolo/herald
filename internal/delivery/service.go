package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SolaTyolo/herald/internal/delivery/resultqueue"
	"github.com/SolaTyolo/herald/internal/domain"
	"github.com/SolaTyolo/herald/internal/realtime/bridge"
	"github.com/SolaTyolo/herald/internal/repository"
	"github.com/SolaTyolo/herald/internal/routing"
	"github.com/SolaTyolo/herald/internal/template"
	"github.com/SolaTyolo/herald/pkg/contract"
)

type Service struct {
	store    repository.Store
	routes   *routing.Table
	events   bridge.Publisher
	remote   *remoteClient
	smtpSend SMTPSender
	smtp     SMTPConfig
}

// Options configures delivery.Service.
type Options struct {
	AsyncResults bool
	Results      *resultqueue.Queue
	SMTP         SMTPConfig
}

func New(store repository.Store, routes *routing.Table, events bridge.Publisher, opts Options) *Service {
	return &Service{
		store:    store,
		routes:   routes,
		events:   events,
		remote:   newRemoteClient(opts.AsyncResults, opts.Results),
		smtpSend: defaultSMTPSender,
		smtp:     opts.SMTP,
	}
}

func (s *Service) SendChannel(ctx context.Context, channel domain.ChannelType, sub *domain.Subscriber, rendered *domain.MessageTemplate, jobID *string, sendCtx *SendContext) (*domain.Message, error) {
	if jobID != nil && *jobID != "" {
		existing, err := s.store.GetMessageByJobID(ctx, *jobID)
		if err == nil {
			return existing, nil
		}
		if !repository.IsNotFound(err) {
			return nil, err
		}
	}

	if channel == domain.ChannelInApp {
		return s.sendInApp(ctx, sub, rendered, jobID)
	}

	pin := ""
	if sendCtx != nil {
		pin = sendCtx.ProviderName
	}
	route, err := s.resolveProvider(channel, pin)
	if err != nil {
		return nil, err
	}
	return s.sendViaRoute(ctx, channel, route, sub, rendered, jobID, sendCtx)
}

// resolveProvider picks exactly one YAML provider. No same-channel failover:
// pin via WorkflowStep.provider, or use the sole provider registered for the channel.
func (s *Service) resolveProvider(channel domain.ChannelType, providerName string) (routing.Provider, error) {
	if providerName != "" {
		route, ok := s.routes.Lookup(providerName)
		if !ok {
			return routing.Provider{}, fmt.Errorf("provider %q not found in PROVIDERS_CONFIG", providerName)
		}
		if route.Channel != string(channel) {
			return routing.Provider{}, fmt.Errorf("provider %q is channel %q, step expects %q", providerName, route.Channel, channel)
		}
		return route, nil
	}
	providers := s.routes.ProvidersForChannel(string(channel))
	switch len(providers) {
	case 0:
		return routing.Provider{}, fmt.Errorf("no provider for channel %s", channel)
	case 1:
		return providers[0], nil
	default:
		names := make([]string, len(providers))
		for i, p := range providers {
			names[i] = p.Name
		}
		return routing.Provider{}, fmt.Errorf("multiple providers for channel %s (%s); set step.provider", channel, strings.Join(names, ", "))
	}
}

func (s *Service) sendInApp(ctx context.Context, sub *domain.Subscriber, rendered *domain.MessageTemplate, jobID *string) (*domain.Message, error) {
	msg := &domain.Message{
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

func (s *Service) sendViaRoute(ctx context.Context, channel domain.ChannelType, route routing.Provider, sub *domain.Subscriber, rendered *domain.MessageTemplate, jobID *string, sendCtx *SendContext) (*domain.Message, error) {
	if route.Type == routing.TypeRemote && route.Service != "" && !s.routes.IsServiceHealthy(route.Service) {
		return nil, &RetryableError{Code: "service_unhealthy", Message: fmt.Sprintf("service %s is unhealthy", route.Service)}
	}

	cfg := contract.ProviderConfig{
		TimeoutSec: int(route.Timeout.Seconds()),
		Raw:        map[string]string{},
	}
	msgPayload := buildContractMessage(sub, rendered, channel, sendCtx)

	var result contract.DeliveryError
	switch route.Type {
	case routing.TypeBuiltin:
		if route.Name != routing.BuiltinSMTP {
			return nil, fmt.Errorf("builtin provider %s is not sendable via channel delivery", route.Name)
		}
		if err := s.smtp.Validate(); err != nil {
			return nil, err
		}
		cfg.Raw = s.smtp.Raw()
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
			ProviderName:   route.Name,
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
		ref = route.Name
	}
	msg := &domain.Message{
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

func buildContractMessage(sub *domain.Subscriber, rendered *domain.MessageTemplate, channel domain.ChannelType, sendCtx *SendContext) contract.Message {
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
	msg := contract.Message{
		Channel:   string(channel),
		To:        to,
		Content:   rendered.Content,
		PushTitle: rendered.Title,
		Metadata:  meta,
	}
	if rendered.ProviderTemplateID != "" {
		msg.TemplateID = rendered.ProviderTemplateID
		msg.TemplateParams = scalarTemplateParams(sendCtx)
	}
	return msg
}

func scalarTemplateParams(sendCtx *SendContext) map[string]string {
	if sendCtx == nil || len(sendCtx.Payload) == 0 {
		return nil
	}
	raw := map[string]any{}
	if err := json.Unmarshal(sendCtx.Payload, &raw); err != nil {
		return nil
	}
	out := map[string]string{}
	for k, v := range raw {
		switch v.(type) {
		case string, float64, bool, json.Number:
			out[k] = fmt.Sprint(v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func BuildRenderContext(sub *domain.Subscriber, payload json.RawMessage, stepMeta map[string]any) template.RenderContext {
	return template.RenderContext{
		Subscriber: template.SubscriberMap(sub),
		Payload:    template.ParsePayload(payload),
		Step:       stepMeta,
	}
}
