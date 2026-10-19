package asynqqueue

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/SolaTyolo/herald/internal/delivery"
)

// Tasks is the in-process work executed by a local queue.
type Tasks interface {
	ProcessJob(ctx context.Context, envID, jobID string) error
	FlushDigest(ctx context.Context, envID, bucketID, jobID string) error
	ProcessTopicFanout(ctx context.Context, p TopicFanoutPayload) error
}

// Enqueuer schedules workflow tasks.
type Enqueuer interface {
	EnqueueJob(ctx context.Context, envID, jobID string, processAt *time.Time) error
	EnqueueDigestFlush(ctx context.Context, envID, bucketID, jobID string, at time.Time) error
	EnqueueTopicFanout(ctx context.Context, p TopicFanoutPayload) error
	Close() error
}

// Local runs tasks in-process. It does not use Redis.
type Local struct {
	mu      sync.Mutex
	tasks   Tasks
	timers  []*time.Timer
	stopped bool
	wg      sync.WaitGroup
}

func NewLocal() *Local {
	return &Local{}
}

func (q *Local) Bind(tasks Tasks) {
	q.mu.Lock()
	q.tasks = tasks
	q.mu.Unlock()
}

func (q *Local) EnqueueJob(_ context.Context, envID, jobID string, processAt *time.Time) error {
	return q.schedule(processAt, func(ctx context.Context) error {
		return q.tasks.ProcessJob(ctx, envID, jobID)
	})
}

func (q *Local) EnqueueDigestFlush(_ context.Context, envID, bucketID, jobID string, at time.Time) error {
	return q.schedule(&at, func(ctx context.Context) error {
		return q.tasks.FlushDigest(ctx, envID, bucketID, jobID)
	})
}

func (q *Local) EnqueueTopicFanout(_ context.Context, p TopicFanoutPayload) error {
	return q.schedule(nil, func(ctx context.Context) error {
		return q.tasks.ProcessTopicFanout(ctx, p)
	})
}

func (q *Local) Close() error {
	q.mu.Lock()
	q.stopped = true
	timers := q.timers
	q.timers = nil
	q.mu.Unlock()
	for _, timer := range timers {
		if timer.Stop() {
			q.wg.Done()
		}
	}
	q.wg.Wait()
	return nil
}

func (q *Local) schedule(at *time.Time, fn func(context.Context) error) error {
	delay := time.Duration(0)
	if at != nil {
		delay = time.Until(*at)
		if delay < 0 {
			delay = 0
		}
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.stopped {
		return errQueueClosed
	}
	if q.tasks == nil {
		return errQueueUnbound
	}
	q.wg.Add(1)
	timer := time.AfterFunc(delay, func() {
		defer q.wg.Done()
		q.exec(fn)
	})
	q.timers = append(q.timers, timer)
	return nil
}

func (q *Local) exec(fn func(context.Context) error) {
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		err = fn(context.Background())
		if err == nil || !delivery.IsRetryable(err) {
			if err != nil {
				slog.Error("local queue task failed", "err", err)
			}
			return
		}
		time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
	}
	slog.Error("local queue task exhausted retries", "err", err)
}

var (
	errQueueClosed  = errString("local queue closed")
	errQueueUnbound = errString("local queue is not bound")
)

type errString string

func (e errString) Error() string { return string(e) }
