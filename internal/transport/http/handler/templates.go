package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/SolaTyolo/herald/internal/domain"
	"github.com/SolaTyolo/herald/internal/repository"
)

func (a *API) createTemplate(w http.ResponseWriter, r *http.Request) {
	var tmpl domain.NotificationTemplate
	if err := json.NewDecoder(r.Body).Decode(&tmpl); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.templates.Create(r.Context(), envID(r), &tmpl); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, tmpl)
}

func (a *API) listTemplates(w http.ResponseWriter, r *http.Request) {
	items, err := a.templates.List(r.Context(), envID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) getTemplate(w http.ResponseWriter, r *http.Request) {
	tmpl, err := a.templates.Get(r.Context(), envID(r), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tmpl)
}

func (a *API) updateTemplate(w http.ResponseWriter, r *http.Request) {
	var tmpl domain.NotificationTemplate
	if err := json.NewDecoder(r.Body).Decode(&tmpl); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	tmpl.ID = chi.URLParam(r, "id")
	if err := a.templates.Update(r.Context(), envID(r), &tmpl); err != nil {
		status := http.StatusBadRequest
		if repository.IsNotFound(err) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	got, err := a.templates.Get(r.Context(), envID(r), tmpl.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, got)
}

func (a *API) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	if err := a.templates.Delete(r.Context(), envID(r), chi.URLParam(r, "id")); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) createTemplateVersion(w http.ResponseWriter, r *http.Request) {
	var ver domain.TemplateVersion
	if err := json.NewDecoder(r.Body).Decode(&ver); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	templateID := chi.URLParam(r, "id")
	if err := a.templates.CreateVersion(r.Context(), envID(r), templateID, &ver); err != nil {
		status := http.StatusBadRequest
		if repository.IsNotFound(err) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, ver)
}

func (a *API) updateTemplateVersion(w http.ResponseWriter, r *http.Request) {
	var ver domain.TemplateVersion
	if err := json.NewDecoder(r.Body).Decode(&ver); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ver.ID = chi.URLParam(r, "versionId")
	templateID := chi.URLParam(r, "id")
	if err := a.templates.UpdateVersion(r.Context(), envID(r), templateID, &ver); err != nil {
		status := http.StatusBadRequest
		if repository.IsNotFound(err) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ver)
}

func (a *API) deleteTemplateVersion(w http.ResponseWriter, r *http.Request) {
	if err := a.templates.DeleteVersion(r.Context(), envID(r), chi.URLParam(r, "id"), chi.URLParam(r, "versionId")); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) previewTemplate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		VersionID  string         `json:"versionId"`
		Data       map[string]any `json:"data"`
		Subscriber map[string]any `json:"subscriber"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	out, err := a.templates.Preview(r.Context(), envID(r), chi.URLParam(r, "id"), body.VersionID, body.Data, body.Subscriber)
	if err != nil {
		status := http.StatusBadRequest
		if repository.IsNotFound(err) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}
