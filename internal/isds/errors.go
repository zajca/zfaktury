package isds

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

// maxServerTextRunes bounds server-provided text copied into error messages.
const maxServerTextRunes = 200

var (
	// ErrUnauthorized is returned when ISDS answers 401 or 403 (bad
	// credentials or a login that may not use the service).
	ErrUnauthorized = errors.New("ISDS rejected the credentials")
	// ErrISDSUnavailable is returned for transport failures and non-2xx
	// answers that carry no SOAP fault.
	ErrISDSUnavailable = errors.New("ISDS unavailable")
	// ErrISDSTimeout is returned when ISDS does not answer in time. Such an
	// error also matches ErrISDSUnavailable.
	ErrISDSTimeout = errors.New("ISDS request timed out")
	// ErrISDSRejected is matched by *StatusError: ISDS answered but its
	// status code is not 0000.
	ErrISDSRejected = errors.New("ISDS rejected the request")
	// ErrSOAPFault is matched by *FaultError.
	ErrSOAPFault = errors.New("ISDS returned a SOAP fault")
	// ErrUnexpectedResponse is returned when a 2xx answer is not the
	// expected SOAP response (for example an HTML maintenance page).
	ErrUnexpectedResponse = errors.New("unexpected ISDS response")
	// ErrResponseTooLarge is returned when the response exceeds the
	// configured size cap.
	ErrResponseTooLarge = errors.New("ISDS response exceeds the size limit")
	// ErrInvalidInput is returned before any request is sent when the
	// caller's input violates an ISDS limit or is malformed.
	ErrInvalidInput = errors.New("invalid ISDS input")
	// ErrInvalidConfig is returned by NewClient for a missing environment,
	// missing credentials or an unusable base URL.
	ErrInvalidConfig = errors.New("invalid ISDS client configuration")
)

// StatusError carries the dmStatus or dbStatus of an ISDS answer whose code
// is not 0000.
type StatusError struct {
	Code      string
	Message   string
	RefNumber string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("ISDS status %s: %s", e.Code, e.Message)
}

// Is makes errors.Is(err, ErrISDSRejected) true.
func (e *StatusError) Is(target error) bool { return errors.Is(ErrISDSRejected, target) }

// FaultError carries a SOAP fault returned by ISDS.
type FaultError struct {
	Code    string
	Message string
}

func (e *FaultError) Error() string {
	return fmt.Sprintf("ISDS SOAP fault %s: %s", e.Code, e.Message)
}

// Is makes errors.Is(err, ErrSOAPFault) true.
func (e *FaultError) Is(target error) bool { return errors.Is(ErrSOAPFault, target) }

// truncate shortens server-provided text so an error message stays small.
func truncate(s string) string {
	if utf8.RuneCountInString(s) <= maxServerTextRunes {
		return s
	}
	r := []rune(s)
	return string(r[:maxServerTextRunes])
}

func invalidInput(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidInput, fmt.Sprintf(format, args...))
}
