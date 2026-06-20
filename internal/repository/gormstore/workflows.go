package gormstore

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/SolaTyolo/herald/internal/domain"
)

func (s *Store) CreateWorkflow(ctx context.Context, envID string, wf *domain.Workflow) error {
	wf.ID = uuid.NewString()
	wf.EnvID = envID
	now := time.Now()
	wf.CreatedAt, wf.UpdatedAt = now, now
	prefs, _ := json.Marshal(wf.PreferenceSettings)

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		m := workflowModel{
			ID: wf.ID, EnvID: envID, Name: wf.Name, TriggerID: wf.TriggerID,
			Critical: wf.Critical, PreferenceSettings: prefs, Active: wf.Active,
			CreatedAt: wf.CreatedAt, UpdatedAt: wf.UpdatedAt,
		}
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
		for i := range wf.Steps {
			step := &wf.Steps[i]
			step.ID = uuid.NewString()
			step.WorkflowID = wf.ID
			step.StepOrder = i + 1
			step.CreatedAt = now
			tmpl, _ := json.Marshal(step.Template)
			filters, _ := json.Marshal(step.Filters)
			meta, _ := json.Marshal(step.Metadata)
			if err := tx.Create(&workflowStepModel{
				ID: step.ID, WorkflowID: wf.ID, StepOrder: step.StepOrder, StepType: string(step.Type),
				Template: tmpl, Filters: filters, Metadata: meta, CreatedAt: step.CreatedAt,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) GetWorkflowByTrigger(ctx context.Context, envID, triggerID string) (*domain.Workflow, error) {
	return s.getWorkflow(ctx, envID, "trigger_id = ? AND active = ?", triggerID, true)
}

func (s *Store) GetWorkflow(ctx context.Context, envID, id string) (*domain.Workflow, error) {
	return s.getWorkflow(ctx, envID, "id = ?", id)
}

func (s *Store) getWorkflow(ctx context.Context, envID string, query string, args ...any) (*domain.Workflow, error) {
	var m workflowModel
	q := s.db.WithContext(ctx).Where("env_id = ?", envID).Where(query, args...)
	if err := q.First(&m).Error; err != nil {
		return nil, err
	}
	wf, err := m.toDomain()
	if err != nil {
		return nil, err
	}
	wf.Steps, err = s.loadWorkflowSteps(ctx, wf.ID)
	return wf, err
}

func (s *Store) loadWorkflowSteps(ctx context.Context, workflowID string) ([]domain.WorkflowStep, error) {
	var rows []workflowStepModel
	if err := s.db.WithContext(ctx).Where("workflow_id = ?", workflowID).Order("step_order ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.WorkflowStep, 0, len(rows))
	for _, r := range rows {
		st, err := r.toDomain()
		if err != nil {
			return nil, err
		}
		out = append(out, *st)
	}
	return out, nil
}

func (s *Store) ListWorkflows(ctx context.Context, envID string) ([]domain.Workflow, error) {
	var rows []workflowModel
	if err := s.db.WithContext(ctx).Where("env_id = ?", envID).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Workflow, 0, len(rows))
	for _, r := range rows {
		wf, err := r.toDomain()
		if err != nil {
			return nil, err
		}
		wf.Steps, _ = s.loadWorkflowSteps(ctx, wf.ID)
		out = append(out, *wf)
	}
	return out, nil
}

func (s *Store) UpdateWorkflow(ctx context.Context, envID string, wf *domain.Workflow) error {
	prefs, _ := json.Marshal(wf.PreferenceSettings)
	wf.UpdatedAt = time.Now()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&workflowModel{}).Where("id = ? AND env_id = ?", wf.ID, envID).Updates(map[string]any{
			"name": wf.Name, "trigger_id": wf.TriggerID, "critical": wf.Critical,
			"preference_settings": prefs, "active": wf.Active, "updated_at": wf.UpdatedAt,
		})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Where("workflow_id = ?", wf.ID).Delete(&workflowStepModel{}).Error; err != nil {
			return err
		}
		for i := range wf.Steps {
			step := &wf.Steps[i]
			if step.ID == "" {
				step.ID = uuid.NewString()
			}
			step.WorkflowID = wf.ID
			step.StepOrder = i + 1
			tmpl, _ := json.Marshal(step.Template)
			filters, _ := json.Marshal(step.Filters)
			meta, _ := json.Marshal(step.Metadata)
			if err := tx.Create(&workflowStepModel{
				ID: step.ID, WorkflowID: wf.ID, StepOrder: step.StepOrder, StepType: string(step.Type),
				Template: tmpl, Filters: filters, Metadata: meta, CreatedAt: time.Now(),
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) DeleteWorkflow(ctx context.Context, envID, id string) error {
	res := s.db.WithContext(ctx).Where("id = ? AND env_id = ?", id, envID).Delete(&workflowModel{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
