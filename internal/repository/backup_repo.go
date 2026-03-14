package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// BackupHistoryRepository implements persistence for backup history records.
type BackupHistoryRepository struct {
	db *sql.DB
}

// NewBackupHistoryRepository creates a new BackupHistoryRepository.
func NewBackupHistoryRepository(db *sql.DB) *BackupHistoryRepository {
	return &BackupHistoryRepository{db: db}
}

// scanBackupRecord scans a backup record from a sql.Row or sql.Rows.
func scanBackupRecord(scanner interface {
	Scan(dest ...any) error
}) (*domain.BackupRecord, error) {
	var rec domain.BackupRecord
	var createdAtStr string
	var completedAtStr sql.NullString

	if err := scanner.Scan(
		&rec.ID,
		&rec.Filename,
		&rec.Status,
		&rec.Trigger,
		&rec.Destination,
		&rec.SizeBytes,
		&rec.FileCount,
		&rec.DBMigrationVersion,
		&rec.DurationMs,
		&rec.ErrorMessage,
		&createdAtStr,
		&completedAtStr,
	); err != nil {
		return nil, err
	}

	var err error
	rec.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parsing backup record created_at: %w", err)
	}

	rec.CompletedAt, err = parseDatePtr(time.RFC3339, completedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parsing backup record completed_at: %w", err)
	}

	return &rec, nil
}

// Create inserts a new backup record into the database.
func (r *BackupHistoryRepository) Create(ctx context.Context, record *domain.BackupRecord) error {
	now := time.Now()
	record.CreatedAt = now

	var completedAt *string
	if record.CompletedAt != nil {
		s := record.CompletedAt.Format(time.RFC3339)
		completedAt = &s
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO backup_history (filename, status, trigger, destination, size_bytes, file_count, db_migration_version, duration_ms, error_message, created_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.Filename,
		record.Status,
		record.Trigger,
		record.Destination,
		record.SizeBytes,
		record.FileCount,
		record.DBMigrationVersion,
		record.DurationMs,
		record.ErrorMessage,
		record.CreatedAt.Format(time.RFC3339),
		completedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting backup record: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for backup record: %w", err)
	}
	record.ID = id
	return nil
}

// Update modifies an existing backup record.
func (r *BackupHistoryRepository) Update(ctx context.Context, record *domain.BackupRecord) error {
	var completedAt *string
	if record.CompletedAt != nil {
		s := record.CompletedAt.Format(time.RFC3339)
		completedAt = &s
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE backup_history SET
			filename = ?, status = ?, trigger = ?, destination = ?,
			size_bytes = ?, file_count = ?, db_migration_version = ?,
			duration_ms = ?, error_message = ?, completed_at = ?
		WHERE id = ?`,
		record.Filename,
		record.Status,
		record.Trigger,
		record.Destination,
		record.SizeBytes,
		record.FileCount,
		record.DBMigrationVersion,
		record.DurationMs,
		record.ErrorMessage,
		completedAt,
		record.ID,
	)
	if err != nil {
		return fmt.Errorf("updating backup record %d: %w", record.ID, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for backup record %d: %w", record.ID, err)
	}
	if rows == 0 {
		return fmt.Errorf("backup record %d: %w", record.ID, domain.ErrNotFound)
	}
	return nil
}

// GetByID retrieves a single backup record by ID.
func (r *BackupHistoryRepository) GetByID(ctx context.Context, id int64) (*domain.BackupRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, filename, status, trigger, destination, size_bytes, file_count,
			db_migration_version, duration_ms, error_message, created_at, completed_at
		FROM backup_history
		WHERE id = ?`, id,
	)

	rec, err := scanBackupRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("backup record %d: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("querying backup record %d: %w", id, err)
	}
	return rec, nil
}

// List retrieves all backup records ordered by created_at DESC.
func (r *BackupHistoryRepository) List(ctx context.Context) ([]domain.BackupRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, filename, status, trigger, destination, size_bytes, file_count,
			db_migration_version, duration_ms, error_message, created_at, completed_at
		FROM backup_history
		ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing backup records: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []domain.BackupRecord
	for rows.Next() {
		rec, err := scanBackupRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning backup record row: %w", err)
		}
		records = append(records, *rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating backup record rows: %w", err)
	}
	return records, nil
}

// Delete removes a backup record by ID.
func (r *BackupHistoryRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM backup_history WHERE id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("deleting backup record %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for backup record %d: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("backup record %d: %w", id, domain.ErrNotFound)
	}
	return nil
}
