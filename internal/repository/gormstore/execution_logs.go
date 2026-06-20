package gormstore

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/SolaTyolo/herald/internal/domain"
)

func (s *Store) CreateExecutionLog(ctx context.Context, log *domain.ExecutionLog) error {
	log.ID = uuid.NewString()
	log.CreatedAt = time.Now()
	if log.Metadata == nil {
		log.Metadata = json.RawMessage("{}")
	}
	return s.db.WithContext(ctx).Create(executionLogModelFromDomain(log)).Error
}

func (s *Store) ListExecutionLogs(ctx context.Context, envID, transactionID string) ([]domain.ExecutionLog, error) {
	var rows []executionLogModel
	if err := s.db.WithContext(ctx).Where("env_id = ? AND transaction_id = ?", envID, transactionID).
		Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ExecutionLog, 0, len(rows))
	for _, r := range rows {
		l, _ := r.toDomain()
		out = append(out, *l)
	}
	return out, nil
}
