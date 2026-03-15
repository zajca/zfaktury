package annualtaxxml

import (
	"errors"
	"path/filepath"
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
		AssessmentBase:      domain.NewAmount(140000, 0),
		MinAssessmentBase:   domain.NewAmount(131736, 0),
		FinalAssessmentBase: domain.NewAmount(140000, 0),
		InsuranceRate:       292,
		TotalInsurance:      domain.NewAmount(40880, 0),
		Prepayments:         domain.NewAmount(38088, 0),
		Difference:          domain.NewAmount(2792, 0),
		NewMonthlyPrepay:    domain.NewAmount(3407, 0),
	}

	settings := map[string]string{
		"cssz_code":             "PSSZ",
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
		AssessmentBase:      domain.NewAmount(140000, 0),
		MinAssessmentBase:   domain.NewAmount(131736, 0),
		FinalAssessmentBase: domain.NewAmount(140000, 0),
		InsuranceRate:       292,
		TotalInsurance:      domain.NewAmount(40880, 0),
		Prepayments:         domain.NewAmount(38088, 0),
		Difference:          domain.NewAmount(2792, 0),
		NewMonthlyPrepay:    domain.NewAmount(3407, 0),
	}

	settings := map[string]string{
		"cssz_code":             "PSSZ",
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
		AssessmentBase:      domain.NewAmount(400000, 0),  // 50% of 800 000
		MinAssessmentBase:   domain.NewAmount(131736, 0),  // minimum for 2025
		FinalAssessmentBase: domain.NewAmount(400000, 0),  // max(400000, 131736)
		InsuranceRate:       292,                          // 29.2%
		TotalInsurance:      domain.NewAmount(116800, 0),  // 400000 * 0.292
		Prepayments:         domain.NewAmount(38088, 0),   // 3174/month * 12
		Difference:          domain.NewAmount(78712, 0),   // 116800 - 38088
		NewMonthlyPrepay:    domain.NewAmount(9734, 0),    // new monthly prepayment
	}

	settings := map[string]string{
		"cssz_code":             "PSSZ",
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
		AssessmentBase:      domain.NewAmount(140000, 0), // 50% of 280 000
		MinAssessmentBase:   domain.NewAmount(131736, 0), // minimum for 2025
		FinalAssessmentBase: domain.NewAmount(140000, 0), // max(140000, 131736)
		InsuranceRate:       292,                         // 29.2%
		TotalInsurance:      domain.NewAmount(40880, 0),  // 140000 * 0.292
		Prepayments:         domain.NewAmount(38088, 0),  // 3174/month * 12
		Difference:          domain.NewAmount(2792, 0),   // 40880 - 38088
		NewMonthlyPrepay:    domain.NewAmount(3407, 0),   // new monthly prepayment
	}

	settings := map[string]string{
		"cssz_code":             "OSSZ Brno-venkov",
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
