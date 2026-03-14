package service

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/config"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

// newBackupService creates a BackupService backed by a real test DB and local storage in a temp dir.
func newBackupService(t *testing.T, retentionCount int) *BackupService {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := repository.NewBackupHistoryRepository(db)
	dataDir := t.TempDir()
	storageDir := filepath.Join(dataDir, "backups")
	if err := os.MkdirAll(storageDir, 0o755); err != nil {
		t.Fatalf("creating storage dir: %v", err)
	}
	storage := NewLocalStorage(storageDir)
	cfg := config.BackupConfig{
		Destination:    storageDir,
		RetentionCount: retentionCount,
	}
	return NewBackupService(repo, db, cfg, dataDir, storage)
}

func TestBackupService_CreateBackup(t *testing.T) {
	svc := newBackupService(t, 10)
	ctx := context.Background()

	record, err := svc.CreateBackup(ctx, domain.BackupTriggerManual)
	if err != nil {
		t.Fatalf("CreateBackup() error: %v", err)
	}

	if record.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if record.Status != domain.BackupStatusCompleted {
		t.Errorf("Status = %q, want %q", record.Status, domain.BackupStatusCompleted)
	}
	if record.Trigger != domain.BackupTriggerManual {
		t.Errorf("Trigger = %q, want %q", record.Trigger, domain.BackupTriggerManual)
	}
	if record.FileCount <= 0 {
		t.Errorf("FileCount = %d, want > 0", record.FileCount)
	}
	if record.SizeBytes <= 0 {
		t.Errorf("SizeBytes = %d, want > 0", record.SizeBytes)
	}
	if record.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
	if record.DurationMs < 0 {
		t.Errorf("DurationMs = %d, want >= 0", record.DurationMs)
	}
	if record.DBMigrationVersion <= 0 {
		t.Errorf("DBMigrationVersion = %d, want > 0", record.DBMigrationVersion)
	}
}

func TestBackupService_CreateBackup_Concurrent(t *testing.T) {
	svc := newBackupService(t, 10)
	ctx := context.Background()

	// Manually set running flag to simulate a backup in progress.
	svc.running.Store(true)
	defer svc.running.Store(false)

	_, err := svc.CreateBackup(ctx, domain.BackupTriggerManual)
	if err == nil {
		t.Fatal("expected error when backup is already running")
	}
	if err.Error() != "backup is already running" {
		t.Errorf("error = %q, want %q", err.Error(), "backup is already running")
	}
}

func TestBackupService_CreateBackup_ConcurrentRealRace(t *testing.T) {
	svc := newBackupService(t, 10)
	ctx := context.Background()

	var wg sync.WaitGroup
	results := make(chan error, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.CreateBackup(ctx, domain.BackupTriggerManual)
			results <- err
		}()
	}

	wg.Wait()
	close(results)

	var errs []error
	for err := range results {
		if err != nil {
			errs = append(errs, err)
		}
	}

	// Exactly one of the two should have failed with "already running".
	if len(errs) != 1 {
		t.Errorf("expected exactly 1 error from concurrent calls, got %d", len(errs))
	}
}

func TestBackupService_ListBackups(t *testing.T) {
	svc := newBackupService(t, 10)
	ctx := context.Background()

	// Create two backups with a delay longer than 1 second so RFC3339 timestamps differ.
	_, err := svc.CreateBackup(ctx, domain.BackupTriggerManual)
	if err != nil {
		t.Fatalf("first CreateBackup() error: %v", err)
	}
	time.Sleep(1100 * time.Millisecond)
	rec2, err := svc.CreateBackup(ctx, domain.BackupTriggerScheduled)
	if err != nil {
		t.Fatalf("second CreateBackup() error: %v", err)
	}

	records, err := svc.ListBackups(ctx)
	if err != nil {
		t.Fatalf("ListBackups() error: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 backups, got %d", len(records))
	}

	// Should be sorted by created_at DESC (newest first).
	if records[0].ID != rec2.ID {
		t.Errorf("first record ID = %d, want %d (newest)", records[0].ID, rec2.ID)
	}
}

func TestBackupService_GetBackup(t *testing.T) {
	svc := newBackupService(t, 10)
	ctx := context.Background()

	created, err := svc.CreateBackup(ctx, domain.BackupTriggerCLI)
	if err != nil {
		t.Fatalf("CreateBackup() error: %v", err)
	}

	got, err := svc.GetBackup(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetBackup() error: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID = %d, want %d", got.ID, created.ID)
	}
	if got.Filename != created.Filename {
		t.Errorf("Filename = %q, want %q", got.Filename, created.Filename)
	}
	if got.Status != domain.BackupStatusCompleted {
		t.Errorf("Status = %q, want %q", got.Status, domain.BackupStatusCompleted)
	}
	if got.Trigger != domain.BackupTriggerCLI {
		t.Errorf("Trigger = %q, want %q", got.Trigger, domain.BackupTriggerCLI)
	}
	if got.SizeBytes != created.SizeBytes {
		t.Errorf("SizeBytes = %d, want %d", got.SizeBytes, created.SizeBytes)
	}
	if got.FileCount != created.FileCount {
		t.Errorf("FileCount = %d, want %d", got.FileCount, created.FileCount)
	}
}

func TestBackupService_GetBackup_NotFound(t *testing.T) {
	svc := newBackupService(t, 10)
	ctx := context.Background()

	_, err := svc.GetBackup(ctx, 99999)
	if err == nil {
		t.Fatal("expected error for non-existent backup")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestBackupService_DeleteBackup(t *testing.T) {
	svc := newBackupService(t, 10)
	ctx := context.Background()

	created, err := svc.CreateBackup(ctx, domain.BackupTriggerManual)
	if err != nil {
		t.Fatalf("CreateBackup() error: %v", err)
	}

	// Verify the archive file exists on disk.
	archivePath := filepath.Join(svc.cfg.Destination, created.Filename)
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("archive file should exist before delete: %v", err)
	}

	// Delete it.
	if err := svc.DeleteBackup(ctx, created.ID); err != nil {
		t.Fatalf("DeleteBackup() error: %v", err)
	}

	// Record should be gone.
	_, err = svc.GetBackup(ctx, created.ID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("after delete, GetBackup() error = %v, want ErrNotFound", err)
	}

	// File should be gone.
	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		t.Errorf("archive file should be deleted, stat error = %v", err)
	}
}

func TestBackupService_GetBackupReader(t *testing.T) {
	svc := newBackupService(t, 10)
	ctx := context.Background()

	created, err := svc.CreateBackup(ctx, domain.BackupTriggerManual)
	if err != nil {
		t.Fatalf("CreateBackup() error: %v", err)
	}

	reader, size, filename, err := svc.GetBackupReader(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetBackupReader() error: %v", err)
	}
	defer reader.Close()

	if filename != created.Filename {
		t.Errorf("filename = %q, want %q", filename, created.Filename)
	}
	if size <= 0 {
		t.Errorf("size = %d, want > 0", size)
	}
	if size != created.SizeBytes {
		t.Errorf("size = %d, want %d (matching record)", size, created.SizeBytes)
	}

	// Read all data and verify it's valid gzip.
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}
	if int64(len(data)) != size {
		t.Errorf("read %d bytes, expected %d", len(data), size)
	}

	// Verify the content is valid gzip/tar by extracting it.
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("gzip.NewReader() error: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	var fileNames []string
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar.Next() error: %v", err)
		}
		fileNames = append(fileNames, header.Name)
	}

	if len(fileNames) == 0 {
		t.Error("expected at least one file in the archive")
	}

	// Should contain database.db and backup-meta.json at minimum.
	hasDB := false
	hasMeta := false
	for _, name := range fileNames {
		if name == "database.db" {
			hasDB = true
		}
		if name == "backup-meta.json" {
			hasMeta = true
		}
	}
	if !hasDB {
		t.Error("archive missing database.db")
	}
	if !hasMeta {
		t.Error("archive missing backup-meta.json")
	}
}

func TestBackupService_RetentionPolicy(t *testing.T) {
	svc := newBackupService(t, 2)
	ctx := context.Background()

	// Create 3 backups; retention is 2, so the oldest should be deleted.
	for i := 0; i < 3; i++ {
		time.Sleep(10 * time.Millisecond) // Ensure distinct timestamps.
		_, err := svc.CreateBackup(ctx, domain.BackupTriggerManual)
		if err != nil {
			t.Fatalf("CreateBackup() #%d error: %v", i+1, err)
		}
	}

	records, err := svc.ListBackups(ctx)
	if err != nil {
		t.Fatalf("ListBackups() error: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("expected 2 backups after retention, got %d", len(records))
	}
}

func TestBackupService_DirExists(t *testing.T) {
	dir := t.TempDir()

	if !dirExists(dir) {
		t.Error("dirExists() = false for existing directory")
	}

	if dirExists(filepath.Join(dir, "nonexistent")) {
		t.Error("dirExists() = true for non-existent path")
	}

	// Create a file (not a directory) and verify it returns false.
	filePath := filepath.Join(dir, "afile.txt")
	if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	if dirExists(filePath) {
		t.Error("dirExists() = true for a regular file")
	}
}

func TestBackupService_AddFileToTar(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "testfile.txt")
	content := []byte("hello tar world")
	if err := os.WriteFile(srcPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	if err := addFileToTar(tw, srcPath, "archived/testfile.txt"); err != nil {
		t.Fatalf("addFileToTar() error: %v", err)
	}
	tw.Close()

	// Read back from the tar.
	tr := tar.NewReader(&buf)
	header, err := tr.Next()
	if err != nil {
		t.Fatalf("tar.Next() error: %v", err)
	}
	if header.Name != "archived/testfile.txt" {
		t.Errorf("header.Name = %q, want %q", header.Name, "archived/testfile.txt")
	}
	if header.Size != int64(len(content)) {
		t.Errorf("header.Size = %d, want %d", header.Size, len(content))
	}

	got, err := io.ReadAll(tr)
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("content = %q, want %q", got, content)
	}
}

func TestBackupService_AddBytesToTar(t *testing.T) {
	data := []byte(`{"key": "value"}`)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	if err := addBytesToTar(tw, data, "meta.json"); err != nil {
		t.Fatalf("addBytesToTar() error: %v", err)
	}
	tw.Close()

	tr := tar.NewReader(&buf)
	header, err := tr.Next()
	if err != nil {
		t.Fatalf("tar.Next() error: %v", err)
	}
	if header.Name != "meta.json" {
		t.Errorf("header.Name = %q, want %q", header.Name, "meta.json")
	}
	if header.Size != int64(len(data)) {
		t.Errorf("header.Size = %d, want %d", header.Size, len(data))
	}

	got, err := io.ReadAll(tr)
	if err != nil {
		t.Fatalf("ReadAll() error: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("content = %q, want %q", got, data)
	}
}

func TestBackupService_AddDirToTar(t *testing.T) {
	dir := t.TempDir()

	// Create a subdirectory with files.
	subdir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(subdir, name), []byte("content-"+name), 0o644); err != nil {
			t.Fatalf("WriteFile() error: %v", err)
		}
	}

	// Create a nested subdirectory with a file.
	nested := filepath.Join(subdir, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, "c.txt"), []byte("nested-c"), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	count, err := addDirToTar(tw, subdir, "documents")
	if err != nil {
		t.Fatalf("addDirToTar() error: %v", err)
	}
	tw.Close()

	if count != 3 {
		t.Errorf("addDirToTar() count = %d, want 3", count)
	}

	// Verify all entries in the tar.
	tr := tar.NewReader(&buf)
	var names []string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar.Next() error: %v", err)
		}
		names = append(names, header.Name)
	}

	expected := []string{"documents/a.txt", "documents/b.txt", "documents/nested/c.txt"}
	if len(names) != len(expected) {
		t.Fatalf("tar entries = %v, want %v", names, expected)
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("entry[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestBackupService_CreateBackup_WithDocuments(t *testing.T) {
	svc := newBackupService(t, 10)
	ctx := context.Background()

	// Create documents directory with a file in the data dir.
	docsDir := filepath.Join(svc.dataDir, "documents")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "invoice.pdf"), []byte("fake-pdf"), 0o644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	record, err := svc.CreateBackup(ctx, domain.BackupTriggerManual)
	if err != nil {
		t.Fatalf("CreateBackup() error: %v", err)
	}

	// database.db + invoice.pdf + backup-meta.json = 3
	if record.FileCount < 3 {
		t.Errorf("FileCount = %d, want >= 3 (db + document + meta)", record.FileCount)
	}
}
