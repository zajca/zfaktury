package domain

import "time"

// Expense represents a business expense / received invoice.
type Expense struct {
	ID            int64
	VendorID      *int64
	Vendor        *Contact
	ExpenseNumber string
	Category      string
	Description   string

	IssueDate    time.Time
	Amount       Amount
	CurrencyCode string
	ExchangeRate Amount

	VATRatePercent int
	VATAmount      Amount

	IsTaxDeductible bool
	BusinessPercent int // 0-100, percentage used for business
	PaymentMethod   string

	DocumentPath string
	Notes        string

	TaxReviewedAt *time.Time

	// Timestamps
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
