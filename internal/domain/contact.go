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
