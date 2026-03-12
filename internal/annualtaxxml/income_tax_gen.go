package annualtaxxml

import (
	"encoding/xml"
	"fmt"

	"github.com/zajca/zfaktury/internal/domain"
)

// GenerateIncomeTaxXML produces EPO XML bytes from an IncomeTaxReturn and settings map.
func GenerateIncomeTaxXML(itr *domain.IncomeTaxReturn, settings map[string]string) ([]byte, error) {
	if itr == nil {
		return nil, fmt.Errorf("income tax return is nil: %w", domain.ErrInvalidInput)
	}

	taxBaseRounded := ToWholeCZK(itr.TaxBaseRounded)

	doc := &DPFOPisemnost{
		NazevSW: "ZFaktury",
		VerzeSW: "1.0",
		DPFDP5: &DPFDP5{
			VerzePis: "05.01",
			VetaD: DPFOVetaD{
				Dokument: "DP5",
				KUladis:  "DPF",
				Rok:      itr.Year,
				DapTyp:   DPFOFilingTypeCode(itr.FilingType),
				CUfoCil:  settings["financni_urad_code"],
				PlnMoc:   "A",
				Audit:    "N",

				// Section 7
				KcZd7: ToWholeCZK(itr.TaxBase),
				PrZd7: ToWholeCZK(itr.TotalRevenue),
				VyZd7: ToWholeCZK(itr.UsedExpenses),

				// Tax calculation (simple case: consolidated = section 7)
				KcZakldan23: taxBaseRounded,
				KcZakldan:   taxBaseRounded,
				KcZdzaokr:   taxBaseRounded,
				DaSlezap:    ToWholeCZK(itr.TotalTax),

				// Tax credits
				SlevaRp:       ToWholeCZK(itr.CreditBasic),
				UhrnSlevy35ba: ToWholeCZK(itr.TotalCredits),
				DaSlevy35ba:   ToWholeCZK(itr.TaxAfterCredits),

				// Child benefit
				KcDazvyhod: ToWholeCZK(itr.ChildBenefit),
				DaSlevy35c: ToWholeCZK(itr.TaxAfterBenefit),

				// Prepayments and result
				KcZalpred:  ToWholeCZK(itr.Prepayments),
				KcZbyvpred: ToWholeCZK(itr.TaxDue),
			},
			VetaP: DPFOVetaP{
				Jmeno:    settings["taxpayer_first_name"],
				Prijmeni: settings["taxpayer_last_name"],
				RodC:     settings["taxpayer_birth_number"],
				DIC:      settings["dic"],
				Ulice:    settings["taxpayer_street"],
				CPop:     settings["taxpayer_house_number"],
				NazObce:  settings["taxpayer_city"],
				PSC:      settings["taxpayer_postal_code"],
				KStat:    "CZ",
				Stat:     "\u010cESK\u00c1 REPUBLIKA",
			},
		},
	}

	output, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling income tax return XML: %w", err)
	}

	header := []byte(xml.Header)
	result := make([]byte, 0, len(header)+len(output))
	result = append(result, header...)
	result = append(result, output...)

	return result, nil
}
