package handler

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// BackupHandler handles HTTP requests for backup management.
type BackupHandler struct {
	svc *service.BackupService
}

// NewBackupHandler creates a new BackupHandler.
func NewBackupHandler(svc *service.BackupService) *BackupHandler {
	return &BackupHandler{svc: svc}
}

// Routes registers backup routes on the given router.
func (h *BackupHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/status", h.Status)
	r.Get("/{id}", h.GetByID)
	r.Delete("/{id}", h.Delete)
	r.Get("/{id}/download", h.Download)
	return r
}

// Create handles POST /api/v1/backups.
func (h *BackupHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h.svc.IsRunning() {
		respondError(w, http.StatusConflict, "backup is already running")
		return
	}

	record, err := h.svc.CreateBackup(r.Context(), domain.BackupTriggerManual)
	if err != nil {
		slog.Error("failed to create backup", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to create backup")
		return
	}

	respondJSON(w, http.StatusCreated, backupFromDomain(record))
}

// List handles GET /api/v1/backups.
func (h *BackupHandler) List(w http.ResponseWriter, r *http.Request) {
	records, err := h.svc.ListBackups(r.Context())
	if err != nil {
		slog.Error("failed to list backups", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list backups")
		return
	}

	items := make([]backupResponse, 0, len(records))
	for i := range records {
		items = append(items, backupFromDomain(&records[i]))
	}

	respondJSON(w, http.StatusOK, items)
}

// GetByID handles GET /api/v1/backups/{id}.
func (h *BackupHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid backup ID")
		return
	}

	record, err := h.svc.GetBackup(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "backup not found")
			return
		}
		slog.Error("failed to get backup", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "failed to get backup")
		return
	}

	respondJSON(w, http.StatusOK, backupFromDomain(record))
}

// Delete handles DELETE /api/v1/backups/{id}.
func (h *BackupHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid backup ID")
		return
	}

	if err := h.svc.DeleteBackup(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "backup not found")
			return
		}
		slog.Error("failed to delete backup", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "failed to delete backup")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Download handles GET /api/v1/backups/{id}/download.
func (h *BackupHandler) Download(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid backup ID")
		return
	}

	reader, size, filename, err := h.svc.GetBackupReader(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "backup not found")
			return
		}
		slog.Error("failed to get backup reader", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "failed to get backup file")
		return
	}
	defer func() { _ = reader.Close() }()

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filename))
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	if _, err := io.Copy(w, reader); err != nil {
		slog.Error("failed to stream backup download", "error", err, "id", id)
	}
}

// Status handles GET /api/v1/backups/status.
func (h *BackupHandler) Status(w http.ResponseWriter, r *http.Request) {
	status, err := h.svc.GetStatus(r.Context())
	if err != nil {
		slog.Error("failed to get backup status", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get backup status")
		return
	}

	resp := backupStatusResponse{
		IsRunning:     status.IsRunning,
		NextScheduled: status.NextScheduled,
	}
	if status.LastBackup != nil {
		lb := backupFromDomain(status.LastBackup)
		resp.LastBackup = &lb
	}

	respondJSON(w, http.StatusOK, resp)
}

// --- Backup DTOs ---

// backupResponse is the JSON response for a backup record.
type backupResponse struct {
	ID                 int64  `json:"id"`
	Filename           string `json:"filename"`
	Status             string `json:"status"`
	Trigger            string `json:"trigger"`
	Destination        string `json:"destination"`
	SizeBytes          int64  `json:"size_bytes"`
	FileCount          int    `json:"file_count"`
	DBMigrationVersion int64  `json:"db_migration_version"`
	DurationMs         int64  `json:"duration_ms"`
	ErrorMessage       string `json:"error_message,omitempty"`
	CreatedAt          string `json:"created_at"`
	CompletedAt        string `json:"completed_at,omitempty"`
}

// backupStatusResponse is the JSON response for backup status.
type backupStatusResponse struct {
	IsRunning     bool            `json:"is_running"`
	LastBackup    *backupResponse `json:"last_backup"`
	NextScheduled string          `json:"next_scheduled,omitempty"`
}

// backupFromDomain converts a domain.BackupRecord to a backupResponse.
func backupFromDomain(rec *domain.BackupRecord) backupResponse {
	dest := "local"
	if rec.Destination == "s3" {
		dest = "s3"
	}

	resp := backupResponse{
		ID:                 rec.ID,
		Filename:           rec.Filename,
		Status:             rec.Status,
		Trigger:            rec.Trigger,
		Destination:        dest,
		SizeBytes:          rec.SizeBytes,
		FileCount:          rec.FileCount,
		DBMigrationVersion: rec.DBMigrationVersion,
		DurationMs:         rec.DurationMs,
		ErrorMessage:       rec.ErrorMessage,
		CreatedAt:          rec.CreatedAt.Format(time.RFC3339),
	}
	if rec.CompletedAt != nil {
		resp.CompletedAt = rec.CompletedAt.Format(time.RFC3339)
	}
	return resp
}
