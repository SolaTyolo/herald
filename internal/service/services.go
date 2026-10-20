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

func (s *SubscriberService) Upsert(ctx context.Context, sub *domain.Subscriber) error {
	return s.store.UpsertSubscriber(ctx, sub)
}

func (s *SubscriberService) Get(ctx context.Context, subscriberID string) (*domain.Subscriber, error) {
	return s.store.GetSubscriber(ctx, subscriberID)
}

func (s *SubscriberService) List(ctx context.Context, limit, offset int) ([]domain.Subscriber, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.store.ListSubscribers(ctx, limit, offset)
}

func (s *SubscriberService) Delete(ctx context.Context, subscriberID string) error {
	return s.store.DeleteSubscriber(ctx, subscriberID)
}

func (s *SubscriberService) SetPreference(ctx context.Context, subscriberID string, pref domain.SubscriberPreference) error {
	return s.store.SetPreference(ctx, subscriberID, pref)
}

type WorkflowService struct {
	store repository.Store
}

func NewWorkflowService(store repository.Store) *WorkflowService {
	return &WorkflowService{store: store}
}

func (s *WorkflowService) Create(ctx context.Context, wf *domain.Workflow) error {
	return s.store.CreateWorkflow(ctx, wf)
}

func (s *WorkflowService) Get(ctx context.Context, id string) (*domain.Workflow, error) {
	return s.store.GetWorkflow(ctx, id)
}

func (s *WorkflowService) List(ctx context.Context) ([]domain.Workflow, error) {
	return s.store.ListWorkflows(ctx)
}

func (s *WorkflowService) Update(ctx context.Context, wf *domain.Workflow) error {
	return s.store.UpdateWorkflow(ctx, wf)
}

func (s *WorkflowService) Delete(ctx context.Context, id string) error {
	return s.store.DeleteWorkflow(ctx, id)
}

type TopicService struct {
	store repository.Store
}

func NewTopicService(store repository.Store) *TopicService {
	return &TopicService{store: store}
}

func (s *TopicService) Create(ctx context.Context, topic *domain.Topic) error {
	return s.store.CreateTopic(ctx, topic)
}

func (s *TopicService) List(ctx context.Context) ([]domain.Topic, error) {
	return s.store.ListTopics(ctx)
}

func (s *TopicService) Delete(ctx context.Context, topicKey string) error {
	return s.store.DeleteTopic(ctx, topicKey)
}

func (s *TopicService) Subscribe(ctx context.Context, topicKey, subscriberID string) error {
	return s.store.AddTopicSubscription(ctx, topicKey, subscriberID)
}

func (s *TopicService) Unsubscribe(ctx context.Context, topicKey, subscriberID string) error {
	return s.store.RemoveTopicSubscription(ctx, topicKey, subscriberID)
}

type NotificationQueryService struct {
	store repository.Store
}

func NewNotificationQueryService(store repository.Store) *NotificationQueryService {
	return &NotificationQueryService{store: store}
}

func (s *NotificationQueryService) List(ctx context.Context, limit, offset int) ([]domain.Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.store.ListNotifications(ctx, limit, offset)
}

func (s *NotificationQueryService) JobsByTransaction(ctx context.Context, transactionID string) ([]domain.Job, error) {
	return s.store.ListJobsByTransaction(ctx, transactionID)
}

func (s *NotificationQueryService) ExecutionLogs(ctx context.Context, transactionID string) ([]domain.ExecutionLog, error) {
	return s.store.ListExecutionLogs(ctx, transactionID)
}

type MessageService struct {
	store repository.Store
}

func NewMessageService(store repository.Store) *MessageService {
	return &MessageService{store: store}
}

func (s *MessageService) List(ctx context.Context, subscriberID string, unreadOnly bool, limit, offset int) ([]domain.Message, error) {
	sub, err := s.store.GetSubscriber(ctx, subscriberID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	return s.store.ListMessages(ctx, sub.ID, unreadOnly, limit, offset)
}

func (s *MessageService) Update(ctx context.Context, subscriberID, messageID string, read, archived *bool) (*domain.Message, error) {
	sub, err := s.store.GetSubscriber(ctx, subscriberID)
	if err != nil {
		return nil, err
	}
	return s.store.UpdateMessageForSubscriber(ctx, sub.ID, messageID, read, archived)
}
