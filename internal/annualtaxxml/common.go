package annualtaxxml

import "github.com/zajca/zfaktury/internal/domain"

// ToWholeCZK converts halere to whole CZK (integer division, truncated toward zero).
func ToWholeCZK(a domain.Amount) int64 {
	return int64(a) / 100
}

// DPFOFilingTypeCode converts a domain filing type to the DPFO XML code.
// B = regular, O = corrective, D = supplementary.
func DPFOFilingTypeCode(filingType string) string {
	switch filingType {
	case domain.FilingTypeCorrective:
		return "O"
	case domain.FilingTypeSupplementary:
		return "D"
	default:
		return "B"
	}
}

// CSSZFilingTypeCode converts a domain filing type to the CSSZ (social security) XML code.
// N = regular (nova), O = corrective (opravna), Z = supplementary (zmena).
func CSSZFilingTypeCode(filingType string) string {
	switch filingType {
	case domain.FilingTypeCorrective:
		return "O"
	case domain.FilingTypeSupplementary:
		return "Z"
	default:
		return "N"
	}
}
