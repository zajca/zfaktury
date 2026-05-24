package domain

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrInvalidInput    = errors.New("invalid input")
	ErrPaidInvoice     = errors.New("invoice already paid")
	ErrNoItems         = errors.New("no items")
	ErrDuplicateNumber = errors.New("duplicate number")

	// VAT filing errors
	ErrFilingAlreadyExists = errors.New("filing already exists for this period")
	ErrFilingAlreadyFiled  = errors.New("filing already filed, cannot modify")

	// Settings errors
	ErrMissingSetting = errors.New("required setting not configured")

	// Reminder errors
	ErrInvoiceNotOverdue = errors.New("invoice is not overdue")
	ErrNoCustomerEmail   = errors.New("customer has no email address")

	// ErrLastCompany indicates an attempt to soft-delete the only remaining company.
	ErrLastCompany = errors.New("cannot delete the last company")

	// ErrInUse indicates an attempt to soft-delete an entity (e.g. a company)
	// that still has non-deleted child records.
	ErrInUse = errors.New("cannot delete: still in use")
)
