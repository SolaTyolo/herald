package bridge

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/redis/go-redis/v9"

	"github.com/SolaTyolo/herald/internal/domain"
)

type redisPublisher struct {
	client *redis.Client
}

func NewRedisPublisher(client *redis.Client) Publisher {
	return &redisPublisher{client: client}
}

func (p *redisPublisher) Publish(subscriberPK string, msg *domain.Message) {
	raw, err := json.Marshal(Event{SubscriberPK: subscriberPK, Message: msg})
	if err != nil {
		return
	}
	if err := p.client.Publish(context.Background(), DefaultChannel, raw).Err(); err != nil {
		slog.Warn("worker-api bridge publish failed", "backend", "redis", "err", err)
	}
}

type redisSubscriber struct {
	client *redis.Client
	cancel context.CancelFunc
}

func NewRedisSubscriber(client *redis.Client) Subscriber {
	return &redisSubscriber{client: client}
}

func (s *redisSubscriber) Run(ctx context.Context, handler EventHandler) {
	if handler == nil {
		return
	}
	ctx, s.cancel = context.WithCancel(ctx)
	go func() {
		sub := s.client.Subscribe(ctx, DefaultChannel)
		defer sub.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-sub.Channel():
				if !ok {
					return
				}
				var ev Event
				if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
					slog.Warn("worker-api bridge decode failed", "backend", "redis", "err", err)
					continue
				}
				if ev.SubscriberPK != "" && ev.Message != nil {
					handler(ev)
				}
			}
		}
	}()
}

func (s *redisSubscriber) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}
