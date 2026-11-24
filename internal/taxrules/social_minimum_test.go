package taxrules

import (
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

func TestSocialSecondary_Constants(t *testing.T) {
	tests := []struct {
		year       int
		minMonthly domain.Amount
		decisive   domain.Amount
		step       domain.Amount
	}{
		{2024, domain.NewAmount(4_837, 0), domain.NewAmount(105_520, 0), domain.NewAmount(8_794, 0)},
		{2025, domain.NewAmount(5_122, 0), domain.NewAmount(111_736, 0), domain.NewAmount(9_312, 0)},
		{2026, domain.NewAmount(5_387, 0), domain.NewAmount(117_521, 0), domain.NewAmount(9_794, 0)},
	}
	for _, tt := range tests {
		rules, err := GetRuleSet(tt.year)
		if err != nil {
			t.Fatalf("GetRuleSet(%d) error: %v", tt.year, err)
		}
		c := rules.Constants
		if c.SocialSecondaryMinMonthly != tt.minMonthly {
			t.Errorf("%d SocialSecondaryMinMonthly = %d, want %d", tt.year, c.SocialSecondaryMinMonthly, tt.minMonthly)
		}
		if c.SocialDecisiveAmount != tt.decisive {
			t.Errorf("%d SocialDecisiveAmount = %d, want %d", tt.year, c.SocialDecisiveAmount, tt.decisive)
		}
		if c.SocialDecisiveAmountMonthlyStep != tt.step {
			t.Errorf("%d SocialDecisiveAmountMonthlyStep = %d, want %d", tt.year, c.SocialDecisiveAmountMonthlyStep, tt.step)
		}
		// The step is the decisive amount divided by 12 and rounded up to whole CZK.
		if diff := c.SocialDecisiveAmountMonthlyStep*12 - c.SocialDecisiveAmount; diff < 0 || diff > domain.NewAmount(11, 0) {
			t.Errorf("%d step*12 - amount = %d halere, want 0..11 CZK", tt.year, diff)
		}
	}
}

func TestValidateRuleSet_RejectsBadSocialSecondary(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*TaxYearConstants)
	}{
		{"zero minimum", func(c *TaxYearConstants) { c.SocialSecondaryMinMonthly = 0 }},
		{"zero decisive amount", func(c *TaxYearConstants) { c.SocialDecisiveAmount = 0 }},
		{"zero step", func(c *TaxYearConstants) { c.SocialDecisiveAmountMonthlyStep = 0 }},
		{"step too small", func(c *TaxYearConstants) { c.SocialDecisiveAmountMonthlyStep -= domain.NewAmount(1, 0) }},
		{"step too large", func(c *TaxYearConstants) { c.SocialDecisiveAmountMonthlyStep += domain.NewAmount(1, 0) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules, err := GetRuleSet(2025)
			if err != nil {
				t.Fatalf("GetRuleSet(2025) error: %v", err)
			}
			tt.mutate(&rules.Constants)
			if err := ValidateRuleSet(rules); !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("ValidateRuleSet error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

// The annual overview uses one minimum monthly base for all twelve months; for
// 2026 CSSZ recalculates January-June as if the July change applied from January.
func TestSocialMinimum_AnnualBase(t *testing.T) {
	tests := []struct {
		year       int
		monthly    domain.Amount
		wantYearly domain.Amount
	}{
		{2024, domain.NewAmount(13_191, 0), domain.NewAmount(158_292, 0)},
		{2025, domain.NewAmount(16_295, 0), domain.NewAmount(195_540, 0)},
		{2026, domain.NewAmount(17_139, 0), domain.NewAmount(205_668, 0)},
	}
	for _, tt := range tests {
		rules, err := GetRuleSet(tt.year)
		if err != nil {
			t.Fatalf("GetRuleSet(%d) error: %v", tt.year, err)
		}
		if rules.Constants.SocialMinMonthly != tt.monthly {
			t.Errorf("%d SocialMinMonthly = %d, want %d", tt.year, rules.Constants.SocialMinMonthly, tt.monthly)
		}
		if got := rules.Constants.SocialMinMonthly.Multiply(12); got != tt.wantYearly {
			t.Errorf("%d yearly minimum = %d, want %d", tt.year, got, tt.wantYearly)
		}
	}
}
