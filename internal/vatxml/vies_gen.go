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

// viesFilingTypeCode maps domain filing type to VIES EPO codes.
// VIES uses different codes: B=regular, O=corrective, N=supplementary.
func viesFilingTypeCode(filingType string) string {
	switch filingType {
	case domain.FilingTypeCorrective:
		return "O"
	case domain.FilingTypeSupplementary:
		return "N"
	default:
		return "B"
	}
}

// Generate produces the EPO XML for a VIES summary.
func (g *VIESSummaryGenerator) Generate(
	vs *domain.VIESSummary,
	lines []domain.VIESSummaryLine,
	dic string,
) ([]byte, error) {
	if vs == nil {
		return nil, fmt.Errorf("VIES summary is nil")
	}

	filerDIC := stripCountryPrefix(dic)

	vetaD := VIESVetaD{
		KDaph:  viesFilingTypeCode(vs.FilingType),
		Rok:    vs.Period.Year,
		Ctvrt:  vs.Period.Quarter,
		DICOdb: filerDIC,
	}

	var vetaP []VIESVetaP
	for _, line := range lines {
		vetaP = append(vetaP, VIESVetaP{
			KStat:   line.CountryCode,
			DICOdbe: stripCountryPrefix(line.PartnerDIC),
			KPlneni: line.ServiceCode,
			Obrat:   ToWholeCZK(line.TotalAmount),
		})
	}

	pisemnost := VIESPisemnost{
		Xmlns: "http://adis.mfcr.cz/rozhranni/",
		DPHSHV: DPHSHV{
			VetaD: vetaD,
			VetaP: vetaP,
		},
	}

	output, err := xml.MarshalIndent(pisemnost, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling VIES XML: %w", err)
	}

	xmlDecl := []byte(xml.Header)
	result := append(xmlDecl, output...)
	return result, nil
}
