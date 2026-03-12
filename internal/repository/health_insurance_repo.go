package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// HealthInsuranceOverviewRepository handles persistence of HealthInsuranceOverview entities.
type HealthInsuranceOverviewRepository struct {
	db *sql.DB
}

// NewHealthInsuranceOverviewRepository creates a new HealthInsuranceOverviewRepository.
func NewHealthInsuranceOverviewRepository(db *sql.DB) *HealthInsuranceOverviewRepository {
	return &HealthInsuranceOverviewRepository{db: db}
}

// healthInsuranceColumns is the list of columns to select for a HealthInsuranceOverview row.
const healthInsuranceColumns = `id, year, filing_type, total_revenue, total_expenses,
	tax_base, assessment_base, min_assessment_base, final_assessment_base,
	insurance_rate, total_insurance, prepayments, difference, new_monthly_prepay,
	xml_data, status, filed_at, created_at, updated_at`

// scanHealthInsuranceOverview scans a HealthInsuranceOverview from a row.
func scanHealthInsuranceOverview(s scanner) (*domain.HealthInsuranceOverview, error) {
	hi := &domain.HealthInsuranceOverview{}
	var filedAtStr sql.NullString
	var createdAtStr, updatedAtStr string
	var xmlData []byte

	err := s.Scan(
		&hi.ID, &hi.Year, &hi.FilingType,
		&hi.TotalRevenue, &hi.TotalExpenses,
		&hi.TaxBase, &hi.AssessmentBase, &hi.MinAssessmentBase, &hi.FinalAssessmentBase,
		&hi.InsuranceRate, &hi.TotalInsurance, &hi.Prepayments, &hi.Difference, &hi.NewMonthlyPrepay,
		&xmlData, &hi.Status, &filedAtStr,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	hi.XMLData = xmlData

	hi.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning health_insurance_overview created_at: %w", err)
	}
	hi.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning health_insurance_overview updated_at: %w", err)
	}
	hi.FiledAt, err = parseDatePtr(time.RFC3339, filedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning health_insurance_overview filed_at: %w", err)
	}

	return hi, nil
}

// Create inserts a new health insurance overview into the database.
func (r *HealthInsuranceOverviewRepository) Create(ctx context.Context, hi *domain.HealthInsuranceOverview) error {
	now := time.Now()
	hi.CreatedAt = now
	hi.UpdatedAt = now
	if hi.Status == "" {
		hi.Status = domain.FilingStatusDraft
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO health_insurance_overviews (
			year, filing_type, total_revenue, total_expenses,
			tax_base, assessment_base, min_assessment_base, final_assessment_base,
			insurance_rate, total_insurance, prepayments, difference, new_monthly_prepay,
			xml_data, status, filed_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		hi.Year, hi.FilingType, hi.TotalRevenue, hi.TotalExpenses,
		hi.TaxBase, hi.AssessmentBase, hi.MinAssessmentBase, hi.FinalAssessmentBase,
		hi.InsuranceRate, hi.TotalInsurance, hi.Prepayments, hi.Difference, hi.NewMonthlyPrepay,
		nil, hi.Status, nil,
		hi.CreatedAt.Format(time.RFC3339), hi.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting health_insurance_overview: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for health_insurance_overview: %w", err)
	}
	hi.ID = id
	return nil
}

// Update modifies an existing health insurance overview.
func (r *HealthInsuranceOverviewRepository) Update(ctx context.Context, hi *domain.HealthInsuranceOverview) error {
	hi.UpdatedAt = time.Now()

	var filedAt any
	if hi.FiledAt != nil {
		filedAt = hi.FiledAt.Format(time.RFC3339)
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE health_insurance_overviews SET
			year = ?, filing_type = ?, total_revenue = ?, total_expenses = ?,
			tax_base = ?, assessment_base = ?, min_assessment_base = ?, final_assessment_base = ?,
			insurance_rate = ?, total_insurance = ?, prepayments = ?, difference = ?, new_monthly_prepay = ?,
			xml_data = ?, status = ?, filed_at = ?, updated_at = ?
		WHERE id = ?`,
		hi.Year, hi.FilingType, hi.TotalRevenue, hi.TotalExpenses,
		hi.TaxBase, hi.AssessmentBase, hi.MinAssessmentBase, hi.FinalAssessmentBase,
		hi.InsuranceRate, hi.TotalInsurance, hi.Prepayments, hi.Difference, hi.NewMonthlyPrepay,
		hi.XMLData, hi.Status, filedAt,
		hi.UpdatedAt.Format(time.RFC3339), hi.ID,
	)
	if err != nil {
		return fmt.Errorf("updating health_insurance_overview %d: %w", hi.ID, err)
	}
	return nil
}

// Delete removes a health insurance overview by ID.
func (r *HealthInsuranceOverviewRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM health_insurance_overviews WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting health_insurance_overview %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for health_insurance_overview %d delete: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("health_insurance_overview %d: %w", id, domain.ErrNotFound)
	}
	return nil
}

// GetByID retrieves a health insurance overview by its ID.
func (r *HealthInsuranceOverviewRepository) GetByID(ctx context.Context, id int64) (*domain.HealthInsuranceOverview, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+healthInsuranceColumns+` FROM health_insurance_overviews WHERE id = ?`, id)
	hi, err := scanHealthInsuranceOverview(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("health_insurance_overview %d: %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("querying health_insurance_overview %d: %w", id, err)
	}
	return hi, nil
}

// List retrieves all health insurance overviews for a given year.
func (r *HealthInsuranceOverviewRepository) List(ctx context.Context, year int) ([]domain.HealthInsuranceOverview, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+healthInsuranceColumns+` FROM health_insurance_overviews WHERE year = ? ORDER BY created_at ASC`,
		year,
	)
	if err != nil {
		return nil, fmt.Errorf("listing health_insurance_overviews for year %d: %w", year, err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.HealthInsuranceOverview
	for rows.Next() {
		hi, err := scanHealthInsuranceOverview(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning health_insurance_overview row: %w", err)
		}
		result = append(result, *hi)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating health_insurance_overview rows: %w", err)
	}
	return result, nil
}

// GetByYear retrieves a health insurance overview for a specific year and filing type.
func (r *HealthInsuranceOverviewRepository) GetByYear(ctx context.Context, year int, filingType string) (*domain.HealthInsuranceOverview, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+healthInsuranceColumns+` FROM health_insurance_overviews WHERE year = ? AND filing_type = ?`,
		year, filingType,
	)
	hi, err := scanHealthInsuranceOverview(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("health_insurance_overview for year %d (%s): %w", year, filingType, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("querying health_insurance_overview by year: %w", err)
	}
	return hi, nil
}
