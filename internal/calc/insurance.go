package calc

import (
	"fmt"

	"github.com/zajca/zfaktury/internal/domain"
)

// InsuranceInput holds the parameters for an insurance calculation.
type InsuranceInput struct {
	Revenue        domain.Amount
	UsedExpenses   domain.Amount // already resolved (flat-rate or actual)
	MinMonthlyBase domain.Amount // e.g. HealthMinMonthly
	RatePermille   int           // 135 for health
	// RoundPremiumUp rounds the total premium up to whole CZK (health insurance,
	// § 2 odst. 2 zakona c. 592/1992 Sb.).
	RoundPremiumUp bool
	Prepayments    domain.Amount
	// ActivityMonths is the number of months of activity in the year (1-12).
	// It prorates the minimum assessment base; the maximum is not prorated.
	ActivityMonths int
	// MinimumExemptMonths is the number of months of activity in which the
	// minimum base does not apply; zero means the minimum applies in every month.
	MinimumExemptMonths int
	// AdvanceBounds carries the minimum of the year the health advance is paid
	// in; only MinMonthly is used, as the floor of the advance. Nil leaves the
	// advance without a floor.
	AdvanceBounds *AdvanceBaseBounds
}

// AdvanceBaseBounds holds the bounds of the monthly social insurance advance
// base for the year in which the advance is paid.
type AdvanceBaseBounds struct {
	MinMonthly domain.Amount
	MaxMonthly domain.Amount
}

// SocialAdvanceMonthlyBase returns the monthly social insurance advance base
// (ř. 35) from the computed annual base and the months of activity. Without
// bounds it spreads the final annual base over the months; with bounds it
// spreads the computed base and raises it to the payment-year minimum, then
// caps it at the payment-year maximum.
func SocialAdvanceMonthlyBase(computedAnnualBase, finalAnnualBase domain.Amount, months int, bounds *AdvanceBaseBounds) domain.Amount {
	if months <= 0 {
		return 0
	}
	monthCount := int64(months)
	if bounds == nil {
		return domain.Amount(ceilToWholeCZK(ceilDiv(int64(finalAnnualBase), monthCount)))
	}
	base := ceilToWholeCZK(ceilDiv(int64(computedAnnualBase), monthCount))
	if base < int64(bounds.MinMonthly) {
		base = int64(bounds.MinMonthly)
	}
	if bounds.MaxMonthly > 0 && base > int64(bounds.MaxMonthly) {
		base = int64(bounds.MaxMonthly)
	}
	return domain.Amount(base)
}

// SocialAdvanceBoundsForMonth returns the advance base bounds for an advance
// paid in the given month of the rule-set year.
func SocialAdvanceBoundsForMonth(c TaxYearConstants, month int) (*AdvanceBaseBounds, error) {
	minimum, err := c.SocialMinAdvanceBaseFor(month)
	if err != nil {
		return nil, fmt.Errorf("social advance bounds: %w", err)
	}
	bounds := &AdvanceBaseBounds{MinMonthly: minimum}
	if c.SocialMaxAnnualBase > 0 {
		bounds.MaxMonthly = domain.Amount(int64(c.SocialMaxAnnualBase) / 12)
	}
	return bounds, nil
}

// SocialAdvancePeriods returns the social advances due in the payment year
// from startMonth on, one per minimum period of the payment year (ř. 35-36);
// consecutive periods with the same base are merged. A secondary activity has
// one period bounded by the secondary minimum, which does not change within a
// year. Without payment-year rules it returns a single unbounded period.
func SocialAdvancePeriods(computedAnnualBase, finalAnnualBase domain.Amount, months, startMonth int, payment *TaxYearConstants, ratePermille int, secondary bool) ([]domain.AdvancePeriod, error) {
	if startMonth < 1 || startMonth > 12 {
		return nil, fmt.Errorf("advance start month %d out of range 1-12: %w", startMonth, domain.ErrInvalidInput)
	}
	if payment == nil {
		base := SocialAdvanceMonthlyBase(computedAnnualBase, finalAnnualBase, months, nil)
		return []domain.AdvancePeriod{{
			FromMonth:     startMonth,
			MonthlyBase:   base,
			MonthlyPrepay: SocialAdvanceFromMonthlyBase(base, ratePermille),
		}}, nil
	}
	if secondary {
		bounds := &AdvanceBaseBounds{
			MinMonthly: payment.SocialSecondaryMinMonthly,
			MaxMonthly: domain.Amount(int64(payment.SocialMaxAnnualBase) / 12),
		}
		base := SocialAdvanceMonthlyBase(computedAnnualBase, finalAnnualBase, months, bounds)
		return []domain.AdvancePeriod{{
			FromMonth:     startMonth,
			MonthlyBase:   base,
			MonthlyPrepay: SocialAdvanceFromMonthlyBase(base, ratePermille),
		}}, nil
	}

	var periods []domain.AdvancePeriod
	for i, p := range payment.SocialMinAdvanceBases {
		periodEnd := 12
		if i+1 < len(payment.SocialMinAdvanceBases) {
			periodEnd = payment.SocialMinAdvanceBases[i+1].FromMonth - 1
		}
		if periodEnd < startMonth {
			continue
		}
		from := max(p.FromMonth, startMonth)
		bounds, err := SocialAdvanceBoundsForMonth(*payment, from)
		if err != nil {
			return nil, err
		}
		base := SocialAdvanceMonthlyBase(computedAnnualBase, finalAnnualBase, months, bounds)
		if n := len(periods); n > 0 && periods[n-1].MonthlyBase == base {
			continue
		}
		periods = append(periods, domain.AdvancePeriod{
			FromMonth:     from,
			MonthlyBase:   base,
			MonthlyPrepay: SocialAdvanceFromMonthlyBase(base, ratePermille),
		})
	}
	return periods, nil
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
	if input.MinimumExemptMonths < 0 || input.MinimumExemptMonths > input.ActivityMonths {
		return InsuranceResult{}, fmt.Errorf("minimum exempt months %d out of range 0-%d: %w", input.MinimumExemptMonths, input.ActivityMonths, domain.ErrInvalidInput)
	}
	months := int64(input.ActivityMonths)

	// taxBase = revenue - usedExpenses, clamped to 0.
	taxBase := int64(input.Revenue) - int64(input.UsedExpenses)
	if taxBase < 0 {
		taxBase = 0
	}

	assessmentBase := taxBase * 500 / 1000

	// minAssessmentBase = minMonthlyBase * months of activity without an exemption
	// (§ 3a odst. 4 zakona c. 592/1992 Sb.).
	minAssessmentBase := int64(input.MinMonthlyBase) * (months - int64(input.MinimumExemptMonths))

	// finalBase = max(assessmentBase, minAssessmentBase).
	finalBase := assessmentBase
	if minAssessmentBase > finalBase {
		finalBase = minAssessmentBase
	}

	totalInsurance := finalBase * int64(input.RatePermille) / 1000
	if input.RoundPremiumUp {
		// § 2 odst. 2 zakona c. 592/1992 Sb.
		totalInsurance = ceilToWholeCZK(totalInsurance)
	}

	// difference = totalInsurance - prepayments.
	difference := totalInsurance - int64(input.Prepayments)

	rate := int64(input.RatePermille)
	var newMonthlyPrepay int64
	switch {
	case input.RoundPremiumUp:
		// Health: the premium is divided by the months of activity in a single
		// step, so the monthly base itself is not rounded to whole CZK.
		newMonthlyPrepay = ceilToWholeCZK(ceilDiv(finalBase*rate, 1000*months))
		if input.AdvanceBounds != nil {
			// The advance is not paid below the minimum advance of the payment year.
			floor := ceilToWholeCZK(ceilDiv(int64(input.AdvanceBounds.MinMonthly)*rate, 1000))
			if floor > newMonthlyPrepay {
				newMonthlyPrepay = floor
			}
		}
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

// SocialAdvanceFromMonthlyBase returns the monthly social insurance advance
// from the monthly assessment base (ř. 36): the rate applied to the base,
// rounded up to whole CZK.
func SocialAdvanceFromMonthlyBase(base domain.Amount, ratePermille int) domain.Amount {
	return domain.Amount(ceilToWholeCZK(ceilDiv(int64(base)*int64(ratePermille), 1000)))
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
