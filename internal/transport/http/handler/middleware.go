package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/SolaTyolo/herald/internal/repository"
	"github.com/SolaTyolo/herald/internal/telemetry"
)

func AuthMiddleware(store repository.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}
			auth := r.Header.Get("Authorization")
			if auth == "" {
				writeError(w, http.StatusUnauthorized, "missing authorization")
				return
			}
			key := strings.TrimPrefix(auth, "Bearer ")
			key = strings.TrimPrefix(key, "ApiKey ")
			env, err := store.ValidateAPIKey(r.Context(), key)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid api key")
				return
			}
			ctx := context.WithValue(r.Context(), envKey{}, env.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func Mount(r chi.Router, store repository.Store, api *API) {
	r.Use(AuthMiddleware(store))
	api.Routes(r)
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		route := r.URL.Path
		if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
			route = rc.RoutePattern()
		}
		telemetry.RecordHTTPRequest(r.Context(), r.Method, route, ww.Status(), time.Since(start))
	})
}
