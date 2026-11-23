package annualtaxxml

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/zajca/zfaktury/internal/domain"
)

var csszDepCodePattern = regexp.MustCompile(`^\d{3}$`)

// GenerateSocialInsuranceXML produces CSSZ OSVC overview XML from a SocialInsuranceOverview,
// application settings containing taxpayer information and the monthly advance
// base for ř. 35 (bounded by the payment-year minimum and maximum by the caller).
func GenerateSocialInsuranceXML(sio *domain.SocialInsuranceOverview, settings map[string]string, advanceMonthlyBase domain.Amount) ([]byte, error) {
	if sio == nil {
		return nil, fmt.Errorf("social insurance overview is nil: %w", domain.ErrInvalidInput)
	}
	if advanceMonthlyBase < 0 {
		return nil, fmt.Errorf("social insurance overview monthly advance base %d is negative: %w", advanceMonthlyBase, domain.ErrInvalidInput)
	}
	if err := validateSocialInsuranceXMLInput(settings); err != nil {
		return nil, err
	}
	if err := sio.ActivityMonths.Validate(); err != nil {
		return nil, fmt.Errorf("social insurance overview activity months: %w", err)
	}

	rok := strconv.Itoa(sio.Year)

	assessmentBase := ToWholeCZK(sio.AssessmentBase)
	finalAssessmentBase := ToWholeCZK(sio.FinalAssessmentBase)
	totalInsurance := ToWholeCZK(sio.TotalInsurance)
	prepayments := ToWholeCZK(sio.Prepayments)
	difference := ToWholeCZK(sio.Difference)
	newMonthlyPrepay := ToWholeCZK(sio.NewMonthlyPrepay)
	months := int64(sio.ActivityMonths.Count())
	// ř. 35: the monthly advance base bounded by the payment-year minimum and
	// maximum; the caller derives it with calc.SocialAdvanceMonthlyBase.
	monthlyAssessmentBase := ToWholeCZK(advanceMonthlyBase)

	priznElektr := "A"
	if v := strings.TrimSpace(settings["tax_return_filed_electronically"]); v != "" {
		priznElektr = v
	}

	doc := &OSVC{
		Xmlns:   "http://schemas.cssz.cz/OSVC2025",
		Version: "1.0",
		Vendor: Vendor{
			ProductName:    "ZFaktury",
			ProductVersion: "1.0",
		},
		Sender: Sender{
			EmailNotifikace: "",
			ISDSreport:      "3",
			VerzeProtokolu:  "1.0",
		},
		Prehled: PrehledOSVC{
			For:  "prehledosvc",
			Dep:  strings.TrimSpace(settings["cssz_code"]),
			Rok:  rok,
			Typ:  CSSZFilingTypeCode(sio.FilingType),
			VSDP: strings.TrimSpace(settings["cssz_variable_symbol"]),
			Dat:  "",

			Client: Client{
				Name: ClientName{
					Fir: settings["taxpayer_first_name"],
					Sur: settings["taxpayer_last_name"],
					Tit: settings["taxpayer_title"],
				},
				Birth: ClientBirth{
					Bno: settings["taxpayer_birth_number"],
					Den: settings["taxpayer_birth_date"],
				},
				Adr: Address{
					Str: settings["taxpayer_street"],
					Num: settings["taxpayer_house_number"],
					Pnu: settings["taxpayer_postal_code"],
					Cit: settings["taxpayer_city"],
					Cnt: "CZ",
				},
				IDDS:  settings["taxpayer_databox_id"],
				Email: settings["taxpayer_email"],
				Tel:   settings["taxpayer_phone"],
				Druc:  "H",
				Hlavc: Hlavc{
					MonthFlags: mainActivityMonthFlags(sio.ActivityMonths),
				},
				Vedc:  Vedc{},
				Sleva: MonthFlags{},
			},

			PVV: PVV{
				Pri:  formatAmount(sio.TaxBase),
				Mesc: HVPair{H: strconv.Itoa(sio.ActivityMonths.Count()), V: ""},
				// ř. 22: months with a whole-month sickness benefit are not tracked, so it equals ř. 21.
				Mesv: HVPair{H: strconv.Itoa(sio.ActivityMonths.Count()), V: ""},
				Mesp: formatMonthlyAverage(sio.TaxBase, months),
				Rdza: HVPair{H: "0", V: "0"},
				VVZ: HVPair{
					H: strconv.FormatInt(assessmentBase, 10),
					V: "0",
				},
				DVZ:       HVPair{H: "0", V: "0"},
				MVZ:       strconv.FormatInt(finalAssessmentBase, 10),
				UVZ:       strconv.FormatInt(finalAssessmentBase, 10),
				Vzsu:      strconv.FormatInt(finalAssessmentBase, 10),
				Vzsvc:     strconv.FormatInt(finalAssessmentBase, 10),
				Poj:       strconv.FormatInt(totalInsurance, 10),
				Slev:      "0",
				Pojposlev: strconv.FormatInt(totalInsurance, 10),
				Zal:       strconv.FormatInt(prepayments, 10),
				Ned:       strconv.FormatInt(difference, 10),
			},

			Zal: Zal{
				Ved:  "H",
				Pau:  settings["flat_tax_regime"],
				VZ:   strconv.FormatInt(monthlyAssessmentBase, 10),
				DP:   strconv.FormatInt(newMonthlyPrepay, 10),
				NP:   strings.TrimSpace(settings["sickness_insurance_monthly"]),
				Duch: strings.TrimSpace(settings["working_pensioner_discount"]),
			},

			Pre: Pre{
				Vra: "0",
				BS:  PreBS{},
			},

			Prizn: Prizn{
				Pau:    settings["flat_tax_regime_reason"],
				Pov:    "A",
				Elektr: priznElektr,
				Por:    settings["tax_advisor_after_april"],
			},

			Spo: Spo{
				Name: SpoName{},
			},
			Prilo: Prilo{
				Coun:    "0",
				Plnamoc: "N",
				Jina:    "N",
			},
		},
	}

	output, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling social insurance XML: %w", err)
	}

	header := []byte(xml.Header)
	result := make([]byte, 0, len(header)+len(output))
	result = append(result, header...)
	result = append(result, output...)

	return result, nil
}

func validateSocialInsuranceXMLInput(settings map[string]string) error {
	required := []struct {
		label string
		key   string
	}{
		{"kód OSSZ", "cssz_code"},
		{"jméno", "taxpayer_first_name"},
		{"příjmení", "taxpayer_last_name"},
		{"rodné číslo / EČP", "taxpayer_birth_number"},
		{"datum narození", "taxpayer_birth_date"},
		{"číslo domu", "taxpayer_house_number"},
		{"PSČ", "taxpayer_postal_code"},
		{"obec", "taxpayer_city"},
	}

	var problems []string
	for _, req := range required {
		if strings.TrimSpace(settings[req.key]) == "" {
			problems = append(problems, "chybí "+req.label)
		}
	}
	if dep := strings.TrimSpace(settings["cssz_code"]); dep != "" && !csszDepCodePattern.MatchString(dep) {
		problems = append(problems, "kód OSSZ musí mít přesně 3 číslice")
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("nelze vygenerovat XML přehledu ČSSZ: %s: %w", strings.Join(problems, "; "), domain.ErrInvalidInput)
}

func formatAmount(amount domain.Amount) string {
	return amount.String()
}

// formatMonthlyAverage divides amount by months, rounded half up to whole halere
// (ř. 23 "Zaokr. na 2 des. místa").
func formatMonthlyAverage(amount domain.Amount, months int64) string {
	if months <= 0 {
		return "0.00"
	}
	value := int64(amount)
	if value < 0 {
		return domain.Amount(-((-2*value + months) / (2 * months))).String()
	}
	return domain.Amount((2*value + months) / (2 * months)).String()
}

// mainActivityMonthFlags marks the months of main activity ("A"); a full year is
// reported only in m13 ("Výkon SVČ celý rok").
func mainActivityMonthFlags(m domain.ActivityMonths) MonthFlags {
	if m.IsFullYear() {
		return MonthFlags{M13: "A"}
	}
	var flags [12]string
	for month := 1; month <= 12; month++ {
		if m.Has(month) {
			flags[month-1] = "A"
		}
	}
	return MonthFlags{
		M1: flags[0], M2: flags[1], M3: flags[2], M4: flags[3],
		M5: flags[4], M6: flags[5], M7: flags[6], M8: flags[7],
		M9: flags[8], M10: flags[9], M11: flags[10], M12: flags[11],
	}
}
