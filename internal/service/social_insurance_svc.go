package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/zajca/zfaktury/internal/annualtaxxml"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// SocialInsuranceService provides business logic for social insurance overview management.
type SocialInsuranceService struct {
	repo                repository.SocialInsuranceOverviewRepo
	invoiceRepo         repository.InvoiceRepo
	expenseRepo         repository.ExpenseRepo
	settingsRepo        repository.SettingsRepo
	taxYearSettingsRepo repository.TaxYearSettingsRepo
	taxPrepaymentRepo   repository.TaxPrepaymentRepo
}

// NewSocialInsuranceService creates a new SocialInsuranceService.
func NewSocialInsuranceService(
	repo repository.SocialInsuranceOverviewRepo,
	invoiceRepo repository.InvoiceRepo,
	expenseRepo repository.ExpenseRepo,
	settingsRepo repository.SettingsRepo,
	taxYearSettingsRepo repository.TaxYearSettingsRepo,
	taxPrepaymentRepo repository.TaxPrepaymentRepo,
) *SocialInsuranceService {
	return &SocialInsuranceService{
		repo:                repo,
		invoiceRepo:         invoiceRepo,
		expenseRepo:         expenseRepo,
		settingsRepo:        settingsRepo,
		taxYearSettingsRepo: taxYearSettingsRepo,
		taxPrepaymentRepo:   taxPrepaymentRepo,
	}
}

// Create validates and persists a new social insurance overview.
func (s *SocialInsuranceService) Create(ctx context.Context, sio *domain.SocialInsuranceOverview) error {
	if sio.Year < 2000 || sio.Year > 2100 {
		return fmt.Errorf("year out of valid range: %w", domain.ErrInvalidInput)
	}

	if sio.FilingType == "" {
		sio.FilingType = domain.FilingTypeRegular
	}
	switch sio.FilingType {
	case domain.FilingTypeRegular, domain.FilingTypeCorrective, domain.FilingTypeSupplementary:
		// ok
	default:
		return fmt.Errorf("invalid filing_type: %w", domain.ErrInvalidInput)
	}

	// Check for existing regular filing for this year.
	if sio.FilingType == domain.FilingTypeRegular {
		existing, err := s.repo.GetByYear(ctx, sio.Year, sio.FilingType)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("checking existing social_insurance_overview: %w", err)
		}
		if existing != nil {
			return domain.ErrFilingAlreadyExists
		}
	}

	if sio.Status == "" {
		sio.Status = domain.FilingStatusDraft
	}

	if err := s.repo.Create(ctx, sio); err != nil {
		return fmt.Errorf("creating social_insurance_overview: %w", err)
	}
	return nil
}

// GetByID retrieves a social insurance overview by its ID.
func (s *SocialInsuranceService) GetByID(ctx context.Context, id int64) (*domain.SocialInsuranceOverview, error) {
	if id == 0 {
		return nil, fmt.Errorf("social_insurance_overview ID is required: %w", domain.ErrInvalidInput)
	}
	sio, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching social_insurance_overview: %w", err)
	}
	return sio, nil
}

// List retrieves all social insurance overviews for a given year.
func (s *SocialInsuranceService) List(ctx context.Context, year int) ([]domain.SocialInsuranceOverview, error) {
	if year == 0 {
		year = time.Now().Year()
	}
	overviews, err := s.repo.List(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("listing social_insurance_overviews: %w", err)
	}
	return overviews, nil
}

// Delete removes a social insurance overview by ID. Filed overviews cannot be deleted.
func (s *SocialInsuranceService) Delete(ctx context.Context, id int64) error {
	if id == 0 {
		return fmt.Errorf("social_insurance_overview ID is required: %w", domain.ErrInvalidInput)
	}

	sio, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching social_insurance_overview for delete: %w", err)
	}
	if sio.Status == domain.FilingStatusFiled {
		return domain.ErrFilingAlreadyFiled
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting social_insurance_overview: %w", err)
	}
	return nil
}

// Recalculate recalculates the social insurance overview from invoices and expenses.
func (s *SocialInsuranceService) Recalculate(ctx context.Context, id int64) (*domain.SocialInsuranceOverview, error) {
	if id == 0 {
		return nil, fmt.Errorf("social_insurance_overview ID is required: %w", domain.ErrInvalidInput)
	}

	sio, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching social_insurance_overview for recalculation: %w", err)
	}
	if sio.Status == domain.FilingStatusFiled {
		return nil, domain.ErrFilingAlreadyFiled
	}

	// Calculate annual base from invoices and expenses.
	annualBase, err := CalculateAnnualBase(ctx, s.invoiceRepo, s.expenseRepo, sio.Year)
	if err != nil {
		return nil, fmt.Errorf("calculating annual base for social insurance: %w", err)
	}

	// Get flat_rate_percent from tax year settings.
	flatRatePercent := 0
	tys, err := s.taxYearSettingsRepo.GetByYear(ctx, sio.Year)
	if err == nil {
		flatRatePercent = tys.FlatRatePercent
	}

	// Get tax constants for the year.
	constants, err := GetTaxConstants(sio.Year)
	if err != nil {
		return nil, fmt.Errorf("getting tax constants for social insurance: %w", err)
	}

	// Compute used expenses (flat-rate or actual).
	revenue := annualBase.Revenue
	usedExpenses := annualBase.Expenses

	if flatRatePercent > 0 {
		flatRateAmount := revenue.Multiply(float64(flatRatePercent) / 100.0)
		// Apply flat-rate cap if defined.
		if cap, ok := constants.FlatRateCaps[flatRatePercent]; ok {
			if flatRateAmount > cap {
				flatRateAmount = cap
			}
		}
		usedExpenses = flatRateAmount
	}

	sio.TotalRevenue = revenue
	sio.TotalExpenses = usedExpenses

	// taxBase = revenue - usedExpenses (clamp to 0).
	taxBase := int64(revenue) - int64(usedExpenses)
	if taxBase < 0 {
		taxBase = 0
	}
	sio.TaxBase = domain.Amount(taxBase)

	// assessmentBase = taxBase / 2 (integer division, in halere).
	assessmentBase := taxBase / 2
	sio.AssessmentBase = domain.Amount(assessmentBase)

	// minBase = constants.SocialMinMonthly * 12.
	minBase := int64(constants.SocialMinMonthly) * 12
	sio.MinAssessmentBase = domain.Amount(minBase)

	// finalBase = max(assessmentBase, minBase).
	finalBase := assessmentBase
	if minBase > finalBase {
		finalBase = minBase
	}
	sio.FinalAssessmentBase = domain.Amount(finalBase)

	// totalInsurance = finalBase * SocialRate / 1000.
	sio.InsuranceRate = constants.SocialRate
	totalInsurance := finalBase * int64(constants.SocialRate) / 1000
	sio.TotalInsurance = domain.Amount(totalInsurance)

	// Read prepayments from tax prepayments table.
	_, socialTotal, _, sumErr := s.taxPrepaymentRepo.SumByYear(ctx, sio.Year)
	if sumErr != nil {
		return nil, fmt.Errorf("summing social prepayments: %w", sumErr)
	}
	sio.Prepayments = socialTotal

	// difference = totalInsurance - prepayments.
	sio.Difference = domain.Amount(totalInsurance - int64(socialTotal))

	// newMonthlyPrepay = ceil(totalInsurance / 12), rounded up to whole CZK (100 halere).
	monthlyHalere := totalInsurance / 12
	if totalInsurance%12 != 0 {
		monthlyHalere++
	}
	// Round up to nearest 100 halere (1 CZK).
	roundedUpCZK := ((monthlyHalere + 99) / 100) * 100
	sio.NewMonthlyPrepay = domain.Amount(roundedUpCZK)

	// Persist updated values.
	if err := s.repo.Update(ctx, sio); err != nil {
		return nil, fmt.Errorf("updating social_insurance_overview after recalculation: %w", err)
	}

	return sio, nil
}

// GenerateXML generates the CSSZ XML for a social insurance overview.
func (s *SocialInsuranceService) GenerateXML(ctx context.Context, id int64) (*domain.SocialInsuranceOverview, error) {
	if id == 0 {
		return nil, fmt.Errorf("social_insurance_overview ID is required: %w", domain.ErrInvalidInput)
	}

	sio, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching social_insurance_overview for XML generation: %w", err)
	}

	// Load settings for taxpayer info.
	settings := make(map[string]string)
	for _, key := range []string{
		"cssz_code", "taxpayer_first_name", "taxpayer_last_name",
		"taxpayer_birth_number", "taxpayer_birth_date", "taxpayer_street",
		"taxpayer_house_number", "taxpayer_postal_code", "taxpayer_city",
	} {
		val, err := s.settingsRepo.Get(ctx, key)
		if err == nil {
			settings[key] = val
		}
	}
	// Read flat_rate_percent from tax year settings for XML generation.
	tys, tysErr := s.taxYearSettingsRepo.GetByYear(ctx, sio.Year)
	if tysErr == nil && tys.FlatRatePercent > 0 {
		settings["flat_rate_percent"] = strconv.Itoa(tys.FlatRatePercent)
		settings["flat_rate_expenses"] = "true"
	}

	xmlData, err := annualtaxxml.GenerateSocialInsuranceXML(sio, settings)
	if err != nil {
		return nil, fmt.Errorf("generating social_insurance_overview XML: %w", err)
	}

	sio.XMLData = xmlData
	if err := s.repo.Update(ctx, sio); err != nil {
		return nil, fmt.Errorf("saving social_insurance_overview XML: %w", err)
	}

	return sio, nil
}

// GetXMLData retrieves the stored XML data for a social insurance overview.
func (s *SocialInsuranceService) GetXMLData(ctx context.Context, id int64) ([]byte, error) {
	if id == 0 {
		return nil, fmt.Errorf("social_insurance_overview ID is required: %w", domain.ErrInvalidInput)
	}
	sio, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching social_insurance_overview for XML data: %w", err)
	}
	return sio.XMLData, nil
}

// MarkFiled marks a social insurance overview as filed and records the timestamp.
func (s *SocialInsuranceService) MarkFiled(ctx context.Context, id int64) (*domain.SocialInsuranceOverview, error) {
	if id == 0 {
		return nil, fmt.Errorf("social_insurance_overview ID is required: %w", domain.ErrInvalidInput)
	}

	sio, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching social_insurance_overview for marking as filed: %w", err)
	}
	if sio.Status == domain.FilingStatusFiled {
		return nil, domain.ErrFilingAlreadyFiled
	}

	now := time.Now()
	sio.Status = domain.FilingStatusFiled
	sio.FiledAt = &now

	if err := s.repo.Update(ctx, sio); err != nil {
		return nil, fmt.Errorf("marking social_insurance_overview as filed: %w", err)
	}
	return sio, nil
}
