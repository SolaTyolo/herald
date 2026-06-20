package workflow

import (
	"testing"
	"time"

	"github.com/SolaTyolo/herald/internal/domain"
)

func TestDurationFromMeta(t *testing.T) {
	d := durationFromMeta(domain.StepMetadata{DelayAmount: 2, DelayUnit: "minutes"})
	if d != 2*time.Minute {
		t.Fatalf("got %v", d)
	}
}

func TestDelayCompletesWhenScheduledAtPassed(t *testing.T) {
	past := time.Now().Add(-time.Second)
	job := &domain.Job{ScheduledAt: &past}
	if job.ScheduledAt == nil || time.Now().Before(*job.ScheduledAt) {
		t.Fatal("expected delay window to be elapsed")
	}
}

func TestResolveChannelPreference(t *testing.T) {
	email := domain.ChannelEmail
	wfOff := domain.ChannelPrefs{email: false}

	if resolveChannelPreference(false, wfOff, email, nil) {
		t.Fatal("workflow default off should skip")
	}
	if !resolveChannelPreference(true, wfOff, email, nil) {
		t.Fatal("critical should bypass preferences")
	}
	subOn := []domain.SubscriberPreference{{Channel: email, Enabled: true}}
	if !resolveChannelPreference(false, wfOff, email, subOn) {
		t.Fatal("subscriber opt-in should override workflow default")
	}
	subOff := []domain.SubscriberPreference{{Channel: email, Enabled: false}}
	if resolveChannelPreference(false, nil, email, subOff) {
		t.Fatal("subscriber opt-out should skip")
	}
}

func TestEvaluateFilters(t *testing.T) {
	e := &Engine{}
	jc := &JobContext{
		Notification: &domain.Notification{Payload: []byte(`{"status":"paid"}`)},
		Step: &domain.WorkflowStep{
			Filters: []domain.StepFilter{{Field: "status", Operator: "eq", Value: "paid"}},
		},
	}
	if !e.evaluateFilters(jc) {
		t.Fatal("filter should pass")
	}
	jc.Notification.Payload = []byte(`{"status":"pending"}`)
	if e.evaluateFilters(jc) {
		t.Fatal("filter should fail")
	}
}
