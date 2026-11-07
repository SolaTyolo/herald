package delivery_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SolaTyolo/herald/internal/delivery"
	"github.com/SolaTyolo/herald/internal/routing"
	"github.com/SolaTyolo/herald/pkg/contract"
)

func TestDeliverDirectSyncRemote(t *testing.T) {
	var gotAsync bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req contract.DeliveryRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotAsync = req.Async || r.Header.Get("X-Herald-Async") == "1"
		_ = json.NewEncoder(w).Encode(contract.OK("biz-direct"))
	}))
	t.Cleanup(srv.Close)

	snap, err := routing.Parse([]byte(`
services:
  dw:
    endpoint: "` + srv.URL + `/delivery"
providers:
  lark_webhook:
    type: remote
    service: dw
    channel: chat
    timeout: 5s
`))
	if err != nil {
		t.Fatal(err)
	}
	svc := delivery.New(nil, routing.NewTable("mem", snap), nil, delivery.Options{AsyncResults: true})
	de := svc.DeliverDirect(context.Background(), contract.DeliveryRequest{
		TaskID:       "t1",
		ProviderName: "lark_webhook",
		Message:      contract.Message{Channel: "chat", Content: "hi"},
	})
	if !de.Success || de.ProviderBizID != "biz-direct" {
		t.Fatalf("de=%+v", de)
	}
	if gotAsync {
		t.Fatal("DeliverDirect must force sync (async=false)")
	}
}

func TestDeliverDirectUnknownProvider(t *testing.T) {
	snap, err := routing.Parse([]byte(`
providers:
  inapp:
    type: builtin
    channel: inapp
`))
	if err != nil {
		t.Fatal(err)
	}
	svc := delivery.New(nil, routing.NewTable("mem", snap), nil, delivery.Options{})
	de := svc.DeliverDirect(context.Background(), contract.DeliveryRequest{ProviderName: "nope"})
	if de.Success || de.Code != "unknown_provider" {
		t.Fatalf("de=%+v", de)
	}
}
