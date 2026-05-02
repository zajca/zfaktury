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
// Element order matches the XSD sequence: VetaD, VetaP, VetaO, VetaS, VetaA, VetaB, VetaT.
type DPFDP7 struct {
	VerzePis string      `xml:"verzePis,attr"`
	VetaD    DPFOVetaD   `xml:"VetaD"`
	VetaP    DPFOVetaP   `xml:"VetaP"`
	VetaO    *DPFOVetaO  `xml:"VetaO,omitempty"`
	VetaS    *DPFOVetaS  `xml:"VetaS,omitempty"`
	VetaA    []DPFOVetaA `xml:"VetaA,omitempty"`
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

	// §6 employment income (závislá činnost) -- omitempty so taxpayers without
	// employment income emit XML identical to the pre-§6-support output.
	KcZalzavc    int64 `xml:"kc_zalzavc,attr,omitempty"`     // ř. 84 §6 -- sražené zálohy zaměstnavateli (po RZ refund)
	KcSraz64     int64 `xml:"kc_sraz_6_4,attr,omitempty"`    // ř. 87 -- sražená daň §36 odst.6 (rezident ČR)
	KcSrazRezEHP int64 `xml:"kc_sraz_rezehp,attr,omitempty"` // ř. 87a -- sražená daň §36 odst.7 nerezident EU/EHP (MVP: 0)
	KcVyplBonus  int64 `xml:"kc_vyplbonus,attr,omitempty"`   // ř. 89 -- úhrn vyplacených měsíčních daňových bonusů (Potvrzení ř.5+ř.13)
}

// DPFOVetaP contains taxpayer identification.
// Per XSD, dic must match pattern [0-9]{1,10} -- numeric portion only, no "CZ" prefix.
//
// CTelef and CPracufo are technically optional per XSD, but EPO emits non-blocking
// warnings when they are missing (control 361 for telephone, control 26 for the
// territorial workplace). Filling them keeps the submission clean.
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
	CTelef   string `xml:"c_telef,attr,omitempty"`   // telefonní číslo poplatníka (EPO warning 361)
	CPracufo string `xml:"c_pracufo,attr,omitempty"` // číslo územního pracoviště FÚ (EPO warning 26)
}

// DPFOVetaO contains per-section tax-base inputs from §6 / §7 / §8 / §9 / §10
// and the consolidated tax base. §8/§9/§10 fields use omitempty: emitting "0"
// triggers the EPO control "if ř.39 or ř.40 is filled, Příloha 2 must accompany".
//
// §6 employment income attributes (kc_prij6 / kc_prij6zahr / kc_dan_zah / kc_zd6 /
// kc_zd6p) all use omitempty so that taxpayers without employment income emit XML
// identical to the pre-§6-support output. kc_zakldan23 is always emitted because
// it represents ř.42 -- the consolidated tax base used by every downstream EPO
// formula control.
type DPFOVetaO struct {
	KcPrij6     int64 `xml:"kc_prij6,attr,omitempty"`     // ř. 31 -- úhrn příjmů §6 (Potvrzení vzor 33 ř.2+ř.4)
	KcPrij6zahr int64 `xml:"kc_prij6zahr,attr,omitempty"` // ř. 35 -- část ř.31 bez záloh dle §38h (informativní)
	KcDanZah    int64 `xml:"kc_dan_zah,attr,omitempty"`   // ř. 33 -- daň zaplacená v zahraničí (§6 odst.13)
	KcZd6       int64 `xml:"kc_zd6,attr,omitempty"`       // ř. 34/36 -- dílčí ZD §6 = ř.31 - ř.33
	KcZd6p      int64 `xml:"kc_zd6p,attr,omitempty"`      // §38f / Příloha 3 alokace §6 portionu pro zápočet zahr. daně (MVP: 0)
	KcZd7       int64 `xml:"kc_zd7,attr"`                 // ř. 37 -- dílčí základ daně §7 (= ř.113 Přílohy 1)
	KcZakldan8  int64 `xml:"kc_zakldan8,attr,omitempty"`  // ř. 38 -- §8 capital income net base
	KcZd9       int64 `xml:"kc_zd9,attr,omitempty"`       // ř. 39 -- §9 rental income net base (Příloha 2 required)
	KcZd10      int64 `xml:"kc_zd10,attr,omitempty"`      // ř. 40 -- §10 other income net base (Příloha 2 required)
	KcUhrn      int64 `xml:"kc_uhrn,attr"`                // ř. 41 -- úhrn ř.37+38+39+40
	KcZakldan23 int64 `xml:"kc_zakldan23,attr"`           // ř. 42 -- celkový základ daně (= ř.36 + max(0,ř.41))
	KcZakldan   int64 `xml:"kc_zakldan,attr"`             // ř. 45 -- ZD po odpočtu ztráty
}

// DPFOVetaS contains §15 deductions, base after deductions, rounded base and § 16 tax.
//
// EPO control 1509 ("Oddíl 3/ř.47 - není současně vyplněn počet měsíců a částka")
// requires m_uroky to be filled whenever kc_op28_5 (mortgage interest) is non-zero.
type DPFOVetaS struct {
	KcOp28_5  int64 `xml:"kc_op28_5,attr"`         // ř. 47 -- úroky z hypoték / stavebního spoření
	MUroky    int   `xml:"m_uroky,attr,omitempty"` // počet měsíců placení úroků (1..12, povinné když kc_op28_5 > 0)
	KcOp15_13 int64 `xml:"kc_op15_13,attr"`        // ř. 49 -- soukromé životní pojištění
	KcOp15_12 int64 `xml:"kc_op15_12,attr"`        // ř. 48 -- penzijní spoření / pojištění
	KcOp15_8  int64 `xml:"kc_op15_8,attr"`         // ř. 46 -- bezúplatná plnění (dary)
	KcOdcelk  int64 `xml:"kc_odcelk,attr"`         // ř. 54 -- úhrn nezdanitelných částí (ř.46+47+...+53)
	KcZdsniz  int64 `xml:"kc_zdsniz,attr"`         // ř. 55 -- ZD snížený o nezdanitelné části (= ř.45 - ř.54)
	KcZdzaokr int64 `xml:"kc_zdzaokr,attr"`        // ř. 56 -- ZD zaokrouhlený na celá sta Kč dolů
	DaDan16   int64 `xml:"da_dan16,attr"`          // ř. 57 -- daň podle § 16
}

// DPFOVetaA is one row of Tabulka č. 2 (oddíl 5) -- "Údaje o vyživovaných dětech
// žijících ve společně hospodařící domácnosti". One row per child. EPO derives the
// m_deti* aggregates in VetaD as column sums across these rows; emitting the
// aggregates in VetaD without matching VetaA rows triggers critical control 176
// ("hodnota Celkem počet měsíců ... se nerovná součtu počtu měsíců jednotlivých řádků").
//
// Each child fills exactly one (pocmes / ztpp{2,3}) slot determined by ChildOrder
// (1, 2, 3+) and the ZTP/P flag; the other slots stay 0 (and are omitted via omitempty).
//
// Identity: vyzdite_r_cislo (rodné číslo, [0-9]{1,10}) OR vyzdite_d_nar (birth date)
// must be filled. We emit r_cislo with non-digits stripped.
type DPFOVetaA struct {
	VyzditeJmeno    string `xml:"vyzdite_jmeno,attr"`
	VyzditePrijmeni string `xml:"vyzdite_prijmeni,attr"`
	VyzditeRCislo   string `xml:"vyzdite_r_cislo,attr,omitempty"`
	VyzditeDNar     string `xml:"vyzdite_d_nar,attr,omitempty"`
	VyzditePocmes   int    `xml:"vyzdite_pocmes,attr,omitempty"`  // 1. dítě bez ZTP/P
	VyzditeZtpp     int    `xml:"vyzdite_ztpp,attr,omitempty"`    // 1. dítě ZTP/P
	VyzditePocmes2  int    `xml:"vyzdite_pocmes2,attr,omitempty"` // 2. dítě bez ZTP/P
	VyzditeZtpp2    int    `xml:"vyzdite_ztpp2,attr,omitempty"`   // 2. dítě ZTP/P
	VyzditePocmes3  int    `xml:"vyzdite_pocmes3,attr,omitempty"` // 3+. dítě bez ZTP/P
	VyzditeZtpp3    int    `xml:"vyzdite_ztpp3,attr,omitempty"`   // 3+. dítě ZTP/P
}

// DPFOVetaB declares which attachments accompany the return.
// XSD critical controls: priloha1="1" required when VetaO.kc_zd7 is filled,
// priloha2="1" required when VetaO.kc_zd9 or kc_zd10 is filled.
//
// §6 employment income attachment counts:
//   - potv_zam: count of "Potvrzení o zdanitelných příjmech ze závislé činnosti"
//     (form 25 5460 vzor 33 -- zálohové)
//   - potv_36: count of "Potvrzení o vyplacených příjmech a sražené dani"
//     (form 25 5460/A vzor 12 -- srážkové), only counted when included in DAP
//   - potv_dazvyh: count of standalone "Potvrzení o vyplaceném daňovém bonusu"
//     forms (separate from vzor 33; MVP: 0 -- standalone bonus form upload OOS)
type DPFOVetaB struct {
	Priloha1   string `xml:"priloha1,attr,omitempty"`
	Priloha2   string `xml:"priloha2,attr,omitempty"`
	PotvZam    int    `xml:"potv_zam,attr,omitempty"`
	Potv36     int    `xml:"potv_36,attr,omitempty"`
	PotvDazvyh int    `xml:"potv_dazvyh,attr,omitempty"`
}

// DPFOVetaV is the §10 summary of Příloha č. 2 (other income / "ostatní příjmy").
// Required when ř.40 (kc_zd10 in VetaO) is filled.
//
// EPO has TWO parallel sets of summary attributes that must both be filled:
//   - kc_prij10 / kc_vyd10 / kc_zd10p  -- úhrny carried to ř.40 in main DAP
//   - uhrn_prijmy10 / uhrn_vydaje10 / uhrn_rozdil10  -- column-sum úhrny shown
//     at the bottom of the §10 table itself
//
// Controls 1821 / 1822 / 1823 fire when the table-bottom úhrns are missing or
// don't equal the sum of values in the corresponding VetaJ rows. We emit both
// trios with identical values; with our single-row VetaJ output they always
// match the row sums.
type DPFOVetaV struct {
	KcPrij10     int64 `xml:"kc_prij10,attr"`     // úhrn appears at bottom of §10 table column 2 (transferred to ř.40 calc)
	KcVyd10      int64 `xml:"kc_vyd10,attr"`      // úhrn at bottom of column 3 (capped per type at revenue)
	KcZd10p      int64 `xml:"kc_zd10p,attr"`      // úhrn of positive rozdíly (transferred to ř.40)
	UhrnPrijmy10 int64 `xml:"uhrn_prijmy10,attr"` // EPO control 1821 -- table column 2 sum
	UhrnVydaje10 int64 `xml:"uhrn_vydaje10,attr"` // EPO control 1822 -- table column 3 sum
	UhrnRozdil10 int64 `xml:"uhrn_rozdil10,attr"` // EPO control 1823 -- table column 4 sum (positives only)
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
//
// Critical EPO controls (kontroly přílohy 1):
//
//   - ř.101 (kc_prij7) and ř.102 (kc_vyd7) MUST always be filled when §7 income is
//     reported, regardless of whether actual expenses or the flat-rate % are used.
//     For flat-rate filers, ř.101 must equal celk_pr_prij7 (Total income from
//     Tabulka 2 of Section B) and ř.102 must equal celk_pr_vyd7 (Total expenses).
//
//   - ř.113 (kc_zd7p) MUST equal the formula:
//     ř.104 + ř.105 - ř.106 - ř.107 + ř.108 + ř.109 - ř.110 + ř.112
//     For a basic OSVC return all terms except ř.104 (kc_hosp_rozd) are 0, so
//     ř.113 = ř.104 = revenue - expenses (may be negative for a loss).
//
// The flat-rate detail (pr_prij7/pr_vyd7/vyd7proc/pr_sazba/celk_pr_prij7/celk_pr_vyd7)
// is only emitted when applying the flat-rate %.
type DPFOVetaT struct {
	// Část B header (main activity identification) -- required when §7 income is reported.
	CNace   string `xml:"c_nace,attr,omitempty"`   // NACE code (číselník okec)
	MPodnik int    `xml:"m_podnik,attr,omitempty"` // počet měsíců provozu činnosti (default 12)

	// Oddíl 2 -- always filled when §7 income is reported. ř.101 and ř.102 are the
	// totals of Tabulka 2 of Section B (sum across all activities).
	KcPrij7 int64 `xml:"kc_prij7,attr,omitempty"` // ř. 101 -- úhrn příjmů §7 (vždy)
	KcVyd7  int64 `xml:"kc_vyd7,attr,omitempty"`  // ř. 102 -- úhrn výdajů §7 (vždy)

	// Flat-rate detail rows of Tabulka 2 of Section B -- only when applying flat-rate %.
	// EPO requires the Total* helper attributes (celk_pr_prij7/celk_pr_vyd7) to equal
	// the sum of the main row + any extra Vetac rows, and to equal kc_prij7/kc_vyd7.
	PrPrij7     int64  `xml:"pr_prij7,attr,omitempty"`      // hlavní řádek -- příjmy (flat-rate filers)
	PrVyd7      int64  `xml:"pr_vyd7,attr,omitempty"`       // hlavní řádek -- výdaje (flat-rate filers)
	Vyd7proc    string `xml:"vyd7proc,attr,omitempty"`      // A/N -- "applying flat-rate %?" flag
	PrSazba     string `xml:"pr_sazba,attr,omitempty"`      // numeric % rate (e.g. 60, 80, 40, 30)
	CelkPrPrij7 int64  `xml:"celk_pr_prij7,attr,omitempty"` // celkem příjmy -- součet hlavní + Vetac.prijmy7
	CelkPrVyd7  int64  `xml:"celk_pr_vyd7,attr,omitempty"`  // celkem výdaje -- součet hlavní + Vetac.vydaje7

	KcHospRozd int64 `xml:"kc_hosp_rozd,attr"`        // ř. 104 -- výsledek hospodaření / rozdíl příjmů a výdajů
	KcZd7p     int64 `xml:"kc_zd7p,attr"`             // ř. 113 -- dílčí ZD §7 přenesený na ř. 37
	KcCisobr   int64 `xml:"kc_cisobr,attr,omitempty"` // ř. 100 -- počet samostatných listů (default 0)
}
