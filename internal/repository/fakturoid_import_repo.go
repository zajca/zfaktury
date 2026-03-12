package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// FakturoidImportLogRepository implements the FakturoidImportLogRepo interface.
type FakturoidImportLogRepository struct {
	db *sql.DB
}

// NewFakturoidImportLogRepository creates a new repository.
func NewFakturoidImportLogRepository(db *sql.DB) *FakturoidImportLogRepository {
	return &FakturoidImportLogRepository{db: db}
}

// scanImportLog scans a single import log row into a domain struct.
func scanImportLog(scanner interface{ Scan(dest ...any) error }) (domain.FakturoidImportLog, error) {
	var e domain.FakturoidImportLog
	var importedAtStr string
	if err := scanner.Scan(&e.ID, &e.FakturoidEntityType, &e.FakturoidID, &e.LocalEntityType, &e.LocalID, &importedAtStr); err != nil {
		return e, err
	}
	var err error
	e.ImportedAt, err = parseDate(time.RFC3339, importedAtStr)
	if err != nil {
		return e, fmt.Errorf("parsing import log date: %w", err)
	}
	return e, nil
}

// Create inserts a new import log entry.
func (r *FakturoidImportLogRepository) Create(ctx context.Context, entry *domain.FakturoidImportLog) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO fakturoid_import_log (fakturoid_entity_type, fakturoid_id, local_entity_type, local_id)
		 VALUES (?, ?, ?, ?)`,
		entry.FakturoidEntityType, entry.FakturoidID, entry.LocalEntityType, entry.LocalID,
	)
	if err != nil {
		return fmt.Errorf("inserting import log: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting import log id: %w", err)
	}
	entry.ID = id
	return nil
}

// FindByFakturoidID looks up an import log entry by Fakturoid entity type and ID.
func (r *FakturoidImportLogRepository) FindByFakturoidID(ctx context.Context, entityType string, fakturoidID int64) (*domain.FakturoidImportLog, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, fakturoid_entity_type, fakturoid_id, local_entity_type, local_id, imported_at
		 FROM fakturoid_import_log WHERE fakturoid_entity_type = ? AND fakturoid_id = ?`,
		entityType, fakturoidID,
	)
	entry, err := scanImportLog(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // not found, return nil without error
		}
		return nil, fmt.Errorf("querying import log: %w", err)
	}
	return &entry, nil
}

// ListByEntityType lists all import logs for a given entity type.
func (r *FakturoidImportLogRepository) ListByEntityType(ctx context.Context, entityType string) ([]domain.FakturoidImportLog, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, fakturoid_entity_type, fakturoid_id, local_entity_type, local_id, imported_at
		 FROM fakturoid_import_log WHERE fakturoid_entity_type = ? ORDER BY id`,
		entityType,
	)
	if err != nil {
		return nil, fmt.Errorf("listing import logs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []domain.FakturoidImportLog
	for rows.Next() {
		entry, err := scanImportLog(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning import log: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}
