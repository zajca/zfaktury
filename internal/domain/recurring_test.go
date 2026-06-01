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
			name:      "May 31 monthly does not skip June",
			frequency: FrequencyMonthly,
			start:     date(2026, time.May, 31),
			want:      date(2026, time.June, 30), // clamped to month-end, June not skipped
		},
		{
			name:      "end-of-month preserved June 30 monthly",
			frequency: FrequencyMonthly,
			start:     date(2026, time.June, 30),
			want:      date(2026, time.July, 31), // last day of June -> last day of July
		},
		{
			name:      "Jan 31 monthly clamps to Feb 28",
			frequency: FrequencyMonthly,
			start:     date(2026, time.January, 31),
			want:      date(2026, time.February, 28), // 2026 not a leap year
		},
		{
			name:      "end-of-month preserved Feb 28 monthly",
			frequency: FrequencyMonthly,
			start:     date(2026, time.February, 28),
			want:      date(2026, time.March, 31), // last day of Feb -> last day of Mar
		},
		{
			name:      "Jan 31 quarterly clamps to Apr 30",
			frequency: FrequencyQuarterly,
			start:     date(2026, time.January, 31),
			want:      date(2026, time.April, 30),
		},
		{
			name:      "leap day yearly clamps to Feb 28",
			frequency: FrequencyYearly,
			start:     date(2024, time.February, 29),
			want:      date(2025, time.February, 28),
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
			name:      "May 31 monthly does not skip June",
			frequency: "monthly",
			start:     date(2026, time.May, 31),
			want:      date(2026, time.June, 30), // clamped to month-end, June not skipped
		},
		{
			name:      "end-of-month preserved June 30 monthly",
			frequency: "monthly",
			start:     date(2026, time.June, 30),
			want:      date(2026, time.July, 31), // last day of June -> last day of July
		},
		{
			name:      "Jan 31 monthly clamps to Feb 28",
			frequency: "monthly",
			start:     date(2026, time.January, 31),
			want:      date(2026, time.February, 28), // 2026 not a leap year
		},
		{
			name:      "end-of-month preserved Feb 28 monthly",
			frequency: "monthly",
			start:     date(2026, time.February, 28),
			want:      date(2026, time.March, 31), // last day of Feb -> last day of Mar
		},
		{
			name:      "Jan 31 quarterly clamps to Apr 30",
			frequency: "quarterly",
			start:     date(2026, time.January, 31),
			want:      date(2026, time.April, 30),
		},
		{
			name:      "leap day yearly clamps to Feb 28",
			frequency: "yearly",
			start:     date(2024, time.February, 29),
			want:      date(2025, time.February, 28),
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
