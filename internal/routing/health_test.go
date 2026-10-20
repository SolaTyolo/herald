package routing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestHealthURL(t *testing.T) {
	t.Parallel()
	u, err := healthURL("http://worker:8080/delivery", "/health")
	if err != nil {
		t.Fatal(err)
	}
	if u != "http://worker:8080/health" {
		t.Fatalf("got %s", u)
	}
}

func TestIsServiceHealthyWithoutHealthCheck(t *testing.T) {
	t.Parallel()
	snap, err := Parse([]byte(`
services:
  w:
    endpoint: "http://w:8080/delivery"
providers:
  foo:
    type: remote
    service: w
    channel: sms
`))
	if err != nil {
		t.Fatal(err)
	}
	table := NewTable("mem", snap)
	if !table.IsServiceHealthy("w") {
		t.Fatal("service without healthCheck should be healthy")
	}
	if table.IsServiceHealthy("missing") {
		t.Fatal("missing should be unhealthy")
	}
}

func TestHealthProbeMarksDown(t *testing.T) {
	t.Parallel()
	var healthy atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !healthy.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	yaml := `
services:
  w:
    endpoint: "` + srv.URL + `/delivery"
    healthCheck:
      path: "/health"
      interval: "50ms"
      timeout: "100ms"
providers:
  foo:
    type: remote
    service: w
    channel: sms
`
	snap, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	// Fix path: healthURL replaces path with /health on same host
	table := NewTable("mem", snap)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	table.StartHealthChecks(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !table.IsServiceHealthy("w") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if table.IsServiceHealthy("w") {
		t.Fatal("expected unhealthy while probe fails")
	}

	healthy.Store(true)
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if table.IsServiceHealthy("w") {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("expected healthy after probe succeeds")
}
