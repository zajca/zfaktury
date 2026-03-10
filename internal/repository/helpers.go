package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// parseDate parses a date string and returns the time value.
// It tries the given layout first, and if the layout is RFC3339 and parsing
// fails, it also tries the SQLite datetime format (space instead of T separator).
func parseDate(layout, value string) (time.Time, error) {
	t, err := time.Parse(layout, value)
	if err != nil && layout == time.RFC3339 && strings.Contains(value, " ") {
		// SQLite sometimes stores datetimes with space separator instead of T.
		// Try replacing the first space between date and time with T.
		normalized := strings.Replace(value, " ", "T", 1)
		// Truncate sub-second precision beyond 9 digits if present.
		if t2, err2 := time.Parse(time.RFC3339Nano, normalized); err2 == nil {
			return t2, nil
		}
	}
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
