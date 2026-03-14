package vatxml

import (
	"strings"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

func TestControlStatementGenerator_Generate_Basic(t *testing.T) {
	gen := NewControlStatementGenerator()

	cs := &domain.VATControlStatement{
		ID: 1,
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 3,
		},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusReady,
	}

	invID := int64(10)
	lines := []domain.VATControlStatementLine{
		{
			ID:                 1,
			ControlStatementID: 1,
			Section:            "A4",
			PartnerDIC:         "CZ12345678",
			DocumentNumber:     "FV20250001",
			DPPD:               "2025-03-15",
			Base:               domain.NewAmount(15000, 0), // 15000 CZK
			VAT:                domain.NewAmount(3150, 0),  // 3150 CZK (21%)
			VATRatePercent:     21,
			InvoiceID:          &invID,
		},
		{
			ID:                 2,
			ControlStatementID: 1,
			Section:            "A5",
			Base:               domain.NewAmount(5000, 0),
			VAT:                domain.NewAmount(1050, 0),
			VATRatePercent:     21,
		},
	}

	xmlData, err := gen.Generate(cs, lines, "CZ87654321")
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	xmlStr := string(xmlData)

	// Verify XML header.
	if !strings.HasPrefix(xmlStr, "<?xml") {
		t.Error("XML should start with XML declaration")
	}

	// Verify root element.
	if !strings.Contains(xmlStr, "<Pisemnost") {
		t.Error("XML should contain <Pisemnost> element")
	}

	// Verify VetaD attributes.
	if !strings.Contains(xmlStr, `rok="2025"`) {
		t.Error("XML should contain rok=2025")
	}
	if !strings.Contains(xmlStr, `mesic="3"`) {
		t.Error("XML should contain mesic=3")
	}
	if !strings.Contains(xmlStr, `d_typ="R"`) {
		t.Error("XML should contain d_typ=R for regular filing")
	}

	// Verify VetaP.
	if !strings.Contains(xmlStr, `dic="87654321"`) {
		t.Error("XML should contain DIC without CZ prefix")
	}

	// Verify A4 element.
	if !strings.Contains(xmlStr, "<VetaA4") {
		t.Error("XML should contain VetaA4 element")
	}
	if !strings.Contains(xmlStr, `c_evid_dd="FV20250001"`) {
		t.Error("XML should contain document number")
	}
	if !strings.Contains(xmlStr, `dppd="15.03.2025"`) {
		t.Error("XML should contain DPPD in DD.MM.YYYY format")
	}
	if !strings.Contains(xmlStr, `dic_odb="12345678"`) {
		t.Error("XML should contain partner DIC without CZ prefix")
	}
	// 15000 CZK = 1500000 halere, toWholeCZK = 15000.
	if !strings.Contains(xmlStr, `zakl_dane1="15000"`) {
		t.Error("XML should contain base amount in whole CZK for rate 21%")
	}
	if !strings.Contains(xmlStr, `dan1="3150"`) {
		t.Error("XML should contain VAT amount in whole CZK for rate 21%")
	}

	// Verify A5 element.
	if !strings.Contains(xmlStr, "<VetaA5") {
		t.Error("XML should contain VetaA5 element")
	}
}

func TestControlStatementGenerator_Generate_CorrectiveFiling(t *testing.T) {
	gen := NewControlStatementGenerator()

	cs := &domain.VATControlStatement{
		ID: 2,
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 1,
		},
		FilingType: domain.FilingTypeCorrective,
		Status:     domain.FilingStatusReady,
	}

	xmlData, err := gen.Generate(cs, nil, "CZ12345678")
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	xmlStr := string(xmlData)
	if !strings.Contains(xmlStr, `d_typ="N"`) {
		t.Error("XML should contain d_typ=N for corrective filing")
	}
}

func TestControlStatementGenerator_Generate_B2B3Lines(t *testing.T) {
	gen := NewControlStatementGenerator()

	cs := &domain.VATControlStatement{
		ID: 3,
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 6,
		},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusReady,
	}

	expID := int64(20)
	lines := []domain.VATControlStatementLine{
		{
			Section:            "B2",
			ControlStatementID: 3,
			PartnerDIC:         "CZ99887766",
			DocumentNumber:     "VF2025001",
			DPPD:               "2025-06-10",
			Base:               domain.NewAmount(20000, 0),
			VAT:                domain.NewAmount(2400, 0),
			VATRatePercent:     12,
			ExpenseID:          &expID,
		},
		{
			Section:            "B3",
			ControlStatementID: 3,
			Base:               domain.NewAmount(3000, 0),
			VAT:                domain.NewAmount(360, 0),
			VATRatePercent:     12,
		},
	}

	xmlData, err := gen.Generate(cs, lines, "CZ12345678")
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	xmlStr := string(xmlData)

	// Verify B2.
	if !strings.Contains(xmlStr, "<VetaB2") {
		t.Error("XML should contain VetaB2 element")
	}
	if !strings.Contains(xmlStr, `dic_dod="99887766"`) {
		t.Error("XML should contain vendor DIC without CZ prefix")
	}
	if !strings.Contains(xmlStr, `zakl_dane2="20000"`) {
		t.Error("XML should contain base at 12% rate")
	}
	if !strings.Contains(xmlStr, `dan2="2400"`) {
		t.Error("XML should contain VAT at 12% rate")
	}

	// Verify B3.
	if !strings.Contains(xmlStr, "<VetaB3") {
		t.Error("XML should contain VetaB3 element")
	}
}

func TestControlStatementGenerator_Generate_EmptyDIC(t *testing.T) {
	gen := NewControlStatementGenerator()

	cs := &domain.VATControlStatement{
		ID: 1,
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 1,
		},
	}

	_, err := gen.Generate(cs, nil, "")
	if err == nil {
		t.Error("Generate() should return error for empty DIC")
	}
}

func TestControlStatementGenerator_Generate_NilStatement(t *testing.T) {
	gen := NewControlStatementGenerator()

	_, err := gen.Generate(nil, nil, "CZ12345678")
	if err == nil {
		t.Error("Generate() should return error for nil statement")
	}
}

func TestFormatDPPD(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2025-03-15", "15.03.2025"},
		{"2025-12-01", "01.12.2025"},
		{"invalid", "invalid"},
	}

	for _, tt := range tests {
		result := formatDPPD(tt.input)
		if result != tt.expected {
			t.Errorf("formatDPPD(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestToWholeCZK(t *testing.T) {
	tests := []struct {
		input    domain.Amount
		expected int64
	}{
		{domain.NewAmount(100, 0), 100},
		{domain.NewAmount(100, 50), 101}, // rounds up (banker's rounding: .50 -> up)
		{domain.NewAmount(0, 99), 1},     // rounds up
		{domain.Amount(-15000), -150},
	}

	for _, tt := range tests {
		result := ToWholeCZK(tt.input)
		if result != tt.expected {
			t.Errorf("ToWholeCZK(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestSharedFilingTypeCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{domain.FilingTypeRegular, "R"},
		{domain.FilingTypeCorrective, "N"},
		{domain.FilingTypeSupplementary, "O"},
		{"unknown", "R"},
	}

	for _, tt := range tests {
		result := FilingTypeCode(tt.input)
		if result != tt.expected {
			t.Errorf("FilingTypeCode(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
