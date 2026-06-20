package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/SolaTyolo/herald/pkg/contract"
)

const maxDeliveryBody = 1 << 20 // 1 MiB

var (
	mu        sync.RWMutex
	providers = map[string]contract.SendProvider{}
	resultRDB *redis.Client
)

// RegisterProvider registers a SendProvider under providerName.
// Must be called before Run.
func RegisterProvider(name string, p contract.SendProvider) {
	mu.Lock()
	defer mu.Unlock()
	providers[name] = p
}

// ConfigureResultRedis sets the Redis address used to publish DeliveryResultEvent.
// Defaults to HERALD_RESULT_REDIS, then REDIS_ADDR.
func ConfigureResultRedis(addr string) {
	if addr == "" {
		return
	}
	resultRDB = redis.NewClient(&redis.Options{Addr: addr})
}

func ensureResultRedis() {
	if resultRDB != nil {
		return
	}
	addr := os.Getenv("HERALD_RESULT_REDIS")
	if addr == "" {
		addr = os.Getenv("REDIS_ADDR")
	}
	if addr == "" {
		return
	}
	resultRDB = redis.NewClient(&redis.Options{Addr: addr})
}

// Run starts an HTTP server with POST /delivery and GET /health.
func Run(addr string) error {
	return RunContext(context.Background(), addr)
}

// RunContext starts the HTTP server and shuts down when ctx is cancelled.
func RunContext(ctx context.Context, addr string) error {
	ensureResultRedis()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /delivery", handleDelivery)

	srv := &http.Server{Addr: addr, Handler: mux}
	errCh := make(chan error, 1)
	go func() {
		slog.Info("herald worker sdk listening", "addr", addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		err := <-errCh
		if err == http.ErrServerClosed {
			return ctx.Err()
		}
		return err
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func handleDelivery(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxDeliveryBody)
	defer r.Body.Close()

	var req contract.DeliveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err == io.EOF {
			writeDelivery(w, contract.Fail("bad_request", "empty body", false))
			return
		}
		writeDelivery(w, contract.Fail("bad_request", err.Error(), false))
		return
	}
	if req.ProviderName == "" {
		writeDelivery(w, contract.Fail("bad_request", "providerName required", false))
		return
	}

	async := req.Async || r.Header.Get("X-Herald-Async") == "1"
	if async && req.TaskID == "" {
		writeDelivery(w, contract.Fail("bad_request", "taskId required for async delivery", false))
		return
	}

	mu.RLock()
	p, ok := providers[req.ProviderName]
	mu.RUnlock()
	if !ok {
		writeDelivery(w, contract.Fail("unknown_provider", fmt.Sprintf("provider %q not registered", req.ProviderName), false))
		return
	}

	if async {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"accepted":true}`))
		go func() {
			result := safeSend(context.Background(), p, req.ProviderConfig, req.Message)
			publishResult(req, result)
		}()
		return
	}

	result := safeSend(r.Context(), p, req.ProviderConfig, req.Message)
	publishResult(req, result)
	writeDelivery(w, result)
}

func publishResult(req contract.DeliveryRequest, de contract.DeliveryError) {
	if req.TaskID == "" || resultRDB == nil {
		return
	}
	ev := contract.DeliveryResultEvent{
		TaskID:        req.TaskID,
		ProviderName:  req.ProviderName,
		TraceID:       req.TraceID,
		DeliveryError: de,
		Context:       req.Context,
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		slog.Warn("result marshal failed", "err", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	key := "herald:delivery:result:" + req.TaskID
	pipe := resultRDB.Pipeline()
	pipe.LPush(ctx, key, raw)
	pipe.Expire(ctx, key, 15*time.Minute)
	pipe.LPush(ctx, "herald:delivery:results", raw)
	pipe.LTrim(ctx, "herald:delivery:results", 0, 9999)
	if _, err := pipe.Exec(ctx); err != nil {
		slog.Warn("result queue publish failed", "taskId", req.TaskID, "err", err)
	}
}

func safeSend(ctx context.Context, p contract.SendProvider, cfg contract.ProviderConfig, msg contract.Message) (result contract.DeliveryError) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("provider panic", "err", rec)
			result = contract.Fail("provider_panic", fmt.Sprintf("%v", rec), false)
		}
	}()
	return p.Send(ctx, cfg, msg)
}

func writeDelivery(w http.ResponseWriter, de contract.DeliveryError) {
	w.Header().Set("Content-Type", "application/json")
	status := http.StatusOK
	if !de.Success && de.Retryable {
		status = http.StatusServiceUnavailable
	} else if !de.Success {
		status = http.StatusUnprocessableEntity
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(de)
}
