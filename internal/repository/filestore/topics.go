package filestore

import (
	"context"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/SolaTyolo/herald/internal/domain"
)

func (s *Store) topicsDir() string {
	return s.collection("topics")
}

func (s *Store) topicPath(topicKey string) string {
	return filepath.Join(s.topicsDir(), safeFilename(topicKey)+".json")
}

func (s *Store) topicSubscriptionsDir() string {
	return s.collection("topic_subscriptions")
}

func (s *Store) topicSubscriptionPath(id string) string {
	return filepath.Join(s.topicSubscriptionsDir(), id+".json")
}

func (s *Store) CreateTopic(ctx context.Context, topic *domain.Topic) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	topic.ID = uuid.NewString()
	topic.CreatedAt = time.Now()
	return writeJSON(s.topicPath(topic.TopicKey), topic)
}

func (s *Store) GetTopic(ctx context.Context, topicKey string) (*domain.Topic, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var topic domain.Topic
	if err := readJSON(s.topicPath(topicKey), &topic); err != nil {
		return nil, err
	}
	return &topic, nil
}

func (s *Store) ListTopics(ctx context.Context) ([]domain.Topic, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := listJSONFiles(s.topicsDir())
	if err != nil {
		return nil, err
	}
	out := make([]domain.Topic, 0, len(files))
	for _, path := range files {
		var topic domain.Topic
		if err := readJSON(path, &topic); err != nil {
			return nil, err
		}
		out = append(out, topic)
	}
	return out, nil
}

func (s *Store) DeleteTopic(ctx context.Context, topicKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return deleteFile(s.topicPath(topicKey))
}

func (s *Store) AddTopicSubscription(ctx context.Context, topicKey, subscriberID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	topic, err := s.readTopicLocked(topicKey)
	if err != nil {
		return err
	}
	sub, err := readSubscriberFile(s.subscriberPath(subscriberID))
	if err != nil {
		return err
	}
	files, err := listJSONFiles(s.topicSubscriptionsDir())
	if err != nil {
		return err
	}
	for _, path := range files {
		var existing domain.TopicSubscription
		if err := readJSON(path, &existing); err != nil {
			return err
		}
		if existing.TopicID == topic.ID && existing.SubscriberPK == sub.ID {
			return nil
		}
	}
	subscription := domain.TopicSubscription{
		ID:           uuid.NewString(),
		TopicID:      topic.ID,
		SubscriberPK: sub.ID,
		CreatedAt:    time.Now(),
	}
	return writeJSON(s.topicSubscriptionPath(subscription.ID), &subscription)
}

func (s *Store) RemoveTopicSubscription(ctx context.Context, topicKey, subscriberID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	topic, err := s.readTopicLocked(topicKey)
	if err != nil {
		return err
	}
	sub, err := readSubscriberFile(s.subscriberPath(subscriberID))
	if err != nil {
		return err
	}
	files, err := listJSONFiles(s.topicSubscriptionsDir())
	if err != nil {
		return err
	}
	for _, path := range files {
		var existing domain.TopicSubscription
		if err := readJSON(path, &existing); err != nil {
			return err
		}
		if existing.TopicID == topic.ID && existing.SubscriberPK == sub.ID {
			return deleteFile(path)
		}
	}
	return nil
}

func (s *Store) ListTopicSubscribers(ctx context.Context, topicID string, excludePK *string) ([]domain.Subscriber, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := listJSONFiles(s.topicSubscriptionsDir())
	if err != nil {
		return nil, err
	}
	var subscriberPKs []string
	for _, path := range files {
		var sub domain.TopicSubscription
		if err := readJSON(path, &sub); err != nil {
			return nil, err
		}
		if sub.TopicID != topicID {
			continue
		}
		if excludePK != nil && sub.SubscriberPK == *excludePK {
			continue
		}
		subscriberPKs = append(subscriberPKs, sub.SubscriberPK)
	}
	out := make([]domain.Subscriber, 0, len(subscriberPKs))
	for _, pk := range subscriberPKs {
		sub, err := s.getSubscriberByPKLocked(pk)
		if err != nil {
			return nil, err
		}
		out = append(out, *sub)
	}
	return out, nil
}

func (s *Store) readTopicLocked(topicKey string) (*domain.Topic, error) {
	var topic domain.Topic
	if err := readJSON(s.topicPath(topicKey), &topic); err != nil {
		return nil, err
	}
	return &topic, nil
}
