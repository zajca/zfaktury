package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/annualtaxxml"
	"github.com/zajca/zfaktury/internal/calc"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// IncomeTaxReturnService provides business logic for income tax return management.
type IncomeTaxReturnService struct {
	repo                repository.IncomeTaxReturnRepo
	invoiceRepo         repository.InvoiceRepo
	expenseRepo         repository.ExpenseRepo
	settingsRepo        repository.SettingsRepo
	taxYearSettingsRepo repository.TaxYearSettingsRepo
	taxPrepaymentRepo   repository.TaxPrepaymentRepo
	taxCreditsSvc       *TaxCreditsService
	investmentSvc       *InvestmentIncomeService  // nullable
	employmentCerts     employmentCertificateRepo // nullable; §6 aggregation
	audit               *AuditService
}

// NewIncomeTaxReturnService creates a new IncomeTaxReturnService.
func NewIncomeTaxReturnService(
	repo repository.IncomeTaxReturnRepo,
	invoiceRepo repository.InvoiceRepo,
	expenseRepo repository.ExpenseRepo,
	settingsRepo repository.SettingsRepo,
	taxYearSettingsRepo repository.TaxYearSettingsRepo,
	taxPrepaymentRepo repository.TaxPrepaymentRepo,
	taxCreditsSvc *TaxCreditsService,
	audit *AuditService,
) *IncomeTaxReturnService {
	return &IncomeTaxReturnService{
		repo:                repo,
		invoiceRepo:         invoiceRepo,
		expenseRepo:         expenseRepo,
		settingsRepo:        settingsRepo,
		taxYearSettingsRepo: taxYearSettingsRepo,
		taxPrepaymentRepo:   taxPrepaymentRepo,
		taxCreditsSvc:       taxCreditsSvc,
		audit:               audit,
	}
}

// SetInvestmentService sets the optional investment income service for §8/§10 integration.
func (s *IncomeTaxReturnService) SetInvestmentService(investmentSvc *InvestmentIncomeService) {
	s.investmentSvc = investmentSvc
}

// SetEmploymentCertificateRepo wires the §6 employment certificate repository
// used during Recalculate to aggregate confirmed Potvrzení into ř.31 / ř.33 /
// ř.34 / ř.84 / ř.87 / ř.89 totals. Optional — when nil, §6 fields stay zero.
func (s *IncomeTaxReturnService) SetEmploymentCertificateRepo(repo employmentCertificateRepo) {
	s.employmentCerts = repo
}

// Create validates and persists a new income tax return.
func (s *IncomeTaxReturnService) Create(ctx context.Context, itr *domain.IncomeTaxReturn) error {
	if itr.Year < 2000 || itr.Year > 2100 {
		return fmt.Errorf("year out of valid range: %w", domain.ErrInvalidInput)
	}
	if itr.FilingType == "" {
		itr.FilingType = domain.FilingTypeRegular
	}
	switch itr.FilingType {
	case domain.FilingTypeRegular, domain.FilingTypeCorrective, domain.FilingTypeSupplementary:
		// ok
	default:
		return fmt.Errorf("invalid filing_type: %w", domain.ErrInvalidInput)
	}

	// Check for existing regular filing for this year.
	if itr.FilingType == domain.FilingTypeRegular {
		existing, err := s.repo.GetByYear(ctx, itr.Year, itr.FilingType)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("checking existing income_tax_return: %w", err)
		}
		if existing != nil {
			return domain.ErrFilingAlreadyExists
		}
	}

	if itr.Status == "" {
		itr.Status = domain.FilingStatusDraft
	}

	if err := s.repo.Create(ctx, itr); err != nil {
		return fmt.Errorf("creating income_tax_return: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "income_tax_return", itr.ID, "create", nil, itr)
	}
	return nil
}

// GetByID retrieves an income tax return by its ID.
func (s *IncomeTaxReturnService) GetByID(ctx context.Context, id int64) (*domain.IncomeTaxReturn, error) {
	if id == 0 {
		return nil, fmt.Errorf("income_tax_return ID is required: %w", domain.ErrInvalidInput)
	}
	itr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching income_tax_return: %w", err)
	}
	return itr, nil
}

// List retrieves all income tax returns for a given year.
func (s *IncomeTaxReturnService) List(ctx context.Context, year int) ([]domain.IncomeTaxReturn, error) {
	if year == 0 {
		year = time.Now().Year()
	}
	returns, err := s.repo.List(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("listing income_tax_returns: %w", err)
	}
	return returns, nil
}

// Delete removes an income tax return by ID. Filed returns cannot be deleted.
func (s *IncomeTaxReturnService) Delete(ctx context.Context, id int64) error {
	if id == 0 {
		return fmt.Errorf("income_tax_return ID is required: %w", domain.ErrInvalidInput)
	}

	itr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching income_tax_return for delete: %w", err)
	}
	if itr.Status == domain.FilingStatusFiled {
		return domain.ErrFilingAlreadyFiled
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting income_tax_return: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "income_tax_return", id, "delete", nil, nil)
	}
	return nil
}

// Recalculate recalculates the income tax return from linked invoices and expenses.
func (s *IncomeTaxReturnService) Recalculate(ctx context.Context, id int64) (*domain.IncomeTaxReturn, error) {
	if id == 0 {
		return nil, fmt.Errorf("income_tax_return ID is required: %w", domain.ErrInvalidInput)
	}

	itr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching income_tax_return for recalculation: %w", err)
	}
	if itr.Status == domain.FilingStatusFiled {
		return nil, domain.ErrFilingAlreadyFiled
	}

	// Step 1: Calculate annual base from invoices and expenses.
	base, err := CalculateAnnualBase(ctx, s.invoiceRepo, s.expenseRepo, itr.Year)
	if err != nil {
		return nil, fmt.Errorf("calculating annual base for income_tax_return: %w", err)
	}

	itr.TotalRevenue = base.Revenue
	itr.ActualExpenses = base.Expenses

	// Step 2: Read flat rate percent from tax year settings.
	flatRatePercent := 0
	tys, err := s.taxYearSettingsRepo.GetByYear(ctx, itr.Year)
	if err == nil {
		flatRatePercent = tys.FlatRatePercent
	}
	itr.FlatRatePercent = flatRatePercent

	// Step 3: Get tax constants for the year.
	constants, err := calc.GetTaxConstants(itr.Year)
	if err != nil {
		return nil, fmt.Errorf("getting tax constants for income_tax_return: %w", err)
	}

	// Step 4b: §8 capital income and §10 other income (investments).
	if s.investmentSvc != nil {
		summary, sumErr := s.investmentSvc.GetYearSummary(ctx, itr.Year)
		if sumErr != nil {
			return nil, fmt.Errorf("computing investment income for income_tax_return: %w", sumErr)
		}
		itr.CapitalIncomeGross = summary.CapitalIncomeGross
		itr.CapitalIncomeTax = summary.CapitalIncomeTax
		itr.CapitalIncomeNet = summary.CapitalIncomeNet
		itr.OtherIncomeGross = summary.OtherIncomeGross
		itr.OtherIncomeExpenses = summary.OtherIncomeExpenses
		itr.OtherIncomeExempt = summary.OtherIncomeExempt
		itr.OtherIncomeNet = summary.OtherIncomeNet
	}

	// Step 4c: §6 employment income aggregation (RFC-016).
	// Only confirmed certificates feed the totals. Section6CertsBonus stays 0
	// in MVP — counts of standalone "Potvrzení o vyplaceném daňovém bonusu"
	// uploads (EmploymentDocBonus kind) are out of scope.
	itr.Section6GrossIncome = 0
	itr.Section6IncomeWithoutAdvance = 0
	itr.Section6ForeignTax = 0
	itr.Section6AdvanceWithheld = 0
	itr.Section6WithholdingCredited = 0
	itr.Section6MonthlyBonusPaid = 0
	itr.Section6CertsAdvance = 0
	itr.Section6CertsWithholding = 0
	itr.Section6CertsBonus = 0
	if s.employmentCerts != nil {
		certs, certsErr := s.employmentCerts.ListConfirmedByYear(ctx, itr.Year)
		if certsErr != nil {
			return nil, fmt.Errorf("listing employment certificates for income_tax_return: %w", certsErr)
		}
		for _, c := range certs {
			switch c.CertificateType {
			case domain.CertificateAdvance:
				itr.Section6GrossIncome += c.GrossIncome
				itr.Section6IncomeWithoutAdvance += c.IncomeWithoutAdvance
				itr.Section6ForeignTax += c.ForeignTaxPaid
				itr.Section6AdvanceWithheld += c.AdvanceTaxWithheld - c.AnnualSettlementRefund
				itr.Section6MonthlyBonusPaid += c.MonthlyBonusPaid
				itr.Section6CertsAdvance++
				// Section6CertsBonus is the count of separate
				// "Potvrzení o vyplaceném daňovém bonusu" forms (potv_dazvyh)
				// — NOT advance certs that paid a bonus. Stays 0 in MVP.
			case domain.CertificateWithholding:
				if c.IncludeWithholdingInDAP {
					itr.Section6GrossIncome += c.GrossIncome
					itr.Section6WithholdingCredited += c.WithheldFinalTax
					itr.Section6CertsWithholding++
				}
			}
		}
	}
	// Dílčí ZD §6 (ř.34/36) = ř.31 - ř.33.
	itr.Section6TaxBase = itr.Section6GrossIncome - itr.Section6ForeignTax

	// Compute credits and deductions from DB (needed as calc inputs).
	var spouseCredit, disabilityCredit, studentCredit, childBenefit, totalDeductions domain.Amount
	if s.taxCreditsSvc != nil {
		var credErr error
		spouseCredit, disabilityCredit, studentCredit, credErr = s.taxCreditsSvc.ComputeCredits(ctx, itr.Year)
		if credErr != nil {
			return nil, fmt.Errorf("computing credits for income_tax_return: %w", credErr)
		}
		childBenefit, credErr = s.taxCreditsSvc.ComputeChildBenefit(ctx, itr.Year)
		if credErr != nil {
			return nil, fmt.Errorf("computing child benefit for income_tax_return: %w", credErr)
		}
		// Compute raw tax base for deduction cap calculation. Mirror the calc
		// package's "drop §7+§8+§10 if negative" guard so deduction caps that
		// reference tax base do not blow up when business expenses exceed revenue.
		otherSectionsBase := itr.TotalRevenue - calc.ResolveUsedExpenses(itr.TotalRevenue, itr.ActualExpenses, flatRatePercent, constants.FlatRateCaps) + itr.CapitalIncomeNet + itr.OtherIncomeNet
		if otherSectionsBase < 0 {
			otherSectionsBase = 0
		}
		rawBase := itr.Section6TaxBase + otherSectionsBase
		if rawBase < 0 {
			rawBase = 0
		}
		totalDeductions, credErr = s.taxCreditsSvc.ComputeDeductions(ctx, itr.Year, rawBase)
		if credErr != nil {
			return nil, fmt.Errorf("computing deductions for income_tax_return: %w", credErr)
		}

		// Fetch per-category breakdown from deductions after compute.
		// ComputeDeductions persisted AllowedAmount on each entry; sum them by category.
		deductions, dedErr := s.taxCreditsSvc.ListDeductions(ctx, itr.Year)
		if dedErr != nil {
			return nil, fmt.Errorf("listing deductions for breakdown: %w", dedErr)
		}
		var mortgage, lifeIns, pension, donation, union domain.Amount
		for _, ded := range deductions {
			switch ded.Category {
			case domain.DeductionMortgage:
				mortgage += ded.AllowedAmount
			case domain.DeductionLifeInsurance:
				lifeIns += ded.AllowedAmount
			case domain.DeductionPension:
				pension += ded.AllowedAmount
			case domain.DeductionDonation:
				donation += ded.AllowedAmount
			case domain.DeductionUnionDues:
				union += ded.AllowedAmount
			}
		}
		itr.DeductionMortgage = mortgage
		itr.DeductionLifeInsurance = lifeIns
		itr.DeductionPension = pension
		itr.DeductionDonation = donation
		itr.DeductionUnionDues = union
	}

	// Prepayments from tax prepayments table.
	taxTotal, _, _, sumErr := s.taxPrepaymentRepo.SumByYear(ctx, itr.Year)
	if sumErr != nil {
		return nil, fmt.Errorf("summing tax prepayments: %w", sumErr)
	}

	// Pure calculation.
	taxResult := calc.CalculateIncomeTax(calc.IncomeTaxInput{
		TotalRevenue:                itr.TotalRevenue,
		ActualExpenses:              itr.ActualExpenses,
		FlatRatePercent:             flatRatePercent,
		Constants:                   constants,
		SpouseCredit:                spouseCredit,
		DisabilityCredit:            disabilityCredit,
		StudentCredit:               studentCredit,
		ChildBenefit:                childBenefit,
		TotalDeductions:             totalDeductions,
		Prepayments:                 taxTotal,
		CapitalIncomeNet:            itr.CapitalIncomeNet,
		OtherIncomeNet:              itr.OtherIncomeNet,
		Section6TaxBase:             itr.Section6TaxBase,
		Section6AdvanceWithheld:     itr.Section6AdvanceWithheld,
		Section6WithholdingCredited: itr.Section6WithholdingCredited,
		Section6MonthlyBonusPaid:    itr.Section6MonthlyBonusPaid,
	})

	// Warnings: §16 odst. 1 ZDP progressive 23% rate review when the
	// consolidated tax base (§6 + §7 + §8 + §10) crosses 36× průměrná mzda
	// for the year. The existing CalculateIncomeTax handles the split
	// correctly; the warning prompts the user to verify the result.
	itr.Warnings = nil
	otherSectionsRaw := itr.TotalRevenue - taxResult.UsedExpenses + itr.CapitalIncomeNet + itr.OtherIncomeNet
	if otherSectionsRaw < 0 {
		otherSectionsRaw = 0
	}
	consolidatedBase := itr.Section6TaxBase + otherSectionsRaw
	if consolidatedBase > constants.ProgressiveThreshold {
		itr.Warnings = append(itr.Warnings, domain.WarningProgressiveRateReview)
	}

	// Map result back to entity.
	itr.FlatRateAmount = taxResult.FlatRateAmount
	itr.UsedExpenses = taxResult.UsedExpenses
	itr.TaxBase = taxResult.TaxBase
	itr.TotalDeductions = totalDeductions
	itr.TaxBaseRounded = taxResult.TaxBaseRounded
	itr.TaxAt15 = taxResult.TaxAt15
	itr.TaxAt23 = taxResult.TaxAt23
	itr.TotalTax = taxResult.TotalTax
	itr.CreditBasic = taxResult.CreditBasic
	itr.CreditSpouse = spouseCredit
	itr.CreditDisability = disabilityCredit
	itr.CreditStudent = studentCredit
	itr.TotalCredits = taxResult.TotalCredits
	itr.TaxAfterCredits = taxResult.TaxAfterCredits
	itr.ChildBenefit = childBenefit
	itr.TaxAfterBenefit = taxResult.TaxAfterBenefit
	itr.Prepayments = taxTotal
	itr.TaxDue = taxResult.TaxDue

	// Step 11: Persist updated values.
	if err := s.repo.Update(ctx, itr); err != nil {
		return nil, fmt.Errorf("updating income_tax_return after recalculation: %w", err)
	}

	// Step 12: Link invoices and expenses.
	if err := s.repo.LinkInvoices(ctx, itr.ID, base.InvoiceIDs); err != nil {
		return nil, fmt.Errorf("linking invoices to income_tax_return: %w", err)
	}
	if err := s.repo.LinkExpenses(ctx, itr.ID, base.ExpenseIDs); err != nil {
		return nil, fmt.Errorf("linking expenses to income_tax_return: %w", err)
	}

	if s.audit != nil {
		s.audit.Log(ctx, "income_tax_return", id, "update", nil, itr)
	}
	return itr, nil
}

// GenerateXML generates the DPFO XML for an income tax return.
func (s *IncomeTaxReturnService) GenerateXML(ctx context.Context, id int64) (*domain.IncomeTaxReturn, error) {
	if id == 0 {
		return nil, fmt.Errorf("income_tax_return ID is required: %w", domain.ErrInvalidInput)
	}

	itr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching income_tax_return for XML generation: %w", err)
	}

	// Load settings for taxpayer info. c_okec is also pulled because the income-tax
	// XML reuses it as the NACE code for Příloha č. 1 part B.
	// The XML generator expects taxpayer_* prefixed keys, but the actual settings
	// store uses unprefixed keys (first_name, last_name, street, ...). Translate here.
	settings := make(map[string]string)
	settingKeyMap := map[string]string{
		"first_name":   "taxpayer_first_name",
		"last_name":    "taxpayer_last_name",
		"birth_number": "taxpayer_birth_number",
		"street":       "taxpayer_street",
		"house_number": "taxpayer_house_number",
		"city":         "taxpayer_city",
		"zip":          "taxpayer_postal_code",
		"phone":        "taxpayer_phone",
	}
	for srcKey, dstKey := range settingKeyMap {
		if val, err := s.settingsRepo.Get(ctx, srcKey); err == nil {
			settings[dstKey] = val
		}
	}
	for _, key := range []string{
		"financni_urad_code", "c_pracufo", "dic", "c_okec",
		"main_activity_nace", "main_activity_months",
		"mortgage_interest_months",
	} {
		if val, err := s.settingsRepo.Get(ctx, key); err == nil {
			settings[key] = val
		}
	}

	// Load child-credit entries so the EPO ř.72 formula control can verify
	// kc_dazvyhod = sum over m_deti* slots × per-order amounts.
	var children []domain.TaxChildCredit
	if s.taxCreditsSvc != nil {
		children, err = s.taxCreditsSvc.ListChildren(ctx, itr.Year)
		if err != nil {
			return nil, fmt.Errorf("listing child credits for XML generation: %w", err)
		}
	}

	xmlData, err := annualtaxxml.GenerateIncomeTaxXML(itr, settings, children)
	if err != nil {
		return nil, fmt.Errorf("generating income_tax_return XML: %w", err)
	}

	itr.XMLData = xmlData
	if err := s.repo.Update(ctx, itr); err != nil {
		return nil, fmt.Errorf("saving income_tax_return XML: %w", err)
	}

	if s.audit != nil {
		s.audit.Log(ctx, "income_tax_return", id, "generate_xml", nil, nil)
	}
	return itr, nil
}

// GetXMLData retrieves the stored XML data for an income tax return.
func (s *IncomeTaxReturnService) GetXMLData(ctx context.Context, id int64) ([]byte, error) {
	if id == 0 {
		return nil, fmt.Errorf("income_tax_return ID is required: %w", domain.ErrInvalidInput)
	}
	itr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching income_tax_return for XML data: %w", err)
	}
	return itr.XMLData, nil
}

// MarkFiled marks an income tax return as filed and records the timestamp.
func (s *IncomeTaxReturnService) MarkFiled(ctx context.Context, id int64) (*domain.IncomeTaxReturn, error) {
	if id == 0 {
		return nil, fmt.Errorf("income_tax_return ID is required: %w", domain.ErrInvalidInput)
	}

	itr, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching income_tax_return for marking as filed: %w", err)
	}
	if itr.Status == domain.FilingStatusFiled {
		return nil, domain.ErrFilingAlreadyFiled
	}

	now := time.Now()
	itr.Status = domain.FilingStatusFiled
	itr.FiledAt = &now

	if err := s.repo.Update(ctx, itr); err != nil {
		return nil, fmt.Errorf("marking income_tax_return as filed: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "income_tax_return", id, "mark_filed", nil, nil)
	}
	return itr, nil
}
