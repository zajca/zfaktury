package taxrules

import (
	"fmt"
	"sort"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

var ruleSets = map[int]RuleSet{
	2024: {
		ID:     "cz-dpfo-2024.v1",
		Year:   2024,
		Status: RuleStatusFinal,
		Sources: []LegalSource{
			{Title: "Zákon č. 586/1992 Sb., § 15, § 16, § 35ba, § 35c"},
			{Title: "Nařízení vlády č. 286/2023 Sb.", Note: "průměrná mzda pro rok 2024"},
			{Title: "Finanční správa: DPFO 2024 dotazy a odpovědi", URL: "https://financnisprava.gov.cz/cs/dane/dane/dan-z-prijmu/dotazy-a-odpovedi/dan-z-prijmu-fyzickych-osob/aktualne-k-dani-z-prijmu-fyzickych-osob-2024"},
		},
		Constants: TaxYearConstants{
			ProgressiveThreshold: domain.NewAmount(1_582_812, 0),
			BasicCredit:          domain.NewAmount(30_840, 0),
			SpouseCredit:         domain.NewAmount(24_840, 0),
			StudentCredit:        domain.NewAmount(0, 0),
			DisabilityCredit1:    domain.NewAmount(2_520, 0),
			DisabilityCredit3:    domain.NewAmount(5_040, 0),
			DisabilityZTPP:       domain.NewAmount(16_140, 0),
			ChildBenefit1:        domain.NewAmount(15_204, 0),
			ChildBenefit2:        domain.NewAmount(22_320, 0),
			ChildBenefit3Plus:    domain.NewAmount(27_840, 0),
			ChildBenefitZTP:      domain.NewAmount(0, 0),
			SocialMinMonthly:     domain.NewAmount(13_191, 0),
			SocialRate:           292,
			HealthMinMonthly:     domain.NewAmount(21_984, 0),
			HealthRate:           135,
			FlatRateCaps: map[int]domain.Amount{
				80: domain.NewAmount(1_600_000, 0),
				60: domain.NewAmount(1_200_000, 0),
				40: domain.NewAmount(800_000, 0),
				30: domain.NewAmount(600_000, 0),
			},
			TimeTestYears:               3,
			SecurityExemptionLimit:      domain.NewAmount(0, 0),
			SpouseIncomeLimit:           domain.NewAmount(68_000, 0),
			DeductionCapMortgage:        domain.NewAmount(150_000, 0),
			DeductionCapLifeInsurance:   domain.NewAmount(0, 0),
			DeductionCapPension:         domain.NewAmount(0, 0),
			DeductionCapSavingsCombined: domain.NewAmount(48_000, 0),
			DeductionCapUnionDues:       domain.NewAmount(3_000, 0),
			MaxChildBonus:               domain.NewAmount(60_300, 0),
		},
		Deductions: DeductionRules{
			Pension: PensionDeductionRules{
				SharedSavingsCap: domain.NewAmount(48_000, 0),
				MonthlyThreshold: Schedule[domain.Amount]{
					{From: NewDate(2024, time.January, 1), To: NewDate(2024, time.June, 30), Value: domain.NewAmount(1_000, 0)},
					{From: NewDate(2024, time.July, 1), To: NewDate(2024, time.December, 31), Value: domain.NewAmount(1_700, 0)},
				},
			},
		},
		Forms: dpfodp7Forms(),
	},
	2025: {
		ID:     "cz-dpfo-2025.v1",
		Year:   2025,
		Status: RuleStatusFinal,
		Sources: []LegalSource{
			{Title: "Zákon č. 586/1992 Sb., § 15, § 16, § 35ba, § 35c"},
			{Title: "Nařízení vlády č. 282/2024 Sb.", Note: "průměrná mzda pro rok 2025"},
		},
		Constants: TaxYearConstants{
			ProgressiveThreshold: domain.NewAmount(1_676_052, 0),
			BasicCredit:          domain.NewAmount(30_840, 0),
			SpouseCredit:         domain.NewAmount(24_840, 0),
			StudentCredit:        domain.NewAmount(0, 0),
			DisabilityCredit1:    domain.NewAmount(2_520, 0),
			DisabilityCredit3:    domain.NewAmount(5_040, 0),
			DisabilityZTPP:       domain.NewAmount(16_140, 0),
			ChildBenefit1:        domain.NewAmount(15_204, 0),
			ChildBenefit2:        domain.NewAmount(22_320, 0),
			ChildBenefit3Plus:    domain.NewAmount(27_840, 0),
			ChildBenefitZTP:      domain.NewAmount(0, 0),
			SocialMinMonthly:     domain.NewAmount(16_295, 0),
			SocialRate:           292,
			HealthMinMonthly:     domain.NewAmount(23_279, 0),
			HealthRate:           135,
			FlatRateCaps: map[int]domain.Amount{
				80: domain.NewAmount(1_600_000, 0),
				60: domain.NewAmount(1_200_000, 0),
				40: domain.NewAmount(800_000, 0),
				30: domain.NewAmount(600_000, 0),
			},
			TimeTestYears:               3,
			SecurityExemptionLimit:      domain.NewAmount(100_000_000, 0),
			SpouseIncomeLimit:           domain.NewAmount(68_000, 0),
			DeductionCapMortgage:        domain.NewAmount(150_000, 0),
			DeductionCapLifeInsurance:   domain.NewAmount(0, 0),
			DeductionCapPension:         domain.NewAmount(0, 0),
			DeductionCapSavingsCombined: domain.NewAmount(48_000, 0),
			DeductionCapUnionDues:       domain.NewAmount(3_000, 0),
			MaxChildBonus:               domain.NewAmount(60_300, 0),
		},
		Deductions: DeductionRules{
			Pension: PensionDeductionRules{
				SharedSavingsCap: domain.NewAmount(48_000, 0),
				MonthlyThreshold: Schedule[domain.Amount]{
					{From: NewDate(2025, time.January, 1), To: NewDate(2025, time.December, 31), Value: domain.NewAmount(1_700, 0)},
				},
			},
		},
		Forms: dpfodp7Forms(),
	},
	2026: {
		ID:     "cz-dpfo-2026.v0-provisional",
		Year:   2026,
		Status: RuleStatusProvisional,
		Sources: []LegalSource{
			{Title: "Zákon č. 586/1992 Sb., § 15, § 16, § 35ba, § 35c"},
			{Title: "Finanční správa: obecné informace pro zaměstnance a zaměstnavatele", URL: "https://financnisprava.gov.cz/cs/dane/dane/dan-z-prijmu/zamestnanci-zamestnavatele/obecne-informace", Note: "36násobek průměrné mzdy pro rok 2026"},
			{Title: "ČSSZ: zálohy na pojistné na důchodové pojištění", URL: "https://www.cssz.cz/web/cz/zalohy-na-pojistne-na-duchodove-pojisteni", Note: "minimální měsíční vyměřovací základ OSVČ pro hlavní SVČ v roce 2026"},
			{Title: "VZP: OSVČ - minimální výše záloh", URL: "https://www.vzp.cz/platci/informace/osvc/osvc-minimalni-vyse-zaloh", Note: "minimální měsíční vyměřovací základ OSVČ pro zdravotní pojištění v roce 2026"},
		},
		Constants: TaxYearConstants{
			ProgressiveThreshold: domain.NewAmount(1_762_812, 0),
			BasicCredit:          domain.NewAmount(30_840, 0),
			SpouseCredit:         domain.NewAmount(24_840, 0),
			StudentCredit:        domain.NewAmount(0, 0),
			DisabilityCredit1:    domain.NewAmount(2_520, 0),
			DisabilityCredit3:    domain.NewAmount(5_040, 0),
			DisabilityZTPP:       domain.NewAmount(16_140, 0),
			ChildBenefit1:        domain.NewAmount(15_204, 0),
			ChildBenefit2:        domain.NewAmount(22_320, 0),
			ChildBenefit3Plus:    domain.NewAmount(27_840, 0),
			ChildBenefitZTP:      domain.NewAmount(0, 0),
			SocialMinMonthly:     domain.NewAmount(19_587, 0),
			SocialRate:           292,
			HealthMinMonthly:     domain.NewAmount(24_484, 0),
			HealthRate:           135,
			FlatRateCaps: map[int]domain.Amount{
				80: domain.NewAmount(1_600_000, 0),
				60: domain.NewAmount(1_200_000, 0),
				40: domain.NewAmount(800_000, 0),
				30: domain.NewAmount(600_000, 0),
			},
			TimeTestYears:               3,
			SecurityExemptionLimit:      domain.NewAmount(100_000_000, 0),
			SpouseIncomeLimit:           domain.NewAmount(68_000, 0),
			DeductionCapMortgage:        domain.NewAmount(150_000, 0),
			DeductionCapLifeInsurance:   domain.NewAmount(0, 0),
			DeductionCapPension:         domain.NewAmount(0, 0),
			DeductionCapSavingsCombined: domain.NewAmount(48_000, 0),
			DeductionCapUnionDues:       domain.NewAmount(3_000, 0),
			MaxChildBonus:               domain.NewAmount(60_300, 0),
		},
		Deductions: DeductionRules{
			Pension: PensionDeductionRules{
				SharedSavingsCap: domain.NewAmount(48_000, 0),
				MonthlyThreshold: Schedule[domain.Amount]{
					{From: NewDate(2026, time.January, 1), To: NewDate(2026, time.December, 31), Value: domain.NewAmount(1_700, 0)},
				},
			},
		},
		Forms: dpfodp7Forms(),
	},
}

// GetRuleSet returns the validated rule set for year. Unknown years fail hard;
// tax calculations must never silently fall back to another year.
func GetRuleSet(year int) (RuleSet, error) {
	rs, ok := ruleSets[year]
	if !ok {
		return RuleSet{}, fmt.Errorf("no tax rule set for year %d: %w", year, domain.ErrInvalidInput)
	}
	if err := ValidateRuleSet(rs); err != nil {
		return RuleSet{}, fmt.Errorf("invalid tax rule set %q: %w", rs.ID, err)
	}
	return cloneRuleSet(rs), nil
}

// SupportedYears returns the years that have an explicit rule set.
func SupportedYears() []int {
	years := make([]int, 0, len(ruleSets))
	for year := range ruleSets {
		years = append(years, year)
	}
	sort.Ints(years)
	return years
}

func cloneRuleSet(rs RuleSet) RuleSet {
	rs.Sources = append([]LegalSource(nil), rs.Sources...)
	rs.Constants.FlatRateCaps = cloneAmountMap(rs.Constants.FlatRateCaps)
	rs.Deductions.Pension.MonthlyThreshold = append(Schedule[domain.Amount](nil), rs.Deductions.Pension.MonthlyThreshold...)
	return rs
}

func cloneAmountMap(in map[int]domain.Amount) map[int]domain.Amount {
	if in == nil {
		return nil
	}
	out := make(map[int]domain.Amount, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func dpfodp7Forms() FormRules {
	return FormRules{
		IncomeTaxXML: IncomeTaxXMLFormRules{
			FormCode:      "DPFDP7",
			SchemaFile:    "docs/xml-schemas/epo/dpfdp7_epo2.xsd",
			ValidFromYear: 2024,
			ValidToYear:   2025,
		},
	}
}
