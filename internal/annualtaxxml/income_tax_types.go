package annualtaxxml

import "encoding/xml"

// EPO income tax return XML types for Czech tax authority (Financni sprava).
// Schema reference: https://adisspr.mfcr.cz/adis/jepo/schema/dpfdp7_epo2.xsd
// Form: "Priznani k dani z prijmu fyzickych osob" (DPFDP7) -- valid for tax years 2024 and 2025.

// DPFOPisemnost is the root element of the EPO income tax return XML document.
type DPFOPisemnost struct {
	XMLName xml.Name `xml:"Pisemnost"`
	NazevSW string   `xml:"nazevSW,attr"`
	VerzeSW string   `xml:"verzeSW,attr"`
	DPFDP7  *DPFDP7  `xml:"DPFDP7"`
}

// DPFDP7 represents the income tax return form (Priznani k dani z prijmu FO).
// Element order matches the XSD sequence: VetaD, VetaP, VetaO, VetaS, VetaB, VetaT.
type DPFDP7 struct {
	VerzePis string     `xml:"verzePis,attr"`
	VetaD    DPFOVetaD  `xml:"VetaD"`
	VetaP    DPFOVetaP  `xml:"VetaP"`
	VetaO    *DPFOVetaO `xml:"VetaO,omitempty"`
	VetaS    *DPFOVetaS `xml:"VetaS,omitempty"`
	VetaB    *DPFOVetaB `xml:"VetaB,omitempty"`
	VetaT    *DPFOVetaT `xml:"VetaT,omitempty"`
}

// DPFOVetaD contains filing metadata and final tax/credit/prepayment totals.
// Per DPFDP7 XSD, VetaD does NOT contain section §7 amounts, tax base or §15 deductions --
// those live in VetaO / VetaS / VetaT respectively.
type DPFOVetaD struct {
	// Filing metadata (required).
	Dokument string `xml:"dokument,attr"`
	KUladis  string `xml:"k_uladis,attr"`
	Rok      int    `xml:"rok,attr"`
	DapTyp   string `xml:"dap_typ,attr"`
	CUfoCil  string `xml:"c_ufo_cil,attr"`
	PlnMoc   string `xml:"pln_moc,attr"`
	Audit    string `xml:"audit,attr"`

	// Tax period (D.M.YYYY). XSD requires zdobd_od=1.1.<rok>, zdobd_do=31.12.<rok>.
	ZdobdOd string `xml:"zdobd_od,attr,omitempty"`
	ZdobdDo string `xml:"zdobd_do,attr,omitempty"`

	// Tax and credits (optional).
	DaSlezap      int64 `xml:"da_slezap,attr"`      // total tax (computed from tax base)
	SlevaRp       int64 `xml:"sleva_rp,attr"`       // basic taxpayer credit
	UhrnSlevy35ba int64 `xml:"uhrn_slevy35ba,attr"` // total §35ba credits
	DaSlevy35ba   int64 `xml:"da_slevy35ba,attr"`   // tax after §35ba credits
	KcDazvyhod    int64 `xml:"kc_dazvyhod,attr"`    // child benefit (§35c)
	DaSlevy35c    int64 `xml:"da_slevy35c,attr"`    // tax after child benefit
	KcZalpred     int64 `xml:"kc_zalpred,attr"`     // prepayments
	KcZbyvpred    int64 `xml:"kc_zbyvpred,attr"`    // amount due (+) or overpayment (-)
}

// DPFOVetaP contains taxpayer identification.
// Per XSD, dic must match pattern [0-9]{1,10} -- numeric portion only, no "CZ" prefix.
type DPFOVetaP struct {
	Jmeno    string `xml:"jmeno,attr"`
	Prijmeni string `xml:"prijmeni,attr"`
	RodC     string `xml:"rod_c,attr"`
	DIC      string `xml:"dic,attr"`
	Ulice    string `xml:"ulice,attr"`
	CPop     string `xml:"c_pop,attr"`
	NazObce  string `xml:"naz_obce,attr"`
	PSC      string `xml:"psc,attr"`
	KStat    string `xml:"k_stat,attr"`
	Stat     string `xml:"stat,attr"`
}

// DPFOVetaO contains tax base aggregation across income sections.
type DPFOVetaO struct {
	KcZd7       int64 `xml:"kc_zd7,attr"`       // partial tax base from §7 (business income)
	KcZakldan23 int64 `xml:"kc_zakldan23,attr"` // consolidated tax base
	KcZakldan   int64 `xml:"kc_zakldan,attr"`   // tax base after loss deduction
}

// DPFOVetaS contains the rounded tax base and §15 deductions (nezdanitelne casti).
// DPFDP7 (forms 2024+) no longer carries kc_op15_14 (union dues) -- the deduction
// was abolished by the 2024 consolidation package.
type DPFOVetaS struct {
	KcZdzaokr int64 `xml:"kc_zdzaokr,attr"` // tax base rounded down to 100 CZK
	KcOp28_5  int64 `xml:"kc_op28_5,attr"`  // mortgage / building-savings interest
	KcOp15_13 int64 `xml:"kc_op15_13,attr"` // private life insurance
	KcOp15_12 int64 `xml:"kc_op15_12,attr"` // pension contributions
	KcOp15_8  int64 `xml:"kc_op15_8,attr"`  // donations (bezuplatna plneni)
}

// DPFOVetaB declares which attachments accompany the return.
// XSD requires priloha1="1" when VetaO.kc_zd7 is filled.
type DPFOVetaB struct {
	Priloha1 string `xml:"priloha1,attr,omitempty"`
}

// DPFOVetaT is Priloha c. 1 -- detail of §7 business income.
type DPFOVetaT struct {
	PrPrij7 int64 `xml:"pr_prij7,attr"` // business revenue (prijmy z §7)
	PrVyd7  int64 `xml:"pr_vyd7,attr"`  // business expenses (vydaje z §7)
}
