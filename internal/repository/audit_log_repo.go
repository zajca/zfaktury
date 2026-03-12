package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// AuditLogRepository implements persistence for audit log entries.
type AuditLogRepository struct {
	db *sql.DB
}

// NewAuditLogRepository creates a new AuditLogRepository.
func NewAuditLogRepository(db *sql.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create inserts a new audit log entry into the database.
func (r *AuditLogRepository) Create(ctx context.Context, entry *domain.AuditLogEntry) error {
	now := time.Now()
	entry.CreatedAt = now
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO audit_log (entity_type, entity_id, action, old_values, new_values, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		entry.EntityType, entry.EntityID, entry.Action, entry.OldValues, entry.NewValues,
		entry.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting audit log entry: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for audit log: %w", err)
	}
	entry.ID = id
	return nil
}

// ListByEntity returns all audit log entries for a given entity, ordered by created_at DESC.
func (r *AuditLogRepository) ListByEntity(ctx context.Context, entityType string, entityID int64) ([]domain.AuditLogEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, entity_type, entity_id, action, old_values, new_values, created_at
		FROM audit_log
		WHERE entity_type = ? AND entity_id = ?
		ORDER BY id DESC`,
		entityType, entityID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing audit log entries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []domain.AuditLogEntry
	for rows.Next() {
		entry, err := scanAuditLogRow(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating audit log rows: %w", err)
	}
	return entries, nil
}

// scanAuditLogRow extracts an AuditLogEntry from a row scanner.
func scanAuditLogRow(scanner interface{ Scan(dest ...any) error }) (domain.AuditLogEntry, error) {
	var entry domain.AuditLogEntry
	var oldValues, newValues sql.NullString
	var createdAtStr string

	if err := scanner.Scan(
		&entry.ID, &entry.EntityType, &entry.EntityID, &entry.Action,
		&oldValues, &newValues, &createdAtStr,
	); err != nil {
		return domain.AuditLogEntry{}, fmt.Errorf("scanning audit log row: %w", err)
	}

	if oldValues.Valid {
		entry.OldValues = oldValues.String
	}
	if newValues.Valid {
		entry.NewValues = newValues.String
	}

	createdAt, err := parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return domain.AuditLogEntry{}, fmt.Errorf("parsing audit log created_at: %w", err)
	}
	entry.CreatedAt = createdAt

	return entry, nil
}
