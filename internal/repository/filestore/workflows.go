package filestore

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/SolaTyolo/herald/internal/domain"
)

func (s *Store) workflowsDir(envID string) string {
	return s.collection(envID, "workflows")
}

func (s *Store) workflowPath(envID, id string) string {
	return filepath.Join(s.workflowsDir(envID), id+".json")
}

func (s *Store) CreateWorkflow(ctx context.Context, envID string, wf *domain.Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	wf.ID = uuid.NewString()
	wf.EnvID = envID
	now := time.Now()
	wf.CreatedAt, wf.UpdatedAt = now, now
	for i := range wf.Steps {
		step := &wf.Steps[i]
		step.ID = uuid.NewString()
		step.WorkflowID = wf.ID
		step.StepOrder = i + 1
		step.CreatedAt = now
	}
	return writeJSON(s.workflowPath(envID, wf.ID), wf)
}

func (s *Store) GetWorkflowByTrigger(ctx context.Context, envID, triggerID string) (*domain.Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.findWorkflow(envID, func(wf *domain.Workflow) bool {
		return wf.TriggerID == triggerID && wf.Active
	})
}

func (s *Store) GetWorkflow(ctx context.Context, envID, id string) (*domain.Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var wf domain.Workflow
	if err := readJSON(s.workflowPath(envID, id), &wf); err != nil {
		return nil, err
	}
	return &wf, nil
}

func (s *Store) findWorkflow(envID string, match func(*domain.Workflow) bool) (*domain.Workflow, error) {
	files, err := listJSONFiles(s.workflowsDir(envID))
	if err != nil {
		return nil, err
	}
	for _, path := range files {
		var wf domain.Workflow
		if err := readJSON(path, &wf); err != nil {
			return nil, err
		}
		if match(&wf) {
			return &wf, nil
		}
	}
	return nil, ErrNotFound
}

func (s *Store) ListWorkflows(ctx context.Context, envID string) ([]domain.Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := listJSONFiles(s.workflowsDir(envID))
	if err != nil {
		return nil, err
	}
	out := make([]domain.Workflow, 0, len(files))
	for _, path := range files {
		var wf domain.Workflow
		if err := readJSON(path, &wf); err != nil {
			return nil, err
		}
		out = append(out, wf)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (s *Store) UpdateWorkflow(ctx context.Context, envID string, wf *domain.Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.workflowPath(envID, wf.ID)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	wf.UpdatedAt = time.Now()
	for i := range wf.Steps {
		step := &wf.Steps[i]
		if step.ID == "" {
			step.ID = uuid.NewString()
		}
		step.WorkflowID = wf.ID
		step.StepOrder = i + 1
		if step.CreatedAt.IsZero() {
			step.CreatedAt = wf.UpdatedAt
		}
	}
	return writeJSON(path, wf)
}

func (s *Store) DeleteWorkflow(ctx context.Context, envID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return deleteFile(s.workflowPath(envID, id))
}
