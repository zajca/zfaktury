package annualtaxxml

import "encoding/xml"

// EPO income tax return XML types for Czech tax authority (Financni sprava).
// Schema reference: https://adisspr.mfcr.cz/adis/jepo/schema/dpfdp7_epo2.xsd
// Form: "Priznani k dani z prijmu fyzickych osob" (DPFDP7) -- valid for tax years 2024 and 2025.
//
// Each XML attribute below is annotated with the row of paper form 25 5405 it represents.
// EPO portal runs critical-control formula checks against these values, so the generator
// must emit every intermediate row referenced by the formulas.

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
	VerzePis string      `xml:"verzePis,attr"`
	VetaD    DPFOVetaD   `xml:"VetaD"`
	VetaP    DPFOVetaP   `xml:"VetaP"`
	VetaO    *DPFOVetaO  `xml:"VetaO,omitempty"`
	VetaS    *DPFOVetaS  `xml:"VetaS,omitempty"`
	VetaB    *DPFOVetaB  `xml:"VetaB,omitempty"`
	VetaT    *DPFOVetaT  `xml:"VetaT,omitempty"`
	VetaV    *DPFOVetaV  `xml:"VetaV,omitempty"`
	VetaJ    []DPFOVetaJ `xml:"VetaJ,omitempty"`
}

// DPFOVetaD contains filing metadata, tax credits and final settlement totals.
type DPFOVetaD struct {
	// Filing metadata (required by XSD).
	Dokument string `xml:"dokument,attr"`
	KUladis  string `xml:"k_uladis,attr"`
	Rok      int    `xml:"rok,attr"`
	DapTyp   string `xml:"dap_typ,attr"`
	CUfoCil  string `xml:"c_ufo_cil,attr"`
	PlnMoc   string `xml:"pln_moc,attr"`
	Audit    string `xml:"audit,attr"`

	// Tax period (XSD critical control: zdobd_od=1.1.<rok>, zdobd_do=31.12.<rok>).
	ZdobdOd string `xml:"zdobd_od,attr,omitempty"`
	ZdobdDo string `xml:"zdobd_do,attr,omitempty"`

	// Tax computation (sec. 4-5). Attribute names verified against ADIS popis_struktury
	// for DPFDP7 (2024+):
	//   - ř.58 (daň podle §16 přenesená z VetaS.da_dan16) = da_slezap (decimal 17/2)
	//   - ř.60 (daň celkem zaokrouhlená nahoru, ř.58 + ř.59) = da_celod13 (integer)
	//   - ř.62 (sleva podle §35 odst.1 -- zaměstnanci ZTP) = da_slevy (NOT for OSVC)
	//   - ř.63 (sleva podle §35a/35b -- investiční pobídky) = sleva_rp (NOT for OSVC)
	//   - ř.64 (základní sleva na poplatníka §35ba 1a) = kc_op15_1a
	KcDztrata     int64  `xml:"kc_dztrata,attr"`     // ř. 61 -- daňová ztráta (zaokr. nahoru, bez znaménka mínus)
	DaSlezap      string `xml:"da_slezap,attr"`      // ř. 58 -- daň podle §16 (formát "Kc.00", decimal 2 frac digits)
	DaCelod13     int64  `xml:"da_celod13,attr"`     // ř. 60 -- daň celkem zaokrouhlená na celé Kč nahoru
	KcOp15_1a     int64  `xml:"kc_op15_1a,attr"`     // ř. 64 -- základní sleva na poplatníka (§ 35ba 1a) -- 30 840 Kč
	UhrnSlevy35ba int64  `xml:"uhrn_slevy35ba,attr"` // ř. 70 -- úhrn slev podle § 35ba
	DaSlevy35ba   int64  `xml:"da_slevy35ba,attr"`   // ř. 71 -- daň po slevách (= ř.60 - ř.70)
	// Per-child months claimed (used by EPO formula for ř.72). Each *2/*3 suffix selects
	// the child order (1st/2nd/3rd+); ztpp variants apply the ZTP/P doubling.
	// EPO formula: kc_dazvyhod = (m_deti × 1267 + m_deti2 × 1860 + m_deti3 × 2320)
	//             + 2 × (m_detiztpp × 1267 + m_detiztpp2 × 1860 + m_detiztpp3 × 2320)
	MDeti      int   `xml:"m_deti,attr,omitempty"`      // měsíců s 1. dítětem (běžným)
	MDeti2     int   `xml:"m_deti2,attr,omitempty"`     // měsíců s 2. dítětem (běžným)
	MDeti3     int   `xml:"m_deti3,attr,omitempty"`     // měsíců s 3+ dítětem (běžným)
	MDetiZtpp  int   `xml:"m_detiztpp,attr,omitempty"`  // měsíců s 1. dítětem ZTP/P
	MDetiZtpp2 int   `xml:"m_detiztpp2,attr,omitempty"` // měsíců s 2. dítětem ZTP/P
	MDetiZtpp3 int   `xml:"m_detiztpp3,attr,omitempty"` // měsíců s 3+ dítětem ZTP/P
	KcDazvyhod int64 `xml:"kc_dazvyhod,attr"`           // ř. 72 -- daňové zvýhodnění na děti
	KcSlevy35c int64 `xml:"kc_slevy35c,attr"`           // ř. 73 -- sleva na děti uplatněná do výše daně
	DaSlevy35c int64 `xml:"da_slevy35c,attr"`           // ř. 74 -- daň po slevě podle § 35c (= ř.71 - ř.73)
	KcDanCelk  int64 `xml:"kc_dan_celk,attr"`           // ř. 75 -- daň celkem (= ř.74 + ř.74a)
	KcDanbonus int64 `xml:"kc_danbonus,attr"`           // ř. 76 -- daňový bonus (= ř.72 - ř.73)
	KcDanPoDb  int64 `xml:"kc_dan_po_db,attr"`          // ř. 77 -- daň celkem po úpravě o bonus (= ř.75 - ř.76, min 0)
	KcDbPoOdpd int64 `xml:"kc_db_po_odpd,attr"`         // ř. 77a -- daňový bonus po odpočtu daně (= ř.76 - ř.75, min 0)
	KcZalpred  int64 `xml:"kc_zalpred,attr"`            // ř. 84 -- úhrn sražených záloh
	KcZbyvpred int64 `xml:"kc_zbyvpred,attr"`           // ř. 91 -- zbývá doplatit / přeplatek
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

// DPFOVetaO contains per-section tax-base inputs from §6 / §7 / §8 / §9 / §10
// and the consolidated tax base. §8/§9/§10 fields use omitempty: emitting "0"
// triggers the EPO control "if ř.39 or ř.40 is filled, Příloha 2 must accompany".
type DPFOVetaO struct {
	KcZd7       int64 `xml:"kc_zd7,attr"`                // ř. 37 -- dílčí základ daně §7 (= ř.113 Přílohy 1)
	KcZakldan8  int64 `xml:"kc_zakldan8,attr,omitempty"` // ř. 38 -- §8 capital income net base
	KcZd9       int64 `xml:"kc_zd9,attr,omitempty"`      // ř. 39 -- §9 rental income net base (Příloha 2 required)
	KcZd10      int64 `xml:"kc_zd10,attr,omitempty"`     // ř. 40 -- §10 other income net base (Příloha 2 required)
	KcUhrn      int64 `xml:"kc_uhrn,attr"`               // ř. 41 -- úhrn ř.37+38+39+40
	KcZakldan23 int64 `xml:"kc_zakldan23,attr"`          // ř. 42 -- celkový základ daně (= ř.36 + max(0,ř.41))
	KcZakldan   int64 `xml:"kc_zakldan,attr"`            // ř. 45 -- ZD po odpočtu ztráty
}

// DPFOVetaS contains §15 deductions, base after deductions, rounded base and § 16 tax.
type DPFOVetaS struct {
	KcOp28_5  int64 `xml:"kc_op28_5,attr"`  // ř. 47 -- úroky z hypoték / stavebního spoření
	KcOp15_13 int64 `xml:"kc_op15_13,attr"` // ř. 49 -- soukromé životní pojištění
	KcOp15_12 int64 `xml:"kc_op15_12,attr"` // ř. 48 -- penzijní spoření / pojištění
	KcOp15_8  int64 `xml:"kc_op15_8,attr"`  // ř. 46 -- bezúplatná plnění (dary)
	KcOdcelk  int64 `xml:"kc_odcelk,attr"`  // ř. 54 -- úhrn nezdanitelných částí (ř.46+47+...+53)
	KcZdsniz  int64 `xml:"kc_zdsniz,attr"`  // ř. 55 -- ZD snížený o nezdanitelné části (= ř.45 - ř.54)
	KcZdzaokr int64 `xml:"kc_zdzaokr,attr"` // ř. 56 -- ZD zaokrouhlený na celá sta Kč dolů
	DaDan16   int64 `xml:"da_dan16,attr"`   // ř. 57 -- daň podle § 16
}

// DPFOVetaB declares which attachments accompany the return.
// XSD critical controls: priloha1="1" required when VetaO.kc_zd7 is filled,
// priloha2="1" required when VetaO.kc_zd9 or kc_zd10 is filled.
type DPFOVetaB struct {
	Priloha1 string `xml:"priloha1,attr,omitempty"`
	Priloha2 string `xml:"priloha2,attr,omitempty"`
}

// DPFOVetaV is the §10 summary of Příloha č. 2 (other income / "ostatní příjmy").
// Required when ř.40 (kc_zd10 in VetaO) is filled.
type DPFOVetaV struct {
	KcPrij10 int64 `xml:"kc_prij10,attr"` // sum of revenue across §10 income types
	KcVyd10  int64 `xml:"kc_vyd10,attr"`  // sum of expenses (capped per type at revenue)
	KcZd10p  int64 `xml:"kc_zd10p,attr"`  // dílčí ZD §10 transferred to ř.40
}

// DPFOVetaJ is one row of the §10 detail table in Příloha č. 2.
// kod_dr_prij10 codes: A=příležitostná činnost, B=prodej nemovitostí, C=prodej movitých věcí,
// D=prodej cenných papírů, E=převod podle §10 odst.1 c), F=jiné ostatní příjmy,
// G=bezúplatné příjmy, H=loterie/tomboly podle §10 odst. 1 h) bod 1.
type DPFOVetaJ struct {
	KodDrPrij10 string `xml:"kod_dr_prij10,attr,omitempty"` // type code (A..H)
	DruhPrij10  string `xml:"druh_prij10,attr,omitempty"`   // textual description
	Prijmy10    int64  `xml:"prijmy10,attr"`                // revenue per row
	Vydaje10    int64  `xml:"vydaje10,attr"`                // expenses per row (capped at revenue)
	Rozdil10    int64  `xml:"rozdil10,attr"`                // revenue minus expenses (>= 0 when summed)
	Kod10       string `xml:"kod10,attr,omitempty"`         // P/S/Z/N source flag
}

// DPFOVetaT is Priloha c. 1 -- detail of §7 business income.
// The form has TWO mutually exclusive sections for revenue/expenses:
//   - Oddíl B/1 (kc_prij7 ř.101 + kc_vyd7 ř.102): when expenses are kept in tax records.
//   - Oddíl B/2 (pr_prij7 + pr_vyd7 + vyd7proc ř.104): when applying flat-rate %.
//
// Filling both at once triggers a critical control. The generator picks one based on
// IncomeTaxReturn.FlatRatePercent.
type DPFOVetaT struct {
	// Část B header (main activity identification) -- required when §7 income is reported.
	CNace   string `xml:"c_nace,attr,omitempty"`   // NACE code (číselník okec)
	MPodnik int    `xml:"m_podnik,attr,omitempty"` // počet měsíců provozu činnosti (default 12)

	// Oddíl 2 part B/1 -- "actual expenses" filers (mutually exclusive with B/2).
	KcPrij7 int64 `xml:"kc_prij7,attr,omitempty"` // ř. 101 -- příjmy (actual-expense filers)
	KcVyd7  int64 `xml:"kc_vyd7,attr,omitempty"`  // ř. 102 -- výdaje (actual-expense filers)

	// Oddíl 2 part B/2 -- "flat-rate %" filers. EPO requires the Total* helper attributes
	// (celk_pr_prij7/celk_pr_vyd7) to equal the sum of the main row + any extra Vetac rows.
	PrPrij7     int64  `xml:"pr_prij7,attr,omitempty"`      // hlavní řádek -- příjmy (flat-rate filers)
	PrVyd7      int64  `xml:"pr_vyd7,attr,omitempty"`       // hlavní řádek -- výdaje (flat-rate filers)
	Vyd7proc    string `xml:"vyd7proc,attr,omitempty"`      // A/N -- "applying flat-rate %?" flag
	PrSazba     string `xml:"pr_sazba,attr,omitempty"`      // numeric % rate (e.g. 60, 80, 40, 30)
	CelkPrPrij7 int64  `xml:"celk_pr_prij7,attr,omitempty"` // celkem příjmy -- součet hlavní + Vetac.prijmy7
	CelkPrVyd7  int64  `xml:"celk_pr_vyd7,attr,omitempty"`  // celkem výdaje -- součet hlavní + Vetac.vydaje7

	KcZd7p   int64 `xml:"kc_zd7p,attr"`             // ř. 113 -- dílčí ZD §7 přenesený na ř. 37
	KcCisobr int64 `xml:"kc_cisobr,attr,omitempty"` // ř. 100 -- počet samostatných listů (default 0)
}
