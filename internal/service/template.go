package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SolaTyolo/herald/internal/domain"
	"github.com/SolaTyolo/herald/internal/repository"
	"github.com/SolaTyolo/herald/internal/template"
)

type TemplateService struct {
	store repository.Store
}

func NewTemplateService(store repository.Store) *TemplateService {
	return &TemplateService{store: store}
}

func (s *TemplateService) Create(ctx context.Context, envID string, tmpl *domain.NotificationTemplate) error {
	tmpl.Name = strings.TrimSpace(tmpl.Name)
	if tmpl.Name == "" {
		return fmt.Errorf("name required")
	}
	if tmpl.Generation == "" {
		tmpl.Generation = domain.TemplateGenerationDynamic
	}
	if tmpl.Generation != domain.TemplateGenerationDynamic {
		return fmt.Errorf("generation must be %s", domain.TemplateGenerationDynamic)
	}
	return s.store.CreateTemplate(ctx, envID, tmpl)
}

func (s *TemplateService) Get(ctx context.Context, envID, id string) (*domain.NotificationTemplate, error) {
	return s.store.GetTemplate(ctx, envID, id)
}

func (s *TemplateService) List(ctx context.Context, envID string) ([]domain.NotificationTemplate, error) {
	return s.store.ListTemplates(ctx, envID)
}

func (s *TemplateService) Update(ctx context.Context, envID string, tmpl *domain.NotificationTemplate) error {
	tmpl.Name = strings.TrimSpace(tmpl.Name)
	if tmpl.Name == "" {
		return fmt.Errorf("name required")
	}
	return s.store.UpdateTemplate(ctx, envID, tmpl)
}

func (s *TemplateService) Delete(ctx context.Context, envID, id string) error {
	return s.store.DeleteTemplate(ctx, envID, id)
}

func (s *TemplateService) CreateVersion(ctx context.Context, envID, templateID string, ver *domain.TemplateVersion) error {
	if err := validateVersion(ver); err != nil {
		return err
	}
	return s.store.CreateTemplateVersion(ctx, envID, templateID, ver)
}

func (s *TemplateService) UpdateVersion(ctx context.Context, envID, templateID string, ver *domain.TemplateVersion) error {
	if err := validateVersion(ver); err != nil {
		return err
	}
	return s.store.UpdateTemplateVersion(ctx, envID, templateID, ver)
}

func (s *TemplateService) DeleteVersion(ctx context.Context, envID, templateID, versionID string) error {
	return s.store.DeleteTemplateVersion(ctx, envID, templateID, versionID)
}

type TemplatePreview struct {
	Subject      string `json:"subject"`
	HTMLContent  string `json:"htmlContent"`
	PlainContent string `json:"plainContent"`
}

func (s *TemplateService) Preview(ctx context.Context, envID, templateID, versionID string, data, subscriber map[string]any) (*TemplatePreview, error) {
	tmpl, err := s.store.GetTemplate(ctx, envID, templateID)
	if err != nil {
		return nil, err
	}
	ver := tmpl.ActiveVersion()
	if versionID != "" {
		ver = nil
		for i := range tmpl.Versions {
			if tmpl.Versions[i].ID == versionID {
				ver = &tmpl.Versions[i]
				break
			}
		}
		if ver == nil {
			return nil, repository.ErrNotFound
		}
	}
	if ver == nil {
		return nil, fmt.Errorf("template %s has no active version", templateID)
	}
	if data == nil {
		data = map[string]any{}
		if len(ver.TestData) > 0 {
			_ = json.Unmarshal(ver.TestData, &data)
		}
	}
	renderCtx := template.RenderContext{Payload: data, Subscriber: subscriber}
	subject, err := template.Render(ver.Subject, renderCtx)
	if err != nil {
		return nil, err
	}
	html, err := template.Render(ver.HTMLContent, renderCtx)
	if err != nil {
		return nil, err
	}
	plain, err := template.Render(ver.PlainContent, renderCtx)
	if err != nil {
		return nil, err
	}
	return &TemplatePreview{Subject: subject, HTMLContent: html, PlainContent: plain}, nil
}

func validateVersion(ver *domain.TemplateVersion) error {
	ver.Name = strings.TrimSpace(ver.Name)
	if ver.Name == "" {
		return fmt.Errorf("name required")
	}
	if strings.TrimSpace(ver.Subject) == "" && strings.TrimSpace(ver.HTMLContent) == "" && strings.TrimSpace(ver.PlainContent) == "" {
		return fmt.Errorf("subject, htmlContent, or plainContent required")
	}
	if ver.Editor == "" {
		ver.Editor = "code"
	}
	return nil
}
