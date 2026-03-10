package domain

import "time"

// Frequency constants for recurring invoices.
const (
	FrequencyMonthly   = "monthly"
	FrequencyQuarterly = "quarterly"
	FrequencyYearly    = "yearly"
	FrequencyWeekly    = "weekly"
)

// RecurringInvoice represents a template for automatically generated invoices.
type RecurringInvoice struct {
	ID             int64
	Name           string
	CustomerID     int64
	Customer       *Contact
	Frequency      string // monthly, quarterly, yearly, weekly
	NextIssueDate  time.Time
	EndDate        *time.Time
	CurrencyCode   string
	ExchangeRate   Amount
	PaymentMethod  string
	BankAccount    string
	BankCode       string
	IBAN           string
	SWIFT          string
	ConstantSymbol string
	Notes          string
	IsActive       bool
	Items          []RecurringInvoiceItem
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

// RecurringInvoiceItem represents a line item on a recurring invoice template.
type RecurringInvoiceItem struct {
	ID                 int64
	RecurringInvoiceID int64
	Description        string
	Quantity           Amount
	Unit               string
	UnitPrice          Amount
	VATRatePercent     int
	SortOrder          int
}

// NextDate calculates the next issue date based on frequency.
func (r *RecurringInvoice) NextDate() time.Time {
	switch r.Frequency {
	case FrequencyWeekly:
		return r.NextIssueDate.AddDate(0, 0, 7)
	case FrequencyMonthly:
		return r.NextIssueDate.AddDate(0, 1, 0)
	case FrequencyQuarterly:
		return r.NextIssueDate.AddDate(0, 3, 0)
	case FrequencyYearly:
		return r.NextIssueDate.AddDate(1, 0, 0)
	default:
		return r.NextIssueDate.AddDate(0, 1, 0)
	}
}
