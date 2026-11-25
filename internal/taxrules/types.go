package taxrules

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	// Embedded tzdata so Europe/Prague loads on every platform, including
	// Windows builds without system tzdata.
	_ "time/tzdata"

	"github.com/zajca/zfaktury/internal/domain"
)

// pragueZone returns the Europe/Prague zone used for calendar-date rule
// values. A load error is returned to the caller; there is no silent fallback.
func pragueZone() (*time.Location, error) {
	return time.LoadLocation("Europe/Prague")
}

// CivilDate is a calendar date without a time of day or zone. Rule values
// stored as CivilDate serialize identically on every platform, so the rule-set
// fingerprint stays deterministic.
type CivilDate struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

// Before reports whether d is an earlier calendar date than o.
func (d CivilDate) Before(o CivilDate) bool {
	if d.Year != o.Year {
		return d.Year < o.Year
	}
	if d.Month != o.Month {
		return d.Month < o.Month
	}
	return d.Day < o.Day
}

// String returns the ISO 8601 calendar date, e.g. "2025-02-15".
func (d CivilDate) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

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

// MonthlyBasePeriod is a minimum monthly assessment base that applies from
// FromMonth (1-12) until the next period of the same year starts.
type MonthlyBasePeriod struct {
	FromMonth int
	Base      domain.Amount
}

// TaxYearConstants holds tax computation constants for a specific year.
type TaxYearConstants struct {
	ProgressiveThreshold  domain.Amount       // prah pro 23% sazbu (in halere)
	BasicCredit           domain.Amount       // sleva na poplatnika
	SpouseCredit          domain.Amount       // sleva na manzela/ku
	StudentCredit         domain.Amount       // student
	DisabilityCredit1     domain.Amount       // invalidita 1. a 2. stupen
	DisabilityCredit3     domain.Amount       // invalidita 3. stupen
	DisabilityZTPP        domain.Amount       // drzitel prukazu ZTP/P
	ChildBenefit1         domain.Amount       // 1. dite
	ChildBenefit2         domain.Amount       // 2. dite
	ChildBenefit3Plus     domain.Amount       // 3+ dite
	ChildBenefitZTP       domain.Amount       // ZTP prirazka (double)
	SocialMinMonthly      domain.Amount       // min mesicni vym. zaklad CSSZ pro rocni Prehled (platny pro vsechny mesice roku)
	SocialRate            int                 // permille*10, e.g. 292 = 29.2%
	SocialMaxAnnualBase   domain.Amount       // max rocni vym. zaklad CSSZ (48nasobek prumerne mzdy)
	SocialMinAdvanceBases []MonthlyBasePeriod // min mesicni vym. zaklad CSSZ pro zalohy placene v tomto roce, po obdobich
	// SocialSecondaryMinMonthly is the minimum monthly base of a secondary
	// activity (vedlejsi SVC), for the annual overview and for advances paid
	// in this year.
	SocialSecondaryMinMonthly domain.Amount
	// SocialDecisiveAmount is the rozhodna castka of a full year of secondary
	// activity; it is reduced by SocialDecisiveAmountMonthlyStep for each month
	// without secondary activity.
	SocialDecisiveAmount            domain.Amount
	SocialDecisiveAmountMonthlyStep domain.Amount
	HealthMinMonthly                domain.Amount         // min mesicni vym. zaklad ZP
	HealthRate                      int                   // permille*10, e.g. 135 = 13.5%
	FlatRateCaps                    map[int]domain.Amount // percent -> max halere amount
	TimeTestYears                   int                   // years to hold for time test exemption
	// CryptoTimeTestFrom is the first calendar date (Europe/Prague) on which the
	// crypto time-test exemption applies; crypto sales before it are never
	// time-test exempt. Zero means no cutoff is configured.
	CryptoTimeTestFrom     CivilDate
	SecurityExemptionLimit domain.Amount // max yearly gross time-test income that stays exempt (0 = no limit)

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

// CryptoTimeTestCovers reports whether a crypto sale dated saleDate falls on
// or after CryptoTimeTestFrom. The sale's calendar date in Europe/Prague is
// compared, not the instant, so a sale at 00:30 local time on the cutoff day
// counts as being on the cutoff day. A zero CryptoTimeTestFrom covers every
// date.
func (c TaxYearConstants) CryptoTimeTestCovers(saleDate time.Time) (bool, error) {
	if c.CryptoTimeTestFrom == (CivilDate{}) {
		return true, nil
	}
	prague, err := pragueZone()
	if err != nil {
		return false, fmt.Errorf("loading Europe/Prague for the crypto time-test cutoff: %w", err)
	}
	y, m, d := saleDate.In(prague).Date()
	sale := CivilDate{Year: y, Month: int(m), Day: d}
	return !sale.Before(c.CryptoTimeTestFrom), nil
}

// SocialMinAdvanceBaseFor returns the minimum monthly base for a social
// insurance advance paid in the given month of the rule-set year.
func (c TaxYearConstants) SocialMinAdvanceBaseFor(month int) (domain.Amount, error) {
	if month < 1 || month > 12 {
		return 0, fmt.Errorf("advance month %d out of range 1-12: %w", month, domain.ErrInvalidInput)
	}
	var base domain.Amount
	for _, period := range c.SocialMinAdvanceBases {
		if period.FromMonth > month {
			break
		}
		base = period.Base
	}
	if base == 0 {
		return 0, fmt.Errorf("no minimum advance base for month %d: %w", month, domain.ErrInvalidInput)
	}
	return base, nil
}

// validateSocialMinAdvanceBases checks that the periods start in January, are
// strictly increasing and carry a positive base.
func validateSocialMinAdvanceBases(id string, periods []MonthlyBasePeriod) error {
	if len(periods) == 0 {
		return fmt.Errorf("%s social minimum advance bases are empty: %w", id, domain.ErrInvalidInput)
	}
	var errs []error
	if periods[0].FromMonth != 1 {
		errs = append(errs, fmt.Errorf("%s social minimum advance bases must start in month 1, got %d: %w", id, periods[0].FromMonth, domain.ErrInvalidInput))
	}
	for i, period := range periods {
		if period.FromMonth < 1 || period.FromMonth > 12 {
			errs = append(errs, fmt.Errorf("%s social minimum advance base month %d out of range 1-12: %w", id, period.FromMonth, domain.ErrInvalidInput))
		}
		if period.Base <= 0 {
			errs = append(errs, fmt.Errorf("%s social minimum advance base from month %d must be positive: %w", id, period.FromMonth, domain.ErrInvalidInput))
		}
		if i > 0 && period.FromMonth <= periods[i-1].FromMonth {
			errs = append(errs, fmt.Errorf("%s social minimum advance base months must increase, got %d after %d: %w", id, period.FromMonth, periods[i-1].FromMonth, domain.ErrInvalidInput))
		}
	}
	return errors.Join(errs...)
}

// validateSocialSecondary checks the secondary-activity constants: all are
// positive and the monthly step is the decisive amount divided by 12, rounded up
// to whole CZK.
func validateSocialSecondary(id string, c TaxYearConstants) error {
	var errs []error
	if c.SocialSecondaryMinMonthly <= 0 {
		errs = append(errs, fmt.Errorf("%s social secondary minimum monthly base must be positive: %w", id, domain.ErrInvalidInput))
	}
	if c.SocialDecisiveAmount <= 0 {
		errs = append(errs, fmt.Errorf("%s social decisive amount must be positive: %w", id, domain.ErrInvalidInput))
	}
	if c.SocialDecisiveAmountMonthlyStep <= 0 {
		errs = append(errs, fmt.Errorf("%s social decisive amount monthly step must be positive: %w", id, domain.ErrInvalidInput))
	}
	if c.SocialDecisiveAmount > 0 && c.SocialDecisiveAmountMonthlyStep > 0 {
		step := c.SocialDecisiveAmountMonthlyStep
		wholeCZK := domain.NewAmount(1, 0)
		if step%wholeCZK != 0 || step*12 < c.SocialDecisiveAmount || (step-wholeCZK)*12 >= c.SocialDecisiveAmount {
			errs = append(errs, fmt.Errorf("%s social decisive amount step %d is not the amount %d divided by 12 rounded up: %w", id, step, c.SocialDecisiveAmount, domain.ErrInvalidInput))
		}
	}
	return errors.Join(errs...)
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
	if err := validateSocialSecondary(r.ID, r.Constants); err != nil {
		errs = append(errs, err)
	}
	if len(r.Constants.FlatRateCaps) == 0 {
		errs = append(errs, fmt.Errorf("%s flat-rate caps are empty: %w", r.ID, domain.ErrInvalidInput))
	}
	if err := validateSocialMinAdvanceBases(r.ID, r.Constants.SocialMinAdvanceBases); err != nil {
		errs = append(errs, err)
	}
	if r.Forms.IncomeTaxXML.FormCode == "" {
		errs = append(errs, fmt.Errorf("%s income-tax XML form code is empty: %w", r.ID, domain.ErrInvalidInput))
	}
	if r.Status == RuleStatusFinal && (r.Year < r.Forms.IncomeTaxXML.ValidFromYear || r.Year > r.Forms.IncomeTaxXML.ValidToYear) {
		errs = append(errs, fmt.Errorf("%s final rule set uses form %s outside declared validity %d..%d: %w", r.ID, r.Forms.IncomeTaxXML.FormCode, r.Forms.IncomeTaxXML.ValidFromYear, r.Forms.IncomeTaxXML.ValidToYear, domain.ErrInvalidInput))
	}
	return errors.Join(errs...)
}
