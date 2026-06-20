package gormstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/SolaTyolo/herald/internal/domain"
)

func (s *Store) CreateMessage(ctx context.Context, msg *domain.Message) error {
	msg.ID = uuid.NewString()
	msg.CreatedAt = time.Now()
	if msg.Metadata == nil {
		msg.Metadata = json.RawMessage("{}")
	}
	return s.db.WithContext(ctx).Create(messageModelFromDomain(msg)).Error
}

func (s *Store) ListMessages(ctx context.Context, subscriberPK string, unreadOnly bool, limit, offset int) ([]domain.Message, error) {
	q := s.db.WithContext(ctx).Where("subscriber_pk = ?", subscriberPK)
	if unreadOnly {
		q = q.Where("read = ?", false)
	}
	var rows []messageModel
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Message, 0, len(rows))
	for _, r := range rows {
		m, _ := r.toDomain()
		out = append(out, *m)
	}
	return out, nil
}

func (s *Store) UpdateMessageForSubscriber(ctx context.Context, envID, subscriberPK, messageID string, read, archived *bool) (*domain.Message, error) {
	updates := map[string]any{}
	if read != nil {
		updates["read"] = *read
	}
	if archived != nil {
		updates["archived"] = *archived
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("nothing to update")
	}
	var m messageModel
	res := s.db.WithContext(ctx).Model(&messageModel{}).
		Where("id = ? AND env_id = ? AND subscriber_pk = ?", messageID, envID, subscriberPK).
		Updates(updates)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if err := s.db.WithContext(ctx).First(&m, "id = ?", messageID).Error; err != nil {
		return nil, err
	}
	return m.toDomain()
}
