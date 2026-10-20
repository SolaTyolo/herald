package service

import (
	"context"

	"github.com/SolaTyolo/herald/internal/domain"
	"github.com/SolaTyolo/herald/internal/repository"
)

type SubscriberService struct {
	store repository.Store
}

func NewSubscriberService(store repository.Store) *SubscriberService {
	return &SubscriberService{store: store}
}

func (s *SubscriberService) Upsert(ctx context.Context, envID string, sub *domain.Subscriber) error {
	return s.store.UpsertSubscriber(ctx, envID, sub)
}

func (s *SubscriberService) Get(ctx context.Context, envID, subscriberID string) (*domain.Subscriber, error) {
	return s.store.GetSubscriber(ctx, envID, subscriberID)
}

func (s *SubscriberService) List(ctx context.Context, envID string, limit, offset int) ([]domain.Subscriber, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.store.ListSubscribers(ctx, envID, limit, offset)
}

func (s *SubscriberService) Delete(ctx context.Context, envID, subscriberID string) error {
	return s.store.DeleteSubscriber(ctx, envID, subscriberID)
}

func (s *SubscriberService) SetPreference(ctx context.Context, envID, subscriberID string, pref domain.SubscriberPreference) error {
	return s.store.SetPreference(ctx, envID, subscriberID, pref)
}

type WorkflowService struct {
	store repository.Store
}

func NewWorkflowService(store repository.Store) *WorkflowService {
	return &WorkflowService{store: store}
}

func (s *WorkflowService) Create(ctx context.Context, envID string, wf *domain.Workflow) error {
	return s.store.CreateWorkflow(ctx, envID, wf)
}

func (s *WorkflowService) Get(ctx context.Context, envID, id string) (*domain.Workflow, error) {
	return s.store.GetWorkflow(ctx, envID, id)
}

func (s *WorkflowService) List(ctx context.Context, envID string) ([]domain.Workflow, error) {
	return s.store.ListWorkflows(ctx, envID)
}

func (s *WorkflowService) Update(ctx context.Context, envID string, wf *domain.Workflow) error {
	return s.store.UpdateWorkflow(ctx, envID, wf)
}

func (s *WorkflowService) Delete(ctx context.Context, envID, id string) error {
	return s.store.DeleteWorkflow(ctx, envID, id)
}

type TopicService struct {
	store repository.Store
}

func NewTopicService(store repository.Store) *TopicService {
	return &TopicService{store: store}
}

func (s *TopicService) Create(ctx context.Context, envID string, topic *domain.Topic) error {
	return s.store.CreateTopic(ctx, envID, topic)
}

func (s *TopicService) List(ctx context.Context, envID string) ([]domain.Topic, error) {
	return s.store.ListTopics(ctx, envID)
}

func (s *TopicService) Delete(ctx context.Context, envID, topicKey string) error {
	return s.store.DeleteTopic(ctx, envID, topicKey)
}

func (s *TopicService) Subscribe(ctx context.Context, envID, topicKey, subscriberID string) error {
	return s.store.AddTopicSubscription(ctx, envID, topicKey, subscriberID)
}

func (s *TopicService) Unsubscribe(ctx context.Context, envID, topicKey, subscriberID string) error {
	return s.store.RemoveTopicSubscription(ctx, envID, topicKey, subscriberID)
}

type NotificationQueryService struct {
	store repository.Store
}

func NewNotificationQueryService(store repository.Store) *NotificationQueryService {
	return &NotificationQueryService{store: store}
}

func (s *NotificationQueryService) List(ctx context.Context, envID string, limit, offset int) ([]domain.Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.store.ListNotifications(ctx, envID, limit, offset)
}

func (s *NotificationQueryService) JobsByTransaction(ctx context.Context, envID, transactionID string) ([]domain.Job, error) {
	return s.store.ListJobsByTransaction(ctx, envID, transactionID)
}

func (s *NotificationQueryService) ExecutionLogs(ctx context.Context, envID, transactionID string) ([]domain.ExecutionLog, error) {
	return s.store.ListExecutionLogs(ctx, envID, transactionID)
}

type MessageService struct {
	store repository.Store
}

func NewMessageService(store repository.Store) *MessageService {
	return &MessageService{store: store}
}

func (s *MessageService) List(ctx context.Context, envID, subscriberID string, unreadOnly bool, limit, offset int) ([]domain.Message, error) {
	sub, err := s.store.GetSubscriber(ctx, envID, subscriberID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	return s.store.ListMessages(ctx, sub.ID, unreadOnly, limit, offset)
}

func (s *MessageService) Update(ctx context.Context, envID, subscriberID, messageID string, read, archived *bool) (*domain.Message, error) {
	sub, err := s.store.GetSubscriber(ctx, envID, subscriberID)
	if err != nil {
		return nil, err
	}
	return s.store.UpdateMessageForSubscriber(ctx, envID, sub.ID, messageID, read, archived)
}
