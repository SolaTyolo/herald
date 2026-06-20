package bridge

import (
	"context"
	"log/slog"
	"sync"

	"github.com/SolaTyolo/herald/internal/domain"
)

// localBus is an in-process bus for tests and single-process runs.
type localBus struct {
	mu          sync.RWMutex
	subscribers []chan Event
}

var sharedLocalBus = &localBus{}

func (b *localBus) publisher() Publisher {
	return &localPublisher{bus: b}
}

func (b *localBus) subscriber() Subscriber {
	return &localSubscriber{bus: b}
}

func (b *localBus) publish(ev Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subscribers {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (b *localBus) subscribe() chan Event {
	ch := make(chan Event, 64)
	b.mu.Lock()
	b.subscribers = append(b.subscribers, ch)
	b.mu.Unlock()
	return ch
}

func (b *localBus) unsubscribe(ch chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, c := range b.subscribers {
		if c == ch {
			b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

type localPublisher struct {
	bus *localBus
}

func (p *localPublisher) Publish(subscriberPK string, msg *domain.Message) {
	p.bus.publish(Event{SubscriberPK: subscriberPK, Message: msg})
}

type localSubscriber struct {
	bus    *localBus
	cancel context.CancelFunc
}

func (s *localSubscriber) Run(ctx context.Context, handler EventHandler) {
	if handler == nil {
		return
	}
	ctx, s.cancel = context.WithCancel(ctx)
	ch := s.bus.subscribe()
	go func() {
		defer s.bus.unsubscribe(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-ch:
				if !ok {
					return
				}
				handler(ev)
			}
		}
	}()
	slog.Warn("worker-api bridge using local backend; not suitable for separate API/Worker processes")
}

func (s *localSubscriber) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}
