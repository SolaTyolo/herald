package routing

import (
	"fmt"
	"log/slog"
	"sort"
	"sync"
)

// Table is a hot-reloadable provider routing table.
type Table struct {
	mu     sync.RWMutex
	snap   *Snapshot
	path   string
	health *healthState
}

func NewTable(path string, snap *Snapshot) *Table {
	return &Table{path: path, snap: snap, health: newHealthState()}
}

func Open(path string) (*Table, error) {
	snap, err := LoadFile(path)
	if err != nil {
		return nil, err
	}
	return NewTable(path, snap), nil
}

func (t *Table) Path() string { return t.path }

func (t *Table) Snapshot() *Snapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.snap
}

func (t *Table) Lookup(providerName string) (Provider, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.snap == nil {
		return Provider{}, false
	}
	p, ok := t.snap.Providers[providerName]
	return p, ok
}

func (t *Table) HasProvider(providerName string) bool {
	_, ok := t.Lookup(providerName)
	return ok
}

// ProvidersForChannel returns providers whose YAML channel matches, sorted by name.
// inapp is never looked up here — SendChannel handles it as a core builtin.
func (t *Table) ProvidersForChannel(channel string) []Provider {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.snap == nil {
		return nil
	}
	out := make([]Provider, 0)
	for _, p := range t.snap.Providers {
		if p.Channel == channel {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (t *Table) ListProviders() []Provider {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.snap == nil {
		return nil
	}
	out := make([]Provider, 0, len(t.snap.Providers))
	for _, p := range t.snap.Providers {
		out = append(out, p)
	}
	return out
}

func (t *Table) Service(name string) (Service, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.snap == nil {
		return Service{}, false
	}
	s, ok := t.snap.Services[name]
	return s, ok
}

// Reload loads path and swaps the snapshot on success.
// Invalid configs are rejected; the previous snapshot is kept.
func (t *Table) Reload() error {
	snap, err := LoadFile(t.path)
	if err != nil {
		return err
	}
	t.mu.Lock()
	t.snap = snap
	t.mu.Unlock()
	slog.Info("providers routing reloaded", "path", t.path, "providers", len(snap.Providers), "services", len(snap.Services))
	return nil
}

func (t *Table) Replace(snap *Snapshot) error {
	if snap == nil {
		return fmt.Errorf("nil snapshot")
	}
	t.mu.Lock()
	t.snap = snap
	t.mu.Unlock()
	return nil
}
