package filestore

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/SolaTyolo/herald/internal/domain"
)

func (s *Store) subscribersDir(envID string) string {
	return s.collection(envID, "subscribers")
}

func (s *Store) subscriberPath(envID, subscriberID string) string {
	return filepath.Join(s.subscribersDir(envID), safeFilename(subscriberID)+".json")
}

func (s *Store) preferencesDir(envID string) string {
	return s.collection(envID, "preferences")
}

func (s *Store) preferencePath(envID, id string) string {
	return filepath.Join(s.preferencesDir(envID), id+".json")
}

func (s *Store) UpsertSubscriber(ctx context.Context, envID string, sub *domain.Subscriber) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	path := s.subscriberPath(envID, sub.SubscriberID)
	if existing, err := readSubscriberFile(path); err == nil {
		sub.ID = existing.ID
		sub.CreatedAt = existing.CreatedAt
	} else if !isNotFound(err) {
		return err
	} else {
		if sub.ID == "" {
			sub.ID = uuid.NewString()
		}
		if sub.CreatedAt.IsZero() {
			sub.CreatedAt = now
		}
	}
	sub.EnvID = envID
	sub.UpdatedAt = now
	return writeJSON(path, sub)
}

func readSubscriberFile(path string) (*domain.Subscriber, error) {
	var sub domain.Subscriber
	if err := readJSON(path, &sub); err != nil {
		return nil, err
	}
	return &sub, nil
}

func (s *Store) GetSubscriber(ctx context.Context, envID, subscriberID string) (*domain.Subscriber, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return readSubscriberFile(s.subscriberPath(envID, subscriberID))
}

func (s *Store) GetSubscriberByPK(ctx context.Context, pk string) (*domain.Subscriber, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

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

func (s *Store) ListSubscribers(ctx context.Context, envID string, limit, offset int) ([]domain.Subscriber, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := listJSONFiles(s.subscribersDir(envID))
	if err != nil {
		return nil, err
	}
	out := make([]domain.Subscriber, 0, len(files))
	for _, path := range files {
		sub, err := readSubscriberFile(path)
		if err != nil {
			return nil, err
		}
		out = append(out, *sub)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	out = paginate(out, limit, offset)
	return out, nil
}

func (s *Store) DeleteSubscriber(ctx context.Context, envID, subscriberID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return deleteFile(s.subscriberPath(envID, subscriberID))
}

func (s *Store) SetPreference(ctx context.Context, envID, subscriberID string, pref domain.SubscriberPreference) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, err := readSubscriberFile(s.subscriberPath(envID, subscriberID))
	if err != nil {
		return err
	}
	pref.SubscriberPK = sub.ID

	files, err := listJSONFiles(s.preferencesDir(envID))
	if err != nil {
		return err
	}
	for _, path := range files {
		var existing domain.SubscriberPreference
		if err := readJSON(path, &existing); err != nil {
			return err
		}
		if existing.SubscriberPK != pref.SubscriberPK || existing.Channel != pref.Channel {
			continue
		}
		if !workflowIDEqual(existing.WorkflowID, pref.WorkflowID) {
			continue
		}
		existing.Enabled = pref.Enabled
		return writeJSON(path, &existing)
	}

	pref.ID = uuid.NewString()
	return writeJSON(s.preferencePath(envID, pref.ID), &pref)
}

func workflowIDEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func (s *Store) GetPreferences(ctx context.Context, subscriberPK string, workflowID *string) ([]domain.SubscriberPreference, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	envIDs, err := s.listEnvIDs()
	if err != nil {
		return nil, err
	}
	var out []domain.SubscriberPreference
	for _, envID := range envIDs {
		files, err := listJSONFiles(s.preferencesDir(envID))
		if err != nil {
			return nil, err
		}
		for _, path := range files {
			var pref domain.SubscriberPreference
			if err := readJSON(path, &pref); err != nil {
				return nil, err
			}
			if pref.SubscriberPK != subscriberPK {
				continue
			}
			if workflowID != nil {
				if pref.WorkflowID == nil || *pref.WorkflowID != *workflowID {
					continue
				}
			}
			out = append(out, pref)
		}
	}
	return out, nil
}
