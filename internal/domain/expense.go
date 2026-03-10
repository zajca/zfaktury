package domain

import "time"

// Expense represents a business expense / received invoice.
type Expense struct {
	ID             int64  `json:"id"`
	VendorID       *int64 `json:"vendor_id,omitempty"`
	Vendor         *Contact `json:"vendor,omitempty"`
	ExpenseNumber  string `json:"expense_number"`
	Category       string `json:"category"`
	Description    string `json:"description"`

	IssueDate    time.Time `json:"issue_date"`
	Amount       Amount    `json:"amount"`
	CurrencyCode string    `json:"currency_code"`
	ExchangeRate Amount    `json:"exchange_rate"`

	VATRatePercent int    `json:"vat_rate_percent"`
	VATAmount      Amount `json:"vat_amount"`

	IsTaxDeductible bool   `json:"is_tax_deductible"`
	BusinessPercent int    `json:"business_percent"` // 0-100, percentage used for business
	PaymentMethod   string `json:"payment_method"`

	DocumentPath string `json:"document_path"`
	Notes        string `json:"notes"`

	TaxReviewedAt *time.Time

	// Timestamps
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}
