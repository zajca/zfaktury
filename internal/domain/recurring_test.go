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
			name:      "end of month rollover Jan 31 monthly",
			frequency: FrequencyMonthly,
			start:     date(2026, time.January, 31),
			want:      date(2026, time.March, 3), // Go normalizes Feb 31 -> Mar 3
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
			name:      "end of month rollover Jan 31 monthly",
			frequency: "monthly",
			start:     date(2026, time.January, 31),
			want:      date(2026, time.March, 3), // Go normalizes Feb 31 -> Mar 3
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
