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

// TaxpayerTypeNaturalPerson is the typ_ds value (F = fyzicka osoba) declared by the
// DPHDP3, DPHKH1 and DPHSHV generators. ZFaktury serves sole proprietors only.
const TaxpayerTypeNaturalPerson = "F"

// TaxpayerCountry is the stat value (EPO "zeme" code list, naz_zeme_c25) of the
// taxpayer's address.
const TaxpayerCountry = "ČESKÁ REPUBLIKA"

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

// FilingTypeCode converts a domain filing type to the control statement khdph_forma code.
// regular -> "B" (radne), corrective -> "O" (opravne), supplementary -> "N" (nasledne).
// EPO XSD restricts khdph_forma to one of [B, O, N, E].
func FilingTypeCode(ft string) string {
	switch ft {
	case domain.FilingTypeCorrective:
		return "O"
	case domain.FilingTypeSupplementary:
		return "N"
	default:
		return "B"
	}
}

// ToWholeCZK converts halere to whole CZK with standard math rounding.
func ToWholeCZK(a domain.Amount) int64 {
	return int64(math.Round(float64(a) / 100.0))
}
