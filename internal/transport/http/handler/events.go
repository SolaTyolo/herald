package handler

import (
	"encoding/json"
	"net/http"

	"github.com/SolaTyolo/herald/internal/domain"
)

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
