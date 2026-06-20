package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	addr := env("HTTP_ADDR", ":8091")
	amqpURL := env("AMQP_URL", "amqp://guest:guest@localhost:5672/")
	exchange := env("AMQP_EXCHANGE", "herald.worker-api")
	routingKey := env("AMQP_ROUTING_KEY", "events")
	queue := env("AMQP_QUEUE", "herald.worker-api.events")

	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		slog.Error("amqp dial failed", "err", err)
		os.Exit(1)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		slog.Error("amqp channel failed", "err", err)
		os.Exit(1)
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		slog.Error("declare exchange failed", "err", err)
		os.Exit(1)
	}
	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		slog.Error("declare queue failed", "err", err)
		os.Exit(1)
	}
	if err := ch.QueueBind(queue, routingKey, exchange, false, nil); err != nil {
		slog.Error("bind queue failed", "err", err)
		os.Exit(1)
	}

	deliveries, err := ch.Consume(queue, "herald-inapp-bridge", false, false, false, false, nil)
	if err != nil {
		slog.Error("consume failed", "err", err)
		os.Exit(1)
	}

	hub := newStreamHub()
	go hub.forwardAMQP(deliveries)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/stream", hub.serveStream)

	slog.Info("rabbit in-app bridge listening", "addr", addr, "exchange", exchange, "queue", queue)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("http server failed", "err", err)
		os.Exit(1)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type streamHub struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

func newStreamHub() *streamHub {
	return &streamHub{clients: map[chan []byte]struct{}{}}
}

func (h *streamHub) forwardAMQP(deliveries <-chan amqp.Delivery) {
	for d := range deliveries {
		h.broadcast(d.Body)
		_ = d.Ack(false)
	}
}

func (h *streamHub) broadcast(payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- payload:
		default:
		}
	}
}

func (h *streamHub) serveStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan []byte, 64)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.clients, ch)
		close(ch)
		h.mu.Unlock()
	}()

	ctx := r.Context()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			_, _ = w.Write([]byte(": ping\n\n"))
			flusher.Flush()
		case msg, open := <-ch:
			if !open {
				return
			}
			if !json.Valid(msg) {
				continue
			}
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(msg)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}
