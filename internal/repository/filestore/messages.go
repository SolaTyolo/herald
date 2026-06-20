package filestore

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/SolaTyolo/herald/internal/domain"
)

func (s *Store) messagesDir(envID string) string {
	return s.collection(envID, "messages")
}

func (s *Store) messagePath(envID, id string) string {
	return filepath.Join(s.messagesDir(envID), id+".json")
}

func (s *Store) CreateMessage(ctx context.Context, msg *domain.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg.ID = uuid.NewString()
	msg.CreatedAt = time.Now()
	if msg.Metadata == nil {
		msg.Metadata = json.RawMessage("{}")
	}
	return writeJSON(s.messagePath(msg.EnvID, msg.ID), msg)
}

func (s *Store) ListMessages(ctx context.Context, subscriberPK string, unreadOnly bool, limit, offset int) ([]domain.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	envIDs, err := s.listEnvIDs()
	if err != nil {
		return nil, err
	}
	var out []domain.Message
	for _, envID := range envIDs {
		files, err := listJSONFiles(s.messagesDir(envID))
		if err != nil {
			return nil, err
		}
		for _, path := range files {
			var msg domain.Message
			if err := readJSON(path, &msg); err != nil {
				return nil, err
			}
			if msg.SubscriberPK != subscriberPK {
				continue
			}
			if unreadOnly && msg.Read {
				continue
			}
			out = append(out, msg)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	out = paginate(out, limit, offset)
	return out, nil
}

func (s *Store) UpdateMessageForSubscriber(ctx context.Context, envID, subscriberPK, messageID string, read, archived *bool) (*domain.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if read == nil && archived == nil {
		return nil, fmt.Errorf("nothing to update")
	}
	path := s.messagePath(envID, messageID)
	var msg domain.Message
	if err := readJSON(path, &msg); err != nil {
		return nil, err
	}
	if msg.SubscriberPK != subscriberPK {
		return nil, ErrNotFound
	}
	if read != nil {
		msg.Read = *read
	}
	if archived != nil {
		msg.Archived = *archived
	}
	if err := writeJSON(path, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
