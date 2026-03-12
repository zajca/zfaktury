package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zajca/zfaktury/internal/domain"
)

// TaxPrepaymentRepository handles persistence of monthly tax prepayments.
type TaxPrepaymentRepository struct {
	db *sql.DB
}

// NewTaxPrepaymentRepository creates a new TaxPrepaymentRepository.
func NewTaxPrepaymentRepository(db *sql.DB) *TaxPrepaymentRepository {
	return &TaxPrepaymentRepository{db: db}
}

// ListByYear retrieves all prepayments for a given year, ordered by month.
func (r *TaxPrepaymentRepository) ListByYear(ctx context.Context, year int) ([]domain.TaxPrepayment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT year, month, tax_amount, social_amount, health_amount
		 FROM tax_prepayments WHERE year = ? ORDER BY month`, year)
	if err != nil {
		return nil, fmt.Errorf("querying tax_prepayments for year %d: %w", year, err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.TaxPrepayment
	for rows.Next() {
		var tp domain.TaxPrepayment
		if err := rows.Scan(&tp.Year, &tp.Month, &tp.TaxAmount, &tp.SocialAmount, &tp.HealthAmount); err != nil {
			return nil, fmt.Errorf("scanning tax_prepayment row: %w", err)
		}
		result = append(result, tp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tax_prepayment rows: %w", err)
	}
	return result, nil
}

// UpsertAll replaces all prepayments for a given year within a transaction.
func (r *TaxPrepaymentRepository) UpsertAll(ctx context.Context, year int, prepayments []domain.TaxPrepayment) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction for tax_prepayments upsert: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM tax_prepayments WHERE year = ?`, year); err != nil {
		return fmt.Errorf("deleting old tax_prepayments for year %d: %w", year, err)
	}

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO tax_prepayments (year, month, tax_amount, social_amount, health_amount)
		 VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("preparing tax_prepayments insert: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, tp := range prepayments {
		if _, err := stmt.ExecContext(ctx, year, tp.Month, tp.TaxAmount, tp.SocialAmount, tp.HealthAmount); err != nil {
			return fmt.Errorf("inserting tax_prepayment month %d: %w", tp.Month, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing tax_prepayments upsert: %w", err)
	}
	return nil
}

// SumByYear returns the annual sum of prepayments for a given year.
func (r *TaxPrepaymentRepository) SumByYear(ctx context.Context, year int) (taxTotal, socialTotal, healthTotal domain.Amount, err error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(tax_amount), 0), COALESCE(SUM(social_amount), 0), COALESCE(SUM(health_amount), 0)
		 FROM tax_prepayments WHERE year = ?`, year)

	if err = row.Scan(&taxTotal, &socialTotal, &healthTotal); err != nil {
		err = fmt.Errorf("summing tax_prepayments for year %d: %w", year, err)
	}
	return
}
