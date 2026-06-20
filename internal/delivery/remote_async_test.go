package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/SolaTyolo/herald/internal/delivery/resultqueue"
	"github.com/SolaTyolo/herald/internal/routing"
	"github.com/SolaTyolo/herald/pkg/contract"
)

func TestRemoteAsyncWaitOnResultQueue(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	q := resultqueue.New(rdb)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Herald-Async") != "1" {
			t.Fatal("missing async header")
		}
		var req contract.DeliveryRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		go func() {
			time.Sleep(30 * time.Millisecond)
			_ = q.Publish(context.Background(), contract.DeliveryResultEvent{
				TaskID:        req.TaskID,
				ProviderName:  req.ProviderName,
				DeliveryError: contract.OK("async-biz"),
			})
		}()
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"accepted":true}`))
	}))
	defer srv.Close()

	c := newRemoteClient(true, q)
	de := c.Send(context.Background(), routing.Provider{
		Name: "echo", Type: routing.TypeRemote,
		Endpoint: srv.URL + "/delivery", Timeout: 2 * time.Second,
	}, contract.DeliveryRequest{
		TaskID:       "job-async-1",
		ProviderName: "echo",
		Message:      contract.Message{Channel: "sms", To: "+1", Content: "hi"},
	})
	if !de.Success || de.ProviderBizID != "async-biz" {
		t.Fatalf("%+v", de)
	}
}
