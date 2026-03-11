package domain

import "time"

// InvoiceFilter holds filtering options for listing invoices.
type InvoiceFilter struct {
	Status     string
	Type       string
	CustomerID *int64
	DateFrom   *time.Time
	DateTo     *time.Time
	Search     string
	Limit      int
	Offset     int
}

// ExpenseFilter holds filtering options for listing expenses.
type ExpenseFilter struct {
	Category    string
	VendorID    *int64
	DateFrom    *time.Time
	DateTo      *time.Time
	Search      string
	TaxReviewed *bool
	Limit       int
	Offset      int
}
