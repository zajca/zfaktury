package domain

import (
	"fmt"
	"time"
)

// FilingMethod says how the income-tax return for a year is filed; it decides
// the deadline of the ČSSZ and health overviews (§ 136 zákona č. 280/2009 Sb.).
type FilingMethod string

const (
	// FilingMethodUnset: not chosen; ČSSZ then assumes the return was filed by 1 April.
	FilingMethodUnset FilingMethod = ""
	// FilingMethodStandard: filed by 1 April, on paper or electronically.
	FilingMethodStandard FilingMethod = "standard"
	// FilingMethodElectronic: filed electronically after 1 April (4-month deadline).
	FilingMethodElectronic FilingMethod = "electronic"
	// FilingMethodAdvisor: filed by a tax advisor after 1 April (6-month deadline).
	FilingMethodAdvisor FilingMethod = "advisor"
)

// Validate returns ErrInvalidInput unless m is one of the defined filing methods.
func (m FilingMethod) Validate() error {
	switch m {
	case FilingMethodUnset, FilingMethodStandard, FilingMethodElectronic, FilingMethodAdvisor:
		return nil
	}
	return fmt.Errorf("invalid filing method %q: %w", m, ErrInvalidInput)
}

// TaxYearSettings stores per-year tax configuration such as flat rate percent.
type TaxYearSettings struct {
	Year            int
	FlatRatePercent int
	FilingMethod    FilingMethod
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TaxPrepayment stores monthly prepayment amounts for a given year.
type TaxPrepayment struct {
	Year         int
	Month        int
	TaxAmount    Amount
	SocialAmount Amount
	HealthAmount Amount
}
