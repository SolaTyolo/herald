package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/SolaTyolo/herald/internal/config"
	"github.com/SolaTyolo/herald/internal/repository"
	"github.com/SolaTyolo/herald/internal/routing"
	"github.com/SolaTyolo/herald/internal/service"
)

type API struct {
	cfg           *config.Config
	trigger       *service.TriggerService
	subscribers   *service.SubscriberService
	workflows     *service.WorkflowService
	topics        *service.TopicService
	integrations  *service.IntegrationService
	notifications *service.NotificationQueryService
	messages      *service.MessageService
	templates     *service.TemplateService
	store         repository.Store
	redis         *redis.Client
	routes        *routing.Table
}

func NewAPI(
	cfg *config.Config,
	trigger *service.TriggerService,
	subscribers *service.SubscriberService,
	workflows *service.WorkflowService,
	topics *service.TopicService,
	integrations *service.IntegrationService,
	notifications *service.NotificationQueryService,
	messages *service.MessageService,
	templates *service.TemplateService,
	store repository.Store,
	redisClient *redis.Client,
	routes *routing.Table,
) *API {
	return &API{
		cfg:     cfg,
		trigger: trigger, subscribers: subscribers, workflows: workflows,
		topics: topics, integrations: integrations, notifications: notifications,
		messages: messages, templates: templates,
		store: store, redis: redisClient, routes: routes,
	}
}

func (a *API) Routes(r chi.Router) {
	r.Post("/v1/events/trigger", a.triggerEvent)

	r.Route("/v1/subscribers", func(r chi.Router) {
		r.Get("/", a.listSubscribers)
		r.Post("/", a.createSubscriber)
		r.Get("/{subscriberId}", a.getSubscriber)
		r.Delete("/{subscriberId}", a.deleteSubscriber)
		r.Put("/{subscriberId}/preferences", a.setPreference)
		r.Get("/{subscriberId}/messages", a.listMessages)
		r.Patch("/{subscriberId}/messages/{messageId}", a.patchMessage)
	})

	r.Route("/v1/workflows", func(r chi.Router) {
		r.Get("/", a.listWorkflows)
		r.Post("/", a.createWorkflow)
		r.Get("/{id}", a.getWorkflow)
		r.Put("/{id}", a.updateWorkflow)
		r.Delete("/{id}", a.deleteWorkflow)
	})

	r.Route("/v1/topics", func(r chi.Router) {
		r.Get("/", a.listTopics)
		r.Post("/", a.createTopic)
		r.Delete("/{topicKey}", a.deleteTopic)
		r.Post("/{topicKey}/subscriptions", a.subscribeTopic)
		r.Delete("/{topicKey}/subscriptions/{subscriberId}", a.unsubscribeTopic)
	})

	r.Route("/v1/integrations", func(r chi.Router) {
		r.Get("/", a.listIntegrations)
		r.Post("/", a.createIntegration)
		r.Delete("/{id}", a.deleteIntegration)
	})

	r.Get("/v1/notifications", a.listNotifications)
	r.Get("/v1/notifications/{transactionId}/jobs", a.listJobs)
	r.Get("/v1/notifications/{transactionId}/logs", a.listLogs)

	r.Route("/v1/templates", func(r chi.Router) {
		r.Get("/", a.listTemplates)
		r.Post("/", a.createTemplate)
		r.Get("/{id}", a.getTemplate)
		r.Put("/{id}", a.updateTemplate)
		r.Delete("/{id}", a.deleteTemplate)
		r.Post("/{id}/preview", a.previewTemplate)
		r.Post("/{id}/versions", a.createTemplateVersion)
		r.Put("/{id}/versions/{versionId}", a.updateTemplateVersion)
		r.Delete("/{id}/versions/{versionId}", a.deleteTemplateVersion)
	})

	r.Get("/v1/providers", a.listProviders)

	r.Get("/health", a.health)
}

func envID(r *http.Request) string {
	return r.Context().Value(envKey{}).(string)
}

type envKey struct{}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
