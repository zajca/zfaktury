package calc

import (
	"fmt"

	"github.com/zajca/zfaktury/internal/domain"
)

// assessmentBasePermille is the share of the tax base used as the social
// insurance assessment base (55 %).
const assessmentBasePermille = 550

// CSSZOverviewInput holds the inputs of the ČSSZ annual overview (rows 20-32).
type CSSZOverviewInput struct {
	TaxBase                   domain.Amount // ř. 20, revenue minus used expenses, not below 0
	MainMonths                int           // ř. 21 h
	SecondaryMonths           int           // ř. 21 v
	MainMinMonthly            domain.Amount
	SecondaryMinMonthly       domain.Amount
	MaxAnnualBase             domain.Amount
	DecisiveAmount            domain.Amount  // rozhodná částka of a full year of secondary activity
	DecisiveAmountMonthlyStep domain.Amount  // reduction per month without secondary activity
	EmploymentBase            *domain.Amount // entered employment base, nil = none
	RatePermille              int
	Prepayments               domain.Amount
}

// CSSZOverviewResult holds rows 23-32 and the computed base of the advance
// (ř. 35 before the bounds of the payment year).
type CSSZOverviewResult struct {
	MonthlyTaxBase          domain.Amount  // ř. 23
	MainTaxBaseShare        domain.Amount  // ř. 24 h, mixed years only
	SecondaryTaxBaseShare   domain.Amount  // ř. 24 v, mixed years only
	MainAssessmentBase      domain.Amount  // ř. 25 h
	SecondaryAssessmentBase domain.Amount  // ř. 25 v, 0 when the secondary part does not participate
	MainPartialBase         domain.Amount  // ř. 26 h, mixed years only
	SecondaryPartialBase    domain.Amount  // ř. 26 v, mixed years only
	MinAssessmentBase       domain.Amount  // ř. 27
	FinalAssessmentBase     domain.Amount  // ř. 28
	ReportedEmploymentBase  *domain.Amount // ř. 29, nil = row empty
	SumBase                 domain.Amount  // ř. 30
	InsuredBase             domain.Amount  // ř. 31
	TotalInsurance          domain.Amount  // ř. 32.1
	Difference              domain.Amount  // TotalInsurance - Prepayments

	// SecondaryDecisiveAmount is the rozhodná částka for the months of secondary
	// activity; 0 without secondary months.
	SecondaryDecisiveAmount domain.Amount
	// SecondaryParticipates reports whether the secondary part reached the
	// decisive amount; true without secondary months.
	SecondaryParticipates bool
	// AdvanceComputedBase is the annual base the advance is spread over (ř. 35).
	AdvanceComputedBase domain.Amount
}

// SocialAdvanceComputedBase returns the 55 % assessment base of the tax base,
// rounded up to whole CZK. The advance base of ř. 35 is derived from it.
func SocialAdvanceComputedBase(taxBase domain.Amount) domain.Amount {
	return domain.Amount(ceilToWholeCZK(int64(taxBase) * assessmentBasePermille / 1000))
}

// CalculateCSSZOverview computes rows 23-32 of the ČSSZ annual overview for a
// year of main activity, secondary activity or both. All amounts are in halere.
func CalculateCSSZOverview(in CSSZOverviewInput) (CSSZOverviewResult, error) {
	if in.MainMonths < 0 || in.SecondaryMonths < 0 || in.MainMonths+in.SecondaryMonths < 1 || in.MainMonths+in.SecondaryMonths > 12 {
		return CSSZOverviewResult{}, fmt.Errorf("main months %d and secondary months %d must be non-negative and total 1-12: %w", in.MainMonths, in.SecondaryMonths, domain.ErrInvalidInput)
	}
	if in.EmploymentBase != nil {
		if *in.EmploymentBase < 0 || *in.EmploymentBase%domain.NewAmount(1, 0) != 0 {
			return CSSZOverviewResult{}, fmt.Errorf("employment assessment base %d must be whole CZK and not negative: %w", *in.EmploymentBase, domain.ErrInvalidInput)
		}
	}

	taxBase := int64(in.TaxBase)
	if taxBase < 0 {
		taxBase = 0
	}
	h, v := int64(in.MainMonths), int64(in.SecondaryMonths)
	maxBase := int64(in.MaxAnnualBase)
	capBase := func(base int64) int64 {
		if maxBase > 0 && base > maxBase {
			return maxBase
		}
		return base
	}

	res := CSSZOverviewResult{SecondaryParticipates: true}
	res.MonthlyTaxBase = domain.Amount((2*taxBase + h + v) / (2 * (h + v)))

	var decisive int64
	if v > 0 {
		decisive = int64(in.DecisiveAmount) - int64(in.DecisiveAmountMonthlyStep)*(12-v)
		res.SecondaryDecisiveAmount = domain.Amount(decisive)
	}

	var minBase, finalBase int64
	switch {
	case v == 0:
		res.MainAssessmentBase = SocialAdvanceComputedBase(domain.Amount(taxBase))
		minBase = int64(in.MainMinMonthly) * h
		finalBase = capBase(max(int64(res.MainAssessmentBase), minBase))
	case h == 0:
		res.SecondaryParticipates = taxBase >= decisive
		if res.SecondaryParticipates {
			res.SecondaryAssessmentBase = SocialAdvanceComputedBase(domain.Amount(taxBase))
			minBase = int64(in.SecondaryMinMonthly) * v
			finalBase = capBase(max(int64(res.SecondaryAssessmentBase), minBase))
		}
	default:
		mainShare := int64(res.MonthlyTaxBase) * h
		secondaryShare := int64(res.MonthlyTaxBase) * v
		res.MainTaxBaseShare = domain.Amount(mainShare)
		res.SecondaryTaxBaseShare = domain.Amount(secondaryShare)
		res.MainAssessmentBase = SocialAdvanceComputedBase(domain.Amount(mainShare))
		res.SecondaryParticipates = secondaryShare >= decisive
		partialMain := int64(in.MainMinMonthly) * h
		var partialSecondary int64
		if res.SecondaryParticipates {
			res.SecondaryAssessmentBase = SocialAdvanceComputedBase(domain.Amount(secondaryShare))
			partialSecondary = int64(in.SecondaryMinMonthly) * v
		}
		res.MainPartialBase = domain.Amount(partialMain)
		res.SecondaryPartialBase = domain.Amount(partialSecondary)
		// The Pokyny take the highest of three sums as the base of a mixed year:
		// computed main plus secondary (ř. 25), minimum main plus secondary (ř. 26)
		// and minimum main plus computed secondary.
		computed := int64(res.MainAssessmentBase) + int64(res.SecondaryAssessmentBase)
		minBase = capBase(partialMain + partialSecondary)
		finalBase = capBase(max(computed, partialMain+partialSecondary, partialMain+int64(res.SecondaryAssessmentBase)))
	}
	res.MinAssessmentBase = domain.Amount(minBase)
	res.FinalAssessmentBase = domain.Amount(finalBase)

	sumBase, insuredBase := finalBase, finalBase
	// A secondary activity below the decisive amount is not insured, and the
	// Pokyny want rows 25-32 empty, so the employment base is not reported either.
	if in.EmploymentBase != nil && maxBase > 0 && (h > 0 || res.SecondaryParticipates) {
		employment := int64(*in.EmploymentBase)
		if employment >= maxBase || finalBase+employment > maxBase {
			reported := *in.EmploymentBase
			res.ReportedEmploymentBase = &reported
			sumBase = finalBase + employment
			if sumBase > maxBase {
				insuredBase = max(0, maxBase-employment)
			}
		}
	}
	res.SumBase = domain.Amount(sumBase)
	res.InsuredBase = domain.Amount(insuredBase)

	total := ceilToWholeCZK(insuredBase * int64(in.RatePermille) / 1000)
	res.TotalInsurance = domain.Amount(total)
	res.Difference = domain.Amount(total - int64(in.Prepayments))
	res.AdvanceComputedBase = SocialAdvanceComputedBase(domain.Amount(taxBase))
	return res, nil
}
