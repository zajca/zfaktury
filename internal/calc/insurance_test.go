package calc

import (
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
			got := CalculateInsurance(tt.input)

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
			}
			got := CalculateInsurance(input)
			if got.NewMonthlyPrepay != tt.expectedMonthly {
				t.Errorf("NewMonthlyPrepay = %d, want %d", got.NewMonthlyPrepay, tt.expectedMonthly)
			}
		})
	}
}
