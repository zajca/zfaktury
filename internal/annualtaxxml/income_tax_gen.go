package annualtaxxml

import (
	"encoding/xml"
	"fmt"
	"strconv"
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

	// Flat-rate expenses: compute directly from revenue × rate / 100 with
	// round-half-up. The calc service stores the value truncated, which differs
	// from EPO's expected rounding (e.g. 1 550 848 × 60 % = 930 508,80 → 930 509).
	var expenses int64
	if itr.FlatRatePercent > 0 {
		expenses = (revenue*int64(itr.FlatRatePercent) + 50) / 100
	} else {
		expenses = ToWholeCZK(itr.UsedExpenses)
	}
	zd7 := revenue - expenses // ř.37 / ř.113 -- partial tax base from §7
	loss := int64(0)          // ř.61 -- daňová ztráta; § 7 pre-loss tracked here
	if zd7 < 0 {
		loss = -zd7
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
	daSlevy := dan16                       // ř.60 -- daň zaokrouhlená nahoru (service already rounds)

	uhrnSlevy35ba := ToWholeCZK(itr.TotalCredits) // ř.70
	creditBasic := ToWholeCZK(itr.CreditBasic)    // ř.64 (§35ba 1a)
	taxAfter35ba := daSlevy - uhrnSlevy35ba       // ř.71
	if taxAfter35ba < 0 {
		taxAfter35ba = 0
	}
	childBenefit := ToWholeCZK(itr.ChildBenefit)                        // ř.72
	slevy35c, danbonus := splitChildBenefit(taxAfter35ba, childBenefit) // ř.73 / ř.76
	taxAfter35c := taxAfter35ba - slevy35c                              // ř.74
	danCelk := taxAfter35c                                              // ř.75 (= ř.74 + ř.74a; no separate-base tax tracked)
	// ř.77 = ř.75 - ř.76 (min 0); ř.77a = ř.76 - ř.75 (min 0). At most one is non-zero.
	danPoDb := danCelk - danbonus
	dbPoOdpd := int64(0)
	if danPoDb < 0 {
		dbPoOdpd = -danPoDb
		danPoDb = 0
	}

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
			KcDztrata:     loss,
			DaSlevy:       daSlevy,
			KcOp15_1a:     creditBasic,
			UhrnSlevy35ba: uhrnSlevy35ba,
			DaSlevy35ba:   taxAfter35ba,
			KcDazvyhod:    childBenefit,
			KcSlevy35c:    slevy35c,
			DaSlevy35c:    taxAfter35c,
			KcDanCelk:     danCelk,
			KcDanbonus:    danbonus,
			KcDanPoDb:     danPoDb,
			KcDbPoOdpd:    dbPoOdpd,
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
		VetaT: buildPriloha1(itr, settings, revenue, expenses, zd7),
	}

	// Příloha č. 2 (§9 + §10). EPO requires it whenever ř.39 or ř.40 (kc_zd9 / kc_zd10)
	// is filled. We currently track only §10 (other income from securities/crypto).
	if v, j := buildPriloha2(itr); v != nil {
		doc.VetaV = v
		doc.VetaJ = j
		if doc.VetaB != nil {
			doc.VetaB.Priloha2 = "1"
		}
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

// buildPriloha2 fills VetaV (§10 summary) and VetaJ (per-type detail rows) for Příloha č. 2.
// Returns (nil, nil) when there is no §10 income to report -- the attachment is then omitted
// and kc_zd10 is suppressed in VetaO via omitempty.
//
// Callers track only an aggregate "other income" net base; the schema requires at least one
// VetaJ row, so we emit a single row using kod_dr_prij10="D" (prodej cenných papírů), which
// matches the typical OSVC investment-income use case (broker statements).
func buildPriloha2(itr *domain.IncomeTaxReturn) (*DPFOVetaV, []DPFOVetaJ) {
	gross := ToWholeCZK(itr.OtherIncomeGross)
	if gross <= 0 {
		return nil, nil
	}
	exp := ToWholeCZK(itr.OtherIncomeExpenses)
	if exp > gross {
		exp = gross // §10 expenses cannot exceed revenue per income type
	}
	diff := gross - exp
	v := &DPFOVetaV{
		KcPrij10: gross,
		KcVyd10:  exp,
		KcZd10p:  diff,
	}
	j := []DPFOVetaJ{{
		KodDrPrij10: "D",
		DruhPrij10:  "Prodej cenných papírů",
		Prijmy10:    gross,
		Vydaje10:    exp,
		Rozdil10:    diff,
	}}
	return v, j
}

// buildPriloha1 fills VetaT (Příloha č. 1). The XSD uses two mutually exclusive
// sections for revenue and expenses: the "actual expenses" pair (kc_prij7/kc_vyd7)
// must NOT be combined with the "flat-rate" pair (pr_prij7/pr_vyd7) or EPO raises
// a critical-control error.
//
// EPO also requires:
//   - c_nace (NACE code from číselník okec) and m_podnik (months active) for
//     identification of the main activity (část B header) when §7 income exists.
//   - celk_pr_prij7 / celk_pr_vyd7 totals matching pr_prij7 / pr_vyd7 plus any
//     additional Vetac rows (we currently emit a single main row, so the totals
//     equal the main row values).
//
// NACE code lookup order:
//  1. main_activity_nace (DPFO-specific override) -- if user wants a different
//     code on the income tax form than on the VAT form
//  2. c_okec (shared NACE code already used for VAT XML) -- this is the existing
//     "NACE kód činnosti" field in firma settings (e.g. 582900)
//
// If both are empty, c_nace is omitted; EPO will warn but the user can fill it
// manually in the portal. Invented defaults must NOT be used because EPO
// validates against the okec číselník.
func buildPriloha1(itr *domain.IncomeTaxReturn, settings map[string]string, revenue, expenses, zd7 int64) *DPFOVetaT {
	months := 12
	if v, err := strconv.Atoi(settings["main_activity_months"]); err == nil && v > 0 && v <= 12 {
		months = v
	}
	nace := settings["main_activity_nace"]
	if nace == "" {
		nace = settings["c_okec"]
	}
	v := &DPFOVetaT{
		CNace:   nace,
		MPodnik: months,
		KcZd7p:  zd7,
	}
	if itr.FlatRatePercent > 0 {
		v.PrPrij7 = revenue
		v.PrVyd7 = expenses
		v.Vyd7proc = "A"
		v.PrSazba = fmt.Sprintf("%d", itr.FlatRatePercent)
		v.CelkPrPrij7 = revenue
		v.CelkPrVyd7 = expenses
	} else {
		v.KcPrij7 = revenue
		v.KcVyd7 = expenses
		v.Vyd7proc = "N"
	}
	return v
}
