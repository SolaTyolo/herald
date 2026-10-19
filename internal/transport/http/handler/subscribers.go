package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/SolaTyolo/herald/internal/domain"
)

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
