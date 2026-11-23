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
		return addMonthsClamped(r.NextIssueDate, 1)
	case FrequencyQuarterly:
		return addMonthsClamped(r.NextIssueDate, 3)
	case FrequencyYearly:
		return addMonthsClamped(r.NextIssueDate, 12)
	default:
		return addMonthsClamped(r.NextIssueDate, 1)
	}
}

// addMonthsClamped adds n months to t without overflowing into the following
// month. A date on the last day of its month stays on the last day of the
// target month; any other day is kept and clamped to the target month's length.
// Time of day and location are preserved.
func addMonthsClamped(t time.Time, n int) time.Time {
	year, month, day := t.Date()
	lastDayOfMonth := daysInMonth(year, month)
	target := time.Date(year, month+time.Month(n), 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	targetLast := daysInMonth(target.Year(), target.Month())

	if day == lastDayOfMonth || day > targetLast {
		day = targetLast
	}
	return time.Date(target.Year(), target.Month(), day, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// daysInMonth returns the number of days in the given month.
func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
