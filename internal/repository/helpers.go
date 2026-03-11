package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// parseDate parses a date string and returns the time value.
// It tries the given layout first, with fallbacks for common mismatches:
//   - RFC3339 layout with space-separated SQLite datetime
//   - Date-only layout ("2006-01-02") with full timestamp value
func parseDate(layout, value string) (time.Time, error) {
	t, err := time.Parse(layout, value)
	if err == nil {
		return t, nil
	}

	// Fallback: RFC3339 layout but value uses space instead of T separator.
	if layout == time.RFC3339 && strings.Contains(value, " ") {
		normalized := strings.Replace(value, " ", "T", 1)
		if t2, err2 := time.Parse(time.RFC3339Nano, normalized); err2 == nil {
			return t2, nil
		}
	}

	// Fallback: date-only layout but value has full timestamp (e.g. "2006-01-02T00:00:00Z").
	if len(layout) == 10 && len(value) > 10 {
		if t2, err2 := time.Parse(layout, value[:10]); err2 == nil {
			return t2, nil
		}
	}

	return time.Time{}, fmt.Errorf("parsing date %q: %w", value, err)
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
