package routing

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// healthState tracks which services are currently considered unhealthy.
type healthState struct {
	mu   sync.RWMutex
	down map[string]bool // service name -> unhealthy
}

func newHealthState() *healthState {
	return &healthState{down: make(map[string]bool)}
}

func (h *healthState) set(name string, healthy bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if healthy {
		delete(h.down, name)
	} else {
		h.down[name] = true
	}
}

func (h *healthState) isHealthy(name string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return !h.down[name]
}

// IsServiceHealthy reports whether a named service passes health checks.
// Services without healthCheck config are always healthy.
func (t *Table) IsServiceHealthy(name string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.snap == nil {
		return false
	}
	svc, ok := t.snap.Services[name]
	if !ok {
		return false
	}
	if svc.HealthCheck == nil {
		return true
	}
	if t.health == nil {
		return true
	}
	return t.health.isHealthy(name)
}

// StartHealthChecks polls service healthCheck endpoints until ctx is cancelled.
func (t *Table) StartHealthChecks(ctx context.Context) {
	t.mu.Lock()
	if t.health == nil {
		t.health = newHealthState()
	}
	t.mu.Unlock()

	go t.healthLoop(ctx)
}

func (t *Table) healthLoop(ctx context.Context) {
	client := &http.Client{}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	lastProbe := map[string]time.Time{}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snap := t.Snapshot()
			if snap == nil {
				continue
			}
			now := time.Now()
			for name, svc := range snap.Services {
				if svc.HealthCheck == nil {
					t.health.set(name, true)
					continue
				}
				interval := svc.HealthCheck.Interval
				if interval <= 0 {
					interval = 5 * time.Second
				}
				if since, ok := lastProbe[name]; ok && now.Sub(since) < interval {
					continue
				}
				lastProbe[name] = now
				ok := probeService(ctx, client, svc)
				was := t.health.isHealthy(name)
				t.health.set(name, ok)
				if ok != was {
					slog.Info("service health changed", "service", name, "healthy", ok)
				}
			}
		}
	}
}

func probeService(ctx context.Context, client *http.Client, svc Service) bool {
	hc := svc.HealthCheck
	timeout := hc.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	u, err := healthURL(svc.Endpoint, hc.Path)
	if err != nil {
		return false
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, u, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func healthURL(endpoint, path string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	if path == "" {
		path = "/health"
	}
	u.Path = path
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}
