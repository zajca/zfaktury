package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/calc"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/taxrules"
)

// HealthInsuranceService provides business logic for health insurance overview management.
type HealthInsuranceService struct {
	repo                repository.HealthInsuranceOverviewRepo
	invoiceRepo         repository.InvoiceRepo
	expenseRepo         repository.ExpenseRepo
	settingsRepo        repository.SettingsRepo
	taxYearSettingsRepo repository.TaxYearSettingsRepo
	taxPrepaymentRepo   repository.TaxPrepaymentRepo
	audit               *AuditService
	now                 func() time.Time
}

// NewHealthInsuranceService creates a new HealthInsuranceService.
func NewHealthInsuranceService(
	repo repository.HealthInsuranceOverviewRepo,
	invoiceRepo repository.InvoiceRepo,
	expenseRepo repository.ExpenseRepo,
	settingsRepo repository.SettingsRepo,
	taxYearSettingsRepo repository.TaxYearSettingsRepo,
	taxPrepaymentRepo repository.TaxPrepaymentRepo,
	audit *AuditService,
) *HealthInsuranceService {
	return &HealthInsuranceService{
		repo:                repo,
		invoiceRepo:         invoiceRepo,
		expenseRepo:         expenseRepo,
		settingsRepo:        settingsRepo,
		taxYearSettingsRepo: taxYearSettingsRepo,
		taxPrepaymentRepo:   taxPrepaymentRepo,
		audit:               audit,
		now:                 time.Now,
	}
}

// Create validates and persists a new health insurance overview.
func (s *HealthInsuranceService) Create(ctx context.Context, hi *domain.HealthInsuranceOverview) error {
	if hi.Year < 2000 || hi.Year > 2100 {
		return fmt.Errorf("year out of valid range: %w", domain.ErrInvalidInput)
	}
	if hi.FilingType == "" {
		hi.FilingType = domain.FilingTypeRegular
	}
	switch hi.FilingType {
	case domain.FilingTypeRegular, domain.FilingTypeCorrective, domain.FilingTypeSupplementary:
		// ok
	default:
		return fmt.Errorf("invalid filing_type: %w", domain.ErrInvalidInput)
	}

	if err := hi.ActivityMonths.Validate(); err != nil {
		return fmt.Errorf("validating activity months: %w", err)
	}

	if hi.FilingType == domain.FilingTypeRegular {
		existing, err := s.repo.GetByYear(ctx, hi.Year, hi.FilingType)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("checking existing health_insurance_overview: %w", err)
		}
		if existing != nil {
			return domain.ErrFilingAlreadyExists
		}
	}

	if hi.Status == "" {
		hi.Status = domain.FilingStatusDraft
	}

	if err := s.repo.Create(ctx, hi); err != nil {
		return fmt.Errorf("creating health_insurance_overview: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "health_insurance", hi.ID, "create", nil, hi)
	}
	return nil
}

// GetByID retrieves a health insurance overview by its ID.
func (s *HealthInsuranceService) GetByID(ctx context.Context, id int64) (*domain.HealthInsuranceOverview, error) {
	if id == 0 {
		return nil, fmt.Errorf("health_insurance_overview ID is required: %w", domain.ErrInvalidInput)
	}
	hi, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching health_insurance_overview: %w", err)
	}
	return hi, nil
}

// List retrieves all health insurance overviews for a given year.
func (s *HealthInsuranceService) List(ctx context.Context, year int) ([]domain.HealthInsuranceOverview, error) {
	if year == 0 {
		year = time.Now().Year()
	}
	overviews, err := s.repo.List(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("listing health_insurance_overviews: %w", err)
	}
	return overviews, nil
}

// Delete removes a health insurance overview by ID. Filed overviews cannot be deleted.
func (s *HealthInsuranceService) Delete(ctx context.Context, id int64) error {
	if id == 0 {
		return fmt.Errorf("health_insurance_overview ID is required: %w", domain.ErrInvalidInput)
	}

	hi, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching health_insurance_overview for delete: %w", err)
	}
	if hi.Status == domain.FilingStatusFiled {
		return domain.ErrFilingAlreadyFiled
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting health_insurance_overview: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "health_insurance", id, "delete", nil, nil)
	}
	return nil
}

// Recalculate recalculates the health insurance overview amounts from invoices and expenses.
func (s *HealthInsuranceService) Recalculate(ctx context.Context, id int64) (*domain.HealthInsuranceOverview, error) {
	if id == 0 {
		return nil, fmt.Errorf("health_insurance_overview ID is required: %w", domain.ErrInvalidInput)
	}

	hi, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching health_insurance_overview for recalculation: %w", err)
	}
	if hi.Status == domain.FilingStatusFiled {
		return nil, domain.ErrFilingAlreadyFiled
	}
	if err := s.applyRecalculatedAmounts(ctx, hi); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, hi); err != nil {
		return nil, fmt.Errorf("updating health_insurance_overview after recalculation: %w", err)
	}

	if s.audit != nil {
		s.audit.Log(ctx, "health_insurance", id, "update", nil, hi)
	}
	return hi, nil
}

// applyRecalculatedAmounts fills the calculated amounts of the overview in memory
// from its activity months, without persisting anything.
func (s *HealthInsuranceService) applyRecalculatedAmounts(ctx context.Context, hi *domain.HealthInsuranceOverview) error {
	if err := hi.ActivityMonths.Validate(); err != nil {
		return fmt.Errorf("validating stored activity months of health_insurance_overview %d: %w", hi.ID, err)
	}

	base, err := CalculateAnnualBase(ctx, s.invoiceRepo, s.expenseRepo, hi.Year)
	if err != nil {
		return fmt.Errorf("calculating annual base for health_insurance_overview: %w", err)
	}

	constants, err := calc.GetTaxConstants(hi.Year)
	if err != nil {
		return fmt.Errorf("getting tax constants for health insurance: %w", err)
	}

	if err := domain.ValidateHealthNoMinimum(hi.ActivityMonths, hi.NoMinimumMonths, hi.NoMinimumReason); err != nil {
		return fmt.Errorf("validating months without minimum of health_insurance_overview %d: %w", hi.ID, err)
	}

	// Determine flat-rate or actual expenses and the filing method.
	tys := &domain.TaxYearSettings{Year: hi.Year}
	if stored, err := s.taxYearSettingsRepo.GetByYear(ctx, hi.Year); err == nil {
		tys = stored
	} else if !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("fetching tax year settings for health insurance: %w", err)
	}
	flatRatePercent := tys.FlatRatePercent

	deadline, err := OverviewDeadline(hi.Year, tys.FilingMethod)
	if err != nil {
		return fmt.Errorf("computing overview deadline for health insurance: %w", err)
	}
	startMonth, err := HealthAdvanceStartMonth(hi.Year, deadline, hi.FiledAt)
	if err != nil {
		return fmt.Errorf("computing health advance start month: %w", err)
	}

	// The payment year is the year after the overview year; without its rules the
	// advance is not raised to a minimum.
	var warnings []string
	var bounds *calc.AdvanceBaseBounds
	payment, err := calc.GetTaxConstants(hi.Year + 1)
	var noRules *taxrules.NoRuleSetError
	switch {
	case errors.As(err, &noRules):
		warnings = append(warnings, domain.WarningAdvanceMinimumMissing)
	case err != nil:
		return fmt.Errorf("getting tax constants for advances paid after %d: %w", hi.Year, err)
	case hi.NoMinimumMonths != hi.ActivityMonths:
		bounds = &calc.AdvanceBaseBounds{MinMonthly: payment.HealthMinMonthly}
	}
	if tys.FilingMethod == domain.FilingMethodUnset {
		warnings = append([]string{domain.WarningFilingMethodMissing}, warnings...)
	}

	usedExpenses := calc.ResolveUsedExpenses(base.Revenue, base.Expenses, flatRatePercent, constants.FlatRateCaps)

	hi.TotalRevenue = base.Revenue
	hi.TotalExpenses = usedExpenses

	_, _, healthTotal, sumErr := s.taxPrepaymentRepo.SumByYear(ctx, hi.Year)
	if sumErr != nil {
		return fmt.Errorf("summing health prepayments: %w", sumErr)
	}

	// Pure calculation.
	insResult, err := calc.CalculateInsurance(calc.InsuranceInput{
		Revenue:        base.Revenue,
		UsedExpenses:   usedExpenses,
		MinMonthlyBase: constants.HealthMinMonthly,
		RatePermille:   constants.HealthRate,
		Prepayments:    healthTotal,
		RoundPremiumUp: true,
		ActivityMonths: hi.ActivityMonths.Count(),

		MinimumExemptMonths: hi.NoMinimumMonths.Count(),
		AdvanceBounds:       bounds,
	})
	if err != nil {
		return fmt.Errorf("calculating health insurance: %w", err)
	}

	hi.TaxBase = insResult.TaxBase
	hi.AssessmentBase = insResult.AssessmentBase
	hi.MinAssessmentBase = insResult.MinAssessmentBase
	hi.FinalAssessmentBase = insResult.FinalAssessmentBase
	hi.InsuranceRate = constants.HealthRate
	hi.TotalInsurance = insResult.TotalInsurance
	hi.Prepayments = healthTotal
	hi.Difference = insResult.Difference
	hi.NewMonthlyPrepay = insResult.NewMonthlyPrepay
	if hi.NoMinimumMonths == hi.ActivityMonths {
		// The minimum base applies in no month of activity, so advances are not
		// obligatory (§ 8 odst. 3 zakona c. 592/1992 Sb.).
		hi.NewMonthlyPrepay = 0
		warnings = append(warnings, domain.WarningHealthAdvanceNotRequired)
	}
	hi.FilingMethod = tys.FilingMethod
	hi.AdvanceStartMonth = startMonth
	hi.Warnings = warnings
	// A stored XML describes the previous amounts and months, so it is dropped
	// and has to be generated again.
	hi.XMLData = nil

	return nil
}

// SetActivityMonths stores the months of activity and recalculates the
// overview so the stored amounts match the months. The overview is written
// once, only after the recalculation succeeds.
func (s *HealthInsuranceService) SetActivityMonths(ctx context.Context, id int64, months domain.ActivityMonths) (*domain.HealthInsuranceOverview, error) {
	if id == 0 {
		return nil, fmt.Errorf("health_insurance_overview ID is required: %w", domain.ErrInvalidInput)
	}
	if err := months.Validate(); err != nil {
		return nil, fmt.Errorf("validating activity months: %w", err)
	}

	hi, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching health_insurance_overview for activity months: %w", err)
	}
	if hi.Status == domain.FilingStatusFiled {
		return nil, domain.ErrFilingAlreadyFiled
	}

	if err := domain.ValidateHealthNoMinimum(months, hi.NoMinimumMonths, hi.NoMinimumReason); err != nil {
		return nil, fmt.Errorf("validating activity months against months without minimum: %w", err)
	}

	hi.ActivityMonths = months
	if err := s.applyRecalculatedAmounts(ctx, hi); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, hi); err != nil {
		return nil, fmt.Errorf("updating health_insurance_overview after changing activity months: %w", err)
	}

	if s.audit != nil {
		s.audit.Log(ctx, "health_insurance", id, "update", nil, hi)
	}
	return hi, nil
}

// SetNoMinimum stores the months in which the minimum assessment base does not
// apply, with their reason, and recalculates the overview. No months and no
// reason clear the setting. The overview is written once, only after the
// recalculation succeeds.
func (s *HealthInsuranceService) SetNoMinimum(ctx context.Context, id int64, months domain.ActivityMonths, reason domain.HealthNoMinimumReason) (*domain.HealthInsuranceOverview, error) {
	if id == 0 {
		return nil, fmt.Errorf("health_insurance_overview ID is required: %w", domain.ErrInvalidInput)
	}

	hi, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching health_insurance_overview for months without minimum: %w", err)
	}
	if hi.Status == domain.FilingStatusFiled {
		return nil, domain.ErrFilingAlreadyFiled
	}
	if err := domain.ValidateHealthNoMinimum(hi.ActivityMonths, months, reason); err != nil {
		return nil, fmt.Errorf("validating months without minimum: %w", err)
	}

	hi.NoMinimumMonths = months
	hi.NoMinimumReason = reason
	if err := s.applyRecalculatedAmounts(ctx, hi); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, hi); err != nil {
		return nil, fmt.Errorf("updating health_insurance_overview after changing months without minimum: %w", err)
	}

	if s.audit != nil {
		s.audit.Log(ctx, "health_insurance", id, "update", nil, hi)
	}
	return hi, nil
}

// GenerateXML generates the XML for a health insurance overview.
// Health insurance currently has no standardized XML format.
func (s *HealthInsuranceService) GenerateXML(ctx context.Context, id int64) (*domain.HealthInsuranceOverview, error) {
	if id == 0 {
		return nil, fmt.Errorf("health_insurance_overview ID is required: %w", domain.ErrInvalidInput)
	}

	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching health_insurance_overview for XML generation: %w", err)
	}

	return nil, fmt.Errorf("ZP XML generation not yet available (awaiting SZP-VZP format): %w", domain.ErrInvalidInput)
}

// MarkFiled marks a health insurance overview as filed and records the timestamp.
func (s *HealthInsuranceService) MarkFiled(ctx context.Context, id int64) (*domain.HealthInsuranceOverview, error) {
	if id == 0 {
		return nil, fmt.Errorf("health_insurance_overview ID is required: %w", domain.ErrInvalidInput)
	}

	hi, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching health_insurance_overview for marking as filed: %w", err)
	}
	if hi.Status == domain.FilingStatusFiled {
		return nil, domain.ErrFilingAlreadyFiled
	}

	now := s.now()
	hi.Status = domain.FilingStatusFiled
	hi.FiledAt = &now

	// An earlier filing date moves the advance start month; amounts stay as calculated.
	if hi.AdvanceStartMonth != 0 {
		deadline, err := OverviewDeadline(hi.Year, hi.FilingMethod)
		if err != nil {
			return nil, fmt.Errorf("computing overview deadline for health insurance: %w", err)
		}
		hi.AdvanceStartMonth, err = HealthAdvanceStartMonth(hi.Year, deadline, hi.FiledAt)
		if err != nil {
			return nil, fmt.Errorf("computing health advance start month: %w", err)
		}
	}

	if err := s.repo.Update(ctx, hi); err != nil {
		return nil, fmt.Errorf("marking health_insurance_overview as filed: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "health_insurance", id, "mark_filed", nil, hi)
	}
	return hi, nil
}
