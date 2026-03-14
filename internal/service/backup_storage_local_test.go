package service

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStorage_Upload_ReturnsNil(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	err := s.Upload(context.Background(), "any-source.tar.gz", "any-dest.tar.gz")
	if err != nil {
		t.Fatalf("Upload() error = %v, want nil", err)
	}
}

func TestLocalStorage_Download_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)
	ctx := context.Background()

	content := []byte("backup-content-12345")
	filename := "test-backup.tar.gz"
	if err := os.WriteFile(filepath.Join(dir, filename), content, 0644); err != nil {
		t.Fatalf("setup: WriteFile error: %v", err)
	}

	rc, size, err := s.Download(ctx, filename)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	defer rc.Close()

	if size != int64(len(content)) {
		t.Errorf("Download() size = %d, want %d", size, len(content))
	}

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("Download() content = %q, want %q", got, content)
	}
}

func TestLocalStorage_Download_NonExistentFile(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	rc, _, err := s.Download(context.Background(), "does-not-exist.tar.gz")
	if err == nil {
		rc.Close()
		t.Fatal("Download() expected error for non-existent file, got nil")
	}
}

func TestLocalStorage_Delete_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	filename := "to-delete.tar.gz"
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatalf("setup: WriteFile error: %v", err)
	}

	if err := s.Delete(context.Background(), filename); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file %s still exists after Delete()", filename)
	}
}

func TestLocalStorage_Delete_NonExistentFile(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir)

	err := s.Delete(context.Background(), "does-not-exist.tar.gz")
	if err != nil {
		t.Fatalf("Delete() error = %v, want nil for missing file", err)
	}
}
