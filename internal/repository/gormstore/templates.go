package gormstore

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/SolaTyolo/herald/internal/domain"
)

func (s *Store) CreateTemplate(ctx context.Context, envID string, tmpl *domain.NotificationTemplate) error {
	now := time.Now()
	tmpl.ID = uuid.NewString()
	tmpl.EnvID = envID
	tmpl.CreatedAt, tmpl.UpdatedAt = now, now
	if tmpl.Generation == "" {
		tmpl.Generation = domain.TemplateGenerationDynamic
	}
	return s.db.WithContext(ctx).Create(&notificationTemplateModel{
		ID: tmpl.ID, EnvID: envID, Name: tmpl.Name, Generation: tmpl.Generation,
		CreatedAt: tmpl.CreatedAt, UpdatedAt: tmpl.UpdatedAt,
	}).Error
}

func (s *Store) GetTemplate(ctx context.Context, envID, id string) (*domain.NotificationTemplate, error) {
	var m notificationTemplateModel
	if err := s.db.WithContext(ctx).Where("id = ? AND env_id = ?", id, envID).First(&m).Error; err != nil {
		return nil, err
	}
	tmpl := m.toDomain()
	versions, err := s.loadTemplateVersions(ctx, tmpl.ID)
	if err != nil {
		return nil, err
	}
	tmpl.Versions = versions
	return tmpl, nil
}

func (s *Store) ListTemplates(ctx context.Context, envID string) ([]domain.NotificationTemplate, error) {
	var rows []notificationTemplateModel
	if err := s.db.WithContext(ctx).Where("env_id = ?", envID).Order("updated_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.NotificationTemplate, 0, len(rows))
	for _, row := range rows {
		tmpl := row.toDomain()
		versions, err := s.loadTemplateVersions(ctx, tmpl.ID)
		if err != nil {
			return nil, err
		}
		tmpl.Versions = versions
		out = append(out, *tmpl)
	}
	return out, nil
}

func (s *Store) UpdateTemplate(ctx context.Context, envID string, tmpl *domain.NotificationTemplate) error {
	tmpl.UpdatedAt = time.Now()
	res := s.db.WithContext(ctx).Model(&notificationTemplateModel{}).
		Where("id = ? AND env_id = ?", tmpl.ID, envID).
		Updates(map[string]any{"name": tmpl.Name, "updated_at": tmpl.UpdatedAt})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Store) DeleteTemplate(ctx context.Context, envID, id string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ? AND env_id = ?", id, envID).Delete(&notificationTemplateModel{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("template_id = ?", id).Delete(&templateVersionModel{}).Error
	})
}

func (s *Store) CreateTemplateVersion(ctx context.Context, envID, templateID string, ver *domain.TemplateVersion) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var parent notificationTemplateModel
		if err := tx.Where("id = ? AND env_id = ?", templateID, envID).First(&parent).Error; err != nil {
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
		active, err := s.shouldActivate(tx, templateID, ver.Active)
		if err != nil {
			return err
		}
		ver.Active = active
		if ver.Active {
			if err := tx.Model(&templateVersionModel{}).Where("template_id = ?", templateID).Update("active", false).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(versionModelFromDomain(ver)).Error; err != nil {
			return err
		}
		return tx.Model(&parent).Update("updated_at", now).Error
	})
}

func (s *Store) UpdateTemplateVersion(ctx context.Context, envID, templateID string, ver *domain.TemplateVersion) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var parent notificationTemplateModel
		if err := tx.Where("id = ? AND env_id = ?", templateID, envID).First(&parent).Error; err != nil {
			return err
		}
		var existing templateVersionModel
		if err := tx.Where("id = ? AND template_id = ?", ver.ID, templateID).First(&existing).Error; err != nil {
			return err
		}
		now := time.Now()
		ver.UpdatedAt = now
		ver.TemplateID = templateID
		ver.CreatedAt = existing.CreatedAt
		if ver.Editor == "" {
			ver.Editor = existing.Editor
		}
		if len(ver.TestData) == 0 {
			ver.TestData = existing.TestData
		}
		if ver.Active {
			if err := tx.Model(&templateVersionModel{}).Where("template_id = ? AND id <> ?", templateID, ver.ID).Update("active", false).Error; err != nil {
				return err
			}
		}
		if err := tx.Save(versionModelFromDomain(ver)).Error; err != nil {
			return err
		}
		return tx.Model(&parent).Update("updated_at", now).Error
	})
}

func (s *Store) DeleteTemplateVersion(ctx context.Context, envID, templateID, versionID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var parent notificationTemplateModel
		if err := tx.Where("id = ? AND env_id = ?", templateID, envID).First(&parent).Error; err != nil {
			return err
		}
		var existing templateVersionModel
		if err := tx.Where("id = ? AND template_id = ?", versionID, templateID).First(&existing).Error; err != nil {
			return err
		}
		if err := tx.Delete(&existing).Error; err != nil {
			return err
		}
		if existing.Active {
			var next templateVersionModel
			err := tx.Where("template_id = ?", templateID).Order("created_at DESC").First(&next).Error
			if err == nil {
				if err := tx.Model(&next).Update("active", true).Error; err != nil {
					return err
				}
			} else if err != gorm.ErrRecordNotFound {
				return err
			}
		}
		return tx.Model(&parent).Update("updated_at", time.Now()).Error
	})
}

func (s *Store) shouldActivate(tx *gorm.DB, templateID string, requested bool) (bool, error) {
	if requested {
		return true, nil
	}
	var count int64
	if err := tx.Model(&templateVersionModel{}).Where("template_id = ? AND active = ?", templateID, true).Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}

func (s *Store) loadTemplateVersions(ctx context.Context, templateID string) ([]domain.TemplateVersion, error) {
	var rows []templateVersionModel
	if err := s.db.WithContext(ctx).Where("template_id = ?", templateID).Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.TemplateVersion, 0, len(rows))
	for _, row := range rows {
		out = append(out, *row.toDomain())
	}
	return out, nil
}
