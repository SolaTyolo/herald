package asynqqueue

import (
	"testing"

	"github.com/hibiken/asynq"
)

func TestParseTopicFanoutPayload(t *testing.T) {
	t.Parallel()
	task := asynq.NewTask(TypeTopicFanout, []byte(`{
		"transactionId":"t1","workflowId":"w1","topicKey":"news","payload":{}
	}`))
	p, err := ParseTopicFanoutPayload(task)
	if err != nil {
		t.Fatal(err)
	}
	if p.TopicKey != "news" || p.TransactionID != "t1" {
		t.Fatalf("%+v", p)
	}

	bad := asynq.NewTask(TypeTopicFanout, []byte(`{"topicKey":"news"}`))
	if _, err := ParseTopicFanoutPayload(bad); err == nil {
		t.Fatal("expected error")
	}
}
