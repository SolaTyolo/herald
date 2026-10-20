package repository

import (
	"context"

	"github.com/SolaTyolo/herald/internal/domain"
)

type Store interface {
	// Lifecycle
	Close() error
	Ping(ctx context.Context) error

	// Bootstrap
	RunMigrations(ctx context.Context) error

	// Workflows
	CreateWorkflow(ctx context.Context, wf *domain.Workflow) error
	GetWorkflowByTrigger(ctx context.Context, triggerID string) (*domain.Workflow, error)
	GetWorkflow(ctx context.Context, id string) (*domain.Workflow, error)
	ListWorkflows(ctx context.Context) ([]domain.Workflow, error)
	UpdateWorkflow(ctx context.Context, wf *domain.Workflow) error
	DeleteWorkflow(ctx context.Context, id string) error

	// Subscribers and channel preferences
	UpsertSubscriber(ctx context.Context, sub *domain.Subscriber) error
	GetSubscriber(ctx context.Context, subscriberID string) (*domain.Subscriber, error)
	GetSubscriberByPK(ctx context.Context, pk string) (*domain.Subscriber, error)
	ListSubscribers(ctx context.Context, limit, offset int) ([]domain.Subscriber, error)
	DeleteSubscriber(ctx context.Context, subscriberID string) error
	SetPreference(ctx context.Context, subscriberID string, pref domain.SubscriberPreference) error
	GetPreferences(ctx context.Context, subscriberPK string, workflowID *string) ([]domain.SubscriberPreference, error)

	// Topics and topic subscriptions
	CreateTopic(ctx context.Context, topic *domain.Topic) error
	GetTopic(ctx context.Context, topicKey string) (*domain.Topic, error)
	ListTopics(ctx context.Context) ([]domain.Topic, error)
	DeleteTopic(ctx context.Context, topicKey string) error
	AddTopicSubscription(ctx context.Context, topicKey, subscriberID string) error
	RemoveTopicSubscription(ctx context.Context, topicKey, subscriberID string) error
	ListTopicSubscribers(ctx context.Context, topicID string, excludePK *string) ([]domain.Subscriber, error)

	// Notifications and delivery jobs
	GetJobsByNotification(ctx context.Context, notificationID string) ([]domain.Job, error)
	GetNotification(ctx context.Context, id string) (*domain.Notification, error)
	CreateNotification(ctx context.Context, n *domain.Notification, jobs []domain.Job) error
	GetNotificationByTransaction(ctx context.Context, transactionID, subscriberPK string) (*domain.Notification, error)
	ListNotifications(ctx context.Context, limit, offset int) ([]domain.Notification, error)
	UpdateNotificationStatus(ctx context.Context, id string, status domain.NotificationStatus) error
	GetJob(ctx context.Context, id string) (*domain.Job, error)
	GetNextJob(ctx context.Context, notificationID string, afterOrder int) (*domain.Job, error)
	UpdateJob(ctx context.Context, job *domain.Job) error
	ListJobsByTransaction(ctx context.Context, transactionID string) ([]domain.Job, error)

	// Digest buckets
	UpsertDigestBucket(ctx context.Context, bucket *domain.DigestBucket) (*domain.DigestBucket, error)
	GetDigestBucket(ctx context.Context, digestKey, subscriberPK string) (*domain.DigestBucket, error)
	GetDigestBucketByID(ctx context.Context, id string) (*domain.DigestBucket, error)
	DeleteDigestBucket(ctx context.Context, id string) error

	// In-app messages
	CreateMessage(ctx context.Context, msg *domain.Message) error
	GetMessageByJobID(ctx context.Context, jobID string) (*domain.Message, error)
	ListMessages(ctx context.Context, subscriberPK string, unreadOnly bool, limit, offset int) ([]domain.Message, error)
	UpdateMessageForSubscriber(ctx context.Context, subscriberPK, messageID string, read, archived *bool) (*domain.Message, error)

	// Execution logs
	CreateExecutionLog(ctx context.Context, log *domain.ExecutionLog) error
	ListExecutionLogs(ctx context.Context, transactionID string) ([]domain.ExecutionLog, error)

	// Notification templates and versions
	CreateTemplate(ctx context.Context, tmpl *domain.NotificationTemplate) error
	GetTemplate(ctx context.Context, id string) (*domain.NotificationTemplate, error)
	ListTemplates(ctx context.Context) ([]domain.NotificationTemplate, error)
	UpdateTemplate(ctx context.Context, tmpl *domain.NotificationTemplate) error
	DeleteTemplate(ctx context.Context, id string) error
	CreateTemplateVersion(ctx context.Context, templateID string, ver *domain.TemplateVersion) error
	UpdateTemplateVersion(ctx context.Context, templateID string, ver *domain.TemplateVersion) error
	DeleteTemplateVersion(ctx context.Context, templateID, versionID string) error
}
