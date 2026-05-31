package taxrules

import (
	"errors"
	"testing"
	"time"

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
	rules.Deductions.Pension.MonthlyThreshold[0].Value = 1

	again, err := GetRuleSet(2025)
	if err != nil {
		t.Fatalf("GetRuleSet(2025) second call error: %v", err)
	}
	if got := again.Constants.FlatRateCaps[60]; got != domain.NewAmount(1_200_000, 0) {
		t.Errorf("GetRuleSet(2025).Constants.FlatRateCaps[60] after mutation = %d, want %d", got, domain.NewAmount(1_200_000, 0))
	}
	if got := again.Deductions.Pension.MonthlyThreshold[0].Value; got != domain.NewAmount(1_700, 0) {
		t.Errorf("GetRuleSet(2025).Pension.MonthlyThreshold[0] after mutation = %d, want %d", got, domain.NewAmount(1_700, 0))
	}
}

func TestSchedule_RejectsGapsAndOverlaps(t *testing.T) {
	periodStart := NewDate(2024, time.January, 1)
	periodEnd := NewDate(2024, time.December, 31)

	tests := []struct {
		name     string
		schedule Schedule[domain.Amount]
	}{
		{
			name: "gap between entries",
			schedule: Schedule[domain.Amount]{
				{From: NewDate(2024, time.January, 1), To: NewDate(2024, time.June, 30), Value: domain.NewAmount(1_000, 0)},
				{From: NewDate(2024, time.August, 1), To: NewDate(2024, time.December, 31), Value: domain.NewAmount(1_700, 0)},
			},
		},
		{
			name: "overlap between entries",
			schedule: Schedule[domain.Amount]{
				{From: NewDate(2024, time.January, 1), To: NewDate(2024, time.July, 31), Value: domain.NewAmount(1_000, 0)},
				{From: NewDate(2024, time.July, 1), To: NewDate(2024, time.December, 31), Value: domain.NewAmount(1_700, 0)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schedule.ValidateCoverage(periodStart, periodEnd)
			if err == nil {
				t.Fatalf("Schedule.ValidateCoverage(%v, %v) error = nil, want ErrInvalidInput", periodStart, periodEnd)
			}
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("Schedule.ValidateCoverage(%v, %v) error = %v, want ErrInvalidInput", periodStart, periodEnd, err)
			}
		})
	}
}

func TestSchedule_AllowsContiguousEntries(t *testing.T) {
	periodStart := NewDate(2024, time.January, 1)
	periodEnd := NewDate(2024, time.December, 31)
	schedule := Schedule[domain.Amount]{
		{From: NewDate(2024, time.January, 1), To: NewDate(2024, time.June, 30), Value: domain.NewAmount(1_000, 0)},
		{From: NewDate(2024, time.July, 1), To: NewDate(2024, time.December, 31), Value: domain.NewAmount(1_700, 0)},
	}

	if err := schedule.ValidateCoverage(periodStart, periodEnd); err != nil {
		t.Fatalf("Schedule.ValidateCoverage(%v, %v) error: %v", periodStart, periodEnd, err)
	}
}

func TestRuleSet_2024PensionMonthlyThresholdSchedule(t *testing.T) {
	rules, err := GetRuleSet(2024)
	if err != nil {
		t.Fatalf("GetRuleSet(2024) error: %v", err)
	}

	tests := []struct {
		at   Date
		want domain.Amount
	}{
		{at: NewDate(2024, time.January, 1), want: domain.NewAmount(1_000, 0)},
		{at: NewDate(2024, time.June, 30), want: domain.NewAmount(1_000, 0)},
		{at: NewDate(2024, time.July, 1), want: domain.NewAmount(1_700, 0)},
		{at: NewDate(2024, time.December, 31), want: domain.NewAmount(1_700, 0)},
	}

	for _, tt := range tests {
		got, err := rules.Deductions.Pension.MonthlyThreshold.ValueAt(tt.at)
		if err != nil {
			t.Errorf("ValueAt(%v) error: %v", tt.at, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ValueAt(%v) = %d, want %d", tt.at, got, tt.want)
		}
	}
}

func TestComputePensionDeduction_ContributionAtThresholdIsZero(t *testing.T) {
	rules, err := GetRuleSet(2024)
	if err != nil {
		t.Fatalf("GetRuleSet(2024) error: %v", err)
	}

	got, err := ComputePensionDeduction(PensionDeductionInput{
		MonthlyContributions: []PensionContribution{
			{Year: 2024, Month: time.December, PaidAmount: domain.NewAmount(1_700, 0)},
		},
	}, rules.Deductions.Pension)
	if err != nil {
		t.Fatalf("ComputePensionDeduction(Dec 2024 1700 CZK) error: %v", err)
	}
	if got.AllowedAmount != 0 {
		t.Errorf("ComputePensionDeduction(Dec 2024 1700 CZK).AllowedAmount = %d, want 0", got.AllowedAmount)
	}
	if len(got.Breakdown) != 1 {
		t.Fatalf("ComputePensionDeduction(Dec 2024 1700 CZK).Breakdown length = %d, want 1", len(got.Breakdown))
	}
	if got.Breakdown[0].Threshold != domain.NewAmount(1_700, 0) {
		t.Errorf("ComputePensionDeduction(Dec 2024 1700 CZK).Breakdown[0].Threshold = %d, want %d", got.Breakdown[0].Threshold, domain.NewAmount(1_700, 0))
	}
}
