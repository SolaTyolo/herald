package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/SolaTyolo/herald/internal/delivery"
	"github.com/SolaTyolo/herald/internal/domain"
	asynqqueue "github.com/SolaTyolo/herald/internal/queue/asynq"
	"github.com/SolaTyolo/herald/internal/repository"
	"github.com/SolaTyolo/herald/internal/template"
)

type Engine struct {
	store    repository.Store
	queue    *asynqqueue.Client
	delivery *delivery.Service
	redis    *redis.Client
}

func NewEngine(store repository.Store, queue *asynqqueue.Client, delivery *delivery.Service, redisClient *redis.Client) *Engine {
	return &Engine{store: store, queue: queue, delivery: delivery, redis: redisClient}
}

type JobContext struct {
	EnvID         string
	TransactionID string
	Workflow      *domain.Workflow
	Notification  *domain.Notification
	Subscriber    *domain.Subscriber
	Step          *domain.WorkflowStep
}

func (e *Engine) ProcessJob(ctx context.Context, envID, jobID string) error {
	job, err := e.store.GetJob(ctx, jobID)
	if err != nil {
		return err
	}
	if job.Status == domain.JobCompleted || job.Status == domain.JobSkipped || job.Status == domain.JobMerged {
		return nil
	}

	jc, err := e.loadNotificationContext(ctx, job)
	if err != nil {
		return err
	}
	if jc.EnvID != envID {
		return fmt.Errorf("env mismatch: job %s", jobID)
	}

	now := time.Now()
	job.Status = domain.JobRunning
	job.StartedAt = &now
	if err := e.store.UpdateJob(ctx, job); err != nil {
		return err
	}
	e.log(ctx, jc, job, "info", "job started")

	var execErr error
	switch job.StepType {
	case domain.StepDelay:
		execErr = e.execDelay(ctx, jc, job)
	case domain.StepDigest:
		execErr = e.execDigest(ctx, jc, job)
	case domain.StepThrottle:
		execErr = e.execThrottle(ctx, jc, job)
	default:
		if job.StepType.IsChannel() {
			execErr = e.execChannel(ctx, jc, job)
		} else {
			execErr = fmt.Errorf("unknown step type: %s", job.StepType)
		}
	}

	if execErr != nil {
		job.Status = domain.JobFailed
		job.Error = execErr.Error()
		completed := time.Now()
		job.CompletedAt = &completed
		_ = e.store.UpdateJob(ctx, job)
		e.log(ctx, jc, job, "error", execErr.Error())
		return execErr
	}

	if job.Status != domain.JobDelayed && job.Status != domain.JobMerged {
		completed := time.Now()
		job.CompletedAt = &completed
		if job.Status == domain.JobRunning {
			job.Status = domain.JobCompleted
		}
		if err := e.store.UpdateJob(ctx, job); err != nil {
			return err
		}
		e.log(ctx, jc, job, "info", "job completed")

		next, err := e.store.GetNextJob(ctx, job.NotificationID, job.StepOrder)
		if err == nil && next != nil {
			return e.queue.EnqueueJob(ctx, envID, next.ID, next.ScheduledAt)
		}
		if err != nil && !repository.IsNotFound(err) {
			return err
		}
		return e.store.UpdateNotificationStatus(ctx, jc.Notification.ID, domain.NotificationCompleted)
	}
	return nil
}

func (e *Engine) loadNotificationContext(ctx context.Context, job *domain.Job) (*JobContext, error) {
	type ctxPayload struct {
		EnvID         string `json:"envId"`
		TransactionID string `json:"transactionId"`
		WorkflowID    string `json:"workflowId"`
		SubscriberPK  string `json:"subscriberPk"`
	}
	var meta ctxPayload
	if err := json.Unmarshal(job.Payload, &meta); err != nil {
		return nil, fmt.Errorf("job payload: %w", err)
	}

	notif, err := e.store.GetNotification(ctx, job.NotificationID)
	if err != nil {
		return nil, err
	}

	sub, err := e.store.GetSubscriberByPK(ctx, meta.SubscriberPK)
	if err != nil {
		return nil, err
	}
	wf, err := e.store.GetWorkflow(ctx, meta.EnvID, meta.WorkflowID)
	if err != nil {
		return nil, err
	}
	var step *domain.WorkflowStep
	for i := range wf.Steps {
		if wf.Steps[i].ID == job.WorkflowStepID {
			step = &wf.Steps[i]
			break
		}
	}
	if step == nil {
		return nil, fmt.Errorf("step not found")
	}

	return &JobContext{
		EnvID:         meta.EnvID,
		TransactionID: meta.TransactionID,
		Workflow:      wf,
		Notification:  notif,
		Subscriber:    sub,
		Step:          step,
	}, nil
}

func (e *Engine) execDelay(ctx context.Context, jc *JobContext, job *domain.Job) error {
	if job.ScheduledAt != nil {
		if time.Now().Before(*job.ScheduledAt) {
			return e.queue.EnqueueJob(ctx, jc.EnvID, job.ID, job.ScheduledAt)
		}
		job.Status = domain.JobCompleted
		return nil
	}
	d := durationFromMeta(jc.Step.Metadata)
	at := time.Now().Add(d)
	job.Status = domain.JobDelayed
	job.ScheduledAt = &at
	if err := e.store.UpdateJob(ctx, job); err != nil {
		return err
	}
	return e.queue.EnqueueJob(ctx, jc.EnvID, job.ID, &at)
}

func (e *Engine) execDigest(ctx context.Context, jc *JobContext, job *domain.Job) error {
	meta := jc.Step.Metadata
	key := meta.DigestKey
	if key == "" {
		key = jc.Workflow.TriggerID
	}
	window := durationFromDigestMeta(meta)
	windowEnd := time.Now().Add(window)

	eventPayload := template.ParsePayload(jc.Notification.Payload)
	eventBytes, _ := json.Marshal(eventPayload)

	bucket, err := e.store.GetDigestBucket(ctx, jc.EnvID, key, jc.Subscriber.ID)
	if err != nil {
		jobID := job.ID
		events, _ := json.Marshal([]json.RawMessage{eventBytes})
		bucket = &domain.DigestBucket{
			EnvID:        jc.EnvID,
			DigestKey:    key,
			SubscriberPK: jc.Subscriber.ID,
			WorkflowID:   jc.Workflow.ID,
			StepID:       jc.Step.ID,
			JobID:        &jobID,
			Events:       events,
			WindowEnd:    windowEnd,
		}
		bucket, err = e.store.UpsertDigestBucket(ctx, bucket)
		if err != nil {
			return err
		}
		_ = e.queue.EnqueueDigestFlush(ctx, jc.EnvID, bucket.ID, job.ID, windowEnd)
	} else {
		var events []json.RawMessage
		_ = json.Unmarshal(bucket.Events, &events)
		events = append(events, eventBytes)
		merged, _ := json.Marshal(events)
		bucket.Events = merged
		_, _ = e.store.UpsertDigestBucket(ctx, bucket)
	}

	job.Status = domain.JobMerged
	return nil
}

func (e *Engine) FlushDigest(ctx context.Context, envID, bucketID, jobID string) error {
	if bucketID != "" {
		if _, err := e.store.GetDigestBucketByID(ctx, bucketID); err != nil {
			return err
		}
	}
	return e.flushDigestByJob(ctx, envID, jobID)
}

func (e *Engine) flushDigestByJob(ctx context.Context, envID, jobID string) error {
	job, err := e.store.GetJob(ctx, jobID)
	if err != nil {
		return err
	}
	jc, err := e.loadNotificationContext(ctx, job)
	if err != nil {
		return err
	}
	key := jc.Step.Metadata.DigestKey
	if key == "" {
		key = jc.Workflow.TriggerID
	}
	bucket, err := e.store.GetDigestBucket(ctx, envID, key, jc.Subscriber.ID)
	if err != nil {
		return err
	}

	var events []map[string]any
	_ = json.Unmarshal(bucket.Events, &events)
	stepMeta := map[string]any{
		"events":      events,
		"total_count": len(events),
	}
	renderCtx := delivery.BuildRenderContext(jc.Subscriber, jc.Notification.Payload, stepMeta)

	nextStep := e.nextChannelStep(jc.Workflow, jc.Step.StepOrder)
	if nextStep == nil {
		_ = e.store.DeleteDigestBucket(ctx, bucket.ID)
		job.Status = domain.JobCompleted
		return e.store.UpdateJob(ctx, job)
	}

	rendered, err := template.RenderMessageTemplate(nextStep.Template, renderCtx)
	if err != nil {
		return err
	}
	jid := job.ID
	sendCtx := &delivery.SendContext{
		TransactionID:  jc.TransactionID,
		WorkflowID:     jc.Workflow.ID,
		NotificationID: jc.Notification.ID,
		Payload:        jc.Notification.Payload,
	}
	_, err = e.delivery.SendChannel(ctx, envID, nextStep.Type.Channel(), jc.Subscriber, rendered, &jid, sendCtx)
	if err != nil {
		return err
	}
	_ = e.store.DeleteDigestBucket(ctx, bucket.ID)
	job.Status = domain.JobCompleted
	completed := time.Now()
	job.CompletedAt = &completed
	_ = e.store.UpdateJob(ctx, job)
	return e.enqueueFollowingSteps(ctx, jc, job)
}

func (e *Engine) nextChannelStep(wf *domain.Workflow, afterOrder int) *domain.WorkflowStep {
	for i := range wf.Steps {
		if wf.Steps[i].StepOrder > afterOrder && wf.Steps[i].Type.IsChannel() {
			return &wf.Steps[i]
		}
	}
	return nil
}

func (e *Engine) enqueueFollowingSteps(ctx context.Context, jc *JobContext, job *domain.Job) error {
	next, err := e.store.GetNextJob(ctx, job.NotificationID, job.StepOrder)
	if err == nil && next != nil {
		return e.queue.EnqueueJob(ctx, jc.EnvID, next.ID, next.ScheduledAt)
	}
	if err != nil && !repository.IsNotFound(err) {
		return err
	}
	return e.store.UpdateNotificationStatus(ctx, jc.Notification.ID, domain.NotificationCompleted)
}

func (e *Engine) execThrottle(ctx context.Context, jc *JobContext, job *domain.Job) error {
	meta := jc.Step.Metadata
	key := meta.ThrottleKey
	if key == "" {
		key = fmt.Sprintf("%s:%s", jc.Workflow.TriggerID, jc.Subscriber.ID)
	}
	limit := meta.ThrottleLimit
	if limit <= 0 {
		limit = 1
	}
	window := meta.ThrottleWindow
	if window <= 0 {
		window = 60
	}
	redisKey := fmt.Sprintf("throttle:%s:%s", jc.EnvID, key)
	count, err := e.redis.Incr(ctx, redisKey).Result()
	if err != nil {
		return err
	}
	if count == 1 {
		e.redis.Expire(ctx, redisKey, time.Duration(window)*time.Second)
	}
	if int(count) > limit {
		action := meta.ThrottleAction
		if action == "delay" {
			at := time.Now().Add(time.Duration(window) * time.Second)
			job.Status = domain.JobDelayed
			job.ScheduledAt = &at
			_ = e.store.UpdateJob(ctx, job)
			return e.queue.EnqueueJob(ctx, jc.EnvID, job.ID, &at)
		}
		job.Status = domain.JobSkipped
		return nil
	}
	job.Status = domain.JobCompleted
	return nil
}

func (e *Engine) execChannel(ctx context.Context, jc *JobContext, job *domain.Job) error {
	if !e.checkPreferences(ctx, jc) {
		job.Status = domain.JobSkipped
		return nil
	}
	if !e.evaluateFilters(jc) {
		job.Status = domain.JobSkipped
		return nil
	}

	stepMeta := map[string]any{}
	renderCtx := delivery.BuildRenderContext(jc.Subscriber, jc.Notification.Payload, stepMeta)
	rendered, err := template.RenderMessageTemplate(jc.Step.Template, renderCtx)
	if err != nil {
		return err
	}
	jid := job.ID
	sendCtx := &delivery.SendContext{
		TransactionID:  jc.TransactionID,
		WorkflowID:     jc.Workflow.ID,
		NotificationID: jc.Notification.ID,
		Payload:        jc.Notification.Payload,
	}
	_, err = e.delivery.SendChannel(ctx, jc.EnvID, jc.Step.Type.Channel(), jc.Subscriber, rendered, &jid, sendCtx)
	return err
}

func (e *Engine) checkPreferences(ctx context.Context, jc *JobContext) bool {
	if !jc.Step.Type.IsChannel() {
		return true
	}
	prefs, _ := e.store.GetPreferences(ctx, jc.Subscriber.ID, &jc.Workflow.ID)
	return resolveChannelPreference(jc.Workflow.Critical, jc.Workflow.PreferenceSettings, jc.Step.Type.Channel(), prefs)
}

// resolveChannelPreference applies workflow defaults, then subscriber overrides when set.
func resolveChannelPreference(critical bool, wfPrefs domain.ChannelPrefs, ch domain.ChannelType, subPrefs []domain.SubscriberPreference) bool {
	if critical {
		return true
	}
	enabled := true
	if len(wfPrefs) > 0 {
		if v, ok := wfPrefs[ch]; ok {
			enabled = v
		}
	}
	for _, p := range subPrefs {
		if p.Channel == ch {
			return p.Enabled
		}
	}
	return enabled
}

func (e *Engine) evaluateFilters(jc *JobContext) bool {
	if len(jc.Step.Filters) == 0 {
		return true
	}
	payload := template.ParsePayload(jc.Notification.Payload)
	for _, f := range jc.Step.Filters {
		val, ok := payload[f.Field]
		if !ok {
			return false
		}
		switch f.Operator {
		case "eq":
			if fmt.Sprint(val) != fmt.Sprint(f.Value) {
				return false
			}
		case "neq":
			if fmt.Sprint(val) == fmt.Sprint(f.Value) {
				return false
			}
		}
	}
	return true
}

func (e *Engine) log(ctx context.Context, jc *JobContext, job *domain.Job, level, message string) {
	nid := jc.Notification.ID
	jid := job.ID
	st := string(job.StepType)
	_ = e.store.CreateExecutionLog(ctx, &domain.ExecutionLog{
		EnvID:          jc.EnvID,
		TransactionID:  jc.TransactionID,
		NotificationID: &nid,
		JobID:          &jid,
		StepType:       st,
		Level:          level,
		Message:        message,
	})
	slog.Info("workflow", "transactionId", jc.TransactionID, "jobId", job.ID, "msg", message)
}

func durationFromMeta(m domain.StepMetadata) time.Duration {
	amount := m.DelayAmount
	if amount <= 0 {
		amount = 1
	}
	switch m.DelayUnit {
	case "hours":
		return time.Duration(amount) * time.Hour
	case "minutes":
		return time.Duration(amount) * time.Minute
	default:
		return time.Duration(amount) * time.Second
	}
}

func durationFromDigestMeta(m domain.StepMetadata) time.Duration {
	amount := m.WindowAmount
	if amount <= 0 {
		amount = 5
	}
	switch m.WindowUnit {
	case "hours":
		return time.Duration(amount) * time.Hour
	case "seconds":
		return time.Duration(amount) * time.Second
	default:
		return time.Duration(amount) * time.Minute
	}
}
