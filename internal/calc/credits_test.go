package calc

import (
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

// testConstants returns TaxYearConstants with known values for testing.
// Uses pre-2024 individual savings caps (24 000 each) to keep legacy test cases
// stable; tests of the combined-cap path explicitly construct the 2024+ shape.
func testConstants() TaxYearConstants {
	return TaxYearConstants{
		SpouseCredit:              domain.NewAmount(24_840, 0),
		SpouseIncomeLimit:         domain.NewAmount(68_000, 0),
		StudentCredit:             domain.NewAmount(4_020, 0), // pre-2024 hodnota pro test
		DisabilityCredit1:         domain.NewAmount(2_520, 0),
		DisabilityCredit3:         domain.NewAmount(5_040, 0),
		DisabilityZTPP:            domain.NewAmount(16_140, 0),
		ChildBenefit1:             domain.NewAmount(15_204, 0),
		ChildBenefit2:             domain.NewAmount(22_320, 0),
		ChildBenefit3Plus:         domain.NewAmount(27_840, 0),
		DeductionCapMortgage:      domain.NewAmount(150_000, 0),
		DeductionCapLifeInsurance: domain.NewAmount(24_000, 0),
		DeductionCapPension:       domain.NewAmount(24_000, 0),
		DeductionCapUnionDues:     domain.NewAmount(3_000, 0),
	}
}

// testConstants2024 returns the 2024+ shape with combined savings cap.
func testConstants2024() TaxYearConstants {
	return TaxYearConstants{
		SpouseCredit:                domain.NewAmount(24_840, 0),
		SpouseIncomeLimit:           domain.NewAmount(68_000, 0),
		StudentCredit:               0, // sleva zrušena
		DisabilityCredit1:           domain.NewAmount(2_520, 0),
		DisabilityCredit3:           domain.NewAmount(5_040, 0),
		DisabilityZTPP:              domain.NewAmount(16_140, 0),
		ChildBenefit1:               domain.NewAmount(15_204, 0),
		ChildBenefit2:               domain.NewAmount(22_320, 0),
		ChildBenefit3Plus:           domain.NewAmount(27_840, 0),
		DeductionCapMortgage:        domain.NewAmount(150_000, 0),
		DeductionCapSavingsCombined: domain.NewAmount(48_000, 0),
		DeductionCapUnionDues:       domain.NewAmount(3_000, 0),
	}
}

func TestComputeSpouseCredit(t *testing.T) {
	c := testConstants()

	tests := []struct {
		name          string
		spouseIncome  domain.Amount
		monthsClaimed int
		spouseZTP     bool
		want          domain.Amount
	}{
		{
			name:          "income below limit, 12 months, no ZTP",
			spouseIncome:  domain.NewAmount(50_000, 0),
			monthsClaimed: 12,
			spouseZTP:     false,
			want:          domain.NewAmount(24_840, 0),
		},
		{
			name:          "income below limit, 6 months proportional",
			spouseIncome:  domain.NewAmount(50_000, 0),
			monthsClaimed: 6,
			spouseZTP:     false,
			want:          domain.NewAmount(12_420, 0),
		},
		{
			name:          "income below limit, ZTP doubled",
			spouseIncome:  domain.NewAmount(50_000, 0),
			monthsClaimed: 12,
			spouseZTP:     true,
			want:          domain.NewAmount(49_680, 0),
		},
		{
			name:          "income at limit returns 0",
			spouseIncome:  domain.NewAmount(68_000, 0),
			monthsClaimed: 12,
			spouseZTP:     false,
			want:          0,
		},
		{
			name:          "income above limit returns 0",
			spouseIncome:  domain.NewAmount(100_000, 0),
			monthsClaimed: 12,
			spouseZTP:     false,
			want:          0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pre-2024 year: the child-under-3 condition does not apply.
			got := ComputeSpouseCredit(tt.spouseIncome, tt.monthsClaimed, 0, 2023, tt.spouseZTP, c)
			if got != tt.want {
				t.Errorf("ComputeSpouseCredit() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestComputeSpouseCredit_ChildUnder3Months verifies the § 35bb odst. 2 rule for 2024+:
// the credit is capped by the months at whose start a dependent child under 3 lived in
// the common household; a ZTP/P spouse only doubles the effective amount.
func TestComputeSpouseCredit_ChildUnder3Months(t *testing.T) {
	c := testConstants2024()

	tests := []struct {
		name          string
		spouseIncome  domain.Amount
		monthsClaimed int
		childMonths   int
		year          int
		spouseZTP     bool
		want          domain.Amount
	}{
		{
			name:          "12 claimed / 6 child months gives 6/12",
			spouseIncome:  domain.NewAmount(50_000, 0),
			monthsClaimed: 12, childMonths: 6, year: 2025,
			want: domain.NewAmount(12_420, 0),
		},
		{
			name:          "ZTP/P spouse doubles the effective amount",
			spouseIncome:  domain.NewAmount(50_000, 0),
			monthsClaimed: 12, childMonths: 6, year: 2025, spouseZTP: true,
			want: domain.NewAmount(24_840, 0),
		},
		{
			name:          "child months above claimed keep the claimed months",
			spouseIncome:  domain.NewAmount(50_000, 0),
			monthsClaimed: 6, childMonths: 12, year: 2024,
			want: domain.NewAmount(12_420, 0),
		},
		{
			name:          "no child months computes 0 for 2024+",
			spouseIncome:  domain.NewAmount(50_000, 0),
			monthsClaimed: 12, childMonths: 0, year: 2025,
			want: 0,
		},
		{
			name:          "pre-2024 ignores child months",
			spouseIncome:  domain.NewAmount(50_000, 0),
			monthsClaimed: 12, childMonths: 0, year: 2023,
			want: domain.NewAmount(24_840, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeSpouseCredit(tt.spouseIncome, tt.monthsClaimed, tt.childMonths, tt.year, tt.spouseZTP, c)
			if got != tt.want {
				t.Errorf("ComputeSpouseCredit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputePersonalCredits(t *testing.T) {
	c := testConstants()

	tests := []struct {
		name            string
		disabilityLevel int
		ztpp            bool
		isStudent       bool
		studentMonths   int
		want12          domain.Amount
		want3           domain.Amount
		wantZTPP        domain.Amount
		wantSum         domain.Amount
		wantStudent     domain.Amount
	}{
		{name: "no disability"},
		{name: "level 1", disabilityLevel: 1, want12: domain.NewAmount(2_520, 0), wantSum: domain.NewAmount(2_520, 0)},
		{name: "level 2", disabilityLevel: 2, want3: domain.NewAmount(5_040, 0), wantSum: domain.NewAmount(5_040, 0)},
		{name: "ZTP/P only", ztpp: true, wantZTPP: domain.NewAmount(16_140, 0), wantSum: domain.NewAmount(16_140, 0)},
		{
			name: "level 2 with ZTP/P", disabilityLevel: 2, ztpp: true,
			want3: domain.NewAmount(5_040, 0), wantZTPP: domain.NewAmount(16_140, 0), wantSum: domain.NewAmount(21_180, 0),
		},
		{name: "student 12 months", isStudent: true, studentMonths: 12, wantStudent: domain.NewAmount(4_020, 0)},
		{name: "student 6 months proportional", isStudent: true, studentMonths: 6, wantStudent: domain.NewAmount(2_010, 0)},
		{name: "not student returns 0", studentMonths: 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputePersonalCredits(tt.disabilityLevel, 12, tt.ztpp, 12, tt.isStudent, tt.studentMonths, c)
			if got.Disability12 != tt.want12 || got.Disability3 != tt.want3 || got.DisabilityZTPP != tt.wantZTPP {
				t.Errorf("disability rows = %v/%v/%v, want %v/%v/%v",
					got.Disability12, got.Disability3, got.DisabilityZTPP, tt.want12, tt.want3, tt.wantZTPP)
			}
			if got.Disability() != tt.wantSum {
				t.Errorf("Disability() = %v, want %v", got.Disability(), tt.wantSum)
			}
			if got.Student != tt.wantStudent {
				t.Errorf("student = %v, want %v", got.Student, tt.wantStudent)
			}
		})
	}

	t.Run("student with zero rule-set constant is 0", func(t *testing.T) {
		got := ComputePersonalCredits(0, 0, false, 0, true, 12, TaxYearConstants{})
		if got.Student != 0 {
			t.Errorf("student = %v, want 0", got.Student)
		}
	})
}

// TestComputePersonalCredits_DisabilityMonths verifies that each disability credit is
// scaled by its months out of 12 (§ 35ba odst. 3), the same pattern as the student credit.
func TestComputePersonalCredits_DisabilityMonths(t *testing.T) {
	c := testConstants()

	tests := []struct {
		name             string
		disabilityLevel  int
		disabilityMonths int
		ztpp             bool
		ztppMonths       int
		want12           domain.Amount
		want3            domain.Amount
		wantZTPP         domain.Amount
	}{
		{name: "level 1/2 for 6 months", disabilityLevel: 1, disabilityMonths: 6, want12: domain.NewAmount(1_260, 0)},
		{name: "level 3 for 3 months", disabilityLevel: 2, disabilityMonths: 3, want3: domain.NewAmount(1_260, 0)},
		{name: "ZTP/P for 9 months", ztpp: true, ztppMonths: 9, wantZTPP: domain.NewAmount(12_105, 0)},
		{name: "level 1/2 full year", disabilityLevel: 1, disabilityMonths: 12, want12: domain.NewAmount(2_520, 0)},
		{name: "months ignored without a level or flag", disabilityMonths: 6, ztppMonths: 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputePersonalCredits(tt.disabilityLevel, tt.disabilityMonths, tt.ztpp, tt.ztppMonths, false, 0, c)
			if got.Disability12 != tt.want12 || got.Disability3 != tt.want3 || got.DisabilityZTPP != tt.wantZTPP {
				t.Errorf("disability rows = %v/%v/%v, want %v/%v/%v",
					got.Disability12, got.Disability3, got.DisabilityZTPP, tt.want12, tt.want3, tt.wantZTPP)
			}
		})
	}
}

func TestComputeChildBenefit(t *testing.T) {
	c := testConstants()

	tests := []struct {
		name     string
		children []ChildCreditInput
		want     domain.Amount
	}{
		{
			name: "1 child order 1, 12 months",
			children: []ChildCreditInput{
				{ChildOrder: 1, MonthsClaimed: 12, ZTP: false},
			},
			want: domain.NewAmount(15_204, 0),
		},
		{
			name: "2 children orders 1 and 2",
			children: []ChildCreditInput{
				{ChildOrder: 1, MonthsClaimed: 12, ZTP: false},
				{ChildOrder: 2, MonthsClaimed: 12, ZTP: false},
			},
			want: domain.NewAmount(15_204, 0) + domain.NewAmount(22_320, 0),
		},
		{
			name: "3 children orders 1, 2, 3+",
			children: []ChildCreditInput{
				{ChildOrder: 1, MonthsClaimed: 12, ZTP: false},
				{ChildOrder: 2, MonthsClaimed: 12, ZTP: false},
				{ChildOrder: 3, MonthsClaimed: 12, ZTP: false},
			},
			want: domain.NewAmount(15_204, 0) + domain.NewAmount(22_320, 0) + domain.NewAmount(27_840, 0),
		},
		{
			name: "child with ZTP doubled",
			children: []ChildCreditInput{
				{ChildOrder: 1, MonthsClaimed: 12, ZTP: true},
			},
			want: domain.NewAmount(30_408, 0),
		},
		{
			name: "child with 6 months proportional",
			children: []ChildCreditInput{
				{ChildOrder: 1, MonthsClaimed: 6, ZTP: false},
			},
			want: domain.NewAmount(7_602, 0),
		},
		{
			name:     "empty children list returns 0",
			children: nil,
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeChildBenefit(tt.children, c)
			if got != tt.want {
				t.Errorf("ComputeChildBenefit() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSplitChildBenefit covers the statutory limits on the refundable child bonus:
// § 35c odst. 3 (bonus >= 100 CZK) and § 35c odst. 4 + § 21g odst. 1 (§ 6 + § 7 gross
// income >= 6x the monthly minimum wage on 1 January; 2025: 6 x 20 800 = 124 800 CZK).
// The sleva part is never limited.
func TestSplitChildBenefit(t *testing.T) {
	c := constants2025() // MinimumWage 2025 = 20 800 CZK -> threshold 124 800 CZK
	if c.MinimumWage != domain.NewAmount(20_800, 0) {
		t.Fatalf("2025 MinimumWage = %d, want 20 800 CZK", c.MinimumWage)
	}

	tests := []struct {
		name           string
		taxAfterCredit domain.Amount
		childBenefit   domain.Amount
		section6Gross  domain.Amount
		section7Rev    domain.Amount
		wantSleva      domain.Amount
		wantBonus      domain.Amount
	}{
		{
			name:           "benefit below tax: all sleva, no bonus",
			taxAfterCredit: domain.NewAmount(10_000, 0),
			childBenefit:   domain.NewAmount(5_000, 0),
			section7Rev:    domain.NewAmount(200_000, 0),
			wantSleva:      domain.NewAmount(5_000, 0),
			wantBonus:      0,
		},
		{
			name:           "bonus 99 CZK below the 100 CZK minimum -> 0",
			taxAfterCredit: 0,
			childBenefit:   domain.NewAmount(99, 0),
			section7Rev:    domain.NewAmount(200_000, 0),
			wantSleva:      0,
			wantBonus:      0,
		},
		{
			name:           "bonus exactly 100 CZK -> paid",
			taxAfterCredit: 0,
			childBenefit:   domain.NewAmount(100, 0),
			section7Rev:    domain.NewAmount(200_000, 0),
			wantSleva:      0,
			wantBonus:      domain.NewAmount(100, 0),
		},
		{
			name:           "income 124 799 via §7 revenue -> bonus 0, sleva still eats the tax",
			taxAfterCredit: domain.NewAmount(62_865, 0),
			childBenefit:   domain.NewAmount(100_000, 0),
			section7Rev:    domain.NewAmount(124_799, 0),
			wantSleva:      domain.NewAmount(62_865, 0),
			wantBonus:      0,
		},
		{
			name:           "income exactly 124 800 via §7 revenue -> bonus paid",
			taxAfterCredit: domain.NewAmount(62_880, 0),
			childBenefit:   domain.NewAmount(100_000, 0),
			section7Rev:    domain.NewAmount(124_800, 0),
			wantSleva:      domain.NewAmount(62_880, 0),
			wantBonus:      domain.NewAmount(37_120, 0),
		},
		{
			name:           "§6 gross income counts toward the threshold",
			taxAfterCredit: 0,
			childBenefit:   domain.NewAmount(15_204, 0),
			section6Gross:  domain.NewAmount(130_000, 0),
			wantSleva:      0,
			wantBonus:      domain.NewAmount(15_204, 0),
		},
		{
			name:           "§6 + §7 combined just below threshold -> bonus 0",
			taxAfterCredit: 0,
			childBenefit:   domain.NewAmount(15_204, 0),
			section6Gross:  domain.NewAmount(100_000, 0),
			section7Rev:    domain.NewAmount(24_799, 0), // 124 799 total
			wantSleva:      0,
			wantBonus:      0,
		},
		{
			name:           "§6 + §7 combined at threshold -> bonus paid",
			taxAfterCredit: 0,
			childBenefit:   domain.NewAmount(15_204, 0),
			section6Gross:  domain.NewAmount(100_000, 0),
			section7Rev:    domain.NewAmount(24_800, 0), // 124 800 total
			wantSleva:      0,
			wantBonus:      domain.NewAmount(15_204, 0),
		},
		{
			name:           "negative tax after credits clamps to 0, whole benefit is excess",
			taxAfterCredit: domain.NewAmount(-5_000, 0),
			childBenefit:   domain.NewAmount(15_204, 0),
			section7Rev:    domain.NewAmount(200_000, 0),
			wantSleva:      0,
			wantBonus:      domain.NewAmount(15_204, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sleva, bonus := SplitChildBenefit(tt.taxAfterCredit, tt.childBenefit, tt.section6Gross, tt.section7Rev, c)
			if sleva != tt.wantSleva {
				t.Errorf("sleva = %d (%s CZK), want %d (%s CZK)", sleva, sleva, tt.wantSleva, tt.wantSleva)
			}
			if bonus != tt.wantBonus {
				t.Errorf("bonus = %d (%s CZK), want %d (%s CZK)", bonus, bonus, tt.wantBonus, tt.wantBonus)
			}
		})
	}
}

func TestComputeDeductions(t *testing.T) {
	c := testConstants()
	taxBase := domain.NewAmount(1_000_000, 0)

	tests := []struct {
		name             string
		deductions       []DeductionInput
		taxBase          domain.Amount
		wantAllowed      []domain.Amount
		wantTotalAllowed domain.Amount
	}{
		{
			name: "single mortgage below cap",
			deductions: []DeductionInput{
				{Category: domain.DeductionMortgage, ClaimedAmount: domain.NewAmount(100_000, 0)},
			},
			taxBase:          taxBase,
			wantAllowed:      []domain.Amount{domain.NewAmount(100_000, 0)},
			wantTotalAllowed: domain.NewAmount(100_000, 0),
		},
		{
			name: "single mortgage above cap is capped",
			deductions: []DeductionInput{
				{Category: domain.DeductionMortgage, ClaimedAmount: domain.NewAmount(200_000, 0)},
			},
			taxBase:          taxBase,
			wantAllowed:      []domain.Amount{domain.NewAmount(150_000, 0)},
			wantTotalAllowed: domain.NewAmount(150_000, 0),
		},
		{
			name: "two mortgages sharing cap, second gets remainder",
			deductions: []DeductionInput{
				{Category: domain.DeductionMortgage, ClaimedAmount: domain.NewAmount(100_000, 0)},
				{Category: domain.DeductionMortgage, ClaimedAmount: domain.NewAmount(100_000, 0)},
			},
			taxBase:          taxBase,
			wantAllowed:      []domain.Amount{domain.NewAmount(100_000, 0), domain.NewAmount(50_000, 0)},
			wantTotalAllowed: domain.NewAmount(150_000, 0),
		},
		{
			name: "donation cap is 15% of tax base",
			deductions: []DeductionInput{
				{Category: domain.DeductionDonation, ClaimedAmount: domain.NewAmount(200_000, 0)},
			},
			taxBase:          taxBase,
			wantAllowed:      []domain.Amount{domain.NewAmount(150_000, 0)},
			wantTotalAllowed: domain.NewAmount(150_000, 0),
		},
		{
			name:             "empty deductions returns 0",
			deductions:       nil,
			taxBase:          taxBase,
			wantAllowed:      []domain.Amount{},
			wantTotalAllowed: 0,
		},
		{
			name: "unknown category allows nothing",
			deductions: []DeductionInput{
				{Category: "nonexistent_category", ClaimedAmount: domain.NewAmount(50_000, 0)},
			},
			taxBase:          taxBase,
			wantAllowed:      []domain.Amount{0},
			wantTotalAllowed: 0,
		},
		{
			name: "negative claimed amount is clamped to 0",
			deductions: []DeductionInput{
				{Category: domain.DeductionMortgage, ClaimedAmount: domain.Amount(-500_000)},
			},
			taxBase:          taxBase,
			wantAllowed:      []domain.Amount{0},
			wantTotalAllowed: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeDeductions(tt.deductions, tt.taxBase, c)
			if len(result.AllowedAmounts) != len(tt.wantAllowed) {
				t.Fatalf("AllowedAmounts length = %d, want %d", len(result.AllowedAmounts), len(tt.wantAllowed))
			}
			for i, got := range result.AllowedAmounts {
				if got != tt.wantAllowed[i] {
					t.Errorf("AllowedAmounts[%d] = %v, want %v", i, got, tt.wantAllowed[i])
				}
			}
			if result.TotalAllowed != tt.wantTotalAllowed {
				t.Errorf("TotalAllowed = %v, want %v", result.TotalAllowed, tt.wantTotalAllowed)
			}
		})
	}
}

// TestComputeDeductions_CombinedSavingsCap verifies that from 2024 životní
// pojištění + penzijní spoření share a single 48 000 Kč pool per § 15 odst. 5
// ZDP and that mortgage / union / donation caps still apply independently.
func TestComputeDeductions_CombinedSavingsCap(t *testing.T) {
	c := testConstants2024()
	taxBase := domain.NewAmount(1_000_000, 0)

	tests := []struct {
		name             string
		deductions       []DeductionInput
		wantAllowed      []domain.Amount
		wantTotalAllowed domain.Amount
	}{
		{
			name: "life 30k + pension 30k -> capped at combined 48k (first claim 30k, second remainder 18k)",
			deductions: []DeductionInput{
				{Category: domain.DeductionLifeInsurance, ClaimedAmount: domain.NewAmount(30_000, 0)},
				{Category: domain.DeductionPension, ClaimedAmount: domain.NewAmount(30_000, 0)},
			},
			wantAllowed:      []domain.Amount{domain.NewAmount(30_000, 0), domain.NewAmount(18_000, 0)},
			wantTotalAllowed: domain.NewAmount(48_000, 0),
		},
		{
			name: "life 50k alone uses entire 48k pool",
			deductions: []DeductionInput{
				{Category: domain.DeductionLifeInsurance, ClaimedAmount: domain.NewAmount(50_000, 0)},
			},
			wantAllowed:      []domain.Amount{domain.NewAmount(48_000, 0)},
			wantTotalAllowed: domain.NewAmount(48_000, 0),
		},
		{
			name: "life 20k + pension 20k -> both fully allowed (40k < 48k pool)",
			deductions: []DeductionInput{
				{Category: domain.DeductionLifeInsurance, ClaimedAmount: domain.NewAmount(20_000, 0)},
				{Category: domain.DeductionPension, ClaimedAmount: domain.NewAmount(20_000, 0)},
			},
			wantAllowed:      []domain.Amount{domain.NewAmount(20_000, 0), domain.NewAmount(20_000, 0)},
			wantTotalAllowed: domain.NewAmount(40_000, 0),
		},
		{
			name: "mortgage independent of savings pool",
			deductions: []DeductionInput{
				{Category: domain.DeductionLifeInsurance, ClaimedAmount: domain.NewAmount(48_000, 0)},
				{Category: domain.DeductionMortgage, ClaimedAmount: domain.NewAmount(100_000, 0)},
			},
			wantAllowed:      []domain.Amount{domain.NewAmount(48_000, 0), domain.NewAmount(100_000, 0)},
			wantTotalAllowed: domain.NewAmount(148_000, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeDeductions(tt.deductions, taxBase, c)
			if len(result.AllowedAmounts) != len(tt.wantAllowed) {
				t.Fatalf("AllowedAmounts length = %d, want %d", len(result.AllowedAmounts), len(tt.wantAllowed))
			}
			for i, got := range result.AllowedAmounts {
				if got != tt.wantAllowed[i] {
					t.Errorf("AllowedAmounts[%d] = %v, want %v", i, got, tt.wantAllowed[i])
				}
			}
			if result.TotalAllowed != tt.wantTotalAllowed {
				t.Errorf("TotalAllowed = %v, want %v", result.TotalAllowed, tt.wantTotalAllowed)
			}
		})
	}
}
