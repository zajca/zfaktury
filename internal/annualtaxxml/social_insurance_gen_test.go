package annualtaxxml

import (
	"encoding/xml"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestSocialInsuranceXML_NilInput(t *testing.T) {
	_, err := GenerateSocialInsuranceXML(nil, map[string]string{}, 0)
	if err == nil {
		t.Fatal("expected error for nil input, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestSocialInsuranceXML_CorrectiveFilingType(t *testing.T) {
	sio := &domain.SocialInsuranceOverview{
		Year:                2025,
		FilingType:          domain.FilingTypeCorrective,
		TotalRevenue:        domain.NewAmount(900000, 0),
		TotalExpenses:       domain.NewAmount(620000, 0),
		TaxBase:             domain.NewAmount(280000, 0),
		AssessmentBase:      domain.NewAmount(154000, 0),
		MinAssessmentBase:   domain.NewAmount(131736, 0),
		FinalAssessmentBase: domain.NewAmount(154000, 0),
		InsuranceRate:       292,
		TotalInsurance:      domain.NewAmount(44968, 0),
		Prepayments:         domain.NewAmount(38088, 0),
		Difference:          domain.NewAmount(6880, 0),
		NewMonthlyPrepay:    domain.NewAmount(3748, 0),
		ActivityMonths:      domain.FullYearActivityMonths,
	}

	settings := map[string]string{
		"cssz_code":             "775",
		"taxpayer_first_name":   "Jan",
		"taxpayer_last_name":    "Novak",
		"taxpayer_birth_number": "8001011234",
		"taxpayer_birth_date":   "1980-01-01",
		"taxpayer_street":       "Hlavni",
		"taxpayer_house_number": "42",
		"taxpayer_postal_code":  "11000",
		"taxpayer_city":         "Praha",
		"flat_rate_expenses":    "false",
	}

	xmlData, err := GenerateSocialInsuranceXML(sio, settings, testAdvanceBase(sio))
	if err != nil {
		t.Fatalf("GenerateSocialInsuranceXML: %v", err)
	}

	goldenPath := filepath.Join("testdata", "social_insurance_corrective.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}

func TestSocialInsuranceXML_SupplementaryFilingType(t *testing.T) {
	sio := &domain.SocialInsuranceOverview{
		Year:                2025,
		FilingType:          domain.FilingTypeSupplementary,
		TotalRevenue:        domain.NewAmount(900000, 0),
		TotalExpenses:       domain.NewAmount(620000, 0),
		TaxBase:             domain.NewAmount(280000, 0),
		AssessmentBase:      domain.NewAmount(154000, 0),
		MinAssessmentBase:   domain.NewAmount(131736, 0),
		FinalAssessmentBase: domain.NewAmount(154000, 0),
		InsuranceRate:       292,
		TotalInsurance:      domain.NewAmount(44968, 0),
		Prepayments:         domain.NewAmount(38088, 0),
		Difference:          domain.NewAmount(6880, 0),
		NewMonthlyPrepay:    domain.NewAmount(3748, 0),
		ActivityMonths:      domain.FullYearActivityMonths,
	}

	settings := map[string]string{
		"cssz_code":             "775",
		"taxpayer_first_name":   "Jan",
		"taxpayer_last_name":    "Novak",
		"taxpayer_birth_number": "8001011234",
		"taxpayer_birth_date":   "1980-01-01",
		"taxpayer_street":       "Hlavni",
		"taxpayer_house_number": "42",
		"taxpayer_postal_code":  "11000",
		"taxpayer_city":         "Praha",
		"flat_rate_expenses":    "true",
	}

	xmlData, err := GenerateSocialInsuranceXML(sio, settings, testAdvanceBase(sio))
	if err != nil {
		t.Fatalf("GenerateSocialInsuranceXML: %v", err)
	}

	goldenPath := filepath.Join("testdata", "social_insurance_supplementary.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}

func TestSocialInsuranceXML_Golden_Regular(t *testing.T) {
	// Standard overview with flat rate expenses.
	sio := &domain.SocialInsuranceOverview{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,

		TotalRevenue:        domain.NewAmount(2000000, 0), // 2 000 000 CZK
		TotalExpenses:       domain.NewAmount(1200000, 0), // flat-rate 60%
		TaxBase:             domain.NewAmount(800000, 0),  // 2M - 1.2M
		AssessmentBase:      domain.NewAmount(440000, 0),  // 55% of 800 000
		MinAssessmentBase:   domain.NewAmount(131736, 0),  // minimum for 2025
		FinalAssessmentBase: domain.NewAmount(440000, 0),  // max(440000, 131736)
		InsuranceRate:       292,                          // 29.2%
		TotalInsurance:      domain.NewAmount(128480, 0),  // 440000 * 0.292
		Prepayments:         domain.NewAmount(38088, 0),   // 3174/month * 12
		Difference:          domain.NewAmount(90392, 0),   // 128480 - 38088
		NewMonthlyPrepay:    domain.NewAmount(10707, 0),   // new monthly prepayment
		ActivityMonths:      domain.FullYearActivityMonths,
	}

	settings := map[string]string{
		"cssz_code":             "775",
		"taxpayer_first_name":   "Jan",
		"taxpayer_last_name":    "Novak",
		"taxpayer_birth_number": "8001011234",
		"taxpayer_birth_date":   "1980-01-01",
		"taxpayer_street":       "Hlavni",
		"taxpayer_house_number": "42",
		"taxpayer_postal_code":  "11000",
		"taxpayer_city":         "Praha",
		"flat_rate_expenses":    "true",
		"flat_rate_percent":     "60",
	}

	xmlData, err := GenerateSocialInsuranceXML(sio, settings, testAdvanceBase(sio))
	if err != nil {
		t.Fatalf("GenerateSocialInsuranceXML: %v", err)
	}

	goldenPath := filepath.Join("testdata", "social_insurance_regular.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}

func TestSocialInsuranceXML_Golden_ActualExpenses(t *testing.T) {
	// Overview without flat rate -- actual expenses used.
	sio := &domain.SocialInsuranceOverview{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,

		TotalRevenue:        domain.NewAmount(900000, 0), // 900 000 CZK
		TotalExpenses:       domain.NewAmount(620000, 0), // actual expenses
		TaxBase:             domain.NewAmount(280000, 0), // 900k - 620k
		AssessmentBase:      domain.NewAmount(154000, 0), // 55% of 280 000
		MinAssessmentBase:   domain.NewAmount(131736, 0), // minimum for 2025
		FinalAssessmentBase: domain.NewAmount(154000, 0), // max(154000, 131736)
		InsuranceRate:       292,                         // 29.2%
		TotalInsurance:      domain.NewAmount(44968, 0),  // 154000 * 0.292
		Prepayments:         domain.NewAmount(38088, 0),  // 3174/month * 12
		Difference:          domain.NewAmount(6880, 0),   // 44968 - 38088
		NewMonthlyPrepay:    domain.NewAmount(3748, 0),   // new monthly prepayment
		ActivityMonths:      domain.FullYearActivityMonths,
	}

	settings := map[string]string{
		"cssz_code":             "775",
		"taxpayer_first_name":   "Eva",
		"taxpayer_last_name":    "Svobodova",
		"taxpayer_birth_number": "9055121234",
		"taxpayer_birth_date":   "1990-05-12",
		"taxpayer_street":       "Namesti Miru",
		"taxpayer_house_number": "7",
		"taxpayer_postal_code":  "60200",
		"taxpayer_city":         "Brno",
		"flat_rate_expenses":    "false",
		"flat_rate_percent":     "0",
	}

	xmlData, err := GenerateSocialInsuranceXML(sio, settings, testAdvanceBase(sio))
	if err != nil {
		t.Fatalf("GenerateSocialInsuranceXML: %v", err)
	}

	goldenPath := filepath.Join("testdata", "social_insurance_actual_expenses.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}

func TestSocialInsuranceXML_SemanticFieldsFromCSSZSpec(t *testing.T) {
	sio := &domain.SocialInsuranceOverview{
		Year:                2025,
		FilingType:          domain.FilingTypeRegular,
		TaxBase:             domain.NewAmount(620339, 20),
		AssessmentBase:      domain.NewAmount(341187, 0),
		FinalAssessmentBase: domain.NewAmount(341187, 0),
		TotalInsurance:      domain.NewAmount(99627, 0),
		Prepayments:         domain.NewAmount(86991, 0),
		Difference:          domain.NewAmount(12636, 0),
		NewMonthlyPrepay:    domain.NewAmount(8303, 0),
		ActivityMonths:      domain.FullYearActivityMonths,
	}
	settings := validSocialInsuranceSettings()
	settings["cssz_variable_symbol"] = "75660248"
	settings["taxpayer_title"] = "Ing."
	settings["taxpayer_databox_id"] = "7rgmm99"
	settings["sickness_insurance_monthly"] = "768"

	xmlData, err := GenerateSocialInsuranceXML(sio, settings, testAdvanceBase(sio))
	if err != nil {
		t.Fatalf("GenerateSocialInsuranceXML: %v", err)
	}

	var doc OSVC
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		t.Fatalf("unmarshal generated XML: %v", err)
	}

	if doc.Prehled.For != "prehledosvc" {
		t.Errorf("prehledosvc@for = %q, want prehledosvc", doc.Prehled.For)
	}
	if doc.Prehled.Dep != "775" {
		t.Errorf("prehledosvc@dep = %q, want 775", doc.Prehled.Dep)
	}
	if doc.Prehled.VSDP != "75660248" {
		t.Errorf("prehledosvc@vsdp = %q, want 75660248", doc.Prehled.VSDP)
	}
	if doc.Prehled.Client.Hlavc.M13 != "A" || doc.Prehled.Client.Hlavc.M1 != "" {
		t.Errorf("hlavc flags = m1:%q m13:%q, want full-year marker only in m13", doc.Prehled.Client.Hlavc.M1, doc.Prehled.Client.Hlavc.M13)
	}
	if doc.Prehled.PVV.Pri != "620339.20" {
		t.Errorf("pvv@pri = %q, want 620339.20", doc.Prehled.PVV.Pri)
	}
	if doc.Prehled.PVV.Mesc.H != "12" || doc.Prehled.PVV.Mesv.H != "12" {
		t.Errorf("pvv months = mesc:%q mesv:%q, want 12/12", doc.Prehled.PVV.Mesc.H, doc.Prehled.PVV.Mesv.H)
	}
	if doc.Prehled.PVV.Mesp != "51694.93" {
		t.Errorf("mesp = %q, want 51694.93", doc.Prehled.PVV.Mesp)
	}
	if doc.Prehled.PVV.VVZ.H != "341187" || doc.Prehled.PVV.Poj != "99627" || doc.Prehled.PVV.Ned != "12636" {
		t.Errorf("pvv totals = vvz:%q poj:%q ned:%q, want 341187/99627/12636", doc.Prehled.PVV.VVZ.H, doc.Prehled.PVV.Poj, doc.Prehled.PVV.Ned)
	}
	if doc.Prehled.Zal.Ved != "H" || doc.Prehled.Zal.VZ != "28433" || doc.Prehled.Zal.DP != "8303" || doc.Prehled.Zal.NP != "768" {
		t.Errorf("zal attrs = ved:%q vz:%q dp:%q np:%q, want H/28433/8303/768", doc.Prehled.Zal.Ved, doc.Prehled.Zal.VZ, doc.Prehled.Zal.DP, doc.Prehled.Zal.NP)
	}
}

func TestSocialInsuranceXML_RequiresCSSZSpecFields(t *testing.T) {
	_, err := GenerateSocialInsuranceXML(&domain.SocialInsuranceOverview{Year: 2025, ActivityMonths: domain.FullYearActivityMonths}, map[string]string{
		"cssz_code": "PSSZ",
	}, 0)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
	for _, want := range []string{"kód OSSZ", "jméno", "rodné číslo", "PSČ"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected error to contain %q, got: %v", want, err)
		}
	}
}

func TestSocialInsuranceXML_ValidatesAgainstOSVC25XSD(t *testing.T) {
	xmllint, err := exec.LookPath("xmllint")
	if err != nil {
		t.Skip("xmllint not available; run with libxml2 installed to perform XSD validation")
	}

	sio := &domain.SocialInsuranceOverview{
		Year:                2025,
		FilingType:          domain.FilingTypeRegular,
		TaxBase:             domain.NewAmount(620339, 20),
		AssessmentBase:      domain.NewAmount(341187, 0),
		FinalAssessmentBase: domain.NewAmount(341187, 0),
		TotalInsurance:      domain.NewAmount(99627, 0),
		Prepayments:         domain.NewAmount(86991, 0),
		Difference:          domain.NewAmount(12636, 0),
		NewMonthlyPrepay:    domain.NewAmount(8303, 0),
		ActivityMonths:      domain.FullYearActivityMonths,
	}
	xmlData, err := GenerateSocialInsuranceXML(sio, validSocialInsuranceSettings(), testAdvanceBase(sio))
	if err != nil {
		t.Fatalf("GenerateSocialInsuranceXML: %v", err)
	}

	xmlPath := filepath.Join(t.TempDir(), "osvc25.xml")
	if err := os.WriteFile(xmlPath, xmlData, 0o600); err != nil {
		t.Fatalf("write generated XML: %v", err)
	}
	xsdPath := filepath.Join("..", "..", "docs", "xml-schemas", "cssz", "OSVC25.xsd")
	cmd := exec.Command(xmllint, "--noout", "--schema", xsdPath, xmlPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("xmllint XSD validation failed: %v\n%s", err, out)
	}
}

// The repository holds no OSVC26 schema; the 2026 overview with the minimum
// assessment base is validated against the newest available one (OSVC25).
func TestSocialInsuranceXML_2026MinimumBaseValidatesAgainstOSVC25XSD(t *testing.T) {
	xmllint, err := exec.LookPath("xmllint")
	if err != nil {
		t.Skip("xmllint not available; run with libxml2 installed to perform XSD validation")
	}

	sio := &domain.SocialInsuranceOverview{
		Year:                2026,
		FilingType:          domain.FilingTypeRegular,
		FinalAssessmentBase: domain.NewAmount(205_668, 0),
		TotalInsurance:      domain.NewAmount(60_056, 0),
		Prepayments:         domain.NewAmount(60_000, 0),
		Difference:          domain.NewAmount(56, 0),
		NewMonthlyPrepay:    domain.NewAmount(5_005, 0),
		ActivityMonths:      domain.FullYearActivityMonths,
	}
	xmlData, err := GenerateSocialInsuranceXML(sio, validSocialInsuranceSettings(), testAdvanceBase(sio))
	if err != nil {
		t.Fatalf("GenerateSocialInsuranceXML: %v", err)
	}

	xmlPath := filepath.Join(t.TempDir(), "osvc26.xml")
	if err := os.WriteFile(xmlPath, xmlData, 0o600); err != nil {
		t.Fatalf("write generated XML: %v", err)
	}
	xsdPath := filepath.Join("..", "..", "docs", "xml-schemas", "cssz", "OSVC25.xsd")
	out, err := exec.Command(xmllint, "--noout", "--schema", xsdPath, xmlPath).CombinedOutput()
	if err != nil {
		t.Fatalf("xmllint XSD validation failed: %v\n%s", err, out)
	}
}

func TestFormatMonthlyAverage(t *testing.T) {
	tests := []struct {
		name   string
		amount domain.Amount
		months int64
		want   string
	}{
		{name: "regular year", amount: domain.NewAmount(800000, 0), months: 12, want: "66666.67"},
		{name: "rounded down", amount: domain.NewAmount(280000, 0), months: 12, want: "23333.33"},
		{name: "half year", amount: domain.NewAmount(100000, 0), months: 6, want: "16666.67"},
		{name: "halere base", amount: domain.NewAmount(620339, 20), months: 12, want: "51694.93"},
		{name: "zero base", amount: domain.NewAmount(0, 0), months: 12, want: "0.00"},
		{name: "negative amount", amount: domain.NewAmount(-100, 0), months: 6, want: "-16.67"},
		{name: "no months", amount: domain.NewAmount(100000, 0), months: 0, want: "0.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatMonthlyAverage(tt.amount, tt.months); got != tt.want {
				t.Errorf("formatMonthlyAverage(%d, %d) = %q, want %q", tt.amount, tt.months, got, tt.want)
			}
		})
	}
}

func TestSocialInsuranceXML_MissingActivityMonths(t *testing.T) {
	sio := &domain.SocialInsuranceOverview{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}

	_, err := GenerateSocialInsuranceXML(sio, validSocialInsuranceSettings(), testAdvanceBase(sio))
	if err == nil {
		t.Fatal("expected error for missing activity months, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

// partialYearSocialInsuranceOverview covers July to December only.
func partialYearSocialInsuranceOverview() *domain.SocialInsuranceOverview {
	return &domain.SocialInsuranceOverview{
		Year:                2025,
		FilingType:          domain.FilingTypeRegular,
		ActivityMonths:      domain.ActivityMonths(4032),
		TotalRevenue:        domain.NewAmount(100000, 0),
		TotalExpenses:       domain.NewAmount(0, 0),
		TaxBase:             domain.NewAmount(100000, 0),
		AssessmentBase:      domain.NewAmount(55000, 0),
		MinAssessmentBase:   domain.NewAmount(97770, 0),
		FinalAssessmentBase: domain.NewAmount(97770, 0),
		InsuranceRate:       292,
		TotalInsurance:      domain.NewAmount(28549, 0),
		Prepayments:         domain.NewAmount(0, 0),
		Difference:          domain.NewAmount(28549, 0),
		NewMonthlyPrepay:    domain.NewAmount(4759, 0),
	}
}

func TestSocialInsuranceXML_PartialYearFields(t *testing.T) {
	xmlData, err := GenerateSocialInsuranceXML(partialYearSocialInsuranceOverview(), validSocialInsuranceSettings(), testAdvanceBase(partialYearSocialInsuranceOverview()))
	if err != nil {
		t.Fatalf("GenerateSocialInsuranceXML: %v", err)
	}

	var doc OSVC
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		t.Fatalf("unmarshal generated XML: %v", err)
	}

	flags := doc.Prehled.Client.Hlavc.MonthFlags
	for month, flag := range []string{flags.M1, flags.M2, flags.M3, flags.M4, flags.M5, flags.M6} {
		if flag != "" {
			t.Errorf("hlavc m%d = %q, want empty for a month without activity", month+1, flag)
		}
	}
	for month, flag := range []string{flags.M7, flags.M8, flags.M9, flags.M10, flags.M11, flags.M12} {
		if flag != "A" {
			t.Errorf("hlavc m%d = %q, want A", month+7, flag)
		}
	}
	if flags.M13 != "" {
		t.Errorf("hlavc m13 = %q, want empty for a partial year", flags.M13)
	}

	if doc.Prehled.PVV.Mesc.H != "6" {
		t.Errorf("mesc = %q, want 6", doc.Prehled.PVV.Mesc.H)
	}
	if doc.Prehled.PVV.Mesv.H != "6" {
		t.Errorf("mesv = %q, want 6", doc.Prehled.PVV.Mesv.H)
	}
	if doc.Prehled.PVV.Mesp != "16666.67" {
		t.Errorf("mesp = %q, want 16666.67", doc.Prehled.PVV.Mesp)
	}
	if doc.Prehled.PVV.MVZ != "97770" || doc.Prehled.PVV.UVZ != "97770" || doc.Prehled.PVV.Vzsvc != "97770" {
		t.Errorf("pv bases = mvz:%q uvz:%q vzsvc:%q, want 97770 for all", doc.Prehled.PVV.MVZ, doc.Prehled.PVV.UVZ, doc.Prehled.PVV.Vzsvc)
	}
	if doc.Prehled.PVV.Poj != "28549" {
		t.Errorf("poj = %q, want 28549", doc.Prehled.PVV.Poj)
	}
	if doc.Prehled.Zal.VZ != "16295" {
		t.Errorf("zal@vz = %q, want 16295", doc.Prehled.Zal.VZ)
	}
	if doc.Prehled.Zal.DP != "4759" {
		t.Errorf("zal@dp = %q, want 4759", doc.Prehled.Zal.DP)
	}

	// The stored advance must follow from the monthly base written to ř. 35.
	monthlyBase := int64(16295)
	wantPrepay := (monthlyBase*292 + 999) / 1000
	if doc.Prehled.Zal.DP != strconv.FormatInt(wantPrepay, 10) {
		t.Errorf("zal@dp = %q, want %d from the monthly base", doc.Prehled.Zal.DP, wantPrepay)
	}
}

func TestSocialInsuranceXML_Golden_PartialYear(t *testing.T) {
	xmlData, err := GenerateSocialInsuranceXML(partialYearSocialInsuranceOverview(), validSocialInsuranceSettings(), testAdvanceBase(partialYearSocialInsuranceOverview()))
	if err != nil {
		t.Fatalf("GenerateSocialInsuranceXML: %v", err)
	}

	goldenPath := filepath.Join("testdata", "social_insurance_partial_year.golden.xml")
	testutil.AssertGolden(t, goldenPath, xmlData)
}

func TestSocialInsuranceXML_PartialYearValidatesAgainstOSVC25XSD(t *testing.T) {
	xmllint, err := exec.LookPath("xmllint")
	if err != nil {
		t.Skip("xmllint not available; run with libxml2 installed to perform XSD validation")
	}

	xmlData, err := GenerateSocialInsuranceXML(partialYearSocialInsuranceOverview(), validSocialInsuranceSettings(), testAdvanceBase(partialYearSocialInsuranceOverview()))
	if err != nil {
		t.Fatalf("GenerateSocialInsuranceXML: %v", err)
	}

	xmlPath := filepath.Join(t.TempDir(), "osvc25-partial-year.xml")
	if err := os.WriteFile(xmlPath, xmlData, 0o600); err != nil {
		t.Fatalf("write generated XML: %v", err)
	}
	xsdPath := filepath.Join("..", "..", "docs", "xml-schemas", "cssz", "OSVC25.xsd")
	cmd := exec.Command(xmllint, "--noout", "--schema", xsdPath, xmlPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("xmllint XSD validation failed: %v\n%s", err, out)
	}
}

func validSocialInsuranceSettings() map[string]string {
	return map[string]string{
		"cssz_code":             "775",
		"taxpayer_first_name":   "Jan",
		"taxpayer_last_name":    "Testovaci",
		"taxpayer_birth_number": "8001010006",
		"taxpayer_birth_date":   "1980-01-01",
		"taxpayer_street":       "Testovaci",
		"taxpayer_house_number": "1",
		"taxpayer_postal_code":  "11000",
		"taxpayer_city":         "Praha",
		"taxpayer_email":        "ja@example.test",
		"taxpayer_phone":        "600000000",
	}
}

// testAdvanceBase returns the ř. 35 base the fixtures are calculated with: the
// final annual base spread over the months of activity, rounded up.
func TestSocialInsuranceXML_FilingMethodFlags(t *testing.T) {
	tests := []struct {
		method      domain.FilingMethod
		elektr, por string
	}{
		{domain.FilingMethodUnset, "N", "N"},
		{domain.FilingMethodStandard, "N", "N"},
		{domain.FilingMethodElectronic, "A", "N"},
		{domain.FilingMethodAdvisor, "N", "A"},
	}
	for _, tt := range tests {
		t.Run(string(tt.method), func(t *testing.T) {
			sio := &domain.SocialInsuranceOverview{
				Year:                2025,
				FilingType:          domain.FilingTypeRegular,
				ActivityMonths:      domain.FullYearActivityMonths,
				FinalAssessmentBase: domain.NewAmount(195_540, 0),
				InsuranceRate:       292,
				FilingMethod:        tt.method,
			}
			settings := map[string]string{
				"cssz_code":             "775",
				"taxpayer_first_name":   "Jan",
				"taxpayer_last_name":    "Novak",
				"taxpayer_birth_number": "8001011234",
				"taxpayer_birth_date":   "1980-01-01",
				"taxpayer_street":       "Hlavni",
				"taxpayer_house_number": "42",
				"taxpayer_postal_code":  "11000",
				"taxpayer_city":         "Praha",
			}
			xmlData, err := GenerateSocialInsuranceXML(sio, settings, testAdvanceBase(sio))
			if err != nil {
				t.Fatalf("GenerateSocialInsuranceXML: %v", err)
			}
			var doc OSVC
			if err := xml.Unmarshal(xmlData, &doc); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if doc.Prehled.Prizn.Elektr != tt.elektr || doc.Prehled.Prizn.Por != tt.por {
				t.Errorf("elektr/por = %q/%q, want %q/%q", doc.Prehled.Prizn.Elektr, doc.Prehled.Prizn.Por, tt.elektr, tt.por)
			}
		})
	}
}

func TestSocialInsuranceXML_InvalidFilingMethod(t *testing.T) {
	sio := &domain.SocialInsuranceOverview{
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		ActivityMonths: domain.FullYearActivityMonths,
		FilingMethod:   domain.FilingMethod("paper"),
	}
	_, err := GenerateSocialInsuranceXML(sio, map[string]string{}, 0)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func testAdvanceBase(sio *domain.SocialInsuranceOverview) domain.Amount {
	months := int64(sio.ActivityMonths.Count())
	if months <= 0 {
		return 0
	}
	baseCZK := ToWholeCZK(sio.FinalAssessmentBase)
	return domain.NewAmount((baseCZK+months-1)/months, 0)
}

func TestSocialInsuranceXML_UsesGivenAdvanceMonthlyBase(t *testing.T) {
	xmlData, err := GenerateSocialInsuranceXML(partialYearSocialInsuranceOverview(), validSocialInsuranceSettings(), domain.NewAmount(20000, 0))
	if err != nil {
		t.Fatalf("GenerateSocialInsuranceXML: %v", err)
	}

	var doc OSVC
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		t.Fatalf("unmarshal generated XML: %v", err)
	}
	if doc.Prehled.Zal.VZ != "20000" {
		t.Errorf("zal@vz = %q, want 20000 (the base passed by the caller)", doc.Prehled.Zal.VZ)
	}
}

func TestSocialInsuranceXML_RejectsNegativeAdvanceMonthlyBase(t *testing.T) {
	_, err := GenerateSocialInsuranceXML(partialYearSocialInsuranceOverview(), validSocialInsuranceSettings(), domain.NewAmount(-1, 0))
	if err == nil {
		t.Fatal("expected error for a negative monthly advance base, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}
