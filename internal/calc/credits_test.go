package calc

import (
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

// testConstants returns TaxYearConstants with known values for testing.
func testConstants() TaxYearConstants {
	return TaxYearConstants{
		SpouseCredit:              domain.NewAmount(24_840, 0),
		SpouseIncomeLimit:         domain.NewAmount(68_000, 0),
		StudentCredit:             domain.NewAmount(4_020, 0),
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
			got := ComputeSpouseCredit(tt.spouseIncome, tt.monthsClaimed, tt.spouseZTP, c)
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
		isStudent       bool
		studentMonths   int
		wantDisability  domain.Amount
		wantStudent     domain.Amount
	}{
		{
			name:            "disability level 0",
			disabilityLevel: 0,
			isStudent:       false,
			studentMonths:   0,
			wantDisability:  0,
			wantStudent:     0,
		},
		{
			name:            "disability level 1",
			disabilityLevel: 1,
			isStudent:       false,
			studentMonths:   0,
			wantDisability:  domain.NewAmount(2_520, 0),
			wantStudent:     0,
		},
		{
			name:            "disability level 2",
			disabilityLevel: 2,
			isStudent:       false,
			studentMonths:   0,
			wantDisability:  domain.NewAmount(5_040, 0),
			wantStudent:     0,
		},
		{
			name:            "disability level 3 ZTPP",
			disabilityLevel: 3,
			isStudent:       false,
			studentMonths:   0,
			wantDisability:  domain.NewAmount(16_140, 0),
			wantStudent:     0,
		},
		{
			name:            "student 12 months",
			disabilityLevel: 0,
			isStudent:       true,
			studentMonths:   12,
			wantDisability:  0,
			wantStudent:     domain.NewAmount(4_020, 0),
		},
		{
			name:            "student 6 months proportional",
			disabilityLevel: 0,
			isStudent:       true,
			studentMonths:   6,
			wantDisability:  0,
			wantStudent:     domain.NewAmount(2_010, 0),
		},
		{
			name:            "not student returns 0",
			disabilityLevel: 0,
			isStudent:       false,
			studentMonths:   6,
			wantDisability:  0,
			wantStudent:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotD, gotS := ComputePersonalCredits(tt.disabilityLevel, tt.isStudent, tt.studentMonths, c)
			if gotD != tt.wantDisability {
				t.Errorf("disability = %v, want %v", gotD, tt.wantDisability)
			}
			if gotS != tt.wantStudent {
				t.Errorf("student = %v, want %v", gotS, tt.wantStudent)
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
