package service

import (
	"testing"

	"github.com/SolaTyolo/herald/internal/domain"
)

func TestIsTopicTarget(t *testing.T) {
	t.Parallel()
	if !isTopicTarget(&domain.TriggerRequest{To: domain.TriggerTarget{Type: "Topic", TopicKey: "a"}}) {
		t.Fatal("type Topic")
	}
	if !isTopicTarget(&domain.TriggerRequest{To: domain.TriggerTarget{TopicKey: "a"}}) {
		t.Fatal("topicKey only")
	}
	if isTopicTarget(&domain.TriggerRequest{To: domain.TriggerTarget{SubscriberID: "u1"}}) {
		t.Fatal("subscriber should not be topic")
	}
}
