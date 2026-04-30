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
	investmentSvc       *InvestmentIncomeService // nullable
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
		// Compute raw tax base for deduction cap calculation.
		rawBase := itr.TotalRevenue - calc.ResolveUsedExpenses(itr.TotalRevenue, itr.ActualExpenses, flatRatePercent, constants.FlatRateCaps) + itr.CapitalIncomeNet + itr.OtherIncomeNet
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
		TotalRevenue:     itr.TotalRevenue,
		ActualExpenses:   itr.ActualExpenses,
		FlatRatePercent:  flatRatePercent,
		Constants:        constants,
		SpouseCredit:     spouseCredit,
		DisabilityCredit: disabilityCredit,
		StudentCredit:    studentCredit,
		ChildBenefit:     childBenefit,
		TotalDeductions:  totalDeductions,
		Prepayments:      taxTotal,
		CapitalIncomeNet: itr.CapitalIncomeNet,
		OtherIncomeNet:   itr.OtherIncomeNet,
	})

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
	settings := make(map[string]string)
	for _, key := range []string{
		"financni_urad_code", "taxpayer_first_name", "taxpayer_last_name",
		"taxpayer_birth_number", "dic", "taxpayer_street",
		"taxpayer_house_number", "taxpayer_city", "taxpayer_postal_code",
		"c_okec", "main_activity_nace", "main_activity_months",
	} {
		val, err := s.settingsRepo.Get(ctx, key)
		if err == nil {
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
