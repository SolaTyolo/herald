package gormstore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/SolaTyolo/herald/internal/config"
	"github.com/SolaTyolo/herald/internal/crypto"
	"github.com/SolaTyolo/herald/internal/domain"
)

type Store struct {
	db *gorm.DB
}

func Open(ctx context.Context, driver config.DBDriver, dsn string) (*Store, error) {
	db, err := openDB(driver, dsn)
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (s *Store) RunMigrations(ctx context.Context) error {
	return s.db.WithContext(ctx).AutoMigrate(allModels()...)
}

func (s *Store) EnsureDefaultTenant(ctx context.Context) (*domain.Environment, string, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&tenantModel{}).Count(&count).Error; err != nil {
		return nil, "", err
	}
	if count > 0 {
		var env environmentModel
		err := s.db.WithContext(ctx).
			Joins("JOIN tenants ON tenants.id = environments.tenant_id").
			Order("environments.created_at ASC").
			First(&env).Error
		if err != nil {
			return nil, "", err
		}
		return env.toDomain(), "", nil
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

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&tenantModel{ID: newTenantID, Name: "Default", CreatedAt: now}).Error; err != nil {
			return err
		}
		env := environmentModel{ID: envID, TenantID: newTenantID, Name: "Development", Slug: "dev", CreatedAt: now}
		if err := tx.Create(&env).Error; err != nil {
			return err
		}
		return tx.Create(&apiKeyModel{
			ID: keyID, EnvID: envID, Name: "default", KeyHash: hash, KeyPrefix: prefix, CreatedAt: now,
		}).Error
	})
	if err != nil {
		return nil, "", err
	}
	return &domain.Environment{ID: envID, TenantID: newTenantID, Name: "Development", Slug: "dev", CreatedAt: now}, apiKey, nil
}

func (s *Store) ValidateAPIKey(ctx context.Context, key string) (*domain.Environment, error) {
	key = strings.TrimPrefix(key, "Bearer ")
	prefix := key
	if len(prefix) > 10 {
		prefix = prefix[:10]
	}
	var keys []apiKeyModel
	if err := s.db.WithContext(ctx).Where("key_prefix = ?", prefix).Find(&keys).Error; err != nil {
		return nil, fmt.Errorf("invalid api key")
	}
	for _, k := range keys {
		if crypto.CompareAPIKey(key, k.KeyHash) {
			var env environmentModel
			if err := s.db.WithContext(ctx).First(&env, "id = ?", k.EnvID).Error; err != nil {
				return nil, fmt.Errorf("invalid api key")
			}
			return env.toDomain(), nil
		}
	}
	return nil, fmt.Errorf("invalid api key")
}
