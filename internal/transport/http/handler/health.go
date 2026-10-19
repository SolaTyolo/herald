package handler

import "net/http"

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
	writeJSON(w, status, checks)
}
