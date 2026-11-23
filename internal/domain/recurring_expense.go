package domain

import "time"

// RecurringExpense represents a template for automatically generated expenses.
type RecurringExpense struct {
	ID              int64
	Name            string
	VendorID        *int64
	Vendor          *Contact
	Category        string
	Description     string
	Amount          Amount
	CurrencyCode    string
	ExchangeRate    Amount
	VATRatePercent  int
	VATAmount       Amount
	IsTaxDeductible bool
	BusinessPercent int
	PaymentMethod   string
	Notes           string
	Frequency       string // weekly, monthly, quarterly, yearly
	NextIssueDate   time.Time
	EndDate         *time.Time
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}

// NextDate calculates the next issue date after the current NextIssueDate
// based on the configured frequency.
func (r *RecurringExpense) NextDate() time.Time {
	switch r.Frequency {
	case "weekly":
		return r.NextIssueDate.AddDate(0, 0, 7)
	case "monthly":
		return addMonthsClamped(r.NextIssueDate, 1)
	case "quarterly":
		return addMonthsClamped(r.NextIssueDate, 3)
	case "yearly":
		return addMonthsClamped(r.NextIssueDate, 12)
	default:
		return addMonthsClamped(r.NextIssueDate, 1)
	}
}
