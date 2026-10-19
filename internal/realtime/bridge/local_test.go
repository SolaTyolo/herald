package bridge_test

import (
	"testing"
	"time"

	"github.com/SolaTyolo/herald/internal/config"
	"github.com/SolaTyolo/herald/internal/domain"
	"github.com/SolaTyolo/herald/internal/realtime/bridge"
)

func TestLocalBridge(t *testing.T) {
	cfg := &config.Config{QueueMode: config.QueueModeLocal}
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
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}
