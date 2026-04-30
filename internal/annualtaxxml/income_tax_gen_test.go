package annualtaxxml

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestIncomeTaxXML_NilInput(t *testing.T) {
	_, err := GenerateIncomeTaxXML(nil, map[string]string{}, nil)
	if err == nil {
		t.Fatal("expected error for nil input, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestIncomeTaxXML_CorrectiveFilingType(t *testing.T) {
	itr := &domain.IncomeTaxReturn{
		Year:            2025,
		FilingType:      domain.FilingTypeCorrective,
		TotalRevenue:    domain.NewAmount(500000, 0),
		UsedExpenses:    domain.NewAmount(300000, 0),
		TaxBase:         domain.NewAmount(200000, 0),
		TaxBaseRounded:  domain.NewAmount(200000, 0),
		TotalTax:        domain.NewAmount(30000, 0),
		TotalCredits:    domain.NewAmount(30840, 0),
		TaxAfterCredits: domain.NewAmount(0, 0),
		TaxAfterBenefit: domain.NewAmount(0, 0),
	}
	settings := map[string]string{
		"financni_urad_code":    "451",
		"taxpayer_first_name":   "Jan",
		"taxpayer_last_name":    "Novak",
		"taxpayer_birth_number": "8001011234",
		"dic":                   "CZ8001011234",
		"taxpayer_street":       "Hlavni",
		"taxpayer_house_number": "42",
		"taxpayer_city":         "Praha",
		"taxpayer_postal_code":  "11000",
	}

	xmlData, err := GenerateIncomeTaxXML(itr, settings, nil)
	if err != nil {
		t.Fatalf("GenerateIncomeTaxXML: %v", err)
	}

	goldenPath := filepath.Join("testdata", "income_tax_corrective.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}

func TestIncomeTaxXML_SupplementaryFilingType(t *testing.T) {
	itr := &domain.IncomeTaxReturn{
		Year:            2025,
		FilingType:      domain.FilingTypeSupplementary,
		TotalRevenue:    domain.NewAmount(500000, 0),
		UsedExpenses:    domain.NewAmount(300000, 0),
		TaxBase:         domain.NewAmount(200000, 0),
		TaxBaseRounded:  domain.NewAmount(200000, 0),
		TotalTax:        domain.NewAmount(30000, 0),
		TotalCredits:    domain.NewAmount(30840, 0),
		TaxAfterCredits: domain.NewAmount(0, 0),
		TaxAfterBenefit: domain.NewAmount(0, 0),
	}
	settings := map[string]string{
		"financni_urad_code":    "451",
		"taxpayer_first_name":   "Jan",
		"taxpayer_last_name":    "Novak",
		"taxpayer_birth_number": "8001011234",
		"dic":                   "CZ8001011234",
		"taxpayer_street":       "Hlavni",
		"taxpayer_house_number": "42",
		"taxpayer_city":         "Praha",
		"taxpayer_postal_code":  "11000",
	}

	xmlData, err := GenerateIncomeTaxXML(itr, settings, nil)
	if err != nil {
		t.Fatalf("GenerateIncomeTaxXML: %v", err)
	}

	goldenPath := filepath.Join("testdata", "income_tax_supplementary.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}

func TestIncomeTaxXML_Golden_Full(t *testing.T) {
	// Full return: revenue 2M CZK, flat rate 60%, progressive tax, all credits, child benefit, prepayments.
	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,

		// Section 7 - business income
		TotalRevenue:    domain.NewAmount(2000000, 0), // 2 000 000 CZK
		FlatRatePercent: 60,
		FlatRateAmount:  domain.NewAmount(1200000, 0), // 1 200 000 CZK
		UsedExpenses:    domain.NewAmount(1200000, 0), // flat-rate 60%

		// Tax base
		TaxBase:        domain.NewAmount(800000, 0), // 2M - 1.2M = 800 000 CZK
		TaxBaseRounded: domain.NewAmount(800000, 0), // already rounded to 100

		// Progressive tax: 15% on first 1 935 552 CZK => 120 000 CZK
		TaxAt15:  domain.NewAmount(120000, 0),
		TaxAt23:  domain.NewAmount(0, 0),
		TotalTax: domain.NewAmount(120000, 0),

		// Credits
		CreditBasic:      domain.NewAmount(30840, 0), // sleva na poplatnika 2025
		CreditSpouse:     domain.NewAmount(24840, 0), // sleva na manzela/ku
		CreditDisability: domain.NewAmount(2520, 0),  // invalidita I. stupne
		CreditStudent:    domain.NewAmount(4020, 0),  // sleva na studenta
		TotalCredits:     domain.NewAmount(62220, 0), // sum of all credits
		TaxAfterCredits:  domain.NewAmount(57780, 0), // 120000 - 62220

		// Child benefit
		ChildBenefit:    domain.NewAmount(15204, 0), // 1 child
		TaxAfterBenefit: domain.NewAmount(42576, 0), // 57780 - 15204

		// Prepayments
		Prepayments: domain.NewAmount(36000, 0), // 3000/month * 12
		TaxDue:      domain.NewAmount(6576, 0),  // 42576 - 36000
	}

	settings := map[string]string{
		"financni_urad_code":    "451",
		"taxpayer_first_name":   "Jan",
		"taxpayer_last_name":    "Novak",
		"taxpayer_birth_number": "8001011234",
		"dic":                   "CZ8001011234",
		"taxpayer_street":       "Hlavni",
		"taxpayer_house_number": "42",
		"taxpayer_city":         "Praha",
		"taxpayer_postal_code":  "11000",
	}

	xmlData, err := GenerateIncomeTaxXML(itr, settings, nil)
	if err != nil {
		t.Fatalf("GenerateIncomeTaxXML: %v", err)
	}

	goldenPath := filepath.Join("testdata", "income_tax_full.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}

func TestIncomeTaxXML_DeductionBreakdown(t *testing.T) {
	// Verify per-category deduction attributes are emitted with correct whole-CZK values.
	itr := &domain.IncomeTaxReturn{
		Year:                   2025,
		FilingType:             domain.FilingTypeRegular,
		TotalRevenue:           domain.NewAmount(1000000, 0),
		UsedExpenses:           domain.NewAmount(600000, 0),
		TaxBase:                domain.NewAmount(400000, 0),
		TaxBaseRounded:         domain.NewAmount(400000, 0),
		TotalDeductions:        domain.NewAmount(74000, 0),
		DeductionMortgage:      domain.NewAmount(50000, 0),
		DeductionLifeInsurance: domain.NewAmount(12000, 0),
		DeductionPension:       domain.NewAmount(8000, 0),
		DeductionDonation:      domain.NewAmount(3000, 0),
		DeductionUnionDues:     domain.NewAmount(1000, 0), // not emitted: removed from §15 in 2024
	}
	settings := map[string]string{
		"financni_urad_code":    "451",
		"taxpayer_first_name":   "Jan",
		"taxpayer_last_name":    "Novak",
		"taxpayer_birth_number": "8001011234",
		"dic":                   "CZ8001011234",
		"taxpayer_street":       "Hlavni",
		"taxpayer_house_number": "42",
		"taxpayer_city":         "Praha",
		"taxpayer_postal_code":  "11000",
	}

	xmlData, err := GenerateIncomeTaxXML(itr, settings, nil)
	if err != nil {
		t.Fatalf("GenerateIncomeTaxXML: %v", err)
	}

	// Verify each §15 deduction attribute is emitted with the expected value
	// using the EPO DPFDP7 attribute names (in VetaS).
	expected := []string{
		`kc_op28_5="50000"`,  // mortgage interest
		`kc_op15_13="12000"`, // life insurance
		`kc_op15_12="8000"`,  // pension
		`kc_op15_8="3000"`,   // donations
	}
	for _, want := range expected {
		if !bytes.Contains(xmlData, []byte(want)) {
			t.Errorf("expected XML to contain %q, got:\n%s", want, string(xmlData))
		}
	}
	// Union dues (kc_op15_14) was removed from §15 in 2024 and must not appear in DPFDP7 XML.
	if bytes.Contains(xmlData, []byte(`kc_op15_14`)) {
		t.Errorf("kc_op15_14 must not appear in DPFDP7 XML, got:\n%s", string(xmlData))
	}
}

func TestIncomeTaxXML_DICStripsCountryPrefix(t *testing.T) {
	// XSD expects dic to match [0-9]{1,10} -- the "CZ" prefix must be stripped.
	itr := &domain.IncomeTaxReturn{
		Year:         2025,
		FilingType:   domain.FilingTypeRegular,
		TotalRevenue: domain.NewAmount(100000, 0),
	}
	xmlData, err := GenerateIncomeTaxXML(itr, map[string]string{
		"financni_urad_code": "451",
		"dic":                "CZ8905244997",
	}, nil)
	if err != nil {
		t.Fatalf("GenerateIncomeTaxXML: %v", err)
	}
	if !bytes.Contains(xmlData, []byte(`dic="8905244997"`)) {
		t.Errorf("expected dic without CZ prefix, got:\n%s", string(xmlData))
	}
	if bytes.Contains(xmlData, []byte(`dic="CZ`)) {
		t.Errorf("dic still contains country prefix, got:\n%s", string(xmlData))
	}
}

func TestIncomeTaxXML_Section10TriggersPriloha2(t *testing.T) {
	// When §10 income is present, ř.40 (kc_zd10) must be filled AND VetaV / VetaJ
	// must accompany the return. EPO blocks the submission otherwise with control 510.
	itr := &domain.IncomeTaxReturn{
		Year:                2025,
		FilingType:          domain.FilingTypeRegular,
		TotalRevenue:        domain.NewAmount(500000, 0),
		UsedExpenses:        domain.NewAmount(300000, 0),
		TaxBase:             domain.NewAmount(201686, 0),
		OtherIncomeGross:    domain.NewAmount(5000, 0),
		OtherIncomeExpenses: domain.NewAmount(3314, 0),
		OtherIncomeNet:      domain.NewAmount(1686, 0),
	}
	xmlData, err := GenerateIncomeTaxXML(itr, map[string]string{
		"financni_urad_code": "451",
		"dic":                "CZ1234567890",
	}, nil)
	if err != nil {
		t.Fatalf("GenerateIncomeTaxXML: %v", err)
	}
	for _, want := range []string{
		`kc_zd10="1686"`,
		`priloha2="1"`,
		`<VetaV `,
		`kc_prij10="5000"`,
		`kc_vyd10="3314"`,
		`kc_zd10p="1686"`,
		`<VetaJ `,
		`kod_dr_prij10="D"`,
		`prijmy10="5000"`,
		`vydaje10="3314"`,
		`rozdil10="1686"`,
	} {
		if !bytes.Contains(xmlData, []byte(want)) {
			t.Errorf("expected XML to contain %q, got:\n%s", want, string(xmlData))
		}
	}
}

func TestIncomeTaxXML_Golden_Minimal(t *testing.T) {
	// Minimal return: revenue only, no credits, no deductions, no prepayments.
	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,

		// Section 7
		TotalRevenue: domain.NewAmount(350000, 0), // 350 000 CZK
		UsedExpenses: domain.NewAmount(210000, 0), // flat-rate 60%

		// Tax base
		TaxBase:        domain.NewAmount(140000, 0),
		TaxBaseRounded: domain.NewAmount(140000, 0),

		// Tax: 15% of 140 000 = 21 000
		TaxAt15:  domain.NewAmount(21000, 0),
		TotalTax: domain.NewAmount(21000, 0),

		// No credits
		TotalCredits:    domain.NewAmount(0, 0),
		TaxAfterCredits: domain.NewAmount(21000, 0),

		// No child benefit
		TaxAfterBenefit: domain.NewAmount(21000, 0),

		// No prepayments
		TaxDue: domain.NewAmount(21000, 0),
	}

	settings := map[string]string{
		"financni_urad_code":    "461",
		"taxpayer_first_name":   "Eva",
		"taxpayer_last_name":    "Svobodova",
		"taxpayer_birth_number": "9055121234",
		"dic":                   "CZ9055121234",
		"taxpayer_street":       "Namesti Miru",
		"taxpayer_house_number": "7",
		"taxpayer_city":         "Brno",
		"taxpayer_postal_code":  "60200",
	}

	xmlData, err := GenerateIncomeTaxXML(itr, settings, nil)
	if err != nil {
		t.Fatalf("GenerateIncomeTaxXML: %v", err)
	}

	goldenPath := filepath.Join("testdata", "income_tax_minimal.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}

func TestIncomeTaxXML_ChildMonthsAggregation(t *testing.T) {
	itr := &domain.IncomeTaxReturn{
		Year:            2025,
		FilingType:      domain.FilingTypeRegular,
		TotalRevenue:    domain.NewAmount(1_550_848, 0),
		FlatRatePercent: 60,
		ChildBenefit:    domain.NewAmount(37_524, 0), // 15204 + 22320 (1st + 2nd child, full year)
		TotalTax:        domain.NewAmount(91_605, 0),
	}
	children := []domain.TaxChildCredit{
		{Year: 2025, ChildOrder: 1, MonthsClaimed: 12, ZTP: false},
		{Year: 2025, ChildOrder: 2, MonthsClaimed: 12, ZTP: false},
	}
	xmlData, err := GenerateIncomeTaxXML(itr, map[string]string{
		"financni_urad_code": "451",
		"dic":                "CZ8905244997",
	}, children)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		` m_deti="12"`, ` m_deti2="12"`,
	} {
		if !bytes.Contains(xmlData, []byte(want)) {
			t.Errorf("expected XML to contain %q, got:\n%s", want, xmlData)
		}
	}
	for _, unwanted := range []string{
		` m_deti3="`, ` m_detiztpp="`, ` m_detiztpp2="`, ` m_detiztpp3="`,
	} {
		if bytes.Contains(xmlData, []byte(unwanted)) {
			t.Errorf("unexpected XML attribute %q present, got:\n%s", unwanted, xmlData)
		}
	}
}

func TestIncomeTaxXML_ChildMonthsZTPAndThirdOrder(t *testing.T) {
	itr := &domain.IncomeTaxReturn{
		Year:            2025,
		FilingType:      domain.FilingTypeRegular,
		TotalRevenue:    domain.NewAmount(2_000_000, 0),
		FlatRatePercent: 60,
		ChildBenefit:    domain.NewAmount(80_000, 0),
		TotalTax:        domain.NewAmount(120_000, 0),
	}
	children := []domain.TaxChildCredit{
		{Year: 2025, ChildOrder: 1, MonthsClaimed: 12, ZTP: true}, // -> m_detiztpp
		{Year: 2025, ChildOrder: 3, MonthsClaimed: 6, ZTP: false}, // -> m_deti3
		{Year: 2025, ChildOrder: 4, MonthsClaimed: 4, ZTP: true},  // -> m_detiztpp3 (4+ treated as 3+)
	}
	xmlData, err := GenerateIncomeTaxXML(itr, map[string]string{
		"financni_urad_code": "451",
		"dic":                "CZ8905244997",
	}, children)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		` m_detiztpp="12"`, ` m_deti3="6"`, ` m_detiztpp3="4"`,
	} {
		if !bytes.Contains(xmlData, []byte(want)) {
			t.Errorf("expected XML to contain %q, got:\n%s", want, xmlData)
		}
	}
}

func TestIncomeTaxXML_InvalidUfoCil(t *testing.T) {
	itr := &domain.IncomeTaxReturn{
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		TotalRevenue:   domain.NewAmount(500000, 0),
		UsedExpenses:   domain.NewAmount(300000, 0),
		TaxBase:        domain.NewAmount(200000, 0),
		TaxBaseRounded: domain.NewAmount(200000, 0),
		TotalTax:       domain.NewAmount(30000, 0),
	}
	baseSettings := func() map[string]string {
		return map[string]string{
			"taxpayer_first_name":   "Jan",
			"taxpayer_last_name":    "Novak",
			"taxpayer_birth_number": "8001011234",
			"dic":                   "CZ8001011234",
			"taxpayer_street":       "Hlavni",
			"taxpayer_house_number": "42",
			"taxpayer_city":         "Praha",
			"taxpayer_postal_code":  "11000",
		}
	}

	tests := []struct {
		name    string
		ufoCil  string
		wantSub string
	}{
		{"empty", "", "není vyplněn"},
		{"4-digit pracoviste code", "3034", "neplatný formát"},
		{"2-digit", "45", "neplatný formát"},
		{"alphanumeric", "45A", "neplatný formát"},
		{"5-digit", "12345", "neplatný formát"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := baseSettings()
			if tt.ufoCil != "" {
				s["financni_urad_code"] = tt.ufoCil
			}
			_, err := GenerateIncomeTaxXML(itr, s, nil)
			if err == nil {
				t.Fatalf("expected error for c_ufo_cil=%q, got nil", tt.ufoCil)
			}
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("expected ErrInvalidInput, got: %v", err)
			}
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("error message %q does not contain %q", err.Error(), tt.wantSub)
			}
		})
	}
}

func TestIncomeTaxXML_ValidUfoCil(t *testing.T) {
	itr := &domain.IncomeTaxReturn{
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		TotalRevenue:   domain.NewAmount(500000, 0),
		UsedExpenses:   domain.NewAmount(300000, 0),
		TaxBase:        domain.NewAmount(200000, 0),
		TaxBaseRounded: domain.NewAmount(200000, 0),
		TotalTax:       domain.NewAmount(30000, 0),
	}
	for _, code := range []string{"451", "461", "591", "  451  "} {
		t.Run(code, func(t *testing.T) {
			s := map[string]string{
				"financni_urad_code":    code,
				"taxpayer_first_name":   "Jan",
				"taxpayer_last_name":    "Novak",
				"taxpayer_birth_number": "8001011234",
				"dic":                   "CZ8001011234",
				"taxpayer_street":       "Hlavni",
				"taxpayer_house_number": "42",
				"taxpayer_city":         "Praha",
				"taxpayer_postal_code":  "11000",
			}
			if _, err := GenerateIncomeTaxXML(itr, s, nil); err != nil {
				t.Errorf("unexpected error for c_ufo_cil=%q: %v", code, err)
			}
		})
	}
}

func TestNormalizeNACE(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"620100", "6201"},
		{"582900", "5829"},
		{"62010", "6201"},
		{"6201", "6201"},
		{"62012", "62012"},
		{"4321", "4321"},
		{"", ""},
		{" 620100 ", "6201"},
		{"100000", "1000"},
	}
	for _, tt := range tests {
		if got := normalizeNACE(tt.in); got != tt.want {
			t.Errorf("normalizeNACE(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
