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

func (s *Store) executionLogsDir(envID string) string {
	return s.collection(envID, "execution_logs")
}

func (s *Store) executionLogPath(envID, id string) string {
	return filepath.Join(s.executionLogsDir(envID), id+".json")
}

func (s *Store) CreateExecutionLog(ctx context.Context, log *domain.ExecutionLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.ID = uuid.NewString()
	log.CreatedAt = time.Now()
	if log.Metadata == nil {
		log.Metadata = json.RawMessage("{}")
	}
	return writeJSON(s.executionLogPath(log.EnvID, log.ID), log)
}

func (s *Store) ListExecutionLogs(ctx context.Context, envID, transactionID string) ([]domain.ExecutionLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := listJSONFiles(s.executionLogsDir(envID))
	if err != nil {
		return nil, err
	}
	out := make([]domain.ExecutionLog, 0, len(files))
	for _, path := range files {
		var log domain.ExecutionLog
		if err := readJSON(path, &log); err != nil {
			return nil, err
		}
		if log.TransactionID != transactionID {
			continue
		}
		out = append(out, log)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}
