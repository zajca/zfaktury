package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// TaxDeadline represents a single tax deadline in the Czech tax calendar.
type TaxDeadline struct {
	Name        string
	Date        time.Time
	Type        string // "vat", "income_tax", "social", "health", "advance"
	Description string
}

// TaxCalendarService provides Czech tax calendar with holiday-aware deadline shifting.
type TaxCalendarService struct {
	taxYearSettingsRepo repository.TaxYearSettingsRepo
}

// NewTaxCalendarService creates a new TaxCalendarService.
func NewTaxCalendarService(taxYearSettingsRepo repository.TaxYearSettingsRepo) *TaxCalendarService {
	return &TaxCalendarService{taxYearSettingsRepo: taxYearSettingsRepo}
}

// GetDeadlines returns all tax deadlines for the given calendar year, shifted
// to business days. The return and insurance overview deadlines follow the
// filing method stored for tax year year-1.
func (s *TaxCalendarService) GetDeadlines(ctx context.Context, year int) ([]TaxDeadline, error) {
	holidays := czechPublicHolidays(year)

	taxYear := year - 1
	method, err := s.filingMethod(ctx, taxYear)
	if err != nil {
		return nil, fmt.Errorf("listing tax deadlines for %d: %w", year, err)
	}
	// Validate the method before any deadline computation so an unknown stored
	// value fails here instead of leaking into a deadline text.
	methodNote, err := filingMethodNote(taxYear, method)
	if err != nil {
		return nil, fmt.Errorf("listing tax deadlines for %d: %w", year, err)
	}
	returnDate, err := IncomeTaxReturnDeadline(taxYear, method)
	if err != nil {
		return nil, fmt.Errorf("listing tax deadlines for %d: %w", year, err)
	}
	overviewDate, err := OverviewDeadline(taxYear, method)
	if err != nil {
		return nil, fmt.Errorf("listing tax deadlines for %d: %w", year, err)
	}

	var deadlines []TaxDeadline

	// Monthly VAT: 25th of the following month for each month's VAT.
	// January VAT is due Feb 25, ..., December VAT is due Jan 25 of next year.
	monthNames := []string{
		"leden", "únor", "březen", "duben", "květen", "červen",
		"červenec", "srpen", "září", "říjen", "listopad", "prosinec",
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
		monthYear := formatYear(year)
		deadlines = append(deadlines, TaxDeadline{
			Name:        fmt.Sprintf("Přiznání k DPH za %s %s", monthNames[m-1], monthYear),
			Date:        date,
			Type:        "vat",
			Description: fmt.Sprintf("Měsíční přiznání k DPH za %s %s", monthNames[m-1], monthYear),
		})
	}

	// Income tax annual return: deadline depends on the filing method of the tax year.
	deadlines = append(deadlines, TaxDeadline{
		Name:        "Přiznání k dani z příjmů",
		Date:        returnDate,
		Type:        "income_tax",
		Description: fmt.Sprintf("Termín podání přiznání k dani z příjmů za rok %d; %s", taxYear, methodNote),
	})

	// Social insurance overview: one month after the return deadline.
	deadlines = append(deadlines, TaxDeadline{
		Name:        "Přehled o příjmech a výdajích pro ČSSZ",
		Date:        overviewDate,
		Type:        "social",
		Description: fmt.Sprintf("Termín podání přehledu o příjmech a výdajích pro ČSSZ za rok %d; %s", taxYear, methodNote),
	})

	// Health insurance overview: one month after the return deadline.
	deadlines = append(deadlines, TaxDeadline{
		Name:        "Přehled pro zdravotní pojišťovnu",
		Date:        overviewDate,
		Type:        "health",
		Description: fmt.Sprintf("Termín podání přehledu pro zdravotní pojišťovnu za rok %d; %s", taxYear, methodNote),
	})

	// Quarterly income tax advances.
	quarterDates := []struct {
		month time.Month
		day   int
		label string
	}{
		{time.March, 15, "1. čtvrtletí"},
		{time.June, 15, "2. čtvrtletí"},
		{time.September, 15, "3. čtvrtletí"},
		{time.December, 15, "4. čtvrtletí"},
	}
	for _, q := range quarterDates {
		deadlines = append(deadlines, TaxDeadline{
			Name:        "Záloha na daň z příjmů – " + q.label,
			Date:        s.nextBusinessDay(time.Date(year, q.month, q.day, 0, 0, 0, 0, time.UTC), holidays),
			Type:        "advance",
			Description: fmt.Sprintf("Čtvrtletní záloha na daň z příjmů za %s %s", q.label, formatYear(year)),
		})
	}

	sort.Slice(deadlines, func(i, j int) bool {
		return deadlines[i].Date.Before(deadlines[j].Date)
	})

	return deadlines, nil
}

// filingMethod returns the filing method stored for the given tax year. A
// missing settings row counts as unset.
func (s *TaxCalendarService) filingMethod(ctx context.Context, taxYear int) (domain.FilingMethod, error) {
	tys, err := s.taxYearSettingsRepo.GetByYear(ctx, taxYear)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.FilingMethodUnset, nil
		}
		return "", fmt.Errorf("fetching tax year settings for %d: %w", taxYear, err)
	}
	return tys.FilingMethod, nil
}

// filingMethodNote describes the filing method behind the return and overview
// deadlines. An unset method states the assumption the deadline computation
// makes: filed by 1 April (on paper or electronically), shifted to the first
// following business day, and says where the method is set. An unknown stored
// value is an error, never shown.
func filingMethodNote(taxYear int, method domain.FilingMethod) (string, error) {
	if method == domain.FilingMethodUnset {
		return fmt.Sprintf("Termíny počítají s podáním přiznání do 1. dubna (papírově nebo elektronicky), případně do prvního následujícího pracovního dne, protože pro rok %d není nastaven způsob podání. Způsob podání lze nastavit v části Daň z příjmů → Zálohy.", taxYear), nil
	}
	label, err := filingMethodLabel(method)
	if err != nil {
		return "", fmt.Errorf("filing method for tax year %d: %w", taxYear, err)
	}
	return fmt.Sprintf("způsob podání přiznání za rok %d: %s", taxYear, label), nil
}

// filingMethodLabel maps a stored filing method to its readable Czech label.
func filingMethodLabel(method domain.FilingMethod) (string, error) {
	switch method {
	case domain.FilingMethodStandard:
		return "do 1. dubna (papírově nebo elektronicky)", nil
	case domain.FilingMethodElectronic:
		return "elektronicky po 1. dubnu", nil
	case domain.FilingMethodAdvisor:
		return "daňový poradce po 1. dubnu", nil
	default:
		return "", fmt.Errorf("unknown filing method: %w", domain.ErrInvalidInput)
	}
}

// nextBusinessDay shifts a date forward if it falls on a weekend or Czech public holiday.
func (s *TaxCalendarService) nextBusinessDay(date time.Time, holidays map[time.Time]bool) time.Time {
	return nextBusinessDayIn(date, holidays)
}

// nextBusinessDayIn shifts a date forward past weekends and the given holidays.
func nextBusinessDayIn(date time.Time, holidays map[time.Time]bool) time.Time {
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
