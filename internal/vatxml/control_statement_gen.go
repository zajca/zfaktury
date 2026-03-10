package vatxml

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// ControlStatementGenerator builds XML for a VAT control statement submission.
type ControlStatementGenerator struct{}

// NewControlStatementGenerator creates a new ControlStatementGenerator.
func NewControlStatementGenerator() *ControlStatementGenerator {
	return &ControlStatementGenerator{}
}

// Generate builds the EPO XML for the given control statement, its lines, and the taxpayer DIC.
func (g *ControlStatementGenerator) Generate(cs *domain.VATControlStatement, lines []domain.VATControlStatementLine, dic string) ([]byte, error) {
	if cs == nil {
		return nil, fmt.Errorf("control statement is nil")
	}

	dicNum := strings.TrimPrefix(strings.ToUpper(dic), "CZ")
	if dicNum == "" {
		return nil, fmt.Errorf("DIC is empty")
	}

	filingCode := FilingTypeCode(cs.FilingType)

	doc := ControlStatementXML{
		Xmlns: "http://adis.mfcr.cz/rozhranni/",
		DPHKH: DPHKH1{
			VetaD: KHVetaD{
				DType:     filingCode,
				Rok:       cs.Period.Year,
				Mesic:     cs.Period.Month,
				DokDPHKH:  "KH",
				KHDPHForm: filingCode,
			},
			VetaP: KHVetaP{
				DIC:  dicNum,
				Type: "P",
			},
		},
	}

	for _, line := range lines {
		switch line.Section {
		case "A4":
			doc.DPHKH.A4 = append(doc.DPHKH.A4, buildVetaA4(line))
		case "A5":
			doc.DPHKH.A5 = append(doc.DPHKH.A5, buildVetaA5(line))
		case "B2":
			doc.DPHKH.B2 = append(doc.DPHKH.B2, buildVetaB2(line))
		case "B3":
			doc.DPHKH.B3 = append(doc.DPHKH.B3, buildVetaB3(line))
		}
	}

	output, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling control statement XML: %w", err)
	}

	header := []byte(xml.Header)
	return append(header, output...), nil
}

// formatDPPD converts a YYYY-MM-DD date string to DD.MM.YYYY for XML.
func formatDPPD(dppd string) string {
	t, err := time.Parse("2006-01-02", dppd)
	if err != nil {
		return dppd
	}
	return t.Format("02.01.2006")
}

// int64Ptr returns a pointer to an int64, or nil if zero.
func int64Ptr(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}

// buildVetaA4 creates a VetaA4 element from a control statement line.
func buildVetaA4(line domain.VATControlStatementLine) VetaA4 {
	v := VetaA4{
		CisloEv:     line.DocumentNumber,
		DPPD:        formatDPPD(line.DPPD),
		DicOdb:      strings.TrimPrefix(strings.ToUpper(line.PartnerDIC), "CZ"),
		KodRezimPln: "0",
	}
	base := ToWholeCZK(line.Base)
	vat := ToWholeCZK(line.VAT)
	switch line.VATRatePercent {
	case 21:
		v.Zaklad1 = int64Ptr(base)
		v.Dan1 = int64Ptr(vat)
	case 12:
		v.Zaklad2 = int64Ptr(base)
		v.Dan2 = int64Ptr(vat)
	}
	return v
}

// buildVetaA5 creates a VetaA5 element from a control statement line.
func buildVetaA5(line domain.VATControlStatementLine) VetaA5 {
	v := VetaA5{KodRezimPln: "0"}
	base := ToWholeCZK(line.Base)
	vat := ToWholeCZK(line.VAT)
	switch line.VATRatePercent {
	case 21:
		v.Zaklad1 = int64Ptr(base)
		v.Dan1 = int64Ptr(vat)
	case 12:
		v.Zaklad2 = int64Ptr(base)
		v.Dan2 = int64Ptr(vat)
	}
	return v
}

// buildVetaB2 creates a VetaB2 element from a control statement line.
func buildVetaB2(line domain.VATControlStatementLine) VetaB2 {
	v := VetaB2{
		CisloEv:     line.DocumentNumber,
		DPPD:        formatDPPD(line.DPPD),
		DicDod:      strings.TrimPrefix(strings.ToUpper(line.PartnerDIC), "CZ"),
		KodRezimPln: "0",
	}
	base := ToWholeCZK(line.Base)
	vat := ToWholeCZK(line.VAT)
	switch line.VATRatePercent {
	case 21:
		v.Zaklad1 = int64Ptr(base)
		v.Dan1 = int64Ptr(vat)
	case 12:
		v.Zaklad2 = int64Ptr(base)
		v.Dan2 = int64Ptr(vat)
	}
	return v
}

// buildVetaB3 creates a VetaB3 element from a control statement line.
func buildVetaB3(line domain.VATControlStatementLine) VetaB3 {
	v := VetaB3{KodRezimPln: "0"}
	base := ToWholeCZK(line.Base)
	vat := ToWholeCZK(line.VAT)
	switch line.VATRatePercent {
	case 21:
		v.Zaklad1 = int64Ptr(base)
		v.Dan1 = int64Ptr(vat)
	case 12:
		v.Zaklad2 = int64Ptr(base)
		v.Dan2 = int64Ptr(vat)
	}
	return v
}
