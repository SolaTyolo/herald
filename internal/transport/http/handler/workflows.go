package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/SolaTyolo/herald/internal/domain"
)

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
