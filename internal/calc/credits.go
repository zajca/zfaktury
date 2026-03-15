package calc

import "github.com/zajca/zfaktury/internal/domain"

// ComputeSpouseCredit computes the spouse tax credit.
// Returns 0 if spouse income is at or above the income limit.
// The credit is proportional to months claimed (out of 12).
// If spouseZTP is true, the credit is doubled.
func ComputeSpouseCredit(spouseIncome domain.Amount, monthsClaimed int, spouseZTP bool, constants TaxYearConstants) domain.Amount {
	if spouseIncome >= constants.SpouseIncomeLimit {
		return 0
	}
	credit := constants.SpouseCredit.Multiply(float64(monthsClaimed) / 12.0)
	if spouseZTP {
		credit = credit.Multiply(2)
	}
	return credit
}

// ComputePersonalCredits computes disability and student credits.
// disabilityLevel: 1 = first/second degree, 2 = third degree, 3 = ZTP/P holder, 0 = none.
// Student credit is proportional to studentMonths (out of 12).
func ComputePersonalCredits(disabilityLevel int, isStudent bool, studentMonths int, constants TaxYearConstants) (disability, student domain.Amount) {
	switch disabilityLevel {
	case 1:
		disability = constants.DisabilityCredit1
	case 2:
		disability = constants.DisabilityCredit3
	case 3:
		disability = constants.DisabilityZTPP
	default:
		disability = 0
	}

	if isStudent && studentMonths > 0 {
		student = constants.StudentCredit.Multiply(float64(studentMonths) / 12.0)
	}

	return disability, student
}

// ChildCreditInput describes a single child for benefit computation.
type ChildCreditInput struct {
	ChildOrder    int // 1, 2, or 3+ (any value >= 3 is treated as 3+)
	MonthsClaimed int
	ZTP           bool
}

// ComputeChildBenefit computes the total child benefit for all children.
func ComputeChildBenefit(children []ChildCreditInput, constants TaxYearConstants) domain.Amount {
	var total domain.Amount
	for _, child := range children {
		var base domain.Amount
		switch child.ChildOrder {
		case 1:
			base = constants.ChildBenefit1
		case 2:
			base = constants.ChildBenefit2
		default:
			base = constants.ChildBenefit3Plus
		}
		amount := base.Multiply(float64(child.MonthsClaimed) / 12.0)
		if child.ZTP {
			amount = amount.Multiply(2)
		}
		total += amount
	}
	return total
}

// DeductionInput describes a single tax deduction claim.
type DeductionInput struct {
	Category      string
	ClaimedAmount domain.Amount
}

// DeductionResult holds the computed allowed deduction amounts.
type DeductionResult struct {
	AllowedAmounts []domain.Amount // parallel to input slice
	TotalAllowed   domain.Amount
}

// ComputeDeductions computes allowed deduction amounts with statutory caps.
// Each category has a maximum cap; multiple deductions of the same category share the cap.
// Donation cap is 15% of the tax base.
func ComputeDeductions(deductions []DeductionInput, taxBase domain.Amount, constants TaxYearConstants) DeductionResult {
	categoryCaps := map[string]domain.Amount{
		domain.DeductionMortgage:      constants.DeductionCapMortgage,
		domain.DeductionLifeInsurance: constants.DeductionCapLifeInsurance,
		domain.DeductionPension:       constants.DeductionCapPension,
		domain.DeductionUnionDues:     constants.DeductionCapUnionDues,
		domain.DeductionDonation:      taxBase.Multiply(0.15),
	}

	remainingCap := make(map[string]domain.Amount, len(categoryCaps))
	for k, v := range categoryCaps {
		remainingCap[k] = v
	}

	result := DeductionResult{
		AllowedAmounts: make([]domain.Amount, len(deductions)),
	}

	for i, d := range deductions {
		remaining, ok := remainingCap[d.Category]
		if !ok {
			// Unknown category, allow nothing.
			result.AllowedAmounts[i] = 0
			continue
		}

		allowed := d.ClaimedAmount
		if allowed > remaining {
			allowed = remaining
		}
		if allowed < 0 {
			allowed = 0
		}

		remainingCap[d.Category] = remaining - allowed
		result.AllowedAmounts[i] = allowed
		result.TotalAllowed += allowed
	}

	return result
}
