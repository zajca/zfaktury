package calc

import (
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

func constants2025() TaxYearConstants {
	c, _ := GetTaxConstants(2025)
	return c
}

func TestCalculateIncomeTax(t *testing.T) {
	c := constants2025()

	tests := []struct {
		name   string
		input  IncomeTaxInput
		expect IncomeTaxResult
	}{
		{
			name: "flat rate 60% with revenue 2M CZK (within cap)",
			input: IncomeTaxInput{
				TotalRevenue:    domain.NewAmount(2_000_000, 0), // 200_000_000 halere
				FlatRatePercent: 60,
				Constants:       c,
			},
			// FlatRateAmount = 2M * 0.6 = 1.2M CZK = 120_000_000 halere (== cap, not exceeded)
			// UsedExpenses = 120_000_000
			// TaxBase = 200_000_000 - 120_000_000 = 80_000_000 halere (800,000 CZK)
			// TaxBaseRounded = 80_000_000 (already divisible by 10000)
			// 80_000_000 <= 167_605_200 -> only 15%
			// TaxAt15 = 80_000_000 * 0.15 = 12_000_000
			// TotalTax = 12_000_000
			// CreditBasic = 3_084_000
			// TotalCredits = 3_084_000
			// TaxAfterCredits = 12_000_000 - 3_084_000 = 8_916_000
			// TaxAfterBenefit = 8_916_000
			// TaxDue = 8_916_000
			expect: IncomeTaxResult{
				FlatRateAmount:  domain.NewAmount(1_200_000, 0),
				UsedExpenses:    domain.NewAmount(1_200_000, 0),
				TaxBase:         domain.NewAmount(800_000, 0),
				TaxBaseRounded:  domain.NewAmount(800_000, 0),
				TaxAt15:         domain.NewAmount(120_000, 0),
				TaxAt23:         0,
				TotalTax:        domain.NewAmount(120_000, 0),
				CreditBasic:     domain.NewAmount(30_840, 0),
				TotalCredits:    domain.NewAmount(30_840, 0),
				TaxAfterCredits: domain.NewAmount(89_160, 0),
				TaxAfterBenefit: domain.NewAmount(89_160, 0),
				TaxDue:          domain.NewAmount(89_160, 0),
			},
		},
		{
			name: "flat rate 60% with revenue 3M CZK (capped at 1.2M)",
			input: IncomeTaxInput{
				TotalRevenue:    domain.NewAmount(3_000_000, 0), // 300_000_000 halere
				FlatRatePercent: 60,
				Constants:       c,
			},
			// FlatRateAmount = 3M * 0.6 = 1.8M, cap = 1.2M -> capped to 120_000_000
			// UsedExpenses = 120_000_000
			// TaxBase = 300_000_000 - 120_000_000 = 180_000_000 (1,800,000 CZK)
			// TaxBaseRounded = 180_000_000
			// 2025 threshold = 36 × 46 557 = 1 676 052 Kč = 167_605_200 halere
			// 180_000_000 > 167_605_200 -> progressive split
			// TaxAt15 = 167_605_200 * 0.15 = 25_140_780
			// TaxAt23 = (180_000_000 - 167_605_200) * 0.23 = 12_394_800 * 0.23 = 2_850_804
			// TotalTax = 25_140_780 + 2_850_804 = 27_991_584
			// CreditBasic = 3_084_000
			// TaxAfterCredits = 27_991_584 - 3_084_000 = 24_907_584
			expect: IncomeTaxResult{
				FlatRateAmount:  domain.NewAmount(1_200_000, 0),
				UsedExpenses:    domain.NewAmount(1_200_000, 0),
				TaxBase:         domain.NewAmount(1_800_000, 0),
				TaxBaseRounded:  domain.NewAmount(1_800_000, 0),
				TaxAt15:         domain.Amount(25_140_780),
				TaxAt23:         domain.Amount(2_850_804),
				TotalTax:        domain.Amount(25_140_780 + 2_850_804),
				CreditBasic:     domain.NewAmount(30_840, 0),
				TotalCredits:    domain.NewAmount(30_840, 0),
				TaxAfterCredits: domain.Amount(25_140_780 + 2_850_804 - 3_084_000),
				TaxAfterBenefit: domain.Amount(25_140_780 + 2_850_804 - 3_084_000),
				TaxDue:          domain.Amount(25_140_780 + 2_850_804 - 3_084_000),
			},
		},
		{
			name: "actual expenses (flatRatePercent = 0)",
			input: IncomeTaxInput{
				TotalRevenue:    domain.NewAmount(1_000_000, 0),
				ActualExpenses:  domain.NewAmount(600_000, 0),
				FlatRatePercent: 0,
				Constants:       c,
			},
			// FlatRateAmount = 0
			// UsedExpenses = 600_000 CZK = 60_000_000
			// TaxBase = 100_000_000 - 60_000_000 = 40_000_000 (400,000 CZK)
			// TaxBaseRounded = 40_000_000
			// TaxAt15 = 40_000_000 * 0.15 = 6_000_000
			// CreditBasic = 3_084_000
			// TaxAfterCredits = 6_000_000 - 3_084_000 = 2_916_000
			expect: IncomeTaxResult{
				FlatRateAmount:  0,
				UsedExpenses:    domain.NewAmount(600_000, 0),
				TaxBase:         domain.NewAmount(400_000, 0),
				TaxBaseRounded:  domain.NewAmount(400_000, 0),
				TaxAt15:         domain.NewAmount(60_000, 0),
				TaxAt23:         0,
				TotalTax:        domain.NewAmount(60_000, 0),
				CreditBasic:     domain.NewAmount(30_840, 0),
				TotalCredits:    domain.NewAmount(30_840, 0),
				TaxAfterCredits: domain.NewAmount(29_160, 0),
				TaxAfterBenefit: domain.NewAmount(29_160, 0),
				TaxDue:          domain.NewAmount(29_160, 0),
			},
		},
		{
			name: "tax base < 0 clamped to 0",
			input: IncomeTaxInput{
				TotalRevenue:    domain.NewAmount(100_000, 0),
				ActualExpenses:  domain.NewAmount(200_000, 0),
				FlatRatePercent: 0,
				Constants:       c,
			},
			// TaxBase = 10_000_000 - 20_000_000 = -10_000_000 -> clamped to 0
			// Everything else is 0, except CreditBasic
			// TaxAfterCredits = 0 - 3_084_000 -> clamped to 0
			expect: IncomeTaxResult{
				FlatRateAmount:  0,
				UsedExpenses:    domain.NewAmount(200_000, 0),
				TaxBase:         0,
				TaxBaseRounded:  0,
				TaxAt15:         0,
				TaxAt23:         0,
				TotalTax:        0,
				CreditBasic:     domain.NewAmount(30_840, 0),
				TotalCredits:    domain.NewAmount(30_840, 0),
				TaxAfterCredits: 0,
				TaxAfterBenefit: 0,
				TaxDue:          0,
			},
		},
		{
			name: "deductions reducing tax base to 0",
			input: IncomeTaxInput{
				TotalRevenue:    domain.NewAmount(500_000, 0),
				ActualExpenses:  domain.NewAmount(300_000, 0),
				FlatRatePercent: 0,
				Constants:       c,
				TotalDeductions: domain.NewAmount(300_000, 0), // more than taxBase of 200,000 CZK
			},
			// TaxBase = 50_000_000 - 30_000_000 = 20_000_000 (200,000 CZK)
			// After deductions: 20_000_000 - 30_000_000 = -10_000_000 -> clamped to 0
			expect: IncomeTaxResult{
				FlatRateAmount:  0,
				UsedExpenses:    domain.NewAmount(300_000, 0),
				TaxBase:         domain.NewAmount(200_000, 0),
				TaxBaseRounded:  0,
				TaxAt15:         0,
				TaxAt23:         0,
				TotalTax:        0,
				CreditBasic:     domain.NewAmount(30_840, 0),
				TotalCredits:    domain.NewAmount(30_840, 0),
				TaxAfterCredits: 0,
				TaxAfterBenefit: 0,
				TaxDue:          0,
			},
		},
		{
			name: "tax base rounding (123456789 halere -> 123450000)",
			input: IncomeTaxInput{
				TotalRevenue:    domain.Amount(123_456_789), // 1,234,567.89 CZK
				ActualExpenses:  0,
				FlatRatePercent: 0,
				Constants:       c,
			},
			// TaxBase = 123_456_789
			// TaxBaseRounded = (123_456_789 / 10000) * 10000 = 12345 * 10000 = 123_450_000
			// TaxAt15 = 123_450_000 * 0.15 = 18_517_500
			// CreditBasic = 3_084_000
			// TaxAfterCredits = 18_517_500 - 3_084_000 = 15_433_500
			expect: IncomeTaxResult{
				FlatRateAmount:  0,
				UsedExpenses:    0,
				TaxBase:         domain.Amount(123_456_789),
				TaxBaseRounded:  domain.Amount(123_450_000),
				TaxAt15:         domain.Amount(18_517_500),
				TaxAt23:         0,
				TotalTax:        domain.Amount(18_517_500),
				CreditBasic:     domain.NewAmount(30_840, 0),
				TotalCredits:    domain.NewAmount(30_840, 0),
				TaxAfterCredits: domain.Amount(18_517_500 - 3_084_000),
				TaxAfterBenefit: domain.Amount(18_517_500 - 3_084_000),
				TaxDue:          domain.Amount(18_517_500 - 3_084_000),
			},
		},
		{
			name: "progressive tax below threshold (only 15%)",
			input: IncomeTaxInput{
				TotalRevenue:    domain.NewAmount(500_000, 0),
				ActualExpenses:  domain.NewAmount(100_000, 0),
				FlatRatePercent: 0,
				Constants:       c,
			},
			// TaxBase = 50_000_000 - 10_000_000 = 40_000_000 (400,000 CZK)
			// 40_000_000 < 167_605_200 -> only 15%
			// TaxAt15 = 40_000_000 * 0.15 = 6_000_000
			expect: IncomeTaxResult{
				FlatRateAmount:  0,
				UsedExpenses:    domain.NewAmount(100_000, 0),
				TaxBase:         domain.NewAmount(400_000, 0),
				TaxBaseRounded:  domain.NewAmount(400_000, 0),
				TaxAt15:         domain.NewAmount(60_000, 0),
				TaxAt23:         0,
				TotalTax:        domain.NewAmount(60_000, 0),
				CreditBasic:     domain.NewAmount(30_840, 0),
				TotalCredits:    domain.NewAmount(30_840, 0),
				TaxAfterCredits: domain.NewAmount(29_160, 0),
				TaxAfterBenefit: domain.NewAmount(29_160, 0),
				TaxDue:          domain.NewAmount(29_160, 0),
			},
		},
		{
			name: "progressive tax above threshold (15% + 23%)",
			input: IncomeTaxInput{
				TotalRevenue:    domain.NewAmount(2_000_000, 0),
				ActualExpenses:  0,
				FlatRatePercent: 0,
				Constants:       c,
			},
			// TaxBase = 200_000_000
			// TaxBaseRounded = 200_000_000
			// 2025 threshold = 36 × 46 557 = 1 676 052 Kč = 167_605_200 halere
			// TaxAt15 = 167_605_200 * 0.15 = 25_140_780
			// TaxAt23 = (200_000_000 - 167_605_200) * 0.23 = 32_394_800 * 0.23 = 7_450_804
			// TotalTax = 25_140_780 + 7_450_804 = 32_591_584
			// CreditBasic = 3_084_000
			// TaxAfterCredits = 32_591_584 - 3_084_000 = 29_507_584
			expect: IncomeTaxResult{
				FlatRateAmount:  0,
				UsedExpenses:    0,
				TaxBase:         domain.NewAmount(2_000_000, 0),
				TaxBaseRounded:  domain.NewAmount(2_000_000, 0),
				TaxAt15:         domain.Amount(25_140_780),
				TaxAt23:         domain.Amount(7_450_804),
				TotalTax:        domain.Amount(32_591_584),
				CreditBasic:     domain.NewAmount(30_840, 0),
				TotalCredits:    domain.NewAmount(30_840, 0),
				TaxAfterCredits: domain.Amount(32_591_584 - 3_084_000),
				TaxAfterBenefit: domain.Amount(32_591_584 - 3_084_000),
				TaxDue:          domain.Amount(32_591_584 - 3_084_000),
			},
		},
		{
			name: "credits exceeding total tax -> TaxAfterCredits clamped to 0",
			input: IncomeTaxInput{
				TotalRevenue:     domain.NewAmount(50_000, 0), // 5_000_000 halere
				ActualExpenses:   0,
				FlatRatePercent:  0,
				Constants:        c,
				SpouseCredit:     domain.NewAmount(24_840, 0),
				DisabilityCredit: domain.NewAmount(2_520, 0),
				StudentCredit:    domain.NewAmount(4_020, 0),
			},
			// TaxBase = 5_000_000
			// TaxBaseRounded = 5_000_000
			// TaxAt15 = 5_000_000 * 0.15 = 750_000
			// CreditBasic = 3_084_000
			// TotalCredits = 3_084_000 + 2_484_000 + 252_000 + 402_000 = 6_222_000
			// TaxAfterCredits = 750_000 - 6_222_000 = -5_472_000 -> clamped to 0
			expect: IncomeTaxResult{
				FlatRateAmount:  0,
				UsedExpenses:    0,
				TaxBase:         domain.NewAmount(50_000, 0),
				TaxBaseRounded:  domain.NewAmount(50_000, 0),
				TaxAt15:         domain.Amount(750_000),
				TaxAt23:         0,
				TotalTax:        domain.Amount(750_000),
				CreditBasic:     domain.NewAmount(30_840, 0),
				TotalCredits:    domain.Amount(3_084_000 + 2_484_000 + 252_000 + 402_000),
				TaxAfterCredits: 0,
				TaxAfterBenefit: 0,
				TaxDue:          0,
			},
		},
		{
			name: "child benefit exceeding TaxAfterCredits -> negative TaxAfterBenefit",
			input: IncomeTaxInput{
				TotalRevenue:    domain.NewAmount(200_000, 0), // 20_000_000 halere
				ActualExpenses:  0,
				FlatRatePercent: 0,
				Constants:       c,
				ChildBenefit:    domain.NewAmount(15_204, 0), // 1_520_400 halere
			},
			// TaxBase = 20_000_000
			// TaxBaseRounded = 20_000_000
			// TaxAt15 = 20_000_000 * 0.15 = 3_000_000
			// CreditBasic = 3_084_000
			// TotalCredits = 3_084_000
			// TaxAfterCredits = 3_000_000 - 3_084_000 -> clamped to 0
			// TaxAfterBenefit = 0 - 1_520_400 = -1_520_400
			// TaxDue = -1_520_400
			expect: IncomeTaxResult{
				FlatRateAmount:  0,
				UsedExpenses:    0,
				TaxBase:         domain.NewAmount(200_000, 0),
				TaxBaseRounded:  domain.NewAmount(200_000, 0),
				TaxAt15:         domain.Amount(3_000_000),
				TaxAt23:         0,
				TotalTax:        domain.Amount(3_000_000),
				CreditBasic:     domain.NewAmount(30_840, 0),
				TotalCredits:    domain.NewAmount(30_840, 0),
				TaxAfterCredits: 0,
				TaxAfterBenefit: domain.Amount(-1_520_400),
				TaxDue:          domain.Amount(-1_520_400),
			},
		},
		{
			name: "prepayments exceeding tax -> negative TaxDue (refund)",
			input: IncomeTaxInput{
				TotalRevenue:    domain.NewAmount(1_000_000, 0),
				ActualExpenses:  domain.NewAmount(500_000, 0),
				FlatRatePercent: 0,
				Constants:       c,
				Prepayments:     domain.NewAmount(100_000, 0), // 10_000_000 halere
			},
			// TaxBase = 100_000_000 - 50_000_000 = 50_000_000
			// TaxBaseRounded = 50_000_000
			// TaxAt15 = 50_000_000 * 0.15 = 7_500_000
			// CreditBasic = 3_084_000
			// TaxAfterCredits = 7_500_000 - 3_084_000 = 4_416_000
			// TaxAfterBenefit = 4_416_000
			// TaxDue = 4_416_000 - 10_000_000 = -5_584_000 (refund)
			expect: IncomeTaxResult{
				FlatRateAmount:  0,
				UsedExpenses:    domain.NewAmount(500_000, 0),
				TaxBase:         domain.NewAmount(500_000, 0),
				TaxBaseRounded:  domain.NewAmount(500_000, 0),
				TaxAt15:         domain.Amount(7_500_000),
				TaxAt23:         0,
				TotalTax:        domain.Amount(7_500_000),
				CreditBasic:     domain.NewAmount(30_840, 0),
				TotalCredits:    domain.NewAmount(30_840, 0),
				TaxAfterCredits: domain.Amount(4_416_000),
				TaxAfterBenefit: domain.Amount(4_416_000),
				TaxDue:          domain.Amount(-5_584_000),
			},
		},
		{
			name: "full realistic 2025 scenario",
			input: IncomeTaxInput{
				TotalRevenue:     domain.NewAmount(1_500_000, 0), // 150_000_000 halere
				FlatRatePercent:  60,
				Constants:        c,
				SpouseCredit:     domain.NewAmount(24_840, 0), // 2_484_000 halere
				DisabilityCredit: 0,
				StudentCredit:    0,
				ChildBenefit:     domain.NewAmount(15_204, 0), // 1_520_400 halere (1 child)
				TotalDeductions:  domain.NewAmount(24_000, 0), // 2_400_000 halere (life insurance)
				Prepayments:      domain.NewAmount(50_000, 0), // 5_000_000 halere
				CapitalIncomeNet: domain.NewAmount(10_000, 0), // 1_000_000 halere
				OtherIncomeNet:   domain.NewAmount(5_000, 0),  // 500_000 halere
			},
			// FlatRateAmount = 1_500_000 * 0.6 = 900_000 CZK = 90_000_000 halere (within 1.2M cap)
			// UsedExpenses = 90_000_000
			// TaxBase = 150_000_000 - 90_000_000 + 1_000_000 + 500_000 = 61_500_000 (615,000 CZK)
			// After deductions: 61_500_000 - 2_400_000 = 59_100_000 (591,000 CZK)
			// TaxBaseRounded = 59_100_000 (already divisible by 10000)
			// 59_100_000 < 167_605_200 -> only 15%
			// TaxAt15 = 59_100_000 * 0.15 = 8_865_000
			// CreditBasic = 3_084_000
			// TotalCredits = 3_084_000 + 2_484_000 + 0 + 0 = 5_568_000
			// TaxAfterCredits = 8_865_000 - 5_568_000 = 3_297_000
			// TaxAfterBenefit = 3_297_000 - 1_520_400 = 1_776_600
			// TaxDue = 1_776_600 - 5_000_000 = -3_223_400 (refund)
			expect: IncomeTaxResult{
				FlatRateAmount:  domain.NewAmount(900_000, 0),
				UsedExpenses:    domain.NewAmount(900_000, 0),
				TaxBase:         domain.Amount(61_500_000),
				TaxBaseRounded:  domain.Amount(59_100_000),
				TaxAt15:         domain.Amount(8_865_000),
				TaxAt23:         0,
				TotalTax:        domain.Amount(8_865_000),
				CreditBasic:     domain.NewAmount(30_840, 0),
				TotalCredits:    domain.Amount(5_568_000),
				TaxAfterCredits: domain.Amount(3_297_000),
				TaxAfterBenefit: domain.Amount(1_776_600),
				TaxDue:          domain.Amount(-3_223_400),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateIncomeTax(tt.input)

			assertAmount(t, "FlatRateAmount", tt.expect.FlatRateAmount, got.FlatRateAmount)
			assertAmount(t, "UsedExpenses", tt.expect.UsedExpenses, got.UsedExpenses)
			assertAmount(t, "TaxBase", tt.expect.TaxBase, got.TaxBase)
			assertAmount(t, "TaxBaseRounded", tt.expect.TaxBaseRounded, got.TaxBaseRounded)
			assertAmount(t, "TaxAt15", tt.expect.TaxAt15, got.TaxAt15)
			assertAmount(t, "TaxAt23", tt.expect.TaxAt23, got.TaxAt23)
			assertAmount(t, "TotalTax", tt.expect.TotalTax, got.TotalTax)
			assertAmount(t, "CreditBasic", tt.expect.CreditBasic, got.CreditBasic)
			assertAmount(t, "TotalCredits", tt.expect.TotalCredits, got.TotalCredits)
			assertAmount(t, "TaxAfterCredits", tt.expect.TaxAfterCredits, got.TaxAfterCredits)
			assertAmount(t, "TaxAfterBenefit", tt.expect.TaxAfterBenefit, got.TaxAfterBenefit)
			assertAmount(t, "TaxDue", tt.expect.TaxDue, got.TaxDue)
		})
	}
}

func assertAmount(t *testing.T, field string, want, got domain.Amount) {
	t.Helper()
	if want != got {
		t.Errorf("%s: want %d (%s CZK), got %d (%s CZK)", field, want, want, got, got)
	}
}

// TestTaxDue_SubtractsSection6Advance verifies ř.91 (kc_zbyvpred = TaxDue)
// reflects employer-withheld §6 advances. With 600 000 Kč gross §6, full
// CreditBasic eats most of the tax; the 60 000 Kč withheld advance pulls
// TaxDue strongly negative (refund owed back to user).
func TestTaxDue_SubtractsSection6Advance(t *testing.T) {
	c := constants2025()
	input := IncomeTaxInput{
		Section6TaxBase:         domain.NewAmount(600_000, 0),
		Section6AdvanceWithheld: domain.NewAmount(60_000, 0),
		Constants:               c,
	}
	result := CalculateIncomeTax(input)
	// Tax = 600 000 × 0.15 = 90 000; minus CreditBasic 30 840 = 59 160 Kč
	// after credits. ř.91 = 59 160 − 0 (prepayments) − 60 000 (ř.84) = −840 Kč.
	wantDue := domain.NewAmount(59_160, 0) - domain.NewAmount(60_000, 0)
	if result.TaxDue != wantDue {
		t.Errorf("TaxDue = %d, want %d (%s CZK)", result.TaxDue, wantDue, wantDue)
	}
}

// TestTaxDue_SubtractsSection6Withholding verifies ř.91 reflects the
// §36/6 withholding amount (ř.87) when the user opted to include a
// vzor-12 Potvrzení in DAP.
func TestTaxDue_SubtractsSection6Withholding(t *testing.T) {
	c := constants2025()
	input := IncomeTaxInput{
		Section6TaxBase:             domain.NewAmount(200_000, 0),
		Section6WithholdingCredited: domain.NewAmount(15_000, 0),
		Constants:                   c,
	}
	result := CalculateIncomeTax(input)
	// Tax = 200 000 × 0.15 = 30 000; minus CreditBasic 30 840 → 0.
	// TaxDue = 0 − 0 − 0 − 15 000 = −15 000 Kč (refund).
	wantDue := -domain.NewAmount(15_000, 0)
	if result.TaxDue != wantDue {
		t.Errorf("TaxDue = %d, want %d (%s CZK)", result.TaxDue, wantDue, wantDue)
	}
}

// TestTaxDue_BonusOverpayReturnsToState verifies that when the employer
// already paid out more monthly bonus than the user is entitled to (ř.89 >
// ř.72), TaxDue increases — the user must return the excess to státu.
func TestTaxDue_BonusOverpayReturnsToState(t *testing.T) {
	c := constants2025()
	input := IncomeTaxInput{
		ChildBenefit:             domain.NewAmount(15_204, 0), // 1 dítě roční nárok
		Section6MonthlyBonusPaid: domain.NewAmount(18_000, 0), // employer paid more than entitled
		Constants:                c,
	}
	result := CalculateIncomeTax(input)
	// TaxAfterCredits = 0 (no tax base). TaxAfterBenefit = 0 − 15 204 = −15 204.
	// TaxDue = −15 204 − 0 − 0 − 0 + 18 000 = 2 796 Kč owed back to státu.
	wantDue := domain.NewAmount(18_000, 0) - domain.NewAmount(15_204, 0)
	if result.TaxDue != wantDue {
		t.Errorf("TaxDue = %d, want %d (%s CZK) — bonus over-pay must return to státu",
			result.TaxDue, wantDue, wantDue)
	}
}

// TestTaxDue_BonusUnderpayClaimsRest verifies that when the employer paid
// out less monthly bonus than the user is entitled to, TaxDue decreases —
// the user can still claim the remaining bonus on DAP.
func TestTaxDue_BonusUnderpayClaimsRest(t *testing.T) {
	c := constants2025()
	input := IncomeTaxInput{
		ChildBenefit:             domain.NewAmount(15_204, 0),
		Section6MonthlyBonusPaid: domain.NewAmount(10_000, 0),
		Constants:                c,
	}
	result := CalculateIncomeTax(input)
	// TaxAfterBenefit = −15 204. TaxDue = −15 204 + 10 000 = −5 204 (still
	// owed to user, but less than full claim because employer already paid 10k).
	wantDue := domain.NewAmount(10_000, 0) - domain.NewAmount(15_204, 0)
	if result.TaxDue != wantDue {
		t.Errorf("TaxDue = %d, want %d (%s CZK)", result.TaxDue, wantDue, wantDue)
	}
}
