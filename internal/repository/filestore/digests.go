package filestore

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/SolaTyolo/herald/internal/domain"
)

func (s *Store) digestsDir() string {
	return s.collection("digests")
}

func (s *Store) digestPath(id string) string {
	return filepath.Join(s.digestsDir(), id+".json")
}

func (s *Store) UpsertDigestBucket(ctx context.Context, bucket *domain.DigestBucket) (*domain.DigestBucket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if bucket.ID == "" {
		bucket.ID = uuid.NewString()
	}
	now := time.Now()
	bucket.UpdatedAt = now
	if bucket.CreatedAt.IsZero() {
		bucket.CreatedAt = now
	}

	path := s.digestPath(bucket.ID)
	if _, err := os.Stat(path); err == nil {
		if err := writeJSON(path, bucket); err != nil {
			return bucket, err
		}
		return bucket, nil
	} else if !os.IsNotExist(err) {
		return bucket, err
	}

	existing, err := s.findActiveDigestBucketLocked(bucket.DigestKey, bucket.SubscriberPK, now)
	if err != nil {
		if isNotFound(err) {
			if err := writeJSON(path, bucket); err != nil {
				return bucket, err
			}
			return bucket, nil
		}
		return bucket, err
	}

	var events []json.RawMessage
	_ = json.Unmarshal(existing.Events, &events)
	var newEvent json.RawMessage
	_ = json.Unmarshal(bucket.Events, &newEvent)
	events = append(events, newEvent)
	merged, _ := json.Marshal(events)
	existing.Events = merged
	existing.UpdatedAt = now
	if err := writeJSON(s.digestPath(existing.ID), &existing); err != nil {
		return bucket, err
	}
	bucket.ID = existing.ID
	bucket.Events = merged
	bucket.WindowEnd = existing.WindowEnd
	return bucket, nil
}

func (s *Store) GetDigestBucket(ctx context.Context, digestKey, subscriberPK string) (*domain.DigestBucket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.findActiveDigestBucketLocked(digestKey, subscriberPK, time.Now())
}

func (s *Store) GetDigestBucketByID(ctx context.Context, id string) (*domain.DigestBucket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var bucket domain.DigestBucket
	if err := readJSON(s.digestPath(id), &bucket); err != nil {
		return nil, err
	}
	return &bucket, nil
}

func (s *Store) DeleteDigestBucket(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := deleteFile(s.digestPath(id))
	if isNotFound(err) {
		return nil
	}
	return err
}

func (s *Store) findActiveDigestBucketLocked(digestKey, subscriberPK string, now time.Time) (*domain.DigestBucket, error) {
	files, err := listJSONFiles(s.digestsDir())
	if err != nil {
		return nil, err
	}
	var matches []domain.DigestBucket
	for _, path := range files {
		var bucket domain.DigestBucket
		if err := readJSON(path, &bucket); err != nil {
			return nil, err
		}
		if bucket.DigestKey != digestKey || bucket.SubscriberPK != subscriberPK || !bucket.WindowEnd.After(now) {
			continue
		}
		matches = append(matches, bucket)
	}
	if len(matches) == 0 {
		return nil, ErrNotFound
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].WindowEnd.Before(matches[j].WindowEnd)
	})
	return &matches[0], nil
}
