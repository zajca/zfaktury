package domain

import (
	"fmt"
	"strings"
	"time"
)

// Contact type constants.
const (
	ContactTypeCompany    = "company"
	ContactTypeIndividual = "individual"
)

// Contact represents a business contact (customer or vendor).
type Contact struct {
	ID   int64
	Type string // "company" or "individual"
	Name string

	// Czech business identifiers
	ICO string
	DIC string

	// Address
	Street  string
	City    string
	ZIP     string
	Country string

	// Contact info
	Email string
	Phone string
	Web   string

	// Bank details
	BankAccount string
	BankCode    string
	IBAN        string
	SWIFT       string

	// Settings
	PaymentTermsDays int
	Tags             string // comma-separated
	Notes            string

	// Flags
	IsFavorite bool

	// VATUnreliableAt is set when the contact is flagged as an unreliable VAT payer.
	VATUnreliableAt *time.Time

	// Timestamps
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// DICCountryCode returns the 2-letter country prefix from the DIC (e.g. "CZ" from "CZ12345678").
// Returns empty string if DIC is too short.
func (c *Contact) DICCountryCode() string {
	if len(c.DIC) < 2 {
		return ""
	}
	return c.DIC[:2]
}

// IsEUPartner returns true if the contact has a non-CZ EU DIC.
func (c *Contact) IsEUPartner() bool {
	code := c.DICCountryCode()
	if code == "" || code == "CZ" {
		return false
	}
	// EU member state codes
	euCodes := map[string]bool{
		"AT": true, "BE": true, "BG": true, "HR": true, "CY": true,
		"DE": true, "DK": true, "EE": true, "ES": true, "FI": true,
		"FR": true, "GR": true, "EL": true, "HU": true, "IE": true,
		"IT": true, "LT": true, "LU": true, "LV": true, "MT": true,
		"NL": true, "PL": true, "PT": true, "RO": true, "SE": true,
		"SI": true, "SK": true,
	}
	return euCodes[code]
}

// HasCZDIC returns true if the contact has a Czech DIC.
func (c *Contact) HasCZDIC() bool {
	return c.DICCountryCode() == "CZ"
}

// ValidateICO validates a Czech IČO (identification number) using the ARES
// modulo-11 checksum algorithm.
//
// Algorithm (per ARES specification):
//  1. Trim whitespace, left-pad with zeros to 8 digits if shorter.
//  2. All 8 characters must be digits.
//  3. Multiply the first 7 digits by weights [8, 7, 6, 5, 4, 3, 2] and sum.
//  4. Compute remainder = sum mod 11.
//  5. Expected check digit:
//     - remainder == 0 → 1
//     - remainder == 1 → 0
//     - otherwise      → 11 − remainder
//  6. Compare expected check digit to the 8th digit of the input.
//
// Returns nil for a valid IČO, otherwise an error wrapping ErrInvalidInput.
func ValidateICO(ico string) error {
	ico = strings.TrimSpace(ico)
	if ico == "" {
		return fmt.Errorf("IČO is empty: %w", ErrInvalidInput)
	}
	// Left-pad with zeros to 8 digits (ARES allows 6-, 7-, 8-digit forms).
	for len(ico) < 8 {
		ico = "0" + ico
	}
	if len(ico) != 8 {
		return fmt.Errorf("IČO %q must be 8 digits: %w", ico, ErrInvalidInput)
	}
	for _, r := range ico {
		if r < '0' || r > '9' {
			return fmt.Errorf("IČO %q must contain only digits: %w", ico, ErrInvalidInput)
		}
	}

	weights := [7]int{8, 7, 6, 5, 4, 3, 2}
	sum := 0
	for i := 0; i < 7; i++ {
		sum += int(ico[i]-'0') * weights[i]
	}
	remainder := sum % 11

	var expected int
	switch remainder {
	case 0:
		expected = 1
	case 1:
		expected = 0
	default:
		expected = 11 - remainder
	}

	actual := int(ico[7] - '0')
	if actual != expected {
		return fmt.Errorf("IČO %q has invalid checksum (expected %d, got %d): %w", ico, expected, actual, ErrInvalidInput)
	}
	return nil
}
