package service_test

import (
	"testing"

	"github.com/SolaTyolo/herald/internal/domain"
)

func TestTriggerTargetTopic(t *testing.T) {
	req := domain.TriggerRequest{
		To: domain.TriggerTarget{Type: "Topic", TopicKey: "news"},
	}
	if req.To.TopicKey != "news" {
		t.Fatal("topic key mismatch")
	}
}

func TestJobStatusConstants(t *testing.T) {
	if domain.JobCompleted != "completed" {
		t.Fatal("unexpected job status")
	}
}
