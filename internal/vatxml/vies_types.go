package vatxml

import "encoding/xml"

// EPO XML types for Souhrnne hlaseni (VIES Recapitulative Statement), following
// the DPHSHV structure (dphshv_epo2.xsd).

// VIESPisemnost is the root XML element for a VIES summary submission.
type VIESPisemnost struct {
	XMLName xml.Name `xml:"Pisemnost"`
	DPHSHV  DPHSHV   `xml:"DPHSHV"`
}

// DPHSHV is the main container for a VIES summary (Souhrnne hlaseni k DPH).
type DPHSHV struct {
	VetaD VIESVetaD   `xml:"VetaD"`
	VetaP VIESVetaP   `xml:"VetaP"`
	VetaR []VIESVetaR `xml:"VetaR,omitempty"`
}

// VIESVetaD contains header information about the filing.
type VIESVetaD struct {
	Dokument    string `xml:"dokument,attr"`     // Document type, always SHV
	KUladis     string `xml:"k_uladis,attr"`     // Tax code, always DPH
	Rok         int    `xml:"rok,attr"`          // Year
	Ctvrt       int    `xml:"ctvrt,attr"`        // Quarter (1-4)
	ShviesForma string `xml:"shvies_forma,attr"` // Filing form: R=regular, N=follow-up
}

// VIESVetaP contains the taxpayer identification. c_ufo, dic and typ_ds are
// required by the schema; the rest is optional.
type VIESVetaP struct {
	CUfo     string `xml:"c_ufo,attr"`
	CPracufo string `xml:"c_pracufo,attr,omitempty"`
	DIC      string `xml:"dic,attr"`
	TypDS    string `xml:"typ_ds,attr"`
	Jmeno    string `xml:"jmeno,attr,omitempty"`
	Prijmeni string `xml:"prijmeni,attr,omitempty"`
	Ulice    string `xml:"ulice,attr,omitempty"`
	CPop     string `xml:"c_pop,attr,omitempty"`
	NazObce  string `xml:"naz_obce,attr,omitempty"`
	PSC      string `xml:"psc,attr,omitempty"`
	Stat     string `xml:"stat,attr,omitempty"`
}

// VIESVetaR contains one line of the summary: the totals for one EU partner and
// type of supply.
type VIESVetaR struct {
	KStat      string `xml:"k_stat,attr"`      // Country code (2-letter)
	CVat       string `xml:"c_vat,attr"`       // Partner VAT ID (without country prefix)
	KPlnEU     string `xml:"k_pln_eu,attr"`    // Type of supply: 0=goods, 1=transfer, 2=triangular, 3=services
	PlnHodnota int64  `xml:"pln_hodnota,attr"` // Total value in whole CZK
}
