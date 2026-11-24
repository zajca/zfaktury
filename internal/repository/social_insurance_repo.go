package repository

import (
	"context"
	"database/sql"
	"encoding/json"
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
const socialInsuranceColumns = `id, year, filing_type, activity_months,
	total_revenue, total_expenses, tax_base,
	assessment_base, min_assessment_base, final_assessment_base,
	insurance_rate, total_insurance, prepayments, difference, new_monthly_prepay,
	new_monthly_base, filing_method, advance_start_month, advance_periods, warnings,
	secondary_months, secondary_reason, employment_assessment_base, reported_employment_base,
	main_tax_base_share, secondary_tax_base_share, secondary_assessment_base,
	main_partial_base, secondary_partial_base, secondary_decisive_amount, sum_base, insured_base,
	xml_data, status, filed_at, created_at, updated_at`

// advancePeriodRow is the JSON form of a domain.AdvancePeriod in the
// advance_periods column; amounts are in halere.
type advancePeriodRow struct {
	FromMonth     int   `json:"from_month"`
	MonthlyBase   int64 `json:"monthly_base"`
	MonthlyPrepay int64 `json:"monthly_prepay"`
}

// encodeAdvancePeriods serialises the periods for storage; no periods give "".
func encodeAdvancePeriods(periods []domain.AdvancePeriod) (string, error) {
	if len(periods) == 0 {
		return "", nil
	}
	rows := make([]advancePeriodRow, 0, len(periods))
	for _, p := range periods {
		rows = append(rows, advancePeriodRow{
			FromMonth:     p.FromMonth,
			MonthlyBase:   int64(p.MonthlyBase),
			MonthlyPrepay: int64(p.MonthlyPrepay),
		})
	}
	data, err := json.Marshal(rows)
	if err != nil {
		return "", fmt.Errorf("encoding advance periods: %w", err)
	}
	return string(data), nil
}

// decodeAdvancePeriods parses the stored periods. They must start in months
// 1-12 in strictly increasing order.
func decodeAdvancePeriods(value string) ([]domain.AdvancePeriod, error) {
	if value == "" {
		return nil, nil
	}
	var rows []advancePeriodRow
	if err := json.Unmarshal([]byte(value), &rows); err != nil {
		return nil, fmt.Errorf("decoding advance periods: %w", err)
	}
	periods := make([]domain.AdvancePeriod, 0, len(rows))
	previous := 0
	for _, row := range rows {
		if row.FromMonth < 1 || row.FromMonth > 12 || row.FromMonth <= previous {
			return nil, fmt.Errorf("advance period from_month %d is out of range or not increasing", row.FromMonth)
		}
		previous = row.FromMonth
		periods = append(periods, domain.AdvancePeriod{
			FromMonth:     row.FromMonth,
			MonthlyBase:   domain.Amount(row.MonthlyBase),
			MonthlyPrepay: domain.Amount(row.MonthlyPrepay),
		})
	}
	return periods, nil
}

// activityMonthsFromDB converts the stored activity_months integer into a mask.
// Values that cannot be a January-December mask are reported as a plain error,
// because they signal a corrupted row rather than invalid client input.
func activityMonthsFromDB(value int64) (domain.ActivityMonths, error) {
	if value < 1 || value > int64(domain.FullYearActivityMonths) {
		return 0, fmt.Errorf("activity_months %d out of range 1..4095", value)
	}
	return domain.ActivityMonths(value), nil
}

// nullableAmount converts an optional amount into a database value; nil is
// stored as NULL.
func nullableAmount(amount *domain.Amount) any {
	if amount == nil {
		return nil
	}
	return int64(*amount)
}

// optionalAmount converts a nullable database integer into an optional amount.
func optionalAmount(value sql.NullInt64) *domain.Amount {
	if !value.Valid {
		return nil
	}
	amount := domain.Amount(value.Int64)
	return &amount
}

// scanSocialInsuranceOverview scans a SocialInsuranceOverview from a row.
func scanSocialInsuranceOverview(s scanner) (*domain.SocialInsuranceOverview, error) {
	sio := &domain.SocialInsuranceOverview{}
	var filedAtStr sql.NullString
	var createdAtStr, updatedAtStr string
	var xmlData []byte
	var activityMonths int64
	var newMonthlyBase sql.NullInt64
	var filingMethod, advancePeriods, warnings, secondaryReason string
	var secondaryMonths int64
	var employmentBase, reportedEmploymentBase sql.NullInt64

	err := s.Scan(
		&sio.ID, &sio.Year, &sio.FilingType, &activityMonths,
		&sio.TotalRevenue, &sio.TotalExpenses, &sio.TaxBase,
		&sio.AssessmentBase, &sio.MinAssessmentBase, &sio.FinalAssessmentBase,
		&sio.InsuranceRate, &sio.TotalInsurance, &sio.Prepayments, &sio.Difference, &sio.NewMonthlyPrepay,
		&newMonthlyBase, &filingMethod, &sio.AdvanceStartMonth, &advancePeriods, &warnings,
		&secondaryMonths, &secondaryReason, &employmentBase, &reportedEmploymentBase,
		&sio.MainTaxBaseShare, &sio.SecondaryTaxBaseShare, &sio.SecondaryAssessmentBase,
		&sio.MainPartialBase, &sio.SecondaryPartialBase, &sio.SecondaryDecisiveAmount, &sio.SumBase, &sio.InsuredBase,
		&xmlData, &sio.Status, &filedAtStr,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	if secondaryMonths < 0 || secondaryMonths > int64(domain.FullYearActivityMonths) {
		return nil, fmt.Errorf("secondary_months %d out of range 0..4095", secondaryMonths)
	}
	sio.SecondaryMonths = domain.ActivityMonths(secondaryMonths)
	sio.SecondaryReason = domain.SecondaryActivityReason(secondaryReason)
	sio.EmploymentAssessmentBase = optionalAmount(employmentBase)
	sio.ReportedEmploymentBase = optionalAmount(reportedEmploymentBase)

	sio.FilingMethod = domain.FilingMethod(filingMethod)
	sio.AdvancePeriods, err = decodeAdvancePeriods(advancePeriods)
	if err != nil {
		return nil, fmt.Errorf("scanning social_insurance_overview %d advance_periods: %w", sio.ID, err)
	}
	sio.Warnings = splitWarnings(warnings)

	months, err := activityMonthsFromDB(activityMonths)
	if err != nil {
		return nil, err
	}
	sio.ActivityMonths = months
	sio.XMLData = xmlData
	if newMonthlyBase.Valid {
		base := domain.Amount(newMonthlyBase.Int64)
		sio.NewMonthlyBase = &base
	}

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
	advancePeriods, err := encodeAdvancePeriods(sio.AdvancePeriods)
	if err != nil {
		return fmt.Errorf("inserting social_insurance_overview: %w", err)
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO social_insurance_overviews (
			year, filing_type, activity_months,
			total_revenue, total_expenses, tax_base,
			assessment_base, min_assessment_base, final_assessment_base,
			insurance_rate, total_insurance, prepayments, difference, new_monthly_prepay,
			new_monthly_base, filing_method, advance_start_month, advance_periods, warnings,
			secondary_months, secondary_reason, employment_assessment_base, reported_employment_base,
			main_tax_base_share, secondary_tax_base_share, secondary_assessment_base,
			main_partial_base, secondary_partial_base, secondary_decisive_amount, sum_base, insured_base,
			xml_data, status, filed_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sio.Year, sio.FilingType, int64(sio.ActivityMonths),
		sio.TotalRevenue, sio.TotalExpenses, sio.TaxBase,
		sio.AssessmentBase, sio.MinAssessmentBase, sio.FinalAssessmentBase,
		sio.InsuranceRate, sio.TotalInsurance, sio.Prepayments, sio.Difference, sio.NewMonthlyPrepay,
		nullableAmount(sio.NewMonthlyBase), string(sio.FilingMethod), sio.AdvanceStartMonth, advancePeriods, joinWarnings(sio.Warnings),
		int64(sio.SecondaryMonths), string(sio.SecondaryReason), nullableAmount(sio.EmploymentAssessmentBase), nullableAmount(sio.ReportedEmploymentBase),
		sio.MainTaxBaseShare, sio.SecondaryTaxBaseShare, sio.SecondaryAssessmentBase,
		sio.MainPartialBase, sio.SecondaryPartialBase, sio.SecondaryDecisiveAmount, sio.SumBase, sio.InsuredBase,
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
	advancePeriods, err := encodeAdvancePeriods(sio.AdvancePeriods)
	if err != nil {
		return fmt.Errorf("updating social_insurance_overview %d: %w", sio.ID, err)
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE social_insurance_overviews SET
			year = ?, filing_type = ?, activity_months = ?,
			total_revenue = ?, total_expenses = ?, tax_base = ?,
			assessment_base = ?, min_assessment_base = ?, final_assessment_base = ?,
			insurance_rate = ?, total_insurance = ?, prepayments = ?, difference = ?, new_monthly_prepay = ?,
			new_monthly_base = ?, filing_method = ?, advance_start_month = ?, advance_periods = ?, warnings = ?,
			secondary_months = ?, secondary_reason = ?, employment_assessment_base = ?, reported_employment_base = ?,
			main_tax_base_share = ?, secondary_tax_base_share = ?, secondary_assessment_base = ?,
			main_partial_base = ?, secondary_partial_base = ?, secondary_decisive_amount = ?, sum_base = ?, insured_base = ?,
			xml_data = ?, status = ?, filed_at = ?, updated_at = ?
		WHERE id = ?`,
		sio.Year, sio.FilingType, int64(sio.ActivityMonths),
		sio.TotalRevenue, sio.TotalExpenses, sio.TaxBase,
		sio.AssessmentBase, sio.MinAssessmentBase, sio.FinalAssessmentBase,
		sio.InsuranceRate, sio.TotalInsurance, sio.Prepayments, sio.Difference, sio.NewMonthlyPrepay,
		nullableAmount(sio.NewMonthlyBase), string(sio.FilingMethod), sio.AdvanceStartMonth, advancePeriods, joinWarnings(sio.Warnings),
		int64(sio.SecondaryMonths), string(sio.SecondaryReason), nullableAmount(sio.EmploymentAssessmentBase), nullableAmount(sio.ReportedEmploymentBase),
		sio.MainTaxBaseShare, sio.SecondaryTaxBaseShare, sio.SecondaryAssessmentBase,
		sio.MainPartialBase, sio.SecondaryPartialBase, sio.SecondaryDecisiveAmount, sio.SumBase, sio.InsuredBase,
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
