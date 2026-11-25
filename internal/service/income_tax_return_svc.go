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
	"github.com/zajca/zfaktury/internal/taxrules"
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

	// Step 3: Get the canonical tax rule set for the year.
	rules, err := taxrules.GetRuleSet(itr.Year)
	if err != nil {
		return nil, fmt.Errorf("getting tax rule set for income_tax_return: %w", err)
	}
	constants := rules.Constants

	// Step 4b: §8 capital income and §10 other income (investments).
	if s.investmentSvc != nil {
		// Recompute FIFO first: results stored by an older build can keep
		// outdated exemptions (e.g. a crypto sale before 2025-02-15 stored as
		// time-test exempt). FIFO is a pure function of the transactions and
		// the current rule set, so recomputing here is always safe.
		if fifoErr := s.investmentSvc.RecalculateFIFO(ctx, itr.Year); fifoErr != nil {
			return nil, fmt.Errorf("recalculating FIFO for income_tax_return: %w", fifoErr)
		}
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
		// § 10 odst. 4 sentence 3: a negative § 10 total never lowers the other
		// parts of the tax base. Store the floored value so the persisted return
		// and XML ř.40 carry the same value calc uses for the tax base.
		if itr.OtherIncomeNet < 0 {
			itr.OtherIncomeNet = 0
		}
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
	var spouseCredit, childBenefit, totalDeductions domain.Amount
	var personal calc.PersonalCreditsResult
	if s.taxCreditsSvc != nil {
		var credErr error
		spouseCredit, personal, credErr = s.taxCreditsSvc.ComputeCreditsWithConstants(ctx, itr.Year, constants)
		if credErr != nil {
			return nil, fmt.Errorf("computing credits for income_tax_return: %w", credErr)
		}
		childBenefit, credErr = s.taxCreditsSvc.ComputeChildBenefitWithConstants(ctx, itr.Year, constants)
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
		totalDeductions, credErr = s.taxCreditsSvc.ComputeDeductionsWithConstants(ctx, itr.Year, rawBase, constants)
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
		DisabilityCredit:            personal.Disability(),
		StudentCredit:               personal.Student,
		ChildBenefit:                childBenefit,
		TotalDeductions:             totalDeductions,
		Prepayments:                 taxTotal,
		CapitalIncomeNet:            itr.CapitalIncomeNet,
		OtherIncomeNet:              itr.OtherIncomeNet,
		Section6TaxBase:             itr.Section6TaxBase,
		Section6GrossIncome:         itr.Section6GrossIncome,
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
	if rules.Status == taxrules.RuleStatusProvisional {
		itr.Warnings = append(itr.Warnings, domain.WarningTaxRuleSetProvisional)
	}
	// § 35c odst. 3/4: when a computed bonus is zeroed by a statutory limit, surface
	// the reason as an advisory warning (the sleva part is never limited).
	if excess := childBenefit - taxResult.ChildBenefitSleva; excess > 0 && taxResult.ChildBenefitBonus == 0 {
		if itr.Section6GrossIncome+itr.TotalRevenue < constants.MinimumWage*6 {
			itr.Warnings = append(itr.Warnings, domain.WarningChildBonusIncomeTest)
		} else {
			itr.Warnings = append(itr.Warnings, domain.WarningChildBonusBelowMinimum)
		}
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
	itr.CreditDisability = personal.Disability()
	itr.CreditDisability12 = personal.Disability12
	itr.CreditDisability3 = personal.Disability3
	itr.CreditDisabilityZTPP = personal.DisabilityZTPP
	itr.CreditStudent = personal.Student
	itr.TotalCredits = taxResult.TotalCredits
	itr.TaxAfterCredits = taxResult.TaxAfterCredits
	itr.ChildBenefit = childBenefit
	itr.TaxAfterBenefit = taxResult.TaxAfterBenefit
	itr.Prepayments = taxTotal
	itr.TaxDue = taxResult.TaxDue
	ruleSetHash, err := rules.Fingerprint()
	if err != nil {
		return nil, fmt.Errorf("fingerprinting tax rule set for income_tax_return: %w", err)
	}
	itr.TaxRuleSetID = rules.ID
	itr.TaxRuleSetStatus = string(rules.Status)
	itr.TaxRuleSetHash = ruleSetHash
	now := time.Now()
	itr.CalculatedAt = &now

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
	var spouse *domain.TaxSpouseCredit
	var personal *domain.TaxPersonalCredits
	if s.taxCreditsSvc != nil {
		children, err = s.taxCreditsSvc.ListChildren(ctx, itr.Year)
		if err != nil {
			return nil, fmt.Errorf("listing child credits for XML generation: %w", err)
		}
		// The spouse record supplies the identity and months of rows 65a/65b.
		spouse, err = s.taxCreditsSvc.GetSpouse(ctx, itr.Year)
		if err != nil {
			if !errors.Is(err, domain.ErrNotFound) {
				return nil, fmt.Errorf("fetching spouse credit for XML generation: %w", err)
			}
			spouse = nil
		}
		// The personal credits record supplies the months of the disability rows 66-68.
		personal, err = s.taxCreditsSvc.GetPersonal(ctx, itr.Year)
		if err != nil {
			if !errors.Is(err, domain.ErrNotFound) {
				return nil, fmt.Errorf("fetching personal credits for XML generation: %w", err)
			}
			personal = nil
		}
	}

	if s.taxCreditsSvc != nil {
		rules, ruleErr := taxrules.GetRuleSet(itr.Year)
		if ruleErr != nil {
			return nil, fmt.Errorf("getting tax rule set for income_tax_return XML generation: %w", ruleErr)
		}
		constants := rules.Constants

		spouseCredit, personalResult, credErr := s.taxCreditsSvc.ComputeCreditsWithConstants(ctx, itr.Year, constants)
		if credErr != nil {
			return nil, fmt.Errorf("computing credits for income_tax_return XML generation: %w", credErr)
		}

		// Legacy returns calculated before migration 037 store only the aggregate
		// disability credit. Re-derive the per-degree components from the stored
		// personal credits so the generator can emit rows 66/67/68. The components
		// are persisted together with the XML by the Update call below.
		if itr.CreditDisability > 0 && itr.CreditDisability12 == 0 && itr.CreditDisability3 == 0 && itr.CreditDisabilityZTPP == 0 {
			if personalResult.Disability() != itr.CreditDisability {
				if itr.Status == domain.FilingStatusFiled {
					return nil, fmt.Errorf("uložená sleva na invaliditu neodpovídá aktuálnímu nastavení osobních slev, přiznání nelze bez obnovení nastavení znovu vygenerovat: %w", domain.ErrInvalidInput)
				}
				return nil, fmt.Errorf("přiznání bylo spočteno starší verzí, přepočítejte ho: %w", domain.ErrInvalidInput)
			}
			itr.CreditDisability12 = personalResult.Disability12
			itr.CreditDisability3 = personalResult.Disability3
			itr.CreditDisabilityZTPP = personalResult.DisabilityZTPP
		}

		// The stored spouse credit and disability components must match a fresh
		// computation from the current credit records, otherwise the XML rows
		// 65-68 would not back the stored totals.
		if spouseCredit != itr.CreditSpouse {
			// A spouse record stored before migration 045 has child_under3_months = 0,
			// so the message names the field the owner has to fill in.
			if itr.Status == domain.FilingStatusFiled {
				return nil, fmt.Errorf("vyplňte u slevy na manžela/ku měsíce s dítětem do 3 let ve společné domácnosti, přiznání nelze bez obnovení nastavení znovu vygenerovat: %w", domain.ErrInvalidInput)
			}
			return nil, fmt.Errorf("vyplňte u slevy na manžela/ku měsíce s dítětem do 3 let ve společné domácnosti a přepočítejte přiznání: %w", domain.ErrInvalidInput)
		}
		if personalResult.Disability12 != itr.CreditDisability12 ||
			personalResult.Disability3 != itr.CreditDisability3 ||
			personalResult.DisabilityZTPP != itr.CreditDisabilityZTPP {
			if itr.Status == domain.FilingStatusFiled {
				return nil, fmt.Errorf("uložená sleva na invaliditu neodpovídá aktuálnímu nastavení osobních slev, přiznání nelze bez obnovení nastavení znovu vygenerovat: %w", domain.ErrInvalidInput)
			}
			return nil, fmt.Errorf("údaje o slevách se od posledního přepočtu změnily, přepočítejte přiznání: %w", domain.ErrInvalidInput)
		}

		// The stored child benefit must match the child data that feeds the m_deti*
		// slots, otherwise EPO rejects the ř.72 control (code 181).
		childBenefit, credErr := s.taxCreditsSvc.ComputeChildBenefitWithConstants(ctx, itr.Year, constants)
		if credErr != nil {
			return nil, fmt.Errorf("computing child benefit for income_tax_return XML generation: %w", credErr)
		}
		if childBenefit != itr.ChildBenefit {
			if itr.Status == domain.FilingStatusFiled {
				return nil, fmt.Errorf("údaje o dětech neodpovídají podanému přiznání, přiznání nelze znovu vygenerovat: %w", domain.ErrInvalidInput)
			}
			return nil, fmt.Errorf("údaje o dětech se od posledního přepočtu změnily, přepočítejte přiznání: %w", domain.ErrInvalidInput)
		}

		// The stored limited split must match the split recomputed from the stored
		// inputs under the current rules (§ 35c odst. 3 and 4). A return calculated
		// before those limits would export ř.73/76 with the new rules while ř.91
		// (kc_zbyvpred) still carries the stale TaxDue — a self-contradicting XML.
		sleva, bonus := calc.SplitChildBenefit(itr.TaxAfterCredits, itr.ChildBenefit, itr.Section6GrossIncome, itr.TotalRevenue, constants)
		if itr.TaxAfterBenefit != itr.TaxAfterCredits-sleva-bonus {
			if itr.Status == domain.FilingStatusFiled {
				return nil, fmt.Errorf("uložený daňový bonus neodpovídá podanému přiznání, přiznání nelze znovu vygenerovat: %w", domain.ErrInvalidInput)
			}
			return nil, fmt.Errorf("přiznání bylo spočteno starší verzí, přepočítejte ho: %w", domain.ErrInvalidInput)
		}
	}

	xmlData, err := annualtaxxml.GenerateIncomeTaxXML(itr, settings, children, spouse, personal)
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
	return s.MarkFiledAt(ctx, id, time.Now())
}

// MarkFiledAt marks an income tax return as filed at the given time. An
// already filed return yields domain.ErrFilingAlreadyFiled and is left unchanged.
func (s *IncomeTaxReturnService) MarkFiledAt(ctx context.Context, id int64, at time.Time) (*domain.IncomeTaxReturn, error) {
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

	itr.Status = domain.FilingStatusFiled
	itr.FiledAt = &at

	if err := s.repo.Update(ctx, itr); err != nil {
		return nil, fmt.Errorf("marking income_tax_return as filed: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "income_tax_return", id, "mark_filed", nil, nil)
	}
	return itr, nil
}
