package repository

import (
	"context"
	"fmt"

	"github.com/SolaTyolo/herald/internal/config"
	"github.com/SolaTyolo/herald/internal/repository/filestore"
	"github.com/SolaTyolo/herald/internal/repository/gormstore"
)

func Open(ctx context.Context, cfg *config.Config) (Store, error) {
	switch cfg.StoreType {
	case config.StoreTypeFile:
		return filestore.Open(cfg.StoreFilePath)
	case config.StoreTypeDB:
		return gormstore.Open(ctx, cfg.DBDriver, cfg.DatabaseURL)
	default:
		return nil, fmt.Errorf("unsupported store type: %s", cfg.StoreType)
	}
}
