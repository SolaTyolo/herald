package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/SolaTyolo/herald/internal/domain"
)

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
