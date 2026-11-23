package taxrules

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zajca/zfaktury/internal/domain"
)

// RuleStatus describes how trustworthy a rule set is for final filing.
type RuleStatus string

const (
	RuleStatusFinal       RuleStatus = "final"
	RuleStatusProvisional RuleStatus = "provisional"
)

// LegalSource identifies the source used for a rule-set value.
type LegalSource struct {
	Title string `json:"title"`
	URL   string `json:"url,omitempty"`
	Note  string `json:"note,omitempty"`
}

// TaxYearConstants holds tax computation constants for a specific year.
type TaxYearConstants struct {
	ProgressiveThreshold   domain.Amount         // prah pro 23% sazbu (in halere)
	BasicCredit            domain.Amount         // sleva na poplatnika
	SpouseCredit           domain.Amount         // sleva na manzela/ku
	StudentCredit          domain.Amount         // student
	DisabilityCredit1      domain.Amount         // invalidita 1. a 2. stupen
	DisabilityCredit3      domain.Amount         // invalidita 3. stupen
	DisabilityZTPP         domain.Amount         // drzitel prukazu ZTP/P
	ChildBenefit1          domain.Amount         // 1. dite
	ChildBenefit2          domain.Amount         // 2. dite
	ChildBenefit3Plus      domain.Amount         // 3+ dite
	ChildBenefitZTP        domain.Amount         // ZTP prirazka (double)
	SocialMinMonthly       domain.Amount         // min mesicni vym. zaklad CSSZ pro rocni Prehled (platny pro vsechny mesice roku)
	SocialRate             int                   // permille*10, e.g. 292 = 29.2%
	SocialMaxAnnualBase    domain.Amount         // max rocni vym. zaklad CSSZ (48nasobek prumerne mzdy)
	HealthMinMonthly       domain.Amount         // min mesicni vym. zaklad ZP
	HealthRate             int                   // permille*10, e.g. 135 = 13.5%
	FlatRateCaps           map[int]domain.Amount // percent -> max halere amount
	TimeTestYears          int                   // years to hold for time test exemption
	SecurityExemptionLimit domain.Amount         // max yearly gross time-test income that stays exempt (0 = no limit)

	SpouseIncomeLimit           domain.Amount
	DeductionCapMortgage        domain.Amount
	DeductionCapLifeInsurance   domain.Amount
	DeductionCapPension         domain.Amount
	DeductionCapSavingsCombined domain.Amount
	DeductionCapUnionDues       domain.Amount

	// SecurityExemptionLimitAssetTypes lists the asset types whose time-test
	// income counts toward SecurityExemptionLimit; empty means all asset types.
	SecurityExemptionLimitAssetTypes []string
}

// RuleSet is the authoritative set of rules for one tax year and tax domain.
type RuleSet struct {
	ID        string           `json:"id"`
	Year      int              `json:"year"`
	Status    RuleStatus       `json:"status"`
	Sources   []LegalSource    `json:"sources"`
	Constants TaxYearConstants `json:"constants"`
	Forms     FormRules        `json:"forms"`
}

// FormRules carries government form/schema metadata that is version-sensitive.
type FormRules struct {
	IncomeTaxXML IncomeTaxXMLFormRules `json:"income_tax_xml"`
}

// IncomeTaxXMLFormRules identifies the XML form version for DPFO generation.
type IncomeTaxXMLFormRules struct {
	FormCode      string `json:"form_code"`
	SchemaFile    string `json:"schema_file"`
	ValidFromYear int    `json:"valid_from_year"`
	ValidToYear   int    `json:"valid_to_year"`
}

// Fingerprint returns a deterministic hash of the rule-set data that affects
// calculations. It is stored with calculated tax returns so old calculations
// can be reproduced and audited. Sources are citation metadata and are not
// hashed, so fixing a title or URL does not change the fingerprint.
func (r RuleSet) Fingerprint() (string, error) {
	r.Sources = nil
	b, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("marshal tax rule set %s: %w", r.ID, err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// ValidateRuleSet checks structural invariants that would make a rule set
// unsafe for calculation if broken.
func ValidateRuleSet(r RuleSet) error {
	var errs []error
	if r.ID == "" {
		errs = append(errs, fmt.Errorf("missing rule-set id: %w", domain.ErrInvalidInput))
	}
	if r.Year < 2000 || r.Year > 2100 {
		errs = append(errs, fmt.Errorf("year %d out of supported range: %w", r.Year, domain.ErrInvalidInput))
	}
	if r.Status != RuleStatusFinal && r.Status != RuleStatusProvisional {
		errs = append(errs, fmt.Errorf("unsupported rule-set status %q: %w", r.Status, domain.ErrInvalidInput))
	}
	if r.Constants.ProgressiveThreshold <= 0 {
		errs = append(errs, fmt.Errorf("%s progressive threshold must be positive: %w", r.ID, domain.ErrInvalidInput))
	}
	if r.Constants.BasicCredit <= 0 {
		errs = append(errs, fmt.Errorf("%s basic credit must be positive: %w", r.ID, domain.ErrInvalidInput))
	}
	if r.Constants.SocialRate <= 0 || r.Constants.HealthRate <= 0 {
		errs = append(errs, fmt.Errorf("%s insurance rates must be positive: %w", r.ID, domain.ErrInvalidInput))
	}
	if r.Constants.SocialMaxAnnualBase <= 0 {
		errs = append(errs, fmt.Errorf("%s social max annual base must be positive: %w", r.ID, domain.ErrInvalidInput))
	}
	if r.Constants.SocialMaxAnnualBase < r.Constants.SocialMinMonthly*12 {
		errs = append(errs, fmt.Errorf("%s social max annual base %d is below the annual minimum base %d: %w", r.ID, r.Constants.SocialMaxAnnualBase, r.Constants.SocialMinMonthly*12, domain.ErrInvalidInput))
	}
	if len(r.Constants.FlatRateCaps) == 0 {
		errs = append(errs, fmt.Errorf("%s flat-rate caps are empty: %w", r.ID, domain.ErrInvalidInput))
	}
	if r.Forms.IncomeTaxXML.FormCode == "" {
		errs = append(errs, fmt.Errorf("%s income-tax XML form code is empty: %w", r.ID, domain.ErrInvalidInput))
	}
	if r.Status == RuleStatusFinal && (r.Year < r.Forms.IncomeTaxXML.ValidFromYear || r.Year > r.Forms.IncomeTaxXML.ValidToYear) {
		errs = append(errs, fmt.Errorf("%s final rule set uses form %s outside declared validity %d..%d: %w", r.ID, r.Forms.IncomeTaxXML.FormCode, r.Forms.IncomeTaxXML.ValidFromYear, r.Forms.IncomeTaxXML.ValidToYear, domain.ErrInvalidInput))
	}
	return errors.Join(errs...)
}
