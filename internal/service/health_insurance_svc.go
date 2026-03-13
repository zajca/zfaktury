package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
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

	base, err := CalculateAnnualBase(ctx, s.invoiceRepo, s.expenseRepo, hi.Year)
	if err != nil {
		return nil, fmt.Errorf("calculating annual base for health_insurance_overview: %w", err)
	}

	constants, err := GetTaxConstants(hi.Year)
	if err != nil {
		return nil, fmt.Errorf("getting tax constants for health insurance: %w", err)
	}

	// Determine flat-rate or actual expenses.
	flatRatePercent := 0
	tys, err := s.taxYearSettingsRepo.GetByYear(ctx, hi.Year)
	if err == nil {
		flatRatePercent = tys.FlatRatePercent
	}

	revenue := base.Revenue
	usedExpenses := base.Expenses

	if flatRatePercent > 0 {
		flatRateAmount := revenue.Multiply(float64(flatRatePercent) / 100.0)
		if cap, ok := constants.FlatRateCaps[flatRatePercent]; ok {
			if flatRateAmount > cap {
				flatRateAmount = cap
			}
		}
		usedExpenses = flatRateAmount
	}

	hi.TotalRevenue = revenue
	hi.TotalExpenses = usedExpenses

	taxBase := int64(revenue) - int64(usedExpenses)
	if taxBase < 0 {
		taxBase = 0
	}
	hi.TaxBase = domain.Amount(taxBase)

	assessmentBase := taxBase / 2
	hi.AssessmentBase = domain.Amount(assessmentBase)

	minBase := int64(constants.HealthMinMonthly) * 12
	hi.MinAssessmentBase = domain.Amount(minBase)

	finalBase := assessmentBase
	if minBase > finalBase {
		finalBase = minBase
	}
	hi.FinalAssessmentBase = domain.Amount(finalBase)

	hi.InsuranceRate = constants.HealthRate
	totalInsurance := finalBase * int64(constants.HealthRate) / 1000
	hi.TotalInsurance = domain.Amount(totalInsurance)

	_, _, healthTotal, sumErr := s.taxPrepaymentRepo.SumByYear(ctx, hi.Year)
	if sumErr != nil {
		return nil, fmt.Errorf("summing health prepayments: %w", sumErr)
	}
	hi.Prepayments = healthTotal

	hi.Difference = domain.Amount(totalInsurance - int64(healthTotal))

	// New monthly prepayment = totalInsurance / 12, rounded up to whole CZK.
	monthlyRaw := totalInsurance / 12
	if totalInsurance%12 != 0 {
		monthlyRaw++
	}
	newMonthly := ((monthlyRaw + 99) / 100) * 100
	hi.NewMonthlyPrepay = domain.Amount(newMonthly)

	if err := s.repo.Update(ctx, hi); err != nil {
		return nil, fmt.Errorf("updating health_insurance_overview after recalculation: %w", err)
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

// GetXMLData retrieves the stored XML data for a health insurance overview.
func (s *HealthInsuranceService) GetXMLData(ctx context.Context, id int64) ([]byte, error) {
	if id == 0 {
		return nil, fmt.Errorf("health_insurance_overview ID is required: %w", domain.ErrInvalidInput)
	}
	hi, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching health_insurance_overview for XML data: %w", err)
	}
	return hi.XMLData, nil
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

	now := time.Now()
	hi.Status = domain.FilingStatusFiled
	hi.FiledAt = &now

	if err := s.repo.Update(ctx, hi); err != nil {
		return nil, fmt.Errorf("marking health_insurance_overview as filed: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "health_insurance", id, "mark_filed", nil, hi)
	}
	return hi, nil
}
