package vatxml

import (
	"encoding/xml"
	"errors"
	"maps"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
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
			TotalAmount:   domain.NewAmount(25000, 50), // 25000.50 CZK -> 25001 whole (rounded)
			ServiceCode:   "3",
		},
	}

	gen := &VIESSummaryGenerator{}
	xmlData, err := gen.Generate(vs, lines, testTaxpayerInfo())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	xmlStr := string(xmlData)

	// Check XML declaration.
	if !strings.HasPrefix(xmlStr, "<?xml") {
		t.Error("XML should start with declaration")
	}

	// Check root element. The DPHSHV schema declares no namespace.
	if !strings.Contains(xmlStr, "<Pisemnost>") {
		t.Error("XML should contain a namespace-less Pisemnost root element")
	}

	// Check VetaD attributes.
	if !strings.Contains(xmlStr, `shvies_forma="R"`) {
		t.Error("regular filing type should produce shvies_forma='R'")
	}
	if !strings.Contains(xmlStr, `rok="2025"`) {
		t.Error("XML should contain year 2025")
	}
	if !strings.Contains(xmlStr, `ctvrt="1"`) {
		t.Error("XML should contain quarter 1")
	}

	// Check VetaR lines.
	if !strings.Contains(xmlStr, `k_stat="SK"`) {
		t.Error("XML should contain SK country code")
	}
	if !strings.Contains(xmlStr, `c_vat="1234567890"`) {
		t.Error("XML should contain SK partner DIC without prefix")
	}
	if !strings.Contains(xmlStr, `k_pln_eu="3"`) {
		t.Error("XML should contain service code 3")
	}
	if !strings.Contains(xmlStr, `pln_hodnota="15000"`) {
		t.Error("XML should contain amount 15000 for SK partner")
	}

	if !strings.Contains(xmlStr, `k_stat="DE"`) {
		t.Error("XML should contain DE country code")
	}
	if !strings.Contains(xmlStr, `c_vat="987654321"`) {
		t.Error("XML should contain DE partner DIC without prefix")
	}
	if !strings.Contains(xmlStr, `pln_hodnota="25001"`) {
		t.Error("XML should round 25000.50 to 25001 whole CZK")
	}
}

func TestVIESSummaryGenerator_Generate_Corrective_Refused(t *testing.T) {
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

	assertVIESFollowUpRefused(t, vs)
}

func TestVIESSummaryGenerator_Generate_Supplementary_Refused(t *testing.T) {
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

	assertVIESFollowUpRefused(t, vs)
}

func assertVIESFollowUpRefused(t *testing.T, vs *domain.VIESSummary) {
	t.Helper()
	xmlData, err := (&VIESSummaryGenerator{}).Generate(vs, nil, testTaxpayerInfo())
	if err == nil {
		t.Fatalf("filing type %q should be refused, got XML:\n%s", vs.FilingType, xmlData)
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("error should wrap ErrInvalidInput, got %v", err)
	}
	var userMessage *domain.UserMessageError
	if !errors.As(err, &userMessage) || !strings.Contains(userMessage.Message, "ručně v EPO") {
		t.Errorf("error should carry the Czech user message, got %v", err)
	}
	if xmlData != nil {
		t.Error("no XML should be returned for a refused summary")
	}
}

func TestVIESSummaryGenerator_Generate_NilSummary(t *testing.T) {
	gen := &VIESSummaryGenerator{}
	_, err := gen.Generate(nil, nil, testTaxpayerInfo())
	if err == nil {
		t.Error("expected error for nil summary")
	}
}

func TestVIESSummaryGenerator_Generate_TaxpayerBlock(t *testing.T) {
	vs := &domain.VIESSummary{
		Period:     domain.TaxPeriod{Year: 2025, Quarter: 1},
		FilingType: domain.FilingTypeRegular,
	}

	xmlData, err := (&VIESSummaryGenerator{}).Generate(vs, nil, testTaxpayerInfo())
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	var doc struct {
		VetaD struct {
			Attrs []xml.Attr `xml:",any,attr"`
		} `xml:"DPHSHV>VetaD"`
		VetaP struct {
			Attrs []xml.Attr `xml:",any,attr"`
		} `xml:"DPHSHV>VetaP"`
	}
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		t.Fatalf("unmarshalling generated XML: %v", err)
	}
	attrs := func(in []xml.Attr) map[string]string {
		out := map[string]string{}
		for _, a := range in {
			out[a.Name.Local] = a.Value
		}
		return out
	}

	wantD := map[string]string{
		"dokument": "SHV", "k_uladis": "DPH", "d_poddp": "25.04.2025", "rok": "2025", "ctvrt": "1", "shvies_forma": "R",
	}
	if got := attrs(doc.VetaD.Attrs); !maps.Equal(got, wantD) {
		t.Errorf("VetaD = %v, want %v", got, wantD)
	}

	wantP := map[string]string{
		"c_ufo": "464", "c_pracufo": "3305", "dic": "8001010006",
		"typ_ds": TaxpayerTypeNaturalPerson, "jmeno": "Jan", "prijmeni": "Testovaci",
		"ulice": "Testovaci", "c_pop": "1", "naz_obce": "Praha", "psc": "11000",
		"stat": TaxpayerCountry,
	}
	if got := attrs(doc.VetaP.Attrs); !maps.Equal(got, wantP) {
		t.Errorf("VetaP = %v, want %v", got, wantP)
	}
}

func TestVIESSummaryGenerator_Generate_MissingUFO(t *testing.T) {
	vs := &domain.VIESSummary{Period: domain.TaxPeriod{Year: 2025, Quarter: 1}}

	info := testTaxpayerInfo()
	info.UFOCode = ""
	_, err := (&VIESSummaryGenerator{}).Generate(vs, nil, info)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("Generate() without c_ufo: err = %v, want ErrInvalidInput", err)
	}
	if !strings.Contains(err.Error(), "c_ufo") {
		t.Errorf("error should name c_ufo, got: %v", err)
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
		{domain.NewAmount(100, 50), 101}, // 100.50 -> 101 (rounded up)
		{domain.NewAmount(100, 99), 101}, // 100.99 -> 101 (rounded up)
		{domain.NewAmount(0, 0), 0},      // 0.00 -> 0
		{domain.Amount(-10050), -101},    // -100.50 -> -101 (rounded)
	}

	for _, tt := range tests {
		result := ToWholeCZK(tt.input)
		if result != tt.expected {
			t.Errorf("ToWholeCZK(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestToCeilCZK(t *testing.T) {
	tests := []struct {
		input    domain.Amount
		expected int64
	}{
		{domain.NewAmount(85500, 0), 85500},
		{domain.NewAmount(85500, 1), 85501}, // 85500.01 -> 85501
		{domain.NewAmount(100, 49), 101},    // 100.49 -> 101
		{domain.NewAmount(100, 50), 101},
		{domain.NewAmount(100, 99), 101},
		{domain.NewAmount(0, 0), 0},
		{domain.Amount(-10050), -100}, // -100.50 -> -100 (towards zero)
		{domain.Amount(-10000), -100},
	}

	for _, tt := range tests {
		if got := toCeilCZK(tt.input); got != tt.expected {
			t.Errorf("toCeilCZK(%d) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestVIESSummaryGenerator_Generate_PlnHodnotaRoundsUp(t *testing.T) {
	vs := &domain.VIESSummary{
		Period:     domain.TaxPeriod{Year: 2025, Quarter: 1},
		FilingType: domain.FilingTypeRegular,
	}
	lines := []domain.VIESSummaryLine{
		{CountryCode: "DE", PartnerDIC: "DE123456789", ServiceCode: "3", TotalAmount: domain.NewAmount(85500, 1)},
	}

	xmlData, err := (&VIESSummaryGenerator{}).Generate(vs, lines, testTaxpayerInfo())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(xmlData), `pln_hodnota="85501"`) {
		t.Errorf("85500.01 CZK should give pln_hodnota=85501, got:\n%s", xmlData)
	}
}

func TestVIESSummary_Golden_Regular(t *testing.T) {
	gen := &VIESSummaryGenerator{}

	vs := &domain.VIESSummary{
		ID: 1,
		Period: domain.TaxPeriod{
			Year:    2025,
			Quarter: 1,
		},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusReady,
	}

	lines := []domain.VIESSummaryLine{
		{
			ID:            1,
			VIESSummaryID: 1,
			PartnerDIC:    "SK2023456789",
			CountryCode:   "SK",
			TotalAmount:   domain.NewAmount(150000, 0),
			ServiceCode:   "3",
		},
		{
			ID:            2,
			VIESSummaryID: 1,
			PartnerDIC:    "DE123456789",
			CountryCode:   "DE",
			TotalAmount:   domain.NewAmount(85500, 50),
			ServiceCode:   "3",
		},
	}

	xmlData, err := gen.Generate(vs, lines, testTaxpayerInfo())
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	goldenPath := filepath.Join("testdata", "vies_summary_regular.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}

func TestVIESSummary_Golden_Empty(t *testing.T) {
	gen := &VIESSummaryGenerator{}

	vs := &domain.VIESSummary{
		ID: 2,
		Period: domain.TaxPeriod{
			Year:    2025,
			Quarter: 2,
		},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusReady,
	}

	xmlData, err := gen.Generate(vs, nil, testTaxpayerInfo())
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	goldenPath := filepath.Join("testdata", "vies_summary_empty.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}

// The VIES summary must carry d_poddp (issue #72), the same date attribute the
// VAT return writes.
func TestVIESSummaryGenerator_Generate_DPoddp(t *testing.T) {
	gen := &VIESSummaryGenerator{}
	vs := &domain.VIESSummary{
		ID:         1,
		Period:     domain.TaxPeriod{Year: 2025, Quarter: 1},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusReady,
	}

	xmlData, err := gen.Generate(vs, nil, testTaxpayerInfo())
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if want := `d_poddp="25.04.2025"`; !strings.Contains(string(xmlData), want) {
		t.Errorf("VIES XML lacks %s:\n%s", want, xmlData)
	}
}
