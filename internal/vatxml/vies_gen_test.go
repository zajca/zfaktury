package vatxml

import (
	"strings"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

func TestVIESSummaryGenerator_Generate(t *testing.T) {
	vs := &domain.VIESSummary{
		ID: 1,
		Period: domain.TaxPeriod{
			Year:    2025,
			Quarter: 1,
		},
		FilingType: domain.FilingTypeRegular,
		Status:     "draft",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	lines := []domain.VIESSummaryLine{
		{
			ID:            1,
			VIESSummaryID: 1,
			PartnerDIC:    "SK1234567890",
			CountryCode:   "SK",
			TotalAmount:   domain.NewAmount(15000, 0), // 15000 CZK
			ServiceCode:   "3",
		},
		{
			ID:            2,
			VIESSummaryID: 1,
			PartnerDIC:    "DE987654321",
			CountryCode:   "DE",
			TotalAmount:   domain.NewAmount(25000, 50), // 25000.50 CZK -> 25000 whole (truncated)
			ServiceCode:   "3",
		},
	}

	gen := &VIESSummaryGenerator{}
	xmlData, err := gen.Generate(vs, lines, "CZ12345678")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	xmlStr := string(xmlData)

	// Check XML declaration.
	if !strings.HasPrefix(xmlStr, "<?xml") {
		t.Error("XML should start with declaration")
	}

	// Check root element.
	if !strings.Contains(xmlStr, "<Pisemnost") {
		t.Error("XML should contain Pisemnost root element")
	}
	if !strings.Contains(xmlStr, "xmlns=") {
		t.Error("XML should have xmlns attribute")
	}

	// Check VetaD attributes.
	if !strings.Contains(xmlStr, `k_daph="B"`) {
		t.Error("regular filing type should produce k_daph='B'")
	}
	if !strings.Contains(xmlStr, `rok="2025"`) {
		t.Error("XML should contain year 2025")
	}
	if !strings.Contains(xmlStr, `ctvrt="1"`) {
		t.Error("XML should contain quarter 1")
	}
	if !strings.Contains(xmlStr, `dic_odb="12345678"`) {
		t.Error("XML should contain filer DIC without CZ prefix")
	}

	// Check VetaP lines.
	if !strings.Contains(xmlStr, `k_stat="SK"`) {
		t.Error("XML should contain SK country code")
	}
	if !strings.Contains(xmlStr, `dic_odbe="1234567890"`) {
		t.Error("XML should contain SK partner DIC without prefix")
	}
	if !strings.Contains(xmlStr, `k_plneni="3"`) {
		t.Error("XML should contain service code 3")
	}
	if !strings.Contains(xmlStr, `obrat="15000"`) {
		t.Error("XML should contain amount 15000 for SK partner")
	}

	if !strings.Contains(xmlStr, `k_stat="DE"`) {
		t.Error("XML should contain DE country code")
	}
	if !strings.Contains(xmlStr, `dic_odbe="987654321"`) {
		t.Error("XML should contain DE partner DIC without prefix")
	}
	if !strings.Contains(xmlStr, `obrat="25000"`) {
		t.Error("XML should truncate 25000.50 to 25000 whole CZK")
	}
}

func TestVIESSummaryGenerator_Generate_Corrective(t *testing.T) {
	vs := &domain.VIESSummary{
		ID: 2,
		Period: domain.TaxPeriod{
			Year:    2025,
			Quarter: 2,
		},
		FilingType: domain.FilingTypeCorrective,
		Status:     "draft",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	gen := &VIESSummaryGenerator{}
	xmlData, err := gen.Generate(vs, nil, "CZ87654321")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	xmlStr := string(xmlData)

	if !strings.Contains(xmlStr, `k_daph="O"`) {
		t.Error("corrective filing type should produce k_daph='O'")
	}
	if !strings.Contains(xmlStr, `ctvrt="2"`) {
		t.Error("XML should contain quarter 2")
	}
}

func TestVIESSummaryGenerator_Generate_Supplementary(t *testing.T) {
	vs := &domain.VIESSummary{
		ID: 3,
		Period: domain.TaxPeriod{
			Year:    2024,
			Quarter: 4,
		},
		FilingType: domain.FilingTypeSupplementary,
		Status:     "draft",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	gen := &VIESSummaryGenerator{}
	xmlData, err := gen.Generate(vs, nil, "12345678")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	xmlStr := string(xmlData)

	if !strings.Contains(xmlStr, `k_daph="N"`) {
		t.Error("supplementary filing type should produce k_daph='N'")
	}
	// DIC without CZ prefix should be passed through as-is.
	if !strings.Contains(xmlStr, `dic_odb="12345678"`) {
		t.Error("DIC without country prefix should be used as-is")
	}
}

func TestVIESSummaryGenerator_Generate_NilSummary(t *testing.T) {
	gen := &VIESSummaryGenerator{}
	_, err := gen.Generate(nil, nil, "CZ12345678")
	if err == nil {
		t.Error("expected error for nil summary")
	}
}

func TestStripCountryPrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CZ12345678", "12345678"},
		{"SK1234567890", "1234567890"},
		{"DE987654321", "987654321"},
		{"12345678", "12345678"},
		{"AB", "AB"},
		{"", ""},
	}

	for _, tt := range tests {
		result := stripCountryPrefix(tt.input)
		if result != tt.expected {
			t.Errorf("stripCountryPrefix(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestVIESAmountToWholeCZK(t *testing.T) {
	tests := []struct {
		input    domain.Amount
		expected int64
	}{
		{domain.NewAmount(100, 0), 100},  // 100.00 -> 100
		{domain.NewAmount(100, 49), 100}, // 100.49 -> 100
		{domain.NewAmount(100, 50), 100}, // 100.50 -> 100 (truncated)
		{domain.NewAmount(100, 99), 100}, // 100.99 -> 100 (truncated)
		{domain.NewAmount(0, 0), 0},      // 0.00 -> 0
		{domain.Amount(-10050), -100},    // -100.50 -> -100 (truncated toward zero)
	}

	for _, tt := range tests {
		result := ToWholeCZK(tt.input)
		if result != tt.expected {
			t.Errorf("ToWholeCZK(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestVIESFilingTypeCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{domain.FilingTypeRegular, "B"},
		{domain.FilingTypeCorrective, "O"},
		{domain.FilingTypeSupplementary, "N"},
		{"", "B"},
		{"unknown", "B"},
	}

	for _, tt := range tests {
		result := viesFilingTypeCode(tt.input)
		if result != tt.expected {
			t.Errorf("viesFilingTypeCode(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
