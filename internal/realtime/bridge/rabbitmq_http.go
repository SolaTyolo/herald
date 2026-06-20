package bridge

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/SolaTyolo/herald/internal/config"
	"github.com/SolaTyolo/herald/internal/domain"
)

type rabbitMQHTTPPublisher struct {
	cfg *config.Config
}

func NewRabbitMQHTTPPublisher(cfg *config.Config) Publisher {
	return &rabbitMQHTTPPublisher{cfg: cfg}
}

func (p *rabbitMQHTTPPublisher) Publish(subscriberPK string, msg *domain.Message) {
	raw, err := json.Marshal(Event{SubscriberPK: subscriberPK, Message: msg})
	if err != nil {
		return
	}
	vhost := p.cfg.WorkerAPIRabbitMQVHost
	if vhost == "" {
		vhost = "/"
	}
	u := fmt.Sprintf("%s/api/exchanges/%s/%s/publish",
		trimHTTPBase(p.cfg.WorkerAPIRabbitMQHTTPURL),
		url.PathEscape(vhost),
		url.PathEscape(p.cfg.WorkerAPIRabbitMQExchange),
	)
	body := map[string]any{
		"properties":       map[string]any{},
		"routing_key":      p.cfg.WorkerAPIRabbitMQRoutingKey,
		"payload":          string(raw),
		"payload_encoding": "string",
	}
	err = postJSON(context.Background(), u, body, nil, rabbitAuth(p.cfg))
	if err != nil {
		slog.Warn("worker-api bridge publish failed", "backend", "rabbitmq-http", "err", err)
	}
}

type rabbitMQHTTPSubscriber struct {
	streamURL string
	cancel    context.CancelFunc
}

func NewRabbitMQHTTPSubscriber(cfg *config.Config) Subscriber {
	stream := strings.TrimRight(cfg.WorkerAPIRabbitMQStreamURL, "/")
	if !strings.HasSuffix(stream, "/stream") {
		stream += "/stream"
	}
	return &rabbitMQHTTPSubscriber{streamURL: stream}
}

func (s *rabbitMQHTTPSubscriber) Run(ctx context.Context, handler EventHandler) {
	if handler == nil {
		return
	}
	ctx, s.cancel = context.WithCancel(ctx)
	go func() {
		for {
			if ctx.Err() != nil {
				return
			}
			if err := s.consumeStream(ctx, handler); err != nil {
				if ctx.Err() != nil {
					return
				}
				slog.Warn("worker-api bridge stream disconnected", "backend", "rabbitmq-http", "err", err)
				time.Sleep(time.Second)
			}
		}
	}()
}

func (s *rabbitMQHTTPSubscriber) consumeStream(ctx context.Context, handler EventHandler) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.streamURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("stream: %s", resp.Status)
	}
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var ev Event
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev); err != nil {
			slog.Warn("worker-api bridge decode failed", "backend", "rabbitmq-http", "err", err)
			continue
		}
		if ev.SubscriberPK != "" && ev.Message != nil {
			handler(ev)
		}
	}
	return sc.Err()
}

func (s *rabbitMQHTTPSubscriber) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}

func rabbitAuth(cfg *config.Config) func(*http.Request) {
	return func(r *http.Request) {
		token := base64.StdEncoding.EncodeToString([]byte(cfg.WorkerAPIRabbitMQUser + ":" + cfg.WorkerAPIRabbitMQPass))
		r.Header.Set("Authorization", "Basic "+token)
	}
}

func checkRabbitMQSidecar(ctx context.Context, cfg *config.Config) error {
	base := strings.TrimSuffix(strings.TrimSpace(cfg.WorkerAPIRabbitMQStreamURL), "/stream")
	base = strings.TrimRight(base, "/")
	return pingURL(ctx, base+"/health")
}
