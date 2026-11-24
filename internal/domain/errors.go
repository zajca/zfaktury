package domain

import "errors"

// UserMessageError is invalid input explained by a sentence meant for the
// user. It matches ErrInvalidInput through errors.Is; the HTTP layer returns
// Message alone, without the wrapping context of the services.
type UserMessageError struct {
	Message string
}

func (e *UserMessageError) Error() string { return e.Message + ": " + ErrInvalidInput.Error() }

func (e *UserMessageError) Unwrap() error { return ErrInvalidInput }

var (
	ErrNotFound        = errors.New("not found")
	ErrInvalidInput    = errors.New("invalid input")
	ErrPaidInvoice     = errors.New("invoice already paid")
	ErrNoItems         = errors.New("no items")
	ErrDuplicateNumber = errors.New("duplicate number")

	// VAT filing errors
	ErrFilingAlreadyExists = errors.New("filing already exists for this period")
	ErrFilingAlreadyFiled  = errors.New("filing already filed, cannot modify")
	ErrXMLNotGenerated     = errors.New("XML has not been generated")

	// Settings errors
	ErrMissingSetting = errors.New("required setting not configured")

	// ErrKeyringUnavailable means the OS keyring cannot be reached (for example
	// no Secret Service on a headless Linux). Credentials are never stored
	// anywhere else.
	ErrKeyringUnavailable = errors.New("system keyring unavailable")

	// ErrKeyringLocked means the keyring is reachable but could not be
	// unlocked (the unlock prompt was dismissed or refused).
	ErrKeyringLocked = errors.New("system keyring locked")

	// Reminder errors
	ErrInvoiceNotOverdue = errors.New("invoice is not overdue")
	ErrNoCustomerEmail   = errors.New("customer has no email address")
)
