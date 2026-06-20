package resultqueue

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/SolaTyolo/herald/pkg/contract"
)

func TestPublishAndWait(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	q := New(rdb)

	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = q.Publish(context.Background(), contract.DeliveryResultEvent{
			TaskID:        "task-1",
			ProviderName:  "echo",
			DeliveryError: contract.OK("biz"),
		})
	}()

	ev, err := q.Wait(context.Background(), "task-1", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !ev.DeliveryError.Success || ev.DeliveryError.ProviderBizID != "biz" {
		t.Fatalf("%+v", ev)
	}
}

func TestWaitTimeout(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	q := New(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	_, err = q.Wait(context.Background(), "missing", 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout")
	}
}
