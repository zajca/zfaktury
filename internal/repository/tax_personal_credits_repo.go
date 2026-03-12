package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// TaxPersonalCreditsRepository handles persistence of TaxPersonalCredits entities.
type TaxPersonalCreditsRepository struct {
	db *sql.DB
}

// NewTaxPersonalCreditsRepository creates a new TaxPersonalCreditsRepository.
func NewTaxPersonalCreditsRepository(db *sql.DB) *TaxPersonalCreditsRepository {
	return &TaxPersonalCreditsRepository{db: db}
}

// taxPersonalCreditsColumns is the list of columns to select for a TaxPersonalCredits row.
const taxPersonalCreditsColumns = `year, is_student, student_months, disability_level, credit_student, credit_disability, created_at, updated_at`

// scanTaxPersonalCredits scans a TaxPersonalCredits from a row.
func scanTaxPersonalCredits(s scanner) (*domain.TaxPersonalCredits, error) {
	c := &domain.TaxPersonalCredits{}
	var createdAtStr, updatedAtStr string
	var isStudent int

	err := s.Scan(
		&c.Year, &isStudent, &c.StudentMonths, &c.DisabilityLevel,
		&c.CreditStudent, &c.CreditDisability,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	c.IsStudent = isStudent != 0

	c.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning tax_personal_credits created_at: %w", err)
	}
	c.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning tax_personal_credits updated_at: %w", err)
	}

	return c, nil
}

// Upsert inserts or replaces the personal credits for a given year.
func (r *TaxPersonalCreditsRepository) Upsert(ctx context.Context, credits *domain.TaxPersonalCredits) error {
	now := time.Now()
	credits.CreatedAt = now
	credits.UpdatedAt = now

	var isStudent int
	if credits.IsStudent {
		isStudent = 1
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO tax_personal_credits (year, is_student, student_months, disability_level, credit_student, credit_disability, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		credits.Year, isStudent, credits.StudentMonths, credits.DisabilityLevel,
		credits.CreditStudent, credits.CreditDisability,
		credits.CreatedAt.Format(time.RFC3339), credits.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("upserting tax_personal_credits for year %d: %w", credits.Year, err)
	}
	return nil
}

// GetByYear retrieves the personal credits for a given year.
func (r *TaxPersonalCreditsRepository) GetByYear(ctx context.Context, year int) (*domain.TaxPersonalCredits, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+taxPersonalCreditsColumns+` FROM tax_personal_credits WHERE year = ?`, year)
	credits, err := scanTaxPersonalCredits(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("tax_personal_credits for year %d: %w", year, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("querying tax_personal_credits for year %d: %w", year, err)
	}
	return credits, nil
}
