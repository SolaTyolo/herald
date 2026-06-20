package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/SolaTyolo/herald/internal/bootstrap"
	"github.com/SolaTyolo/herald/internal/telemetry"
	"github.com/SolaTyolo/herald/internal/transport/http/handler"
)

func main() {
	ctx := context.Background()
	app, err := bootstrap.New(ctx, bootstrap.RoleAPI)
	if err != nil {
		slog.Error("bootstrap failed", "err", err)
		os.Exit(1)
	}
	defer app.Close()

	r := chi.NewRouter()
	r.Use(chimw.RequestID, chimw.RealIP, chimw.Logger, chimw.Recoverer)
	r.Use(handler.MetricsMiddleware)
	handler.Mount(r, app.Store, app.Handler)

	srv := &http.Server{Addr: app.Config.HTTPAddr, Handler: telemetry.WrapHandler("herald-api", r)}
	go func() {
		slog.Info("api listening", "addr", app.Config.HTTPAddr, "store", app.Config.StoreType)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	_ = srv.Shutdown(context.Background())
}
