package vatxml

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

func TestVATReturnGenerator_Generate(t *testing.T) {
	gen := &VATReturnGenerator{}

	vr := &domain.VATReturn{
		ID: 1,
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 3,
		},
		FilingType:        domain.FilingTypeRegular,
		OutputVATBase21:   1000000, // 10000.00 CZK
		OutputVATAmount21: 210000,  // 2100.00 CZK
		OutputVATBase12:   500000,  // 5000.00 CZK
		OutputVATAmount12: 60000,   // 600.00 CZK
		InputVATBase21:    300000,  // 3000.00 CZK
		InputVATAmount21:  63000,   // 630.00 CZK
		InputVATBase12:    100000,  // 1000.00 CZK
		InputVATAmount12:  12000,   // 120.00 CZK
		TotalOutputVAT:    270000,  // 2700.00 CZK
		TotalInputVAT:     75000,   // 750.00 CZK
		NetVAT:            195000,  // 1950.00 CZK
		Status:            "draft",
	}

	xmlData, err := gen.Generate(vr, "CZ12345678")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	xmlStr := string(xmlData)

	// Verify XML declaration.
	if !strings.HasPrefix(xmlStr, "<?xml version=") {
		t.Error("expected XML declaration at start")
	}

	// Verify it can be parsed back.
	var doc DPHPisemnost
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		t.Fatalf("failed to unmarshal generated XML: %v", err)
	}

	if doc.DPHDAP3 == nil {
		t.Fatal("expected DPHDAP3 element")
	}

	// Verify header.
	if doc.DPHDAP3.VetaD.DTyp != "R" {
		t.Errorf("VetaD.DTyp = %q, want %q", doc.DPHDAP3.VetaD.DTyp, "R")
	}
	if doc.DPHDAP3.VetaD.Rok != 2025 {
		t.Errorf("VetaD.Rok = %d, want 2025", doc.DPHDAP3.VetaD.Rok)
	}
	if doc.DPHDAP3.VetaD.Mesic != 3 {
		t.Errorf("VetaD.Mesic = %d, want 3", doc.DPHDAP3.VetaD.Mesic)
	}

	// Verify taxpayer.
	if doc.DPHDAP3.VetaP.DIC != "CZ12345678" {
		t.Errorf("VetaP.DIC = %q, want %q", doc.DPHDAP3.VetaP.DIC, "CZ12345678")
	}

	// Verify output VAT at 21% (amounts in whole CZK).
	if doc.DPHDAP3.Veta1 == nil {
		t.Fatal("expected Veta1 element")
	}
	if doc.DPHDAP3.Veta1.Obrat21 != 10000 {
		t.Errorf("Veta1.Obrat21 = %d, want 10000", doc.DPHDAP3.Veta1.Obrat21)
	}
	if doc.DPHDAP3.Veta1.Dan21 != 2100 {
		t.Errorf("Veta1.Dan21 = %d, want 2100", doc.DPHDAP3.Veta1.Dan21)
	}

	// Verify output VAT at 12%.
	if doc.DPHDAP3.Veta2 == nil {
		t.Fatal("expected Veta2 element")
	}
	if doc.DPHDAP3.Veta2.Obrat12 != 5000 {
		t.Errorf("Veta2.Obrat12 = %d, want 5000", doc.DPHDAP3.Veta2.Obrat12)
	}

	// Verify input VAT at 21%.
	if doc.DPHDAP3.Veta4 == nil {
		t.Fatal("expected Veta4 element")
	}
	if doc.DPHDAP3.Veta4.ZdPlnOdp21 != 3000 {
		t.Errorf("Veta4.ZdPlnOdp21 = %d, want 3000", doc.DPHDAP3.Veta4.ZdPlnOdp21)
	}
	if doc.DPHDAP3.Veta4.OdpTuz21Nar != 630 {
		t.Errorf("Veta4.OdpTuz21Nar = %d, want 630", doc.DPHDAP3.Veta4.OdpTuz21Nar)
	}

	// Verify summary.
	if doc.DPHDAP3.Veta6 == nil {
		t.Fatal("expected Veta6 element")
	}
	if doc.DPHDAP3.Veta6.DanDalOdp != 1950 {
		t.Errorf("Veta6.DanDalOdp = %d, want 1950", doc.DPHDAP3.Veta6.DanDalOdp)
	}
}

func TestVATReturnGenerator_Generate_NilReturn(t *testing.T) {
	gen := &VATReturnGenerator{}
	_, err := gen.Generate(nil, "CZ12345678")
	if err == nil {
		t.Error("expected error for nil vat return")
	}
}

func TestVATReturnGenerator_Generate_EmptyDIC(t *testing.T) {
	gen := &VATReturnGenerator{}
	vr := &domain.VATReturn{
		Period: domain.TaxPeriod{Year: 2025, Month: 1},
	}
	_, err := gen.Generate(vr, "")
	if err == nil {
		t.Error("expected error for empty DIC")
	}
}

func TestVATReturnGenerator_Generate_CorrectiveType(t *testing.T) {
	gen := &VATReturnGenerator{}
	vr := &domain.VATReturn{
		Period:     domain.TaxPeriod{Year: 2025, Month: 6},
		FilingType: domain.FilingTypeCorrective,
	}

	xmlData, err := gen.Generate(vr, "CZ12345678")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	var doc DPHPisemnost
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if doc.DPHDAP3 == nil {
		t.Fatal("expected DPHDAP3 element")
	}
	if doc.DPHDAP3.VetaD.DTyp != "N" {
		t.Errorf("VetaD.DTyp = %q, want %q for corrective", doc.DPHDAP3.VetaD.DTyp, "N")
	}
}

func TestVATReturnGenerator_Generate_SupplementaryType(t *testing.T) {
	gen := &VATReturnGenerator{}
	vr := &domain.VATReturn{
		Period:     domain.TaxPeriod{Year: 2025, Month: 6},
		FilingType: domain.FilingTypeSupplementary,
	}

	xmlData, err := gen.Generate(vr, "CZ12345678")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	var doc DPHPisemnost
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if doc.DPHDAP3 == nil {
		t.Fatal("expected DPHDAP3 element")
	}
	if doc.DPHDAP3.VetaD.DTyp != "O" {
		t.Errorf("VetaD.DTyp = %q, want %q for supplementary", doc.DPHDAP3.VetaD.DTyp, "O")
	}
}

func TestVATReturnGenerator_Generate_ZeroAmounts(t *testing.T) {
	gen := &VATReturnGenerator{}
	vr := &domain.VATReturn{
		Period:     domain.TaxPeriod{Year: 2025, Quarter: 1},
		FilingType: domain.FilingTypeRegular,
	}

	xmlData, err := gen.Generate(vr, "CZ12345678")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	var doc DPHPisemnost
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// With zero amounts, optional sections should be nil.
	if doc.DPHDAP3.Veta1 != nil {
		t.Error("expected Veta1 to be nil for zero output VAT 21%")
	}
	if doc.DPHDAP3.Veta2 != nil {
		t.Error("expected Veta2 to be nil for zero output VAT 12%")
	}
}
