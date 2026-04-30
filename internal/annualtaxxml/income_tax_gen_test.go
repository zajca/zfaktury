package annualtaxxml

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestIncomeTaxXML_NilInput(t *testing.T) {
	_, err := GenerateIncomeTaxXML(nil, map[string]string{})
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

	xmlData, err := GenerateIncomeTaxXML(itr, settings)
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

	xmlData, err := GenerateIncomeTaxXML(itr, settings)
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

	xmlData, err := GenerateIncomeTaxXML(itr, settings)
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
		DeductionUnionDues:     domain.NewAmount(1000, 0),
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

	xmlData, err := GenerateIncomeTaxXML(itr, settings)
	if err != nil {
		t.Fatalf("GenerateIncomeTaxXML: %v", err)
	}

	// Verify each §15 deduction attribute is emitted with the expected value
	// using the EPO DPFDP5 attribute names (in VetaS).
	expected := []string{
		`kc_op28_5="50000"`,  // mortgage interest
		`kc_op15_13="12000"`, // life insurance
		`kc_op15_12="8000"`,  // pension
		`kc_op15_8="3000"`,   // donations
		`kc_op15_14="1000"`,  // union dues
	}
	for _, want := range expected {
		if !bytes.Contains(xmlData, []byte(want)) {
			t.Errorf("expected XML to contain %q, got:\n%s", want, string(xmlData))
		}
	}
}

func TestIncomeTaxXML_DICStripsCountryPrefix(t *testing.T) {
	// XSD expects dic to match [0-9]{1,10} -- the "CZ" prefix must be stripped.
	itr := &domain.IncomeTaxReturn{
		Year:         2025,
		FilingType:   domain.FilingTypeRegular,
		TotalRevenue: domain.NewAmount(100000, 0),
	}
	xmlData, err := GenerateIncomeTaxXML(itr, map[string]string{"dic": "CZ8905244997"})
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

	xmlData, err := GenerateIncomeTaxXML(itr, settings)
	if err != nil {
		t.Fatalf("GenerateIncomeTaxXML: %v", err)
	}

	goldenPath := filepath.Join("testdata", "income_tax_minimal.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}
