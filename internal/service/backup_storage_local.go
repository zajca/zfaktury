package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage stores backup archives on the local filesystem.
type LocalStorage struct {
	destDir string
}

// NewLocalStorage creates a LocalStorage that stores archives in destDir.
func NewLocalStorage(destDir string) *LocalStorage {
	return &LocalStorage{destDir: destDir}
}

// Upload is a no-op for local storage because the archive is already
// created directly in the destination directory.
func (s *LocalStorage) Upload(_ context.Context, _, _ string) error {
	return nil
}

// Download opens the archive file and returns a reader with its size.
func (s *LocalStorage) Download(_ context.Context, filename string) (io.ReadCloser, int64, error) {
	path := filepath.Join(s.destDir, filename)
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("opening local backup file: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, 0, fmt.Errorf("stat local backup file: %w", err)
	}
	return f, info.Size(), nil
}

// Delete removes the archive file. Missing files are silently ignored.
func (s *LocalStorage) Delete(_ context.Context, filename string) error {
	path := filepath.Join(s.destDir, filename)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing local backup file: %w", err)
	}
	return nil
}
