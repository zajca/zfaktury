package annualtaxxml

import (
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/zajca/zfaktury/internal/domain"
)

// GenerateSocialInsuranceXML produces CSSZ OSVC overview XML from a SocialInsuranceOverview
// and application settings containing taxpayer information.
func GenerateSocialInsuranceXML(sio *domain.SocialInsuranceOverview, settings map[string]string) ([]byte, error) {
	if sio == nil {
		return nil, fmt.Errorf("social insurance overview is nil: %w", domain.ErrInvalidInput)
	}

	rok := strconv.Itoa(sio.Year)

	assessmentBase := ToWholeCZK(sio.AssessmentBase)
	minAssessmentBase := ToWholeCZK(sio.MinAssessmentBase)
	finalAssessmentBase := ToWholeCZK(sio.FinalAssessmentBase)
	totalInsurance := ToWholeCZK(sio.TotalInsurance)
	prepayments := ToWholeCZK(sio.Prepayments)
	difference := ToWholeCZK(sio.Difference)
	totalRevenue := ToWholeCZK(sio.TotalRevenue)
	totalExpenses := ToWholeCZK(sio.TotalExpenses)
	newMonthlyPrepay := ToWholeCZK(sio.NewMonthlyPrepay)

	// Determine flat-rate flag for zal section.
	pauFlag := "N"
	if settings["flat_rate_expenses"] == "true" {
		pauFlag = "A"
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
			VerzeProtokolu:  "1",
		},
		Prehled: PrehledOSVC{
			For:  settings["cssz_code"],
			Dep:  "122",
			Rok:  rok,
			Typ:  CSSZFilingTypeCode(sio.FilingType),
			VSDP: "",
			Dat:  "",

			Client: Client{
				Name: ClientName{
					Fir: settings["taxpayer_first_name"],
					Sur: settings["taxpayer_last_name"],
					Tit: "",
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
				Druc: "H",
				Hlavc: Hlavc{
					MonthFlags: MonthFlags{M1: "A"},
				},
				Vedc:  Vedc{},
				Sleva: MonthFlags{M1: "n"},
			},

			PVV: PVV{
				Pri:  "1",
				Mesc: HVPair{H: "1", V: ""},
				Mesv: HVPair{H: "1", V: ""},
				Mesp: "1",
				Rdza: HVPair{
					H: strconv.FormatInt(totalRevenue, 10),
					V: strconv.FormatInt(totalExpenses, 10),
				},
				VVZ: HVPair{
					H: strconv.FormatInt(assessmentBase, 10),
					V: "0",
				},
				DVZ:   HVPair{H: "0", V: "0"},
				MVZ:   strconv.FormatInt(minAssessmentBase, 10),
				UVZ:   strconv.FormatInt(finalAssessmentBase, 10),
				Vzsu:  strconv.FormatInt(finalAssessmentBase, 10),
				Vzsvc: strconv.FormatInt(finalAssessmentBase, 10),
				Poj:   strconv.FormatInt(totalInsurance, 10),
				Zal:   strconv.FormatInt(prepayments, 10),
				Ned:   strconv.FormatInt(difference, 10),
			},

			Zal: Zal{
				Pau:  pauFlag,
				VZ:   strconv.FormatInt(finalAssessmentBase, 10),
				DP:   strconv.FormatInt(newMonthlyPrepay, 10),
				NP:   "0",
				Duch: "",
			},

			Pre: Pre{
				Vra: "0",
				BS:  PreBS{},
			},

			Prizn: Prizn{
				Pau:    "H",
				Pov:    "A",
				Elektr: "N",
				Por:    "N",
			},

			Spo: Spo{
				Name: SpoName{},
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
