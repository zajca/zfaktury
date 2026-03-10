package repository

import (
	"database/sql"
	"fmt"
	"time"
)

// parseDate parses a date string and returns the time value.
func parseDate(layout, value string) (time.Time, error) {
	t, err := time.Parse(layout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing date %q: %w", value, err)
	}
	return t, nil
}

// parseDateOptional parses an optional date from a sql.NullString.
func parseDateOptional(layout string, ns sql.NullString) (time.Time, error) {
	if !ns.Valid || ns.String == "" {
		return time.Time{}, nil
	}
	return parseDate(layout, ns.String)
}

// parseDatePtr parses an optional date from a sql.NullString and returns a pointer.
// Returns nil if the NullString is not valid or empty.
func parseDatePtr(layout string, ns sql.NullString) (*time.Time, error) {
	if !ns.Valid || ns.String == "" {
		return nil, nil
	}
	t, err := parseDate(layout, ns.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
