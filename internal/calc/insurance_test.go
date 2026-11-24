package calc

import (
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

func TestCalculateInsurance(t *testing.T) {
	tests := []struct {
		name   string
		input  InsuranceInput
		expect InsuranceResult
	}{
		{
			name: "social rate 292, revenue above minimum",
			input: InsuranceInput{
				Revenue:        domain.NewAmount(1_000_000, 0), // 1M CZK = 100000000 halere
				UsedExpenses:   domain.NewAmount(600_000, 0),   // 600k CZK
				MinMonthlyBase: domain.NewAmount(11_024, 0),    // 2024 social min
				RatePermille:   292,
				ActivityMonths: 12,
				Prepayments:    domain.NewAmount(40_000, 0), // 40k CZK
			},
			// taxBase = 100000000 - 60000000 = 40000000
			// assessmentBase = 40000000 / 2 = 20000000
			// minBase = 1102400 * 12 = 13228800
			// finalBase = max(20000000, 13228800) = 20000000
			// total = 20000000 * 292 / 1000 = 5840000
			// diff = 5840000 - 4000000 = 1840000
			// monthly = 5840000/12 = 486666 rem 8 -> 486667 -> ((486667+99)/100)*100 = 486700
			expect: InsuranceResult{
				TaxBase:             domain.Amount(40_000_000),
				AssessmentBase:      domain.Amount(20_000_000),
				MinAssessmentBase:   domain.Amount(13_228_800),
				FinalAssessmentBase: domain.Amount(20_000_000),
				TotalInsurance:      domain.Amount(5_840_000),
				Difference:          domain.Amount(1_840_000),
				NewMonthlyPrepay:    domain.Amount(486_700),
			},
		},
		{
			name: "health rate 135, revenue above minimum",
			input: InsuranceInput{
				Revenue:        domain.NewAmount(1_000_000, 0),
				UsedExpenses:   domain.NewAmount(600_000, 0),
				MinMonthlyBase: domain.NewAmount(10_081, 0), // 2024 health min
				RatePermille:   135,
				ActivityMonths: 12,
				Prepayments:    domain.NewAmount(20_000, 0),
			},
			// taxBase = 40000000, assessment = 20000000
			// minBase = 1008100 * 12 = 12097200
			// finalBase = 20000000
			// total = 20000000 * 135 / 1000 = 2700000
			// diff = 2700000 - 2000000 = 700000
			// monthly = 2700000/12 = 225000 exact -> ((225000+99)/100)*100 = 225000
			expect: InsuranceResult{
				TaxBase:             domain.Amount(40_000_000),
				AssessmentBase:      domain.Amount(20_000_000),
				MinAssessmentBase:   domain.Amount(12_097_200),
				FinalAssessmentBase: domain.Amount(20_000_000),
				TotalInsurance:      domain.Amount(2_700_000),
				Difference:          domain.Amount(700_000),
				NewMonthlyPrepay:    domain.Amount(225_000),
			},
		},
		{
			name: "revenue below minimum, uses minimum base",
			input: InsuranceInput{
				Revenue:        domain.NewAmount(100_000, 0),
				UsedExpenses:   domain.NewAmount(80_000, 0),
				MinMonthlyBase: domain.NewAmount(11_024, 0),
				RatePermille:   292,
				ActivityMonths: 12,
				Prepayments:    0,
			},
			// taxBase = 10000000 - 8000000 = 2000000
			// assessment = 1000000
			// minBase = 13228800
			// finalBase = 13228800
			// total = 13228800 * 292 / 1000 = 3862809 (integer: 13228800*292 = 3862809600, /1000 = 3862809)
			// diff = 3862809
			// monthly = 3862809/12 = 321900 rem 9 -> 321901 -> ((321901+99)/100)*100 = 322000
			expect: InsuranceResult{
				TaxBase:             domain.Amount(2_000_000),
				AssessmentBase:      domain.Amount(1_000_000),
				MinAssessmentBase:   domain.Amount(13_228_800),
				FinalAssessmentBase: domain.Amount(13_228_800),
				TotalInsurance:      domain.Amount(3_862_809),
				Difference:          domain.Amount(3_862_809),
				NewMonthlyPrepay:    domain.Amount(322_000),
			},
		},
		{
			name: "revenue = 0, uses minimum base",
			input: InsuranceInput{
				Revenue:        0,
				UsedExpenses:   0,
				MinMonthlyBase: domain.NewAmount(11_024, 0),
				RatePermille:   292,
				ActivityMonths: 12,
				Prepayments:    0,
			},
			// taxBase = 0, assessment = 0
			// minBase = 13228800, finalBase = 13228800
			// total = 3862809, diff = 3862809
			// monthly = 322000
			expect: InsuranceResult{
				TaxBase:             0,
				AssessmentBase:      0,
				MinAssessmentBase:   domain.Amount(13_228_800),
				FinalAssessmentBase: domain.Amount(13_228_800),
				TotalInsurance:      domain.Amount(3_862_809),
				Difference:          domain.Amount(3_862_809),
				NewMonthlyPrepay:    domain.Amount(322_000),
			},
		},
		{
			name: "expenses exceed revenue, taxBase clamped to 0",
			input: InsuranceInput{
				Revenue:        domain.NewAmount(100_000, 0),
				UsedExpenses:   domain.NewAmount(200_000, 0),
				MinMonthlyBase: domain.NewAmount(11_024, 0),
				RatePermille:   292,
				ActivityMonths: 12,
				Prepayments:    0,
			},
			// taxBase = 10000000 - 20000000 = -10000000 -> clamped to 0
			// assessment = 0
			// minBase = 13228800, finalBase = 13228800
			// total = 3862809, diff = 3862809
			// monthly = 322000
			expect: InsuranceResult{
				TaxBase:             0,
				AssessmentBase:      0,
				MinAssessmentBase:   domain.Amount(13_228_800),
				FinalAssessmentBase: domain.Amount(13_228_800),
				TotalInsurance:      domain.Amount(3_862_809),
				Difference:          domain.Amount(3_862_809),
				NewMonthlyPrepay:    domain.Amount(322_000),
			},
		},
		{
			name: "prepayments > totalInsurance, negative difference",
			input: InsuranceInput{
				Revenue:        domain.NewAmount(200_000, 0),
				UsedExpenses:   domain.NewAmount(100_000, 0),
				MinMonthlyBase: domain.NewAmount(11_024, 0),
				RatePermille:   292,
				Prepayments:    domain.NewAmount(500_000, 0), // 50000000 > total
				ActivityMonths: 12,
			},
			// taxBase = 20000000 - 10000000 = 10000000
			// assessment = 5000000
			// minBase = 13228800, finalBase = 13228800
			// total = 3862809
			// diff = 3862809 - 50000000 = -46137191
			expect: InsuranceResult{
				TaxBase:             domain.Amount(10_000_000),
				AssessmentBase:      domain.Amount(5_000_000),
				MinAssessmentBase:   domain.Amount(13_228_800),
				FinalAssessmentBase: domain.Amount(13_228_800),
				TotalInsurance:      domain.Amount(3_862_809),
				Difference:          domain.Amount(-46_137_191),
				NewMonthlyPrepay:    domain.Amount(322_000),
			},
		},
		{
			name: "prepayments = 0, revenue above minimum",
			input: InsuranceInput{
				Revenue:        domain.NewAmount(500_000, 0),
				UsedExpenses:   domain.NewAmount(200_000, 0),
				MinMonthlyBase: domain.NewAmount(11_024, 0),
				RatePermille:   292,
				ActivityMonths: 12,
				Prepayments:    0,
			},
			// taxBase = 50000000 - 20000000 = 30000000
			// assessment = 15000000
			// minBase = 13228800, finalBase = 15000000
			// total = 15000000 * 292 / 1000 = 4380000
			// diff = 4380000
			// monthly = 4380000/12 = 365000 exact -> 365000
			expect: InsuranceResult{
				TaxBase:             domain.Amount(30_000_000),
				AssessmentBase:      domain.Amount(15_000_000),
				MinAssessmentBase:   domain.Amount(13_228_800),
				FinalAssessmentBase: domain.Amount(15_000_000),
				TotalInsurance:      domain.Amount(4_380_000),
				Difference:          domain.Amount(4_380_000),
				NewMonthlyPrepay:    domain.Amount(365_000),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateInsurance(tt.input)
			if err != nil {
				t.Fatalf("CalculateInsurance() error: %v", err)
			}

			if got.TaxBase != tt.expect.TaxBase {
				t.Errorf("TaxBase = %d, want %d", got.TaxBase, tt.expect.TaxBase)
			}
			if got.AssessmentBase != tt.expect.AssessmentBase {
				t.Errorf("AssessmentBase = %d, want %d", got.AssessmentBase, tt.expect.AssessmentBase)
			}
			if got.MinAssessmentBase != tt.expect.MinAssessmentBase {
				t.Errorf("MinAssessmentBase = %d, want %d", got.MinAssessmentBase, tt.expect.MinAssessmentBase)
			}
			if got.FinalAssessmentBase != tt.expect.FinalAssessmentBase {
				t.Errorf("FinalAssessmentBase = %d, want %d", got.FinalAssessmentBase, tt.expect.FinalAssessmentBase)
			}
			if got.TotalInsurance != tt.expect.TotalInsurance {
				t.Errorf("TotalInsurance = %d, want %d", got.TotalInsurance, tt.expect.TotalInsurance)
			}
			if got.Difference != tt.expect.Difference {
				t.Errorf("Difference = %d, want %d", got.Difference, tt.expect.Difference)
			}
			if got.NewMonthlyPrepay != tt.expect.NewMonthlyPrepay {
				t.Errorf("NewMonthlyPrepay = %d, want %d", got.NewMonthlyPrepay, tt.expect.NewMonthlyPrepay)
			}
		})
	}
}

// TestCalculateInsuranceMonthlyRounding verifies the ceiling + round-up-to-CZK logic.
func TestCalculateInsuranceMonthlyRounding(t *testing.T) {
	tests := []struct {
		name            string
		totalInsurance  int64 // desired totalInsurance in halere
		expectedMonthly domain.Amount
	}{
		{
			name:            "exact division by 12",
			totalInsurance:  120_000, // 120000/12 = 10000 -> ((10000+99)/100)*100 = 10000
			expectedMonthly: domain.Amount(10_000),
		},
		{
			name:            "not divisible by 12, rounds up to CZK",
			totalInsurance:  130_000, // 130000/12 = 10833 rem 4 -> 10834 -> ((10834+99)/100)*100 = 10900
			expectedMonthly: domain.Amount(10_900),
		},
		{
			name:            "zero total insurance",
			totalInsurance:  0,
			expectedMonthly: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Construct input so totalInsurance = finalBase * 1000 / 1000 = finalBase.
			// With rate=1000: total = finalBase * 1000/1000 = finalBase.
			// With MinMonthlyBase=0, finalBase = assessmentBase = taxBase/2.
			// taxBase = revenue - 0 = revenue, so revenue = 2 * finalBase = 2 * tt.totalInsurance.
			input := InsuranceInput{
				Revenue:        domain.Amount(tt.totalInsurance * 2),
				UsedExpenses:   0,
				MinMonthlyBase: 0,
				RatePermille:   1000,
				Prepayments:    0,
				ActivityMonths: 12,
			}
			got, err := CalculateInsurance(input)
			if err != nil {
				t.Fatalf("CalculateInsurance() error: %v", err)
			}
			if got.NewMonthlyPrepay != tt.expectedMonthly {
				t.Errorf("NewMonthlyPrepay = %d, want %d", got.NewMonthlyPrepay, tt.expectedMonthly)
			}
		})
	}
}

// The 2026 health minimum is stored as 24 483,50 CZK (50 % of 48 967 CZK), so
// the yearly minimum base is 293 802 CZK. 13,5 % of it is 39 663,27 CZK, which
// is rounded up to 39 664 CZK (§ 2 odst. 2 zakona c. 592/1992 Sb.).
func TestCalculateInsuranceHealthMinimum2026Halere(t *testing.T) {
	c, err := GetTaxConstants(2026)
	if err != nil {
		t.Fatalf("GetTaxConstants(2026) error: %v", err)
	}
	got, err := CalculateInsurance(InsuranceInput{
		MinMonthlyBase: c.HealthMinMonthly,
		RatePermille:   c.HealthRate,
		RoundPremiumUp: true,
		ActivityMonths: 12,
	})
	if err != nil {
		t.Fatalf("CalculateInsurance() error: %v", err)
	}
	if got.MinAssessmentBase != domain.NewAmount(293_802, 0) {
		t.Errorf("MinAssessmentBase = %s, want 293802.00", got.MinAssessmentBase)
	}
	if got.TotalInsurance != domain.NewAmount(39_664, 0) {
		t.Errorf("TotalInsurance = %s, want 39664.00", got.TotalInsurance)
	}
}

func TestCalculateInsuranceSocialAssessmentBase55Percent(t *testing.T) {
	got, err := CalculateInsurance(InsuranceInput{
		Revenue:                domain.NewAmount(1_550_848, 0),
		UsedExpenses:           domain.NewAmount(930_508, 80),
		MinMonthlyBase:         domain.NewAmount(16_295, 0),
		AssessmentBasePermille: 550,
		RatePermille:           292,
		Prepayments:            domain.NewAmount(86_991, 0),
		ActivityMonths:         12,
	})
	if err != nil {
		t.Fatalf("CalculateInsurance() error: %v", err)
	}

	if got.TaxBase != domain.NewAmount(620_339, 20) {
		t.Errorf("TaxBase = %s, want 620339.20", got.TaxBase)
	}
	if got.AssessmentBase != domain.NewAmount(341_187, 0) {
		t.Errorf("AssessmentBase = %s, want 341187.00", got.AssessmentBase)
	}
	if got.FinalAssessmentBase != domain.NewAmount(341_187, 0) {
		t.Errorf("FinalAssessmentBase = %s, want 341187.00", got.FinalAssessmentBase)
	}
	if got.TotalInsurance != domain.NewAmount(99_627, 0) {
		t.Errorf("TotalInsurance = %s, want 99627.00", got.TotalInsurance)
	}
	if got.NewMonthlyPrepay != domain.NewAmount(8_303, 0) {
		t.Errorf("NewMonthlyPrepay = %s, want 8303.00", got.NewMonthlyPrepay)
	}
}

// CSSZ caps the annual assessment base at 48x the average wage; health
// insurance passes MaxAnnualBase 0 and therefore stays uncapped.
func TestCalculateInsuranceSocialMaxBase(t *testing.T) {
	tests := []struct {
		name                 string
		input                InsuranceInput
		wantAssessmentBase   domain.Amount
		wantFinalBase        domain.Amount
		wantTotalInsurance   domain.Amount
		wantNewMonthlyPrepay domain.Amount
	}{
		{
			name: "assessment base capped at the social maximum",
			input: InsuranceInput{
				Revenue:                domain.NewAmount(5_000_000, 0),
				UsedExpenses:           domain.NewAmount(0, 0),
				MinMonthlyBase:         domain.NewAmount(16_295, 0),
				MaxAnnualBase:          domain.NewAmount(2_234_736, 0),
				AssessmentBasePermille: 550,
				RatePermille:           292,
				ActivityMonths:         12,
			},
			wantAssessmentBase:   domain.NewAmount(2_750_000, 0),
			wantFinalBase:        domain.NewAmount(2_234_736, 0),
			wantTotalInsurance:   domain.NewAmount(652_543, 0),
			wantNewMonthlyPrepay: domain.NewAmount(54_379, 0),
		},
		{
			name: "assessment base below the maximum is unchanged",
			input: InsuranceInput{
				Revenue:                domain.NewAmount(1_000_000, 0),
				UsedExpenses:           domain.NewAmount(0, 0),
				MinMonthlyBase:         domain.NewAmount(16_295, 0),
				MaxAnnualBase:          domain.NewAmount(2_234_736, 0),
				AssessmentBasePermille: 550,
				RatePermille:           292,
				ActivityMonths:         12,
			},
			wantAssessmentBase:   domain.NewAmount(550_000, 0),
			wantFinalBase:        domain.NewAmount(550_000, 0),
			wantTotalInsurance:   domain.NewAmount(160_600, 0),
			wantNewMonthlyPrepay: domain.NewAmount(13_384, 0),
		},
		{
			name: "zero maximum means no cap",
			input: InsuranceInput{
				Revenue:                domain.NewAmount(5_000_000, 0),
				UsedExpenses:           domain.NewAmount(0, 0),
				MinMonthlyBase:         domain.NewAmount(16_295, 0),
				MaxAnnualBase:          domain.NewAmount(0, 0),
				AssessmentBasePermille: 550,
				RatePermille:           292,
				ActivityMonths:         12,
			},
			wantAssessmentBase:   domain.NewAmount(2_750_000, 0),
			wantFinalBase:        domain.NewAmount(2_750_000, 0),
			wantTotalInsurance:   domain.NewAmount(803_000, 0),
			wantNewMonthlyPrepay: domain.NewAmount(66_917, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateInsurance(tt.input)
			if err != nil {
				t.Fatalf("CalculateInsurance() error: %v", err)
			}
			if got.AssessmentBase != tt.wantAssessmentBase {
				t.Errorf("AssessmentBase = %s, want %s", got.AssessmentBase, tt.wantAssessmentBase)
			}
			if got.FinalAssessmentBase != tt.wantFinalBase {
				t.Errorf("FinalAssessmentBase = %s, want %s", got.FinalAssessmentBase, tt.wantFinalBase)
			}
			if got.TotalInsurance != tt.wantTotalInsurance {
				t.Errorf("TotalInsurance = %s, want %s", got.TotalInsurance, tt.wantTotalInsurance)
			}
			if got.NewMonthlyPrepay != tt.wantNewMonthlyPrepay {
				t.Errorf("NewMonthlyPrepay = %s, want %s", got.NewMonthlyPrepay, tt.wantNewMonthlyPrepay)
			}
		})
	}
}

// The minimum annual base is the minimum monthly base times the months of
// activity, and the advance spreads the annual premium over those months. CSSZ
// rounds the monthly base up to whole CZK before applying the rate (§ 14 odst.
// 13 zakona c. 589/1992 Sb.); health insurance divides the annual premium over
// the months of activity in one step.
func TestCalculateInsuranceActivityMonths(t *testing.T) {
	tests := []struct {
		name                 string
		input                InsuranceInput
		wantAssessmentBase   domain.Amount
		wantMinBase          domain.Amount
		wantFinalBase        domain.Amount
		wantTotalInsurance   domain.Amount
		wantNewMonthlyPrepay domain.Amount
	}{
		{
			name: "social full year",
			input: InsuranceInput{
				Revenue:                domain.NewAmount(1_000_000, 0),
				UsedExpenses:           domain.NewAmount(0, 0),
				MinMonthlyBase:         domain.NewAmount(16_295, 0),
				MaxAnnualBase:          domain.NewAmount(2_234_736, 0),
				AssessmentBasePermille: 550,
				RatePermille:           292,
				ActivityMonths:         12,
			},
			wantAssessmentBase:   domain.NewAmount(550_000, 0),
			wantMinBase:          domain.NewAmount(195_540, 0),
			wantFinalBase:        domain.NewAmount(550_000, 0),
			wantTotalInsurance:   domain.NewAmount(160_600, 0),
			wantNewMonthlyPrepay: domain.NewAmount(13_384, 0),
		},
		{
			name: "social six months spreads the same premium over six advances",
			input: InsuranceInput{
				Revenue:                domain.NewAmount(1_000_000, 0),
				UsedExpenses:           domain.NewAmount(0, 0),
				MinMonthlyBase:         domain.NewAmount(16_295, 0),
				MaxAnnualBase:          domain.NewAmount(2_234_736, 0),
				AssessmentBasePermille: 550,
				RatePermille:           292,
				ActivityMonths:         6,
			},
			wantAssessmentBase:   domain.NewAmount(550_000, 0),
			wantMinBase:          domain.NewAmount(97_770, 0),
			wantFinalBase:        domain.NewAmount(550_000, 0),
			wantTotalInsurance:   domain.NewAmount(160_600, 0),
			wantNewMonthlyPrepay: domain.NewAmount(26_767, 0),
		},
		{
			name: "social six months below the six-month minimum uses the minimum",
			input: InsuranceInput{
				Revenue:                domain.NewAmount(100_000, 0),
				UsedExpenses:           domain.NewAmount(0, 0),
				MinMonthlyBase:         domain.NewAmount(16_295, 0),
				MaxAnnualBase:          domain.NewAmount(2_234_736, 0),
				AssessmentBasePermille: 550,
				RatePermille:           292,
				ActivityMonths:         6,
			},
			wantAssessmentBase:   domain.NewAmount(55_000, 0),
			wantMinBase:          domain.NewAmount(97_770, 0),
			wantFinalBase:        domain.NewAmount(97_770, 0),
			wantTotalInsurance:   domain.NewAmount(28_549, 0),
			wantNewMonthlyPrepay: domain.NewAmount(4_759, 0),
		},
		{
			name: "health full year",
			input: InsuranceInput{
				Revenue:        domain.NewAmount(1_000_000, 0),
				UsedExpenses:   domain.NewAmount(0, 0),
				MinMonthlyBase: domain.NewAmount(24_483, 50),
				RatePermille:   135,
				RoundPremiumUp: true,
				ActivityMonths: 12,
			},
			wantAssessmentBase:   domain.NewAmount(500_000, 0),
			wantMinBase:          domain.NewAmount(293_802, 0),
			wantFinalBase:        domain.NewAmount(500_000, 0),
			wantTotalInsurance:   domain.NewAmount(67_500, 0),
			wantNewMonthlyPrepay: domain.NewAmount(5_625, 0),
		},
		{
			name: "health six months",
			input: InsuranceInput{
				Revenue:        domain.NewAmount(1_000_000, 0),
				UsedExpenses:   domain.NewAmount(0, 0),
				MinMonthlyBase: domain.NewAmount(24_483, 50),
				RatePermille:   135,
				RoundPremiumUp: true,
				ActivityMonths: 6,
			},
			wantAssessmentBase:   domain.NewAmount(500_000, 0),
			wantMinBase:          domain.NewAmount(146_901, 0),
			wantFinalBase:        domain.NewAmount(500_000, 0),
			wantTotalInsurance:   domain.NewAmount(67_500, 0),
			wantNewMonthlyPrepay: domain.NewAmount(11_250, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateInsurance(tt.input)
			if err != nil {
				t.Fatalf("CalculateInsurance() error: %v", err)
			}
			if got.AssessmentBase != tt.wantAssessmentBase {
				t.Errorf("AssessmentBase = %s, want %s", got.AssessmentBase, tt.wantAssessmentBase)
			}
			if got.MinAssessmentBase != tt.wantMinBase {
				t.Errorf("MinAssessmentBase = %s, want %s", got.MinAssessmentBase, tt.wantMinBase)
			}
			if got.FinalAssessmentBase != tt.wantFinalBase {
				t.Errorf("FinalAssessmentBase = %s, want %s", got.FinalAssessmentBase, tt.wantFinalBase)
			}
			if got.TotalInsurance != tt.wantTotalInsurance {
				t.Errorf("TotalInsurance = %s, want %s", got.TotalInsurance, tt.wantTotalInsurance)
			}
			if got.NewMonthlyPrepay != tt.wantNewMonthlyPrepay {
				t.Errorf("NewMonthlyPrepay = %s, want %s", got.NewMonthlyPrepay, tt.wantNewMonthlyPrepay)
			}
		})
	}
}

func TestCalculateInsuranceActivityMonthsInvalid(t *testing.T) {
	for _, months := range []int{0, 13} {
		_, err := CalculateInsurance(InsuranceInput{
			Revenue:                domain.NewAmount(1_000_000, 0),
			UsedExpenses:           domain.NewAmount(0, 0),
			MinMonthlyBase:         domain.NewAmount(16_295, 0),
			AssessmentBasePermille: 550,
			RatePermille:           292,
			ActivityMonths:         months,
		})
		if err == nil {
			t.Errorf("CalculateInsurance(months=%d) error = nil, want ErrInvalidInput", months)
			continue
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("CalculateInsurance(months=%d) error = %v, want ErrInvalidInput", months, err)
		}
	}
}

func TestSocialAdvanceMonthlyBase(t *testing.T) {
	bounds2025 := &AdvanceBaseBounds{
		MinMonthly: domain.NewAmount(16_295, 0),
		MaxMonthly: domain.NewAmount(186_228, 0),
	}
	tests := []struct {
		name      string
		computed  domain.Amount
		final     domain.Amount
		months    int
		bounds    *AdvanceBaseBounds
		wantInCZK int64
	}{
		{
			name:      "no bounds spreads the final base",
			computed:  domain.NewAmount(500_000, 0),
			final:     domain.NewAmount(97_770, 0),
			months:    6,
			bounds:    nil,
			wantInCZK: 16_295,
		},
		{
			name:      "below the payment-year minimum is raised",
			computed:  domain.NewAmount(100_000, 0),
			final:     domain.NewAmount(158_292, 0),
			months:    12,
			bounds:    bounds2025,
			wantInCZK: 16_295,
		},
		{
			name:      "above the payment-year maximum is capped",
			computed:  domain.NewAmount(3_000_000, 0),
			final:     domain.NewAmount(3_000_000, 0),
			months:    12,
			bounds:    bounds2025,
			wantInCZK: 186_228,
		},
		{
			name:      "inside the bounds stays unchanged",
			computed:  domain.NewAmount(600_000, 0),
			final:     domain.NewAmount(600_000, 0),
			months:    12,
			bounds:    bounds2025,
			wantInCZK: 50_000,
		},
		{
			name:      "uses the computed base, not the current-year minimum",
			computed:  domain.NewAmount(200_000, 0),
			final:     domain.NewAmount(195_540, 0),
			months:    12,
			bounds:    bounds2025,
			wantInCZK: 16_667,
		},
		{
			name:      "no months",
			computed:  domain.NewAmount(200_000, 0),
			final:     domain.NewAmount(200_000, 0),
			months:    0,
			bounds:    bounds2025,
			wantInCZK: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SocialAdvanceMonthlyBase(tt.computed, tt.final, tt.months, tt.bounds)
			want := domain.NewAmount(tt.wantInCZK, 0)
			if got != want {
				t.Errorf("SocialAdvanceMonthlyBase = %d, want %d", got, want)
			}
		})
	}
}

func TestSocialAdvancePeriods(t *testing.T) {
	czk := func(v int64) domain.Amount { return domain.NewAmount(v, 0) }
	period := func(from int, base, prepay int64) domain.AdvancePeriod {
		return domain.AdvancePeriod{FromMonth: from, MonthlyBase: czk(base), MonthlyPrepay: czk(prepay)}
	}
	rules := func(year int) *TaxYearConstants {
		c, err := GetTaxConstants(year)
		if err != nil {
			t.Fatalf("GetTaxConstants(%d) error: %v", year, err)
		}
		return &c
	}

	tests := []struct {
		name     string
		computed domain.Amount
		final    domain.Amount
		start    int
		payment  *TaxYearConstants
		want     []domain.AdvancePeriod
	}{
		{"minimum split from June", 0, czk(195_540), 6, rules(2026), []domain.AdvancePeriod{period(6, 19_587, 5_720), period(7, 17_139, 5_005)}},
		{"start in July", 0, czk(195_540), 7, rules(2026), []domain.AdvancePeriod{period(7, 17_139, 5_005)}},
		{"start in January", 0, czk(195_540), 1, rules(2026), []domain.AdvancePeriod{period(1, 19_587, 5_720), period(7, 17_139, 5_005)}},
		{"base between the minima", czk(216_000), czk(216_000), 6, rules(2026), []domain.AdvancePeriod{period(6, 19_587, 5_720), period(7, 18_000, 5_256)}},
		{"base between the minima, late start", czk(216_000), czk(216_000), 9, rules(2026), []domain.AdvancePeriod{period(9, 18_000, 5_256)}},
		{"above both minima merges", czk(300_000), czk(300_000), 6, rules(2026), []domain.AdvancePeriod{period(6, 25_000, 7_300)}},
		{"capped at the maximum", czk(3_000_000), czk(2_234_736), 6, rules(2026), []domain.AdvancePeriod{period(6, 195_868, 57_194)}},
		{"no payment-year rules", 0, czk(205_668), 6, nil, []domain.AdvancePeriod{period(6, 17_139, 5_005)}},
		{"single-period year", 0, czk(158_292), 6, rules(2025), []domain.AdvancePeriod{period(6, 16_295, 4_759)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SocialAdvancePeriods(tt.computed, tt.final, 12, tt.start, tt.payment, 292)
			if err != nil {
				t.Fatalf("SocialAdvancePeriods error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("periods = %+v, want %+v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("period %d = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}

	for _, start := range []int{0, 13} {
		if _, err := SocialAdvancePeriods(0, czk(195_540), 12, start, rules(2026), 292); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("start %d error = %v, want ErrInvalidInput", start, err)
		}
	}
}

func TestCalculateInsuranceWithPaymentYearBounds(t *testing.T) {
	result, err := CalculateInsurance(InsuranceInput{
		Revenue:                domain.NewAmount(100_000, 0),
		UsedExpenses:           domain.NewAmount(0, 0),
		MinMonthlyBase:         domain.NewAmount(16_295, 0),
		MaxAnnualBase:          domain.NewAmount(2_234_736, 0),
		AssessmentBasePermille: 550,
		RatePermille:           292,
		ActivityMonths:         12,
		AdvanceBounds: &AdvanceBaseBounds{
			MinMonthly: domain.NewAmount(16_295, 0),
			MaxMonthly: domain.NewAmount(186_228, 0),
		},
	})
	if err != nil {
		t.Fatalf("CalculateInsurance error: %v", err)
	}

	if want := domain.NewAmount(4_759, 0); result.NewMonthlyPrepay != want {
		t.Errorf("NewMonthlyPrepay = %d, want %d", result.NewMonthlyPrepay, want)
	}
}
