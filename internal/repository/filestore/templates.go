package filestore

import (
	"context"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/SolaTyolo/herald/internal/domain"
)

func (s *Store) templatesDir(envID string) string {
	return s.collection(envID, "templates")
}

func (s *Store) templatePath(envID, id string) string {
	return filepath.Join(s.templatesDir(envID), id+".json")
}

func (s *Store) CreateTemplate(ctx context.Context, envID string, tmpl *domain.NotificationTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	tmpl.ID = uuid.NewString()
	tmpl.EnvID = envID
	tmpl.CreatedAt, tmpl.UpdatedAt = now, now
	if tmpl.Generation == "" {
		tmpl.Generation = domain.TemplateGenerationDynamic
	}
	if tmpl.Versions == nil {
		tmpl.Versions = []domain.TemplateVersion{}
	}
	return writeJSON(s.templatePath(envID, tmpl.ID), tmpl)
}

func (s *Store) GetTemplate(ctx context.Context, envID, id string) (*domain.NotificationTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.readTemplate(envID, id)
}

func (s *Store) ListTemplates(ctx context.Context, envID string) ([]domain.NotificationTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := listJSONFiles(s.templatesDir(envID))
	if err != nil {
		return nil, err
	}
	out := make([]domain.NotificationTemplate, 0, len(files))
	for _, path := range files {
		var tmpl domain.NotificationTemplate
		if err := readJSON(path, &tmpl); err != nil {
			return nil, err
		}
		out = append(out, tmpl)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out, nil
}

func (s *Store) UpdateTemplate(ctx context.Context, envID string, tmpl *domain.NotificationTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.readTemplate(envID, tmpl.ID)
	if err != nil {
		return err
	}
	existing.Name = tmpl.Name
	existing.UpdatedAt = time.Now()
	tmpl.UpdatedAt = existing.UpdatedAt
	tmpl.EnvID = existing.EnvID
	tmpl.Generation = existing.Generation
	tmpl.CreatedAt = existing.CreatedAt
	tmpl.Versions = existing.Versions
	return writeJSON(s.templatePath(envID, tmpl.ID), existing)
}

func (s *Store) DeleteTemplate(ctx context.Context, envID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return deleteFile(s.templatePath(envID, id))
}

func (s *Store) CreateTemplateVersion(ctx context.Context, envID, templateID string, ver *domain.TemplateVersion) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tmpl, err := s.readTemplate(envID, templateID)
	if err != nil {
		return err
	}
	now := time.Now()
	ver.ID = uuid.NewString()
	ver.TemplateID = templateID
	ver.CreatedAt, ver.UpdatedAt = now, now
	if ver.Editor == "" {
		ver.Editor = "code"
	}
	if len(ver.TestData) == 0 {
		ver.TestData = []byte("{}")
	}
	ver.Active = ver.Active || !hasActiveVersion(tmpl.Versions)
	if ver.Active {
		deactivateVersions(tmpl.Versions)
	}
	tmpl.Versions = append(tmpl.Versions, *ver)
	tmpl.UpdatedAt = now
	return writeJSON(s.templatePath(envID, templateID), tmpl)
}

func (s *Store) UpdateTemplateVersion(ctx context.Context, envID, templateID string, ver *domain.TemplateVersion) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tmpl, err := s.readTemplate(envID, templateID)
	if err != nil {
		return err
	}
	idx := versionIndex(tmpl.Versions, ver.ID)
	if idx < 0 {
		return ErrNotFound
	}
	prev := tmpl.Versions[idx]
	now := time.Now()
	ver.TemplateID = templateID
	ver.CreatedAt = prev.CreatedAt
	ver.UpdatedAt = now
	if ver.Editor == "" {
		ver.Editor = prev.Editor
	}
	if len(ver.TestData) == 0 {
		ver.TestData = prev.TestData
	}
	if ver.Active {
		deactivateVersions(tmpl.Versions)
	}
	tmpl.Versions[idx] = *ver
	tmpl.UpdatedAt = now
	return writeJSON(s.templatePath(envID, templateID), tmpl)
}

func (s *Store) DeleteTemplateVersion(ctx context.Context, envID, templateID, versionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tmpl, err := s.readTemplate(envID, templateID)
	if err != nil {
		return err
	}
	idx := versionIndex(tmpl.Versions, versionID)
	if idx < 0 {
		return ErrNotFound
	}
	wasActive := tmpl.Versions[idx].Active
	tmpl.Versions = append(tmpl.Versions[:idx], tmpl.Versions[idx+1:]...)
	if wasActive && len(tmpl.Versions) > 0 {
		tmpl.Versions[len(tmpl.Versions)-1].Active = true
	}
	tmpl.UpdatedAt = time.Now()
	return writeJSON(s.templatePath(envID, templateID), tmpl)
}

func (s *Store) readTemplate(envID, id string) (*domain.NotificationTemplate, error) {
	var tmpl domain.NotificationTemplate
	if err := readJSON(s.templatePath(envID, id), &tmpl); err != nil {
		return nil, err
	}
	return &tmpl, nil
}

func hasActiveVersion(versions []domain.TemplateVersion) bool {
	for _, v := range versions {
		if v.Active {
			return true
		}
	}
	return false
}

func deactivateVersions(versions []domain.TemplateVersion) {
	for i := range versions {
		versions[i].Active = false
	}
}

func versionIndex(versions []domain.TemplateVersion, id string) int {
	for i := range versions {
		if versions[i].ID == id {
			return i
		}
	}
	return -1
}
