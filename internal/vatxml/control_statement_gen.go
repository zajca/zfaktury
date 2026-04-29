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
				Dokument:  "KH1",
				KUladis:   "DPH",
				Rok:       cs.Period.Year,
				Mesic:     cs.Period.Month,
				KHDPHForm: filingCode,
			},
			VetaP: KHVetaP{
				DIC:  dicNum,
				Type: "P",
			},
		},
	}

	var a5Base21, a5VAT21, a5Base12, a5VAT12 int64
	var b3Base21, b3VAT21, b3Base12, b3VAT12 int64
	var hasA5, hasB3 bool
	var a4Base21, a4VAT21, a4Base12, a4VAT12 int64
	var b2Base21, b2VAT21, b2Base12, b2VAT12 int64

	for _, line := range lines {
		base := ToWholeCZK(line.Base)
		vat := ToWholeCZK(line.VAT)
		switch line.Section {
		case "A4":
			doc.DPHKH.A4 = append(doc.DPHKH.A4, buildVetaA4(line))
			switch line.VATRatePercent {
			case 21:
				a4Base21 += base
				a4VAT21 += vat
			case 12:
				a4Base12 += base
				a4VAT12 += vat
			}
		case "A5":
			hasA5 = true
			switch line.VATRatePercent {
			case 21:
				a5Base21 += base
				a5VAT21 += vat
			case 12:
				a5Base12 += base
				a5VAT12 += vat
			}
		case "B2":
			doc.DPHKH.B2 = append(doc.DPHKH.B2, buildVetaB2(line))
			switch line.VATRatePercent {
			case 21:
				b2Base21 += base
				b2VAT21 += vat
			case 12:
				b2Base12 += base
				b2VAT12 += vat
			}
		case "B3":
			hasB3 = true
			switch line.VATRatePercent {
			case 21:
				b3Base21 += base
				b3VAT21 += vat
			case 12:
				b3Base12 += base
				b3VAT12 += vat
			}
		}
	}

	if hasA5 {
		doc.DPHKH.A5 = &VetaA5{
			Zaklad1: int64Ptr(a5Base21),
			Dan1:    int64Ptr(a5VAT21),
			Zaklad2: int64Ptr(a5Base12),
			Dan2:    int64Ptr(a5VAT12),
		}
	}
	if hasB3 {
		doc.DPHKH.B3 = &VetaB3{
			Zaklad1: int64Ptr(b3Base21),
			Dan1:    int64Ptr(b3VAT21),
			Zaklad2: int64Ptr(b3Base12),
			Dan2:    int64Ptr(b3VAT12),
		}
	}

	obrat23 := a4Base21 + a5Base21
	obrat5 := a4Base12 + a5Base12
	pln23 := b2Base21 + b3Base21
	pln5 := b2Base12 + b3Base12
	if obrat23 != 0 || obrat5 != 0 || pln23 != 0 || pln5 != 0 {
		doc.DPHKH.C = &VetaC{
			Obrat23: int64Ptr(obrat23),
			Obrat5:  int64Ptr(obrat5),
			Pln23:   int64Ptr(pln23),
			Pln5:    int64Ptr(pln5),
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
		CisloEv:    line.DocumentNumber,
		DPPD:       formatDPPD(line.DPPD),
		DicOdb:     strings.TrimPrefix(strings.ToUpper(line.PartnerDIC), "CZ"),
		KodRezimPl: "0",
		Zdph44:     "N",
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

// buildVetaB2 creates a VetaB2 element from a control statement line.
func buildVetaB2(line domain.VATControlStatementLine) VetaB2 {
	v := VetaB2{
		CisloEv: line.DocumentNumber,
		DPPD:    formatDPPD(line.DPPD),
		DicDod:  strings.TrimPrefix(strings.ToUpper(line.PartnerDIC), "CZ"),
		Pomer:   "N",
		Zdph44:  "N",
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
