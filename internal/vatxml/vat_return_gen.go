package vatxml

import (
	"encoding/xml"
	"fmt"

	"github.com/zajca/zfaktury/internal/domain"
)

// VATReturnGenerator generates EPO XML for Czech VAT returns.
type VATReturnGenerator struct{}

// Generate produces XML bytes from a VATReturn and the taxpayer's DIC.
func (g *VATReturnGenerator) Generate(vr *domain.VATReturn, dic string) ([]byte, error) {
	if vr == nil {
		return nil, fmt.Errorf("vat return is nil: %w", domain.ErrInvalidInput)
	}
	if dic == "" {
		return nil, fmt.Errorf("DIC is required for XML generation: %w", domain.ErrInvalidInput)
	}

	dTyp := FilingTypeCode(vr.FilingType)

	doc := &DPHPisemnost{
		DPHDAP3: &DPHDAP3{
			VetaD: DPHVetaD{
				DTyp:    dTyp,
				Rok:     vr.Period.Year,
				Mesic:   vr.Period.Month,
				Ctvrt:   vr.Period.Quarter,
				DPocetL: 0,
				DPocetP: 0,
			},
			VetaP: DPHVetaP{
				Zast: 0,
				DIC:  dic,
				Typ:  "F", // Natural person (OSVC).
			},
		},
	}

	// Output VAT at 21%.
	if vr.OutputVATBase21 != 0 || vr.OutputVATAmount21 != 0 {
		doc.DPHDAP3.Veta1 = &Veta1{
			Obrat21: ToWholeCZK(vr.OutputVATBase21),
			Dan21:   ToWholeCZK(vr.OutputVATAmount21),
		}
	}

	// Output VAT at 12%.
	if vr.OutputVATBase12 != 0 || vr.OutputVATAmount12 != 0 {
		doc.DPHDAP3.Veta2 = &Veta2{
			Obrat12: ToWholeCZK(vr.OutputVATBase12),
			Dan12:   ToWholeCZK(vr.OutputVATAmount12),
		}
	}

	// Total output VAT.
	totalOutputVAT := ToWholeCZK(vr.TotalOutputVAT)
	if totalOutputVAT != 0 {
		doc.DPHDAP3.Veta3 = &Veta3{
			DanOdpOdpSazba: totalOutputVAT,
		}
	}

	// Input VAT at 21%.
	if vr.InputVATBase21 != 0 || vr.InputVATAmount21 != 0 {
		doc.DPHDAP3.Veta4 = &Veta4{
			ZdPlnOdp21:  ToWholeCZK(vr.InputVATBase21),
			OdpTuz21Nar: ToWholeCZK(vr.InputVATAmount21),
		}
	}

	// Input VAT at 12%.
	if vr.InputVATBase12 != 0 || vr.InputVATAmount12 != 0 {
		doc.DPHDAP3.Veta5 = &Veta5{
			ZdPlnOdp12:  ToWholeCZK(vr.InputVATBase12),
			OdpTuz12Nar: ToWholeCZK(vr.InputVATAmount12),
		}
	}

	// Summary: total input VAT credit and net VAT.
	totalInputVAT := ToWholeCZK(vr.TotalInputVAT)
	netVAT := ToWholeCZK(vr.NetVAT)
	if totalInputVAT != 0 || netVAT != 0 {
		doc.DPHDAP3.Veta6 = &Veta6{
			DanOdpOdpSazba: totalInputVAT,
			DanDalOdp:      netVAT,
		}
	}

	output, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling VAT return XML: %w", err)
	}

	header := []byte(xml.Header)
	result := make([]byte, 0, len(header)+len(output))
	result = append(result, header...)
	result = append(result, output...)

	return result, nil
}
