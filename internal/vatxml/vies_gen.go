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

// viesFormRegular is the shvies_forma code of a regular summary. A follow-up
// summary (N) corrects earlier rows through storno rows, which are not
// generated, so only the regular form is produced.
const viesFormRegular = "R"

// msgVIESFollowUpUnsupported is the user-facing text for a corrective or
// supplementary summary.
const msgVIESFollowUpUnsupported = "Následné souhrnné hlášení zatím nelze vygenerovat, podejte ho ručně v EPO."

// toCeilCZK converts halere to whole CZK rounded up, as DPHSHV requires for
// pln_hodnota. Integer arithmetic only. A negative total is rounded towards
// zero (-85500.01 CZK gives -85500), which is the ceiling.
func toCeilCZK(a domain.Amount) int64 {
	whole := int64(a) / 100
	if int64(a)%100 > 0 {
		whole++
	}
	return whole
}

// Generate produces the EPO XML for a regular VIES summary. Corrective and
// supplementary summaries are refused. The taxpayer is validated with
// ValidateTaxpayerIdentity.
func (g *VIESSummaryGenerator) Generate(
	vs *domain.VIESSummary,
	lines []domain.VIESSummaryLine,
	info TaxpayerInfo,
) ([]byte, error) {
	if vs == nil {
		return nil, fmt.Errorf("VIES summary is nil: %w", domain.ErrInvalidInput)
	}
	if vs.FilingType == domain.FilingTypeCorrective || vs.FilingType == domain.FilingTypeSupplementary {
		return nil, &domain.UserMessageError{Message: msgVIESFollowUpUnsupported}
	}
	if err := ValidateTaxpayerIdentity(info, TaxpayerDocVIESSummary); err != nil {
		return nil, err
	}

	vetaD := VIESVetaD{
		Dokument:    "SHV",
		KUladis:     "DPH",
		Rok:         vs.Period.Year,
		Ctvrt:       vs.Period.Quarter,
		ShviesForma: viesFormRegular,
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
			PlnHodnota: toCeilCZK(line.TotalAmount),
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
