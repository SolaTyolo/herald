package gormstore

import (
	"encoding/json"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/SolaTyolo/herald/internal/config"
	"github.com/SolaTyolo/herald/internal/domain"
)

func openDB(driver config.DBDriver, dsn string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch driver {
	case config.DBDriverMySQL:
		dialector = mysql.Open(dsn)
	case config.DBDriverSQLite:
		dialector = sqlite.Open(dsn)
	default:
		dialector = postgres.Open(dsn)
	}
	return gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
}

func allModels() []any {
	return []any{
		&tenantModel{}, &environmentModel{}, &apiKeyModel{},
		&workflowModel{}, &workflowStepModel{},
		&subscriberModel{}, &subscriberPreferenceModel{},
		&topicModel{}, &topicSubscriptionModel{},
		&integrationModel{},
		&notificationModel{}, &jobModel{},
		&messageModel{}, &digestBucketModel{}, &executionLogModel{},
	}
}

type tenantModel struct {
	ID        string    `gorm:"primaryKey;size:36"`
	Name      string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (tenantModel) TableName() string { return "tenants" }

type environmentModel struct {
	ID        string    `gorm:"primaryKey;size:36"`
	TenantID  string    `gorm:"size:36;not null;uniqueIndex:idx_env_tenant_slug,priority:1"`
	Name      string    `gorm:"not null"`
	Slug      string    `gorm:"not null;uniqueIndex:idx_env_tenant_slug,priority:2"`
	CreatedAt time.Time `gorm:"not null"`
}

func (environmentModel) TableName() string { return "environments" }

func (m environmentModel) toDomain() *domain.Environment {
	return &domain.Environment{ID: m.ID, TenantID: m.TenantID, Name: m.Name, Slug: m.Slug, CreatedAt: m.CreatedAt}
}

type apiKeyModel struct {
	ID        string    `gorm:"primaryKey;size:36"`
	EnvID     string    `gorm:"size:36;not null;index"`
	Name      string    `gorm:"not null"`
	KeyHash   string    `gorm:"not null;uniqueIndex"`
	KeyPrefix string    `gorm:"not null;index"`
	CreatedAt time.Time `gorm:"not null"`
}

func (apiKeyModel) TableName() string { return "api_keys" }

type workflowModel struct {
	ID                 string    `gorm:"primaryKey;size:36"`
	EnvID              string    `gorm:"size:36;not null;uniqueIndex:idx_workflow_trigger,priority:1"`
	Name               string    `gorm:"not null"`
	TriggerID          string    `gorm:"not null;uniqueIndex:idx_workflow_trigger,priority:2"`
	Critical           bool      `gorm:"not null;default:false"`
	PreferenceSettings []byte    `gorm:"type:json;not null"`
	Active             bool      `gorm:"not null;default:true"`
	CreatedAt          time.Time `gorm:"not null"`
	UpdatedAt          time.Time `gorm:"not null"`
}

func (workflowModel) TableName() string { return "workflows" }

func (m workflowModel) toDomain() (*domain.Workflow, error) {
	wf := &domain.Workflow{
		ID: m.ID, EnvID: m.EnvID, Name: m.Name, TriggerID: m.TriggerID,
		Critical: m.Critical, Active: m.Active, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
	if len(m.PreferenceSettings) > 0 {
		_ = json.Unmarshal(m.PreferenceSettings, &wf.PreferenceSettings)
	}
	return wf, nil
}

type workflowStepModel struct {
	ID         string    `gorm:"primaryKey;size:36"`
	WorkflowID string    `gorm:"size:36;not null;index;uniqueIndex:idx_step_order,priority:1"`
	StepOrder  int       `gorm:"not null;uniqueIndex:idx_step_order,priority:2"`
	StepType   string    `gorm:"not null"`
	Template   []byte    `gorm:"type:json"`
	Filters    []byte    `gorm:"type:json;not null"`
	Metadata   []byte    `gorm:"type:json;not null"`
	CreatedAt  time.Time `gorm:"not null"`
}

func (workflowStepModel) TableName() string { return "workflow_steps" }

func (m workflowStepModel) toDomain() (*domain.WorkflowStep, error) {
	st := &domain.WorkflowStep{
		ID: m.ID, WorkflowID: m.WorkflowID, StepOrder: m.StepOrder,
		Type: domain.StepType(m.StepType), CreatedAt: m.CreatedAt,
	}
	if len(m.Template) > 0 {
		st.Template = &domain.MessageTemplate{}
		_ = json.Unmarshal(m.Template, st.Template)
	}
	if len(m.Filters) > 0 {
		_ = json.Unmarshal(m.Filters, &st.Filters)
	}
	if len(m.Metadata) > 0 {
		_ = json.Unmarshal(m.Metadata, &st.Metadata)
	}
	return st, nil
}

type subscriberModel struct {
	ID              string    `gorm:"primaryKey;size:36"`
	EnvID           string    `gorm:"size:36;not null;index;uniqueIndex:idx_subscriber_env,priority:1"`
	SubscriberID    string    `gorm:"not null;uniqueIndex:idx_subscriber_env,priority:2"`
	Email           string
	Phone           string
	FirstName       string
	LastName        string
	Locale          string
	WebhookURL      string
	Data            []byte    `gorm:"type:json;not null"`
	DeviceTokens    []byte    `gorm:"type:json;not null"`
	ChatCredentials []byte    `gorm:"type:json;not null"`
	CreatedAt       time.Time `gorm:"not null"`
	UpdatedAt       time.Time `gorm:"not null"`
}

func (subscriberModel) TableName() string { return "subscribers" }

func subscriberModelFromDomain(sub *domain.Subscriber) subscriberModel {
	return subscriberModel{
		ID: sub.ID, EnvID: sub.EnvID, SubscriberID: sub.SubscriberID,
		Email: sub.Email, Phone: sub.Phone, FirstName: sub.FirstName, LastName: sub.LastName, Locale: sub.Locale,
		WebhookURL: sub.WebhookURL,
		Data: sub.Data, DeviceTokens: sub.DeviceTokens, ChatCredentials: sub.ChatCredentials,
		CreatedAt: sub.CreatedAt, UpdatedAt: sub.UpdatedAt,
	}
}

func (m subscriberModel) toDomain() (*domain.Subscriber, error) {
	return &domain.Subscriber{
		ID: m.ID, EnvID: m.EnvID, SubscriberID: m.SubscriberID,
		Email: m.Email, Phone: m.Phone, FirstName: m.FirstName, LastName: m.LastName, Locale: m.Locale,
		WebhookURL: m.WebhookURL,
		Data: m.Data, DeviceTokens: m.DeviceTokens, ChatCredentials: m.ChatCredentials,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}, nil
}

type subscriberPreferenceModel struct {
	ID           string  `gorm:"primaryKey;size:36"`
	SubscriberPK string  `gorm:"size:36;not null;uniqueIndex:idx_pref,priority:1"`
	WorkflowID   *string `gorm:"size:36;uniqueIndex:idx_pref,priority:2"`
	Channel      string  `gorm:"not null;uniqueIndex:idx_pref,priority:3"`
	Enabled      bool    `gorm:"not null;default:true"`
}

func (subscriberPreferenceModel) TableName() string { return "subscriber_preferences" }

func (m subscriberPreferenceModel) toDomain() domain.SubscriberPreference {
	return domain.SubscriberPreference{ID: m.ID, SubscriberPK: m.SubscriberPK, WorkflowID: m.WorkflowID, Channel: domain.ChannelType(m.Channel), Enabled: m.Enabled}
}

type topicModel struct {
	ID        string    `gorm:"primaryKey;size:36"`
	EnvID     string    `gorm:"size:36;not null;index;uniqueIndex:idx_topic_env,priority:1"`
	TopicKey  string    `gorm:"not null;uniqueIndex:idx_topic_env,priority:2"`
	Name      string
	CreatedAt time.Time `gorm:"not null"`
}

func (topicModel) TableName() string { return "topics" }

func (m topicModel) toDomain() (*domain.Topic, error) {
	return &domain.Topic{ID: m.ID, EnvID: m.EnvID, TopicKey: m.TopicKey, Name: m.Name, CreatedAt: m.CreatedAt}, nil
}

type topicSubscriptionModel struct {
	ID           string    `gorm:"primaryKey;size:36"`
	TopicID      string    `gorm:"size:36;not null;uniqueIndex:idx_topic_sub,priority:1"`
	SubscriberPK string    `gorm:"size:36;not null;uniqueIndex:idx_topic_sub,priority:2"`
	CreatedAt    time.Time `gorm:"not null"`
}

func (topicSubscriptionModel) TableName() string { return "topic_subscriptions" }

type integrationModel struct {
	ID                   string    `gorm:"primaryKey;size:36"`
	EnvID                string    `gorm:"size:36;not null;index"`
	Channel              string    `gorm:"not null"`
	ProviderID           string    `gorm:"not null"`
	Name                 string    `gorm:"not null"`
	CredentialsEncrypted []byte    `gorm:"not null"`
	IsPrimary            bool      `gorm:"not null;default:false"`
	Active               bool      `gorm:"not null;default:true"`
	CreatedAt            time.Time `gorm:"not null"`
	UpdatedAt            time.Time `gorm:"not null"`
}

func (integrationModel) TableName() string { return "integrations" }

func integrationModelFromDomain(i *domain.Integration) integrationModel {
	return integrationModel{
		ID: i.ID, EnvID: i.EnvID, Channel: string(i.Channel), ProviderID: i.ProviderID, Name: i.Name,
		CredentialsEncrypted: i.CredentialsEncrypted, IsPrimary: i.IsPrimary, Active: i.Active,
		CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt,
	}
}

func (m integrationModel) toDomain() (*domain.Integration, error) {
	return &domain.Integration{
		ID: m.ID, EnvID: m.EnvID, Channel: domain.ChannelType(m.Channel), ProviderID: m.ProviderID, Name: m.Name,
		CredentialsEncrypted: m.CredentialsEncrypted, IsPrimary: m.IsPrimary, Active: m.Active,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}, nil
}

type notificationModel struct {
	ID            string    `gorm:"primaryKey;size:36"`
	EnvID         string    `gorm:"size:36;not null;index;uniqueIndex:idx_notif_tx,priority:1"`
	TransactionID string    `gorm:"not null;index;uniqueIndex:idx_notif_tx,priority:2"`
	WorkflowID    string    `gorm:"size:36;not null"`
	SubscriberPK  string    `gorm:"size:36;not null;uniqueIndex:idx_notif_tx,priority:3"`
	Payload       []byte    `gorm:"type:json;not null"`
	Status        string    `gorm:"not null;default:pending"`
	CreatedAt     time.Time `gorm:"not null"`
}

func (notificationModel) TableName() string { return "notifications" }

func notificationModelFromDomain(n *domain.Notification) notificationModel {
	return notificationModel{
		ID: n.ID, EnvID: n.EnvID, TransactionID: n.TransactionID, WorkflowID: n.WorkflowID,
		SubscriberPK: n.SubscriberPK, Payload: n.Payload, Status: string(n.Status), CreatedAt: n.CreatedAt,
	}
}

func (m notificationModel) toDomain() (*domain.Notification, error) {
	return &domain.Notification{
		ID: m.ID, EnvID: m.EnvID, TransactionID: m.TransactionID, WorkflowID: m.WorkflowID,
		SubscriberPK: m.SubscriberPK, Payload: m.Payload, Status: domain.NotificationStatus(m.Status), CreatedAt: m.CreatedAt,
	}, nil
}

type jobModel struct {
	ID             string     `gorm:"primaryKey;size:36"`
	NotificationID string     `gorm:"size:36;not null;index"`
	WorkflowStepID string     `gorm:"size:36;not null"`
	StepOrder      int        `gorm:"not null;index"`
	StepType       string     `gorm:"not null"`
	Status         string     `gorm:"not null;default:pending"`
	Payload        []byte     `gorm:"type:json;not null"`
	Error          string
	ScheduledAt    *time.Time
	StartedAt      *time.Time
	CompletedAt    *time.Time
	CreatedAt      time.Time  `gorm:"not null"`
	UpdatedAt      time.Time  `gorm:"not null"`
}

func (jobModel) TableName() string { return "jobs" }

func jobModelFromDomain(j *domain.Job) *jobModel {
	return &jobModel{
		ID: j.ID, NotificationID: j.NotificationID, WorkflowStepID: j.WorkflowStepID,
		StepOrder: j.StepOrder, StepType: string(j.StepType), Status: string(j.Status),
		Payload: j.Payload, Error: j.Error, ScheduledAt: j.ScheduledAt,
		StartedAt: j.StartedAt, CompletedAt: j.CompletedAt, CreatedAt: j.CreatedAt, UpdatedAt: j.UpdatedAt,
	}
}

func (m jobModel) toDomain() (*domain.Job, error) {
	return &domain.Job{
		ID: m.ID, NotificationID: m.NotificationID, WorkflowStepID: m.WorkflowStepID,
		StepOrder: m.StepOrder, StepType: domain.StepType(m.StepType), Status: domain.JobStatus(m.Status),
		Payload: m.Payload, Error: m.Error, ScheduledAt: m.ScheduledAt,
		StartedAt: m.StartedAt, CompletedAt: m.CompletedAt, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}, nil
}

type messageModel struct {
	ID           string    `gorm:"primaryKey;size:36"`
	EnvID        string    `gorm:"size:36;not null;index"`
	JobID        *string   `gorm:"size:36"`
	SubscriberPK string    `gorm:"size:36;not null;index"`
	Channel      string    `gorm:"not null"`
	Subject      string
	Title        string
	Content      string    `gorm:"not null"`
	Metadata     []byte    `gorm:"type:json;not null"`
	ProviderRef  string
	Read         bool      `gorm:"not null;default:false"`
	Archived     bool      `gorm:"not null;default:false"`
	CreatedAt    time.Time `gorm:"not null;index"`
}

func (messageModel) TableName() string { return "messages" }

func messageModelFromDomain(msg *domain.Message) *messageModel {
	return &messageModel{
		ID: msg.ID, EnvID: msg.EnvID, JobID: msg.JobID, SubscriberPK: msg.SubscriberPK,
		Channel: string(msg.Channel), Subject: msg.Subject, Title: msg.Title, Content: msg.Content,
		Metadata: msg.Metadata, ProviderRef: msg.ProviderRef, Read: msg.Read, Archived: msg.Archived, CreatedAt: msg.CreatedAt,
	}
}

func (m messageModel) toDomain() (*domain.Message, error) {
	return &domain.Message{
		ID: m.ID, EnvID: m.EnvID, JobID: m.JobID, SubscriberPK: m.SubscriberPK,
		Channel: domain.ChannelType(m.Channel), Subject: m.Subject, Title: m.Title, Content: m.Content,
		Metadata: m.Metadata, ProviderRef: m.ProviderRef, Read: m.Read, Archived: m.Archived, CreatedAt: m.CreatedAt,
	}, nil
}

type digestBucketModel struct {
	ID           string    `gorm:"primaryKey;size:36"`
	EnvID        string    `gorm:"size:36;not null;index;uniqueIndex:idx_digest_active,priority:1"`
	DigestKey    string    `gorm:"not null;index;uniqueIndex:idx_digest_active,priority:2"`
	SubscriberPK string    `gorm:"size:36;not null;index;uniqueIndex:idx_digest_active,priority:3"`
	WorkflowID   string    `gorm:"size:36;not null"`
	StepID       string    `gorm:"size:36;not null"`
	JobID        *string   `gorm:"size:36"`
	Events       []byte    `gorm:"type:json;not null"`
	WindowEnd    time.Time `gorm:"not null;index;uniqueIndex:idx_digest_active,priority:4"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
}

func (digestBucketModel) TableName() string { return "digest_buckets" }

func digestBucketModelFromDomain(b *domain.DigestBucket) digestBucketModel {
	return digestBucketModel{
		ID: b.ID, EnvID: b.EnvID, DigestKey: b.DigestKey, SubscriberPK: b.SubscriberPK,
		WorkflowID: b.WorkflowID, StepID: b.StepID, JobID: b.JobID,
		Events: b.Events, WindowEnd: b.WindowEnd, CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
	}
}

func (m digestBucketModel) toDomain() (*domain.DigestBucket, error) {
	return &domain.DigestBucket{
		ID: m.ID, EnvID: m.EnvID, DigestKey: m.DigestKey, SubscriberPK: m.SubscriberPK,
		WorkflowID: m.WorkflowID, StepID: m.StepID, JobID: m.JobID,
		Events: m.Events, WindowEnd: m.WindowEnd, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}, nil
}

type executionLogModel struct {
	ID             string    `gorm:"primaryKey;size:36"`
	EnvID          string    `gorm:"size:36;not null;index"`
	TransactionID  string    `gorm:"not null;index"`
	NotificationID *string   `gorm:"size:36"`
	JobID          *string   `gorm:"size:36"`
	StepType       string
	Level          string    `gorm:"not null;default:info"`
	Message        string    `gorm:"not null"`
	Metadata       []byte    `gorm:"type:json;not null"`
	CreatedAt      time.Time `gorm:"not null;index"`
}

func (executionLogModel) TableName() string { return "execution_logs" }

func executionLogModelFromDomain(l *domain.ExecutionLog) *executionLogModel {
	return &executionLogModel{
		ID: l.ID, EnvID: l.EnvID, TransactionID: l.TransactionID, NotificationID: l.NotificationID,
		JobID: l.JobID, StepType: l.StepType, Level: l.Level, Message: l.Message,
		Metadata: l.Metadata, CreatedAt: l.CreatedAt,
	}
}

func (m executionLogModel) toDomain() (*domain.ExecutionLog, error) {
	return &domain.ExecutionLog{
		ID: m.ID, EnvID: m.EnvID, TransactionID: m.TransactionID, NotificationID: m.NotificationID,
		JobID: m.JobID, StepType: m.StepType, Level: m.Level, Message: m.Message,
		Metadata: m.Metadata, CreatedAt: m.CreatedAt,
	}, nil
}

// unique indexes are declared on model fields above.
