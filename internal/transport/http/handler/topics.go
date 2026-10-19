package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/SolaTyolo/herald/internal/domain"
)

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
