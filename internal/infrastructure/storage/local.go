package storage

import (
	"context"
	"os"
	"path/filepath"
)

type LocalStore struct {
	root string
}

func NewLocalStore(root string) LocalStore {
	return LocalStore{root: root}
}

func (s LocalStore) Ping(ctx context.Context) error {
	return os.MkdirAll(s.root, 0o755)
}

func (s LocalStore) Put(ctx context.Context, key string, data []byte) error {
	path := filepath.Join(s.root, filepath.Clean(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (s LocalStore) Get(ctx context.Context, key string) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.root, filepath.Clean(key)))
}

func (s LocalStore) Delete(ctx context.Context, key string) error {
	return os.Remove(filepath.Join(s.root, filepath.Clean(key)))
}
