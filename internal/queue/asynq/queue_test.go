package asynqqueue

import (
	"testing"

	"github.com/hibiken/asynq"
)

func TestParseTopicFanoutPayload(t *testing.T) {
	t.Parallel()
	task := asynq.NewTask(TypeTopicFanout, []byte(`{
		"envId":"e1","transactionId":"t1","workflowId":"w1","topicKey":"news","payload":{}
	}`))
	p, err := ParseTopicFanoutPayload(task)
	if err != nil {
		t.Fatal(err)
	}
	if p.TopicKey != "news" || p.EnvID != "e1" {
		t.Fatalf("%+v", p)
	}

	bad := asynq.NewTask(TypeTopicFanout, []byte(`{"envId":"e1"}`))
	if _, err := ParseTopicFanoutPayload(bad); err == nil {
		t.Fatal("expected error")
	}
}
