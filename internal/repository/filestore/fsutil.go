package filestore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ErrNotFound is returned when a JSON record file does not exist.
var ErrNotFound = errors.New("record not found")

type Store struct {
	root string
	mu   sync.RWMutex
}

func (s *Store) Close() error { return nil }

func (s *Store) ensureLayout() error {
	for _, dir := range []string{
		"envs",
	} {
		if err := os.MkdirAll(filepath.Join(s.root, dir), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) envRoot(envID string) string {
	return filepath.Join(s.root, "envs", envID)
}

func (s *Store) collection(envID, name string) string {
	dir := filepath.Join(s.envRoot(envID), name)
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

func safeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "_empty"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

func readJSON(path string, dst any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func deleteFile(path string) error {
	err := os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return nil
	}
	return err
}

func listJSONFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		out = append(out, filepath.Join(dir, e.Name()))
	}
	return out, nil
}

func isNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func (s *Store) listEnvIDs() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(s.root, "envs"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

func paginate[T any](items []T, limit, offset int) []T {
	if offset >= len(items) {
		return nil
	}
	items = items[offset:]
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}
