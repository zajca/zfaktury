package service

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/zajca/zfaktury/internal/config"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/version"
)

// BackupStatus holds the current state of the backup system.
type BackupStatus struct {
	IsRunning     bool
	LastBackup    *domain.BackupRecord
	NextScheduled string
}

// backupMeta is written as backup-meta.json inside each archive.
type backupMeta struct {
	AppVersion         string `json:"app_version"`
	DBMigrationVersion int64  `json:"db_migration_version"`
	CreatedAt          string `json:"created_at"`
	FileCount          int    `json:"file_count"`
	DBSizeBytes        int64  `json:"db_size_bytes"`
}

// BackupService handles creating, listing, and managing backup archives.
type BackupService struct {
	repo    repository.BackupHistoryRepo
	db      *sql.DB
	cfg     config.BackupConfig
	dataDir string
	storage BackupStorage
	isS3    bool
	running atomic.Bool
}

// NewBackupService creates a new BackupService.
func NewBackupService(repo repository.BackupHistoryRepo, db *sql.DB, cfg config.BackupConfig, dataDir string, storage BackupStorage) *BackupService {
	_, isS3 := storage.(*S3Storage)
	return &BackupService{
		repo:    repo,
		db:      db,
		cfg:     cfg,
		dataDir: dataDir,
		storage: storage,
		isS3:    isS3,
	}
}

// CreateBackup performs a full backup: VACUUM INTO temp file, tar.gz archive
// containing the database, documents, tax-documents, and metadata, then
// applies retention policy.
func (s *BackupService) CreateBackup(ctx context.Context, trigger string) (*domain.BackupRecord, error) {
	if !s.running.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("backup is already running")
	}
	defer s.running.Store(false)

	startTime := time.Now()

	// Determine working directory for archive creation.
	// For S3 storage, use a temp dir; for local, use the configured destination.
	var destDir string
	if s.isS3 {
		destDir = os.TempDir()
	} else {
		destDir = s.cfg.Destination
		if destDir == "" {
			destDir = filepath.Join(s.dataDir, "backups")
		}
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating backup destination directory: %w", err)
	}

	// Generate archive filename.
	ts := startTime.Format("2006-01-02T15-04-05")
	filename := fmt.Sprintf("zfaktury-backup-%s.tar.gz", ts)

	// Determine storage destination label for the record.
	destination := destDir
	if s.isS3 {
		destination = "s3"
	}

	// Create the initial record in DB.
	record := &domain.BackupRecord{
		Filename:    filename,
		Status:      domain.BackupStatusRunning,
		Trigger:     trigger,
		Destination: destination,
		CreatedAt:   startTime,
	}
	if err := s.repo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("creating backup record: %w", err)
	}

	// Run the actual backup; on failure update the record.
	if err := s.performBackup(ctx, record, destDir, filename); err != nil {
		now := time.Now()
		record.Status = domain.BackupStatusFailed
		record.ErrorMessage = err.Error()
		record.CompletedAt = &now
		record.DurationMs = now.Sub(startTime).Milliseconds()
		if updateErr := s.repo.Update(ctx, record); updateErr != nil {
			slog.Error("failed to update backup record after failure", "error", updateErr)
		}
		return record, fmt.Errorf("performing backup: %w", err)
	}

	// Mark completed.
	now := time.Now()
	record.Status = domain.BackupStatusCompleted
	record.CompletedAt = &now
	record.DurationMs = now.Sub(startTime).Milliseconds()
	if err := s.repo.Update(ctx, record); err != nil {
		return nil, fmt.Errorf("updating backup record: %w", err)
	}

	// Apply retention policy.
	if err := s.applyRetention(ctx); err != nil {
		slog.Error("failed to apply backup retention", "error", err)
	}

	slog.Info("backup completed",
		"filename", filename,
		"size_bytes", record.SizeBytes,
		"file_count", record.FileCount,
		"duration_ms", record.DurationMs,
	)

	return record, nil
}

// performBackup executes the core backup logic: vacuum, tar, archive.
func (s *BackupService) performBackup(ctx context.Context, record *domain.BackupRecord, destDir, filename string) error {
	// Get DB migration version.
	migrationVersion, err := s.getDBMigrationVersion(ctx)
	if err != nil {
		slog.Warn("failed to get DB migration version", "error", err)
	}
	record.DBMigrationVersion = migrationVersion

	// VACUUM INTO a temp database file.
	tempDBPath := filepath.Join(destDir, "backup-temp.db")
	defer os.Remove(tempDBPath)

	escapedPath := strings.ReplaceAll(tempDBPath, "'", "''")
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf("VACUUM INTO '%s'", escapedPath)); err != nil {
		return fmt.Errorf("vacuum into temp db: %w", err)
	}

	// Get the vacuumed DB size.
	dbInfo, err := os.Stat(tempDBPath)
	if err != nil {
		return fmt.Errorf("stat temp db: %w", err)
	}
	dbSizeBytes := dbInfo.Size()

	// Create tar.gz archive.
	archivePath := filepath.Join(destDir, filename)
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("creating archive file: %w", err)
	}
	defer archiveFile.Close()

	gzWriter := gzip.NewWriter(archiveFile)
	tarWriter := tar.NewWriter(gzWriter)

	fileCount := 0

	// Add database.db to archive.
	if err := addFileToTar(tarWriter, tempDBPath, "database.db"); err != nil {
		return fmt.Errorf("adding database to archive: %w", err)
	}
	fileCount++

	// Add documents/ directory if it exists.
	documentsDir := filepath.Join(s.dataDir, "documents")
	if dirExists(documentsDir) {
		count, err := addDirToTar(tarWriter, documentsDir, "documents")
		if err != nil {
			return fmt.Errorf("adding documents to archive: %w", err)
		}
		fileCount += count
	}

	// Add tax-documents/ directory if it exists.
	taxDocsDir := filepath.Join(s.dataDir, "tax-documents")
	if dirExists(taxDocsDir) {
		count, err := addDirToTar(tarWriter, taxDocsDir, "tax-documents")
		if err != nil {
			return fmt.Errorf("adding tax-documents to archive: %w", err)
		}
		fileCount += count
	}

	// Create and add backup-meta.json.
	meta := backupMeta{
		AppVersion:         version.Version,
		DBMigrationVersion: migrationVersion,
		CreatedAt:          record.CreatedAt.Format(time.RFC3339),
		FileCount:          fileCount,
		DBSizeBytes:        dbSizeBytes,
	}
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling backup meta: %w", err)
	}
	if err := addBytesToTar(tarWriter, metaBytes, "backup-meta.json"); err != nil {
		return fmt.Errorf("adding meta to archive: %w", err)
	}
	fileCount++

	// Close writers to flush all data before getting file size.
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("closing tar writer: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return fmt.Errorf("closing gzip writer: %w", err)
	}
	if err := archiveFile.Close(); err != nil {
		return fmt.Errorf("closing archive file: %w", err)
	}

	// Get archive size.
	archiveInfo, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("stat archive: %w", err)
	}

	record.SizeBytes = archiveInfo.Size()
	record.FileCount = fileCount

	// Upload to storage backend.
	if err := s.storage.Upload(ctx, archivePath, filename); err != nil {
		return fmt.Errorf("uploading backup archive: %w", err)
	}

	// For S3 storage, remove the local temp archive after successful upload.
	if s.isS3 {
		os.Remove(archivePath)
	}

	return nil
}

// ListBackups returns all backup records ordered by created_at DESC.
func (s *BackupService) ListBackups(ctx context.Context) ([]domain.BackupRecord, error) {
	records, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing backups: %w", err)
	}
	return records, nil
}

// GetBackup returns a single backup record by ID.
func (s *BackupService) GetBackup(ctx context.Context, id int64) (*domain.BackupRecord, error) {
	record, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching backup: %w", err)
	}
	return record, nil
}

// DeleteBackup removes the archive from storage and deletes the DB record.
func (s *BackupService) DeleteBackup(ctx context.Context, id int64) error {
	record, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching backup for deletion: %w", err)
	}

	// Remove the archive from storage (best-effort).
	if err := s.storage.Delete(ctx, record.Filename); err != nil {
		slog.Warn("failed to remove backup archive", "filename", record.Filename, "error", err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting backup record: %w", err)
	}

	slog.Info("backup deleted", "id", id, "filename", record.Filename)
	return nil
}

// GetBackupReader returns a reader for the backup archive, its size, and filename.
func (s *BackupService) GetBackupReader(ctx context.Context, id int64) (io.ReadCloser, int64, string, error) {
	record, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, 0, "", fmt.Errorf("fetching backup for download: %w", err)
	}

	reader, size, err := s.storage.Download(ctx, record.Filename)
	if err != nil {
		return nil, 0, "", fmt.Errorf("downloading backup archive: %w", err)
	}

	return reader, size, record.Filename, nil
}

// GetStatus returns the current backup system status.
func (s *BackupService) GetStatus(ctx context.Context) (*BackupStatus, error) {
	status := &BackupStatus{
		IsRunning: s.running.Load(),
	}

	// Get last backup.
	records, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing backups for status: %w", err)
	}
	if len(records) > 0 {
		status.LastBackup = &records[0]
	}

	// NextScheduled is set from config schedule if present.
	if s.cfg.Schedule != "" {
		status.NextScheduled = s.cfg.Schedule
	}

	return status, nil
}

// IsRunning returns true if a backup is currently in progress.
func (s *BackupService) IsRunning() bool {
	return s.running.Load()
}

// getDBMigrationVersion queries goose for the latest applied migration version.
func (s *BackupService) getDBMigrationVersion(ctx context.Context) (int64, error) {
	var v int64
	err := s.db.QueryRowContext(ctx, "SELECT MAX(version_id) FROM goose_db_version WHERE is_applied = 1").Scan(&v)
	if err != nil {
		return 0, fmt.Errorf("querying migration version: %w", err)
	}
	return v, nil
}

// applyRetention deletes the oldest backups beyond the configured retention count.
func (s *BackupService) applyRetention(ctx context.Context) error {
	if s.cfg.RetentionCount <= 0 {
		return nil
	}

	records, err := s.repo.List(ctx)
	if err != nil {
		return fmt.Errorf("listing backups for retention: %w", err)
	}

	// Only consider completed backups for retention.
	var completed []domain.BackupRecord
	for _, r := range records {
		if r.Status == domain.BackupStatusCompleted {
			completed = append(completed, r)
		}
	}

	if len(completed) <= s.cfg.RetentionCount {
		return nil
	}

	// Records are ordered by created_at DESC from the repo. Delete the oldest ones.
	toDelete := completed[s.cfg.RetentionCount:]
	for _, r := range toDelete {
		slog.Info("retention: deleting old backup", "id", r.ID, "filename", r.Filename)
		if err := s.DeleteBackup(ctx, r.ID); err != nil {
			slog.Error("retention: failed to delete backup", "id", r.ID, "error", err)
		}
	}

	return nil
}

// addFileToTar adds a single file to the tar archive under the given archive name.
func addFileToTar(tw *tar.Writer, filePath, archiveName string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening %s: %w", filePath, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat %s: %w", filePath, err)
	}

	header := &tar.Header{
		Name:    archiveName,
		Size:    info.Size(),
		Mode:    int64(info.Mode()),
		ModTime: info.ModTime(),
	}
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("writing tar header for %s: %w", archiveName, err)
	}
	if _, err := io.Copy(tw, f); err != nil {
		return fmt.Errorf("writing tar content for %s: %w", archiveName, err)
	}
	return nil
}

// addBytesToTar adds in-memory bytes to the tar archive under the given archive name.
func addBytesToTar(tw *tar.Writer, data []byte, archiveName string) error {
	header := &tar.Header{
		Name:    archiveName,
		Size:    int64(len(data)),
		Mode:    0o644,
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("writing tar header for %s: %w", archiveName, err)
	}
	if _, err := tw.Write(data); err != nil {
		return fmt.Errorf("writing tar content for %s: %w", archiveName, err)
	}
	return nil
}

// addDirToTar recursively adds all files in a directory to the tar archive
// under the given archive prefix. Returns the number of files added.
func addDirToTar(tw *tar.Writer, dirPath, archivePrefix string) (int, error) {
	count := 0

	// Collect entries first and sort for deterministic output.
	var entries []string
	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		entries = append(entries, path)
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("walking directory %s: %w", dirPath, err)
	}

	sort.Strings(entries)

	for _, path := range entries {
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return 0, fmt.Errorf("computing relative path for %s: %w", path, err)
		}
		archiveName := filepath.Join(archivePrefix, relPath)
		// Normalize to forward slashes for tar.
		archiveName = filepath.ToSlash(archiveName)

		if err := addFileToTar(tw, path, archiveName); err != nil {
			return 0, err
		}
		count++
	}

	return count, nil
}

// dirExists checks whether a directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
