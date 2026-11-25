package taxrules

import (
	"fmt"
	"sort"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// cryptoTimeTestFrom is 15 Feb 2025 (Europe/Prague calendar date): the crypto
// time-test exemption applies to sales on or after this date, regardless of
// when the coins were bought.
var cryptoTimeTestFrom = time.Date(2025, 2, 15, 0, 0, 0, 0, pragueLocation)

var ruleSets = map[int]RuleSet{
	2024: {
		ID:     "cz-dpfo-2024.v5",
		Year:   2024,
		Status: RuleStatusFinal,
		Sources: []LegalSource{
			{Title: "Zákon č. 586/1992 Sb., § 15, § 16, § 35ba, § 35c"},
			{Title: "Nařízení vlády č. 286/2023 Sb.", Note: "průměrná mzda pro rok 2024"},
			{Title: "Finanční správa: DPFO 2024 dotazy a odpovědi", URL: "https://financnisprava.gov.cz/cs/dane/dane/dan-z-prijmu/dotazy-a-odpovedi/dan-z-prijmu-fyzickych-osob/aktualne-k-dani-z-prijmu-fyzickych-osob-2024"},
			{Title: "ČSSZ: stanovení výše pojistného na důchodové pojištění OSVČ", URL: "https://www.cssz.gov.cz/osvc-duchodove-pojisteni-stanoveni-vyse-pojistneho", Note: "maximální vyměřovací základ 2 110 416 Kč (48násobek průměrné mzdy)"},
			{Title: "ČSSZ: výše minimálních vyměřovacích základů OSVČ", URL: "https://www.cssz.gov.cz/web/cz/vyse-minimalnich-vymerovacich-zakladu-osvc", Note: "minimální roční vyměřovací základ 2024 (hlavní činnost): 158 292 Kč, tj. 13 191 Kč měsíčně"},
			{Title: "ČSSZ: Pokyny k vyplnění Přehledu o příjmech a výdajích OSVČ za rok 2024", URL: "https://www.cssz.gov.cz/documents/20143/2860132/Pokyny+k+vypln%C4%9Bn%C3%AD+p%C5%99ehledu+o+p%C5%99%C3%ADjmech+a+v%C3%BDdaj%C3%ADch+OSV%C4%8C+za+rok+2024.pdf/a1e348bc-e138-3952-b228-c37fb305b374", Note: "vedlejší činnost: minimální měsíční základ 4 837 Kč, rozhodná částka 105 520 Kč snížená o 8 794 Kč za každý měsíc bez vedlejší činnosti"},
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
			SocialMaxAnnualBase:  domain.NewAmount(2_110_416, 0),
			SocialMinAdvanceBases: []MonthlyBasePeriod{
				{FromMonth: 1, Base: domain.NewAmount(13_191, 0)},
			},
			SocialSecondaryMinMonthly:       domain.NewAmount(4_837, 0),
			SocialDecisiveAmount:            domain.NewAmount(105_520, 0),
			SocialDecisiveAmountMonthlyStep: domain.NewAmount(8_794, 0),
			HealthMinMonthly:                domain.NewAmount(21_983, 50),
			HealthRate:                      135,
			FlatRateCaps: map[int]domain.Amount{
				80: domain.NewAmount(1_600_000, 0),
				60: domain.NewAmount(1_200_000, 0),
				40: domain.NewAmount(800_000, 0),
				30: domain.NewAmount(600_000, 0),
			},
			TimeTestYears:               3,
			CryptoTimeTestFrom:          cryptoTimeTestFrom,
			SecurityExemptionLimit:      domain.NewAmount(0, 0),
			SpouseIncomeLimit:           domain.NewAmount(68_000, 0),
			DeductionCapMortgage:        domain.NewAmount(150_000, 0),
			DeductionCapLifeInsurance:   domain.NewAmount(0, 0),
			DeductionCapPension:         domain.NewAmount(0, 0),
			DeductionCapSavingsCombined: domain.NewAmount(48_000, 0),
			DeductionCapUnionDues:       domain.NewAmount(3_000, 0),
		},
		Forms: dpfodp7Forms(),
	},
	2025: {
		ID:     "cz-dpfo-2025.v5",
		Year:   2025,
		Status: RuleStatusFinal,
		Sources: []LegalSource{
			{Title: "Zákon č. 586/1992 Sb., § 15, § 16, § 35ba, § 35c"},
			{Title: "Nařízení vlády č. 282/2024 Sb.", Note: "průměrná mzda pro rok 2025"},
			{Title: "MF: změny v daňovém osvobození prodeje podílů a akcií", URL: "https://mf.gov.cz/cs/ministerstvo/media/tiskove-zpravy/2025/zmeny-v-daneni-prodeje-podilu-ve-firmach-a-akcii-a-61234", Note: "od ledna 2025 limit 40 mil. Kč na úhrn příjmů ve zdaňovacím období, příjmy nad limit se zdaňují poměrně"},
			{Title: "ČSSZ: stanovení výše pojistného na důchodové pojištění OSVČ", URL: "https://www.cssz.gov.cz/osvc-duchodove-pojisteni-stanoveni-vyse-pojistneho", Note: "maximální vyměřovací základ 2 234 736 Kč (48násobek průměrné mzdy)"},
			{Title: "ČSSZ: výše minimálních vyměřovacích základů OSVČ", URL: "https://www.cssz.gov.cz/web/cz/vyse-minimalnich-vymerovacich-zakladu-osvc", Note: "minimální roční vyměřovací základ 2025 (hlavní činnost): 195 540 Kč, tj. 16 295 Kč měsíčně; minimální záloha 4 759 Kč"},
			{Title: "ČSSZ: Pokyny k vyplnění Přehledu o příjmech a výdajích OSVČ za rok 2025", URL: "https://www.cssz.gov.cz/documents/20143/3201321/OSVC25+Pokyny+k+vypln%C4%9Bn%C3%AD+za+rok+2025.pdf/8291cdc3-d6d7-eb67-4995-0dd316d9ccbf", Note: "vedlejší činnost: minimální měsíční základ 5 122 Kč, rozhodná částka 111 736 Kč snížená o 9 312 Kč za každý měsíc bez vedlejší činnosti"},
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
			SocialMaxAnnualBase:  domain.NewAmount(2_234_736, 0),
			SocialMinAdvanceBases: []MonthlyBasePeriod{
				{FromMonth: 1, Base: domain.NewAmount(16_295, 0)},
			},
			SocialSecondaryMinMonthly:       domain.NewAmount(5_122, 0),
			SocialDecisiveAmount:            domain.NewAmount(111_736, 0),
			SocialDecisiveAmountMonthlyStep: domain.NewAmount(9_312, 0),
			HealthMinMonthly:                domain.NewAmount(23_278, 50),
			HealthRate:                      135,
			FlatRateCaps: map[int]domain.Amount{
				80: domain.NewAmount(1_600_000, 0),
				60: domain.NewAmount(1_200_000, 0),
				40: domain.NewAmount(800_000, 0),
				30: domain.NewAmount(600_000, 0),
			},
			TimeTestYears:               3,
			CryptoTimeTestFrom:          cryptoTimeTestFrom,
			SecurityExemptionLimit:      domain.NewAmount(40_000_000, 0),
			SpouseIncomeLimit:           domain.NewAmount(68_000, 0),
			DeductionCapMortgage:        domain.NewAmount(150_000, 0),
			DeductionCapLifeInsurance:   domain.NewAmount(0, 0),
			DeductionCapPension:         domain.NewAmount(0, 0),
			DeductionCapSavingsCombined: domain.NewAmount(48_000, 0),
			DeductionCapUnionDues:       domain.NewAmount(3_000, 0),
		},
		Forms: dpfodp7Forms(),
	},
	2026: {
		ID:     "cz-dpfo-2026.v4-provisional",
		Year:   2026,
		Status: RuleStatusProvisional,
		Sources: []LegalSource{
			{Title: "Zákon č. 586/1992 Sb., § 15, § 16, § 35ba, § 35c"},
			{Title: "Finanční správa: obecné informace pro zaměstnance a zaměstnavatele", URL: "https://financnisprava.gov.cz/cs/dane/dane/dan-z-prijmu/zamestnanci-zamestnavatele/obecne-informace", Note: "36násobek průměrné mzdy pro rok 2026"},
			{Title: "MF: změny v daňovém osvobození prodeje podílů a akcií", URL: "https://mf.gov.cz/cs/ministerstvo/media/tiskove-zpravy/2025/zmeny-v-daneni-prodeje-podilu-ve-firmach-a-akcii-a-61234", Note: "od 1. 1. 2026 se limit 40 mil. Kč u podílů a akcií ruší, u kryptoaktiv zůstává"},
			{Title: "ČSSZ: změna pojistného na důchodové pojištění OSVČ od 1. 7. 2026", URL: "https://www.cssz.gov.cz/-/informace-pro-osvc-o-zmene-pojistneho-na-duchodove-pojisteni-od-1-7-2026", Note: "zálohy: základ 19 587 Kč (leden-červen), 17 139 Kč (od července); Přehled za 2026 přepočítá leden-červen, jako by změna platila od ledna, tedy minimum 17 139 Kč za všech 12 měsíců"},
			{Title: "VZP: OSVČ - minimální výše záloh", URL: "https://www.vzp.cz/platci/informace/povinnosti-platcu-metodika/osvc/osvc-minimalni-vyse-zaloh", Note: "minimální měsíční vyměřovací základ OSVČ pro zdravotní pojištění v roce 2026: 24 483,50 Kč"},
			{Title: "Nařízení vlády č. 365/2025 Sb.", URL: "https://www.vzp.cz/platci/informace/povinnosti-platcu-metodika/osvc/osvc-prumerna-mzda", Note: "průměrná mzda pro rok 2026: 48 967 Kč"},
			{Title: "Zákon č. 90/2026 Sb., kterým se mění zákon č. 589/1992 Sb.", URL: "https://www.epravo.cz/_dataPublic/sbirky/2026/Sb_2026_90_PZZ.pdf", Note: "minimální vyměřovací základ OSVČ 35 % průměrné mzdy od 1. 7. 2026"},
			{Title: "ČSSZ: stanovení výše pojistného na důchodové pojištění OSVČ", URL: "https://www.cssz.gov.cz/osvc-duchodove-pojisteni-stanoveni-vyse-pojistneho", Note: "maximální vyměřovací základ 2 350 416 Kč (48násobek průměrné mzdy)"},
			{Title: "ČSSZ: přehled nejdůležitějších údajů pro sociální zabezpečení v roce 2026", URL: "https://www.cssz.gov.cz/-/prehled-nejdulezitejsich-udaju-pro-socialni-zabezpeceni-v-roce-2026", Note: "vedlejší činnost: minimální měsíční základ 5 387 Kč, rozhodná částka 117 521 Kč; krok 9 794 Kč odvozen jako 117 521 / 12 zaokrouhleno nahoru, ověřit v Pokynech 2026"},
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
			SocialMinMonthly:     domain.NewAmount(17_139, 0),
			SocialRate:           292,
			SocialMaxAnnualBase:  domain.NewAmount(2_350_416, 0),
			SocialMinAdvanceBases: []MonthlyBasePeriod{
				{FromMonth: 1, Base: domain.NewAmount(19_587, 0)},
				{FromMonth: 7, Base: domain.NewAmount(17_139, 0)},
			},
			SocialSecondaryMinMonthly:       domain.NewAmount(5_387, 0),
			SocialDecisiveAmount:            domain.NewAmount(117_521, 0),
			SocialDecisiveAmountMonthlyStep: domain.NewAmount(9_794, 0),
			HealthMinMonthly:                domain.NewAmount(24_483, 50),
			HealthRate:                      135,
			FlatRateCaps: map[int]domain.Amount{
				80: domain.NewAmount(1_600_000, 0),
				60: domain.NewAmount(1_200_000, 0),
				40: domain.NewAmount(800_000, 0),
				30: domain.NewAmount(600_000, 0),
			},
			TimeTestYears:               3,
			CryptoTimeTestFrom:          cryptoTimeTestFrom,
			SecurityExemptionLimit:      domain.NewAmount(40_000_000, 0),
			SpouseIncomeLimit:           domain.NewAmount(68_000, 0),
			DeductionCapMortgage:        domain.NewAmount(150_000, 0),
			DeductionCapLifeInsurance:   domain.NewAmount(0, 0),
			DeductionCapPension:         domain.NewAmount(0, 0),
			DeductionCapSavingsCombined: domain.NewAmount(48_000, 0),
			DeductionCapUnionDues:       domain.NewAmount(3_000, 0),

			SecurityExemptionLimitAssetTypes: []string{domain.AssetTypeCrypto},
		},
		Forms: dpfodp7Forms(),
	},
}

// NoRuleSetError reports a tax year without a rule set. It matches
// domain.ErrInvalidInput through errors.Is.
type NoRuleSetError struct {
	Year int
}

func (e *NoRuleSetError) Error() string {
	return fmt.Sprintf("no tax rule set for year %d: %s", e.Year, domain.ErrInvalidInput)
}

func (e *NoRuleSetError) Unwrap() error { return domain.ErrInvalidInput }

// GetRuleSet returns the validated rule set for year. Unknown years fail hard;
// tax calculations must never silently fall back to another year.
func GetRuleSet(year int) (RuleSet, error) {
	rs, ok := ruleSets[year]
	if !ok {
		return RuleSet{}, &NoRuleSetError{Year: year}
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
	rs.Constants.SecurityExemptionLimitAssetTypes = append([]string(nil), rs.Constants.SecurityExemptionLimitAssetTypes...)
	rs.Constants.SocialMinAdvanceBases = append([]MonthlyBasePeriod(nil), rs.Constants.SocialMinAdvanceBases...)
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
