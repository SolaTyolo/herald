package filestore

import (
	"context"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/SolaTyolo/herald/internal/domain"
)

type integrationFile struct {
	ID                   string             `json:"id"`
	EnvID                string             `json:"envId"`
	Channel              domain.ChannelType `json:"channel"`
	ProviderID           string             `json:"providerId"`
	Name                 string             `json:"name"`
	CredentialsEncrypted []byte             `json:"credentialsEncrypted"`
	IsPrimary            bool               `json:"primary"`
	Active               bool               `json:"active"`
	CreatedAt            time.Time          `json:"createdAt"`
	UpdatedAt            time.Time          `json:"updatedAt"`
}

func integrationFromDomain(i *domain.Integration) integrationFile {
	return integrationFile{
		ID: i.ID, EnvID: i.EnvID, Channel: i.Channel, ProviderID: i.ProviderID, Name: i.Name,
		CredentialsEncrypted: i.CredentialsEncrypted, IsPrimary: i.IsPrimary, Active: i.Active,
		CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt,
	}
}

func (f integrationFile) toDomain() domain.Integration {
	return domain.Integration{
		ID: f.ID, EnvID: f.EnvID, Channel: f.Channel, ProviderID: f.ProviderID, Name: f.Name,
		CredentialsEncrypted: f.CredentialsEncrypted, IsPrimary: f.IsPrimary, Active: f.Active,
		CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt,
	}
}

func (s *Store) integrationsDir(envID string) string {
	return s.collection(envID, "integrations")
}

func (s *Store) integrationPath(envID, id string) string {
	return filepath.Join(s.integrationsDir(envID), id+".json")
}

func (s *Store) CreateIntegration(ctx context.Context, integration *domain.Integration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	integration.ID = uuid.NewString()
	now := time.Now()
	integration.CreatedAt, integration.UpdatedAt = now, now
	return writeJSON(s.integrationPath(integration.EnvID, integration.ID), integrationFromDomain(integration))
}

func (s *Store) ListIntegrations(ctx context.Context, envID string) ([]domain.Integration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.listIntegrationsLocked(envID, nil)
}

func (s *Store) DeleteIntegration(ctx context.Context, envID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return deleteFile(s.integrationPath(envID, id))
}

func (s *Store) ListActiveIntegrations(ctx context.Context, envID string, channel domain.ChannelType, primaryOnly bool) ([]domain.Integration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.listIntegrationsLocked(envID, func(i *integrationFile) bool {
		if i.Channel != channel || !i.Active {
			return false
		}
		if primaryOnly && !i.IsPrimary {
			return false
		}
		return true
	})
}

func (s *Store) listIntegrationsLocked(envID string, match func(*integrationFile) bool) ([]domain.Integration, error) {
	files, err := listJSONFiles(s.integrationsDir(envID))
	if err != nil {
		return nil, err
	}
	out := make([]domain.Integration, 0, len(files))
	for _, path := range files {
		var f integrationFile
		if err := readJSON(path, &f); err != nil {
			return nil, err
		}
		if match != nil && !match(&f) {
			continue
		}
		out = append(out, f.toDomain())
	}
	return out, nil
}
