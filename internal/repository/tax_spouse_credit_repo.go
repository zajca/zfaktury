package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// TaxSpouseCreditRepository handles persistence of TaxSpouseCredit entities.
type TaxSpouseCreditRepository struct {
	db *sql.DB
}

// NewTaxSpouseCreditRepository creates a new TaxSpouseCreditRepository.
func NewTaxSpouseCreditRepository(db *sql.DB) *TaxSpouseCreditRepository {
	return &TaxSpouseCreditRepository{db: db}
}

// taxSpouseCreditColumns is the list of columns to select for a TaxSpouseCredit row.
const taxSpouseCreditColumns = `id, year, spouse_name, spouse_birth_number, spouse_income, spouse_ztp, months_claimed, credit_amount, created_at, updated_at`

// scanTaxSpouseCredit scans a TaxSpouseCredit from a row.
func scanTaxSpouseCredit(s scanner) (*domain.TaxSpouseCredit, error) {
	c := &domain.TaxSpouseCredit{}
	var createdAtStr, updatedAtStr string
	var spouseZTP int

	err := s.Scan(
		&c.ID, &c.Year, &c.SpouseName, &c.SpouseBirthNumber,
		&c.SpouseIncome, &spouseZTP, &c.MonthsClaimed, &c.CreditAmount,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	c.SpouseZTP = spouseZTP != 0

	c.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning tax_spouse_credit created_at: %w", err)
	}
	c.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning tax_spouse_credit updated_at: %w", err)
	}

	return c, nil
}

// Upsert inserts or replaces a spouse credit for a given year.
func (r *TaxSpouseCreditRepository) Upsert(ctx context.Context, credit *domain.TaxSpouseCredit) error {
	now := time.Now()
	credit.CreatedAt = now
	credit.UpdatedAt = now

	var spouseZTP int
	if credit.SpouseZTP {
		spouseZTP = 1
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO tax_spouse_credits (year, spouse_name, spouse_birth_number, spouse_income, spouse_ztp, months_claimed, credit_amount, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		credit.Year, credit.SpouseName, credit.SpouseBirthNumber,
		credit.SpouseIncome, spouseZTP, credit.MonthsClaimed, credit.CreditAmount,
		credit.CreatedAt.Format(time.RFC3339), credit.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("upserting tax_spouse_credit for year %d: %w", credit.Year, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for tax_spouse_credit: %w", err)
	}
	credit.ID = id
	return nil
}

// GetByYear retrieves the spouse credit for a given year.
func (r *TaxSpouseCreditRepository) GetByYear(ctx context.Context, year int) (*domain.TaxSpouseCredit, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+taxSpouseCreditColumns+` FROM tax_spouse_credits WHERE year = ?`, year)
	credit, err := scanTaxSpouseCredit(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("tax_spouse_credit for year %d: %w", year, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("querying tax_spouse_credit for year %d: %w", year, err)
	}
	return credit, nil
}

// DeleteByYear removes the spouse credit for a given year.
func (r *TaxSpouseCreditRepository) DeleteByYear(ctx context.Context, year int) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM tax_spouse_credits WHERE year = ?`, year)
	if err != nil {
		return fmt.Errorf("deleting tax_spouse_credit for year %d: %w", year, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for tax_spouse_credit year %d delete: %w", year, err)
	}
	if rows == 0 {
		return fmt.Errorf("tax_spouse_credit for year %d: %w", year, domain.ErrNotFound)
	}
	return nil
}
