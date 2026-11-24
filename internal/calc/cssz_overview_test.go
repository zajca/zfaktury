package calc

import (
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

// overview2025 returns the 2025 constants of the ČSSZ overview: main minimum
// 16 295, secondary minimum 5 122, rozhodná částka 111 736 with step 9 312,
// maximum 2 234 736 and rate 29,2 %.
func overview2025(taxBase domain.Amount, mainMonths, secondaryMonths int) CSSZOverviewInput {
	return CSSZOverviewInput{
		TaxBase:                   taxBase,
		MainMonths:                mainMonths,
		SecondaryMonths:           secondaryMonths,
		MainMinMonthly:            domain.NewAmount(16_295, 0),
		SecondaryMinMonthly:       domain.NewAmount(5_122, 0),
		MaxAnnualBase:             domain.NewAmount(2_234_736, 0),
		DecisiveAmount:            domain.NewAmount(111_736, 0),
		DecisiveAmountMonthlyStep: domain.NewAmount(9_312, 0),
		RatePermille:              292,
	}
}

func TestCalculateCSSZOverview(t *testing.T) {
	czk := func(v int64) domain.Amount { return domain.NewAmount(v, 0) }
	employment := func(v int64) *domain.Amount { a := czk(v); return &a }

	tests := []struct {
		name  string
		input CSSZOverviewInput

		wantMonthlyTaxBase    domain.Amount
		wantMainShare         domain.Amount
		wantSecondaryShare    domain.Amount
		wantMainBase          domain.Amount
		wantSecondaryBase     domain.Amount
		wantMainPartial       domain.Amount
		wantSecondaryPartial  domain.Amount
		wantMin               domain.Amount
		wantFinal             domain.Amount
		wantReportedEmploy    *domain.Amount
		wantSum               domain.Amount
		wantInsured           domain.Amount
		wantTotal             domain.Amount
		wantDecisive          domain.Amount
		wantParticipates      bool
		wantAdvanceComputedOf domain.Amount
	}{
		{
			name:                  "T1 secondary only below the decisive amount",
			input:                 overview2025(czk(100_000), 0, 12),
			wantMonthlyTaxBase:    domain.Amount(833_333),
			wantDecisive:          czk(111_736),
			wantParticipates:      false,
			wantAdvanceComputedOf: czk(55_000),
		},
		{
			name:                  "T2 secondary only above the decisive amount",
			input:                 overview2025(czk(200_000), 0, 12),
			wantMonthlyTaxBase:    domain.Amount(1_666_667),
			wantSecondaryBase:     czk(110_000),
			wantMin:               czk(61_464),
			wantFinal:             czk(110_000),
			wantSum:               czk(110_000),
			wantInsured:           czk(110_000),
			wantTotal:             czk(32_120),
			wantDecisive:          czk(111_736),
			wantParticipates:      true,
			wantAdvanceComputedOf: czk(110_000),
		},
		{
			name:                  "T3 secondary only July to December",
			input:                 overview2025(czk(60_000), 0, 6),
			wantMonthlyTaxBase:    czk(10_000),
			wantSecondaryBase:     czk(33_000),
			wantMin:               czk(30_732),
			wantFinal:             czk(33_000),
			wantSum:               czk(33_000),
			wantInsured:           czk(33_000),
			wantTotal:             czk(9_636),
			wantDecisive:          czk(55_864),
			wantParticipates:      true,
			wantAdvanceComputedOf: czk(33_000),
		},
		{
			name:                  "T3b secondary only below the prorated decisive amount",
			input:                 overview2025(czk(50_000), 0, 6),
			wantMonthlyTaxBase:    domain.Amount(833_333),
			wantDecisive:          czk(55_864),
			wantParticipates:      false,
			wantAdvanceComputedOf: czk(27_500),
		},
		{
			name:                  "T4 mixed year",
			input:                 overview2025(czk(240_000), 6, 6),
			wantMonthlyTaxBase:    czk(20_000),
			wantMainShare:         czk(120_000),
			wantSecondaryShare:    czk(120_000),
			wantMainBase:          czk(66_000),
			wantSecondaryBase:     czk(66_000),
			wantMainPartial:       czk(97_770),
			wantSecondaryPartial:  czk(30_732),
			wantMin:               czk(128_502),
			wantFinal:             czk(163_770),
			wantSum:               czk(163_770),
			wantInsured:           czk(163_770),
			wantTotal:             czk(47_821),
			wantDecisive:          czk(55_864),
			wantParticipates:      true,
			wantAdvanceComputedOf: czk(132_000),
		},
		{
			name:                  "T5 mixed year, secondary part below the decisive amount",
			input:                 overview2025(czk(100_000), 6, 6),
			wantMonthlyTaxBase:    domain.Amount(833_333),
			wantMainShare:         domain.Amount(4_999_998),
			wantSecondaryShare:    domain.Amount(4_999_998),
			wantMainBase:          czk(27_500),
			wantMainPartial:       czk(97_770),
			wantMin:               czk(97_770),
			wantFinal:             czk(97_770),
			wantSum:               czk(97_770),
			wantInsured:           czk(97_770),
			wantTotal:             czk(28_549),
			wantDecisive:          czk(55_864),
			wantParticipates:      false,
			wantAdvanceComputedOf: czk(55_000),
		},
		{
			name: "T6 employment above the maximum lowers the insured base",
			input: func() CSSZOverviewInput {
				in := overview2025(czk(4_000_000), 12, 0)
				in.EmploymentBase = employment(500_000)
				return in
			}(),
			wantMonthlyTaxBase:    domain.Amount(33_333_333),
			wantMainBase:          czk(2_200_000),
			wantMin:               czk(195_540),
			wantFinal:             czk(2_200_000),
			wantReportedEmploy:    employment(500_000),
			wantSum:               czk(2_700_000),
			wantInsured:           czk(1_734_736),
			wantTotal:             czk(506_543),
			wantParticipates:      true,
			wantAdvanceComputedOf: czk(2_200_000),
		},
		{
			name: "T7 employment base above the maximum leaves nothing to insure",
			input: func() CSSZOverviewInput {
				in := overview2025(czk(4_000_000), 12, 0)
				in.EmploymentBase = employment(2_300_000)
				return in
			}(),
			wantMonthlyTaxBase:    domain.Amount(33_333_333),
			wantMainBase:          czk(2_200_000),
			wantMin:               czk(195_540),
			wantFinal:             czk(2_200_000),
			wantReportedEmploy:    employment(2_300_000),
			wantSum:               czk(4_500_000),
			wantInsured:           0,
			wantTotal:             0,
			wantParticipates:      true,
			wantAdvanceComputedOf: czk(2_200_000),
		},
		{
			name: "T8 employment base that fits below the maximum is not reported",
			input: func() CSSZOverviewInput {
				in := overview2025(0, 12, 0)
				in.EmploymentBase = employment(30_000)
				return in
			}(),
			wantMin:          czk(195_540),
			wantFinal:        czk(195_540),
			wantSum:          czk(195_540),
			wantInsured:      czk(195_540),
			wantTotal:        czk(57_098),
			wantParticipates: true,
		},
		{
			name: "T10 main only, tax base above the minimum",
			input: func() CSSZOverviewInput {
				in := overview2025(domain.NewAmount(620_339, 20), 12, 0)
				in.Prepayments = czk(86_991)
				return in
			}(),
			wantMonthlyTaxBase:    domain.NewAmount(51_694, 93),
			wantMainBase:          czk(341_187),
			wantMin:               czk(195_540),
			wantFinal:             czk(341_187),
			wantSum:               czk(341_187),
			wantInsured:           czk(341_187),
			wantTotal:             czk(99_627),
			wantParticipates:      true,
			wantAdvanceComputedOf: czk(341_187),
		},
		{
			name:                  "T10 main only, capped at the maximum",
			input:                 overview2025(czk(5_000_000), 12, 0),
			wantMonthlyTaxBase:    domain.Amount(41_666_667),
			wantMainBase:          czk(2_750_000),
			wantMin:               czk(195_540),
			wantFinal:             czk(2_234_736),
			wantSum:               czk(2_234_736),
			wantInsured:           czk(2_234_736),
			wantTotal:             czk(652_543),
			wantParticipates:      true,
			wantAdvanceComputedOf: czk(2_750_000),
		},
		{
			name:                  "T10 main only, six months below the six-month minimum",
			input:                 overview2025(czk(100_000), 6, 0),
			wantMonthlyTaxBase:    domain.Amount(1_666_667),
			wantMainBase:          czk(55_000),
			wantMin:               czk(97_770),
			wantFinal:             czk(97_770),
			wantSum:               czk(97_770),
			wantInsured:           czk(97_770),
			wantTotal:             czk(28_549),
			wantParticipates:      true,
			wantAdvanceComputedOf: czk(55_000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateCSSZOverview(tt.input)
			if err != nil {
				t.Fatalf("CalculateCSSZOverview() error: %v", err)
			}
			checks := []struct {
				row       string
				got, want domain.Amount
			}{
				{"MonthlyTaxBase", got.MonthlyTaxBase, tt.wantMonthlyTaxBase},
				{"MainTaxBaseShare", got.MainTaxBaseShare, tt.wantMainShare},
				{"SecondaryTaxBaseShare", got.SecondaryTaxBaseShare, tt.wantSecondaryShare},
				{"MainAssessmentBase", got.MainAssessmentBase, tt.wantMainBase},
				{"SecondaryAssessmentBase", got.SecondaryAssessmentBase, tt.wantSecondaryBase},
				{"MainPartialBase", got.MainPartialBase, tt.wantMainPartial},
				{"SecondaryPartialBase", got.SecondaryPartialBase, tt.wantSecondaryPartial},
				{"MinAssessmentBase", got.MinAssessmentBase, tt.wantMin},
				{"FinalAssessmentBase", got.FinalAssessmentBase, tt.wantFinal},
				{"SumBase", got.SumBase, tt.wantSum},
				{"InsuredBase", got.InsuredBase, tt.wantInsured},
				{"TotalInsurance", got.TotalInsurance, tt.wantTotal},
				{"SecondaryDecisiveAmount", got.SecondaryDecisiveAmount, tt.wantDecisive},
				{"AdvanceComputedBase", got.AdvanceComputedBase, tt.wantAdvanceComputedOf},
			}
			for _, c := range checks {
				if c.got != c.want {
					t.Errorf("%s = %d, want %d", c.row, c.got, c.want)
				}
			}
			if got.SecondaryParticipates != tt.wantParticipates {
				t.Errorf("SecondaryParticipates = %v, want %v", got.SecondaryParticipates, tt.wantParticipates)
			}
			switch {
			case tt.wantReportedEmploy == nil && got.ReportedEmploymentBase != nil:
				t.Errorf("ReportedEmploymentBase = %d, want nil", *got.ReportedEmploymentBase)
			case tt.wantReportedEmploy != nil && (got.ReportedEmploymentBase == nil || *got.ReportedEmploymentBase != *tt.wantReportedEmploy):
				t.Errorf("ReportedEmploymentBase = %v, want %d", got.ReportedEmploymentBase, *tt.wantReportedEmploy)
			}
			if want := got.TotalInsurance - tt.input.Prepayments; got.Difference != want {
				t.Errorf("Difference = %d, want %d", got.Difference, want)
			}
		})
	}
}

func TestCalculateCSSZOverview_NoMaximumMeansNoCap(t *testing.T) {
	in := overview2025(domain.NewAmount(5_000_000, 0), 12, 0)
	in.MaxAnnualBase = 0
	got, err := CalculateCSSZOverview(in)
	if err != nil {
		t.Fatalf("CalculateCSSZOverview() error: %v", err)
	}
	if got.FinalAssessmentBase != domain.NewAmount(2_750_000, 0) || got.TotalInsurance != domain.NewAmount(803_000, 0) {
		t.Errorf("final/total = %d/%d, want 2750000/803000 CZK", got.FinalAssessmentBase, got.TotalInsurance)
	}
}

func TestCalculateCSSZOverview_Invalid(t *testing.T) {
	employment := func(v domain.Amount) *domain.Amount { return &v }
	tests := []struct {
		name  string
		input CSSZOverviewInput
	}{
		{"negative employment base", func() CSSZOverviewInput {
			in := overview2025(domain.NewAmount(100_000, 0), 12, 0)
			in.EmploymentBase = employment(-100)
			return in
		}()},
		{"employment base with halere", func() CSSZOverviewInput {
			in := overview2025(domain.NewAmount(100_000, 0), 12, 0)
			in.EmploymentBase = employment(domain.NewAmount(12_345, 67))
			return in
		}()},
		{"no months", overview2025(domain.NewAmount(100_000, 0), 0, 0)},
		{"thirteen months", overview2025(domain.NewAmount(100_000, 0), 7, 6)},
		{"negative main months", overview2025(domain.NewAmount(100_000, 0), -1, 6)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := CalculateCSSZOverview(tt.input); !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestSocialAdvanceComputedBase(t *testing.T) {
	if got, want := SocialAdvanceComputedBase(domain.NewAmount(620_339, 20)), domain.NewAmount(341_187, 0); got != want {
		t.Errorf("SocialAdvanceComputedBase = %d, want %d", got, want)
	}
	if got := SocialAdvanceComputedBase(0); got != 0 {
		t.Errorf("SocialAdvanceComputedBase(0) = %d, want 0", got)
	}
}
