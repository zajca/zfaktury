package calc

import "github.com/zajca/zfaktury/internal/taxrules"

// TaxYearConstants is kept as a compatibility alias while the canonical
// year-specific values live in internal/taxrules.RuleSet.
type TaxYearConstants = taxrules.TaxYearConstants

// GetTaxConstants returns the tax constants for a given year.
func GetTaxConstants(year int) (TaxYearConstants, error) {
	rules, err := taxrules.GetRuleSet(year)
	if err != nil {
		return TaxYearConstants{}, err
	}
	return rules.Constants, nil
}
