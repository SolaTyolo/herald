package wasm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const hostModuleName = "herald"

type hostState struct {
	mu    sync.Mutex
	call  CallContext
	perms []string
}

func (r *Runtime) ensureHostModule(ctx context.Context) error {
	r.hostOnce.Do(func() {
		_, r.hostInitErr = r.runtime.NewHostModuleBuilder(hostModuleName).
			NewFunctionBuilder().WithFunc(r.hostLog).Export("log").
			NewFunctionBuilder().WithFunc(r.hostGetContext).Export("get_context").
			NewFunctionBuilder().WithFunc(r.hostHTTPFetch).Export("http_fetch").
			Instantiate(ctx)
	})
	return r.hostInitErr
}

func (r *Runtime) bindHostCall(cc CallContext, perms []string) func() {
	r.host.mu.Lock()
	r.host.call = cc
	r.host.perms = perms
	r.host.mu.Unlock()
	return func() {
		r.host.mu.Lock()
		r.host.call = CallContext{}
		r.host.perms = nil
		r.host.mu.Unlock()
	}
}

func (r *Runtime) hostLog(_ context.Context, m api.Module, level, ptr, ln uint32) {
	msg, ok := readGuestString(m, ptr, ln)
	if !ok {
		return
	}
	r.host.mu.Lock()
	provider := r.host.call.ProviderID
	r.host.mu.Unlock()
	switch level {
	case 3:
		slog.Error("wasm provider", "provider", provider, "msg", msg)
	case 2:
		slog.Warn("wasm provider", "provider", provider, "msg", msg)
	case 1:
		slog.Info("wasm provider", "provider", provider, "msg", msg)
	default:
		slog.Debug("wasm provider", "provider", provider, "msg", msg)
	}
}

func (r *Runtime) hostGetContext(_ context.Context, m api.Module, outPtr, outCap uint32) uint32 {
	r.host.mu.Lock()
	cc := r.host.call
	r.host.mu.Unlock()
	raw, err := json.Marshal(cc)
	if err != nil {
		return 0
	}
	if uint32(len(raw)) > outCap {
		return 0
	}
	if !m.Memory().Write(outPtr, raw) {
		return 0
	}
	return uint32(len(raw))
}

type httpFetchRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    json.RawMessage   `json:"body,omitempty"`
}

type httpFetchResponse struct {
	StatusCode int             `json:"statusCode"`
	Body       json.RawMessage `json:"body"`
	Error      string          `json:"error,omitempty"`
}

func (r *Runtime) hostHTTPFetch(ctx context.Context, m api.Module, reqPtr, reqLen, outPtr, outCap uint32) uint32 {
	reqJSON, ok := readGuestBytes(m, reqPtr, reqLen)
	if !ok {
		return writeHTTPResponse(m, outPtr, outCap, httpFetchResponse{Error: "invalid request buffer"})
	}
	var req httpFetchRequest
	if err := json.Unmarshal(reqJSON, &req); err != nil {
		return writeHTTPResponse(m, outPtr, outCap, httpFetchResponse{Error: "invalid request json"})
	}
	if req.Method == "" {
		req.Method = http.MethodGet
	}
	r.host.mu.Lock()
	perms := append([]string(nil), r.host.perms...)
	r.host.mu.Unlock()
	if !allowsNetwork(perms, req.URL) {
		return writeHTTPResponse(m, outPtr, outCap, httpFetchResponse{Error: "network permission denied"})
	}

	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(callCtx, req.Method, req.URL, bytes.NewReader(req.Body))
	if err != nil {
		return writeHTTPResponse(m, outPtr, outCap, httpFetchResponse{Error: err.Error()})
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	if len(req.Body) > 0 && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return writeHTTPResponse(m, outPtr, outCap, httpFetchResponse{Error: err.Error()})
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return writeHTTPResponse(m, outPtr, outCap, httpFetchResponse{Error: err.Error()})
	}
	return writeHTTPResponse(m, outPtr, outCap, httpFetchResponse{
		StatusCode: resp.StatusCode,
		Body:       json.RawMessage(body),
	})
}

func writeHTTPResponse(m api.Module, outPtr, outCap uint32, resp httpFetchResponse) uint32 {
	raw, _ := json.Marshal(resp)
	if uint32(len(raw)) > outCap {
		return 0
	}
	if !m.Memory().Write(outPtr, raw) {
		return 0
	}
	return uint32(len(raw))
}

func readGuestString(m api.Module, ptr, ln uint32) (string, bool) {
	b, ok := readGuestBytes(m, ptr, ln)
	if !ok {
		return "", false
	}
	return string(b), true
}

func readGuestBytes(m api.Module, ptr, ln uint32) ([]byte, bool) {
	if ln == 0 {
		return nil, true
	}
	mem := m.Memory()
	if mem == nil {
		return nil, false
	}
	b, ok := mem.Read(ptr, ln)
	if !ok {
		return nil, false
	}
	out := make([]byte, ln)
	copy(out, b)
	return out, true
}

func (r *Runtime) instantiateGuest(ctx context.Context, compiled wazero.CompiledModule, cc CallContext, perms []string) (api.Module, error) {
	if err := r.ensureHostModule(ctx); err != nil {
		return nil, fmt.Errorf("host module: %w", err)
	}
	unbind := r.bindHostCall(cc, perms)
	defer unbind()
	return r.runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
}
