package vatxml

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/zajca/zfaktury/internal/domain"
)

// VIESSummaryGenerator generates EPO XML for VIES summary submissions.
type VIESSummaryGenerator struct{}

// stripCountryPrefix removes the 2-letter country code prefix from a VAT ID.
func stripCountryPrefix(dic string) string {
	dic = strings.TrimSpace(dic)
	if len(dic) > 2 {
		prefix := strings.ToUpper(dic[:2])
		if prefix[0] >= 'A' && prefix[0] <= 'Z' && prefix[1] >= 'A' && prefix[1] <= 'Z' {
			return dic[2:]
		}
	}
	return dic
}

// viesFilingTypeCode maps domain filing type to the shvies_forma code:
// R=regular, N=follow-up (the schema has no separate corrective form).
func viesFilingTypeCode(filingType string) string {
	switch filingType {
	case domain.FilingTypeCorrective, domain.FilingTypeSupplementary:
		return "N"
	default:
		return "R"
	}
}

// Generate produces the EPO XML for a VIES summary. The taxpayer is validated
// with ValidateTaxpayerIdentity.
func (g *VIESSummaryGenerator) Generate(
	vs *domain.VIESSummary,
	lines []domain.VIESSummaryLine,
	info TaxpayerInfo,
) ([]byte, error) {
	if vs == nil {
		return nil, fmt.Errorf("VIES summary is nil: %w", domain.ErrInvalidInput)
	}
	if err := ValidateTaxpayerIdentity(info, TaxpayerDocVIESSummary); err != nil {
		return nil, err
	}

	vetaD := VIESVetaD{
		Dokument:    "SHV",
		KUladis:     "DPH",
		Rok:         vs.Period.Year,
		Ctvrt:       vs.Period.Quarter,
		ShviesForma: viesFilingTypeCode(vs.FilingType),
	}

	vetaP := VIESVetaP{
		CUfo:     info.UFOCode,
		CPracufo: info.PracUFO,
		DIC:      info.DIC,
		TypDS:    TaxpayerTypeNaturalPerson,
		Jmeno:    info.FirstName,
		Prijmeni: info.LastName,
		Ulice:    info.Street,
		CPop:     info.HouseNum,
		NazObce:  info.City,
		PSC:      info.ZIP,
		Stat:     TaxpayerCountry,
	}

	var vetaR []VIESVetaR
	for _, line := range lines {
		vetaR = append(vetaR, VIESVetaR{
			KStat:      line.CountryCode,
			CVat:       stripCountryPrefix(line.PartnerDIC),
			KPlnEU:     line.ServiceCode,
			PlnHodnota: ToWholeCZK(line.TotalAmount),
		})
	}

	pisemnost := VIESPisemnost{
		DPHSHV: DPHSHV{
			VetaD: vetaD,
			VetaP: vetaP,
			VetaR: vetaR,
		},
	}

	output, err := xml.MarshalIndent(pisemnost, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling VIES XML: %w", err)
	}

	xmlDecl := []byte(xml.Header)
	xmlDecl = append(xmlDecl, output...)
	return xmlDecl, nil
}
