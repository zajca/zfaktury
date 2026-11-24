package annualtaxxml

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func secondaryActivitySettings() map[string]string {
	return map[string]string{
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
}

func assertValidatesAgainstOSVC25(t *testing.T, xmlData []byte) {
	t.Helper()
	xmllint, err := exec.LookPath("xmllint")
	if err != nil {
		t.Skip("xmllint not available; run with libxml2 installed to perform XSD validation")
	}
	xmlPath := filepath.Join(t.TempDir(), "osvc25.xml")
	if err := os.WriteFile(xmlPath, xmlData, 0o600); err != nil {
		t.Fatalf("write generated XML: %v", err)
	}
	xsdPath := filepath.Join("..", "..", "docs", "xml-schemas", "cssz", "OSVC25.xsd")
	if out, err := exec.Command(xmllint, "--noout", "--schema", xsdPath, xmlPath).CombinedOutput(); err != nil {
		t.Fatalf("xmllint XSD validation failed: %v\n%s", err, out)
	}
}

func months(t *testing.T, m ...int) domain.ActivityMonths {
	t.Helper()
	mask, err := domain.NewActivityMonths(m)
	if err != nil {
		t.Fatalf("NewActivityMonths(%v) error: %v", m, err)
	}
	return mask
}

// mixedYearOverview is a year with main activity January-June and secondary
// activity July-December (employment) and a tax base of 240 000 CZK.
func mixedYearOverview(t *testing.T) *domain.SocialInsuranceOverview {
	return &domain.SocialInsuranceOverview{
		Year:                    2025,
		FilingType:              domain.FilingTypeRegular,
		ActivityMonths:          domain.FullYearActivityMonths,
		SecondaryMonths:         months(t, 7, 8, 9, 10, 11, 12),
		SecondaryReason:         domain.SecondaryReasonEmployment,
		FilingMethod:            domain.FilingMethodStandard,
		TotalRevenue:            domain.NewAmount(240_000, 0),
		TaxBase:                 domain.NewAmount(240_000, 0),
		AssessmentBase:          domain.NewAmount(66_000, 0),
		MainTaxBaseShare:        domain.NewAmount(120_000, 0),
		SecondaryTaxBaseShare:   domain.NewAmount(120_000, 0),
		SecondaryAssessmentBase: domain.NewAmount(66_000, 0),
		MainPartialBase:         domain.NewAmount(97_770, 0),
		SecondaryPartialBase:    domain.NewAmount(30_732, 0),
		MinAssessmentBase:       domain.NewAmount(128_502, 0),
		FinalAssessmentBase:     domain.NewAmount(163_770, 0),
		SumBase:                 domain.NewAmount(163_770, 0),
		InsuredBase:             domain.NewAmount(163_770, 0),
		InsuranceRate:           292,
		TotalInsurance:          domain.NewAmount(47_821, 0),
		Prepayments:             domain.NewAmount(0, 0),
		Difference:              domain.NewAmount(47_821, 0),
		NewMonthlyPrepay:        domain.NewAmount(3_212, 0),
	}
}

func TestSocialInsuranceXML_Golden_MixedYear(t *testing.T) {
	sio := mixedYearOverview(t)
	xmlData, err := GenerateSocialInsuranceXML(sio, secondaryActivitySettings(), domain.NewAmount(11_000, 0))
	if err != nil {
		t.Fatalf("GenerateSocialInsuranceXML: %v", err)
	}
	testutil.AssertGolden(t, filepath.Join("testdata", "social_insurance_mixed.golden.xml"), xmlData)
	assertValidatesAgainstOSVC25(t, xmlData)
}

func TestSocialInsuranceXML_Golden_SecondaryOnly(t *testing.T) {
	sio := &domain.SocialInsuranceOverview{
		Year:            2025,
		FilingType:      domain.FilingTypeRegular,
		ActivityMonths:  domain.FullYearActivityMonths,
		SecondaryMonths: domain.FullYearActivityMonths,
		SecondaryReason: domain.SecondaryReasonStudent,
		FilingMethod:    domain.FilingMethodStandard,
		TotalRevenue:    domain.NewAmount(100_000, 0),
		TaxBase:         domain.NewAmount(100_000, 0),
		InsuranceRate:   292,
	}
	xmlData, err := GenerateSocialInsuranceXML(sio, secondaryActivitySettings(), 0)
	if err != nil {
		t.Fatalf("GenerateSocialInsuranceXML: %v", err)
	}
	testutil.AssertGolden(t, filepath.Join("testdata", "social_insurance_secondary_only.golden.xml"), xmlData)
	assertValidatesAgainstOSVC25(t, xmlData)
}

func TestSocialInsuranceXML_Golden_EmploymentBase(t *testing.T) {
	reported := domain.NewAmount(500_000, 0)
	sio := &domain.SocialInsuranceOverview{
		Year:                     2025,
		FilingType:               domain.FilingTypeRegular,
		ActivityMonths:           domain.FullYearActivityMonths,
		FilingMethod:             domain.FilingMethodStandard,
		TotalRevenue:             domain.NewAmount(4_000_000, 0),
		TaxBase:                  domain.NewAmount(4_000_000, 0),
		AssessmentBase:           domain.NewAmount(2_200_000, 0),
		MinAssessmentBase:        domain.NewAmount(195_540, 0),
		FinalAssessmentBase:      domain.NewAmount(2_200_000, 0),
		EmploymentAssessmentBase: &reported,
		ReportedEmploymentBase:   &reported,
		SumBase:                  domain.NewAmount(2_700_000, 0),
		InsuredBase:              domain.NewAmount(1_734_736, 0),
		InsuranceRate:            292,
		TotalInsurance:           domain.NewAmount(506_543, 0),
		Difference:               domain.NewAmount(506_543, 0),
		NewMonthlyPrepay:         domain.NewAmount(53_534, 0),
	}
	xmlData, err := GenerateSocialInsuranceXML(sio, secondaryActivitySettings(), domain.NewAmount(183_334, 0))
	if err != nil {
		t.Fatalf("GenerateSocialInsuranceXML: %v", err)
	}
	testutil.AssertGolden(t, filepath.Join("testdata", "social_insurance_employment.golden.xml"), xmlData)
	assertValidatesAgainstOSVC25(t, xmlData)
}

func TestSocialInsuranceXML_SecondaryFields(t *testing.T) {
	tests := []struct {
		name      string
		secondary domain.ActivityMonths
		reason    domain.SecondaryActivityReason
		want      []string
	}{
		{"mixed year", months(t, 7, 8, 9, 10, 11, 12), domain.SecondaryReasonEmployment, []string{
			`<druc>S</druc>`, `<zam>A</zam>`, `<duchod>N</duchod>`, `<ndite>N</ndite>`,
			`<mesc h="6" v="6">`, `<zal ved="V"`, `<rdza h="120000.00" v="120000.00">`, `<dvz h="97770" v="30732">`,
		}},
		{"pension reason", months(t, 7, 8, 9, 10, 11, 12), domain.SecondaryReasonPension, []string{`<duchod>A</duchod>`, `<zam>N</zam>`}},
		{"child care reason", months(t, 7, 8, 9, 10, 11, 12), domain.SecondaryReasonChildCare, []string{`<pdite>A</pdite>`}},
		{"maternity reason", months(t, 7, 8, 9, 10, 11, 12), domain.SecondaryReasonMaternity, []string{`<ppm>A</ppm>`}},
		{"care reason", months(t, 7, 8, 9, 10, 11, 12), domain.SecondaryReasonCare, []string{`<pece>A</pece>`}},
		{"student reason", months(t, 7, 8, 9, 10, 11, 12), domain.SecondaryReasonStudent, []string{`<ndite>A</ndite>`}},
		{"secondary first half keeps advance kind H", months(t, 1, 2, 3, 4, 5, 6), domain.SecondaryReasonEmployment, []string{`<zal ved="H"`, `<m1>A</m1>`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sio := mixedYearOverview(t)
			sio.SecondaryMonths = tt.secondary
			sio.SecondaryReason = tt.reason
			xmlData, err := GenerateSocialInsuranceXML(sio, secondaryActivitySettings(), domain.NewAmount(11_000, 0))
			if err != nil {
				t.Fatalf("GenerateSocialInsuranceXML: %v", err)
			}
			for _, want := range tt.want {
				if !strings.Contains(string(xmlData), want) {
					t.Errorf("XML does not contain %q", want)
				}
			}
			assertValidatesAgainstOSVC25(t, xmlData)
		})
	}
}

func TestSocialInsuranceXML_MainOnlyKeepsEmptySecondaryElement(t *testing.T) {
	sio := mixedYearOverview(t)
	sio.SecondaryMonths = 0
	sio.SecondaryReason = domain.SecondaryReasonNone
	xmlData, err := GenerateSocialInsuranceXML(sio, secondaryActivitySettings(), domain.NewAmount(11_000, 0))
	if err != nil {
		t.Fatalf("GenerateSocialInsuranceXML: %v", err)
	}
	for _, want := range []string{`<druc>H</druc>`, `<zal ved="H"`, `<rdza h="0" v="0">`, `<vzza></vzza>`, `<zam></zam>`} {
		if !strings.Contains(string(xmlData), want) {
			t.Errorf("XML does not contain %q", want)
		}
	}
	if strings.Contains(string(xmlData), `<zam>A</zam>`) || strings.Contains(string(xmlData), `<zam>N</zam>`) {
		t.Error("XML contains a secondary-activity reason for a main-only year")
	}
}

func TestSocialInsuranceXML_RejectsSecondaryWithoutReason(t *testing.T) {
	sio := mixedYearOverview(t)
	sio.SecondaryReason = domain.SecondaryReasonNone
	if _, err := GenerateSocialInsuranceXML(sio, secondaryActivitySettings(), domain.NewAmount(11_000, 0)); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("error = %v, want ErrInvalidInput", err)
	}
}
