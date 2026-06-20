package bridge

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/SolaTyolo/herald/internal/config"
	"github.com/SolaTyolo/herald/internal/domain"
)

type kafkaHTTPPublisher struct {
	baseURL string
	topic   string
}

func NewKafkaHTTPPublisher(cfg *config.Config) Publisher {
	return &kafkaHTTPPublisher{
		baseURL: trimHTTPBase(cfg.WorkerAPIKafkaHTTPURL),
		topic:   cfg.WorkerAPIKafkaTopic,
	}
}

func (p *kafkaHTTPPublisher) Publish(subscriberPK string, msg *domain.Message) {
	ev := Event{SubscriberPK: subscriberPK, Message: msg}
	body := map[string]any{
		"records": []map[string]any{{
			"key":   subscriberPK,
			"value": ev,
		}},
	}
	u := fmt.Sprintf("%s/topics/%s", p.baseURL, url.PathEscape(p.topic))
	err := postJSON(context.Background(), u, body, map[string]string{
		"Content-Type": "application/vnd.kafka.json.v2+json",
	}, nil)
	if err != nil {
		slog.Warn("worker-api bridge publish failed", "backend", "kafka-http", "err", err)
	}
}

type kafkaHTTPSubscriber struct {
	baseURL string
	topic   string
	cancel  context.CancelFunc
}

func NewKafkaHTTPSubscriber(cfg *config.Config) Subscriber {
	return &kafkaHTTPSubscriber{
		baseURL: trimHTTPBase(cfg.WorkerAPIKafkaHTTPURL),
		topic:   cfg.WorkerAPIKafkaTopic,
	}
}

func (s *kafkaHTTPSubscriber) Run(ctx context.Context, handler EventHandler) {
	if handler == nil {
		return
	}
	ctx, s.cancel = context.WithCancel(ctx)
	go func() {
		partition := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			u := fmt.Sprintf("%s/topics/%s/partitions/%d/records?timeout=30000&max_bytes=1048576",
				s.baseURL, url.PathEscape(s.topic), partition)
			var records []struct {
				Value Event `json:"value"`
			}
			err := getJSON(ctx, u, map[string]string{
				"Accept": "application/vnd.kafka.json.v2+json",
			}, nil, &records)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				slog.Warn("worker-api bridge consume failed", "backend", "kafka-http", "err", err)
				time.Sleep(time.Second)
				continue
			}
			for _, rec := range records {
				ev := rec.Value
				if ev.SubscriberPK != "" && ev.Message != nil {
					handler(ev)
				}
			}
		}
	}()
}

func (s *kafkaHTTPSubscriber) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}

func checkKafkaHTTP(ctx context.Context, cfg *config.Config) error {
	base := trimHTTPBase(cfg.WorkerAPIKafkaHTTPURL)
	u := base + "/brokers"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		metaURL := fmt.Sprintf("%s/topics/%s", base, url.PathEscape(cfg.WorkerAPIKafkaTopic))
		return pingURL(ctx, metaURL)
	}
	resp.Body.Close()
	if resp.StatusCode/100 == 2 {
		return nil
	}
	return fmt.Errorf("kafka http brokers: %s", resp.Status)
}

func pingURL(ctx context.Context, u string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("ping %s: %s", u, resp.Status)
	}
	return nil
}
