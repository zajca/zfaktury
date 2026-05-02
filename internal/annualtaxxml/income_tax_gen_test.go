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
		// EPO controls 1821/1822/1823 require these table-bottom úhrn attributes
		// to also be present and equal to the row column sums.
		`uhrn_prijmy10="5000"`,
		`uhrn_vydaje10="3314"`,
		`uhrn_rozdil10="1686"`,
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
		// VetaA rows must mirror the aggregates (EPO controls 176/318).
		` vyzdite_pocmes="12"`, ` vyzdite_pocmes2="12"`,
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

// EPO critical control 176 ("hodnota Celkem počet měsíců ... se nerovná součtu
// počtu měsíců jednotlivých řádků") fires when the m_deti* aggregates in VetaD
// have no matching VetaA rows. Verify that one VetaA row is emitted per child
// with the months landing in the slot that matches order + ZTP, and that the
// per-row name/RČ pair uses the EPO-required attribute names.
func TestIncomeTaxXML_VetaARowsMatchAggregates(t *testing.T) {
	itr := &domain.IncomeTaxReturn{
		Year:            2025,
		FilingType:      domain.FilingTypeRegular,
		TotalRevenue:    domain.NewAmount(2_000_000, 0),
		FlatRatePercent: 60,
		ChildBenefit:    domain.NewAmount(50_000, 0),
		TotalTax:        domain.NewAmount(120_000, 0),
	}
	children := []domain.TaxChildCredit{
		{Year: 2025, ChildName: "Jan Novák", BirthNumber: "200101/1234",
			ChildOrder: 1, MonthsClaimed: 12, ZTP: false},
		{Year: 2025, ChildName: "Anna Nováková", BirthNumber: "215050/5678",
			ChildOrder: 2, MonthsClaimed: 8, ZTP: true},
		{Year: 2025, ChildName: "Petr Novák", BirthNumber: "2310116789",
			ChildOrder: 3, MonthsClaimed: 6, ZTP: false},
	}
	xmlData, err := GenerateIncomeTaxXML(itr, map[string]string{
		"financni_urad_code": "451",
		"dic":                "CZ8905244997",
	}, children)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// One VetaA row per child, names split into jmeno/prijmeni, RČ slash stripped.
	for _, want := range []string{
		`<VetaA vyzdite_jmeno="Jan" vyzdite_prijmeni="Novák" vyzdite_r_cislo="2001011234" vyzdite_pocmes="12">`,
		`<VetaA vyzdite_jmeno="Anna" vyzdite_prijmeni="Nováková" vyzdite_r_cislo="2150505678" vyzdite_ztpp2="8">`,
		`<VetaA vyzdite_jmeno="Petr" vyzdite_prijmeni="Novák" vyzdite_r_cislo="2310116789" vyzdite_pocmes3="6">`,
	} {
		if !bytes.Contains(xmlData, []byte(want)) {
			t.Errorf("expected XML to contain %q, got:\n%s", want, xmlData)
		}
	}

	// VetaD aggregates must equal the column sums above (control 176).
	for _, want := range []string{` m_deti="12"`, ` m_detiztpp2="8"`, ` m_deti3="6"`} {
		if !bytes.Contains(xmlData, []byte(want)) {
			t.Errorf("expected aggregate %q, got:\n%s", want, xmlData)
		}
	}
}

func TestIncomeTaxXML_VetaAOmittedWithoutChildren(t *testing.T) {
	itr := &domain.IncomeTaxReturn{
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		TotalRevenue:   domain.NewAmount(500_000, 0),
		UsedExpenses:   domain.NewAmount(300_000, 0),
		TaxBase:        domain.NewAmount(200_000, 0),
		TaxBaseRounded: domain.NewAmount(200_000, 0),
		TotalTax:       domain.NewAmount(30_000, 0),
	}
	xmlData, err := GenerateIncomeTaxXML(itr, map[string]string{
		"financni_urad_code": "451",
		"dic":                "CZ8001011234",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bytes.Contains(xmlData, []byte("<VetaA ")) {
		t.Errorf("expected no VetaA element when no children, got:\n%s", xmlData)
	}
}

func TestSplitChildName(t *testing.T) {
	tests := []struct {
		in, wantFirst, wantLast string
	}{
		{"Jan Novák", "Jan", "Novák"},
		{"Anna van der Berg", "Anna", "van der Berg"},
		{"Novák", "", "Novák"},
		{"  Jan  Novák  ", "Jan", "Novák"},
		{"", "", ""},
	}
	for _, tt := range tests {
		first, last := splitChildName(tt.in)
		if first != tt.wantFirst || last != tt.wantLast {
			t.Errorf("splitChildName(%q) = (%q, %q), want (%q, %q)",
				tt.in, first, last, tt.wantFirst, tt.wantLast)
		}
	}
}

func TestStripBirthNumberSeparators(t *testing.T) {
	tests := []struct{ in, want string }{
		{"950101/1234", "9501011234"},
		{"950101 1234", "9501011234"},
		{"9501011234", "9501011234"},
		{"", ""},
	}
	for _, tt := range tests {
		got := stripBirthNumberSeparators(tt.in)
		if got != tt.want {
			t.Errorf("stripBirthNumberSeparators(%q) = %q, want %q", tt.in, got, tt.want)
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
		{"4-digit pracoviste code", "3034", "není v EPO číselníku"},
		{"3-digit not in codebook (legacy +10 misconception)", "581", "není v EPO číselníku"},
		{"2-digit", "45", "není v EPO číselníku"},
		{"alphanumeric", "45A", "není v EPO číselníku"},
		{"5-digit", "12345", "není v EPO číselníku"},
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
	for _, code := range []string{"451", "461", "464", "13", "  451  "} {
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

// EPO non-blocking warnings 361 (telephone) and 26 (territorial workplace) and the
// blocking control 1509 (mortgage interest months alongside amount) are all silenced
// by passing the corresponding settings through to VetaP / VetaS.
func TestIncomeTaxXML_PhonePracufoMortgageMonths(t *testing.T) {
	itr := &domain.IncomeTaxReturn{
		Year:              2025,
		FilingType:        domain.FilingTypeRegular,
		TotalRevenue:      domain.NewAmount(1_000_000, 0),
		UsedExpenses:      domain.NewAmount(600_000, 0),
		TaxBase:           domain.NewAmount(400_000, 0),
		TaxBaseRounded:    domain.NewAmount(400_000, 0),
		TotalDeductions:   domain.NewAmount(50_000, 0),
		DeductionMortgage: domain.NewAmount(50_000, 0),
		TotalTax:          domain.NewAmount(60_000, 0),
	}
	settings := map[string]string{
		"financni_urad_code":  "451",
		"c_pracufo":           "2001",
		"dic":                 "CZ8905244997",
		"taxpayer_first_name": "Jan",
		"taxpayer_last_name":  "Novak",
		"taxpayer_phone":      "+420123456789",
	}
	xmlData, err := GenerateIncomeTaxXML(itr, settings, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		` c_telef="+420123456789"`,
		` c_pracufo="2001"`,
		// Default to full year when settings["mortgage_interest_months"] is unset.
		` m_uroky="12"`,
	} {
		if !bytes.Contains(xmlData, []byte(want)) {
			t.Errorf("expected XML to contain %q, got:\n%s", want, xmlData)
		}
	}

	// Override the mortgage months via settings.
	settings["mortgage_interest_months"] = "7"
	xmlData, err = GenerateIncomeTaxXML(itr, settings, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(xmlData, []byte(` m_uroky="7"`)) {
		t.Errorf("expected m_uroky=7 from settings override, got:\n%s", xmlData)
	}

	// No mortgage deduction => m_uroky must be omitted entirely.
	itr.DeductionMortgage = domain.NewAmount(0, 0)
	itr.TotalDeductions = domain.NewAmount(0, 0)
	delete(settings, "mortgage_interest_months")
	xmlData, err = GenerateIncomeTaxXML(itr, settings, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bytes.Contains(xmlData, []byte(` m_uroky=`)) {
		t.Errorf("m_uroky must be omitted when no mortgage deduction, got:\n%s", xmlData)
	}
}

func TestPadNACEto6(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"621000", "621000"},
		{"6210", "621000"},
		{"62100", "621000"},
		{"6201", "620100"},
		{"772", "772000"},
		{"4321", "432100"},
		{"", ""},
		{" 621000 ", "621000"},
		{"1000", "100000"},
	}
	for _, tt := range tests {
		if got := padNACEto6(tt.in); got != tt.want {
			t.Errorf("padNACEto6(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// section6BaseSettings returns a minimal valid settings map for §6 tests.
func section6BaseSettings() map[string]string {
	return map[string]string{
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
}

// TestIncomeTaxXML_Section6Advance covers the typical advance case: two zálohové
// Potvrzení (vzor 33) aggregating to 240 000 Kč gross, 36 000 Kč withheld advances,
// 15 300 Kč paid monthly bonuses. potv_dazvyh stays 0 in MVP because we don't yet
// support the standalone "Potvrzení o vyplaceném daňovém bonusu" form.
func TestIncomeTaxXML_Section6Advance(t *testing.T) {
	itr := &domain.IncomeTaxReturn{
		Year:                     2025,
		FilingType:               domain.FilingTypeRegular,
		TotalRevenue:             domain.NewAmount(0, 0),
		Section6GrossIncome:      domain.NewAmount(240000, 0), // ř.31
		Section6TaxBase:          domain.NewAmount(240000, 0), // ř.34/36 (no foreign tax)
		Section6AdvanceWithheld:  domain.NewAmount(36000, 0),  // ř.84
		Section6MonthlyBonusPaid: domain.NewAmount(15300, 0),  // ř.89
		Section6CertsAdvance:     2,
		// Section6CertsBonus stays 0 -- standalone bonus form OOS in MVP
	}
	xmlData, err := GenerateIncomeTaxXML(itr, section6BaseSettings(), nil)
	if err != nil {
		t.Fatalf("GenerateIncomeTaxXML: %v", err)
	}
	for _, want := range []string{
		`kc_prij6="240000"`,
		`kc_zd6="240000"`,
		`kc_zalzavc="36000"`,
		`kc_vyplbonus="15300"`,
		`potv_zam="2"`,
	} {
		if !bytes.Contains(xmlData, []byte(want)) {
			t.Errorf("expected XML to contain %q, got:\n%s", want, xmlData)
		}
	}
	// potv_dazvyh stays 0 (omitempty drops it) in MVP -- attribute must NOT appear.
	if bytes.Contains(xmlData, []byte("potv_dazvyh=")) {
		t.Errorf("potv_dazvyh must be omitted in MVP, got:\n%s", xmlData)
	}
	// kc_zakldan23 (ř.42) must equal kc_zd6 alone -- there is no §7/§8/§10 income.
	if !bytes.Contains(xmlData, []byte(`kc_zakldan23="240000"`)) {
		t.Errorf("expected kc_zakldan23=240000 (= kc_zd6 with no §7/8/10 income), got:\n%s", xmlData)
	}
}

// TestIncomeTaxXML_Section6Withholding covers the user opting to include srážková
// daň in DAP per §36 odst. 6/7. The withholding amount must land on kc_sraz_6_4 (ř.87)
// and the attachment count potv_36 must be set.
func TestIncomeTaxXML_Section6Withholding(t *testing.T) {
	itr := &domain.IncomeTaxReturn{
		Year:                        2025,
		FilingType:                  domain.FilingTypeRegular,
		TotalRevenue:                domain.NewAmount(0, 0),
		Section6GrossIncome:         domain.NewAmount(50000, 0),
		Section6TaxBase:             domain.NewAmount(50000, 0),
		Section6WithholdingCredited: domain.NewAmount(7500, 0), // ř.87 -- §36 odst.6 sražená daň
		Section6CertsWithholding:    1,
	}
	xmlData, err := GenerateIncomeTaxXML(itr, section6BaseSettings(), nil)
	if err != nil {
		t.Fatalf("GenerateIncomeTaxXML: %v", err)
	}
	for _, want := range []string{
		`kc_prij6="50000"`,
		`kc_zd6="50000"`,
		`kc_sraz_6_4="7500"`,
		`potv_36="1"`,
	} {
		if !bytes.Contains(xmlData, []byte(want)) {
			t.Errorf("expected XML to contain %q, got:\n%s", want, xmlData)
		}
	}
}

// TestIncomeTaxXML_Section6OnlyNegativeSection7 covers the XSD critical control on
// kc_zakldan23 (ř.42): "Pokud je ř.41 záporný, uveďte pouze hodnotu z ř.36". When
// the §7+§8+§10 sum is negative, the consolidated tax base equals §6 alone.
func TestIncomeTaxXML_Section6OnlyNegativeSection7(t *testing.T) {
	itr := &domain.IncomeTaxReturn{
		Year:                2025,
		FilingType:          domain.FilingTypeRegular,
		TotalRevenue:        domain.NewAmount(100000, 0), // §7 income 100k
		UsedExpenses:        domain.NewAmount(180000, 0), // §7 expenses 180k -> loss 80k
		Section6GrossIncome: domain.NewAmount(300000, 0),
		Section6TaxBase:     domain.NewAmount(300000, 0),
	}
	xmlData, err := GenerateIncomeTaxXML(itr, section6BaseSettings(), nil)
	if err != nil {
		t.Fatalf("GenerateIncomeTaxXML: %v", err)
	}
	// kc_zakldan23 (ř.42) must equal kc_zd6 (300000) -- the §7 loss does NOT add.
	if !bytes.Contains(xmlData, []byte(`kc_zakldan23="300000"`)) {
		t.Errorf("expected kc_zakldan23=300000 (= kc_zd6, ř.41 negative dropped), got:\n%s", xmlData)
	}
	if !bytes.Contains(xmlData, []byte(`kc_zd6="300000"`)) {
		t.Errorf("expected kc_zd6=300000, got:\n%s", xmlData)
	}
	// kc_dztrata (ř.61) must hold the §7 loss (80000), unaffected by §6.
	if !bytes.Contains(xmlData, []byte(`kc_dztrata="80000"`)) {
		t.Errorf("expected kc_dztrata=80000 (§7 loss), got:\n%s", xmlData)
	}
}

// TestIncomeTaxXML_BonusReportedSeparately verifies that MonthlyBonusPaid (ř.89) is
// reported separately on kc_vyplbonus and does NOT reduce ChildBenefit (ř.72) or
// the computed bonus on ř.76. Regression for K3 in RFC-016 v2.
//
// Also asserts kc_zbyvpred (ř.91) reflects the bonus reconciliation: the user
// claimed 20 000 Kč bonus but employer only paid out 10 000 Kč → user still
// has 10 000 Kč refund coming on the DAP, i.e. kc_zbyvpred = −10 000.
func TestIncomeTaxXML_BonusReportedSeparately(t *testing.T) {
	itr := &domain.IncomeTaxReturn{
		Year:                     2025,
		FilingType:               domain.FilingTypeRegular,
		TotalRevenue:             domain.NewAmount(0, 0),
		Section6GrossIncome:      domain.NewAmount(150000, 0),
		Section6TaxBase:          domain.NewAmount(150000, 0),
		Section6MonthlyBonusPaid: domain.NewAmount(10000, 0), // ř.89 vyplacený bonus zaměstnavatelem
		Section6CertsAdvance:     1,
		ChildBenefit:             domain.NewAmount(20000, 0), // ř.72 nárok -- must remain unchanged
		// Tax 0 (no §16 calc setup here) so all of ChildBenefit becomes ř.76 bonus.
		TotalTax: domain.NewAmount(0, 0),
		// TaxAfterBenefit = −20 000 (user has full claim). TaxDue (ř.91) =
		// −20 000 + 10 000 (ř.89) = −10 000 — see calc.CalculateIncomeTax.
		// The XML generator only echoes itr.TaxDue here; assert the wired
		// value flows through to kc_zbyvpred.
		TaxDue: -domain.NewAmount(10000, 0),
	}
	xmlData, err := GenerateIncomeTaxXML(itr, section6BaseSettings(), nil)
	if err != nil {
		t.Fatalf("GenerateIncomeTaxXML: %v", err)
	}
	// kc_vyplbonus (ř.89) reflects the employer-paid bonus.
	if !bytes.Contains(xmlData, []byte(`kc_vyplbonus="10000"`)) {
		t.Errorf("expected kc_vyplbonus=10000, got:\n%s", xmlData)
	}
	// kc_dazvyhod (ř.72) keeps the full ChildBenefit -- no double-counting.
	if !bytes.Contains(xmlData, []byte(`kc_dazvyhod="20000"`)) {
		t.Errorf("expected kc_dazvyhod=20000 (ChildBenefit unchanged by ř.89), got:\n%s", xmlData)
	}
	// ř.76 (kc_danbonus) = ChildBenefit when tax after credits is 0; verify it equals 20000.
	if !bytes.Contains(xmlData, []byte(`kc_danbonus="20000"`)) {
		t.Errorf("expected kc_danbonus=20000 (ř.76 nárok unchanged), got:\n%s", xmlData)
	}
	// kc_zbyvpred (ř.91) reflects ř.89 reconciliation: user has 10 000 Kč
	// refund coming back from the státu (employer paid 10k, user entitled
	// to 20k → 10k still owed). Negative TaxDue serialises with leading "-".
	if !bytes.Contains(xmlData, []byte(`kc_zbyvpred="-10000"`)) {
		t.Errorf("expected kc_zbyvpred=-10000 (ř.91 reflects ř.72 − ř.89 = −10 000), got:\n%s", xmlData)
	}
}
