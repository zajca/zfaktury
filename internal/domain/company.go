package domain

import (
	"fmt"
	"regexp"
	"time"
)

// Company represents a legal entity managed inside zfaktury.
// All per-company entities (invoices, contacts, expenses, etc.) reference
// exactly one Company via company_id.
//
// Pure domain struct — no DB or JSON tags. Repository and handler layers
// own their own representations.
type Company struct {
	ID            int64
	Name          string // short label shown in switcher dropdown
	LegalName     string // full legal name printed on invoices
	ICO           string // 8-digit Czech business ID
	DIC           string // VAT ID, format CZ\d{8,10} when VATRegistered
	VATRegistered bool

	// Address
	Street      string
	HouseNumber string
	City        string
	ZIP         string

	// Contact
	Email string
	Phone string

	// Personal name (OSVČ tax filings name the human, not the brand)
	FirstName string
	LastName  string

	// Bank
	BankAccount string
	BankCode    string
	IBAN        string
	SWIFT       string

	// Presentation
	LogoPath    string
	AccentColor string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

var dicPattern = regexp.MustCompile(`^CZ\d{8,10}$`)

// Validate enforces the invariants that hold at the domain level.
// DB-level constraints (uniqueness of ICO, FKs) are enforced by the schema.
func (c *Company) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("%w: company name is required", ErrInvalidInput)
	}
	if c.LegalName == "" {
		return fmt.Errorf("%w: legal name is required", ErrInvalidInput)
	}
	if c.ICO == "" {
		return fmt.Errorf("%w: ICO is required", ErrInvalidInput)
	}
	if c.VATRegistered {
		if c.DIC == "" {
			return fmt.Errorf("%w: DIC is required for VAT-registered companies", ErrInvalidInput)
		}
		if !dicPattern.MatchString(c.DIC) {
			return fmt.Errorf("%w: DIC must match CZ\\d{8,10}, got %q", ErrInvalidInput, c.DIC)
		}
	}
	return nil
}
