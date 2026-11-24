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

	again, err := GetRuleSet(2025)
	if err != nil {
		t.Fatalf("GetRuleSet(2025) second call error: %v", err)
	}
	if got := again.Constants.FlatRateCaps[60]; got != domain.NewAmount(1_200_000, 0) {
		t.Errorf("GetRuleSet(2025).Constants.FlatRateCaps[60] after mutation = %d, want %d", got, domain.NewAmount(1_200_000, 0))
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
