package domain

import (
	"errors"
	"testing"
)

func TestFilingMethodValidate(t *testing.T) {
	valid := []FilingMethod{FilingMethodUnset, FilingMethodStandard, FilingMethodElectronic, FilingMethodAdvisor}
	for _, m := range valid {
		if err := m.Validate(); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", m, err)
		}
	}

	for _, m := range []FilingMethod{"paper", "Electronic"} {
		if err := m.Validate(); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("Validate(%q) = %v, want ErrInvalidInput", m, err)
		}
	}
}
