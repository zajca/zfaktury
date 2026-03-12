package annualtaxxml

import "encoding/xml"

// EPO income tax return XML types for Czech tax authority (Financni sprava).
// Based on the EPO submission format for "Priznani k dani z prijmu fyzickych osob" (DPFDP5).

// DPFOPisemnost is the root element of the EPO income tax return XML document.
type DPFOPisemnost struct {
	XMLName xml.Name `xml:"Pisemnost"`
	NazevSW string   `xml:"nazevSW,attr"`
	VerzeSW string   `xml:"verzeSW,attr"`
	DPFDP5  *DPFDP5  `xml:"DPFDP5"`
}

// DPFDP5 represents the income tax return form (Priznani k dani z prijmu FO).
type DPFDP5 struct {
	VerzePis string    `xml:"verzePis,attr"`
	VetaD    DPFOVetaD `xml:"VetaD"`
	VetaP    DPFOVetaP `xml:"VetaP"`
}

// DPFOVetaD contains the tax calculation data for the income tax return.
type DPFOVetaD struct {
	// Filing metadata
	Dokument string `xml:"dokument,attr"`
	KUladis  string `xml:"k_uladis,attr"`
	Rok      int    `xml:"rok,attr"`
	DapTyp   string `xml:"dap_typ,attr"`
	CUfoCil  string `xml:"c_ufo_cil,attr"`
	PlnMoc   string `xml:"pln_moc,attr"`
	Audit    string `xml:"audit,attr"`

	// Section 7 - Business income
	KcZd7 int64 `xml:"kc_zd7,attr"` // tax base from business income
	PrZd7 int64 `xml:"pr_zd7,attr"` // revenue
	VyZd7 int64 `xml:"vy_zd7,attr"` // expenses

	// Tax calculation
	KcZakldan23 int64 `xml:"kc_zakldan23,attr"` // consolidated tax base
	KcZakldan   int64 `xml:"kc_zakldan,attr"`   // tax base after loss deduction
	KcZdzaokr   int64 `xml:"kc_zdzaokr,attr"`   // tax base rounded down to 100 CZK
	DaSlezap    int64 `xml:"da_slezap,attr"`    // total tax

	// Tax credits (slevy §35ba)
	SlevaRp       int64 `xml:"sleva_rp,attr"`       // basic taxpayer credit
	UhrnSlevy35ba int64 `xml:"uhrn_slevy35ba,attr"` // total credits
	DaSlevy35ba   int64 `xml:"da_slevy35ba,attr"`   // tax after credits

	// Child benefit (§35c)
	KcDazvyhod int64 `xml:"kc_dazvyhod,attr"` // child benefit
	DaSlevy35c int64 `xml:"da_slevy35c,attr"` // tax after benefit

	// Prepayments and result
	KcZalpred  int64 `xml:"kc_zalpred,attr"`  // prepayments
	KcZbyvpred int64 `xml:"kc_zbyvpred,attr"` // amount due or overpayment
}

// DPFOVetaP contains taxpayer identification for the income tax return.
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
