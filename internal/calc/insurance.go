package calc

import (
	"fmt"

	"github.com/zajca/zfaktury/internal/domain"
)

// InsuranceInput holds the parameters for an insurance calculation.
type InsuranceInput struct {
	Revenue        domain.Amount
	UsedExpenses   domain.Amount // already resolved (flat-rate or actual)
	MinMonthlyBase domain.Amount // e.g. constants.SocialMinMonthly or HealthMinMonthly
	// MaxAnnualBase caps the assessment base (social insurance, CSSZ). Zero
	// means no cap (health insurance has no maximum assessment base).
	MaxAnnualBase domain.Amount
	// AssessmentBasePermille is the share of tax base used as the assessment
	// base. Zero keeps the historical/default 50% behavior.
	AssessmentBasePermille int
	RatePermille           int // 292 for social, 135 for health
	// RoundPremiumUp rounds the total premium up to whole CZK regardless of
	// AssessmentBasePermille (health insurance, § 2 odst. 2 zakona c. 592/1992 Sb.).
	RoundPremiumUp bool
	Prepayments    domain.Amount
	// ActivityMonths is the number of months of activity in the year (1-12).
	// It prorates the minimum assessment base; the maximum is not prorated.
	ActivityMonths int
}

// InsuranceResult holds the computed insurance values.
type InsuranceResult struct {
	TaxBase             domain.Amount
	AssessmentBase      domain.Amount // taxBase / 2
	MinAssessmentBase   domain.Amount // minMonthlyBase * activityMonths
	FinalAssessmentBase domain.Amount // min(max(assessmentBase, minAssessmentBase), maxAnnualBase)
	TotalInsurance      domain.Amount // finalBase * rate / 1000
	Difference          domain.Amount // totalInsurance - prepayments
	// NewMonthlyPrepay is the advance on premiums per month of activity, rounded
	// up to whole CZK.
	NewMonthlyPrepay domain.Amount
}

// CalculateInsurance computes the annual insurance overview from the given input.
func CalculateInsurance(input InsuranceInput) (InsuranceResult, error) {
	if input.ActivityMonths < 1 || input.ActivityMonths > 12 {
		return InsuranceResult{}, fmt.Errorf("activity months %d out of range 1-12: %w", input.ActivityMonths, domain.ErrInvalidInput)
	}
	months := int64(input.ActivityMonths)

	// taxBase = revenue - usedExpenses, clamped to 0.
	taxBase := int64(input.Revenue) - int64(input.UsedExpenses)
	if taxBase < 0 {
		taxBase = 0
	}

	assessmentBasePermille := input.AssessmentBasePermille
	if assessmentBasePermille == 0 {
		assessmentBasePermille = 500
	}

	assessmentBase := taxBase * int64(assessmentBasePermille) / 1000
	if input.AssessmentBasePermille != 0 {
		// CSSZ expects the computed assessment base rounded up to whole CZK.
		assessmentBase = ceilToWholeCZK(assessmentBase)
	}

	// minAssessmentBase = minMonthlyBase * months of activity.
	minAssessmentBase := int64(input.MinMonthlyBase) * months

	// finalBase = max(assessmentBase, minAssessmentBase).
	finalBase := assessmentBase
	if minAssessmentBase > finalBase {
		finalBase = minAssessmentBase
	}
	if input.MaxAnnualBase > 0 && finalBase > int64(input.MaxAnnualBase) {
		finalBase = int64(input.MaxAnnualBase)
	}

	totalInsurance := finalBase * int64(input.RatePermille) / 1000
	if input.AssessmentBasePermille != 0 || input.RoundPremiumUp {
		// The premium is rounded up to whole CZK: CSSZ form line D32.1 (social)
		// and § 2 odst. 2 zakona c. 592/1992 Sb. (health).
		totalInsurance = ceilToWholeCZK(totalInsurance)
	}

	// difference = totalInsurance - prepayments.
	difference := totalInsurance - int64(input.Prepayments)

	rate := int64(input.RatePermille)
	var newMonthlyPrepay int64
	switch {
	case input.AssessmentBasePermille != 0:
		// CSSZ: the monthly base is the annual base spread over the months of
		// activity, rounded up to whole CZK (§ 14 odst. 13 zakona c. 589/1992 Sb.);
		// the advance is the rate applied to it, rounded up to whole CZK.
		monthlyBase := ceilToWholeCZK(ceilDiv(finalBase, months))
		newMonthlyPrepay = ceilToWholeCZK(ceilDiv(monthlyBase*rate, 1000))
	case input.RoundPremiumUp:
		// Health: the premium is divided by the months of activity in a single
		// step, so the monthly base itself is not rounded to whole CZK.
		newMonthlyPrepay = ceilToWholeCZK(ceilDiv(finalBase*rate, 1000*months))
	default:
		monthlyHalere := ceilDiv(totalInsurance, months)
		newMonthlyPrepay = ((monthlyHalere + 99) / 100) * 100
	}

	return InsuranceResult{
		TaxBase:             domain.Amount(taxBase),
		AssessmentBase:      domain.Amount(assessmentBase),
		MinAssessmentBase:   domain.Amount(minAssessmentBase),
		FinalAssessmentBase: domain.Amount(finalBase),
		TotalInsurance:      domain.Amount(totalInsurance),
		Difference:          domain.Amount(difference),
		NewMonthlyPrepay:    domain.Amount(newMonthlyPrepay),
	}, nil
}

// ceilDiv divides non-negative a by positive b, rounding the result up.
func ceilDiv(a, b int64) int64 {
	if a <= 0 {
		return 0
	}
	return (a + b - 1) / b
}

func ceilToWholeCZK(halere int64) int64 {
	if halere <= 0 {
		return halere
	}
	if halere%100 == 0 {
		return halere
	}
	return ((halere / 100) + 1) * 100
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
