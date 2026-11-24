package domain

import "time"

// InvoiceFilter holds filtering options for listing invoices.
type InvoiceFilter struct {
	Status     string
	Type       string
	CustomerID *int64
	DateFrom   *time.Time
	DateTo     *time.Time
	// DeliveryDateFrom and DeliveryDateTo select by the date of the taxable
	// supply (DUZP). An invoice without a delivery date counts as delivered
	// on its issue date.
	DeliveryDateFrom *time.Time
	DeliveryDateTo   *time.Time
	Search           string
	Limit            int
	Offset           int
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
