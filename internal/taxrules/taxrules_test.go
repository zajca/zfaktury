package taxrules

import (
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

func TestRuleSets_ValidateSupportedYears(t *testing.T) {
	years := SupportedYears()
	if len(years) == 0 {
		t.Fatal("SupportedYears() is empty")
	}

	for _, year := range years {
		rules, err := GetRuleSet(year)
		if err != nil {
			t.Errorf("GetRuleSet(%d) error: %v", year, err)
			continue
		}
		if rules.Year != year {
			t.Errorf("GetRuleSet(%d).Year = %d, want %d", year, rules.Year, year)
		}
		if rules.ID == "" {
			t.Errorf("GetRuleSet(%d).ID is empty", year)
		}
		got, err := rules.Fingerprint()
		if err != nil {
			t.Errorf("GetRuleSet(%d).Fingerprint() error: %v", year, err)
			continue
		}
		if len(got) != 64 {
			t.Errorf("GetRuleSet(%d).Fingerprint() length = %d, want 64", year, len(got))
		}
	}
}

func TestGetRuleSet_UnsupportedYearsFailHard(t *testing.T) {
	for _, year := range []int{1999, 2023, 2099} {
		_, err := GetRuleSet(year)
		if err == nil {
			t.Fatalf("GetRuleSet(%d) error = nil, want ErrInvalidInput", year)
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("GetRuleSet(%d) error = %v, want ErrInvalidInput", year, err)
		}
	}
}

func TestGetRuleSet_ReturnsDefensiveCopy(t *testing.T) {
	rules, err := GetRuleSet(2025)
	if err != nil {
		t.Fatalf("GetRuleSet(2025) error: %v", err)
	}
	rules.Constants.FlatRateCaps[60] = 1
	rules.Constants.SocialMinAdvanceBases[0].Base = 1

	again, err := GetRuleSet(2025)
	if err != nil {
		t.Fatalf("GetRuleSet(2025) second call error: %v", err)
	}
	if got := again.Constants.FlatRateCaps[60]; got != domain.NewAmount(1_200_000, 0) {
		t.Errorf("GetRuleSet(2025).Constants.FlatRateCaps[60] after mutation = %d, want %d", got, domain.NewAmount(1_200_000, 0))
	}
	if got := again.Constants.SocialMinAdvanceBases[0].Base; got != domain.NewAmount(16_295, 0) {
		t.Errorf("GetRuleSet(2025).Constants.SocialMinAdvanceBases[0].Base after mutation = %d, want %d", got, domain.NewAmount(16_295, 0))
	}
}

func TestSocialMinAdvanceBaseFor(t *testing.T) {
	tests := []struct {
		year  int
		month int
		want  domain.Amount
	}{
		{year: 2024, month: 1, want: domain.NewAmount(13_191, 0)},
		{year: 2024, month: 12, want: domain.NewAmount(13_191, 0)},
		{year: 2025, month: 1, want: domain.NewAmount(16_295, 0)},
		{year: 2025, month: 12, want: domain.NewAmount(16_295, 0)},
		{year: 2026, month: 1, want: domain.NewAmount(19_587, 0)},
		{year: 2026, month: 6, want: domain.NewAmount(19_587, 0)},
		{year: 2026, month: 7, want: domain.NewAmount(17_139, 0)},
		{year: 2026, month: 12, want: domain.NewAmount(17_139, 0)},
	}

	for _, tt := range tests {
		rules, err := GetRuleSet(tt.year)
		if err != nil {
			t.Fatalf("GetRuleSet(%d) error: %v", tt.year, err)
		}
		got, err := rules.Constants.SocialMinAdvanceBaseFor(tt.month)
		if err != nil {
			t.Fatalf("SocialMinAdvanceBaseFor(%d) error: %v", tt.month, err)
		}
		if got != tt.want {
			t.Errorf("SocialMinAdvanceBaseFor(%d, month %d) = %d, want %d", tt.year, tt.month, got, tt.want)
		}
	}

	rules, err := GetRuleSet(2025)
	if err != nil {
		t.Fatalf("GetRuleSet(2025) error: %v", err)
	}
	for _, month := range []int{0, 13} {
		if _, err := rules.Constants.SocialMinAdvanceBaseFor(month); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("SocialMinAdvanceBaseFor(%d) error = %v, want ErrInvalidInput", month, err)
		}
	}
}

func TestValidateRuleSet_MinimumWage(t *testing.T) {
	for _, year := range SupportedYears() {
		rules, err := GetRuleSet(year)
		if err != nil {
			t.Fatalf("GetRuleSet(%d) error: %v", year, err)
		}
		if rules.Constants.MinimumWage <= 0 {
			t.Errorf("rule set %s has non-positive minimum wage %d", rules.ID, rules.Constants.MinimumWage)
		}
		if rules.Constants.MinChildBonus <= 0 {
			t.Errorf("rule set %s has non-positive minimum child bonus %d", rules.ID, rules.Constants.MinChildBonus)
		}
	}

	base, err := GetRuleSet(2025)
	if err != nil {
		t.Fatalf("GetRuleSet(2025) error: %v", err)
	}
	for _, mw := range []domain.Amount{0, -domain.NewAmount(20_800, 0)} {
		rules := base
		rules.Constants.MinimumWage = mw
		err := ValidateRuleSet(rules)
		if err == nil {
			t.Errorf("ValidateRuleSet() with minimum wage %d error = nil, want ErrInvalidInput", mw)
		} else if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("ValidateRuleSet() with minimum wage %d error = %v, want ErrInvalidInput", mw, err)
		}
	}
	for _, mb := range []domain.Amount{0, -domain.NewAmount(100, 0)} {
		rules := base
		rules.Constants.MinChildBonus = mb
		err := ValidateRuleSet(rules)
		if err == nil {
			t.Errorf("ValidateRuleSet() with minimum child bonus %d error = nil, want ErrInvalidInput", mb)
		} else if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("ValidateRuleSet() with minimum child bonus %d error = %v, want ErrInvalidInput", mb, err)
		}
	}
}

func TestValidateRuleSet_RejectsBadSocialMinAdvanceBases(t *testing.T) {
	tests := []struct {
		name    string
		periods []MonthlyBasePeriod
	}{
		{name: "empty", periods: nil},
		{name: "does not start in January", periods: []MonthlyBasePeriod{{FromMonth: 2, Base: domain.NewAmount(16_295, 0)}}},
		{name: "repeated month", periods: []MonthlyBasePeriod{{FromMonth: 1, Base: domain.NewAmount(16_295, 0)}, {FromMonth: 1, Base: domain.NewAmount(17_139, 0)}}},
		{name: "month out of range", periods: []MonthlyBasePeriod{{FromMonth: 1, Base: domain.NewAmount(16_295, 0)}, {FromMonth: 13, Base: domain.NewAmount(17_139, 0)}}},
		{name: "decreasing months", periods: []MonthlyBasePeriod{{FromMonth: 1, Base: domain.NewAmount(19_587, 0)}, {FromMonth: 7, Base: domain.NewAmount(17_139, 0)}, {FromMonth: 3, Base: domain.NewAmount(16_295, 0)}}},
		{name: "zero base", periods: []MonthlyBasePeriod{{FromMonth: 1, Base: domain.NewAmount(0, 0)}}},
		{name: "negative base", periods: []MonthlyBasePeriod{{FromMonth: 1, Base: domain.NewAmount(16_295, 0)}, {FromMonth: 7, Base: -domain.NewAmount(17_139, 0)}}},
	}

	base, err := GetRuleSet(2025)
	if err != nil {
		t.Fatalf("GetRuleSet(2025) error: %v", err)
	}
	if err := ValidateRuleSet(base); err != nil {
		t.Fatalf("ValidateRuleSet(2025) error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := base
			rules.Constants.SocialMinAdvanceBases = tt.periods

			err := ValidateRuleSet(rules)
			if err == nil {
				t.Fatal("ValidateRuleSet() error = nil, want ErrInvalidInput")
			}
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("ValidateRuleSet() error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestValidateRuleSet_RejectsMissingSocialMaxAnnualBase(t *testing.T) {
	tests := []struct {
		name  string
		value domain.Amount
	}{
		{name: "zero", value: domain.NewAmount(0, 0)},
		{name: "below the annual minimum base", value: domain.NewAmount(100_000, 0)},
	}

	base, err := GetRuleSet(2025)
	if err != nil {
		t.Fatalf("GetRuleSet(2025) error: %v", err)
	}
	if err := ValidateRuleSet(base); err != nil {
		t.Fatalf("ValidateRuleSet(2025) error: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := base
			rules.Constants.SocialMaxAnnualBase = tt.value

			err := ValidateRuleSet(rules)
			if err == nil {
				t.Fatalf("ValidateRuleSet() error = nil, want ErrInvalidInput")
			}
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("ValidateRuleSet() error = %v, want ErrInvalidInput", err)
			}
		})
	}
}
