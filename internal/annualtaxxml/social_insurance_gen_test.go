package annualtaxxml

import (
	"encoding/xml"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestSocialInsuranceXML_NilInput(t *testing.T) {
	_, err := GenerateSocialInsuranceXML(nil, map[string]string{})
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

	xmlData, err := GenerateSocialInsuranceXML(sio, settings)
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

	xmlData, err := GenerateSocialInsuranceXML(sio, settings)
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

	xmlData, err := GenerateSocialInsuranceXML(sio, settings)
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

	xmlData, err := GenerateSocialInsuranceXML(sio, settings)
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
	}
	settings := validSocialInsuranceSettings()
	settings["cssz_variable_symbol"] = "75660248"
	settings["taxpayer_title"] = "Ing."
	settings["taxpayer_databox_id"] = "7rgmm99"
	settings["sickness_insurance_monthly"] = "768"

	xmlData, err := GenerateSocialInsuranceXML(sio, settings)
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
	_, err := GenerateSocialInsuranceXML(&domain.SocialInsuranceOverview{Year: 2025}, map[string]string{
		"cssz_code": "PSSZ",
	})
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
	}
	xmlData, err := GenerateSocialInsuranceXML(sio, validSocialInsuranceSettings())
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

func validSocialInsuranceSettings() map[string]string {
	return map[string]string{
		"cssz_code":             "775",
		"taxpayer_first_name":   "Martin",
		"taxpayer_last_name":    "Zajic",
		"taxpayer_birth_number": "8905244997",
		"taxpayer_birth_date":   "1989-05-24",
		"taxpayer_street":       "Ludkovice",
		"taxpayer_house_number": "189",
		"taxpayer_postal_code":  "76341",
		"taxpayer_city":         "Ludkovice",
		"taxpayer_email":        "ja@example.test",
		"taxpayer_phone":        "776598983",
	}
}
