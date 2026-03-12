package service

import (
	"testing"
	"time"
)

func TestTaxCalendarService_GetDeadlines_Count(t *testing.T) {
	svc := NewTaxCalendarService()

	deadlines := svc.GetDeadlines(2026)

	// 12 VAT + 1 income tax + 1 social + 1 health + 4 quarterly advances = 19
	expected := 19
	if len(deadlines) != expected {
		t.Errorf("GetDeadlines() returned %d deadlines, want %d", len(deadlines), expected)
	}
}

func TestTaxCalendarService_GetDeadlines_Sorted(t *testing.T) {
	svc := NewTaxCalendarService()

	deadlines := svc.GetDeadlines(2026)

	for i := 1; i < len(deadlines); i++ {
		if deadlines[i].Date.Before(deadlines[i-1].Date) {
			t.Errorf("deadlines not sorted: [%d] %s (%s) is before [%d] %s (%s)",
				i, deadlines[i].Name, deadlines[i].Date.Format("2006-01-02"),
				i-1, deadlines[i-1].Name, deadlines[i-1].Date.Format("2006-01-02"))
		}
	}
}

func TestTaxCalendarService_GetDeadlines_WeekendShift(t *testing.T) {
	svc := NewTaxCalendarService()

	deadlines := svc.GetDeadlines(2026)

	for _, d := range deadlines {
		wd := d.Date.Weekday()
		if wd == time.Saturday || wd == time.Sunday {
			t.Errorf("deadline %q falls on %s (%s), expected business day",
				d.Name, wd, d.Date.Format("2006-01-02"))
		}
	}
}

func TestTaxCalendarService_GetDeadlines_Types(t *testing.T) {
	svc := NewTaxCalendarService()

	deadlines := svc.GetDeadlines(2026)

	typeCounts := make(map[string]int)
	for _, d := range deadlines {
		typeCounts[d.Type]++
	}

	expectedTypes := map[string]int{
		"vat":        12,
		"income_tax": 1,
		"social":     1,
		"health":     1,
		"advance":    4,
	}

	for typ, wantCount := range expectedTypes {
		if got := typeCounts[typ]; got != wantCount {
			t.Errorf("type %q count = %d, want %d", typ, got, wantCount)
		}
	}

	// Verify no unexpected types.
	for typ := range typeCounts {
		if _, ok := expectedTypes[typ]; !ok {
			t.Errorf("unexpected deadline type %q", typ)
		}
	}
}

func TestEasterSunday(t *testing.T) {
	tests := []struct {
		year int
		want time.Time
	}{
		{2024, time.Date(2024, time.March, 31, 0, 0, 0, 0, time.UTC)},
		{2025, time.Date(2025, time.April, 20, 0, 0, 0, 0, time.UTC)},
		{2026, time.Date(2026, time.April, 5, 0, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(time.Date(tt.year, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006"), func(t *testing.T) {
			got := easterSunday(tt.year)
			if !got.Equal(tt.want) {
				t.Errorf("easterSunday(%d) = %s, want %s", tt.year, got.Format("2006-01-02"), tt.want.Format("2006-01-02"))
			}
		})
	}
}

func TestCzechPublicHolidays(t *testing.T) {
	holidays := czechPublicHolidays(2026)

	// Czech Republic has 13 public holidays per year.
	if len(holidays) != 13 {
		t.Errorf("czechPublicHolidays(2026) returned %d holidays, want 13", len(holidays))
	}

	// Verify some specific fixed holidays exist.
	expectedDates := []time.Time{
		time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),   // Restoration Day
		time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC),       // Labour Day
		time.Date(2026, time.May, 8, 0, 0, 0, 0, time.UTC),       // Liberation Day
		time.Date(2026, time.July, 5, 0, 0, 0, 0, time.UTC),      // SS. Cyril and Methodius
		time.Date(2026, time.July, 6, 0, 0, 0, 0, time.UTC),      // Jan Hus Day
		time.Date(2026, time.September, 28, 0, 0, 0, 0, time.UTC), // Czech Statehood Day
		time.Date(2026, time.October, 28, 0, 0, 0, 0, time.UTC),  // Independent Czechoslovak State Day
		time.Date(2026, time.November, 17, 0, 0, 0, 0, time.UTC), // Struggle for Freedom and Democracy Day
		time.Date(2026, time.December, 24, 0, 0, 0, 0, time.UTC), // Christmas Eve
		time.Date(2026, time.December, 25, 0, 0, 0, 0, time.UTC), // Christmas Day
		time.Date(2026, time.December, 26, 0, 0, 0, 0, time.UTC), // St. Stephen's Day
	}

	for _, d := range expectedDates {
		if !holidays[d] {
			t.Errorf("expected holiday %s not found", d.Format("2006-01-02"))
		}
	}

	// Verify Easter-based holidays for 2026 (Easter Sunday = April 5).
	goodFriday := time.Date(2026, time.April, 3, 0, 0, 0, 0, time.UTC)
	easterMonday := time.Date(2026, time.April, 6, 0, 0, 0, 0, time.UTC)
	if !holidays[goodFriday] {
		t.Errorf("Good Friday 2026 (%s) not found in holidays", goodFriday.Format("2006-01-02"))
	}
	if !holidays[easterMonday] {
		t.Errorf("Easter Monday 2026 (%s) not found in holidays", easterMonday.Format("2006-01-02"))
	}
}

func TestNextBusinessDay_Weekend(t *testing.T) {
	svc := NewTaxCalendarService()
	holidays := make(map[time.Time]bool)

	// 2026-01-03 is a Saturday.
	saturday := time.Date(2026, time.January, 3, 0, 0, 0, 0, time.UTC)
	got := svc.nextBusinessDay(saturday, holidays)
	// Should shift to Monday 2026-01-05.
	want := time.Date(2026, time.January, 5, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("nextBusinessDay(%s) = %s, want %s", saturday.Format("2006-01-02"), got.Format("2006-01-02"), want.Format("2006-01-02"))
	}

	// 2026-01-04 is a Sunday.
	sunday := time.Date(2026, time.January, 4, 0, 0, 0, 0, time.UTC)
	got = svc.nextBusinessDay(sunday, holidays)
	if !got.Equal(want) {
		t.Errorf("nextBusinessDay(%s) = %s, want %s", sunday.Format("2006-01-02"), got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}

func TestNextBusinessDay_Holiday(t *testing.T) {
	svc := NewTaxCalendarService()

	// 2026-01-01 is a Thursday (public holiday).
	holidays := map[time.Time]bool{
		time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC): true,
	}
	date := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	got := svc.nextBusinessDay(date, holidays)
	// Should shift to Friday 2026-01-02.
	want := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("nextBusinessDay(%s) = %s, want %s", date.Format("2006-01-02"), got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}

func TestNextBusinessDay_HolidayBeforeWeekend(t *testing.T) {
	svc := NewTaxCalendarService()

	// Create a scenario: Friday is a holiday, so it should shift to Monday.
	friday := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC) // 2026-01-02 is Friday
	holidays := map[time.Time]bool{
		friday: true,
	}
	got := svc.nextBusinessDay(friday, holidays)
	// Friday is holiday -> Saturday -> Sunday -> Monday = 2026-01-05
	want := time.Date(2026, time.January, 5, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("nextBusinessDay(%s with holiday) = %s, want %s", friday.Format("2006-01-02"), got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}

func TestNextBusinessDay_AlreadyBusinessDay(t *testing.T) {
	svc := NewTaxCalendarService()
	holidays := make(map[time.Time]bool)

	// 2026-01-05 is a Monday (not a holiday).
	monday := time.Date(2026, time.January, 5, 0, 0, 0, 0, time.UTC)
	got := svc.nextBusinessDay(monday, holidays)
	if !got.Equal(monday) {
		t.Errorf("nextBusinessDay(%s) = %s, want same date", monday.Format("2006-01-02"), got.Format("2006-01-02"))
	}
}
