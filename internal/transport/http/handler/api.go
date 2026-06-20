package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/SolaTyolo/herald/internal/config"
	"github.com/SolaTyolo/herald/internal/domain"
	"github.com/SolaTyolo/herald/internal/platform/plugin/wasm"
	"github.com/SolaTyolo/herald/internal/realtime/bridge"
	"github.com/SolaTyolo/herald/internal/repository"
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
	store         repository.Store
	redis         *redis.Client
	wasmManifest  func() []wasm.Manifest
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
	store repository.Store,
	redisClient *redis.Client,
	wasmList func() []wasm.Manifest,
) *API {
	return &API{
		cfg: cfg,
		trigger: trigger, subscribers: subscribers, workflows: workflows,
		topics: topics, integrations: integrations, notifications: notifications,
		messages: messages,
		store: store, redis: redisClient, wasmManifest: wasmList,
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

func (a *API) triggerEvent(w http.ResponseWriter, r *http.Request) {
	var req domain.TriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := a.trigger.Trigger(r.Context(), envID(r), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *API) createSubscriber(w http.ResponseWriter, r *http.Request) {
	var sub domain.Subscriber
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if sub.SubscriberID == "" {
		writeError(w, http.StatusBadRequest, "subscriberId required")
		return
	}
	if err := a.subscribers.Upsert(r.Context(), envID(r), &sub); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, sub)
}

func (a *API) getSubscriber(w http.ResponseWriter, r *http.Request) {
	sub, err := a.subscribers.Get(r.Context(), envID(r), chi.URLParam(r, "subscriberId"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

func (a *API) listSubscribers(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	subs, err := a.subscribers.List(r.Context(), envID(r), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, subs)
}

func (a *API) deleteSubscriber(w http.ResponseWriter, r *http.Request) {
	if err := a.subscribers.Delete(r.Context(), envID(r), chi.URLParam(r, "subscriberId")); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) setPreference(w http.ResponseWriter, r *http.Request) {
	var pref domain.SubscriberPreference
	if err := json.NewDecoder(r.Body).Decode(&pref); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.subscribers.SetPreference(r.Context(), envID(r), chi.URLParam(r, "subscriberId"), pref); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pref)
}

func (a *API) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var wf domain.Workflow
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if wf.TriggerID == "" {
		writeError(w, http.StatusBadRequest, "triggerId required")
		return
	}
	if err := a.workflows.Create(r.Context(), envID(r), &wf); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, wf)
}

func (a *API) getWorkflow(w http.ResponseWriter, r *http.Request) {
	wf, err := a.workflows.Get(r.Context(), envID(r), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, wf)
}

func (a *API) listWorkflows(w http.ResponseWriter, r *http.Request) {
	wfs, err := a.workflows.List(r.Context(), envID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, wfs)
}

func (a *API) updateWorkflow(w http.ResponseWriter, r *http.Request) {
	var wf domain.Workflow
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	wf.ID = chi.URLParam(r, "id")
	if err := a.workflows.Update(r.Context(), envID(r), &wf); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, wf)
}

func (a *API) deleteWorkflow(w http.ResponseWriter, r *http.Request) {
	if err := a.workflows.Delete(r.Context(), envID(r), chi.URLParam(r, "id")); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) createTopic(w http.ResponseWriter, r *http.Request) {
	var t domain.Topic
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if t.TopicKey == "" {
		writeError(w, http.StatusBadRequest, "topicKey required")
		return
	}
	if err := a.topics.Create(r.Context(), envID(r), &t); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (a *API) listTopics(w http.ResponseWriter, r *http.Request) {
	topics, err := a.topics.List(r.Context(), envID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, topics)
}

func (a *API) deleteTopic(w http.ResponseWriter, r *http.Request) {
	if err := a.topics.Delete(r.Context(), envID(r), chi.URLParam(r, "topicKey")); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) subscribeTopic(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SubscriberID string `json:"subscriberId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.topics.Subscribe(r.Context(), envID(r), chi.URLParam(r, "topicKey"), body.SubscriberID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) unsubscribeTopic(w http.ResponseWriter, r *http.Request) {
	if err := a.topics.Unsubscribe(r.Context(), envID(r), chi.URLParam(r, "topicKey"), chi.URLParam(r, "subscriberId")); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) createIntegration(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Channel     domain.ChannelType `json:"channel"`
		ProviderID  string             `json:"providerId"`
		Name        string             `json:"name"`
		Credentials map[string]any     `json:"credentials"`
		Primary     bool               `json:"primary"`
		Active      bool               `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	i, err := a.integrations.Create(r.Context(), envID(r), body.Channel, body.ProviderID, body.Name, body.Credentials, body.Primary, body.Active)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, i)
}

func (a *API) listIntegrations(w http.ResponseWriter, r *http.Request) {
	items, err := a.integrations.List(r.Context(), envID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) deleteIntegration(w http.ResponseWriter, r *http.Request) {
	if err := a.integrations.Delete(r.Context(), envID(r), chi.URLParam(r, "id")); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) listNotifications(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := a.notifications.List(r.Context(), envID(r), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) listJobs(w http.ResponseWriter, r *http.Request) {
	items, err := a.notifications.JobsByTransaction(r.Context(), envID(r), chi.URLParam(r, "transactionId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) listLogs(w http.ResponseWriter, r *http.Request) {
	items, err := a.notifications.ExecutionLogs(r.Context(), envID(r), chi.URLParam(r, "transactionId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) listMessages(w http.ResponseWriter, r *http.Request) {
	unread := r.URL.Query().Get("read") == "false"
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := a.messages.List(r.Context(), envID(r), chi.URLParam(r, "subscriberId"), unread, limit, offset)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) patchMessage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Read     *bool `json:"read"`
		Archived *bool `json:"archived"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	msg, err := a.messages.Update(r.Context(), envID(r), chi.URLParam(r, "subscriberId"), chi.URLParam(r, "messageId"), body.Read, body.Archived)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, msg)
}

func (a *API) listProviders(w http.ResponseWriter, r *http.Request) {
	manifests := a.wasmManifest()
	out := make([]map[string]any, 0, len(manifests))
	for _, m := range manifests {
		out = append(out, map[string]any{
			"id":          m.ID,
			"channel":     m.Channel,
			"version":     m.Version,
			"runtime":     "wasm",
			"permissions": m.Permissions,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": out})
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	checks := map[string]string{"status": "ok"}
	status := http.StatusOK

	if err := a.store.Ping(ctx); err != nil {
		checks["database"] = err.Error()
		checks["status"] = "degraded"
		status = http.StatusServiceUnavailable
	} else {
		checks["database"] = "ok"
	}
	if a.redis != nil {
		if err := a.redis.Ping(ctx).Err(); err != nil {
			checks["redis"] = err.Error()
			checks["status"] = "degraded"
			status = http.StatusServiceUnavailable
		} else {
			checks["redis"] = "ok"
		}
	}
	if a.cfg != nil && needsWorkerAPIBridgeHealthCheck(a.cfg.WorkerAPIPubSub) {
		if err := bridge.CheckBackend(ctx, a.cfg); err != nil {
			checks["worker_api_bridge"] = err.Error()
			checks["status"] = "degraded"
			status = http.StatusServiceUnavailable
		} else {
			checks["worker_api_bridge"] = "ok"
		}
	}
	writeJSON(w, status, checks)
}

func needsWorkerAPIBridgeHealthCheck(backend string) bool {
	switch bridge.Backend(backend) {
	case bridge.BackendRabbitMQHTTP, bridge.BackendKafkaHTTP:
		return true
	default:
		return false
	}
}
