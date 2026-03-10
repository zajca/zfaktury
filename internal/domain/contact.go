package domain

import "time"

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
