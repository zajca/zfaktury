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

	// Data-box filing errors
	//
	// ErrFilingAlreadySent means the same XML was already sent (or may have
	// been sent) to the same environment.
	ErrFilingAlreadySent = errors.New("filing already sent through the data box")
	// ErrFilingSendInProgress means another send of the same document is running.
	ErrFilingSendInProgress = errors.New("filing send already in progress")
	// ErrConfirmTokenInvalid means the confirmation token is unknown, expired,
	// already used, or does not match the document, environment, recipient and
	// XML it was issued for.
	ErrConfirmTokenInvalid = errors.New("confirmation token invalid or expired")
	// ErrWarningsNotAccepted means the pre-send check reported warnings and
	// the caller did not explicitly accept them.
	ErrWarningsNotAccepted = errors.New("pre-send warnings not accepted")
	// ErrISDSCredentialsMissing means no data-box login is stored for the
	// requested environment.
	ErrISDSCredentialsMissing = errors.New("data box credentials not configured")
	// ErrSubmissionNotDelivered means an action needs a delivered submission.
	ErrSubmissionNotDelivered = errors.New("submission not delivered")

	// Reminder errors
	ErrInvoiceNotOverdue = errors.New("invoice is not overdue")
	ErrNoCustomerEmail   = errors.New("customer has no email address")
)
