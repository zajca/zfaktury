package calc

import "github.com/zajca/zfaktury/internal/domain"

// IncomeTaxInput holds all inputs needed for income tax calculation.
type IncomeTaxInput struct {
	TotalRevenue     domain.Amount
	ActualExpenses   domain.Amount
	FlatRatePercent  int
	Constants        TaxYearConstants
	SpouseCredit     domain.Amount
	DisabilityCredit domain.Amount
	StudentCredit    domain.Amount
	ChildBenefit     domain.Amount
	TotalDeductions  domain.Amount
	Prepayments      domain.Amount
	CapitalIncomeNet domain.Amount
	OtherIncomeNet   domain.Amount
	// Section6TaxBase is the §6 employment dílčí základ (ř.34/36 = ř.31 − ř.33).
	// Added to the §16 progressive tax base. XSD rule for ř.42 ("if úhrn §7+§8+§10
	// is negative, use only §6") is enforced here: §7+§8+§10 contributions are
	// dropped when their sum is negative, so §6 alone forms the tax base in that
	// case (matches the XML generator emission).
	Section6TaxBase domain.Amount

	// Section6 reconciliation values (RFC-016). Following Pokyny DPFO 2025
	// oddíl 7, ř.91 (kc_zbyvpred = "zbývá doplatit") is computed as
	//   ř.77 − ř.84 − ř.85 − ř.86 − ř.87 − ř.87a − úhrn záloh §7
	// where ř.84 is the employer-withheld advance and ř.87 is §36 odst. 6
	// withholding voluntarily included in DAP. We subtract those here so
	// TaxDue (which feeds kc_zbyvpred) reflects the user's true balance.
	//
	// Section6MonthlyBonusPaid (ř.89) is the bonus the employer already paid
	// out monthly. Reconciliation: TaxAfterBenefit holds the user's claim
	// (negative when claimed); adding back the paid-out bonus reduces the
	// claim or, if it exceeds the entitlement, increases TaxDue (the user
	// must return the excess to the státu).
	Section6AdvanceWithheld     domain.Amount // ř.84
	Section6WithholdingCredited domain.Amount // ř.87 (only when user opted to include §36/6 in DAP)
	Section6MonthlyBonusPaid    domain.Amount // ř.89 (kc_vyplbonus)
}

// IncomeTaxResult holds all computed values from the income tax calculation.
type IncomeTaxResult struct {
	FlatRateAmount  domain.Amount
	UsedExpenses    domain.Amount
	TaxBase         domain.Amount
	TaxBaseRounded  domain.Amount
	TaxAt15         domain.Amount
	TaxAt23         domain.Amount
	TotalTax        domain.Amount
	CreditBasic     domain.Amount
	TotalCredits    domain.Amount
	TaxAfterCredits domain.Amount
	TaxAfterBenefit domain.Amount
	TaxDue          domain.Amount
}

// CalculateIncomeTax performs the full income tax calculation (steps 4-10).
func CalculateIncomeTax(input IncomeTaxInput) IncomeTaxResult {
	var result IncomeTaxResult

	// Step 4: Determine used expenses (flat rate vs actual).
	if input.FlatRatePercent > 0 {
		flatRateAmount := input.TotalRevenue.Multiply(float64(input.FlatRatePercent) / 100.0)
		cap, hasCap := input.Constants.FlatRateCaps[input.FlatRatePercent]
		if hasCap && flatRateAmount > cap {
			flatRateAmount = cap
		}
		result.FlatRateAmount = flatRateAmount
		result.UsedExpenses = flatRateAmount
	} else {
		result.FlatRateAmount = 0
		result.UsedExpenses = input.ActualExpenses
	}

	// Step 5: Tax base = §6 + max(0, §7 + §8 + §10).
	// §7 (business): TotalRevenue - UsedExpenses
	// §8 (capital):  CapitalIncomeNet
	// §10 (other):   OtherIncomeNet
	// XSD rule for DPFO ř.42 ("Pokud je ř.41 záporný, uveďte pouze hodnotu z
	// ř.36"): if úhrn §7+§8+§10 is negative, drop it and use just §6.
	otherSectionsBase := input.TotalRevenue - result.UsedExpenses + input.CapitalIncomeNet + input.OtherIncomeNet
	if otherSectionsBase < 0 {
		otherSectionsBase = 0
	}
	taxBase := input.Section6TaxBase + otherSectionsBase
	if taxBase < 0 {
		taxBase = 0
	}
	result.TaxBase = taxBase

	// Step 5b: Apply deductions (reduce tax base before rounding).
	taxBase -= input.TotalDeductions
	if taxBase < 0 {
		taxBase = 0
	}

	// Step 6: Round down to 100 CZK (10000 halere).
	result.TaxBaseRounded = (taxBase / 10000) * 10000

	// Step 7: Progressive tax calculation.
	threshold := input.Constants.ProgressiveThreshold
	taxBaseRounded := result.TaxBaseRounded

	if taxBaseRounded <= threshold {
		result.TaxAt15 = taxBaseRounded.Multiply(0.15)
		result.TaxAt23 = 0
	} else {
		result.TaxAt15 = threshold.Multiply(0.15)
		result.TaxAt23 = (taxBaseRounded - threshold).Multiply(0.23)
	}
	result.TotalTax = result.TaxAt15 + result.TaxAt23

	// Step 8: Tax credits.
	result.CreditBasic = input.Constants.BasicCredit
	result.TotalCredits = result.CreditBasic + input.SpouseCredit + input.DisabilityCredit + input.StudentCredit

	taxAfterCredits := result.TotalTax - result.TotalCredits
	if taxAfterCredits < 0 {
		taxAfterCredits = 0
	}
	result.TaxAfterCredits = taxAfterCredits

	// Step 9: Child benefit (can go negative - it's a bonus).
	result.TaxAfterBenefit = result.TaxAfterCredits - input.ChildBenefit

	// Step 10: Reconcile prepayments and §6 withholdings/advances (Pokyny
	// DPFO 2025 oddíl 7, ř.91 = ř.77 − ř.84 − ř.87 − ř.87a − úhrn záloh §7).
	// Section6MonthlyBonusPaid (ř.89) is added back: TaxAfterBenefit already
	// captured the user's bonus claim (negative); the paid-out portion must
	// be netted off (if equal → 0, if employer over-paid → positive doplatek).
	result.TaxDue = result.TaxAfterBenefit -
		input.Prepayments -
		input.Section6AdvanceWithheld -
		input.Section6WithholdingCredited +
		input.Section6MonthlyBonusPaid

	return result
}
