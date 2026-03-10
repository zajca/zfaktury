package domain

import "time"

// InvoiceFilter holds filtering options for listing invoices.
type InvoiceFilter struct {
	Status     string     `json:"status"`
	CustomerID *int64     `json:"customer_id"`
	DateFrom   *time.Time `json:"date_from"`
	DateTo     *time.Time `json:"date_to"`
	Search     string     `json:"search"`
	Limit      int        `json:"limit"`
	Offset     int        `json:"offset"`
}

// ExpenseFilter holds filtering options for listing expenses.
type ExpenseFilter struct {
	Category string     `json:"category"`
	VendorID *int64     `json:"vendor_id"`
	DateFrom *time.Time `json:"date_from"`
	DateTo   *time.Time `json:"date_to"`
	Search   string     `json:"search"`
	Limit    int        `json:"limit"`
	Offset   int        `json:"offset"`
}
