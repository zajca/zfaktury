package vatxml

import (
	"encoding/xml"
	"math"
	"strconv"

	"github.com/zajca/zfaktury/internal/domain"
)

// XMLFloat is a float64 that always marshals with one decimal place in XML attributes
// (e.g. 125598 -> "125598.0") to match the EPO/Fakturoid format.
type XMLFloat float64

func (f XMLFloat) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	s := strconv.FormatFloat(float64(f), 'f', 1, 64)
	return xml.Attr{Name: name, Value: s}, nil
}

func (f *XMLFloat) UnmarshalXMLAttr(attr xml.Attr) error {
	val, err := strconv.ParseFloat(attr.Value, 64)
	if err != nil {
		return err
	}
	*f = XMLFloat(val)
	return nil
}

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
