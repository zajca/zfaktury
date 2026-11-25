package vatxml

import (
	"encoding/xml"
	"errors"
	"maps"
	"path/filepath"
	"reflect"
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
	xmlData, err := gen.Generate(vs, lines, nil, testTaxpayerInfo())
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

// viesTestRow is a parsed VetaR line; an absent pln_hodnota stays "".
type viesTestRow struct {
	KStorno    string `xml:"k_storno,attr"`
	KStat      string `xml:"k_stat,attr"`
	CVat       string `xml:"c_vat,attr"`
	KPlnEU     string `xml:"k_pln_eu,attr"`
	PlnHodnota string `xml:"pln_hodnota,attr"`
}

type viesTestDoc struct {
	VetaD struct {
		ShviesForma string `xml:"shvies_forma,attr"`
	} `xml:"DPHSHV>VetaD"`
	VetaR []viesTestRow `xml:"DPHSHV>VetaR"`
}

func parseVIESRows(t *testing.T, xmlData []byte) viesTestDoc {
	t.Helper()
	var doc viesTestDoc
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		t.Fatalf("unmarshalling generated XML: %v", err)
	}
	return doc
}

func TestVIESSummaryGenerator_Generate_Corrective_NoChange_Refused(t *testing.T) {
	vs := &domain.VIESSummary{
		ID:         2,
		Period:     domain.TaxPeriod{Year: 2025, Quarter: 2},
		FilingType: domain.FilingTypeCorrective,
		Status:     "draft",
	}
	line := domain.VIESSummaryLine{
		PartnerDIC:  "SK1234567890",
		CountryCode: "SK",
		ServiceCode: "3",
		TotalAmount: domain.NewAmount(15000, 0),
	}

	xmlData, err := (&VIESSummaryGenerator{}).Generate(vs, []domain.VIESSummaryLine{line}, []domain.VIESSummaryLine{line}, testTaxpayerInfo())
	if err == nil {
		t.Fatalf("corrective summary without any change should be refused, got XML:\n%s", xmlData)
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("error should wrap ErrInvalidInput, got %v", err)
	}
	var userMessage *domain.UserMessageError
	if !errors.As(err, &userMessage) || !strings.Contains(userMessage.Message, "žádnou změnu") {
		t.Errorf("error should carry the Czech no-change message, got %v", err)
	}
	if xmlData != nil {
		t.Error("no XML should be returned for a no-change follow-up summary")
	}
}

func TestVIESSummaryGenerator_Generate_Supplementary_StornoRows(t *testing.T) {
	vs := &domain.VIESSummary{
		ID:         3,
		Period:     domain.TaxPeriod{Year: 2024, Quarter: 4},
		FilingType: domain.FilingTypeSupplementary,
		Status:     "draft",
	}
	baseline := []domain.VIESSummaryLine{
		{PartnerDIC: "DE987654321", CountryCode: "DE", ServiceCode: "3", TotalAmount: domain.NewAmount(25000, 0)},
	}
	lines := []domain.VIESSummaryLine{
		{PartnerDIC: "DE987654321", CountryCode: "DE", ServiceCode: "3", TotalAmount: domain.NewAmount(30000, 0)},
		{PartnerDIC: "SK1234567890", CountryCode: "SK", ServiceCode: "3", TotalAmount: domain.NewAmount(5000, 0)},
	}

	xmlData, err := (&VIESSummaryGenerator{}).Generate(vs, lines, baseline, testTaxpayerInfo())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	doc := parseVIESRows(t, xmlData)
	if doc.VetaD.ShviesForma != "N" {
		t.Errorf("shvies_forma = %q, want N for a supplementary summary", doc.VetaD.ShviesForma)
	}
	want := []viesTestRow{
		{KStorno: "A", KStat: "DE", CVat: "987654321", KPlnEU: "3", PlnHodnota: ""},
		{KStat: "DE", CVat: "987654321", KPlnEU: "3", PlnHodnota: "30000"},
		{KStat: "SK", CVat: "1234567890", KPlnEU: "3", PlnHodnota: "5000"},
	}
	if !reflect.DeepEqual(doc.VetaR, want) {
		t.Errorf("VetaR = %+v, want %+v", doc.VetaR, want)
	}
}

func TestVIESSummaryGenerator_Generate_FollowUpDiff(t *testing.T) {
	de := func(amount domain.Amount) domain.VIESSummaryLine {
		return domain.VIESSummaryLine{PartnerDIC: "DE111111111", CountryCode: "DE", ServiceCode: "3", TotalAmount: amount}
	}
	sk := func(amount domain.Amount) domain.VIESSummaryLine {
		return domain.VIESSummaryLine{PartnerDIC: "SK222222222", CountryCode: "SK", ServiceCode: "3", TotalAmount: amount}
	}
	at := func(amount domain.Amount) domain.VIESSummaryLine {
		return domain.VIESSummaryLine{PartnerDIC: "AT333333333", CountryCode: "AT", ServiceCode: "3", TotalAmount: amount}
	}
	fr := func(amount domain.Amount) domain.VIESSummaryLine {
		return domain.VIESSummaryLine{PartnerDIC: "FR444444444", CountryCode: "FR", ServiceCode: "3", TotalAmount: amount}
	}

	tests := []struct {
		name     string
		baseline []domain.VIESSummaryLine
		lines    []domain.VIESSummaryLine
		want     []viesTestRow
	}{
		{
			name:     "changed partner gets a storno row then the new row",
			baseline: []domain.VIESSummaryLine{de(domain.NewAmount(10000, 0))},
			lines:    []domain.VIESSummaryLine{de(domain.NewAmount(20000, 0))},
			want: []viesTestRow{
				{KStorno: "A", KStat: "DE", CVat: "111111111", KPlnEU: "3", PlnHodnota: ""},
				{KStat: "DE", CVat: "111111111", KPlnEU: "3", PlnHodnota: "20000"},
			},
		},
		{
			name:     "removed partner gets a storno row",
			baseline: []domain.VIESSummaryLine{sk(domain.NewAmount(5000, 0))},
			lines:    nil,
			want: []viesTestRow{
				{KStorno: "A", KStat: "SK", CVat: "222222222", KPlnEU: "3", PlnHodnota: ""},
			},
		},
		{
			name:     "added partner gets a new row",
			baseline: nil,
			lines:    []domain.VIESSummaryLine{fr(domain.NewAmount(3000, 0))},
			want: []viesTestRow{
				{KStat: "FR", CVat: "444444444", KPlnEU: "3", PlnHodnota: "3000"},
			},
		},
		{
			name:     "unchanged partner is left out",
			baseline: []domain.VIESSummaryLine{at(domain.NewAmount(7000, 0)), de(domain.NewAmount(10000, 0))},
			lines:    []domain.VIESSummaryLine{at(domain.NewAmount(7000, 0)), de(domain.NewAmount(20000, 0))},
			want: []viesTestRow{
				{KStorno: "A", KStat: "DE", CVat: "111111111", KPlnEU: "3", PlnHodnota: ""},
				{KStat: "DE", CVat: "111111111", KPlnEU: "3", PlnHodnota: "20000"},
			},
		},
		{
			name:     "values equal after rounding to whole CZK are unchanged",
			baseline: []domain.VIESSummaryLine{de(domain.NewAmount(10000, 49)), sk(domain.NewAmount(5000, 0))},
			lines:    []domain.VIESSummaryLine{de(domain.NewAmount(10000, 50)), sk(domain.NewAmount(6000, 0))},
			want: []viesTestRow{
				{KStorno: "A", KStat: "SK", CVat: "222222222", KPlnEU: "3", PlnHodnota: ""},
				{KStat: "SK", CVat: "222222222", KPlnEU: "3", PlnHodnota: "6000"},
			},
		},
		{
			name:     "zero-value real row keeps pln_hodnota 0",
			baseline: []domain.VIESSummaryLine{de(domain.NewAmount(10000, 0))},
			lines:    []domain.VIESSummaryLine{de(domain.NewAmount(20000, 0)), fr(domain.NewAmount(0, 0))},
			want: []viesTestRow{
				{KStorno: "A", KStat: "DE", CVat: "111111111", KPlnEU: "3", PlnHodnota: ""},
				{KStat: "DE", CVat: "111111111", KPlnEU: "3", PlnHodnota: "20000"},
				{KStat: "FR", CVat: "444444444", KPlnEU: "3", PlnHodnota: "0"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs := &domain.VIESSummary{
				Period:     domain.TaxPeriod{Year: 2025, Quarter: 1},
				FilingType: domain.FilingTypeCorrective,
			}
			xmlData, err := (&VIESSummaryGenerator{}).Generate(vs, tt.lines, tt.baseline, testTaxpayerInfo())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			doc := parseVIESRows(t, xmlData)
			if doc.VetaD.ShviesForma != "N" {
				t.Errorf("shvies_forma = %q, want N", doc.VetaD.ShviesForma)
			}
			if !reflect.DeepEqual(doc.VetaR, tt.want) {
				t.Errorf("VetaR = %+v, want %+v", doc.VetaR, tt.want)
			}
			for _, row := range doc.VetaR {
				if row.KStorno == "A" && row.PlnHodnota != "" {
					t.Errorf("storno row %+v must not carry pln_hodnota", row)
				}
			}
		})
	}
}

func TestVIESSummaryGenerator_Generate_RegularWithBaseline_Refused(t *testing.T) {
	vs := &domain.VIESSummary{
		Period:     domain.TaxPeriod{Year: 2025, Quarter: 1},
		FilingType: domain.FilingTypeRegular,
	}
	lines := []domain.VIESSummaryLine{
		{PartnerDIC: "DE111111111", CountryCode: "DE", ServiceCode: "3", TotalAmount: domain.NewAmount(10000, 0)},
	}
	baseline := []domain.VIESSummaryLine{
		{PartnerDIC: "DE111111111", CountryCode: "DE", ServiceCode: "3", TotalAmount: domain.NewAmount(9000, 0)},
	}

	xmlData, err := (&VIESSummaryGenerator{}).Generate(vs, lines, baseline, testTaxpayerInfo())
	if err == nil {
		t.Fatalf("regular summary with a baseline is a programming error, got XML:\n%s", xmlData)
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("error should wrap ErrInvalidInput, got %v", err)
	}
	var userMessage *domain.UserMessageError
	if errors.As(err, &userMessage) {
		t.Errorf("a regular summary with a baseline is a programming error, not a user message: %v", err)
	}
	if xmlData != nil {
		t.Error("no XML should be returned")
	}
}

func TestVIESSummaryGenerator_Generate_FollowUp_NilBaseline_AllAdded(t *testing.T) {
	vs := &domain.VIESSummary{
		Period:     domain.TaxPeriod{Year: 2025, Quarter: 1},
		FilingType: domain.FilingTypeCorrective,
	}
	lines := []domain.VIESSummaryLine{
		{PartnerDIC: "DE111111111", CountryCode: "DE", ServiceCode: "3", TotalAmount: domain.NewAmount(10000, 0)},
	}

	// A nil baseline means the filed summary of the quarter had no lines, so
	// every line is an added partner.
	xmlData, err := (&VIESSummaryGenerator{}).Generate(vs, lines, nil, testTaxpayerInfo())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	doc := parseVIESRows(t, xmlData)
	want := []viesTestRow{
		{KStat: "DE", CVat: "111111111", KPlnEU: "3", PlnHodnota: "10000"},
	}
	if !reflect.DeepEqual(doc.VetaR, want) {
		t.Errorf("VetaR = %+v, want %+v", doc.VetaR, want)
	}
}

func TestVIESSummaryGenerator_Generate_NilSummary(t *testing.T) {
	gen := &VIESSummaryGenerator{}
	_, err := gen.Generate(nil, nil, nil, testTaxpayerInfo())
	if err == nil {
		t.Error("expected error for nil summary")
	}
}

func TestVIESSummaryGenerator_Generate_TaxpayerBlock(t *testing.T) {
	vs := &domain.VIESSummary{
		Period:     domain.TaxPeriod{Year: 2025, Quarter: 1},
		FilingType: domain.FilingTypeRegular,
	}

	xmlData, err := (&VIESSummaryGenerator{}).Generate(vs, nil, nil, testTaxpayerInfo())
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
	_, err := (&VIESSummaryGenerator{}).Generate(vs, nil, nil, info)
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

	xmlData, err := (&VIESSummaryGenerator{}).Generate(vs, lines, nil, testTaxpayerInfo())
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

	xmlData, err := gen.Generate(vs, lines, nil, testTaxpayerInfo())
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	goldenPath := filepath.Join("testdata", "vies_summary_regular.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}

func TestVIESSummary_Golden_Corrective(t *testing.T) {
	gen := &VIESSummaryGenerator{}

	vs := &domain.VIESSummary{
		ID: 3,
		Period: domain.TaxPeriod{
			Year:    2025,
			Quarter: 1,
		},
		FilingType: domain.FilingTypeCorrective,
		Status:     domain.FilingStatusReady,
	}

	// Lines of the regular summary already filed for the quarter.
	baseline := []domain.VIESSummaryLine{
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
		{
			ID:            3,
			VIESSummaryID: 1,
			PartnerDIC:    "AT33333333333",
			CountryCode:   "AT",
			TotalAmount:   domain.NewAmount(7000, 0),
			ServiceCode:   "3",
		},
	}

	// Recomputed state of the quarter: SK unchanged, DE changed, AT removed,
	// FR added.
	lines := []domain.VIESSummaryLine{
		{
			ID:            4,
			VIESSummaryID: 3,
			PartnerDIC:    "SK2023456789",
			CountryCode:   "SK",
			TotalAmount:   domain.NewAmount(150000, 0),
			ServiceCode:   "3",
		},
		{
			ID:            5,
			VIESSummaryID: 3,
			PartnerDIC:    "DE123456789",
			CountryCode:   "DE",
			TotalAmount:   domain.NewAmount(90000, 25),
			ServiceCode:   "3",
		},
		{
			ID:            6,
			VIESSummaryID: 3,
			PartnerDIC:    "FR44444444444",
			CountryCode:   "FR",
			TotalAmount:   domain.NewAmount(12000, 0),
			ServiceCode:   "3",
		},
	}

	xmlData, err := gen.Generate(vs, lines, baseline, testTaxpayerInfo())
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	doc := parseVIESRows(t, xmlData)
	if doc.VetaD.ShviesForma != "N" {
		t.Errorf("shvies_forma = %q, want N", doc.VetaD.ShviesForma)
	}
	want := []viesTestRow{
		{KStorno: "A", KStat: "AT", CVat: "33333333333", KPlnEU: "3", PlnHodnota: ""},
		{KStorno: "A", KStat: "DE", CVat: "123456789", KPlnEU: "3", PlnHodnota: ""},
		{KStat: "DE", CVat: "123456789", KPlnEU: "3", PlnHodnota: "90001"},
		{KStat: "FR", CVat: "44444444444", KPlnEU: "3", PlnHodnota: "12000"},
	}
	if !reflect.DeepEqual(doc.VetaR, want) {
		t.Errorf("VetaR = %+v, want %+v", doc.VetaR, want)
	}

	goldenPath := filepath.Join("testdata", "vies_summary_corrective.golden.xml")
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

	xmlData, err := gen.Generate(vs, nil, nil, testTaxpayerInfo())
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

	xmlData, err := gen.Generate(vs, nil, nil, testTaxpayerInfo())
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if want := `d_poddp="25.04.2025"`; !strings.Contains(string(xmlData), want) {
		t.Errorf("VIES XML lacks %s:\n%s", want, xmlData)
	}
}
