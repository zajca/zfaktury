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
// base for ř. 35 that the overview was calculated with.
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
	if err := sio.FilingMethod.Validate(); err != nil {
		return nil, fmt.Errorf("social insurance overview filing method: %w", err)
	}
	if err := domain.ValidateSecondaryActivity(sio.ActivityMonths, sio.SecondaryMonths, sio.SecondaryReason); err != nil {
		return nil, fmt.Errorf("social insurance overview secondary activity: %w", err)
	}

	rok := strconv.Itoa(sio.Year)

	assessmentBase := ToWholeCZK(sio.AssessmentBase)
	finalAssessmentBase := ToWholeCZK(sio.FinalAssessmentBase)
	totalInsurance := ToWholeCZK(sio.TotalInsurance)
	prepayments := ToWholeCZK(sio.Prepayments)
	difference := ToWholeCZK(sio.Difference)
	newMonthlyPrepay := ToWholeCZK(sio.NewMonthlyPrepay)
	months := int64(sio.ActivityMonths.Count())

	mainMonths := sio.ActivityMonths &^ sio.SecondaryMonths
	secondaryCount := sio.SecondaryMonths.Count()
	mixed := mainMonths != 0 && secondaryCount > 0
	// ř. 13: kind of activity of the year.
	activityKind := "H"
	switch {
	case mixed:
		activityKind = "S"
	case secondaryCount > 0:
		activityKind = "V"
	}
	var mainFlags MonthFlags
	if mainMonths != 0 {
		mainFlags = mainActivityMonthFlags(mainMonths)
	}
	rdza, dvz := HVPair{H: "0", V: "0"}, HVPair{H: "0", V: "0"}
	if mixed {
		rdza = HVPair{H: formatAmount(sio.MainTaxBaseShare), V: formatAmount(sio.SecondaryTaxBaseShare)}
		dvz = HVPair{H: strconv.FormatInt(ToWholeCZK(sio.MainPartialBase), 10), V: strconv.FormatInt(ToWholeCZK(sio.SecondaryPartialBase), 10)}
	}
	mesc := HVPair{H: strconv.Itoa(mainMonths.Count())}
	if secondaryCount > 0 {
		mesc.V = strconv.Itoa(secondaryCount)
	}
	// ř. 22: months with a whole-month sickness benefit are not tracked, so it equals ř. 21.
	mesv := mesc
	var employmentBase string
	if sio.ReportedEmploymentBase != nil {
		employmentBase = strconv.FormatInt(ToWholeCZK(*sio.ReportedEmploymentBase), 10)
	}
	advanceKind := "H"
	if sio.AdvanceAsSecondary() {
		advanceKind = "V"
	}
	// ř. 35: the monthly advance base of the stored calculation.
	monthlyAssessmentBase := ToWholeCZK(advanceMonthlyBase)

	// ř. 40 and ř. 41: how the income-tax return was filed after 1 April.
	priznElektr, priznPor := "N", "N"
	switch sio.FilingMethod {
	case domain.FilingMethodElectronic:
		priznElektr = "A"
	case domain.FilingMethodAdvisor:
		priznPor = "A"
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
				Druc:  activityKind,
				Hlavc: Hlavc{
					MonthFlags: mainFlags,
				},
				Vedc:  secondaryActivity(sio.SecondaryMonths, sio.SecondaryReason),
				Sleva: MonthFlags{},
			},

			PVV: PVV{
				Pri:  formatAmount(sio.TaxBase),
				Mesc: mesc,
				Mesv: mesv,
				Mesp: formatMonthlyAverage(sio.TaxBase, months),
				Rdza: rdza,
				VVZ: HVPair{
					H: strconv.FormatInt(assessmentBase, 10),
					V: strconv.FormatInt(ToWholeCZK(sio.SecondaryAssessmentBase), 10),
				},
				DVZ:       dvz,
				MVZ:       strconv.FormatInt(finalAssessmentBase, 10),
				UVZ:       strconv.FormatInt(finalAssessmentBase, 10),
				Vzza:      employmentBase,
				Vzsu:      strconv.FormatInt(ToWholeCZK(sio.SumBase), 10),
				Vzsvc:     strconv.FormatInt(ToWholeCZK(sio.InsuredBase), 10),
				Poj:       strconv.FormatInt(totalInsurance, 10),
				Slev:      "0",
				Pojposlev: strconv.FormatInt(totalInsurance, 10),
				Zal:       strconv.FormatInt(prepayments, 10),
				Ned:       strconv.FormatInt(difference, 10),
			},

			Zal: Zal{
				Ved:  advanceKind,
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
				Por:    priznPor,
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

// secondaryActivity marks the months of secondary activity and the reason (ř. 13
// and ř. 14-19). Without secondary months the element stays empty.
func secondaryActivity(months domain.ActivityMonths, reason domain.SecondaryActivityReason) Vedc {
	if months == 0 {
		return Vedc{}
	}
	flags := mainActivityMonthFlags(months)
	reasons := map[domain.SecondaryActivityReason]*string{}
	vedc := Vedc{
		M1: flags.M1, M2: flags.M2, M3: flags.M3, M4: flags.M4, M5: flags.M5, M6: flags.M6,
		M7: flags.M7, M8: flags.M8, M9: flags.M9, M10: flags.M10, M11: flags.M11, M12: flags.M12,
		M13: flags.M13,
		Zam: "N", Duchod: "N", Pdite: "N", PPM: "N", Pece: "N", Ndite: "N",
	}
	reasons[domain.SecondaryReasonEmployment] = &vedc.Zam
	reasons[domain.SecondaryReasonPension] = &vedc.Duchod
	reasons[domain.SecondaryReasonChildCare] = &vedc.Pdite
	reasons[domain.SecondaryReasonMaternity] = &vedc.PPM
	reasons[domain.SecondaryReasonCare] = &vedc.Pece
	reasons[domain.SecondaryReasonStudent] = &vedc.Ndite
	if field, ok := reasons[reason]; ok {
		*field = "A"
	}
	return vedc
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
