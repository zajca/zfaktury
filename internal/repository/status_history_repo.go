package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// StatusHistoryRepository handles persistence of invoice status change records.
type StatusHistoryRepository struct {
	db *sql.DB
}

// NewStatusHistoryRepository creates a new StatusHistoryRepository.
func NewStatusHistoryRepository(db *sql.DB) *StatusHistoryRepository {
	return &StatusHistoryRepository{db: db}
}

// Create inserts a new status change record.
func (r *StatusHistoryRepository) Create(ctx context.Context, change *domain.InvoiceStatusChange) error {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO invoice_status_history (invoice_id, old_status, new_status, changed_at, note)
		VALUES (?, ?, ?, ?, ?)`,
		change.InvoiceID, change.OldStatus, change.NewStatus,
		change.ChangedAt.Format(time.RFC3339), change.Note,
	)
	if err != nil {
		return fmt.Errorf("inserting status history for invoice %d: %w", change.InvoiceID, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for status history: %w", err)
	}
	change.ID = id
	return nil
}

// ListByInvoiceID returns all status changes for the given invoice, ordered by changed_at ASC.
func (r *StatusHistoryRepository) ListByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.InvoiceStatusChange, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, invoice_id, old_status, new_status, changed_at, note
		FROM invoice_status_history
		WHERE invoice_id = ?
		ORDER BY changed_at ASC, id ASC`, invoiceID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing status history for invoice %d: %w", invoiceID, err)
	}
	defer func() { _ = rows.Close() }()

	var changes []domain.InvoiceStatusChange
	for rows.Next() {
		var c domain.InvoiceStatusChange
		var changedAtStr string
		if err := rows.Scan(&c.ID, &c.InvoiceID, &c.OldStatus, &c.NewStatus, &changedAtStr, &c.Note); err != nil {
			return nil, fmt.Errorf("scanning status history row: %w", err)
		}
		c.ChangedAt, err = parseDate(time.RFC3339, changedAtStr)
		if err != nil {
			return nil, fmt.Errorf("scanning status history row: %w", err)
		}
		changes = append(changes, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating status history rows: %w", err)
	}
	return changes, nil
}
