package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// CapitalIncomeRepository handles persistence of CapitalIncomeEntry entities.
type CapitalIncomeRepository struct {
	db *sql.DB
}

// NewCapitalIncomeRepository creates a new CapitalIncomeRepository.
func NewCapitalIncomeRepository(db *sql.DB) *CapitalIncomeRepository {
	return &CapitalIncomeRepository{db: db}
}

// capitalIncomeColumns is the list of columns to select for a CapitalIncomeEntry row.
const capitalIncomeColumns = `id, year, document_id, category, description, income_date, gross_amount, withheld_tax_cz, withheld_tax_foreign, country_code, needs_declaring, net_amount, created_at, updated_at`

// scanCapitalIncomeEntry scans a CapitalIncomeEntry from a row.
func scanCapitalIncomeEntry(s scanner) (*domain.CapitalIncomeEntry, error) {
	e := &domain.CapitalIncomeEntry{}
	var documentID sql.NullInt64
	var incomeDateStr string
	var needsDeclaring int
	var createdAtStr, updatedAtStr string

	err := s.Scan(
		&e.ID, &e.Year, &documentID, &e.Category, &e.Description,
		&incomeDateStr, &e.GrossAmount, &e.WithheldTaxCZ, &e.WithheldTaxForeign,
		&e.CountryCode, &needsDeclaring, &e.NetAmount,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	if documentID.Valid {
		e.DocumentID = &documentID.Int64
	}

	e.NeedsDeclaring = needsDeclaring != 0

	e.IncomeDate, err = parseDate("2006-01-02", incomeDateStr)
	if err != nil {
		return nil, fmt.Errorf("scanning capital_income_entry income_date: %w", err)
	}
	e.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning capital_income_entry created_at: %w", err)
	}
	e.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning capital_income_entry updated_at: %w", err)
	}

	return e, nil
}

// boolToInt converts a bool to an int (0 or 1) for SQLite storage.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Create inserts a new capital income entry into the database.
func (r *CapitalIncomeRepository) Create(ctx context.Context, entry *domain.CapitalIncomeEntry) error {
	now := time.Now()
	entry.CreatedAt = now
	entry.UpdatedAt = now

	var documentID *int64
	if entry.DocumentID != nil {
		documentID = entry.DocumentID
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO capital_income_entries (year, document_id, category, description, income_date, gross_amount, withheld_tax_cz, withheld_tax_foreign, country_code, needs_declaring, net_amount, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.Year, documentID, entry.Category, entry.Description,
		entry.IncomeDate.Format("2006-01-02"),
		entry.GrossAmount, entry.WithheldTaxCZ, entry.WithheldTaxForeign,
		entry.CountryCode, boolToInt(entry.NeedsDeclaring), entry.NetAmount,
		entry.CreatedAt.Format(time.RFC3339), entry.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting capital_income_entry: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for capital_income_entry: %w", err)
	}
	entry.ID = id
	return nil
}

// Update modifies an existing capital income entry.
func (r *CapitalIncomeRepository) Update(ctx context.Context, entry *domain.CapitalIncomeEntry) error {
	entry.UpdatedAt = time.Now()

	var documentID *int64
	if entry.DocumentID != nil {
		documentID = entry.DocumentID
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE capital_income_entries SET
			year = ?, document_id = ?, category = ?, description = ?, income_date = ?,
			gross_amount = ?, withheld_tax_cz = ?, withheld_tax_foreign = ?,
			country_code = ?, needs_declaring = ?, net_amount = ?, updated_at = ?
		WHERE id = ?`,
		entry.Year, documentID, entry.Category, entry.Description,
		entry.IncomeDate.Format("2006-01-02"),
		entry.GrossAmount, entry.WithheldTaxCZ, entry.WithheldTaxForeign,
		entry.CountryCode, boolToInt(entry.NeedsDeclaring), entry.NetAmount,
		entry.UpdatedAt.Format(time.RFC3339), entry.ID,
	)
	if err != nil {
		return fmt.Errorf("updating capital_income_entry %d: %w", entry.ID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for capital_income_entry %d: %w", entry.ID, err)
	}
	if rows == 0 {
		return fmt.Errorf("capital_income_entry %d: %w", entry.ID, domain.ErrNotFound)
	}
	return nil
}

// Delete removes a capital income entry by ID.
func (r *CapitalIncomeRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM capital_income_entries WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting capital_income_entry %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for capital_income_entry %d delete: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("capital_income_entry %d: %w", id, domain.ErrNotFound)
	}
	return nil
}

// GetByID retrieves a capital income entry by its ID.
func (r *CapitalIncomeRepository) GetByID(ctx context.Context, id int64) (*domain.CapitalIncomeEntry, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+capitalIncomeColumns+` FROM capital_income_entries WHERE id = ?`, id)
	entry, err := scanCapitalIncomeEntry(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("capital_income_entry %d: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("querying capital_income_entry %d: %w", id, err)
	}
	return entry, nil
}

// ListByYear retrieves all capital income entries for a given year, ordered by income_date.
func (r *CapitalIncomeRepository) ListByYear(ctx context.Context, year int) ([]domain.CapitalIncomeEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+capitalIncomeColumns+` FROM capital_income_entries WHERE year = ? ORDER BY income_date ASC, id ASC`,
		year,
	)
	if err != nil {
		return nil, fmt.Errorf("listing capital_income_entries for year %d: %w", year, err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.CapitalIncomeEntry
	for rows.Next() {
		entry, err := scanCapitalIncomeEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning capital_income_entry row: %w", err)
		}
		result = append(result, *entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating capital_income_entry rows: %w", err)
	}
	return result, nil
}

// ListByDocumentID retrieves all capital income entries for a given document ID.
func (r *CapitalIncomeRepository) ListByDocumentID(ctx context.Context, documentID int64) ([]domain.CapitalIncomeEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+capitalIncomeColumns+` FROM capital_income_entries WHERE document_id = ? ORDER BY income_date ASC, id ASC`,
		documentID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing capital_income_entries for document %d: %w", documentID, err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.CapitalIncomeEntry
	for rows.Next() {
		entry, err := scanCapitalIncomeEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning capital_income_entry row: %w", err)
		}
		result = append(result, *entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating capital_income_entry rows: %w", err)
	}
	return result, nil
}

// SumByYear returns the sum of gross_amount, withheld taxes, and net_amount
// for entries where needs_declaring = 1 in a given year.
func (r *CapitalIncomeRepository) SumByYear(ctx context.Context, year int) (grossTotal, taxTotal, netTotal domain.Amount, err error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(gross_amount), 0), COALESCE(SUM(withheld_tax_cz + withheld_tax_foreign), 0), COALESCE(SUM(net_amount), 0)
		FROM capital_income_entries
		WHERE year = ? AND needs_declaring = 1`,
		year,
	)
	if err = row.Scan(&grossTotal, &taxTotal, &netTotal); err != nil {
		err = fmt.Errorf("summing capital_income_entries for year %d: %w", year, err)
		return
	}
	return
}

// DeleteByDocumentID removes all capital income entries for a given document ID.
func (r *CapitalIncomeRepository) DeleteByDocumentID(ctx context.Context, documentID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM capital_income_entries WHERE document_id = ?`, documentID)
	if err != nil {
		return fmt.Errorf("deleting capital_income_entries for document %d: %w", documentID, err)
	}
	return nil
}
