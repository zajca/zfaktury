package taxrules

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

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

// Date is a civil date without time-of-day or timezone semantics.
type Date struct {
	Year  int        `json:"year"`
	Month time.Month `json:"month"`
	Day   int        `json:"day"`
}

// NewDate constructs a civil date value.
func NewDate(year int, month time.Month, day int) Date {
	return Date{Year: year, Month: month, Day: day}
}

func yearStart(year int) Date {
	return NewDate(year, time.January, 1)
}

func yearEnd(year int) Date {
	return NewDate(year, time.December, 31)
}

func (d Date) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

func (d Date) validate() error {
	if d.Year < 1 {
		return fmt.Errorf("invalid year %d", d.Year)
	}
	if d.Month < time.January || d.Month > time.December {
		return fmt.Errorf("invalid month %d", d.Month)
	}
	t := time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
	if t.Year() != d.Year || t.Month() != d.Month || t.Day() != d.Day {
		return fmt.Errorf("invalid day %d for %04d-%02d", d.Day, d.Year, d.Month)
	}
	return nil
}

func (d Date) compare(other Date) int {
	switch {
	case d.Year != other.Year:
		return d.Year - other.Year
	case d.Month != other.Month:
		return int(d.Month - other.Month)
	default:
		return d.Day - other.Day
	}
}

func (d Date) addDays(days int) Date {
	t := time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC).AddDate(0, 0, days)
	return NewDate(t.Year(), t.Month(), t.Day())
}

// EffectivePeriod stores a value that applies over an inclusive civil-date range.
type EffectivePeriod[T any] struct {
	From  Date `json:"from"`
	To    Date `json:"to"`
	Value T    `json:"value"`
}

// Schedule is a set of effective periods for a value that can change over time.
type Schedule[T any] []EffectivePeriod[T]

// ValueAt resolves the value effective on date.
func (s Schedule[T]) ValueAt(date Date) (T, error) {
	var zero T
	if err := date.validate(); err != nil {
		return zero, fmt.Errorf("invalid effective date %s: %w", date, err)
	}
	for _, period := range s {
		if period.From.compare(date) <= 0 && period.To.compare(date) >= 0 {
			return period.Value, nil
		}
	}
	return zero, fmt.Errorf("no rule value effective on %s: %w", date, domain.ErrInvalidInput)
}

// ValidateCoverage verifies that the schedule covers the inclusive range
// exactly once with no gaps or overlaps.
func (s Schedule[T]) ValidateCoverage(start, end Date) error {
	if err := start.validate(); err != nil {
		return fmt.Errorf("invalid coverage start %s: %w", start, err)
	}
	if err := end.validate(); err != nil {
		return fmt.Errorf("invalid coverage end %s: %w", end, err)
	}
	if start.compare(end) > 0 {
		return fmt.Errorf("coverage start %s is after end %s: %w", start, end, domain.ErrInvalidInput)
	}
	if len(s) == 0 {
		return fmt.Errorf("empty effective schedule for %s..%s: %w", start, end, domain.ErrInvalidInput)
	}

	periods := append([]EffectivePeriod[T](nil), s...)
	sort.Slice(periods, func(i, j int) bool {
		return periods[i].From.compare(periods[j].From) < 0
	})

	wantFrom := start
	for _, period := range periods {
		if err := period.From.validate(); err != nil {
			return fmt.Errorf("invalid period start %s: %w", period.From, err)
		}
		if err := period.To.validate(); err != nil {
			return fmt.Errorf("invalid period end %s: %w", period.To, err)
		}
		if period.From.compare(period.To) > 0 {
			return fmt.Errorf("period %s..%s is inverted: %w", period.From, period.To, domain.ErrInvalidInput)
		}
		if period.From.compare(wantFrom) != 0 {
			return fmt.Errorf("schedule gap or overlap: got period starting %s, want %s: %w", period.From, wantFrom, domain.ErrInvalidInput)
		}
		wantFrom = period.To.addDays(1)
	}
	if wantFrom.compare(end.addDays(1)) != 0 {
		return fmt.Errorf("schedule ends at %s, want %s: %w", wantFrom.addDays(-1), end, domain.ErrInvalidInput)
	}
	return nil
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
	SocialMinMonthly       domain.Amount         // min mesicni vym. zaklad CSSZ
	SocialRate             int                   // permille*10, e.g. 292 = 29.2%
	HealthMinMonthly       domain.Amount         // min mesicni vym. zaklad ZP
	HealthRate             int                   // permille*10, e.g. 135 = 13.5%
	FlatRateCaps           map[int]domain.Amount // percent -> max halere amount
	TimeTestYears          int                   // years to hold for time test exemption
	SecurityExemptionLimit domain.Amount         // max exempt amount per year (0 = no limit)

	SpouseIncomeLimit           domain.Amount
	DeductionCapMortgage        domain.Amount
	DeductionCapLifeInsurance   domain.Amount
	DeductionCapPension         domain.Amount
	DeductionCapSavingsCombined domain.Amount
	DeductionCapUnionDues       domain.Amount
	MaxChildBonus               domain.Amount
}

// RuleSet is the authoritative set of rules for one tax year and tax domain.
type RuleSet struct {
	ID         string           `json:"id"`
	Year       int              `json:"year"`
	Status     RuleStatus       `json:"status"`
	Sources    []LegalSource    `json:"sources"`
	Constants  TaxYearConstants `json:"constants"`
	Deductions DeductionRules   `json:"deductions"`
	Forms      FormRules        `json:"forms"`
}

// DeductionRules contains rules for §15 deductions.
type DeductionRules struct {
	Pension PensionDeductionRules `json:"pension"`
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

// PensionDeductionRules describes the pension-savings deduction mechanics.
type PensionDeductionRules struct {
	SharedSavingsCap domain.Amount           `json:"shared_savings_cap"`
	MonthlyThreshold Schedule[domain.Amount] `json:"monthly_threshold"`
}

// Fingerprint returns a deterministic hash of the rule-set data. It is stored
// with calculated tax returns so old calculations can be reproduced and audited.
func (r RuleSet) Fingerprint() (string, error) {
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
	if len(r.Constants.FlatRateCaps) == 0 {
		errs = append(errs, fmt.Errorf("%s flat-rate caps are empty: %w", r.ID, domain.ErrInvalidInput))
	}
	if r.Deductions.Pension.SharedSavingsCap != r.Constants.DeductionCapSavingsCombined {
		errs = append(errs, fmt.Errorf("%s pension shared cap %d does not match constants cap %d: %w", r.ID, r.Deductions.Pension.SharedSavingsCap, r.Constants.DeductionCapSavingsCombined, domain.ErrInvalidInput))
	}
	if r.Deductions.Pension.SharedSavingsCap < 0 {
		errs = append(errs, fmt.Errorf("%s pension shared cap must not be negative: %w", r.ID, domain.ErrInvalidInput))
	}
	if err := r.Deductions.Pension.MonthlyThreshold.ValidateCoverage(yearStart(r.Year), yearEnd(r.Year)); err != nil {
		errs = append(errs, fmt.Errorf("%s pension monthly threshold: %w", r.ID, err))
	}
	if r.Forms.IncomeTaxXML.FormCode == "" {
		errs = append(errs, fmt.Errorf("%s income-tax XML form code is empty: %w", r.ID, domain.ErrInvalidInput))
	}
	if r.Status == RuleStatusFinal && (r.Year < r.Forms.IncomeTaxXML.ValidFromYear || r.Year > r.Forms.IncomeTaxXML.ValidToYear) {
		errs = append(errs, fmt.Errorf("%s final rule set uses form %s outside declared validity %d..%d: %w", r.ID, r.Forms.IncomeTaxXML.FormCode, r.Forms.IncomeTaxXML.ValidFromYear, r.Forms.IncomeTaxXML.ValidToYear, domain.ErrInvalidInput))
	}
	return errors.Join(errs...)
}
