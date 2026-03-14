package vatxml

import (
	"math"

	"github.com/zajca/zfaktury/internal/domain"
)

// DPHFilingTypeCode converts a domain filing type to the DPHDP3 dapdph_forma code.
// regular -> "B" (radne), corrective -> "O" (opravne), supplementary -> "D" (dodatecne).
func DPHFilingTypeCode(ft string) string {
	switch ft {
	case domain.FilingTypeCorrective:
		return "O"
	case domain.FilingTypeSupplementary:
		return "D"
	default:
		return "B"
	}
}

// FilingTypeCode converts a domain filing type to the control statement d_typ code.
// regular -> "R", corrective -> "N", supplementary -> "O".
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

// ToWholeCZK converts halere to whole CZK with standard math rounding.
func ToWholeCZK(a domain.Amount) int64 {
	return int64(math.Round(float64(a) / 100.0))
}
