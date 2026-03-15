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

	// Step 5: Tax base (revenue - expenses + capital income + other income).
	taxBase := input.TotalRevenue - result.UsedExpenses + input.CapitalIncomeNet + input.OtherIncomeNet
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

	// Step 10: Prepayments (can be negative = refund).
	result.TaxDue = result.TaxAfterBenefit - input.Prepayments

	return result
}
