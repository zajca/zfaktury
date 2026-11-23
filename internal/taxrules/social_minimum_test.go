package taxrules

import (
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

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
