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
	ID         int64
	Name       string
	CustomerID int64
	Customer   *Contact
	// SequenceID selects which invoice-number sequence generated invoices use.
	// Zero means "auto-assign" (the company's FV sequence), preserving the
	// pre-sequence-picker behaviour.
	SequenceID     int64
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
	// AutoSend, when true, makes the daily scheduler email each generated
	// invoice automatically. AutoSendRecipient overrides the destination; when
	// blank the customer's contact email is used.
	AutoSend          bool
	AutoSendRecipient string
	Items             []RecurringInvoiceItem
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
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
	case FrequencyQuarterly:
		return addMonthsEOM(r.NextIssueDate, 3)
	case FrequencyYearly:
		return addMonthsEOM(r.NextIssueDate, 12)
	case FrequencyMonthly:
		return addMonthsEOM(r.NextIssueDate, 1)
	default:
		return addMonthsEOM(r.NextIssueDate, 1)
	}
}

// daysInMonth returns the number of days in the given year/month.
func daysInMonth(y int, m time.Month) int {
	// Day 0 of the next month is the last day of month m.
	return time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// addMonthsEOM adds n months to t. If t is the last day of its month, the result
// is the last day of the target month (so end-of-month schedules stay at
// month-end). Otherwise the day is preserved, clamped to the target month's
// length so a month is never skipped (e.g. 31 May + 1 month -> 30 June, not 1 July).
func addMonthsEOM(t time.Time, n int) time.Time {
	y, m, d := t.Date()
	lastOfSource := daysInMonth(y, m)
	// time.Date normalizes an overflowing month into the correct year/month.
	target := time.Date(y, m+time.Month(n), 1,
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	lastOfTarget := daysInMonth(target.Year(), target.Month())

	day := d
	if d >= lastOfSource || d > lastOfTarget {
		day = lastOfTarget
	}
	return time.Date(target.Year(), target.Month(), day,
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}
