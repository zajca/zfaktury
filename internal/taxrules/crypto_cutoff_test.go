package taxrules

import (
	"testing"
	"time"
)

// The crypto time-test cutoff is a calendar date in Europe/Prague, compared by
// calendar date and not by instant: a sale at 00:30 local time on 15 Feb 2025
// is on the cutoff day and covered.
func TestCryptoTimeTestCovers_CalendarDates(t *testing.T) {
	rules, err := GetRuleSet(2025)
	if err != nil {
		t.Fatalf("GetRuleSet(2025) error: %v", err)
	}
	c := rules.Constants
	if c.CryptoTimeTestFrom.IsZero() {
		t.Fatal("2025 rule set has no CryptoTimeTestFrom")
	}

	prague := pragueLocation
	tests := []struct {
		name string
		date time.Time
		want bool
	}{
		{"day before cutoff", time.Date(2025, 2, 14, 12, 0, 0, 0, prague), false},
		{"cutoff day at 00:30 local", time.Date(2025, 2, 15, 0, 30, 0, 0, prague), true},
		{"cutoff day at 23:59 local", time.Date(2025, 2, 15, 23, 59, 0, 0, prague), true},
		{"day after cutoff", time.Date(2025, 2, 16, 0, 0, 0, 0, prague), true},
		// 2025-02-14 23:30 UTC is already 2025-02-15 00:30 in Prague.
		{"utc instant on the cutoff day in Prague", time.Date(2025, 2, 14, 23, 30, 0, 0, time.UTC), true},
		{"utc instant before the cutoff day in Prague", time.Date(2025, 2, 14, 22, 30, 0, 0, time.UTC), false},
		{"crypto sale in 2024", time.Date(2024, 6, 1, 12, 0, 0, 0, prague), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.CryptoTimeTestCovers(tt.date); got != tt.want {
				t.Errorf("CryptoTimeTestCovers(%s) = %v, want %v", tt.date, got, tt.want)
			}
		})
	}
}

func TestCryptoTimeTestCovers_ZeroCutoffCoversEverything(t *testing.T) {
	var c TaxYearConstants
	if !c.CryptoTimeTestCovers(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Error("zero CryptoTimeTestFrom should cover every date")
	}
}
