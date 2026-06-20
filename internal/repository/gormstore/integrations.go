package gormstore

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/SolaTyolo/herald/internal/domain"
)

func (s *Store) CreateIntegration(ctx context.Context, integration *domain.Integration) error {
	integration.ID = uuid.NewString()
	now := time.Now()
	integration.CreatedAt, integration.UpdatedAt = now, now
	m := integrationModelFromDomain(integration)
	return s.db.WithContext(ctx).Create(&m).Error
}

func (s *Store) ListIntegrations(ctx context.Context, envID string) ([]domain.Integration, error) {
	var rows []integrationModel
	if err := s.db.WithContext(ctx).Where("env_id = ?", envID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Integration, 0, len(rows))
	for _, r := range rows {
		i, _ := r.toDomain()
		out = append(out, *i)
	}
	return out, nil
}

func (s *Store) DeleteIntegration(ctx context.Context, envID, id string) error {
	res := s.db.WithContext(ctx).Where("id = ? AND env_id = ?", id, envID).Delete(&integrationModel{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Store) ListActiveIntegrations(ctx context.Context, envID string, channel domain.ChannelType, primaryOnly bool) ([]domain.Integration, error) {
	q := s.db.WithContext(ctx).Where("env_id = ? AND channel = ? AND active = ?", envID, channel, true)
	if primaryOnly {
		q = q.Where("is_primary = ?", true)
	}
	var rows []integrationModel
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Integration, 0, len(rows))
	for _, r := range rows {
		i, _ := r.toDomain()
		out = append(out, *i)
	}
	return out, nil
}
