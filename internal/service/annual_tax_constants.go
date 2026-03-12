package service

import (
	"fmt"

	"github.com/zajca/zfaktury/internal/domain"
)

// TaxYearConstants holds tax computation constants for a specific year.
type TaxYearConstants struct {
	ProgressiveThreshold domain.Amount            // prah pro 23% sazbu (in halere)
	BasicCredit          domain.Amount            // sleva na poplatnika
	SpouseCredit         domain.Amount            // sleva na manzela/ku
	StudentCredit        domain.Amount            // student
	DisabilityCredit1    domain.Amount            // invalidita 1. a 2. stupen
	DisabilityCredit3    domain.Amount            // invalidita 3. stupen
	DisabilityZTPP       domain.Amount            // drzitel prukazu ZTP/P
	ChildBenefit1        domain.Amount            // 1. dite
	ChildBenefit2        domain.Amount            // 2. dite
	ChildBenefit3Plus    domain.Amount            // 3+ dite
	ChildBenefitZTP      domain.Amount            // ZTP prirazka (double)
	SocialMinMonthly     domain.Amount            // min mesicni vym. zaklad CSSZ
	SocialRate           int                      // permille*10, e.g. 292 = 29.2%
	HealthMinMonthly     domain.Amount            // min mesicni vym. zaklad ZP
	HealthRate           int                      // permille*10, e.g. 135 = 13.5%
	FlatRateCaps           map[int]domain.Amount    // percent -> max halere amount
	TimeTestYears          int                      // years to hold for time test exemption
	SecurityExemptionLimit domain.Amount             // max exempt amount per year (0 = no limit)
}

// taxConstantsDB stores year-specific tax constants.
var taxConstantsDB = map[int]TaxYearConstants{
	2024: {
		ProgressiveThreshold: domain.NewAmount(1_582_812, 0),
		BasicCredit:          domain.NewAmount(30_840, 0),
		SpouseCredit:         domain.NewAmount(24_840, 0),
		StudentCredit:        domain.NewAmount(4_020, 0),
		DisabilityCredit1:    domain.NewAmount(2_520, 0),
		DisabilityCredit3:    domain.NewAmount(5_040, 0),
		DisabilityZTPP:       domain.NewAmount(16_140, 0),
		ChildBenefit1:        domain.NewAmount(15_204, 0),
		ChildBenefit2:        domain.NewAmount(22_320, 0),
		ChildBenefit3Plus:    domain.NewAmount(27_840, 0),
		ChildBenefitZTP:      domain.NewAmount(0, 0), // doubled automatically
		SocialMinMonthly:     domain.NewAmount(11_024, 0),
		SocialRate:           292,
		HealthMinMonthly:     domain.NewAmount(10_081, 0),
		HealthRate:           135,
		FlatRateCaps: map[int]domain.Amount{
			80: domain.NewAmount(1_600_000, 0),
			60: domain.NewAmount(1_200_000, 0),
			40: domain.NewAmount(800_000, 0),
			30: domain.NewAmount(600_000, 0),
		},
		TimeTestYears:          3,
		SecurityExemptionLimit: domain.NewAmount(0, 0), // no limit before 2025
	},
	2025: {
		ProgressiveThreshold: domain.NewAmount(1_582_812, 0),
		BasicCredit:          domain.NewAmount(30_840, 0),
		SpouseCredit:         domain.NewAmount(24_840, 0),
		StudentCredit:        domain.NewAmount(4_020, 0),
		DisabilityCredit1:    domain.NewAmount(2_520, 0),
		DisabilityCredit3:    domain.NewAmount(5_040, 0),
		DisabilityZTPP:       domain.NewAmount(16_140, 0),
		ChildBenefit1:        domain.NewAmount(15_204, 0),
		ChildBenefit2:        domain.NewAmount(22_320, 0),
		ChildBenefit3Plus:    domain.NewAmount(27_840, 0),
		ChildBenefitZTP:      domain.NewAmount(0, 0),
		SocialMinMonthly:     domain.NewAmount(11_584, 0),
		SocialRate:           292,
		HealthMinMonthly:     domain.NewAmount(10_874, 0),
		HealthRate:           135,
		FlatRateCaps: map[int]domain.Amount{
			80: domain.NewAmount(1_600_000, 0),
			60: domain.NewAmount(1_200_000, 0),
			40: domain.NewAmount(800_000, 0),
			30: domain.NewAmount(600_000, 0),
		},
		TimeTestYears:          3,
		SecurityExemptionLimit: domain.NewAmount(100_000_000, 0), // 1M CZK
	},
	2026: {
		ProgressiveThreshold: domain.NewAmount(1_582_812, 0),
		BasicCredit:          domain.NewAmount(30_840, 0),
		SpouseCredit:         domain.NewAmount(24_840, 0),
		StudentCredit:        domain.NewAmount(4_020, 0),
		DisabilityCredit1:    domain.NewAmount(2_520, 0),
		DisabilityCredit3:    domain.NewAmount(5_040, 0),
		DisabilityZTPP:       domain.NewAmount(16_140, 0),
		ChildBenefit1:        domain.NewAmount(15_204, 0),
		ChildBenefit2:        domain.NewAmount(22_320, 0),
		ChildBenefit3Plus:    domain.NewAmount(27_840, 0),
		ChildBenefitZTP:      domain.NewAmount(0, 0),
		SocialMinMonthly:     domain.NewAmount(12_139, 0),
		SocialRate:           292,
		HealthMinMonthly:     domain.NewAmount(11_396, 0),
		HealthRate:           135,
		FlatRateCaps: map[int]domain.Amount{
			80: domain.NewAmount(1_600_000, 0),
			60: domain.NewAmount(1_200_000, 0),
			40: domain.NewAmount(800_000, 0),
			30: domain.NewAmount(600_000, 0),
		},
		TimeTestYears:          3,
		SecurityExemptionLimit: domain.NewAmount(100_000_000, 0), // 1M CZK
	},
}

// GetTaxConstants returns the tax constants for a given year.
func GetTaxConstants(year int) (TaxYearConstants, error) {
	c, ok := taxConstantsDB[year]
	if !ok {
		return TaxYearConstants{}, fmt.Errorf("no tax constants for year %d: %w", year, domain.ErrInvalidInput)
	}
	return c, nil
}
