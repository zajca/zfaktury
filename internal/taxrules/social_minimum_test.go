package taxrules

import (
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

func TestSocialMinimum_AnnualAndAdvanceBases(t *testing.T) {
	tests := []struct {
		name        string
		year        int
		annualBase  domain.Amount
		advanceBase map[time.Month]domain.Amount
	}{
		{
			name:        "2024 unchanged",
			year:        2024,
			annualBase:  domain.NewAmount(13_191, 0),
			advanceBase: map[time.Month]domain.Amount{time.January: domain.NewAmount(13_191, 0), time.December: domain.NewAmount(13_191, 0)},
		},
		{
			name:        "2025 unchanged",
			year:        2025,
			annualBase:  domain.NewAmount(16_295, 0),
			advanceBase: map[time.Month]domain.Amount{time.January: domain.NewAmount(16_295, 0), time.December: domain.NewAmount(16_295, 0)},
		},
		{
			name:       "2026 advances change in July, annual overview uses the lower base",
			year:       2026,
			annualBase: domain.NewAmount(17_139, 0),
			advanceBase: map[time.Month]domain.Amount{
				time.January:  domain.NewAmount(19_587, 0),
				time.June:     domain.NewAmount(19_587, 0),
				time.July:     domain.NewAmount(17_139, 0),
				time.December: domain.NewAmount(17_139, 0),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules, err := GetRuleSet(tt.year)
			if err != nil {
				t.Fatalf("GetRuleSet(%d) error: %v", tt.year, err)
			}
			if rules.Constants.SocialMinMonthly != tt.annualBase {
				t.Errorf("SocialMinMonthly = %d, want %d", rules.Constants.SocialMinMonthly, tt.annualBase)
			}
			if got := int64(rules.Constants.SocialMinMonthly) * 12; tt.year == 2026 && got != int64(domain.NewAmount(205_668, 0)) {
				t.Errorf("yearly minimum = %d, want %d", got, domain.NewAmount(205_668, 0))
			}
			for month, want := range tt.advanceBase {
				got, err := rules.Constants.SocialMinAdvanceBaseAt(NewDate(tt.year, month, 1))
				if err != nil {
					t.Fatalf("SocialMinAdvanceBaseAt(%s) error: %v", month, err)
				}
				if got != want {
					t.Errorf("advance base in %s = %d, want %d", month, got, want)
				}
			}
		})
	}
}

func TestSocialMinimum_2026MinimumAdvances(t *testing.T) {
	rules, err := GetRuleSet(2026)
	if err != nil {
		t.Fatalf("GetRuleSet(2026) error: %v", err)
	}
	// Minimum advance = base x 29.2 %, rounded up to whole CZK (CSSZ: 5 720 and 5 005 CZK).
	for date, want := range map[Date]domain.Amount{
		NewDate(2026, time.March, 15): domain.NewAmount(5_720, 0),
		NewDate(2026, time.August, 1): domain.NewAmount(5_005, 0),
	} {
		base, err := rules.Constants.SocialMinAdvanceBaseAt(date)
		if err != nil {
			t.Fatalf("SocialMinAdvanceBaseAt(%s) error: %v", date, err)
		}
		advance := int64(base) * int64(rules.Constants.SocialRate) / 1000
		advance = ((advance + 99) / 100) * 100
		if advance != int64(want) {
			t.Errorf("minimum advance on %s = %d, want %d", date, advance, want)
		}
	}
}

func TestValidateRuleSet_RejectsIncompleteAdvanceSchedule(t *testing.T) {
	rules, err := GetRuleSet(2026)
	if err != nil {
		t.Fatalf("GetRuleSet(2026) error: %v", err)
	}
	rules.Constants.SocialMinAdvanceBase = rules.Constants.SocialMinAdvanceBase[:1]
	if err := ValidateRuleSet(rules); err == nil {
		t.Fatal("ValidateRuleSet accepted an advance schedule that stops in June")
	}
}
