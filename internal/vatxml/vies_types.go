package vatxml

import "encoding/xml"

// EPO XML types for Souhrnne hlaseni (VIES Recapitulative Statement).
// These types represent the Czech tax authority EPO format for VIES reporting.

// VIESPisemnost is the root XML element for a VIES summary submission.
type VIESPisemnost struct {
	XMLName xml.Name `xml:"Pisemnost"`
	Xmlns   string   `xml:"xmlns,attr"`
	DPHSHV  DPHSHV   `xml:"DPHSHV"`
}

// DPHSHV is the main container for a VIES summary (Souhrnne hlaseni k DPH).
type DPHSHV struct {
	VetaD VIESVetaD  `xml:"VetaD"`
	VetaP []VIESVetaP `xml:"VetaP,omitempty"`
	VetaR []VIESVetaR `xml:"VetaR,omitempty"`
}

// VIESVetaD contains header information about the filing.
type VIESVetaD struct {
	// Filing metadata.
	KDaph  string `xml:"k_daph,attr"`  // Filing type: B=regular, O=corrective, N=supplementary
	Rok    int    `xml:"rok,attr"`      // Year
	Ctvrt  int    `xml:"ctvrt,attr"`    // Quarter (1-4)
	DICOdb string `xml:"dic_odb,attr"`  // VAT ID of the filer (without CZ prefix)
}

// VIESVetaP contains a line for goods deliveries to EU partners.
// Not typically used for service providers, included for completeness.
type VIESVetaP struct {
	KStat   string `xml:"k_stat,attr"`     // Country code (2-letter)
	DICOdbe string `xml:"dic_odbe,attr"`   // Partner VAT ID (without country prefix)
	KPlneni string `xml:"k_plneni,attr"`   // Type of supply: 0=goods, 1=triangular, 3=services
	Obrat   int64  `xml:"obrat,attr"`      // Amount in whole CZK
}

// VIESVetaR contains a correction/amendment line.
type VIESVetaR struct {
	KStat   string `xml:"k_stat,attr"`     // Country code (2-letter)
	DICOdbe string `xml:"dic_odbe,attr"`   // Partner VAT ID (without country prefix)
	KPlneni string `xml:"k_plneni,attr"`   // Type of supply
	Obrat   int64  `xml:"obrat,attr"`      // Corrected amount in whole CZK
	RokOpr  int    `xml:"rok_opr,attr"`    // Original year being corrected
	CtvOpr  int    `xml:"ctv_opr,attr"`    // Original quarter being corrected
}
