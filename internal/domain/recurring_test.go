package domain

import (
	"testing"
	"time"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestRecurringInvoice_NextDate(t *testing.T) {
	tests := []struct {
		name      string
		frequency string
		start     time.Time
		want      time.Time
	}{
		{
			name:      "weekly adds 7 days",
			frequency: FrequencyWeekly,
			start:     date(2026, time.January, 1),
			want:      date(2026, time.January, 8),
		},
		{
			name:      "monthly adds 1 month",
			frequency: FrequencyMonthly,
			start:     date(2026, time.January, 15),
			want:      date(2026, time.February, 15),
		},
		{
			name:      "quarterly adds 3 months",
			frequency: FrequencyQuarterly,
			start:     date(2026, time.January, 1),
			want:      date(2026, time.April, 1),
		},
		{
			name:      "yearly adds 1 year",
			frequency: FrequencyYearly,
			start:     date(2026, time.March, 1),
			want:      date(2027, time.March, 1),
		},
		{
			name:      "unknown frequency defaults to monthly",
			frequency: "biweekly",
			start:     date(2026, time.January, 15),
			want:      date(2026, time.February, 15),
		},
		{
			name:      "Jan 31 monthly clamps to Feb 28",
			frequency: FrequencyMonthly,
			start:     date(2027, time.January, 31),
			want:      date(2027, time.February, 28),
		},
		{
			name:      "Jan 31 monthly clamps to Feb 29 in leap year",
			frequency: FrequencyMonthly,
			start:     date(2028, time.January, 31),
			want:      date(2028, time.February, 29),
		},
		{
			name:      "May 31 monthly gives Jun 30",
			frequency: FrequencyMonthly,
			start:     date(2026, time.May, 31),
			want:      date(2026, time.June, 30),
		},
		{
			name:      "Jun 30 monthly gives Jul 31",
			frequency: FrequencyMonthly,
			start:     date(2026, time.June, 30),
			want:      date(2026, time.July, 31),
		},
		{
			name:      "mid month day is kept",
			frequency: FrequencyMonthly,
			start:     date(2026, time.March, 15),
			want:      date(2026, time.April, 15),
		},
		{
			name:      "Nov 30 quarterly gives Feb 28",
			frequency: FrequencyQuarterly,
			start:     date(2026, time.November, 30),
			want:      date(2027, time.February, 28),
		},
		{
			name:      "Nov 30 quarterly gives Feb 29 in leap year",
			frequency: FrequencyQuarterly,
			start:     date(2027, time.November, 30),
			want:      date(2028, time.February, 29),
		},
		{
			name:      "Feb 29 yearly gives Feb 28",
			frequency: FrequencyYearly,
			start:     date(2028, time.February, 29),
			want:      date(2029, time.February, 28),
		},
		{
			name:      "Dec 31 monthly gives Jan 31 of next year",
			frequency: FrequencyMonthly,
			start:     date(2026, time.December, 31),
			want:      date(2027, time.January, 31),
		},
		{
			name:      "unknown frequency on Jan 31 clamps",
			frequency: "biweekly",
			start:     date(2027, time.January, 31),
			want:      date(2027, time.February, 28),
		},
		{
			name:      "weekly is unaffected by month end",
			frequency: FrequencyWeekly,
			start:     date(2026, time.January, 31),
			want:      date(2026, time.February, 7),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RecurringInvoice{
				Frequency:     tt.frequency,
				NextIssueDate: tt.start,
			}
			got := r.NextDate()
			if !got.Equal(tt.want) {
				t.Errorf("NextDate() = %v, want %v", got.Format("2006-01-02"), tt.want.Format("2006-01-02"))
			}
		})
	}
}

func TestRecurringExpense_NextDate(t *testing.T) {
	tests := []struct {
		name      string
		frequency string
		start     time.Time
		want      time.Time
	}{
		{
			name:      "weekly adds 7 days",
			frequency: "weekly",
			start:     date(2026, time.January, 1),
			want:      date(2026, time.January, 8),
		},
		{
			name:      "monthly adds 1 month",
			frequency: "monthly",
			start:     date(2026, time.January, 15),
			want:      date(2026, time.February, 15),
		},
		{
			name:      "quarterly adds 3 months",
			frequency: "quarterly",
			start:     date(2026, time.January, 1),
			want:      date(2026, time.April, 1),
		},
		{
			name:      "yearly adds 1 year",
			frequency: "yearly",
			start:     date(2026, time.March, 1),
			want:      date(2027, time.March, 1),
		},
		{
			name:      "unknown frequency defaults to monthly",
			frequency: "biweekly",
			start:     date(2026, time.January, 15),
			want:      date(2026, time.February, 15),
		},
		{
			name:      "Jan 31 monthly clamps to Feb 28",
			frequency: "monthly",
			start:     date(2027, time.January, 31),
			want:      date(2027, time.February, 28),
		},
		{
			name:      "Jan 31 monthly clamps to Feb 29 in leap year",
			frequency: "monthly",
			start:     date(2028, time.January, 31),
			want:      date(2028, time.February, 29),
		},
		{
			name:      "May 31 monthly gives Jun 30",
			frequency: "monthly",
			start:     date(2026, time.May, 31),
			want:      date(2026, time.June, 30),
		},
		{
			name:      "Jun 30 monthly gives Jul 31",
			frequency: "monthly",
			start:     date(2026, time.June, 30),
			want:      date(2026, time.July, 31),
		},
		{
			name:      "mid month day is kept",
			frequency: "monthly",
			start:     date(2026, time.March, 15),
			want:      date(2026, time.April, 15),
		},
		{
			name:      "Nov 30 quarterly gives Feb 28",
			frequency: "quarterly",
			start:     date(2026, time.November, 30),
			want:      date(2027, time.February, 28),
		},
		{
			name:      "Nov 30 quarterly gives Feb 29 in leap year",
			frequency: "quarterly",
			start:     date(2027, time.November, 30),
			want:      date(2028, time.February, 29),
		},
		{
			name:      "Feb 29 yearly gives Feb 28",
			frequency: "yearly",
			start:     date(2028, time.February, 29),
			want:      date(2029, time.February, 28),
		},
		{
			name:      "Dec 31 monthly gives Jan 31 of next year",
			frequency: "monthly",
			start:     date(2026, time.December, 31),
			want:      date(2027, time.January, 31),
		},
		{
			name:      "unknown frequency on Jan 31 clamps",
			frequency: "biweekly",
			start:     date(2027, time.January, 31),
			want:      date(2027, time.February, 28),
		},
		{
			name:      "weekly is unaffected by month end",
			frequency: "weekly",
			start:     date(2026, time.January, 31),
			want:      date(2026, time.February, 7),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RecurringExpense{
				Frequency:     tt.frequency,
				NextIssueDate: tt.start,
			}
			got := r.NextDate()
			if !got.Equal(tt.want) {
				t.Errorf("NextDate() = %v, want %v", got.Format("2006-01-02"), tt.want.Format("2006-01-02"))
			}
		})
	}
}
