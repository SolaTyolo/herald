package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SolaTyolo/herald/internal/routing"
	"github.com/SolaTyolo/herald/pkg/contract"
)

func TestRemoteClientSend(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/delivery" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		var req contract.DeliveryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.ProviderName != "tencent_sms" || req.Message.To == "" {
			t.Fatalf("bad req: %+v", req)
		}
		if req.Context.WorkflowID != "wf-1" {
			t.Fatalf("missing context: %+v", req.Context)
		}
		_ = json.NewEncoder(w).Encode(contract.OK("biz-1"))
	}))
	defer srv.Close()

	c := newRemoteClient(false, nil)
	de := c.Send(context.Background(), routing.Provider{
		Name: "tencent_sms", Type: routing.TypeRemote,
		Endpoint: srv.URL + "/delivery", Timeout: 5 * time.Second,
	}, contract.DeliveryRequest{
		TaskID:       "job-1",
		ProviderName: "tencent_sms",
		Message:      contract.Message{Channel: "sms", To: "+86138", Content: "hi"},
		ProviderConfig: contract.ProviderConfig{
			TimeoutSec: 5,
			Raw:        map[string]string{"appId": "1"},
		},
		Context: contract.DeliveryContext{WorkflowID: "wf-1", NotificationID: "n-1"},
	})
	if !de.Success || de.ProviderBizID != "biz-1" {
		t.Fatalf("%+v", de)
	}
}

func TestCredentialsToRaw(t *testing.T) {
	t.Parallel()
	raw := credentialsToRaw(map[string]any{
		"secretId": "a",
		"appId":    float64(123),
		"nested":   map[string]any{"k": "v"},
	})
	if raw["secretId"] != "a" || raw["appId"] != "123" {
		t.Fatalf("%v", raw)
	}
	if raw["nested"] == "" {
		t.Fatal("nested should be json")
	}
}
