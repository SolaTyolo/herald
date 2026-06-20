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
	queue         *asynqqueue.Client
	maxRecipients int
}

func NewTriggerService(store repository.Store, queue *asynqqueue.Client, maxRecipients int) *TriggerService {
	return &TriggerService{store: store, queue: queue, maxRecipients: maxRecipients}
}

func (s *TriggerService) Trigger(ctx context.Context, envID string, req *domain.TriggerRequest) (*domain.TriggerResponse, error) {
	wf, err := s.store.GetWorkflowByTrigger(ctx, envID, req.Name)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %s", req.Name)
	}

	txID := req.TransactionID
	if txID == "" {
		txID = uuid.NewString()
	}

	subscribers, err := s.resolveSubscribers(ctx, envID, req)
	if err != nil {
		return nil, err
	}
	if len(subscribers) == 0 {
		return nil, fmt.Errorf("no recipients resolved")
	}
	if len(subscribers) > s.maxRecipients {
		return nil, fmt.Errorf("max %d recipients per trigger", s.maxRecipients)
	}

	if req.Payload == nil {
		req.Payload = json.RawMessage("{}")
	}

	resp := &domain.TriggerResponse{TransactionID: txID}
	for _, sub := range subscribers {
		nid, err := s.createAndEnqueue(ctx, envID, wf, sub, req.Payload, txID)
		if err != nil {
			return nil, err
		}
		resp.NotificationIDs = append(resp.NotificationIDs, nid)
	}
	return resp, nil
}

func (s *TriggerService) resolveSubscribers(ctx context.Context, envID string, req *domain.TriggerRequest) ([]domain.Subscriber, error) {
	if req.To.Type == "Topic" || req.To.TopicKey != "" {
		key := req.To.TopicKey
		if key == "" {
			return nil, fmt.Errorf("topicKey required")
		}
		topic, err := s.store.GetTopic(ctx, envID, key)
		if err != nil {
			return nil, err
		}
		var exclude *string
		if req.Actor != "" {
			if actorSub, err := s.store.GetSubscriber(ctx, envID, req.Actor); err == nil {
				exclude = &actorSub.ID
			}
		}
		return s.store.ListTopicSubscribers(ctx, topic.ID, exclude)
	}

	sid := req.To.SubscriberID
	if sid == "" {
		return nil, fmt.Errorf("subscriberId required")
	}
	sub, err := s.store.GetSubscriber(ctx, envID, sid)
	if err != nil {
		sub = &domain.Subscriber{SubscriberID: sid}
		if err := s.store.UpsertSubscriber(ctx, envID, sub); err != nil {
			return nil, err
		}
	}
	return []domain.Subscriber{*sub}, nil
}

func (s *TriggerService) createAndEnqueue(ctx context.Context, envID string, wf *domain.Workflow, sub domain.Subscriber, payload json.RawMessage, txID string) (string, error) {
	if existing, err := s.store.GetNotificationByTransaction(ctx, envID, txID, sub.ID); err == nil {
		return existing.ID, nil
	}

	jobMeta, _ := json.Marshal(map[string]string{
		"envId":         envID,
		"transactionId": txID,
		"workflowId":    wf.ID,
		"subscriberPk":  sub.ID,
	})

	notif := &domain.Notification{
		EnvID:         envID,
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

	// reload first job id
	jobs, err := s.store.GetJobsByNotification(ctx, notif.ID)
	if err != nil || len(jobs) == 0 {
		return notif.ID, err
	}
	first := jobs[0]
	if err := s.queue.EnqueueJob(ctx, envID, first.ID, nil); err != nil {
		return notif.ID, err
	}
	_ = s.store.UpdateNotificationStatus(ctx, notif.ID, domain.NotificationRunning)
	return notif.ID, nil
}
