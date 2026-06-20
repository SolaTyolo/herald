package repository

import (
	"context"

	"github.com/SolaTyolo/herald/internal/domain"
)

type Store interface {
	Close() error
	Ping(ctx context.Context) error

	RunMigrations(ctx context.Context) error
	EnsureDefaultTenant(ctx context.Context) (*domain.Environment, string, error)

	ValidateAPIKey(ctx context.Context, key string) (*domain.Environment, error)

	CreateWorkflow(ctx context.Context, envID string, wf *domain.Workflow) error
	GetWorkflowByTrigger(ctx context.Context, envID, triggerID string) (*domain.Workflow, error)
	GetWorkflow(ctx context.Context, envID, id string) (*domain.Workflow, error)
	ListWorkflows(ctx context.Context, envID string) ([]domain.Workflow, error)
	UpdateWorkflow(ctx context.Context, envID string, wf *domain.Workflow) error
	DeleteWorkflow(ctx context.Context, envID, id string) error

	UpsertSubscriber(ctx context.Context, envID string, sub *domain.Subscriber) error
	GetSubscriber(ctx context.Context, envID, subscriberID string) (*domain.Subscriber, error)
	GetSubscriberByPK(ctx context.Context, pk string) (*domain.Subscriber, error)
	ListSubscribers(ctx context.Context, envID string, limit, offset int) ([]domain.Subscriber, error)
	DeleteSubscriber(ctx context.Context, envID, subscriberID string) error
	SetPreference(ctx context.Context, envID, subscriberID string, pref domain.SubscriberPreference) error
	GetPreferences(ctx context.Context, subscriberPK string, workflowID *string) ([]domain.SubscriberPreference, error)

	CreateTopic(ctx context.Context, envID string, topic *domain.Topic) error
	GetTopic(ctx context.Context, envID, topicKey string) (*domain.Topic, error)
	ListTopics(ctx context.Context, envID string) ([]domain.Topic, error)
	DeleteTopic(ctx context.Context, envID, topicKey string) error
	AddTopicSubscription(ctx context.Context, envID, topicKey, subscriberID string) error
	RemoveTopicSubscription(ctx context.Context, envID, topicKey, subscriberID string) error
	ListTopicSubscribers(ctx context.Context, topicID string, excludePK *string) ([]domain.Subscriber, error)

	CreateIntegration(ctx context.Context, integration *domain.Integration) error
	ListIntegrations(ctx context.Context, envID string) ([]domain.Integration, error)
	DeleteIntegration(ctx context.Context, envID, id string) error
	ListActiveIntegrations(ctx context.Context, envID string, channel domain.ChannelType, primaryOnly bool) ([]domain.Integration, error)

	GetJobsByNotification(ctx context.Context, notificationID string) ([]domain.Job, error)
	GetNotification(ctx context.Context, id string) (*domain.Notification, error)
	CreateNotification(ctx context.Context, n *domain.Notification, jobs []domain.Job) error
	GetNotificationByTransaction(ctx context.Context, envID, transactionID, subscriberPK string) (*domain.Notification, error)
	ListNotifications(ctx context.Context, envID string, limit, offset int) ([]domain.Notification, error)
	UpdateNotificationStatus(ctx context.Context, id string, status domain.NotificationStatus) error
	GetJob(ctx context.Context, id string) (*domain.Job, error)
	GetNextJob(ctx context.Context, notificationID string, afterOrder int) (*domain.Job, error)
	UpdateJob(ctx context.Context, job *domain.Job) error
	ListJobsByTransaction(ctx context.Context, envID, transactionID string) ([]domain.Job, error)

	UpsertDigestBucket(ctx context.Context, bucket *domain.DigestBucket) (*domain.DigestBucket, error)
	GetDigestBucket(ctx context.Context, envID, digestKey, subscriberPK string) (*domain.DigestBucket, error)
	GetDigestBucketByID(ctx context.Context, id string) (*domain.DigestBucket, error)
	DeleteDigestBucket(ctx context.Context, id string) error

	CreateMessage(ctx context.Context, msg *domain.Message) error
	ListMessages(ctx context.Context, subscriberPK string, unreadOnly bool, limit, offset int) ([]domain.Message, error)
	UpdateMessageForSubscriber(ctx context.Context, envID, subscriberPK, messageID string, read, archived *bool) (*domain.Message, error)

	CreateExecutionLog(ctx context.Context, log *domain.ExecutionLog) error
	ListExecutionLogs(ctx context.Context, envID, transactionID string) ([]domain.ExecutionLog, error)
}
