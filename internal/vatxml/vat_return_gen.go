package vatxml

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// VATReturnGenerator generates EPO XML for Czech VAT returns in DPHDP3 format.
type VATReturnGenerator struct{}

// Generate produces DPHDP3 XML bytes from a VATReturn and taxpayer info.
func (g *VATReturnGenerator) Generate(vr *domain.VATReturn, info TaxpayerInfo) ([]byte, error) {
	if vr == nil {
		return nil, fmt.Errorf("vat return is nil: %w", domain.ErrInvalidInput)
	}
	if info.DIC == "" {
		return nil, fmt.Errorf("DIC is required for XML generation: %w", domain.ErrInvalidInput)
	}

	filingCode := DPHFilingTypeCode(vr.FilingType)

	// Compute rounded values.
	obrat23 := float64(ToWholeCZK(vr.OutputVATBase21))
	dan23 := float64(ToWholeCZK(vr.OutputVATAmount21))
	obrat5 := float64(ToWholeCZK(vr.OutputVATBase12))
	dan5 := float64(ToWholeCZK(vr.OutputVATAmount12))

	pln23 := float64(ToWholeCZK(vr.InputVATBase21))
	odpTuz23Nar := float64(ToWholeCZK(vr.InputVATAmount21))
	pln5 := float64(ToWholeCZK(vr.InputVATBase12))
	odpTuz5Nar := float64(ToWholeCZK(vr.InputVATAmount12))

	// Total deductions.
	odpSumNar := odpTuz23Nar + odpTuz5Nar

	// Total output tax.
	danZocelk := dan23 + dan5

	// Total input deductions.
	odpZocelk := odpSumNar

	// Net VAT.
	netVAT := danZocelk - odpZocelk

	// Determine trans: "A" if any amounts, "N" otherwise.
	trans := "N"
	if obrat23 != 0 || dan23 != 0 || obrat5 != 0 || dan5 != 0 ||
		pln23 != 0 || odpTuz23Nar != 0 || pln5 != 0 || odpTuz5Nar != 0 {
		trans = "A"
	}

	doc := &DPHPisemnost{
		NazevSW: "ZFaktury",
		DPHDP3: &DPHDP3{
			VerzePis: "01.02.16",
			VetaD: VetaD{
				Dokument:    "DP3",
				KUladis:     "DPH",
				DapdphForma: filingCode,
				TypPlatce:   "P",
				Trans:       trans,
				COkec:       info.OKEC,
				DPoddp:      time.Now().Format("02.01.2006"),
				Rok:         vr.Period.Year,
				Mesic:       vr.Period.Month,
				Ctvrt:       vr.Period.Quarter,
			},
			VetaP: VetaP{
				CPracufo: info.PracUFO,
				CUfo:     info.UFOCode,
				DIC:      info.DIC,
				Email:    info.Email,
				CTelef:   info.Phone,
				Ulice:    info.Street,
				NazObce:  info.City,
				PSC:      info.ZIP,
				Stat:     "CESKA REPUBLIKA",
				CPop:     info.HouseNum,
				Jmeno:    info.FirstName,
				Prijmeni: info.LastName,
				TypDS:    "F",
			},
			Veta1: &Veta1{
				Obrat23: obrat23,
				Dan23:   dan23,
				Obrat5:  obrat5,
				Dan5:    dan5,
			},
			Veta2: &Veta2{},
			Veta4: &Veta4{
				Pln23:       pln23,
				OdpTuz23Nar: odpTuz23Nar,
				Pln5:        pln5,
				OdpTuz5Nar:  odpTuz5Nar,
				OdpSumKr:    "0",
				OdpSumNar:   odpSumNar,
			},
			Veta6: &Veta6{
				Dano:      "0",
				DanZocelk: danZocelk,
				OdpZocelk: odpZocelk,
			},
		},
	}

	// Set dano_da or dano_no based on net VAT.
	if netVAT > 0 {
		doc.DPHDP3.Veta6.DanoDa = netVAT
		doc.DPHDP3.Veta6.DanoNo = "0"
	} else if netVAT < 0 {
		doc.DPHDP3.Veta6.DanoNo = fmt.Sprintf("%.1f", -netVAT)
		doc.DPHDP3.Veta6.DanoDa = 0
	} else {
		doc.DPHDP3.Veta6.DanoNo = "0"
		doc.DPHDP3.Veta6.DanoDa = 0
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
