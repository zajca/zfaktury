package vatxml

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/zajca/zfaktury/internal/domain"
)

var (
	digitsOKEC    = regexp.MustCompile(`^\d{5}$`)
	digitsPSC     = regexp.MustCompile(`^\d{5}$`)
	digitsDIC     = regexp.MustCompile(`^\d{8,10}$`)
	digitsUFO     = regexp.MustCompile(`^\d{3}$`)
	digitsPracUFO = regexp.MustCompile(`^\d{4}$`)
)

// ValidateTaxpayerInfo verifies that all fields required by the EPO DPHDP3 schema
// are present and well-formed. Returns ErrInvalidInput with a list of human-readable
// problems (in Czech, since this surfaces directly to the user) on failure.
func ValidateTaxpayerInfo(info TaxpayerInfo) error {
	var problems []string

	required := []struct {
		label string
		value string
	}{
		{"DIČ", info.DIC},
		{"Jméno", info.FirstName},
		{"Příjmení", info.LastName},
		{"Ulice (nebo část obce)", info.Street},
		{"Obec", info.City},
		{"PSČ", info.ZIP},
		{"Číslo popisné", info.HouseNum},
		{"Kód FÚ (c_ufo)", info.UFOCode},
		{"Kód pracoviště FÚ (c_pracufo)", info.PracUFO},
		{"NACE kód hlavní činnosti (c_okec)", info.OKEC},
	}
	for _, r := range required {
		if strings.TrimSpace(r.value) == "" {
			problems = append(problems, fmt.Sprintf("chybí %s", r.label))
		}
	}

	if info.DIC != "" && !digitsDIC.MatchString(info.DIC) {
		problems = append(problems, "DIČ musí být 8-10 číslic (bez prefixu CZ)")
	}
	if info.OKEC != "" && !digitsOKEC.MatchString(info.OKEC) {
		problems = append(problems, "NACE kód musí být přesně 5 číslic (např. 62010)")
	}
	if info.ZIP != "" && !digitsPSC.MatchString(strings.ReplaceAll(info.ZIP, " ", "")) {
		problems = append(problems, "PSČ musí být 5 číslic")
	}
	if info.UFOCode != "" && !digitsUFO.MatchString(info.UFOCode) {
		problems = append(problems, "Kód FÚ (c_ufo) musí být 3 číslice")
	}
	if info.PracUFO != "" && !digitsPracUFO.MatchString(info.PracUFO) {
		problems = append(problems, "Kód pracoviště FÚ (c_pracufo) musí být 4 číslice")
	}

	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("nelze vygenerovat XML přiznání k DPH: %s: %w", strings.Join(problems, "; "), domain.ErrInvalidInput)
}
