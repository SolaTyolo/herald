package gormstore

import (
	"context"

	"gorm.io/gorm"

	"github.com/SolaTyolo/herald/internal/config"
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
