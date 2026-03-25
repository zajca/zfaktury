package vatxml

import (
	"encoding/xml"
	"time"
)

// Official DPHDP3 EPO format for Czech VAT return (Priznani k DPH).
// Based on XSD: adisspr.mfcr.cz/adis/jepo/schema/dphdp3_epo2.xsd

// DPHPisemnost is the root element of the EPO VAT return XML document.
type DPHPisemnost struct {
	XMLName xml.Name `xml:"Pisemnost"`
	NazevSW string   `xml:"nazevSW,attr,omitempty"`
	DPHDP3  *DPHDP3  `xml:"DPHDP3,omitempty"`
}

// DPHDP3 represents the VAT return form (Priznani k DPH).
type DPHDP3 struct {
	VerzePis string `xml:"verzePis,attr"`
	VetaD    VetaD  `xml:"VetaD"`
	VetaP    VetaP  `xml:"VetaP"`
	Veta1    *Veta1 `xml:"Veta1,omitempty"`
	Veta2    *Veta2 `xml:"Veta2,omitempty"`
	Veta4    *Veta4 `xml:"Veta4,omitempty"`
	Veta6    *Veta6 `xml:"Veta6,omitempty"`
}

// VetaD contains filing metadata for VAT return.
type VetaD struct {
	Dokument    string `xml:"dokument,attr"`
	KUladis     string `xml:"k_uladis,attr"`
	DapdphForma string `xml:"dapdph_forma,attr"`
	TypPlatce   string `xml:"typ_platce,attr"`
	Trans       string `xml:"trans,attr"`
	COkec       string `xml:"c_okec,attr,omitempty"`
	DPoddp      string `xml:"d_poddp,attr"`
	Rok         int    `xml:"rok,attr"`
	Mesic       int    `xml:"mesic,attr,omitempty"`
	Ctvrt       int    `xml:"ctvrt,attr,omitempty"`
}

// VetaP contains taxpayer identification for VAT return.
type VetaP struct {
	CPracufo string `xml:"c_pracufo,attr,omitempty"`
	CUfo     string `xml:"c_ufo,attr,omitempty"`
	DIC      string `xml:"dic,attr"`
	Email    string `xml:"email,attr,omitempty"`
	CTelef   string `xml:"c_telef,attr,omitempty"`
	Ulice    string `xml:"ulice,attr,omitempty"`
	NazObce  string `xml:"naz_obce,attr,omitempty"`
	PSC      string `xml:"psc,attr,omitempty"`
	Stat     string `xml:"stat,attr,omitempty"`
	CPop     string `xml:"c_pop,attr,omitempty"`
	COrient  string `xml:"c_orient,attr"`
	Jmeno    string `xml:"jmeno,attr,omitempty"`
	Prijmeni string `xml:"prijmeni,attr,omitempty"`
	Titul    string `xml:"titul,attr"`
	TypDS    string `xml:"typ_ds,attr"`
}

// Veta1 contains output VAT (dan na vystupu) - section I.
type Veta1 struct {
	Obrat23    XMLFloat `xml:"obrat23,attr"`
	Dan23      XMLFloat `xml:"dan23,attr"`
	Obrat5     XMLFloat `xml:"obrat5,attr"`
	Dan5       XMLFloat `xml:"dan5,attr"`
	PSl23E     XMLFloat `xml:"p_sl23_e,attr"`
	DanPsl23E  XMLFloat `xml:"dan_psl23_e,attr"`
	PSl5E      XMLFloat `xml:"p_sl5_e,attr"`
	DanPsl5E   XMLFloat `xml:"dan_psl5_e,attr"`
	PSl23Z     XMLFloat `xml:"p_sl23_z,attr"`
	DanPsl23Z  XMLFloat `xml:"dan_psl23_z,attr"`
	PSl5Z      XMLFloat `xml:"p_sl5_z,attr"`
	DanPsl5Z   XMLFloat `xml:"dan_psl5_z,attr"`
	RezPren23  XMLFloat `xml:"rez_pren23,attr"`
	DanRpren23 XMLFloat `xml:"dan_rpren23,attr"`
	RezPren5   XMLFloat `xml:"rez_pren5,attr"`
	DanRpren5  XMLFloat `xml:"dan_rpren5,attr"`
}

// Veta2 contains EU acquisitions and services - section II.
type Veta2 struct {
	DodZb      XMLFloat `xml:"dod_zb,attr"`
	PlnSluzby  XMLFloat `xml:"pln_sluzby,attr"`
	PlnRezPren XMLFloat `xml:"pln_rez_pren,attr"`
	PlnZaslani XMLFloat `xml:"pln_zaslani,attr"`
	PlnOst     XMLFloat `xml:"pln_ost,attr"`
}

// Veta4 contains input VAT (dan na vstupu) - section IV.
type Veta4 struct {
	Pln23       XMLFloat `xml:"pln23,attr"`
	OdpTuz23Nar XMLFloat `xml:"odp_tuz23_nar,attr"`
	Pln5        XMLFloat `xml:"pln5,attr"`
	OdpTuz5Nar  XMLFloat `xml:"odp_tuz5_nar,attr"`
	NarZdp23    XMLFloat `xml:"nar_zdp23,attr"`
	OdZdp23     XMLFloat `xml:"od_zdp23,attr"`
	NarZdp5     XMLFloat `xml:"nar_zdp5,attr"`
	OdZdp5      XMLFloat `xml:"od_zdp5,attr"`
	OdpSumKr    string   `xml:"odp_sum_kr,attr"`
	OdpSumNar   XMLFloat `xml:"odp_sum_nar,attr"`
}

// Veta6 contains final VAT calculation summary - section VI.
type Veta6 struct {
	Dano      string   `xml:"dano,attr"`
	DanoNo    string   `xml:"dano_no,attr"`
	DanoDa    XMLFloat `xml:"dano_da,attr"`
	DanZocelk XMLFloat `xml:"dan_zocelk,attr"`
	OdpZocelk XMLFloat `xml:"odp_zocelk,attr"`
}

// TaxpayerInfo contains taxpayer details needed for DPHDP3 XML generation.
type TaxpayerInfo struct {
	DIC            string // without CZ prefix
	FirstName      string
	LastName       string
	Street         string
	HouseNum       string
	ZIP            string
	City           string
	Phone          string
	Email          string
	UFOCode        string    // c_ufo (3-digit)
	PracUFO        string    // c_pracufo (4-digit)
	OKEC           string    // NACE code
	SubmissionDate time.Time // date of submission; zero value uses time.Now()
}
