package service

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// AuditService provides audit logging for entity changes.
// Audit failures are logged but never returned to callers.
type AuditService struct {
	repo repository.AuditLogRepo
}

// NewAuditService creates a new AuditService.
func NewAuditService(repo repository.AuditLogRepo) *AuditService {
	return &AuditService{repo: repo}
}

// Log records an audit event. Errors are logged via slog but never returned,
// so that audit logging does not break main operations.
func (s *AuditService) Log(ctx context.Context, entityType string, entityID int64, action string, oldValues, newValues any) {
	entry := &domain.AuditLogEntry{
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
	}

	if oldValues != nil {
		data, err := json.Marshal(oldValues)
		if err != nil {
			slog.Error("marshaling audit old values", "error", err, "entity_type", entityType, "entity_id", entityID)
		} else {
			entry.OldValues = string(data)
		}
	}

	if newValues != nil {
		data, err := json.Marshal(newValues)
		if err != nil {
			slog.Error("marshaling audit new values", "error", err, "entity_type", entityType, "entity_id", entityID)
		} else {
			entry.NewValues = string(data)
		}
	}

	if err := s.repo.Create(ctx, entry); err != nil {
		slog.Error("creating audit log entry", "error", err, "entity_type", entityType, "entity_id", entityID, "action", action)
	}
}

// ListByEntity returns all audit log entries for a given entity.
func (s *AuditService) ListByEntity(ctx context.Context, entityType string, entityID int64) ([]domain.AuditLogEntry, error) {
	return s.repo.ListByEntity(ctx, entityType, entityID)
}
