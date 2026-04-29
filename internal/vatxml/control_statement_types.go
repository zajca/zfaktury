package vatxml

import "encoding/xml"

// EPO XML types for Kontrolni hlaseni (VAT Control Statement).

// KHVetaD contains metadata about the control statement filing.
type KHVetaD struct {
	XMLName   xml.Name `xml:"VetaD"`
	Dokument  string   `xml:"dokument,attr"`
	KUladis   string   `xml:"k_uladis,attr"`
	Rok       int      `xml:"rok,attr"`
	Mesic     int      `xml:"mesic,attr"`
	KHDPHForm string   `xml:"khdph_forma,attr"`
}

// KHVetaP contains taxpayer identification for control statement.
type KHVetaP struct {
	XMLName xml.Name `xml:"VetaP"`
	DIC     string   `xml:"dic,attr"`
	Type    string   `xml:"typ_ds,attr"`
}

// VetaA4 represents an individual output transaction above 10,000 CZK.
type VetaA4 struct {
	XMLName    xml.Name `xml:"VetaA4"`
	CisloEv    string   `xml:"c_evid_dd,attr"`
	DPPD       string   `xml:"dppd,attr"`
	DicOdb     string   `xml:"dic_odb,attr"`
	KodRezimPl string   `xml:"kod_rezim_pl,attr"`
	Zdph44     string   `xml:"zdph_44,attr"`
	Zaklad1    *int64   `xml:"zakl_dane1,attr"`
	Dan1       *int64   `xml:"dan1,attr"`
	Zaklad2    *int64   `xml:"zakl_dane2,attr"`
	Dan2       *int64   `xml:"dan2,attr"`
}

// VetaA5 represents aggregated output transactions at or below 10,000 CZK.
type VetaA5 struct {
	XMLName xml.Name `xml:"VetaA5"`
	Zaklad1 *int64   `xml:"zakl_dane1,attr"`
	Dan1    *int64   `xml:"dan1,attr"`
	Zaklad2 *int64   `xml:"zakl_dane2,attr"`
	Dan2    *int64   `xml:"dan2,attr"`
}

// VetaB2 represents an individual input transaction above 10,000 CZK.
type VetaB2 struct {
	XMLName xml.Name `xml:"VetaB2"`
	CisloEv string   `xml:"c_evid_dd,attr"`
	DPPD    string   `xml:"dppd,attr"`
	DicDod  string   `xml:"dic_dod,attr"`
	Zaklad1 *int64   `xml:"zakl_dane1,attr"`
	Dan1    *int64   `xml:"dan1,attr"`
	Zaklad2 *int64   `xml:"zakl_dane2,attr"`
	Dan2    *int64   `xml:"dan2,attr"`
	Pomer   string   `xml:"pomer,attr"`
	Zdph44  string   `xml:"zdph_44,attr"`
}

// VetaB3 represents aggregated input transactions at or below 10,000 CZK.
type VetaB3 struct {
	XMLName xml.Name `xml:"VetaB3"`
	Zaklad1 *int64   `xml:"zakl_dane1,attr"`
	Dan1    *int64   `xml:"dan1,attr"`
	Zaklad2 *int64   `xml:"zakl_dane2,attr"`
	Dan2    *int64   `xml:"dan2,attr"`
}

// ControlStatementXML is the serializable root structure for control statement XML.
type ControlStatementXML struct {
	XMLName xml.Name `xml:"Pisemnost"`
	Xmlns   string   `xml:"xmlns,attr"`
	DPHKH   DPHKH1   `xml:"DPHKH1"`
}

// DPHKH1 is the typed container for control statement marshalling.
type DPHKH1 struct {
	VetaD KHVetaD  `xml:"VetaD"`
	VetaP KHVetaP  `xml:"VetaP"`
	A4    []VetaA4 `xml:"VetaA4,omitempty"`
	A5    []VetaA5 `xml:"VetaA5,omitempty"`
	B2    []VetaB2 `xml:"VetaB2,omitempty"`
	B3    []VetaB3 `xml:"VetaB3,omitempty"`
}
