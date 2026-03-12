package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// SocialInsuranceOverviewRepository handles persistence of SocialInsuranceOverview entities.
type SocialInsuranceOverviewRepository struct {
	db *sql.DB
}

// NewSocialInsuranceOverviewRepository creates a new SocialInsuranceOverviewRepository.
func NewSocialInsuranceOverviewRepository(db *sql.DB) *SocialInsuranceOverviewRepository {
	return &SocialInsuranceOverviewRepository{db: db}
}

// socialInsuranceColumns is the list of columns to select for a SocialInsuranceOverview row.
const socialInsuranceColumns = `id, year, filing_type,
	total_revenue, total_expenses, tax_base,
	assessment_base, min_assessment_base, final_assessment_base,
	insurance_rate, total_insurance, prepayments, difference, new_monthly_prepay,
	xml_data, status, filed_at, created_at, updated_at`

// scanSocialInsuranceOverview scans a SocialInsuranceOverview from a row.
func scanSocialInsuranceOverview(s scanner) (*domain.SocialInsuranceOverview, error) {
	sio := &domain.SocialInsuranceOverview{}
	var filedAtStr sql.NullString
	var createdAtStr, updatedAtStr string
	var xmlData []byte

	err := s.Scan(
		&sio.ID, &sio.Year, &sio.FilingType,
		&sio.TotalRevenue, &sio.TotalExpenses, &sio.TaxBase,
		&sio.AssessmentBase, &sio.MinAssessmentBase, &sio.FinalAssessmentBase,
		&sio.InsuranceRate, &sio.TotalInsurance, &sio.Prepayments, &sio.Difference, &sio.NewMonthlyPrepay,
		&xmlData, &sio.Status, &filedAtStr,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	sio.XMLData = xmlData

	sio.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning social_insurance_overview created_at: %w", err)
	}
	sio.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning social_insurance_overview updated_at: %w", err)
	}
	sio.FiledAt, err = parseDatePtr(time.RFC3339, filedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning social_insurance_overview filed_at: %w", err)
	}

	return sio, nil
}

// Create inserts a new social insurance overview into the database.
func (r *SocialInsuranceOverviewRepository) Create(ctx context.Context, sio *domain.SocialInsuranceOverview) error {
	now := time.Now()
	sio.CreatedAt = now
	sio.UpdatedAt = now
	if sio.Status == "" {
		sio.Status = domain.FilingStatusDraft
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO social_insurance_overviews (
			year, filing_type,
			total_revenue, total_expenses, tax_base,
			assessment_base, min_assessment_base, final_assessment_base,
			insurance_rate, total_insurance, prepayments, difference, new_monthly_prepay,
			xml_data, status, filed_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sio.Year, sio.FilingType,
		sio.TotalRevenue, sio.TotalExpenses, sio.TaxBase,
		sio.AssessmentBase, sio.MinAssessmentBase, sio.FinalAssessmentBase,
		sio.InsuranceRate, sio.TotalInsurance, sio.Prepayments, sio.Difference, sio.NewMonthlyPrepay,
		nil, sio.Status, nil,
		sio.CreatedAt.Format(time.RFC3339), sio.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting social_insurance_overview: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for social_insurance_overview: %w", err)
	}
	sio.ID = id
	return nil
}

// Update modifies an existing social insurance overview.
func (r *SocialInsuranceOverviewRepository) Update(ctx context.Context, sio *domain.SocialInsuranceOverview) error {
	sio.UpdatedAt = time.Now()

	var filedAt any
	if sio.FiledAt != nil {
		filedAt = sio.FiledAt.Format(time.RFC3339)
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE social_insurance_overviews SET
			year = ?, filing_type = ?,
			total_revenue = ?, total_expenses = ?, tax_base = ?,
			assessment_base = ?, min_assessment_base = ?, final_assessment_base = ?,
			insurance_rate = ?, total_insurance = ?, prepayments = ?, difference = ?, new_monthly_prepay = ?,
			xml_data = ?, status = ?, filed_at = ?, updated_at = ?
		WHERE id = ?`,
		sio.Year, sio.FilingType,
		sio.TotalRevenue, sio.TotalExpenses, sio.TaxBase,
		sio.AssessmentBase, sio.MinAssessmentBase, sio.FinalAssessmentBase,
		sio.InsuranceRate, sio.TotalInsurance, sio.Prepayments, sio.Difference, sio.NewMonthlyPrepay,
		sio.XMLData, sio.Status, filedAt,
		sio.UpdatedAt.Format(time.RFC3339), sio.ID,
	)
	if err != nil {
		return fmt.Errorf("updating social_insurance_overview %d: %w", sio.ID, err)
	}
	return nil
}

// Delete removes a social insurance overview by ID.
func (r *SocialInsuranceOverviewRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM social_insurance_overviews WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting social_insurance_overview %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for social_insurance_overview %d delete: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("social_insurance_overview %d: %w", id, domain.ErrNotFound)
	}
	return nil
}

// GetByID retrieves a social insurance overview by its ID.
func (r *SocialInsuranceOverviewRepository) GetByID(ctx context.Context, id int64) (*domain.SocialInsuranceOverview, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+socialInsuranceColumns+` FROM social_insurance_overviews WHERE id = ?`, id)
	sio, err := scanSocialInsuranceOverview(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("social_insurance_overview %d: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("querying social_insurance_overview %d: %w", id, err)
	}
	return sio, nil
}

// List retrieves all social insurance overviews for a given year.
func (r *SocialInsuranceOverviewRepository) List(ctx context.Context, year int) ([]domain.SocialInsuranceOverview, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+socialInsuranceColumns+` FROM social_insurance_overviews WHERE year = ? ORDER BY created_at ASC`,
		year,
	)
	if err != nil {
		return nil, fmt.Errorf("listing social_insurance_overviews for year %d: %w", year, err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.SocialInsuranceOverview
	for rows.Next() {
		sio, err := scanSocialInsuranceOverview(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning social_insurance_overview row: %w", err)
		}
		result = append(result, *sio)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating social_insurance_overview rows: %w", err)
	}
	return result, nil
}

// GetByYear retrieves a social insurance overview for a specific year and filing type.
func (r *SocialInsuranceOverviewRepository) GetByYear(ctx context.Context, year int, filingType string) (*domain.SocialInsuranceOverview, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+socialInsuranceColumns+` FROM social_insurance_overviews WHERE year = ? AND filing_type = ?`,
		year, filingType,
	)
	sio, err := scanSocialInsuranceOverview(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("social_insurance_overview for year %d (%s): %w", year, filingType, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("querying social_insurance_overview by year: %w", err)
	}
	return sio, nil
}
