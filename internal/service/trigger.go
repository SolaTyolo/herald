package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/SolaTyolo/herald/internal/domain"
	asynqqueue "github.com/SolaTyolo/herald/internal/queue/asynq"
	"github.com/SolaTyolo/herald/internal/repository"
)

type TriggerService struct {
	store         repository.Store
	queue         asynqqueue.Enqueuer
	maxRecipients int
}

func NewTriggerService(store repository.Store, queue asynqqueue.Enqueuer, maxRecipients int) *TriggerService {
	return &TriggerService{store: store, queue: queue, maxRecipients: maxRecipients}
}

func (s *TriggerService) Trigger(ctx context.Context, req *domain.TriggerRequest) (*domain.TriggerResponse, error) {
	wf, err := s.store.GetWorkflowByTrigger(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %s", req.Name)
	}

	txID := req.TransactionID
	if txID == "" {
		txID = uuid.NewString()
	}

	if req.Payload == nil {
		req.Payload = json.RawMessage("{}")
	}

	// Topic triggers: validate + enqueue async fan-out (do not block HTTP on N subscribers).
	if isTopicTarget(req) {
		return s.enqueueTopicFanout(ctx, wf, txID, req)
	}

	subscribers, err := s.resolveSubscribers(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(subscribers) == 0 {
		return nil, fmt.Errorf("no recipients resolved")
	}
	if len(subscribers) > s.maxRecipients {
		return nil, fmt.Errorf("max %d recipients per trigger", s.maxRecipients)
	}

	resp := &domain.TriggerResponse{TransactionID: txID}
	for _, sub := range subscribers {
		nid, err := s.createAndEnqueue(ctx, wf, sub, req.Payload, txID)
		if err != nil {
			return nil, err
		}
		resp.NotificationIDs = append(resp.NotificationIDs, nid)
	}
	return resp, nil
}

func isTopicTarget(req *domain.TriggerRequest) bool {
	return req.To.Type == "Topic" || req.To.TopicKey != ""
}

func (s *TriggerService) enqueueTopicFanout(ctx context.Context, wf *domain.Workflow, txID string, req *domain.TriggerRequest) (*domain.TriggerResponse, error) {
	key := req.To.TopicKey
	if key == "" {
		return nil, fmt.Errorf("topicKey required")
	}
	topic, err := s.store.GetTopic(ctx, key)
	if err != nil {
		return nil, err
	}
	var exclude *string
	if req.Actor != "" {
		if actorSub, err := s.store.GetSubscriber(ctx, req.Actor); err == nil {
			exclude = &actorSub.ID
		}
	}
	subs, err := s.store.ListTopicSubscribers(ctx, topic.ID, exclude)
	if err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return nil, fmt.Errorf("no recipients resolved")
	}
	if len(subs) > s.maxRecipients {
		return nil, fmt.Errorf("max %d recipients per trigger", s.maxRecipients)
	}

	if err := s.queue.EnqueueTopicFanout(ctx, asynqqueue.TopicFanoutPayload{
		TransactionID: txID,
		WorkflowID:    wf.ID,
		TopicKey:      key,
		Actor:         req.Actor,
		Payload:       req.Payload,
	}); err != nil {
		return nil, err
	}
	return &domain.TriggerResponse{TransactionID: txID, Async: true}, nil
}

// ProcessTopicFanout creates notifications for each topic subscriber (worker-side).
func (s *TriggerService) ProcessTopicFanout(ctx context.Context, p asynqqueue.TopicFanoutPayload) error {
	wf, err := s.store.GetWorkflow(ctx, p.WorkflowID)
	if err != nil {
		return err
	}
	topic, err := s.store.GetTopic(ctx, p.TopicKey)
	if err != nil {
		return err
	}
	var exclude *string
	if p.Actor != "" {
		if actorSub, err := s.store.GetSubscriber(ctx, p.Actor); err == nil {
			exclude = &actorSub.ID
		}
	}
	subs, err := s.store.ListTopicSubscribers(ctx, topic.ID, exclude)
	if err != nil {
		return err
	}
	if len(subs) == 0 {
		return nil
	}
	if len(subs) > s.maxRecipients {
		return fmt.Errorf("max %d recipients per trigger", s.maxRecipients)
	}

	payload := p.Payload
	if payload == nil {
		payload = json.RawMessage("{}")
	}
	var firstErr error
	for _, sub := range subs {
		if _, err := s.createAndEnqueue(ctx, wf, sub, payload, p.TransactionID); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (s *TriggerService) resolveSubscribers(ctx context.Context, req *domain.TriggerRequest) ([]domain.Subscriber, error) {
	if isTopicTarget(req) {
		key := req.To.TopicKey
		if key == "" {
			return nil, fmt.Errorf("topicKey required")
		}
		topic, err := s.store.GetTopic(ctx, key)
		if err != nil {
			return nil, err
		}
		var exclude *string
		if req.Actor != "" {
			if actorSub, err := s.store.GetSubscriber(ctx, req.Actor); err == nil {
				exclude = &actorSub.ID
			}
		}
		return s.store.ListTopicSubscribers(ctx, topic.ID, exclude)
	}

	sid := req.To.SubscriberID
	if sid == "" {
		return nil, fmt.Errorf("subscriberId required")
	}
	sub, err := s.store.GetSubscriber(ctx, sid)
	if err != nil {
		sub = &domain.Subscriber{SubscriberID: sid}
		if err := s.store.UpsertSubscriber(ctx, sub); err != nil {
			return nil, err
		}
	}
	return []domain.Subscriber{*sub}, nil
}

func (s *TriggerService) createAndEnqueue(ctx context.Context, wf *domain.Workflow, sub domain.Subscriber, payload json.RawMessage, txID string) (string, error) {
	if existing, err := s.store.GetNotificationByTransaction(ctx, txID, sub.ID); err == nil {
		return existing.ID, nil
	}

	jobMeta, _ := json.Marshal(map[string]string{
		"transactionId": txID,
		"workflowId":    wf.ID,
		"subscriberPk":  sub.ID,
	})

	notif := &domain.Notification{
		TransactionID: txID,
		WorkflowID:    wf.ID,
		SubscriberPK:  sub.ID,
		Payload:       payload,
		Status:        domain.NotificationPending,
	}

	jobs := make([]domain.Job, len(wf.Steps))
	for i, step := range wf.Steps {
		jobs[i] = domain.Job{
			WorkflowStepID: step.ID,
			StepOrder:      step.StepOrder,
			StepType:       step.Type,
			Payload:        jobMeta,
		}
	}

	if err := s.store.CreateNotification(ctx, notif, jobs); err != nil {
		return "", err
	}

	jobs, err := s.store.GetJobsByNotification(ctx, notif.ID)
	if err != nil || len(jobs) == 0 {
		return notif.ID, err
	}
	first := jobs[0]
	if err := s.queue.EnqueueJob(ctx, first.ID, nil); err != nil {
		return notif.ID, err
	}
	_ = s.store.UpdateNotificationStatus(ctx, notif.ID, domain.NotificationRunning)
	return notif.ID, nil
}
