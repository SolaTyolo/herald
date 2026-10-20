package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (a *API) listNotifications(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := a.notifications.List(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) listJobs(w http.ResponseWriter, r *http.Request) {
	items, err := a.notifications.JobsByTransaction(r.Context(), chi.URLParam(r, "transactionId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) listLogs(w http.ResponseWriter, r *http.Request) {
	items, err := a.notifications.ExecutionLogs(r.Context(), chi.URLParam(r, "transactionId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}
