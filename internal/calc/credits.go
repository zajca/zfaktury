package calc

import "github.com/zajca/zfaktury/internal/domain"

// SpouseEffectiveMonths returns the months for which the spouse credit applies. From
// 2024 the claim is capped by the months at whose start a dependent child under 3
// lived in the common household (§ 35bb odst. 2); earlier years use the claimed months.
// Used both by the calc input and by the XML rows 65a/65b, so the rule lives here only.
func SpouseEffectiveMonths(year, monthsClaimed, childUnder3Months int) int {
	if year >= 2024 && childUnder3Months < monthsClaimed {
		return childUnder3Months
	}
	return monthsClaimed
}

// ComputeSpouseCredit computes the spouse tax credit.
// Returns 0 if spouse income is at or above the income limit.
// The credit is proportional to the effective months (out of 12); see
// SpouseEffectiveMonths for the 2024+ child-under-3 cap.
// If spouseZTP is true, the credit is doubled.
func ComputeSpouseCredit(spouseIncome domain.Amount, monthsClaimed, childUnder3Months, year int, spouseZTP bool, constants TaxYearConstants) domain.Amount {
	if spouseIncome >= constants.SpouseIncomeLimit {
		return 0
	}
	months := SpouseEffectiveMonths(year, monthsClaimed, childUnder3Months)
	credit := constants.SpouseCredit.Multiply(float64(months) / 12.0)
	if spouseZTP {
		credit = credit.Multiply(2)
	}
	return credit
}

// PersonalCreditsResult holds the personal credits per EPO row.
type PersonalCreditsResult struct {
	Disability12   domain.Amount // 1st/2nd degree (row 66)
	Disability3    domain.Amount // 3rd degree (row 67)
	DisabilityZTPP domain.Amount // ZTP/P holder (row 68)
	Student        domain.Amount
}

// Disability returns the sum of the three disability credits.
func (r PersonalCreditsResult) Disability() domain.Amount {
	return r.Disability12 + r.Disability3 + r.DisabilityZTPP
}

// ComputePersonalCredits computes disability and student credits.
// disabilityLevel is one of the domain.DisabilityLevel* constants; ztpp adds the
// ZTP/P credit on top of the degree credit.
// Each disability credit is proportional to its months out of 12 (§ 35ba odst. 3),
// the same pattern as the student credit.
func ComputePersonalCredits(disabilityLevel int, disabilityMonths int, ztpp bool, ztppMonths int, isStudent bool, studentMonths int, constants TaxYearConstants) PersonalCreditsResult {
	var res PersonalCreditsResult

	switch disabilityLevel {
	case domain.DisabilityLevel1And2:
		res.Disability12 = constants.DisabilityCredit1.Multiply(float64(disabilityMonths) / 12.0)
	case domain.DisabilityLevel3:
		res.Disability3 = constants.DisabilityCredit3.Multiply(float64(disabilityMonths) / 12.0)
	}
	if ztpp {
		res.DisabilityZTPP = constants.DisabilityZTPP.Multiply(float64(ztppMonths) / 12.0)
	}

	if isStudent && studentMonths > 0 {
		res.Student = constants.StudentCredit.Multiply(float64(studentMonths) / 12.0)
	}

	return res
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

// minChildBonusCZK is the minimum yearly daňový bonus that can be claimed
// (§ 35c odst. 3): a computed bonus of 1-99 CZK becomes 0.
const minChildBonusCZK = 100

// SplitChildBenefit splits the child benefit into the sleva part (reduces the
// tax, floored at 0) and the refundable daňový bonus, applying the two statutory
// conditions on the bonus:
//   - § 35c odst. 3: the bonus must be at least 100 CZK, otherwise it is 0;
//   - § 35c odst. 4 with § 21g odst. 1: the taxpayer's § 6 + § 7 income in the tax
//     year must reach 6x the monthly minimum wage effective on 1 January. "Income"
//     is the gross příjem (§ 6 gross income plus § 7 revenue), not the partial tax
//     base after expenses. Below the threshold the benefit only reduces the tax to
//     0 (sleva) and the bonus is 0.
//
// The sleva part is unaffected by both conditions. This is the single place that
// computes the limited split; the income tax calculation and the XML generator
// (ř.73/ř.76) both use it.
func SplitChildBenefit(taxAfterCredits, childBenefit, section6GrossIncome, section7Revenue domain.Amount, constants TaxYearConstants) (sleva, bonus domain.Amount) {
	if taxAfterCredits < 0 {
		taxAfterCredits = 0
	}
	sleva = childBenefit
	if sleva > taxAfterCredits {
		sleva = taxAfterCredits
	}
	excess := childBenefit - sleva
	if excess > 0 {
		income := section6GrossIncome + section7Revenue
		if income >= constants.MinimumWage*6 && excess >= domain.NewAmount(minChildBonusCZK, 0) {
			bonus = excess
		}
	}
	return sleva, bonus
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

// savingsPoolKey is an internal sentinel used to share a single cap across
// životní pojištění + penzijní spoření + DIP from 2024 (§ 15 odst. 5 ZDP).
const savingsPoolKey = "__savings_pool"

// ComputeDeductions computes allowed deduction amounts with statutory caps.
// Each category has a maximum cap; multiple deductions of the same category share the cap.
// Donation cap is 15% of the tax base.
//
// From 2024, životní pojištění (DeductionLifeInsurance) and penzijní spoření
// (DeductionPension) share a single combined cap of DeductionCapSavingsCombined
// (48 000 Kč) per § 15 odst. 5 ZDP. When that field is non-zero in constants,
// both categories draw from one shared pool. Pre-2024 callers that supply the
// legacy individual caps (DeductionCapLifeInsurance / DeductionCapPension) keep
// the original per-category behaviour.
func ComputeDeductions(deductions []DeductionInput, taxBase domain.Amount, constants TaxYearConstants) DeductionResult {
	useCombined := constants.DeductionCapSavingsCombined > 0

	categoryCaps := map[string]domain.Amount{
		domain.DeductionMortgage:  constants.DeductionCapMortgage,
		domain.DeductionUnionDues: constants.DeductionCapUnionDues,
		domain.DeductionDonation:  taxBase.Multiply(0.15),
	}
	if useCombined {
		categoryCaps[savingsPoolKey] = constants.DeductionCapSavingsCombined
	} else {
		categoryCaps[domain.DeductionLifeInsurance] = constants.DeductionCapLifeInsurance
		categoryCaps[domain.DeductionPension] = constants.DeductionCapPension
	}

	remainingCap := make(map[string]domain.Amount, len(categoryCaps))
	for k, v := range categoryCaps {
		remainingCap[k] = v
	}

	resolveKey := func(cat string) string {
		if useCombined && (cat == domain.DeductionLifeInsurance || cat == domain.DeductionPension) {
			return savingsPoolKey
		}
		return cat
	}

	result := DeductionResult{
		AllowedAmounts: make([]domain.Amount, len(deductions)),
	}

	for i, d := range deductions {
		key := resolveKey(d.Category)
		remaining, ok := remainingCap[key]
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

		remainingCap[key] = remaining - allowed
		result.AllowedAmounts[i] = allowed
		result.TotalAllowed += allowed
	}

	return result
}
