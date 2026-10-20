package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (a *API) listMessages(w http.ResponseWriter, r *http.Request) {
	unread := r.URL.Query().Get("read") == "false"
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := a.messages.List(r.Context(), chi.URLParam(r, "subscriberId"), unread, limit, offset)
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
	msg, err := a.messages.Update(r.Context(), chi.URLParam(r, "subscriberId"), chi.URLParam(r, "messageId"), body.Read, body.Archived)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, msg)
}
