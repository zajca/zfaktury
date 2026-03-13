package service

import (
	"sort"
	"time"
)

// TaxDeadline represents a single tax deadline in the Czech tax calendar.
type TaxDeadline struct {
	Name        string
	Date        time.Time
	Type        string // "vat", "income_tax", "social", "health", "advance"
	Description string
}

// TaxCalendarService provides Czech tax calendar with holiday-aware deadline shifting.
type TaxCalendarService struct{}

// NewTaxCalendarService creates a new TaxCalendarService.
func NewTaxCalendarService() *TaxCalendarService {
	return &TaxCalendarService{}
}

// GetDeadlines returns all tax deadlines for the given year, shifted to business days.
func (s *TaxCalendarService) GetDeadlines(year int) []TaxDeadline {
	holidays := czechPublicHolidays(year)

	var deadlines []TaxDeadline

	// Monthly VAT: 25th of the following month for each month's VAT.
	// January VAT is due Feb 25, ..., December VAT is due Jan 25 of next year.
	monthNames := []string{
		"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}
	for m := 1; m <= 12; m++ {
		// VAT for month m is due on the 25th of month m+1.
		dueYear := year
		dueMonth := m + 1
		if dueMonth > 12 {
			dueMonth = 1
			dueYear = year + 1
		}
		// Build holidays for the due year (may differ from calendar year for December VAT).
		dueHolidays := holidays
		if dueYear != year {
			dueHolidays = czechPublicHolidays(dueYear)
		}
		date := s.nextBusinessDay(time.Date(dueYear, time.Month(dueMonth), 25, 0, 0, 0, 0, time.UTC), dueHolidays)
		deadlines = append(deadlines, TaxDeadline{
			Name:        "VAT return - " + monthNames[m-1],
			Date:        date,
			Type:        "vat",
			Description: "Monthly VAT return for " + monthNames[m-1] + " " + formatYear(year),
		})
	}

	// Income tax annual return: April 1.
	deadlines = append(deadlines, TaxDeadline{
		Name:        "Income tax return",
		Date:        s.nextBusinessDay(time.Date(year, time.April, 1, 0, 0, 0, 0, time.UTC), holidays),
		Type:        "income_tax",
		Description: "Annual income tax return filing deadline",
	})

	// Social insurance overview: May 2.
	deadlines = append(deadlines, TaxDeadline{
		Name:        "Social insurance overview",
		Date:        s.nextBusinessDay(time.Date(year, time.May, 2, 0, 0, 0, 0, time.UTC), holidays),
		Type:        "social",
		Description: "Annual social insurance overview filing deadline",
	})

	// Health insurance overview: May 2.
	deadlines = append(deadlines, TaxDeadline{
		Name:        "Health insurance overview",
		Date:        s.nextBusinessDay(time.Date(year, time.May, 2, 0, 0, 0, 0, time.UTC), holidays),
		Type:        "health",
		Description: "Annual health insurance overview filing deadline",
	})

	// Quarterly income tax advances.
	quarterDates := []struct {
		month time.Month
		day   int
		label string
	}{
		{time.March, 15, "Q1"},
		{time.June, 15, "Q2"},
		{time.September, 15, "Q3"},
		{time.December, 15, "Q4"},
	}
	for _, q := range quarterDates {
		deadlines = append(deadlines, TaxDeadline{
			Name:        "Income tax advance " + q.label,
			Date:        s.nextBusinessDay(time.Date(year, q.month, q.day, 0, 0, 0, 0, time.UTC), holidays),
			Type:        "advance",
			Description: "Quarterly income tax advance payment " + q.label,
		})
	}

	sort.Slice(deadlines, func(i, j int) bool {
		return deadlines[i].Date.Before(deadlines[j].Date)
	})

	return deadlines
}

// nextBusinessDay shifts a date forward if it falls on a weekend or Czech public holiday.
func (s *TaxCalendarService) nextBusinessDay(date time.Time, holidays map[time.Time]bool) time.Time {
	for date.Weekday() == time.Saturday || date.Weekday() == time.Sunday || holidays[date] {
		date = date.AddDate(0, 0, 1)
	}
	return date
}

// czechPublicHolidays returns a set of Czech public holidays for a given year (Law 245/2000 Sb.).
func czechPublicHolidays(year int) map[time.Time]bool {
	easter := easterSunday(year)
	goodFriday := easter.AddDate(0, 0, -2)
	easterMonday := easter.AddDate(0, 0, 1)

	holidays := map[time.Time]bool{
		time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC): true, // Restoration Day
		goodFriday:   true, // Good Friday
		easterMonday: true, // Easter Monday
		time.Date(year, time.May, 1, 0, 0, 0, 0, time.UTC):        true, // Labour Day
		time.Date(year, time.May, 8, 0, 0, 0, 0, time.UTC):        true, // Liberation Day
		time.Date(year, time.July, 5, 0, 0, 0, 0, time.UTC):       true, // SS. Cyril and Methodius
		time.Date(year, time.July, 6, 0, 0, 0, 0, time.UTC):       true, // Jan Hus Day
		time.Date(year, time.September, 28, 0, 0, 0, 0, time.UTC): true, // Czech Statehood Day
		time.Date(year, time.October, 28, 0, 0, 0, 0, time.UTC):   true, // Independent Czechoslovak State Day
		time.Date(year, time.November, 17, 0, 0, 0, 0, time.UTC):  true, // Struggle for Freedom and Democracy Day
		time.Date(year, time.December, 24, 0, 0, 0, 0, time.UTC):  true, // Christmas Eve
		time.Date(year, time.December, 25, 0, 0, 0, 0, time.UTC):  true, // Christmas Day
		time.Date(year, time.December, 26, 0, 0, 0, 0, time.UTC):  true, // St. Stephen's Day
	}
	return holidays
}

// easterSunday computes Easter Sunday for a given year using the Anonymous Gregorian algorithm (Computus).
func easterSunday(year int) time.Time {
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := ((h + l - 7*m + 114) % 31) + 1
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// formatYear returns the year as a string.
func formatYear(year int) string {
	return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006")
}
