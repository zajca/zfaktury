package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/zajca/zfaktury/internal/annualtaxxml"
	"github.com/zajca/zfaktury/internal/calc"
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
	audit               *AuditService
}

// NewSocialInsuranceService creates a new SocialInsuranceService.
func NewSocialInsuranceService(
	repo repository.SocialInsuranceOverviewRepo,
	invoiceRepo repository.InvoiceRepo,
	expenseRepo repository.ExpenseRepo,
	settingsRepo repository.SettingsRepo,
	taxYearSettingsRepo repository.TaxYearSettingsRepo,
	taxPrepaymentRepo repository.TaxPrepaymentRepo,
	audit *AuditService,
) *SocialInsuranceService {
	return &SocialInsuranceService{
		repo:                repo,
		invoiceRepo:         invoiceRepo,
		expenseRepo:         expenseRepo,
		settingsRepo:        settingsRepo,
		taxYearSettingsRepo: taxYearSettingsRepo,
		taxPrepaymentRepo:   taxPrepaymentRepo,
		audit:               audit,
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
	if s.audit != nil {
		s.audit.Log(ctx, "social_insurance", sio.ID, "create", nil, sio)
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
	if s.audit != nil {
		s.audit.Log(ctx, "social_insurance", id, "delete", nil, nil)
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
	constants, err := calc.GetTaxConstants(sio.Year)
	if err != nil {
		return nil, fmt.Errorf("getting tax constants for social insurance: %w", err)
	}

	// Resolve used expenses (flat-rate or actual).
	usedExpenses := calc.ResolveUsedExpenses(annualBase.Revenue, annualBase.Expenses, flatRatePercent, constants.FlatRateCaps)

	sio.TotalRevenue = annualBase.Revenue
	sio.TotalExpenses = usedExpenses

	// Read prepayments from tax prepayments table.
	_, socialTotal, _, sumErr := s.taxPrepaymentRepo.SumByYear(ctx, sio.Year)
	if sumErr != nil {
		return nil, fmt.Errorf("summing social prepayments: %w", sumErr)
	}

	// Pure calculation.
	insResult := calc.CalculateInsurance(calc.InsuranceInput{
		Revenue:        annualBase.Revenue,
		UsedExpenses:   usedExpenses,
		MinMonthlyBase: constants.SocialMinMonthly,
		RatePermille:   constants.SocialRate,
		Prepayments:    socialTotal,
	})

	sio.TaxBase = insResult.TaxBase
	sio.AssessmentBase = insResult.AssessmentBase
	sio.MinAssessmentBase = insResult.MinAssessmentBase
	sio.FinalAssessmentBase = insResult.FinalAssessmentBase
	sio.InsuranceRate = constants.SocialRate
	sio.TotalInsurance = insResult.TotalInsurance
	sio.Prepayments = socialTotal
	sio.Difference = insResult.Difference
	sio.NewMonthlyPrepay = insResult.NewMonthlyPrepay

	// Persist updated values.
	if err := s.repo.Update(ctx, sio); err != nil {
		return nil, fmt.Errorf("updating social_insurance_overview after recalculation: %w", err)
	}

	if s.audit != nil {
		s.audit.Log(ctx, "social_insurance", id, "update", nil, sio)
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

	if s.audit != nil {
		s.audit.Log(ctx, "social_insurance", id, "generate_xml", nil, sio)
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
	if s.audit != nil {
		s.audit.Log(ctx, "social_insurance", id, "mark_filed", nil, sio)
	}
	return sio, nil
}
