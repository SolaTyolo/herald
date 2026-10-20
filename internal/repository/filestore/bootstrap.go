package filestore

import (
	"context"
	"os"
)

func (s *Store) Ping(ctx context.Context) error {
	_, err := os.Stat(s.root)
	return err
}

func (s *Store) RunMigrations(ctx context.Context) error {
	return s.ensureLayout()
}
