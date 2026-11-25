package vatxml

import (
	"encoding/xml"
	"fmt"
	"sort"
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

// viesForm codes of shvies_forma: R is a regular summary, N a follow-up
// (corrective or supplementary) one that corrects earlier rows through storno
// rows.
const (
	viesFormRegular  = "R"
	viesFormFollowUp = "N"
)

// kStornoAttr marks a VetaR row as a storno row (k_storno="A").
const kStornoAttr = "A"

// msgVIESFollowUpNoChange is the user-facing text for a follow-up summary that
// changes nothing against the filed baseline.
const msgVIESFollowUpNoChange = "Následné souhrnné hlášení neobsahuje žádnou změnu oproti podanému hlášení za toto čtvrtletí."

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

// viesPartnerKey identifies a VetaR row the way the tax office matches storno
// rows: country code + partner VAT ID without the country prefix + supply code.
func viesPartnerKey(countryCode, partnerDIC, serviceCode string) string {
	return countryCode + "\x00" + stripCountryPrefix(partnerDIC) + "\x00" + serviceCode
}

// realVIESRow builds a regular (non-storno) VetaR line. pln_hodnota is always
// written, even for a zero value.
func realVIESRow(line domain.VIESSummaryLine) VIESVetaR {
	value := toCeilCZK(line.TotalAmount)
	return VIESVetaR{
		KStat:      line.CountryCode,
		CVat:       stripCountryPrefix(line.PartnerDIC),
		KPlnEU:     line.ServiceCode,
		PlnHodnota: &value,
	}
}

// stornoVIESRow builds the storno row that cancels the baseline row of the same
// partner. A storno row carries no pln_hodnota.
func stornoVIESRow(line domain.VIESSummaryLine) VIESVetaR {
	return VIESVetaR{
		KStorno: kStornoAttr,
		KStat:   line.CountryCode,
		CVat:    stripCountryPrefix(line.PartnerDIC),
		KPlnEU:  line.ServiceCode,
	}
}

// followUpRows diffs the new lines against the baseline lines, comparing values
// after toCeilCZK (whole CZK, as filed), and returns the VetaR rows of a
// follow-up summary: a storno row plus a new row for every changed partner, a
// storno row for every removed partner, a new row for every added partner;
// unchanged partners are left out. Rows are sorted by partner key, the storno
// row before its new row. A summary without any change is refused.
func followUpRows(lines, baseline []domain.VIESSummaryLine) ([]VIESVetaR, error) {
	type entry struct {
		key    string
		storno bool
		row    VIESVetaR
	}

	baselineValue := make(map[string]int64, len(baseline))
	for _, line := range baseline {
		baselineValue[viesPartnerKey(line.CountryCode, line.PartnerDIC, line.ServiceCode)] = toCeilCZK(line.TotalAmount)
	}
	newValue := make(map[string]int64, len(lines))
	for _, line := range lines {
		newValue[viesPartnerKey(line.CountryCode, line.PartnerDIC, line.ServiceCode)] = toCeilCZK(line.TotalAmount)
	}

	var entries []entry
	for _, line := range lines {
		key := viesPartnerKey(line.CountryCode, line.PartnerDIC, line.ServiceCode)
		old, wasFiled := baselineValue[key]
		if wasFiled && old == newValue[key] {
			continue // unchanged
		}
		if wasFiled {
			entries = append(entries, entry{key: key, storno: true, row: stornoVIESRow(line)})
		}
		entries = append(entries, entry{key: key, row: realVIESRow(line)})
	}
	for _, line := range baseline {
		key := viesPartnerKey(line.CountryCode, line.PartnerDIC, line.ServiceCode)
		if _, ok := newValue[key]; !ok {
			entries = append(entries, entry{key: key, storno: true, row: stornoVIESRow(line)})
		}
	}

	if len(entries) == 0 {
		return nil, &domain.UserMessageError{Message: msgVIESFollowUpNoChange}
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].key != entries[j].key {
			return entries[i].key < entries[j].key
		}
		return entries[i].storno && !entries[j].storno
	})

	rows := make([]VIESVetaR, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, e.row)
	}
	return rows, nil
}

// Generate produces the EPO XML for a VIES summary. A regular summary passes a
// nil baseline and lists all lines with shvies_forma="R". A corrective or
// supplementary summary (shvies_forma="N") takes the baseline lines of the last
// filed summary of the quarter (nil means none were filed) and emits storno
// rows and new rows for the changed, removed and added partners only. The
// taxpayer is validated with ValidateTaxpayerIdentity.
func (g *VIESSummaryGenerator) Generate(
	vs *domain.VIESSummary,
	lines []domain.VIESSummaryLine,
	baseline []domain.VIESSummaryLine,
	info TaxpayerInfo,
) ([]byte, error) {
	if vs == nil {
		return nil, fmt.Errorf("VIES summary is nil: %w", domain.ErrInvalidInput)
	}
	if err := ValidateTaxpayerIdentity(info, TaxpayerDocVIESSummary); err != nil {
		return nil, err
	}

	followUp := vs.FilingType == domain.FilingTypeCorrective || vs.FilingType == domain.FilingTypeSupplementary
	if !followUp && baseline != nil {
		return nil, fmt.Errorf("generating a regular VIES summary with baseline lines: %w", domain.ErrInvalidInput)
	}

	forma := viesFormRegular
	var vetaR []VIESVetaR
	if followUp {
		forma = viesFormFollowUp
		rows, err := followUpRows(lines, baseline)
		if err != nil {
			return nil, err
		}
		vetaR = rows
	} else {
		for _, line := range lines {
			vetaR = append(vetaR, realVIESRow(line))
		}
	}

	vetaD := VIESVetaD{
		Dokument:    "SHV",
		KUladis:     "DPH",
		DPoddp:      submissionDate(info).Format("02.01.2006"),
		Rok:         vs.Period.Year,
		Ctvrt:       vs.Period.Quarter,
		ShviesForma: forma,
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
