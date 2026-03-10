package domain

import "time"

// Contact type constants.
const (
	ContactTypeCompany    = "company"
	ContactTypeIndividual = "individual"
)

// Contact represents a business contact (customer or vendor).
type Contact struct {
	ID   int64  `json:"id"`
	Type string `json:"type"` // "company" or "individual"
	Name string `json:"name"`

	// Czech business identifiers
	ICO string `json:"ico"`
	DIC string `json:"dic"`

	// Address
	Street  string `json:"street"`
	City    string `json:"city"`
	ZIP     string `json:"zip"`
	Country string `json:"country"`

	// Contact info
	Email string `json:"email"`
	Phone string `json:"phone"`
	Web   string `json:"web"`

	// Bank details
	BankAccount string `json:"bank_account"`
	BankCode    string `json:"bank_code"`
	IBAN        string `json:"iban"`
	SWIFT       string `json:"swift"`

	// Settings
	PaymentTermsDays int    `json:"payment_terms_days"`
	Tags             string `json:"tags"` // comma-separated
	Notes            string `json:"notes"`

	// Flags
	IsFavorite    bool `json:"is_favorite"`
	VATUnreliable bool `json:"vat_unreliable"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}
