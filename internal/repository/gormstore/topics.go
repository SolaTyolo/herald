package gormstore

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/SolaTyolo/herald/internal/domain"
)

func (s *Store) CreateTopic(ctx context.Context, envID string, topic *domain.Topic) error {
	topic.ID = uuid.NewString()
	topic.EnvID = envID
	topic.CreatedAt = time.Now()
	m := topicModel{ID: topic.ID, EnvID: envID, TopicKey: topic.TopicKey, Name: topic.Name, CreatedAt: topic.CreatedAt}
	return s.db.WithContext(ctx).Create(&m).Error
}

func (s *Store) GetTopic(ctx context.Context, envID, topicKey string) (*domain.Topic, error) {
	var m topicModel
	err := s.db.WithContext(ctx).Where("env_id = ? AND topic_key = ?", envID, topicKey).First(&m).Error
	if err != nil {
		return nil, err
	}
	return m.toDomain()
}

func (s *Store) ListTopics(ctx context.Context, envID string) ([]domain.Topic, error) {
	var rows []topicModel
	if err := s.db.WithContext(ctx).Where("env_id = ?", envID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Topic, 0, len(rows))
	for _, r := range rows {
		t, _ := r.toDomain()
		out = append(out, *t)
	}
	return out, nil
}

func (s *Store) DeleteTopic(ctx context.Context, envID, topicKey string) error {
	res := s.db.WithContext(ctx).Where("env_id = ? AND topic_key = ?", envID, topicKey).Delete(&topicModel{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Store) AddTopicSubscription(ctx context.Context, envID, topicKey, subscriberID string) error {
	topic, err := s.GetTopic(ctx, envID, topicKey)
	if err != nil {
		return err
	}
	sub, err := s.GetSubscriber(ctx, envID, subscriberID)
	if err != nil {
		return err
	}
	m := topicSubscriptionModel{
		ID: uuid.NewString(), TopicID: topic.ID, SubscriberPK: sub.ID, CreatedAt: time.Now(),
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&m).Error
}

func (s *Store) RemoveTopicSubscription(ctx context.Context, envID, topicKey, subscriberID string) error {
	topic, err := s.GetTopic(ctx, envID, topicKey)
	if err != nil {
		return err
	}
	sub, err := s.GetSubscriber(ctx, envID, subscriberID)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Where("topic_id = ? AND subscriber_pk = ?", topic.ID, sub.ID).
		Delete(&topicSubscriptionModel{}).Error
}

func (s *Store) ListTopicSubscribers(ctx context.Context, topicID string, excludePK *string) ([]domain.Subscriber, error) {
	q := s.db.WithContext(ctx).
		Table("subscribers").
		Joins("JOIN topic_subscriptions ON topic_subscriptions.subscriber_pk = subscribers.id").
		Where("topic_subscriptions.topic_id = ?", topicID)
	if excludePK != nil {
		q = q.Where("subscribers.id != ?", *excludePK)
	}
	var rows []subscriberModel
	if err := q.Find(&rows).Error; err != nil {
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
