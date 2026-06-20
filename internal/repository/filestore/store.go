package filestore

import "os"

// Open creates a JSON file-backed store at root.
func Open(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	s := &Store{root: root}
	if err := s.ensureLayout(); err != nil {
		return nil, err
	}
	return s, nil
}
