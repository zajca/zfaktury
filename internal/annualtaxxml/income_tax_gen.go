package annualtaxxml

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/zajca/zfaktury/internal/domain"
)

// stripDICPrefix removes the leading 2-letter ISO country code (e.g. "CZ") from a DIC.
// EPO DPFDP7 schema expects the numeric-only portion ([0-9]{1,10}).
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

// splitChildBenefit splits child benefit into the sleva component (form ř.73)
// and the daňový bonus component (ř.76). Sleva eats tax up to its remaining value;
// any excess child benefit becomes a refundable bonus.
func splitChildBenefit(taxAfterCredits, childBenefit int64) (slevy35c, danbonus int64) {
	if taxAfterCredits < 0 {
		taxAfterCredits = 0
	}
	if childBenefit <= taxAfterCredits {
		return childBenefit, 0
	}
	return taxAfterCredits, childBenefit - taxAfterCredits
}

// GenerateIncomeTaxXML produces EPO XML bytes from an IncomeTaxReturn and settings map.
func GenerateIncomeTaxXML(itr *domain.IncomeTaxReturn, settings map[string]string) ([]byte, error) {
	if itr == nil {
		return nil, fmt.Errorf("income tax return is nil: %w", domain.ErrInvalidInput)
	}

	// Whole-CZK conversions of every domain field referenced below.
	revenue := ToWholeCZK(itr.TotalRevenue)
	expenses := ToWholeCZK(itr.UsedExpenses)
	zd7 := revenue - expenses // ř.37 / ř.113 -- partial tax base from §7
	if zd7 < 0 {
		zd7 = 0
	}
	zd8 := ToWholeCZK(itr.CapitalIncomeNet) // ř.38
	zd10 := ToWholeCZK(itr.OtherIncomeNet)  // ř.40
	uhrn := zd7 + zd8 + zd10                // ř.41 (no §9 rental income tracked yet)
	zakldan23 := uhrn                       // ř.42 -- assumes §6 employment base = 0
	zakldanLoss := zakldan23                // ř.45 -- no carry-forward losses applied
	totalDeductions := ToWholeCZK(itr.TotalDeductions)
	zdsniz := zakldanLoss - totalDeductions // ř.55
	if zdsniz < 0 {
		zdsniz = 0
	}
	taxBaseRounded := (zdsniz / 100) * 100 // ř.56 (round down to whole 100 CZK)
	dan16 := ToWholeCZK(itr.TotalTax)      // ř.57 -- §16 tax (already computed by service)
	daSlezap := dan16                      // ř.60 -- tax rounded up; the service already rounds

	uhrnSlevy35ba := ToWholeCZK(itr.TotalCredits) // ř.70
	taxAfter35ba := daSlezap - uhrnSlevy35ba      // ř.71
	if taxAfter35ba < 0 {
		taxAfter35ba = 0
	}
	childBenefit := ToWholeCZK(itr.ChildBenefit)                        // ř.72
	slevy35c, danbonus := splitChildBenefit(taxAfter35ba, childBenefit) // ř.73 / ř.76
	taxAfter35c := taxAfter35ba - slevy35c                              // ř.74
	danCelk := taxAfter35c                                              // ř.75 (= ř.74 + ř.74a; no separate-base tax tracked)

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
			DaSlezap:      daSlezap,
			SlevaRp:       ToWholeCZK(itr.CreditBasic),
			UhrnSlevy35ba: uhrnSlevy35ba,
			DaSlevy35ba:   taxAfter35ba,
			KcDazvyhod:    childBenefit,
			KcSlevy35c:    slevy35c,
			DaSlevy35c:    taxAfter35c,
			KcDanCelk:     danCelk,
			KcDanbonus:    danbonus,
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
			KcZd7:       zd7,
			KcZakldan8:  zd8,
			KcZd10:      zd10,
			KcUhrn:      uhrn,
			KcZakldan23: zakldan23,
			KcZakldan:   zakldanLoss,
		},
		VetaS: &DPFOVetaS{
			KcOp28_5:  ToWholeCZK(itr.DeductionMortgage),
			KcOp15_13: ToWholeCZK(itr.DeductionLifeInsurance),
			KcOp15_12: ToWholeCZK(itr.DeductionPension),
			KcOp15_8:  ToWholeCZK(itr.DeductionDonation),
			KcOdcelk:  totalDeductions,
			KcZdsniz:  zdsniz,
			KcZdzaokr: taxBaseRounded,
			DaDan16:   dan16,
		},
		VetaB: &DPFOVetaB{
			Priloha1: "1",
		},
		VetaT: buildPriloha1(itr, revenue, expenses, zd7),
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

// buildPriloha1 fills VetaT (Příloha č. 1). The XSD uses two mutually exclusive
// sections for revenue and expenses: the "actual expenses" pair (kc_prij7/kc_vyd7)
// must NOT be combined with the "flat-rate" pair (pr_prij7/pr_vyd7) or EPO raises
// a critical-control error.
func buildPriloha1(itr *domain.IncomeTaxReturn, revenue, expenses, zd7 int64) *DPFOVetaT {
	v := &DPFOVetaT{KcZd7p: zd7}
	if itr.FlatRatePercent > 0 {
		v.PrPrij7 = revenue
		v.PrVyd7 = expenses
		v.Vyd7proc = "A"
		v.PrSazba = fmt.Sprintf("%d", itr.FlatRatePercent)
	} else {
		v.KcPrij7 = revenue
		v.KcVyd7 = expenses
		v.Vyd7proc = "N"
	}
	return v
}
