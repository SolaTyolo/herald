package gormstore

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/SolaTyolo/herald/internal/domain"
)

func (s *Store) UpsertDigestBucket(ctx context.Context, bucket *domain.DigestBucket) (*domain.DigestBucket, error) {
	if bucket.ID == "" {
		bucket.ID = uuid.NewString()
	}
	now := time.Now()
	bucket.UpdatedAt = now
	if bucket.CreatedAt.IsZero() {
		bucket.CreatedAt = now
	}
	m := digestBucketModelFromDomain(bucket)
	err := s.db.WithContext(ctx).Save(&m).Error
	if err == nil {
		bucket.Events = m.Events
		bucket.WindowEnd = m.WindowEnd
		return bucket, nil
	}
	var existing digestBucketModel
	e := s.db.WithContext(ctx).
		Where("env_id = ? AND digest_key = ? AND subscriber_pk = ? AND window_end > ?", bucket.EnvID, bucket.DigestKey, bucket.SubscriberPK, now).
		Order("window_end ASC").First(&existing).Error
	if e != nil {
		return bucket, err
	}
	var events []json.RawMessage
	_ = json.Unmarshal(existing.Events, &events)
	var newEvent json.RawMessage
	_ = json.Unmarshal(bucket.Events, &newEvent)
	events = append(events, newEvent)
	merged, _ := json.Marshal(events)
	if err := s.db.WithContext(ctx).Model(&existing).Updates(map[string]any{"events": merged, "updated_at": now}).Error; err != nil {
		return bucket, err
	}
	bucket.ID = existing.ID
	bucket.Events = merged
	bucket.WindowEnd = existing.WindowEnd
	return bucket, nil
}

func (s *Store) GetDigestBucket(ctx context.Context, envID, digestKey, subscriberPK string) (*domain.DigestBucket, error) {
	var m digestBucketModel
	err := s.db.WithContext(ctx).
		Where("env_id = ? AND digest_key = ? AND subscriber_pk = ? AND window_end > ?", envID, digestKey, subscriberPK, time.Now()).
		Order("window_end ASC").First(&m).Error
	if err != nil {
		return nil, err
	}
	return m.toDomain()
}

func (s *Store) GetDigestBucketByID(ctx context.Context, id string) (*domain.DigestBucket, error) {
	var m digestBucketModel
	err := s.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return m.toDomain()
}

func (s *Store) DeleteDigestBucket(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&digestBucketModel{}, "id = ?", id).Error
}
