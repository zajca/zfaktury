package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/annualtaxxml"
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
) *IncomeTaxReturnService {
	return &IncomeTaxReturnService{
		repo:                repo,
		invoiceRepo:         invoiceRepo,
		expenseRepo:         expenseRepo,
		settingsRepo:        settingsRepo,
		taxYearSettingsRepo: taxYearSettingsRepo,
		taxPrepaymentRepo:   taxPrepaymentRepo,
		taxCreditsSvc:       taxCreditsSvc,
	}
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
	constants, err := GetTaxConstants(itr.Year)
	if err != nil {
		return nil, fmt.Errorf("getting tax constants for income_tax_return: %w", err)
	}

	// Step 4: Determine used expenses (flat rate vs actual).
	if flatRatePercent > 0 {
		flatRateAmount := itr.TotalRevenue.Multiply(float64(flatRatePercent) / 100.0)
		cap, hasCap := constants.FlatRateCaps[flatRatePercent]
		if hasCap && flatRateAmount > cap {
			flatRateAmount = cap
		}
		itr.FlatRateAmount = flatRateAmount
		itr.UsedExpenses = flatRateAmount
	} else {
		itr.FlatRateAmount = 0
		itr.UsedExpenses = itr.ActualExpenses
	}

	// Step 5: Tax base.
	taxBase := itr.TotalRevenue - itr.UsedExpenses
	if taxBase < 0 {
		taxBase = 0
	}
	itr.TaxBase = taxBase

	// Step 5b: Apply deductions (nezdanitelne casti) - reduce tax base before rounding.
	var totalDeductions domain.Amount
	if s.taxCreditsSvc != nil {
		deductions, deductErr := s.taxCreditsSvc.ComputeDeductions(ctx, itr.Year, taxBase)
		if deductErr != nil {
			return nil, fmt.Errorf("computing deductions for income_tax_return: %w", deductErr)
		}
		totalDeductions = deductions
	}
	itr.TotalDeductions = totalDeductions
	taxBase -= totalDeductions
	if taxBase < 0 {
		taxBase = 0
	}

	// Step 6: Round down to 100 CZK (10000 halere).
	itr.TaxBaseRounded = (taxBase / 10000) * 10000

	// Step 7: Progressive tax calculation.
	threshold := constants.ProgressiveThreshold
	taxBaseRounded := itr.TaxBaseRounded

	var tax15, tax23 domain.Amount
	if taxBaseRounded <= threshold {
		tax15 = taxBaseRounded.Multiply(0.15)
		tax23 = 0
	} else {
		tax15 = threshold.Multiply(0.15)
		tax23 = (taxBaseRounded - threshold).Multiply(0.23)
	}
	itr.TaxAt15 = tax15
	itr.TaxAt23 = tax23
	itr.TotalTax = tax15 + tax23

	// Step 8: Tax credits - load from tax_credits tables.
	itr.CreditBasic = constants.BasicCredit
	if s.taxCreditsSvc != nil {
		spouseCredit, disabilityCredit, studentCredit, credErr := s.taxCreditsSvc.ComputeCredits(ctx, itr.Year)
		if credErr != nil {
			return nil, fmt.Errorf("computing credits for income_tax_return: %w", credErr)
		}
		itr.CreditSpouse = spouseCredit
		itr.CreditDisability = disabilityCredit
		itr.CreditStudent = studentCredit
	}
	itr.TotalCredits = itr.CreditBasic + itr.CreditSpouse + itr.CreditDisability + itr.CreditStudent

	taxAfterCredits := itr.TotalTax - itr.TotalCredits
	if taxAfterCredits < 0 {
		taxAfterCredits = 0
	}
	itr.TaxAfterCredits = taxAfterCredits

	// Step 9: Child benefit - load from tax_child_credits table.
	if s.taxCreditsSvc != nil {
		childBenefit, cbErr := s.taxCreditsSvc.ComputeChildBenefit(ctx, itr.Year)
		if cbErr != nil {
			return nil, fmt.Errorf("computing child benefit for income_tax_return: %w", cbErr)
		}
		itr.ChildBenefit = childBenefit
	}
	itr.TaxAfterBenefit = itr.TaxAfterCredits - itr.ChildBenefit

	// Step 10: Prepayments from tax prepayments table.
	taxTotal, _, _, sumErr := s.taxPrepaymentRepo.SumByYear(ctx, itr.Year)
	if sumErr != nil {
		return nil, fmt.Errorf("summing tax prepayments: %w", sumErr)
	}
	itr.Prepayments = taxTotal
	itr.TaxDue = itr.TaxAfterBenefit - itr.Prepayments

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

	// Load settings for taxpayer info.
	settings := make(map[string]string)
	for _, key := range []string{
		"financni_urad_code", "taxpayer_first_name", "taxpayer_last_name",
		"taxpayer_birth_number", "dic", "taxpayer_street",
		"taxpayer_house_number", "taxpayer_city", "taxpayer_postal_code",
	} {
		val, err := s.settingsRepo.Get(ctx, key)
		if err == nil {
			settings[key] = val
		}
	}

	xmlData, err := annualtaxxml.GenerateIncomeTaxXML(itr, settings)
	if err != nil {
		return nil, fmt.Errorf("generating income_tax_return XML: %w", err)
	}

	itr.XMLData = xmlData
	if err := s.repo.Update(ctx, itr); err != nil {
		return nil, fmt.Errorf("saving income_tax_return XML: %w", err)
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
	return itr, nil
}
