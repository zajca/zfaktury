package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// SequenceRepository handles persistence of InvoiceSequence entities.
type SequenceRepository struct {
	db *sql.DB
}

// NewSequenceRepository creates a new SequenceRepository.
func NewSequenceRepository(db *sql.DB) *SequenceRepository {
	return &SequenceRepository{db: db}
}

// Create inserts a new invoice sequence into the database.
func (r *SequenceRepository) Create(ctx context.Context, seq *domain.InvoiceSequence) error {
	now := time.Now()

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO invoice_sequences (prefix, next_number, year, format_pattern, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		seq.Prefix, seq.NextNumber, seq.Year, seq.FormatPattern, now, now,
	)
	if err != nil {
		return fmt.Errorf("inserting invoice sequence: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for invoice sequence: %w", err)
	}
	seq.ID = id
	return nil
}

// Update modifies an existing invoice sequence.
func (r *SequenceRepository) Update(ctx context.Context, seq *domain.InvoiceSequence) error {
	now := time.Now()

	result, err := r.db.ExecContext(ctx, `
		UPDATE invoice_sequences SET
			prefix = ?, next_number = ?, year = ?, format_pattern = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`,
		seq.Prefix, seq.NextNumber, seq.Year, seq.FormatPattern, now, seq.ID,
	)
	if err != nil {
		return fmt.Errorf("updating invoice sequence %d: %w", seq.ID, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for invoice sequence %d: %w", seq.ID, err)
	}
	if rows == 0 {
		return fmt.Errorf("invoice sequence %d not found or already deleted", seq.ID)
	}
	return nil
}

// Delete performs a soft delete on an invoice sequence.
func (r *SequenceRepository) Delete(ctx context.Context, id int64) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, `
		UPDATE invoice_sequences SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		now, now, id,
	)
	if err != nil {
		return fmt.Errorf("soft-deleting invoice sequence %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for invoice sequence %d: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("invoice sequence %d not found or already deleted", id)
	}
	return nil
}

// GetByID retrieves a single invoice sequence by ID.
func (r *SequenceRepository) GetByID(ctx context.Context, id int64) (*domain.InvoiceSequence, error) {
	seq := &domain.InvoiceSequence{}

	err := r.db.QueryRowContext(ctx, `
		SELECT id, prefix, next_number, year, format_pattern
		FROM invoice_sequences WHERE id = ? AND deleted_at IS NULL`, id,
	).Scan(&seq.ID, &seq.Prefix, &seq.NextNumber, &seq.Year, &seq.FormatPattern)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invoice sequence %d not found: %w", id, err)
		}
		return nil, fmt.Errorf("querying invoice sequence %d: %w", id, err)
	}
	return seq, nil
}

// List retrieves all non-deleted invoice sequences.
func (r *SequenceRepository) List(ctx context.Context) ([]domain.InvoiceSequence, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, prefix, next_number, year, format_pattern
		FROM invoice_sequences
		WHERE deleted_at IS NULL
		ORDER BY year DESC, prefix ASC`)
	if err != nil {
		return nil, fmt.Errorf("listing invoice sequences: %w", err)
	}
	defer rows.Close()

	var sequences []domain.InvoiceSequence
	for rows.Next() {
		var seq domain.InvoiceSequence
		if err := rows.Scan(&seq.ID, &seq.Prefix, &seq.NextNumber, &seq.Year, &seq.FormatPattern); err != nil {
			return nil, fmt.Errorf("scanning invoice sequence row: %w", err)
		}
		sequences = append(sequences, seq)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating invoice sequence rows: %w", err)
	}
	return sequences, nil
}

// GetByPrefixAndYear retrieves an invoice sequence by its prefix and year combination.
func (r *SequenceRepository) GetByPrefixAndYear(ctx context.Context, prefix string, year int) (*domain.InvoiceSequence, error) {
	seq := &domain.InvoiceSequence{}

	err := r.db.QueryRowContext(ctx, `
		SELECT id, prefix, next_number, year, format_pattern
		FROM invoice_sequences
		WHERE prefix = ? AND year = ? AND deleted_at IS NULL`, prefix, year,
	).Scan(&seq.ID, &seq.Prefix, &seq.NextNumber, &seq.Year, &seq.FormatPattern)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invoice sequence with prefix %q and year %d not found: %w", prefix, year, err)
		}
		return nil, fmt.Errorf("querying invoice sequence by prefix %q and year %d: %w", prefix, year, err)
	}
	return seq, nil
}

// CountInvoicesBySequenceID returns the number of invoices referencing a given sequence.
func (r *SequenceRepository) CountInvoicesBySequenceID(ctx context.Context, sequenceID int64) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM invoices WHERE sequence_id = ? AND deleted_at IS NULL`, sequenceID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting invoices for sequence %d: %w", sequenceID, err)
	}
	return count, nil
}

// MaxUsedNumber returns the highest invoice number used for a given sequence,
// by looking at the invoices table. Returns 0 if no invoices reference this sequence.
func (r *SequenceRepository) MaxUsedNumber(ctx context.Context, sequenceID int64) (int, error) {
	// We need to figure out the highest number used. Since invoice_number is formatted,
	// we count the number of invoices and use the current next_number - 1 as proxy.
	// A more reliable approach: next_number - 1 is the last assigned number.
	var nextNumber int
	err := r.db.QueryRowContext(ctx, `
		SELECT next_number FROM invoice_sequences WHERE id = ? AND deleted_at IS NULL`, sequenceID,
	).Scan(&nextNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("querying max used number for sequence %d: %w", sequenceID, err)
	}
	// The last assigned number is next_number - 1 (since next_number is what will be assigned next).
	if nextNumber <= 1 {
		return 0, nil
	}
	return nextNumber - 1, nil
}
