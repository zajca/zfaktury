package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/config"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/testutil"
)

// mockBackupStorage implements service.BackupStorage for testing.
type mockBackupStorage struct {
	uploadFn   func(ctx context.Context, localPath, filename string) error
	downloadFn func(ctx context.Context, filename string) (io.ReadCloser, int64, error)
	deleteFn   func(ctx context.Context, filename string) error
}

func (m *mockBackupStorage) Upload(ctx context.Context, localPath, filename string) error {
	if m.uploadFn != nil {
		return m.uploadFn(ctx, localPath, filename)
	}
	return nil
}

func (m *mockBackupStorage) Download(ctx context.Context, filename string) (io.ReadCloser, int64, error) {
	if m.downloadFn != nil {
		return m.downloadFn(ctx, filename)
	}
	content := []byte("fake backup data")
	return io.NopCloser(bytes.NewReader(content)), int64(len(content)), nil
}

func (m *mockBackupStorage) Delete(ctx context.Context, filename string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, filename)
	}
	return nil
}

func setupBackupHandler(t *testing.T) (*BackupHandler, *repository.BackupHistoryRepository) {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := repository.NewBackupHistoryRepository(db)
	storage := &mockBackupStorage{}
	cfg := config.BackupConfig{
		RetentionCount: 10,
	}
	svc := service.NewBackupService(repo, db, cfg, t.TempDir(), storage)
	h := NewBackupHandler(svc)
	return h, repo
}

func setupBackupHandlerWithStorage(t *testing.T, storage *mockBackupStorage) (*BackupHandler, *repository.BackupHistoryRepository) {
	t.Helper()
	db := testutil.NewTestDB(t)
	repo := repository.NewBackupHistoryRepository(db)
	cfg := config.BackupConfig{
		RetentionCount: 10,
	}
	svc := service.NewBackupService(repo, db, cfg, t.TempDir(), storage)
	h := NewBackupHandler(svc)
	return h, repo
}

// seedBackupRecord inserts a backup record directly into the database.
func seedBackupRecord(t *testing.T, repo *repository.BackupHistoryRepository, status, trigger string) *domain.BackupRecord {
	t.Helper()
	now := time.Now()
	completedAt := now.Add(5 * time.Second)
	rec := &domain.BackupRecord{
		Filename:           fmt.Sprintf("zfaktury-backup-%s.tar.gz", now.Format("2006-01-02T15-04-05")),
		Status:             status,
		Trigger:            trigger,
		Destination:        "/tmp/backups",
		SizeBytes:          1024,
		FileCount:          5,
		DBMigrationVersion: 42,
		DurationMs:         500,
		CreatedAt:          now,
		CompletedAt:        &completedAt,
	}
	if err := repo.Create(context.Background(), rec); err != nil {
		t.Fatalf("seeding backup record: %v", err)
	}
	return rec
}

func TestBackupHandler_List_Empty(t *testing.T) {
	h, _ := setupBackupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var backups []backupResponse
	if err := json.NewDecoder(w.Body).Decode(&backups); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(backups) != 0 {
		t.Errorf("expected 0 backups, got %d", len(backups))
	}
}

func TestBackupHandler_List_WithRecords(t *testing.T) {
	h, repo := setupBackupHandler(t)

	rec1 := seedBackupRecord(t, repo, domain.BackupStatusCompleted, domain.BackupTriggerManual)
	seedBackupRecord(t, repo, domain.BackupStatusFailed, domain.BackupTriggerScheduled)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var backups []backupResponse
	if err := json.NewDecoder(w.Body).Decode(&backups); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(backups) != 2 {
		t.Fatalf("expected 2 backups, got %d", len(backups))
	}

	// Verify the fields are correctly mapped.
	// Records come back in created_at DESC order; both were seeded at ~same time,
	// so just find rec1 by ID.
	var found bool
	for _, b := range backups {
		if b.ID == rec1.ID {
			found = true
			if b.Status != domain.BackupStatusCompleted {
				t.Errorf("Status = %q, want %q", b.Status, domain.BackupStatusCompleted)
			}
			if b.Trigger != domain.BackupTriggerManual {
				t.Errorf("Trigger = %q, want %q", b.Trigger, domain.BackupTriggerManual)
			}
			if b.SizeBytes != 1024 {
				t.Errorf("SizeBytes = %d, want %d", b.SizeBytes, 1024)
			}
			if b.FileCount != 5 {
				t.Errorf("FileCount = %d, want %d", b.FileCount, 5)
			}
			if b.DBMigrationVersion != 42 {
				t.Errorf("DBMigrationVersion = %d, want %d", b.DBMigrationVersion, 42)
			}
		}
	}
	if !found {
		t.Errorf("expected to find backup with ID %d", rec1.ID)
	}
}

func TestBackupHandler_Create(t *testing.T) {
	h, _ := setupBackupHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp backupResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.Trigger != domain.BackupTriggerManual {
		t.Errorf("Trigger = %q, want %q", resp.Trigger, domain.BackupTriggerManual)
	}
	if !strings.HasPrefix(resp.Filename, "zfaktury-backup-") {
		t.Errorf("Filename = %q, expected to start with 'zfaktury-backup-'", resp.Filename)
	}
}

func TestBackupHandler_GetByID(t *testing.T) {
	h, repo := setupBackupHandler(t)
	rec := seedBackupRecord(t, repo, domain.BackupStatusCompleted, domain.BackupTriggerManual)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%d", rec.ID), nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp backupResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.ID != rec.ID {
		t.Errorf("ID = %d, want %d", resp.ID, rec.ID)
	}
	if resp.Filename != rec.Filename {
		t.Errorf("Filename = %q, want %q", resp.Filename, rec.Filename)
	}
}

func TestBackupHandler_GetByID_NotFound(t *testing.T) {
	h, _ := setupBackupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/99999", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestBackupHandler_GetByID_InvalidID(t *testing.T) {
	h, _ := setupBackupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/abc", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestBackupHandler_Delete(t *testing.T) {
	h, repo := setupBackupHandler(t)
	rec := seedBackupRecord(t, repo, domain.BackupStatusCompleted, domain.BackupTriggerManual)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/%d", rec.ID), nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify it's really gone.
	reqList := httptest.NewRequest(http.MethodGet, "/", nil)
	wList := httptest.NewRecorder()

	r2 := chi.NewRouter()
	r2.Mount("/", h.Routes())
	r2.ServeHTTP(wList, reqList)

	var backups []backupResponse
	if err := json.NewDecoder(wList.Body).Decode(&backups); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	for _, b := range backups {
		if b.ID == rec.ID {
			t.Errorf("backup with ID %d should have been deleted", rec.ID)
		}
	}
}

func TestBackupHandler_Delete_NotFound(t *testing.T) {
	h, _ := setupBackupHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/99999", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestBackupHandler_Delete_InvalidID(t *testing.T) {
	h, _ := setupBackupHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/abc", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestBackupHandler_Download(t *testing.T) {
	content := []byte("fake gzip backup content for testing")
	storage := &mockBackupStorage{
		downloadFn: func(ctx context.Context, filename string) (io.ReadCloser, int64, error) {
			return io.NopCloser(bytes.NewReader(content)), int64(len(content)), nil
		},
	}
	h, repo := setupBackupHandlerWithStorage(t, storage)
	rec := seedBackupRecord(t, repo, domain.BackupStatusCompleted, domain.BackupTriggerManual)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%d/download", rec.ID), nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify Content-Type header.
	ct := w.Header().Get("Content-Type")
	if ct != "application/gzip" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/gzip")
	}

	// Verify Content-Disposition header.
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") {
		t.Errorf("Content-Disposition = %q, expected to contain 'attachment'", cd)
	}
	if !strings.Contains(cd, rec.Filename) {
		t.Errorf("Content-Disposition = %q, expected to contain filename %q", cd, rec.Filename)
	}

	// Verify Content-Length header.
	cl := w.Header().Get("Content-Length")
	expectedCL := fmt.Sprintf("%d", len(content))
	if cl != expectedCL {
		t.Errorf("Content-Length = %q, want %q", cl, expectedCL)
	}

	// Verify response body matches the content.
	body := w.Body.Bytes()
	if !bytes.Equal(body, content) {
		t.Errorf("body = %q, want %q", string(body), string(content))
	}
}

func TestBackupHandler_Download_NotFound(t *testing.T) {
	h, _ := setupBackupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/99999/download", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestBackupHandler_Download_InvalidID(t *testing.T) {
	h, _ := setupBackupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/abc/download", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestBackupHandler_Status_NoBackups(t *testing.T) {
	h, _ := setupBackupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp backupStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.IsRunning {
		t.Error("expected IsRunning to be false")
	}
	if resp.LastBackup != nil {
		t.Errorf("expected LastBackup to be nil, got %+v", resp.LastBackup)
	}
}

func TestBackupHandler_Status_WithLastBackup(t *testing.T) {
	h, repo := setupBackupHandler(t)
	rec := seedBackupRecord(t, repo, domain.BackupStatusCompleted, domain.BackupTriggerManual)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp backupStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.IsRunning {
		t.Error("expected IsRunning to be false")
	}
	if resp.LastBackup == nil {
		t.Fatal("expected LastBackup to be non-nil")
	}
	if resp.LastBackup.ID != rec.ID {
		t.Errorf("LastBackup.ID = %d, want %d", resp.LastBackup.ID, rec.ID)
	}
	if resp.LastBackup.Status != domain.BackupStatusCompleted {
		t.Errorf("LastBackup.Status = %q, want %q", resp.LastBackup.Status, domain.BackupStatusCompleted)
	}
}

func TestBackupHandler_Status_CompletedAtField(t *testing.T) {
	h, repo := setupBackupHandler(t)
	seedBackupRecord(t, repo, domain.BackupStatusCompleted, domain.BackupTriggerManual)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp backupStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.LastBackup == nil {
		t.Fatal("expected LastBackup to be non-nil")
	}
	if resp.LastBackup.CompletedAt == "" {
		t.Error("expected CompletedAt to be non-empty for completed backup")
	}
	if resp.LastBackup.CreatedAt == "" {
		t.Error("expected CreatedAt to be non-empty")
	}
}

func TestBackupHandler_Download_StreamsCorrectly(t *testing.T) {
	// Use a larger content to test streaming behavior.
	content := bytes.Repeat([]byte("ABCDEFGHIJ"), 1000) // 10KB
	storage := &mockBackupStorage{
		downloadFn: func(ctx context.Context, filename string) (io.ReadCloser, int64, error) {
			return io.NopCloser(bytes.NewReader(content)), int64(len(content)), nil
		},
	}
	h, repo := setupBackupHandlerWithStorage(t, storage)
	rec := seedBackupRecord(t, repo, domain.BackupStatusCompleted, domain.BackupTriggerManual)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%d/download", rec.ID), nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if w.Body.Len() != len(content) {
		t.Errorf("body length = %d, want %d", w.Body.Len(), len(content))
	}

	cl := w.Header().Get("Content-Length")
	if cl != fmt.Sprintf("%d", len(content)) {
		t.Errorf("Content-Length = %q, want %q", cl, fmt.Sprintf("%d", len(content)))
	}
}

func TestBackupHandler_List_DestinationMapping(t *testing.T) {
	h, repo := setupBackupHandler(t)

	// Seed a record with s3 destination.
	now := time.Now()
	completedAt := now.Add(5 * time.Second)
	s3Rec := &domain.BackupRecord{
		Filename:    "s3-backup.tar.gz",
		Status:      domain.BackupStatusCompleted,
		Trigger:     domain.BackupTriggerScheduled,
		Destination: "s3",
		SizeBytes:   2048,
		FileCount:   3,
		CreatedAt:   now,
		CompletedAt: &completedAt,
	}
	if err := repo.Create(context.Background(), s3Rec); err != nil {
		t.Fatalf("seeding s3 backup record: %v", err)
	}

	// Seed a record with local destination.
	localRec := &domain.BackupRecord{
		Filename:    "local-backup.tar.gz",
		Status:      domain.BackupStatusCompleted,
		Trigger:     domain.BackupTriggerManual,
		Destination: "/tmp/backups",
		SizeBytes:   1024,
		FileCount:   2,
		CreatedAt:   now.Add(time.Second),
		CompletedAt: &completedAt,
	}
	if err := repo.Create(context.Background(), localRec); err != nil {
		t.Fatalf("seeding local backup record: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", h.Routes())
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var backups []backupResponse
	if err := json.NewDecoder(w.Body).Decode(&backups); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	for _, b := range backups {
		if b.Filename == "s3-backup.tar.gz" {
			if b.Destination != "s3" {
				t.Errorf("s3 backup destination = %q, want %q", b.Destination, "s3")
			}
		}
		if b.Filename == "local-backup.tar.gz" {
			if b.Destination != "local" {
				t.Errorf("local backup destination = %q, want %q", b.Destination, "local")
			}
		}
	}
}
