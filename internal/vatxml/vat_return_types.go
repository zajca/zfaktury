package vatxml

import "encoding/xml"

// EPO VAT return XML types for Czech tax authority (Financni sprava).
// Based on the EPO submission format for "Priznani k dani z pridane hodnoty".

// DPHPisemnost is the root element of the EPO VAT return XML document.
type DPHPisemnost struct {
	XMLName xml.Name `xml:"Pisemnost"`
	Xmlns   string   `xml:"xmlns,attr,omitempty"`
	DPHDAP3 *DPHDAP3 `xml:"DPHDAP3,omitempty"`
}

// DPHDAP3 represents the VAT return form (Priznani k DPH).
type DPHDAP3 struct {
	VetaD DPHVetaD `xml:"VetaD"`
	VetaP DPHVetaP `xml:"VetaP"`
	Veta1 *Veta1   `xml:"Veta1,omitempty"`
	Veta2 *Veta2   `xml:"Veta2,omitempty"`
	Veta3 *Veta3   `xml:"Veta3,omitempty"`
	Veta4 *Veta4   `xml:"Veta4,omitempty"`
	Veta5 *Veta5   `xml:"Veta5,omitempty"`
	Veta6 *Veta6   `xml:"Veta6,omitempty"`
}

// DPHVetaD contains filing metadata for VAT return.
type DPHVetaD struct {
	DTyp    string `xml:"d_typ,attr"`
	Rok     int    `xml:"rok,attr"`
	Mesic   int    `xml:"mesic,attr"`
	Ctvrt   int    `xml:"ctvrt,attr"`
	DPocetL int    `xml:"d_pocetl,attr"`
	DPocetP int    `xml:"d_pocetp,attr"`
}

// DPHVetaP contains taxpayer identification for VAT return.
type DPHVetaP struct {
	Zast int    `xml:"zast,attr"`
	DIC  string `xml:"dic,attr"`
	Typ  string `xml:"typ,attr"`
}

// Veta1 contains output VAT at standard rate (21%).
type Veta1 struct {
	Obrat21 int64 `xml:"obrat21,attr"`
	Dan21   int64 `xml:"dan21,attr"`
}

// Veta2 contains output VAT at reduced rate (12%).
type Veta2 struct {
	Obrat12 int64 `xml:"obrat12,attr"`
	Dan12   int64 `xml:"dan12,attr"`
}

// Veta3 contains total output VAT summary.
type Veta3 struct {
	DanOdpOdpSazba int64 `xml:"dan_odp_odp_sazba,attr"`
}

// Veta4 contains input VAT at standard rate (21%).
type Veta4 struct {
	ZdPlnOdp21  int64 `xml:"zd_pln_odp21,attr"`
	OdpTuz21Nar int64 `xml:"odp_tuz21_nar,attr"`
}

// Veta5 contains input VAT at reduced rate (12%).
type Veta5 struct {
	ZdPlnOdp12  int64 `xml:"zd_pln_odp12,attr"`
	OdpTuz12Nar int64 `xml:"odp_tuz12_nar,attr"`
}

// Veta6 contains the final VAT calculation summary.
type Veta6 struct {
	DanOdpOdpSazba int64 `xml:"dan_odp_odp_sazba,attr"`
	DanDalOdp      int64 `xml:"dan_dal_odp,attr"`
}
