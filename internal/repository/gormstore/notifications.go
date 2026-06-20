package gormstore

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/SolaTyolo/herald/internal/domain"
)

func (s *Store) GetNotification(ctx context.Context, id string) (*domain.Notification, error) {
	var m notificationModel
	err := s.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return m.toDomain()
}

func (s *Store) GetJobsByNotification(ctx context.Context, notificationID string) ([]domain.Job, error) {
	var rows []jobModel
	if err := s.db.WithContext(ctx).Where("notification_id = ?", notificationID).Order("step_order ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return jobsToDomain(rows)
}

func (s *Store) CreateNotification(ctx context.Context, n *domain.Notification, jobs []domain.Job) error {
	n.ID = uuid.NewString()
	n.CreatedAt = time.Now()
	if n.Status == "" {
		n.Status = domain.NotificationPending
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		nm := notificationModelFromDomain(n)
		if err := tx.Create(&nm).Error; err != nil {
			return err
		}
		for i := range jobs {
			j := &jobs[i]
			j.ID = uuid.NewString()
			j.NotificationID = n.ID
			j.Status = domain.JobPending
			j.CreatedAt = time.Now()
			j.UpdatedAt = j.CreatedAt
			if err := tx.Create(jobModelFromDomain(j)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) GetNotificationByTransaction(ctx context.Context, envID, transactionID, subscriberPK string) (*domain.Notification, error) {
	var m notificationModel
	err := s.db.WithContext(ctx).Where("env_id = ? AND transaction_id = ? AND subscriber_pk = ?", envID, transactionID, subscriberPK).First(&m).Error
	if err != nil {
		return nil, err
	}
	return m.toDomain()
}

func (s *Store) ListNotifications(ctx context.Context, envID string, limit, offset int) ([]domain.Notification, error) {
	var rows []notificationModel
	if err := s.db.WithContext(ctx).Where("env_id = ?", envID).
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Notification, 0, len(rows))
	for _, r := range rows {
		n, _ := r.toDomain()
		out = append(out, *n)
	}
	return out, nil
}

func (s *Store) UpdateNotificationStatus(ctx context.Context, id string, status domain.NotificationStatus) error {
	return s.db.WithContext(ctx).Model(&notificationModel{}).Where("id = ?", id).Update("status", status).Error
}

func (s *Store) GetJob(ctx context.Context, id string) (*domain.Job, error) {
	var m jobModel
	err := s.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return m.toDomain()
}

func (s *Store) GetNextJob(ctx context.Context, notificationID string, afterOrder int) (*domain.Job, error) {
	var m jobModel
	err := s.db.WithContext(ctx).Where("notification_id = ? AND step_order > ?", notificationID, afterOrder).
		Order("step_order ASC").First(&m).Error
	if err != nil {
		return nil, err
	}
	return m.toDomain()
}

func (s *Store) UpdateJob(ctx context.Context, job *domain.Job) error {
	job.UpdatedAt = time.Now()
	return s.db.WithContext(ctx).Model(&jobModel{}).Where("id = ?", job.ID).Updates(map[string]any{
		"status": job.Status, "payload": job.Payload, "error": job.Error,
		"scheduled_at": job.ScheduledAt, "started_at": job.StartedAt,
		"completed_at": job.CompletedAt, "updated_at": job.UpdatedAt,
	}).Error
}

func (s *Store) ListJobsByTransaction(ctx context.Context, envID, transactionID string) ([]domain.Job, error) {
	var rows []jobModel
	err := s.db.WithContext(ctx).
		Table("jobs").
		Joins("JOIN notifications ON notifications.id = jobs.notification_id").
		Where("notifications.env_id = ? AND notifications.transaction_id = ?", envID, transactionID).
		Order("jobs.step_order ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return jobsToDomain(rows)
}

func jobsToDomain(rows []jobModel) ([]domain.Job, error) {
	out := make([]domain.Job, 0, len(rows))
	for _, r := range rows {
		j, err := r.toDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, *j)
	}
	return out, nil
}
