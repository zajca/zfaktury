package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// TaxYearSettingsRepository handles persistence of per-year tax settings.
type TaxYearSettingsRepository struct {
	db *sql.DB
}

// NewTaxYearSettingsRepository creates a new TaxYearSettingsRepository.
func NewTaxYearSettingsRepository(db *sql.DB) *TaxYearSettingsRepository {
	return &TaxYearSettingsRepository{db: db}
}

// GetByYear retrieves tax year settings for a specific year within the given company.
func (r *TaxYearSettingsRepository) GetByYear(ctx context.Context, companyID int64, year int) (*domain.TaxYearSettings, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT year, flat_rate_percent, created_at, updated_at FROM tax_year_settings WHERE company_id = ? AND year = ?`,
		companyID, year)

	var tys domain.TaxYearSettings
	var createdAt, updatedAt string
	err := row.Scan(&tys.Year, &tys.FlatRatePercent, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("querying tax_year_settings for year %d: %w", year, err)
	}

	tys.CreatedAt, err = parseDate(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parsing created_at for tax_year_settings year %d: %w", year, err)
	}
	tys.UpdatedAt, err = parseDate(time.RFC3339, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing updated_at for tax_year_settings year %d: %w", year, err)
	}
	return &tys, nil
}

// Upsert inserts or updates tax year settings for a specific year within the given company.
//
// Note: the underlying tax_year_settings table has PRIMARY KEY(year) — multi-company
// support for this table is limited until a follow-up migration adds composite
// PRIMARY KEY(company_id, year). For now, two companies cannot have distinct
// flat_rate_percent for the same year.
func (r *TaxYearSettingsRepository) Upsert(ctx context.Context, companyID int64, tys *domain.TaxYearSettings) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tax_year_settings (company_id, year, flat_rate_percent, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(year) DO UPDATE SET flat_rate_percent = excluded.flat_rate_percent, updated_at = excluded.updated_at`,
		companyID, tys.Year, tys.FlatRatePercent, now, now,
	)
	if err != nil {
		return fmt.Errorf("upserting tax_year_settings for year %d: %w", tys.Year, err)
	}
	return nil
}
