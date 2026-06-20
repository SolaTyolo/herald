package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/SolaTyolo/herald/pkg/plugin"
)

const (
	maxWASMMemoryPages = 256 // 16MB
	wasmCallTimeout    = 10 * time.Second
)

type Runtime struct {
	mu       sync.RWMutex
	runtime  wazero.Runtime
	compiled map[string]wazero.CompiledModule
	manifest map[string]Manifest
	dir      string

	host      hostState
	hostOnce  sync.Once
	hostInitErr error
}

type Manifest struct {
	ID          string   `json:"id"`
	Channel     string   `json:"channel"`
	Version     string   `json:"version"`
	Permissions []string `json:"permissions"`
}

func NewRuntime(pluginDir string) *Runtime {
	ctx := context.Background()
	r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithMemoryLimitPages(maxWASMMemoryPages))
	_, _ = wasi_snapshot_preview1.Instantiate(ctx, r)
	return &Runtime{
		runtime:  r,
		compiled: map[string]wazero.CompiledModule{},
		manifest: map[string]Manifest{},
		dir:      pluginDir,
	}
}

func (r *Runtime) LoadFromDir() error {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if err := r.loadPluginDir(filepath.Join(r.dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) loadPluginDir(dir string) error {
	manifestPath := filepath.Join(dir, "provider.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil
	}
	wasmPath := filepath.Join(dir, "provider.wasm")
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	absWasm, err := filepath.Abs(wasmPath)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absDir, absWasm)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("invalid wasm path for plugin")
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	if err := validateManifestPermissions(m.Permissions); err != nil {
		return fmt.Errorf("provider %s: %w", m.ID, err)
	}
	if _, err := os.Stat(absWasm); err != nil {
		return nil
	}
	return r.loadModule(m.ID, absWasm, m)
}

func (r *Runtime) loadModule(id, wasmPath string, m Manifest) error {
	data, err := os.ReadFile(wasmPath)
	if err != nil {
		return err
	}
	ctx := context.Background()
	compiled, err := r.runtime.CompileModule(ctx, data)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.compiled[id] = compiled
	r.manifest[id] = m
	return nil
}

func (r *Runtime) Send(ctx context.Context, providerID string, cfg plugin.Config, req *plugin.SendRequest, callCtx CallContext) (*plugin.SendResult, error) {
	compiled, m, err := r.module(providerID)
	if err != nil {
		return nil, err
	}
	callCtx.ProviderID = providerID
	callCtx.Channel = m.Channel

	callCtxInner, cancel := context.WithTimeout(ctx, wasmCallTimeout)
	defer cancel()

	mod, err := r.instantiateGuest(callCtxInner, compiled, callCtx, m.Permissions)
	if err != nil {
		return nil, err
	}
	defer mod.Close(callCtxInner)

	payload, _ := json.Marshal(map[string]any{
		"config":  cfg,
		"request": req,
		"context": callCtx,
	})
	out, err := callExportJSON(callCtxInner, mod, "Send", payload)
	if err != nil {
		return nil, err
	}
	var result plugin.SendResult
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *Runtime) HasProvider(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.compiled[id]
	return ok
}

type validateResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func (r *Runtime) ValidateConfig(ctx context.Context, providerID string, cfg plugin.Config) error {
	compiled, m, err := r.module(providerID)
	if err != nil {
		return err
	}

	callCtxInner, cancel := context.WithTimeout(ctx, wasmCallTimeout)
	defer cancel()

	cc := CallContext{ProviderID: providerID, Channel: m.Channel}
	mod, err := r.instantiateGuest(callCtxInner, compiled, cc, m.Permissions)
	if err != nil {
		return err
	}
	defer mod.Close(callCtxInner)

	if mod.ExportedFunction("ValidateConfig") == nil {
		return nil
	}

	cfgPayload, _ := json.Marshal(cfg)
	out, err := callExportJSON(callCtxInner, mod, "ValidateConfig", cfgPayload)
	if err != nil {
		return err
	}
	var resp validateResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return err
	}
	if !resp.OK {
		if resp.Error != "" {
			return fmt.Errorf("%s", resp.Error)
		}
		return fmt.Errorf("invalid provider config")
	}
	return nil
}

func (r *Runtime) ListManifests() []Manifest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Manifest, 0, len(r.manifest))
	for _, m := range r.manifest {
		out = append(out, m)
	}
	return out
}

func (r *Runtime) Close(ctx context.Context) error {
	return r.runtime.Close(ctx)
}

func (r *Runtime) module(providerID string) (wazero.CompiledModule, Manifest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	compiled, ok := r.compiled[providerID]
	if !ok {
		return nil, Manifest{}, fmt.Errorf("wasm provider not loaded: %s", providerID)
	}
	return compiled, r.manifest[providerID], nil
}

func callExportJSON(ctx context.Context, mod api.Module, export string, payload []byte) ([]byte, error) {
	fn := mod.ExportedFunction(export)
	if fn == nil {
		return nil, fmt.Errorf("%s export missing", export)
	}
	malloc := mod.ExportedFunction("malloc")
	if malloc == nil {
		return nil, fmt.Errorf("malloc export missing")
	}
	results, err := malloc.Call(ctx, uint64(len(payload)))
	if err != nil {
		return nil, err
	}
	ptr := results[0]
	mem := mod.Memory()
	if !mem.Write(uint32(ptr), payload) {
		return nil, fmt.Errorf("write memory failed")
	}
	out, err := fn.Call(ctx, ptr, uint64(len(payload)))
	if err != nil {
		return nil, err
	}
	if len(out) < 2 {
		return nil, fmt.Errorf("invalid %s return", export)
	}
	outPtr, outLen := uint32(out[0]), uint32(out[1])
	outBytes, ok := mem.Read(outPtr, outLen)
	if !ok {
		return nil, fmt.Errorf("read output failed")
	}
	return outBytes, nil
}
