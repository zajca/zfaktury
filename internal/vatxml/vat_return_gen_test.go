package vatxml

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

var fixedSubmissionDate = time.Date(2025, 4, 25, 0, 0, 0, 0, time.UTC)

func testTaxpayerInfo() TaxpayerInfo {
	return TaxpayerInfo{
		DIC:            "8905244997",
		FirstName:      "Martin",
		LastName:       "Zajic",
		Street:         "Ludkovice",
		HouseNum:       "189",
		ZIP:            "76341",
		City:           "Ludkovice",
		Phone:          "776598983",
		Email:          "ja@mzajic.cz",
		UFOCode:        "464",
		PracUFO:        "3305",
		OKEC:           "582900",
		SubmissionDate: fixedSubmissionDate,
	}
}

func TestToWholeCZK_Rounding(t *testing.T) {
	tests := []struct {
		name   string
		input  domain.Amount
		expect int64
	}{
		{"rounds up 2637558 haleru", 2637558, 26376},
		{"rounds up 161086 haleru", 161086, 1611},
		{"exact division", 100000, 1000},
		{"rounds down 149", 149, 1},
		{"rounds up 150", 150, 2},
		{"negative rounds", -2637558, -26376},
		{"zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToWholeCZK(tt.input)
			if got != tt.expect {
				t.Errorf("ToWholeCZK(%d) = %d, want %d", tt.input, got, tt.expect)
			}
		})
	}
}

func TestDPHFilingTypeCode(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{domain.FilingTypeRegular, "B"},
		{domain.FilingTypeCorrective, "O"},
		{domain.FilingTypeSupplementary, "D"},
		{"", "B"},
	}
	for _, tt := range tests {
		got := DPHFilingTypeCode(tt.input)
		if got != tt.expect {
			t.Errorf("DPHFilingTypeCode(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestVATReturnGenerator_Generate(t *testing.T) {
	gen := &VATReturnGenerator{}

	vr := &domain.VATReturn{
		ID: 1,
		Period: domain.TaxPeriod{
			Year:  2026,
			Month: 2,
		},
		FilingType:        domain.FilingTypeRegular,
		OutputVATBase21:   12559800, // 125598.00 CZK
		OutputVATAmount21: 2637558,  // 26375.58 CZK -> rounds to 26376
		InputVATBase21:    767100,   // 7671.00 CZK
		InputVATAmount21:  161086,   // 1610.86 CZK -> rounds to 1611
		TotalOutputVAT:    2637558,
		TotalInputVAT:     161086,
		NetVAT:            2476472,
		Status:            "draft",
	}

	xmlData, err := gen.Generate(vr, testTaxpayerInfo())
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

	if doc.DPHDP3 == nil {
		t.Fatal("expected DPHDP3 element")
	}

	// Verify DPHDP3 attributes.
	if doc.DPHDP3.VerzePis != "01.02.16" {
		t.Errorf("DPHDP3.VerzePis = %q, want %q", doc.DPHDP3.VerzePis, "01.02.16")
	}

	// Verify root attributes.
	if doc.NazevSW != "ZFaktury" {
		t.Errorf("Pisemnost.nazevSW = %q, want %q", doc.NazevSW, "ZFaktury")
	}

	// Verify VetaD.
	d := doc.DPHDP3.VetaD
	if d.Dokument != "DP3" {
		t.Errorf("VetaD.dokument = %q, want %q", d.Dokument, "DP3")
	}
	if d.KUladis != "DPH" {
		t.Errorf("VetaD.k_uladis = %q, want %q", d.KUladis, "DPH")
	}
	if d.DapdphForma != "B" {
		t.Errorf("VetaD.dapdph_forma = %q, want %q", d.DapdphForma, "B")
	}
	if d.TypPlatce != "P" {
		t.Errorf("VetaD.typ_platce = %q, want %q", d.TypPlatce, "P")
	}
	if d.Trans != "A" {
		t.Errorf("VetaD.trans = %q, want %q", d.Trans, "A")
	}
	if d.COkec != "582900" {
		t.Errorf("VetaD.c_okec = %q, want %q", d.COkec, "582900")
	}
	if d.Rok != 2026 {
		t.Errorf("VetaD.rok = %d, want 2026", d.Rok)
	}
	if d.Mesic != 2 {
		t.Errorf("VetaD.mesic = %d, want 2", d.Mesic)
	}

	// Verify VetaP.
	p := doc.DPHDP3.VetaP
	if p.DIC != "8905244997" {
		t.Errorf("VetaP.dic = %q, want %q", p.DIC, "8905244997")
	}
	if p.TypDS != "F" {
		t.Errorf("VetaP.typ_ds = %q, want %q", p.TypDS, "F")
	}
	if p.CUfo != "464" {
		t.Errorf("VetaP.c_ufo = %q, want %q", p.CUfo, "464")
	}
	if p.CPracufo != "3305" {
		t.Errorf("VetaP.c_pracufo = %q, want %q", p.CPracufo, "3305")
	}
	if p.Jmeno != "Martin" {
		t.Errorf("VetaP.jmeno = %q, want %q", p.Jmeno, "Martin")
	}
	if p.Prijmeni != "Zajic" {
		t.Errorf("VetaP.prijmeni = %q, want %q", p.Prijmeni, "Zajic")
	}
	if p.Stat != "CESKA REPUBLIKA" {
		t.Errorf("VetaP.stat = %q, want %q", p.Stat, "CESKA REPUBLIKA")
	}

	// Verify Veta1 - output VAT with correct rounding.
	if doc.DPHDP3.Veta1 == nil {
		t.Fatal("expected Veta1 element")
	}
	if doc.DPHDP3.Veta1.Obrat23 != 125598 {
		t.Errorf("Veta1.obrat23 = %v, want 125598", doc.DPHDP3.Veta1.Obrat23)
	}
	if doc.DPHDP3.Veta1.Dan23 != 26376 {
		t.Errorf("Veta1.dan23 = %v, want 26376", doc.DPHDP3.Veta1.Dan23)
	}

	// Verify Veta4 - input VAT with correct rounding.
	if doc.DPHDP3.Veta4 == nil {
		t.Fatal("expected Veta4 element")
	}
	if doc.DPHDP3.Veta4.Pln23 != 7671 {
		t.Errorf("Veta4.pln23 = %v, want 7671", doc.DPHDP3.Veta4.Pln23)
	}
	if doc.DPHDP3.Veta4.OdpTuz23Nar != 1611 {
		t.Errorf("Veta4.odp_tuz23_nar = %v, want 1611", doc.DPHDP3.Veta4.OdpTuz23Nar)
	}
	if doc.DPHDP3.Veta4.OdpSumNar != 1611 {
		t.Errorf("Veta4.odp_sum_nar = %v, want 1611", doc.DPHDP3.Veta4.OdpSumNar)
	}

	// Verify Veta6 - summary.
	if doc.DPHDP3.Veta6 == nil {
		t.Fatal("expected Veta6 element")
	}
	if doc.DPHDP3.Veta6.DanZocelk != 26376 {
		t.Errorf("Veta6.dan_zocelk = %v, want 26376", doc.DPHDP3.Veta6.DanZocelk)
	}
	if doc.DPHDP3.Veta6.OdpZocelk != 1611 {
		t.Errorf("Veta6.odp_zocelk = %v, want 1611", doc.DPHDP3.Veta6.OdpZocelk)
	}
	if doc.DPHDP3.Veta6.DanoDa != 24765 {
		t.Errorf("Veta6.dano_da = %v, want 24765", doc.DPHDP3.Veta6.DanoDa)
	}

	// Verify DPHDP3 element name in raw XML.
	if !strings.Contains(xmlStr, "<DPHDP3") {
		t.Error("expected DPHDP3 element in XML output")
	}
	if strings.Contains(xmlStr, "DPHDAP3") {
		t.Error("should not contain old DPHDAP3 element name")
	}
}

func TestSubmissionDate_ZeroFallback(t *testing.T) {
	// When SubmissionDate is zero, submissionDate() should return approximately time.Now().
	info := TaxpayerInfo{
		DIC: "12345678",
	}
	before := time.Now()
	got := submissionDate(info)
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Errorf("submissionDate() with zero SubmissionDate returned %v, expected between %v and %v", got, before, after)
	}
}

func TestSubmissionDate_ExplicitDate(t *testing.T) {
	explicit := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	info := TaxpayerInfo{
		DIC:            "12345678",
		SubmissionDate: explicit,
	}
	got := submissionDate(info)
	if !got.Equal(explicit) {
		t.Errorf("submissionDate() = %v, want %v", got, explicit)
	}
}

func TestVATReturnGenerator_Generate_ZeroSubmissionDate(t *testing.T) {
	gen := &VATReturnGenerator{}
	vr := &domain.VATReturn{
		Period:     domain.TaxPeriod{Year: 2025, Month: 1},
		FilingType: domain.FilingTypeRegular,
	}

	// Info with zero SubmissionDate should use time.Now() and not error.
	info := TaxpayerInfo{
		DIC:     "12345678",
		PracUFO: "3305",
		UFOCode: "464",
	}

	xmlData, err := gen.Generate(vr, info)
	if err != nil {
		t.Fatalf("Generate() with zero SubmissionDate returned error: %v", err)
	}

	// Verify the XML was generated with today's date.
	xmlStr := string(xmlData)
	todayFormatted := time.Now().Format("02.01.2006")
	if !strings.Contains(xmlStr, fmt.Sprintf(`d_poddp="%s"`, todayFormatted)) {
		t.Errorf("expected d_poddp to contain today's date %q in XML output", todayFormatted)
	}
}

func TestVATReturnGenerator_Generate_NilReturn(t *testing.T) {
	gen := &VATReturnGenerator{}
	_, err := gen.Generate(nil, testTaxpayerInfo())
	if err == nil {
		t.Error("expected error for nil vat return")
	}
}

func TestVATReturnGenerator_Generate_EmptyDIC(t *testing.T) {
	gen := &VATReturnGenerator{}
	vr := &domain.VATReturn{
		Period: domain.TaxPeriod{Year: 2025, Month: 1},
	}
	_, err := gen.Generate(vr, TaxpayerInfo{})
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

	info := testTaxpayerInfo()
	xmlData, err := gen.Generate(vr, info)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	var doc DPHPisemnost
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if doc.DPHDP3 == nil {
		t.Fatal("expected DPHDP3 element")
	}
	if doc.DPHDP3.VetaD.DapdphForma != "O" {
		t.Errorf("VetaD.dapdph_forma = %q, want %q for corrective", doc.DPHDP3.VetaD.DapdphForma, "O")
	}
}

func TestVATReturnGenerator_Generate_SupplementaryType(t *testing.T) {
	gen := &VATReturnGenerator{}
	vr := &domain.VATReturn{
		Period:     domain.TaxPeriod{Year: 2025, Month: 6},
		FilingType: domain.FilingTypeSupplementary,
	}

	info := testTaxpayerInfo()
	xmlData, err := gen.Generate(vr, info)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	var doc DPHPisemnost
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if doc.DPHDP3 == nil {
		t.Fatal("expected DPHDP3 element")
	}
	if doc.DPHDP3.VetaD.DapdphForma != "D" {
		t.Errorf("VetaD.dapdph_forma = %q, want %q for supplementary", doc.DPHDP3.VetaD.DapdphForma, "D")
	}
}

func TestVATReturnGenerator_Generate_ZeroAmounts(t *testing.T) {
	gen := &VATReturnGenerator{}
	vr := &domain.VATReturn{
		Period:     domain.TaxPeriod{Year: 2025, Quarter: 1},
		FilingType: domain.FilingTypeRegular,
	}

	info := testTaxpayerInfo()
	xmlData, err := gen.Generate(vr, info)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	var doc DPHPisemnost
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Zero amounts: trans should be "N".
	if doc.DPHDP3.VetaD.Trans != "N" {
		t.Errorf("VetaD.trans = %q, want %q for zero amounts", doc.DPHDP3.VetaD.Trans, "N")
	}

	// Veta1 should still exist (DPHDP3 always includes all sections).
	if doc.DPHDP3.Veta1 == nil {
		t.Error("expected Veta1 even with zero amounts")
	}
	if doc.DPHDP3.Veta1.Obrat23 != 0 {
		t.Errorf("Veta1.obrat23 = %v, want 0", doc.DPHDP3.Veta1.Obrat23)
	}
}

func TestVATReturnGenerator_Generate_NegativeNetVAT(t *testing.T) {
	gen := &VATReturnGenerator{}
	vr := &domain.VATReturn{
		Period:           domain.TaxPeriod{Year: 2025, Month: 3},
		FilingType:       domain.FilingTypeRegular,
		InputVATBase21:   500000,
		InputVATAmount21: 105000,
		TotalInputVAT:    105000,
		NetVAT:           -105000,
	}

	info := testTaxpayerInfo()
	xmlData, err := gen.Generate(vr, info)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	var doc DPHPisemnost
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if doc.DPHDP3.Veta6.DanoDa != 0 {
		t.Errorf("Veta6.dano_da = %v, want 0 for negative net VAT", doc.DPHDP3.Veta6.DanoDa)
	}
	// dano_no should be positive abs value as string.
	if doc.DPHDP3.Veta6.DanoNo != "1050.0" {
		t.Errorf("Veta6.dano_no = %q, want %q", doc.DPHDP3.Veta6.DanoNo, "1050.0")
	}
}

func TestVATReturnGenerator_Generate_Golden_Regular(t *testing.T) {
	gen := &VATReturnGenerator{}
	info := testTaxpayerInfo()

	vr := &domain.VATReturn{
		ID: 1,
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 1,
		},
		FilingType:        domain.FilingTypeRegular,
		OutputVATBase21:   domain.NewAmount(125598, 0), // 125598 CZK
		OutputVATAmount21: domain.NewAmount(26375, 58), // 26375.58 CZK -> 26376
		OutputVATBase12:   domain.NewAmount(45000, 0),  // 45000 CZK
		OutputVATAmount12: domain.NewAmount(5400, 0),   // 5400 CZK
		InputVATBase21:    domain.NewAmount(7671, 0),   // 7671 CZK
		InputVATAmount21:  domain.NewAmount(1610, 86),  // 1610.86 CZK -> 1611
		InputVATBase12:    domain.NewAmount(3200, 0),   // 3200 CZK
		InputVATAmount12:  domain.NewAmount(384, 0),    // 384 CZK
	}

	xmlData, err := gen.Generate(vr, info)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	goldenPath := filepath.Join("testdata", "vat_return_regular.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}

func TestVATReturnGenerator_Generate_Golden_Zero(t *testing.T) {
	gen := &VATReturnGenerator{}
	info := testTaxpayerInfo()

	vr := &domain.VATReturn{
		ID: 2,
		Period: domain.TaxPeriod{
			Year:    2025,
			Quarter: 1,
		},
		FilingType: domain.FilingTypeRegular,
	}

	xmlData, err := gen.Generate(vr, info)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	goldenPath := filepath.Join("testdata", "vat_return_zero.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}

func TestVATReturnGenerator_Generate_Golden_Negative(t *testing.T) {
	gen := &VATReturnGenerator{}
	info := testTaxpayerInfo()

	// Refund scenario: large input VAT, no output VAT.
	vr := &domain.VATReturn{
		ID: 3,
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 3,
		},
		FilingType:       domain.FilingTypeRegular,
		InputVATBase21:   domain.NewAmount(80000, 0),
		InputVATAmount21: domain.NewAmount(16800, 0),
		InputVATBase12:   domain.NewAmount(20000, 0),
		InputVATAmount12: domain.NewAmount(2400, 0),
	}

	xmlData, err := gen.Generate(vr, info)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	goldenPath := filepath.Join("testdata", "vat_return_negative.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}

func TestVATReturnGenerator_Generate_Golden_Corrective(t *testing.T) {
	gen := &VATReturnGenerator{}
	info := testTaxpayerInfo()

	vr := &domain.VATReturn{
		ID: 4,
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 2,
		},
		FilingType:        domain.FilingTypeCorrective,
		OutputVATBase21:   domain.NewAmount(50000, 0),
		OutputVATAmount21: domain.NewAmount(10500, 0),
		InputVATBase21:    domain.NewAmount(10000, 0),
		InputVATAmount21:  domain.NewAmount(2100, 0),
	}

	xmlData, err := gen.Generate(vr, info)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	goldenPath := filepath.Join("testdata", "vat_return_corrective.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}
