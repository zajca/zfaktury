package vatxml

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
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
	if !strings.Contains(xmlStr, `khdph_forma="B"`) {
		t.Error("XML should contain khdph_forma=B for regular filing")
	}
	if !strings.Contains(xmlStr, `dokument="KH1"`) {
		t.Error("XML should contain dokument=KH1")
	}
	if !strings.Contains(xmlStr, `k_uladis="DPH"`) {
		t.Error("XML should contain k_uladis=DPH")
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
	if !strings.Contains(xmlStr, `kod_rezim_pl="0"`) {
		t.Error("XML should contain kod_rezim_pl=0 on VetaA4")
	}
	if !strings.Contains(xmlStr, `zdph_44="N"`) {
		t.Error("XML should contain zdph_44=N on VetaA4")
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
	if !strings.Contains(xmlStr, `khdph_forma="O"`) {
		t.Error("XML should contain khdph_forma=O for corrective filing")
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

func TestInt64Ptr(t *testing.T) {
	// Non-zero value returns pointer to that value.
	v := int64Ptr(42)
	if v == nil {
		t.Fatal("int64Ptr(42) returned nil, want non-nil")
	}
	if *v != 42 {
		t.Errorf("int64Ptr(42) = %d, want 42", *v)
	}

	// Zero value returns nil (the uncovered branch).
	if got := int64Ptr(0); got != nil {
		t.Errorf("int64Ptr(0) = %v, want nil", got)
	}
}

func TestControlStatementGenerator_Generate_ZeroAmountLine(t *testing.T) {
	gen := NewControlStatementGenerator()

	cs := &domain.VATControlStatement{
		ID: 1,
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 5,
		},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusReady,
	}

	// Line with zero base and VAT exercises int64Ptr(0) -> nil branch.
	lines := []domain.VATControlStatementLine{
		{
			Section:        "A4",
			PartnerDIC:     "CZ12345678",
			DocumentNumber: "FV20250099",
			DPPD:           "2025-05-01",
			Base:           0,
			VAT:            0,
			VATRatePercent: 21,
		},
	}

	xmlData, err := gen.Generate(cs, lines, "CZ87654321")
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	xmlStr := string(xmlData)
	// With zero amounts, zakl_dane1 and dan1 should be omitted (nil pointers).
	if strings.Contains(xmlStr, `zakl_dane1=`) {
		t.Error("zero base should result in omitted zakl_dane1 attribute")
	}
	if strings.Contains(xmlStr, `dan1=`) {
		t.Error("zero VAT should result in omitted dan1 attribute")
	}
}

func TestControlStatementGenerator_Generate_SupplementaryFiling(t *testing.T) {
	gen := NewControlStatementGenerator()

	cs := &domain.VATControlStatement{
		ID: 4,
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 2,
		},
		FilingType: domain.FilingTypeSupplementary,
		Status:     domain.FilingStatusReady,
	}

	xmlData, err := gen.Generate(cs, nil, "CZ12345678")
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	xmlStr := string(xmlData)
	if !strings.Contains(xmlStr, `khdph_forma="N"`) {
		t.Error("XML should contain khdph_forma=N for supplementary filing")
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
		{domain.FilingTypeRegular, "B"},
		{domain.FilingTypeCorrective, "O"},
		{domain.FilingTypeSupplementary, "N"},
		{"unknown", "B"},
	}

	for _, tt := range tests {
		result := FilingTypeCode(tt.input)
		if result != tt.expected {
			t.Errorf("FilingTypeCode(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestControlStatement_Golden_A4A5(t *testing.T) {
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

	invID1 := int64(10)
	invID2 := int64(11)
	lines := []domain.VATControlStatementLine{
		{
			ID:                 1,
			ControlStatementID: 1,
			Section:            "A4",
			PartnerDIC:         "CZ12345678",
			DocumentNumber:     "FV20250001",
			DPPD:               "2025-03-05",
			Base:               domain.NewAmount(25000, 0),
			VAT:                domain.NewAmount(5250, 0),
			VATRatePercent:     21,
			InvoiceID:          &invID1,
		},
		{
			ID:                 2,
			ControlStatementID: 1,
			Section:            "A4",
			PartnerDIC:         "CZ87654321",
			DocumentNumber:     "FV20250002",
			DPPD:               "2025-03-18",
			Base:               domain.NewAmount(18000, 0),
			VAT:                domain.NewAmount(2160, 0),
			VATRatePercent:     12,
			InvoiceID:          &invID2,
		},
		{
			ID:                 3,
			ControlStatementID: 1,
			Section:            "A5",
			Base:               domain.NewAmount(8500, 50),
			VAT:                domain.NewAmount(1785, 11),
			VATRatePercent:     21,
		},
		{
			ID:                 4,
			ControlStatementID: 1,
			Section:            "A5",
			Base:               domain.NewAmount(4200, 0),
			VAT:                domain.NewAmount(504, 0),
			VATRatePercent:     12,
		},
	}

	xmlData, err := gen.Generate(cs, lines, "CZ89052449")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	goldenPath := filepath.Join("testdata", "control_statement_a4a5.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}

func TestControlStatement_Golden_B2B3(t *testing.T) {
	gen := NewControlStatementGenerator()

	cs := &domain.VATControlStatement{
		ID: 2,
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
			ID:                 5,
			ControlStatementID: 2,
			Section:            "B2",
			PartnerDIC:         "CZ99887766",
			DocumentNumber:     "VF2025001",
			DPPD:               "2025-06-10",
			Base:               domain.NewAmount(35000, 0),
			VAT:                domain.NewAmount(7350, 0),
			VATRatePercent:     21,
			ExpenseID:          &expID,
		},
		{
			ID:                 6,
			ControlStatementID: 2,
			Section:            "B2",
			PartnerDIC:         "CZ11223344",
			DocumentNumber:     "VF2025002",
			DPPD:               "2025-06-22",
			Base:               domain.NewAmount(15000, 0),
			VAT:                domain.NewAmount(1800, 0),
			VATRatePercent:     12,
		},
		{
			ID:                 7,
			ControlStatementID: 2,
			Section:            "B3",
			Base:               domain.NewAmount(6000, 0),
			VAT:                domain.NewAmount(1260, 0),
			VATRatePercent:     21,
		},
		{
			ID:                 8,
			ControlStatementID: 2,
			Section:            "B3",
			Base:               domain.NewAmount(2500, 0),
			VAT:                domain.NewAmount(300, 0),
			VATRatePercent:     12,
		},
	}

	xmlData, err := gen.Generate(cs, lines, "CZ89052449")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	goldenPath := filepath.Join("testdata", "control_statement_b2b3.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}

func TestControlStatement_Golden_Corrective(t *testing.T) {
	gen := NewControlStatementGenerator()

	cs := &domain.VATControlStatement{
		ID: 3,
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 1,
		},
		FilingType: domain.FilingTypeCorrective,
		Status:     domain.FilingStatusReady,
	}

	invID := int64(30)
	lines := []domain.VATControlStatementLine{
		{
			ID:                 9,
			ControlStatementID: 3,
			Section:            "A4",
			PartnerDIC:         "CZ55667788",
			DocumentNumber:     "FV20250010",
			DPPD:               "2025-01-20",
			Base:               domain.NewAmount(42000, 0),
			VAT:                domain.NewAmount(8820, 0),
			VATRatePercent:     21,
			InvoiceID:          &invID,
		},
	}

	xmlData, err := gen.Generate(cs, lines, "CZ89052449")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	goldenPath := filepath.Join("testdata", "control_statement_corrective.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}
