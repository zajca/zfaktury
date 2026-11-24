package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/zajca/zfaktury/internal/annualtaxxml"
	"github.com/zajca/zfaktury/internal/calc"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/taxrules"
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
	now                 func() time.Time
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
		now:                 time.Now,
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

	if err := sio.ActivityMonths.Validate(); err != nil {
		return fmt.Errorf("validating activity months: %w", err)
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
	if err := s.applyRecalculatedAmounts(ctx, sio); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, sio); err != nil {
		return nil, fmt.Errorf("updating social_insurance_overview after recalculation: %w", err)
	}

	if s.audit != nil {
		s.audit.Log(ctx, "social_insurance", id, "update", nil, sio)
	}
	return sio, nil
}

// paymentYearConstants returns the rule set of the year after the overview
// year, in which the new advance is paid, or nil when that year has no rule set.
func (s *SocialInsuranceService) paymentYearConstants(year int) (*calc.TaxYearConstants, error) {
	constants, err := calc.GetTaxConstants(year + 1)
	if err != nil {
		var noRules *taxrules.NoRuleSetError
		if errors.As(err, &noRules) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting tax constants for advances paid after %d: %w", year, err)
	}
	return &constants, nil
}

// taxYearSettingsFor returns the settings stored for the tax year; a year
// without settings gets empty ones (flat rate 0, filing method unset).
func (s *SocialInsuranceService) taxYearSettingsFor(ctx context.Context, year int) (*domain.TaxYearSettings, error) {
	tys, err := s.taxYearSettingsRepo.GetByYear(ctx, year)
	if errors.Is(err, domain.ErrNotFound) {
		return &domain.TaxYearSettings{Year: year}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fetching tax year settings for social insurance: %w", err)
	}
	return tys, nil
}

// applyRecalculatedAmounts fills the calculated amounts of the overview in memory
// from its activity months, without persisting anything.
func (s *SocialInsuranceService) applyRecalculatedAmounts(ctx context.Context, sio *domain.SocialInsuranceOverview) error {
	if err := sio.ActivityMonths.Validate(); err != nil {
		return fmt.Errorf("validating stored activity months of social_insurance_overview %d: %w", sio.ID, err)
	}
	if err := domain.ValidateSecondaryActivity(sio.ActivityMonths, sio.SecondaryMonths, sio.SecondaryReason); err != nil {
		return fmt.Errorf("validating secondary activity of social_insurance_overview %d: %w", sio.ID, err)
	}

	// Calculate annual base from invoices and expenses.
	annualBase, err := CalculateAnnualBase(ctx, s.invoiceRepo, s.expenseRepo, sio.Year)
	if err != nil {
		return fmt.Errorf("calculating annual base for social insurance: %w", err)
	}

	// Get flat_rate_percent from tax year settings.
	tys, err := s.taxYearSettingsFor(ctx, sio.Year)
	if err != nil {
		return err
	}
	flatRatePercent := tys.FlatRatePercent
	deadline, err := OverviewDeadline(sio.Year, tys.FilingMethod)
	if err != nil {
		return fmt.Errorf("computing overview deadline for social insurance: %w", err)
	}
	startMonth, err := SocialAdvanceStartMonth(sio.Year, deadline, sio.FiledAt)
	if err != nil {
		return fmt.Errorf("computing social advance start month: %w", err)
	}

	// Get tax constants for the year.
	constants, err := calc.GetTaxConstants(sio.Year)
	if err != nil {
		return fmt.Errorf("getting tax constants for social insurance: %w", err)
	}

	// Resolve used expenses (flat-rate or actual).
	usedExpenses := calc.ResolveUsedExpenses(annualBase.Revenue, annualBase.Expenses, flatRatePercent, constants.FlatRateCaps)

	sio.TotalRevenue = annualBase.Revenue
	sio.TotalExpenses = usedExpenses

	// Read prepayments from tax prepayments table.
	_, socialTotal, _, sumErr := s.taxPrepaymentRepo.SumByYear(ctx, sio.Year)
	if sumErr != nil {
		return fmt.Errorf("summing social prepayments: %w", sumErr)
	}

	// Pure calculation.
	payment, err := s.paymentYearConstants(sio.Year)
	if err != nil {
		return err
	}
	mainMonths := (sio.ActivityMonths &^ sio.SecondaryMonths).Count()
	secondaryMonths := sio.SecondaryMonths.Count()
	taxBase := max(annualBase.Revenue-usedExpenses, 0)
	res, err := calc.CalculateCSSZOverview(calc.CSSZOverviewInput{
		TaxBase:                   taxBase,
		MainMonths:                mainMonths,
		SecondaryMonths:           secondaryMonths,
		MainMinMonthly:            constants.SocialMinMonthly,
		SecondaryMinMonthly:       constants.SocialSecondaryMinMonthly,
		MaxAnnualBase:             constants.SocialMaxAnnualBase,
		DecisiveAmount:            constants.SocialDecisiveAmount,
		DecisiveAmountMonthlyStep: constants.SocialDecisiveAmountMonthlyStep,
		EmploymentBase:            sio.EmploymentAssessmentBase,
		RatePermille:              constants.SocialRate,
		Prepayments:               socialTotal,
	})
	if err != nil {
		return fmt.Errorf("calculating social insurance: %w", err)
	}

	periods, err := calc.SocialAdvancePeriods(res.AdvanceComputedBase, res.FinalAssessmentBase, mainMonths+secondaryMonths, startMonth, payment, constants.SocialRate, sio.AdvanceAsSecondary())
	if err != nil {
		return fmt.Errorf("calculating social advance periods: %w", err)
	}
	var warnings []string
	if tys.FilingMethod == domain.FilingMethodUnset {
		warnings = append(warnings, domain.WarningFilingMethodMissing)
	}
	if payment == nil {
		warnings = append(warnings, domain.WarningAdvanceMinimumMissing)
	}
	if mainMonths == 0 && !res.SecondaryParticipates {
		// A secondary activity below the decisive amount is not insured, so no
		// advance is due.
		periods = []domain.AdvancePeriod{{FromMonth: startMonth}}
		warnings = append(warnings, domain.WarningSocialAdvanceNotRequired)
	}

	sio.FilingMethod = tys.FilingMethod
	sio.AdvanceStartMonth = startMonth
	sio.AdvancePeriods = periods
	sio.Warnings = warnings
	sio.TaxBase = taxBase
	sio.AssessmentBase = res.MainAssessmentBase
	sio.MinAssessmentBase = res.MinAssessmentBase
	sio.FinalAssessmentBase = res.FinalAssessmentBase
	sio.MainTaxBaseShare = res.MainTaxBaseShare
	sio.SecondaryTaxBaseShare = res.SecondaryTaxBaseShare
	sio.SecondaryAssessmentBase = res.SecondaryAssessmentBase
	sio.MainPartialBase = res.MainPartialBase
	sio.SecondaryPartialBase = res.SecondaryPartialBase
	sio.SecondaryDecisiveAmount = res.SecondaryDecisiveAmount
	sio.ReportedEmploymentBase = res.ReportedEmploymentBase
	sio.SumBase = res.SumBase
	sio.InsuredBase = res.InsuredBase
	sio.InsuranceRate = constants.SocialRate
	sio.TotalInsurance = res.TotalInsurance
	sio.Prepayments = socialTotal
	sio.Difference = res.Difference
	sio.NewMonthlyPrepay = periods[0].MonthlyPrepay
	newMonthlyBase := periods[0].MonthlyBase
	// The base is stored with the amounts it produced, so a later export reports
	// ř. 35 and ř. 36 from the same calculation.
	sio.NewMonthlyBase = &newMonthlyBase
	// A stored XML describes the previous amounts and months, so it is dropped
	// and has to be generated again.
	sio.XMLData = nil

	return nil
}

// SetActivityMonths stores the months of activity and recalculates the
// overview so the stored amounts match the months. The overview is written
// once, only after the recalculation succeeds.
func (s *SocialInsuranceService) SetActivityMonths(ctx context.Context, id int64, months domain.ActivityMonths) (*domain.SocialInsuranceOverview, error) {
	if id == 0 {
		return nil, fmt.Errorf("social_insurance_overview ID is required: %w", domain.ErrInvalidInput)
	}
	if err := months.Validate(); err != nil {
		return nil, fmt.Errorf("validating activity months: %w", err)
	}

	sio, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching social_insurance_overview for activity months: %w", err)
	}
	if sio.Status == domain.FilingStatusFiled {
		return nil, domain.ErrFilingAlreadyFiled
	}

	sio.ActivityMonths = months
	if err := s.applyRecalculatedAmounts(ctx, sio); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, sio); err != nil {
		return nil, fmt.Errorf("updating social_insurance_overview after changing activity months: %w", err)
	}

	if s.audit != nil {
		s.audit.Log(ctx, "social_insurance", id, "update", nil, sio)
	}
	return sio, nil
}

// SetSecondaryActivity stores the months of secondary activity with their
// reason and recalculates the overview. Empty months with an empty reason clear
// the secondary activity. The overview is written once, only after the
// recalculation succeeds.
func (s *SocialInsuranceService) SetSecondaryActivity(ctx context.Context, id int64, months domain.ActivityMonths, reason domain.SecondaryActivityReason) (*domain.SocialInsuranceOverview, error) {
	return s.changeAndRecalculate(ctx, id, "secondary activity", func(sio *domain.SocialInsuranceOverview) error {
		if err := domain.ValidateSecondaryActivity(sio.ActivityMonths, months, reason); err != nil {
			return fmt.Errorf("validating secondary activity: %w", err)
		}
		sio.SecondaryMonths = months
		sio.SecondaryReason = reason
		return nil
	})
}

// SetEmploymentBase stores the employment assessment base entered by the user
// (nil removes it) and recalculates the overview.
func (s *SocialInsuranceService) SetEmploymentBase(ctx context.Context, id int64, base *domain.Amount) (*domain.SocialInsuranceOverview, error) {
	return s.changeAndRecalculate(ctx, id, "employment base", func(sio *domain.SocialInsuranceOverview) error {
		if base != nil && (*base < 0 || *base%domain.NewAmount(1, 0) != 0) {
			return fmt.Errorf("employment assessment base %d must be whole CZK and not negative: %w", *base, domain.ErrInvalidInput)
		}
		if base == nil {
			sio.EmploymentAssessmentBase = nil
			return nil
		}
		value := *base
		sio.EmploymentAssessmentBase = &value
		return nil
	})
}

// changeAndRecalculate loads an unfiled overview, applies change, recalculates
// and writes it once.
func (s *SocialInsuranceService) changeAndRecalculate(ctx context.Context, id int64, what string, change func(*domain.SocialInsuranceOverview) error) (*domain.SocialInsuranceOverview, error) {
	if id == 0 {
		return nil, fmt.Errorf("social_insurance_overview ID is required: %w", domain.ErrInvalidInput)
	}
	sio, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching social_insurance_overview for %s: %w", what, err)
	}
	if sio.Status == domain.FilingStatusFiled {
		return nil, domain.ErrFilingAlreadyFiled
	}
	if err := change(sio); err != nil {
		return nil, err
	}
	if err := s.applyRecalculatedAmounts(ctx, sio); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, sio); err != nil {
		return nil, fmt.Errorf("updating social_insurance_overview after changing %s: %w", what, err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "social_insurance", id, "update", nil, sio)
	}
	return sio, nil
}

// Messages GenerateXML shows to the user when the stored calculation cannot be
// written to the XML.
const (
	msgOverviewNotCalculated      = "přehled je potřeba nejdřív přepočítat"
	msgOverviewNotCalculatedFiled = "přehled je už podaný a nemá uložený začátek zálohy, uložené XML zůstává beze změny"
	msgFilingMethodChanged        = "způsob podání daňového přiznání se od výpočtu změnil, přepočítejte přehled"
	msgFilingMethodChangedFiled   = "přehled je už podaný a způsob podání daňového přiznání se od výpočtu změnil, uložené XML zůstává beze změny"
)

// GenerateXML generates the CSSZ XML for a social insurance overview.
func (s *SocialInsuranceService) GenerateXML(ctx context.Context, id int64) (*domain.SocialInsuranceOverview, error) {
	if id == 0 {
		return nil, fmt.Errorf("social_insurance_overview ID is required: %w", domain.ErrInvalidInput)
	}

	sio, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching social_insurance_overview for XML generation: %w", err)
	}
	if sio.AdvanceStartMonth == 0 || sio.NewMonthlyBase == nil {
		if sio.Status == domain.FilingStatusFiled || sio.FiledAt != nil {
			return nil, &domain.UserMessageError{Message: msgOverviewNotCalculatedFiled}
		}
		return nil, &domain.UserMessageError{Message: msgOverviewNotCalculated}
	}

	// Load settings for taxpayer info.
	// The XML generator expects taxpayer_* prefixed keys, but the actual settings
	// store uses unprefixed keys (first_name, last_name, street, ...). Translate here.
	settings := make(map[string]string)
	settingKeyMap := map[string]string{
		"first_name":   "taxpayer_first_name",
		"last_name":    "taxpayer_last_name",
		"title":        "taxpayer_title",
		"birth_number": "taxpayer_birth_number",
		"birth_date":   "taxpayer_birth_date",
		"street":       "taxpayer_street",
		"house_number": "taxpayer_house_number",
		"city":         "taxpayer_city",
		"zip":          "taxpayer_postal_code",
		"email":        "taxpayer_email",
		"phone":        "taxpayer_phone",
		"databox_id":   "taxpayer_databox_id",
	}
	for srcKey, dstKey := range settingKeyMap {
		if val, err := s.settingsRepo.Get(ctx, srcKey); err == nil {
			settings[dstKey] = val
		}
	}
	if val, err := s.settingsRepo.Get(ctx, "cssz_code"); err == nil {
		settings["cssz_code"] = val
	}
	if val, err := s.settingsRepo.Get(ctx, "cssz_variable_symbol"); err == nil {
		settings["cssz_variable_symbol"] = val
	}
	// Read flat_rate_percent from tax year settings for XML generation.
	tys, err := s.taxYearSettingsFor(ctx, sio.Year)
	if err != nil {
		return nil, err
	}
	if tys.FlatRatePercent > 0 {
		settings["flat_rate_percent"] = strconv.Itoa(tys.FlatRatePercent)
		settings["flat_rate_expenses"] = "true"
	}
	if tys.FilingMethod != sio.FilingMethod {
		if sio.Status == domain.FilingStatusFiled || sio.FiledAt != nil {
			return nil, &domain.UserMessageError{Message: msgFilingMethodChangedFiled}
		}
		return nil, &domain.UserMessageError{Message: msgFilingMethodChanged}
	}

	advanceBase := *sio.NewMonthlyBase
	if want := calc.SocialAdvanceFromMonthlyBase(advanceBase, sio.InsuranceRate); want != sio.NewMonthlyPrepay {
		if sio.Status == domain.FilingStatusFiled || sio.FiledAt != nil {
			return nil, &domain.UserMessageError{Message: "uložená záloha neodpovídá měsíčnímu základu a přehled je už podaný, uložené XML zůstává beze změny"}
		}
		return nil, &domain.UserMessageError{Message: "uložená záloha neodpovídá měsíčnímu základu, přepočítejte přehled"}
	}
	xmlData, err := annualtaxxml.GenerateSocialInsuranceXML(sio, settings, advanceBase)
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

// moveAdvanceStart recomputes the start month and the advance periods from the
// filing date, which moves the start earlier when the overview was filed before
// its deadline. The advance reported in the XML stays as filed; a warning says
// when the first period then differs from it.
func (s *SocialInsuranceService) moveAdvanceStart(sio *domain.SocialInsuranceOverview) error {
	deadline, err := OverviewDeadline(sio.Year, sio.FilingMethod)
	if err != nil {
		return fmt.Errorf("computing overview deadline for social insurance: %w", err)
	}
	start, err := SocialAdvanceStartMonth(sio.Year, deadline, sio.FiledAt)
	if err != nil {
		return fmt.Errorf("computing social advance start month: %w", err)
	}
	if start == sio.AdvanceStartMonth {
		return nil
	}
	payment, err := s.paymentYearConstants(sio.Year)
	if err != nil {
		return err
	}
	periods, err := calc.SocialAdvancePeriods(calc.SocialAdvanceComputedBase(sio.TaxBase), sio.FinalAssessmentBase, sio.ActivityMonths.Count(), start, payment, sio.InsuranceRate, sio.AdvanceAsSecondary())
	if err != nil {
		return fmt.Errorf("calculating social advance periods: %w", err)
	}
	if sio.ActivityMonths&^sio.SecondaryMonths == 0 && sio.SecondaryAssessmentBase == 0 {
		periods = []domain.AdvancePeriod{{FromMonth: start}}
	}
	sio.AdvanceStartMonth = start
	sio.AdvancePeriods = periods
	if sio.NewMonthlyBase != nil && periods[0].MonthlyBase != *sio.NewMonthlyBase && !slices.Contains(sio.Warnings, domain.WarningAdvanceDiffersFromXML) {
		sio.Warnings = append(sio.Warnings, domain.WarningAdvanceDiffersFromXML)
	}
	return nil
}

// MarkFiled marks a social insurance overview as filed and records the timestamp.
func (s *SocialInsuranceService) MarkFiled(ctx context.Context, id int64) (*domain.SocialInsuranceOverview, error) {
	return s.MarkFiledAt(ctx, id, s.now())
}

// MarkFiledAt marks a social insurance overview as filed at the given time. An
// already filed overview yields domain.ErrFilingAlreadyFiled and is left unchanged.
func (s *SocialInsuranceService) MarkFiledAt(ctx context.Context, id int64, at time.Time) (*domain.SocialInsuranceOverview, error) {
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

	sio.Status = domain.FilingStatusFiled
	sio.FiledAt = &at

	if sio.AdvanceStartMonth != 0 {
		if err := s.moveAdvanceStart(sio); err != nil {
			return nil, err
		}
	}

	if err := s.repo.Update(ctx, sio); err != nil {
		return nil, fmt.Errorf("marking social_insurance_overview as filed: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "social_insurance", id, "mark_filed", nil, sio)
	}
	return sio, nil
}
