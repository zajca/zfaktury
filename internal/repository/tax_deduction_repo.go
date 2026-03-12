package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// TaxDeductionRepository handles persistence of TaxDeduction entities.
type TaxDeductionRepository struct {
	db *sql.DB
}

// NewTaxDeductionRepository creates a new TaxDeductionRepository.
func NewTaxDeductionRepository(db *sql.DB) *TaxDeductionRepository {
	return &TaxDeductionRepository{db: db}
}

// taxDeductionColumns is the list of columns to select for a TaxDeduction row.
const taxDeductionColumns = `id, year, category, description, claimed_amount, max_amount, allowed_amount, created_at, updated_at`

// scanTaxDeduction scans a TaxDeduction from a row.
func scanTaxDeduction(s scanner) (*domain.TaxDeduction, error) {
	d := &domain.TaxDeduction{}
	var createdAtStr, updatedAtStr string

	err := s.Scan(
		&d.ID, &d.Year, &d.Category, &d.Description,
		&d.ClaimedAmount, &d.MaxAmount, &d.AllowedAmount,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	d.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning tax_deduction created_at: %w", err)
	}
	d.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning tax_deduction updated_at: %w", err)
	}

	return d, nil
}

// Create inserts a new tax deduction into the database.
func (r *TaxDeductionRepository) Create(ctx context.Context, ded *domain.TaxDeduction) error {
	now := time.Now()
	ded.CreatedAt = now
	ded.UpdatedAt = now

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO tax_deductions (year, category, description, claimed_amount, max_amount, allowed_amount, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		ded.Year, ded.Category, ded.Description,
		ded.ClaimedAmount, ded.MaxAmount, ded.AllowedAmount,
		ded.CreatedAt.Format(time.RFC3339), ded.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting tax_deduction: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for tax_deduction: %w", err)
	}
	ded.ID = id
	return nil
}

// Update modifies an existing tax deduction.
func (r *TaxDeductionRepository) Update(ctx context.Context, ded *domain.TaxDeduction) error {
	ded.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, `
		UPDATE tax_deductions SET
			year = ?, category = ?, description = ?, claimed_amount = ?,
			max_amount = ?, allowed_amount = ?, updated_at = ?
		WHERE id = ?`,
		ded.Year, ded.Category, ded.Description, ded.ClaimedAmount,
		ded.MaxAmount, ded.AllowedAmount,
		ded.UpdatedAt.Format(time.RFC3339), ded.ID,
	)
	if err != nil {
		return fmt.Errorf("updating tax_deduction %d: %w", ded.ID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for tax_deduction %d: %w", ded.ID, err)
	}
	if rows == 0 {
		return fmt.Errorf("tax_deduction %d: %w", ded.ID, domain.ErrNotFound)
	}
	return nil
}

// Delete removes a tax deduction by ID.
func (r *TaxDeductionRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM tax_deductions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting tax_deduction %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for tax_deduction %d delete: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("tax_deduction %d: %w", id, domain.ErrNotFound)
	}
	return nil
}

// GetByID retrieves a tax deduction by its ID.
func (r *TaxDeductionRepository) GetByID(ctx context.Context, id int64) (*domain.TaxDeduction, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+taxDeductionColumns+` FROM tax_deductions WHERE id = ?`, id)
	ded, err := scanTaxDeduction(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("tax_deduction %d: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("querying tax_deduction %d: %w", id, err)
	}
	return ded, nil
}

// ListByYear retrieves all tax deductions for a given year, ordered by category and id.
func (r *TaxDeductionRepository) ListByYear(ctx context.Context, year int) ([]domain.TaxDeduction, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+taxDeductionColumns+` FROM tax_deductions WHERE year = ? ORDER BY category, id`,
		year,
	)
	if err != nil {
		return nil, fmt.Errorf("listing tax_deductions for year %d: %w", year, err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.TaxDeduction
	for rows.Next() {
		ded, err := scanTaxDeduction(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning tax_deduction row: %w", err)
		}
		result = append(result, *ded)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tax_deduction rows: %w", err)
	}
	return result, nil
}
