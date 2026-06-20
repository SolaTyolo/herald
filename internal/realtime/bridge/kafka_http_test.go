package bridge_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SolaTyolo/herald/internal/config"
	"github.com/SolaTyolo/herald/internal/domain"
	"github.com/SolaTyolo/herald/internal/realtime/bridge"
)

func TestKafkaHTTPPubSub(t *testing.T) {
	gotBodyCh := make(chan []byte, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/topics/"):
			body, _ := io.ReadAll(r.Body)
			gotBodyCh <- body
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/records"):
			ev := bridge.Event{
				SubscriberPK: "sub-1",
				Message:      &domain.Message{ID: "m1", Title: "hi"},
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{{"value": ev}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cfg := &config.Config{
		WorkerAPIPubSub:       "kafka-http",
		WorkerAPIKafkaHTTPURL: srv.URL,
		WorkerAPIKafkaTopic:   "herald.worker-api.events",
	}

	gotCh := make(chan bridge.Event, 1)
	sub := bridge.NewKafkaHTTPSubscriber(cfg)
	ctx := t.Context()
	sub.Run(ctx, func(ev bridge.Event) { gotCh <- ev })
	defer sub.Close()

	pub := bridge.NewKafkaHTTPPublisher(cfg)
	pub.Publish("sub-1", &domain.Message{ID: "m1", EnvID: "env-1", Title: "hi"})

	gotBody := <-gotBodyCh
	if !strings.Contains(string(gotBody), "sub-1") {
		t.Fatalf("publish body missing key: %s", gotBody)
	}

	select {
	case got := <-gotCh:
		if got.Message == nil || got.Message.ID != "m1" {
			t.Fatalf("got %+v", got.Message)
		}
	case <-timeAfter(t, 2*time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestRabbitMQHTTPPublish(t *testing.T) {
	type publishCapture struct {
		auth    string
		payload string
	}
	gotCh := make(chan publishCapture, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotCh <- publishCapture{
			auth:    r.Header.Get("Authorization"),
			payload: string(body),
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &config.Config{
		WorkerAPIPubSub:             "rabbitmq-http",
		WorkerAPIRabbitMQHTTPURL:    srv.URL,
		WorkerAPIRabbitMQUser:       "herald",
		WorkerAPIRabbitMQPass:       "secret",
		WorkerAPIRabbitMQVHost:      "/",
		WorkerAPIRabbitMQExchange:   "herald.worker-api",
		WorkerAPIRabbitMQRoutingKey: "events",
		WorkerAPIRabbitMQStreamURL:  "http://localhost:8091/stream",
	}

	pub := bridge.NewRabbitMQHTTPPublisher(cfg)
	pub.Publish("sub-1", &domain.Message{ID: "m1", EnvID: "env-1", Title: "hello"})

	got := <-gotCh
	if got.auth == "" {
		t.Fatal("expected basic auth")
	}
	if !strings.Contains(got.payload, "sub-1") {
		t.Fatalf("payload: %s", got.payload)
	}
}

func TestLocalBridge(t *testing.T) {
	// use internal local bus via OpenPublisher with local config
	cfg := &config.Config{WorkerAPIPubSub: "local"}
	pub, err := bridge.OpenPublisher(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	sub, err := bridge.OpenSubscriber(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}

	gotCh := make(chan bridge.Event, 1)
	ctx := t.Context()
	sub.Run(ctx, func(ev bridge.Event) { gotCh <- ev })
	defer sub.Close()

	msg := &domain.Message{ID: "msg-1", Title: "hello"}
	pub.Publish("sub-1", msg)

	select {
	case got := <-gotCh:
		if got.Message == nil || got.Message.ID != msg.ID {
			t.Fatalf("got %+v", got.Message)
		}
	case <-timeAfter(t, 2*time.Second):
		t.Fatal("timeout")
	}
}

func timeAfter(t *testing.T, d time.Duration) <-chan time.Time {
	t.Helper()
	return time.After(d)
}
