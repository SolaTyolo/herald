package filestore

import (
	"context"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/SolaTyolo/herald/internal/domain"
)

func (s *Store) topicsDir(envID string) string {
	return s.collection(envID, "topics")
}

func (s *Store) topicPath(envID, topicKey string) string {
	return filepath.Join(s.topicsDir(envID), safeFilename(topicKey)+".json")
}

func (s *Store) topicSubscriptionsDir(envID string) string {
	return s.collection(envID, "topic_subscriptions")
}

func (s *Store) topicSubscriptionPath(envID, id string) string {
	return filepath.Join(s.topicSubscriptionsDir(envID), id+".json")
}

func (s *Store) CreateTopic(ctx context.Context, envID string, topic *domain.Topic) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	topic.ID = uuid.NewString()
	topic.EnvID = envID
	topic.CreatedAt = time.Now()
	return writeJSON(s.topicPath(envID, topic.TopicKey), topic)
}

func (s *Store) GetTopic(ctx context.Context, envID, topicKey string) (*domain.Topic, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var topic domain.Topic
	if err := readJSON(s.topicPath(envID, topicKey), &topic); err != nil {
		return nil, err
	}
	return &topic, nil
}

func (s *Store) ListTopics(ctx context.Context, envID string) ([]domain.Topic, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := listJSONFiles(s.topicsDir(envID))
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

func (s *Store) DeleteTopic(ctx context.Context, envID, topicKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return deleteFile(s.topicPath(envID, topicKey))
}

func (s *Store) AddTopicSubscription(ctx context.Context, envID, topicKey, subscriberID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	topic, err := s.readTopicLocked(envID, topicKey)
	if err != nil {
		return err
	}
	sub, err := readSubscriberFile(s.subscriberPath(envID, subscriberID))
	if err != nil {
		return err
	}
	files, err := listJSONFiles(s.topicSubscriptionsDir(envID))
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
	return writeJSON(s.topicSubscriptionPath(envID, subscription.ID), &subscription)
}

func (s *Store) RemoveTopicSubscription(ctx context.Context, envID, topicKey, subscriberID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	topic, err := s.readTopicLocked(envID, topicKey)
	if err != nil {
		return err
	}
	sub, err := readSubscriberFile(s.subscriberPath(envID, subscriberID))
	if err != nil {
		return err
	}
	files, err := listJSONFiles(s.topicSubscriptionsDir(envID))
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

	envIDs, err := s.listEnvIDs()
	if err != nil {
		return nil, err
	}
	var subscriberPKs []string
	for _, envID := range envIDs {
		files, err := listJSONFiles(s.topicSubscriptionsDir(envID))
		if err != nil {
			return nil, err
		}
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

func (s *Store) readTopicLocked(envID, topicKey string) (*domain.Topic, error) {
	var topic domain.Topic
	if err := readJSON(s.topicPath(envID, topicKey), &topic); err != nil {
		return nil, err
	}
	return &topic, nil
}

func (s *Store) getSubscriberByPKLocked(pk string) (*domain.Subscriber, error) {
	envIDs, err := s.listEnvIDs()
	if err != nil {
		return nil, err
	}
	for _, envID := range envIDs {
		files, err := listJSONFiles(s.subscribersDir(envID))
		if err != nil {
			return nil, err
		}
		for _, path := range files {
			sub, err := readSubscriberFile(path)
			if err != nil {
				return nil, err
			}
			if sub.ID == pk {
				return sub, nil
			}
		}
	}
	return nil, ErrNotFound
}
