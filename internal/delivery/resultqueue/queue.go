package resultqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/SolaTyolo/herald/pkg/contract"
)

const (
	// AuditList is a capped list of recent results for observability.
	AuditList = "herald:delivery:results"
	// taskKeyPrefix holds per-task wait lists: herald:delivery:result:{taskId}
	taskKeyPrefix = "herald:delivery:result:"
	taskTTL       = 15 * time.Minute
	auditMaxLen   = int64(10000)
)

// Queue publishes and waits for delivery results over Redis.
type Queue struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Queue {
	return &Queue{rdb: rdb}
}

func taskKey(taskID string) string {
	return taskKeyPrefix + taskID
}

// Publish writes the result for waiters and appends to the audit list.
func (q *Queue) Publish(ctx context.Context, ev contract.DeliveryResultEvent) error {
	if q == nil || q.rdb == nil {
		return fmt.Errorf("result queue not configured")
	}
	if ev.TaskID == "" {
		return fmt.Errorf("taskId required")
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	key := taskKey(ev.TaskID)
	pipe := q.rdb.Pipeline()
	pipe.LPush(ctx, key, raw)
	pipe.Expire(ctx, key, taskTTL)
	pipe.LPush(ctx, AuditList, raw)
	pipe.LTrim(ctx, AuditList, 0, auditMaxLen-1)
	_, err = pipe.Exec(ctx)
	return err
}

// Wait blocks until a result for taskID arrives or timeout elapses.
func (q *Queue) Wait(ctx context.Context, taskID string, timeout time.Duration) (contract.DeliveryResultEvent, error) {
	var zero contract.DeliveryResultEvent
	if q == nil || q.rdb == nil {
		return zero, fmt.Errorf("result queue not configured")
	}
	if taskID == "" {
		return zero, fmt.Errorf("taskId required")
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	deadline, ok := ctx.Deadline()
	if ok {
		if rem := time.Until(deadline); rem > 0 && rem < timeout {
			timeout = rem
		}
	}

	res, err := q.rdb.BRPop(ctx, timeout, taskKey(taskID)).Result()
	if err == redis.Nil {
		return zero, fmt.Errorf("result wait timeout for task %s", taskID)
	}
	if err != nil {
		return zero, err
	}
	if len(res) < 2 {
		return zero, fmt.Errorf("empty result payload")
	}
	var ev contract.DeliveryResultEvent
	if err := json.Unmarshal([]byte(res[1]), &ev); err != nil {
		return zero, err
	}
	return ev, nil
}
