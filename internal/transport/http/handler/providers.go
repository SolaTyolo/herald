package handler

import (
	"net/http"

	"github.com/SolaTyolo/herald/internal/routing"
)

func (a *API) listProviders(w http.ResponseWriter, r *http.Request) {
	providers := a.routes.ListProviders()
	out := make([]map[string]any, 0, len(providers))
	for _, p := range providers {
		item := map[string]any{
			"id":      p.Name,
			"channel": p.Channel,
			"type":    p.Type,
		}
		if p.Type == routing.TypeRemote {
			item["service"] = p.Service
			item["endpoint"] = p.Endpoint
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": out})
}
