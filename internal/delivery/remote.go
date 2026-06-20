package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/SolaTyolo/herald/internal/delivery/resultqueue"
	"github.com/SolaTyolo/herald/internal/routing"
	"github.com/SolaTyolo/herald/pkg/contract"
)

type remoteClient struct {
	http   *http.Client
	async  bool
	results *resultqueue.Queue
}

func newRemoteClient(async bool, results *resultqueue.Queue) *remoteClient {
	// Timeout: 0 — rely on per-request context (YAML / providerConfig timeout).
	return &remoteClient{
		http:    &http.Client{Timeout: 0},
		async:   async,
		results: results,
	}
}

func (c *remoteClient) Send(ctx context.Context, route routing.Provider, req contract.DeliveryRequest) contract.DeliveryError {
	ctx, span := otel.Tracer("herald-delivery").Start(ctx, "delivery.remote")
	defer span.End()
	span.SetAttributes(
		attribute.String("herald.provider", req.ProviderName),
		attribute.String("herald.endpoint", route.Endpoint),
		attribute.Bool("herald.async", c.async),
	)

	timeout := route.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if req.ProviderConfig.TimeoutSec > 0 {
		timeout = time.Duration(req.ProviderConfig.TimeoutSec) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if c.async {
		req.Async = true
	}

	body, err := json.Marshal(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return contract.Fail("encode_error", err.Error(), false)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, route.Endpoint, bytes.NewReader(body))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return contract.Fail("http_error", err.Error(), true)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if req.TraceID != "" {
		httpReq.Header.Set("X-Trace-Id", req.TraceID)
	}
	if c.async {
		httpReq.Header.Set("X-Herald-Async", "1")
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return contract.Fail("http_error", err.Error(), true)
	}
	defer resp.Body.Close()
	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

	// Async path: worker accepted the job; wait on Redis result queue.
	if c.async && (resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusOK) {
		if req.TaskID == "" {
			return contract.Fail("async_missing_task", "taskId required for async delivery", false)
		}
		if c.results == nil {
			return contract.Fail("async_no_queue", "result queue not configured", true)
		}
		// Drain HTTP body if worker also returned inline JSON (sync-compatible workers).
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if resp.StatusCode == http.StatusOK && len(data) > 0 {
			var de contract.DeliveryError
			if err := json.Unmarshal(data, &de); err == nil && (de.Success || de.Code != "" || de.Message != "") {
				return de
			}
		}
		ev, err := c.results.Wait(ctx, req.TaskID, timeout)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return contract.Fail("result_wait", err.Error(), true)
		}
		return ev.DeliveryError
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return contract.Fail("http_error", err.Error(), true)
	}

	var de contract.DeliveryError
	if err := json.Unmarshal(data, &de); err != nil {
		if resp.StatusCode >= 500 {
			return contract.Fail("worker_error", fmt.Sprintf("status %d: %s", resp.StatusCode, truncate(string(data), 200)), true)
		}
		return contract.Fail("worker_bad_response", fmt.Sprintf("status %d: invalid DeliveryError JSON", resp.StatusCode), false)
	}
	if resp.StatusCode >= 500 && !de.Success {
		de.Retryable = true
	}
	return de
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func credentialsToRaw(cfg map[string]any) map[string]string {
	out := make(map[string]string, len(cfg))
	for k, v := range cfg {
		switch t := v.(type) {
		case string:
			out[k] = t
		case json.Number:
			out[k] = t.String()
		case float64:
			out[k] = fmt.Sprintf("%v", t)
		case bool:
			out[k] = fmt.Sprintf("%t", t)
		case nil:
			out[k] = ""
		default:
			b, err := json.Marshal(t)
			if err != nil {
				out[k] = fmt.Sprintf("%v", t)
			} else {
				out[k] = string(b)
			}
		}
	}
	return out
}
