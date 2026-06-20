package filestore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/SolaTyolo/herald/internal/crypto"
	"github.com/SolaTyolo/herald/internal/domain"
)

type bootstrapFile struct {
	Tenant      domain.Tenant      `json:"tenant"`
	Environment domain.Environment `json:"environment"`
	APIKeys     []apiKeyFile       `json:"apiKeys"`
}

type apiKeyFile struct {
	ID        string    `json:"id"`
	EnvID     string    `json:"envId"`
	Name      string    `json:"name"`
	KeyHash   string    `json:"keyHash"`
	KeyPrefix string    `json:"keyPrefix"`
	CreatedAt time.Time `json:"createdAt"`
}

func (s *Store) bootstrapPath() string {
	return filepath.Join(s.root, "bootstrap.json")
}

func (s *Store) loadBootstrap() (*bootstrapFile, error) {
	var b bootstrapFile
	if err := readJSON(s.bootstrapPath(), &b); err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Store) saveBootstrap(b *bootstrapFile) error {
	return writeJSON(s.bootstrapPath(), b)
}

func (s *Store) Ping(ctx context.Context) error {
	_, err := os.Stat(s.root)
	return err
}

func (s *Store) RunMigrations(ctx context.Context) error {
	return s.ensureLayout()
}

func (s *Store) EnsureDefaultTenant(ctx context.Context) (*domain.Environment, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.bootstrapPath()); err == nil {
		b, err := s.loadBootstrap()
		if err != nil {
			return nil, "", err
		}
		return &b.Environment, "", nil
	}

	newTenantID := uuid.NewString()
	envID := uuid.NewString()
	keyID := uuid.NewString()
	apiKey := "hr_" + uuid.NewString()
	hash, err := crypto.HashAPIKey(apiKey)
	if err != nil {
		return nil, "", err
	}
	prefix := apiKey[:10]
	now := time.Now()

	b := &bootstrapFile{
		Tenant: domain.Tenant{ID: newTenantID, Name: "Default", CreatedAt: now},
		Environment: domain.Environment{
			ID: envID, TenantID: newTenantID, Name: "Development", Slug: "dev", CreatedAt: now,
		},
		APIKeys: []apiKeyFile{{
			ID: keyID, EnvID: envID, Name: "default", KeyHash: hash, KeyPrefix: prefix, CreatedAt: now,
		}},
	}
	if err := s.saveBootstrap(b); err != nil {
		return nil, "", err
	}
	if err := os.MkdirAll(s.envRoot(envID), 0o755); err != nil {
		return nil, "", err
	}
	return &b.Environment, apiKey, nil
}

func (s *Store) ValidateAPIKey(ctx context.Context, key string) (*domain.Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	b, err := s.loadBootstrap()
	if err != nil {
		return nil, fmt.Errorf("invalid api key")
	}
	key = strings.TrimPrefix(strings.TrimPrefix(key, "ApiKey "), "Bearer ")
	prefix := key
	if len(prefix) > 10 {
		prefix = prefix[:10]
	}
	for _, k := range b.APIKeys {
		if k.KeyPrefix != prefix {
			continue
		}
		if crypto.CompareAPIKey(key, k.KeyHash) {
			return &b.Environment, nil
		}
	}
	return nil, fmt.Errorf("invalid api key")
}
