package calc

import "github.com/zajca/zfaktury/internal/domain"

// InsuranceInput holds the parameters for an insurance calculation.
type InsuranceInput struct {
	Revenue        domain.Amount
	UsedExpenses   domain.Amount // already resolved (flat-rate or actual)
	MinMonthlyBase domain.Amount // e.g. constants.SocialMinMonthly or HealthMinMonthly
	RatePermille   int           // 292 for social, 135 for health
	Prepayments    domain.Amount
}

// InsuranceResult holds the computed insurance values.
type InsuranceResult struct {
	TaxBase             domain.Amount
	AssessmentBase      domain.Amount // taxBase / 2
	MinAssessmentBase   domain.Amount // minMonthlyBase * 12
	FinalAssessmentBase domain.Amount // max(assessmentBase, minAssessmentBase)
	TotalInsurance      domain.Amount // finalBase * rate / 1000
	Difference          domain.Amount // totalInsurance - prepayments
	NewMonthlyPrepay    domain.Amount // ceil(totalInsurance/12) rounded up to nearest CZK
}

// CalculateInsurance computes the annual insurance overview from the given input.
func CalculateInsurance(input InsuranceInput) InsuranceResult {
	// taxBase = revenue - usedExpenses, clamped to 0.
	taxBase := int64(input.Revenue) - int64(input.UsedExpenses)
	if taxBase < 0 {
		taxBase = 0
	}

	// assessmentBase = taxBase / 2 (integer division in halere).
	assessmentBase := taxBase / 2

	// minAssessmentBase = minMonthlyBase * 12.
	minAssessmentBase := int64(input.MinMonthlyBase) * 12

	// finalBase = max(assessmentBase, minAssessmentBase).
	finalBase := assessmentBase
	if minAssessmentBase > finalBase {
		finalBase = minAssessmentBase
	}

	// totalInsurance = finalBase * ratePermille / 1000.
	totalInsurance := finalBase * int64(input.RatePermille) / 1000

	// difference = totalInsurance - prepayments.
	difference := totalInsurance - int64(input.Prepayments)

	// newMonthlyPrepay = ceil(totalInsurance / 12), rounded up to nearest 100 halere (1 CZK).
	monthlyHalere := totalInsurance / 12
	if totalInsurance%12 != 0 {
		monthlyHalere++
	}
	roundedUpCZK := ((monthlyHalere + 99) / 100) * 100

	return InsuranceResult{
		TaxBase:             domain.Amount(taxBase),
		AssessmentBase:      domain.Amount(assessmentBase),
		MinAssessmentBase:   domain.Amount(minAssessmentBase),
		FinalAssessmentBase: domain.Amount(finalBase),
		TotalInsurance:      domain.Amount(totalInsurance),
		Difference:          domain.Amount(difference),
		NewMonthlyPrepay:    domain.Amount(roundedUpCZK),
	}
}

// ResolveUsedExpenses determines the expenses to use in tax calculations.
// If flatRatePercent > 0, it computes revenue * flatRatePercent/100, applies the
// cap from caps map if one exists for the given percent, and returns the result.
// Otherwise it returns actualExpenses unchanged.
func ResolveUsedExpenses(revenue, actualExpenses domain.Amount, flatRatePercent int, caps map[int]domain.Amount) domain.Amount {
	if flatRatePercent > 0 {
		amount := revenue.Multiply(float64(flatRatePercent) / 100.0)
		if cap, ok := caps[flatRatePercent]; ok && amount > cap {
			amount = cap
		}
		return amount
	}
	return actualExpenses
}
