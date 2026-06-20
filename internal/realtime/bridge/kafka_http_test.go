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
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/topics/"):
			gotBody, _ = io.ReadAll(r.Body)
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

	var got bridge.Event
	sub := bridge.NewKafkaHTTPSubscriber(cfg)
	ctx := t.Context()
	sub.Run(ctx, func(ev bridge.Event) { got = ev })
	defer sub.Close()

	pub := bridge.NewKafkaHTTPPublisher(cfg)
	pub.Publish("sub-1", &domain.Message{ID: "m1", EnvID: "env-1", Title: "hi"})

	if !strings.Contains(string(gotBody), "sub-1") {
		t.Fatalf("publish body missing key: %s", gotBody)
	}

	deadline := timeAfter(t, 2*time.Second)
	for got.Message == nil {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for event")
		default:
		}
	}
	if got.Message.ID != "m1" {
		t.Fatalf("got %+v", got.Message)
	}
}

func TestRabbitMQHTTPPublish(t *testing.T) {
	var auth string
	var payload string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		payload = string(body)
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

	if auth == "" {
		t.Fatal("expected basic auth")
	}
	if !strings.Contains(payload, "sub-1") {
		t.Fatalf("payload: %s", payload)
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

	var got bridge.Event
	ctx := t.Context()
	sub.Run(ctx, func(ev bridge.Event) { got = ev })
	defer sub.Close()

	msg := &domain.Message{ID: "msg-1", Title: "hello"}
	pub.Publish("sub-1", msg)

	deadline := timeAfter(t, 2*time.Second)
	for got.Message == nil {
		select {
		case <-deadline:
			t.Fatal("timeout")
		default:
		}
	}
	if got.Message.ID != msg.ID {
		t.Fatalf("got %+v", got.Message)
	}
}

func timeAfter(t *testing.T, d time.Duration) <-chan time.Time {
	t.Helper()
	return time.After(d)
}
