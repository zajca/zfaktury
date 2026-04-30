package annualtaxxml

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/zajca/zfaktury/internal/domain"
)

// stripDICPrefix removes the leading 2-letter ISO country code (e.g. "CZ") from a DIC.
// EPO DPFDP5 schema expects the numeric-only portion ([0-9]{1,10}).
func stripDICPrefix(dic string) string {
	dic = strings.TrimSpace(dic)
	if len(dic) > 2 {
		c0, c1 := dic[0], dic[1]
		if (c0 >= 'A' && c0 <= 'Z') || (c0 >= 'a' && c0 <= 'z') {
			if (c1 >= 'A' && c1 <= 'Z') || (c1 >= 'a' && c1 <= 'z') {
				return dic[2:]
			}
		}
	}
	return dic
}

// GenerateIncomeTaxXML produces EPO XML bytes from an IncomeTaxReturn and settings map.
func GenerateIncomeTaxXML(itr *domain.IncomeTaxReturn, settings map[string]string) ([]byte, error) {
	if itr == nil {
		return nil, fmt.Errorf("income tax return is nil: %w", domain.ErrInvalidInput)
	}

	revenue := ToWholeCZK(itr.TotalRevenue)
	expenses := ToWholeCZK(itr.UsedExpenses)
	taxBase := ToWholeCZK(itr.TaxBase)
	taxBaseRounded := ToWholeCZK(itr.TaxBaseRounded)

	doc := &DPFDP7{
		VerzePis: "01.01.02",
		VetaD: DPFOVetaD{
			Dokument:      "DP7",
			KUladis:       "DPF",
			Rok:           itr.Year,
			DapTyp:        DPFOFilingTypeCode(itr.FilingType),
			CUfoCil:       settings["financni_urad_code"],
			PlnMoc:        "N",
			Audit:         "N",
			ZdobdOd:       fmt.Sprintf("1.1.%d", itr.Year),
			ZdobdDo:       fmt.Sprintf("31.12.%d", itr.Year),
			DaSlezap:      ToWholeCZK(itr.TotalTax),
			SlevaRp:       ToWholeCZK(itr.CreditBasic),
			UhrnSlevy35ba: ToWholeCZK(itr.TotalCredits),
			DaSlevy35ba:   ToWholeCZK(itr.TaxAfterCredits),
			KcDazvyhod:    ToWholeCZK(itr.ChildBenefit),
			DaSlevy35c:    ToWholeCZK(itr.TaxAfterBenefit),
			KcZalpred:     ToWholeCZK(itr.Prepayments),
			KcZbyvpred:    ToWholeCZK(itr.TaxDue),
		},
		VetaP: DPFOVetaP{
			Jmeno:    settings["taxpayer_first_name"],
			Prijmeni: settings["taxpayer_last_name"],
			RodC:     settings["taxpayer_birth_number"],
			DIC:      stripDICPrefix(settings["dic"]),
			Ulice:    settings["taxpayer_street"],
			CPop:     settings["taxpayer_house_number"],
			NazObce:  settings["taxpayer_city"],
			PSC:      settings["taxpayer_postal_code"],
			KStat:    "CZ",
			Stat:     "ČESKÁ REPUBLIKA",
		},
		VetaO: &DPFOVetaO{
			KcZd7:       taxBase,
			KcZakldan23: taxBaseRounded,
			KcZakldan:   taxBaseRounded,
		},
		VetaS: &DPFOVetaS{
			KcZdzaokr: taxBaseRounded,
			KcOp28_5:  ToWholeCZK(itr.DeductionMortgage),
			KcOp15_13: ToWholeCZK(itr.DeductionLifeInsurance),
			KcOp15_12: ToWholeCZK(itr.DeductionPension),
			KcOp15_8:  ToWholeCZK(itr.DeductionDonation),
		},
		VetaB: &DPFOVetaB{
			Priloha1: "1",
		},
		VetaT: &DPFOVetaT{
			PrPrij7: revenue,
			PrVyd7:  expenses,
		},
	}

	pisemnost := &DPFOPisemnost{
		NazevSW: "ZFaktury",
		VerzeSW: "1.0",
		DPFDP7:  doc,
	}

	output, err := xml.MarshalIndent(pisemnost, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling income tax return XML: %w", err)
	}

	header := []byte(xml.Header)
	result := make([]byte, 0, len(header)+len(output))
	result = append(result, header...)
	result = append(result, output...)

	return result, nil
}
