package vatxml

import "github.com/zajca/zfaktury/internal/domain"

// FilingTypeCode converts a domain filing type to the EPO XML code.
func FilingTypeCode(ft string) string {
	switch ft {
	case domain.FilingTypeCorrective:
		return "N"
	case domain.FilingTypeSupplementary:
		return "O"
	default:
		return "R"
	}
}

// ToWholeCZK converts halere to whole CZK (integer division, truncated toward zero).
func ToWholeCZK(a domain.Amount) int64 {
	return int64(a) / 100
}
