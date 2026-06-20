package gormstore

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/SolaTyolo/herald/internal/domain"
)

func (s *Store) UpsertSubscriber(ctx context.Context, envID string, sub *domain.Subscriber) error {
	if sub.Data == nil {
		sub.Data = json.RawMessage("{}")
	}
	if sub.DeviceTokens == nil {
		sub.DeviceTokens = json.RawMessage("[]")
	}
	if sub.ChatCredentials == nil {
		sub.ChatCredentials = json.RawMessage("{}")
	}
	now := time.Now()
	if sub.ID == "" {
		sub.ID = uuid.NewString()
	}
	sub.EnvID = envID
	sub.UpdatedAt = now
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = now
	}
	m := subscriberModelFromDomain(sub)
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "env_id"}, {Name: "subscriber_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"email", "phone", "first_name", "last_name", "locale", "data",
			"device_tokens", "chat_credentials", "webhook_url", "updated_at",
		}),
	}).Create(&m).Error
}

func (s *Store) GetSubscriber(ctx context.Context, envID, subscriberID string) (*domain.Subscriber, error) {
	var m subscriberModel
	err := s.db.WithContext(ctx).Where("env_id = ? AND subscriber_id = ?", envID, subscriberID).First(&m).Error
	if err != nil {
		return nil, err
	}
	return m.toDomain()
}

func (s *Store) GetSubscriberByPK(ctx context.Context, pk string) (*domain.Subscriber, error) {
	var m subscriberModel
	err := s.db.WithContext(ctx).First(&m, "id = ?", pk).Error
	if err != nil {
		return nil, err
	}
	return m.toDomain()
}

func (s *Store) ListSubscribers(ctx context.Context, envID string, limit, offset int) ([]domain.Subscriber, error) {
	var rows []subscriberModel
	if err := s.db.WithContext(ctx).Where("env_id = ?", envID).
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Subscriber, 0, len(rows))
	for _, r := range rows {
		sub, err := r.toDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, *sub)
	}
	return out, nil
}

func (s *Store) DeleteSubscriber(ctx context.Context, envID, subscriberID string) error {
	res := s.db.WithContext(ctx).Where("env_id = ? AND subscriber_id = ?", envID, subscriberID).Delete(&subscriberModel{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Store) SetPreference(ctx context.Context, envID, subscriberID string, pref domain.SubscriberPreference) error {
	sub, err := s.GetSubscriber(ctx, envID, subscriberID)
	if err != nil {
		return err
	}
	pref.ID = uuid.NewString()
	pref.SubscriberPK = sub.ID
	m := subscriberPreferenceModel{
		ID: pref.ID, SubscriberPK: pref.SubscriberPK, WorkflowID: pref.WorkflowID,
		Channel: string(pref.Channel), Enabled: pref.Enabled,
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "subscriber_pk"}, {Name: "workflow_id"}, {Name: "channel"}},
		DoUpdates: clause.AssignmentColumns([]string{"enabled"}),
	}).Create(&m).Error
}

func (s *Store) GetPreferences(ctx context.Context, subscriberPK string, workflowID *string) ([]domain.SubscriberPreference, error) {
	q := s.db.WithContext(ctx).Where("subscriber_pk = ?", subscriberPK)
	if workflowID != nil {
		q = q.Where("workflow_id = ?", *workflowID)
	}
	var rows []subscriberPreferenceModel
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.SubscriberPreference, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toDomain())
	}
	return out, nil
}
