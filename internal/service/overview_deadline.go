package service

import (
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// pragueZone is the zone in which a filing timestamp is turned into a calendar date.
const pragueZone = "Europe/Prague"

// IncomeTaxReturnDeadline returns the last day for filing the income-tax
// return for taxYear: 3 months after the year, 4 months when filed
// electronically after that, 6 months via a tax advisor (§ 136 zákona č.
// 280/2009 Sb.), moved to the next working day (§ 33 odst. 4). An unset
// method counts as filed by 1 April, as ČSSZ does when the overview does not
// say otherwise.
func IncomeTaxReturnDeadline(taxYear int, method domain.FilingMethod) (time.Time, error) {
	if err := method.Validate(); err != nil {
		return time.Time{}, fmt.Errorf("computing income tax return deadline: %w", err)
	}

	month := time.April
	switch method {
	case domain.FilingMethodElectronic:
		month = time.May
	case domain.FilingMethodAdvisor:
		month = time.July
	}

	holidays := czechPublicHolidays(taxYear + 1)
	return nextBusinessDayIn(time.Date(taxYear+1, month, 1, 0, 0, 0, 0, time.UTC), holidays), nil
}

// OverviewDeadline returns the last day for the ČSSZ and health overviews for
// overviewYear: one month after the income-tax return deadline (§ 15 odst. 1
// zákona č. 589/1992 Sb., § 24 odst. 2 zákona č. 592/1992 Sb.), moved to the
// next working day (§ 33 odst. 4).
func OverviewDeadline(overviewYear int, method domain.FilingMethod) (time.Time, error) {
	taxReturn, err := IncomeTaxReturnDeadline(overviewYear, method)
	if err != nil {
		return time.Time{}, fmt.Errorf("computing overview deadline: %w", err)
	}

	holidays := czechPublicHolidays(overviewYear + 1)
	return nextBusinessDayIn(taxReturn.AddDate(0, 1, 0), holidays), nil
}

// SocialAdvanceStartMonth returns the month of overviewYear+1 for which the new
// social advance is first paid: the month after the month the overview was
// filed, or had to be filed when that is earlier (§ 14 odst. 6 zákona č. 589/1992 Sb.).
func SocialAdvanceStartMonth(overviewYear int, deadline time.Time, filedAt *time.Time) (int, error) {
	ref, err := advanceReferenceDate(overviewYear, deadline, filedAt)
	if err != nil {
		return 0, err
	}
	return int(ref.Month()) + 1, nil
}

// HealthAdvanceStartMonth returns the month of overviewYear+1 for which the new
// health advance is first paid: the month the overview was filed, or had to be
// filed when that is earlier (§ 8 odst. 2 zákona č. 592/1992 Sb.).
func HealthAdvanceStartMonth(overviewYear int, deadline time.Time, filedAt *time.Time) (int, error) {
	ref, err := advanceReferenceDate(overviewYear, deadline, filedAt)
	if err != nil {
		return 0, err
	}
	return int(ref.Month()), nil
}

// advanceReferenceDate returns the earlier of the deadline and the Prague
// calendar date of the filing, never before 1 January of overviewYear+1 (an
// overview cannot be filed before its year ends).
func advanceReferenceDate(overviewYear int, deadline time.Time, filedAt *time.Time) (time.Time, error) {
	ref := time.Date(deadline.Year(), deadline.Month(), deadline.Day(), 0, 0, 0, 0, time.UTC)

	if filedAt != nil {
		prague, err := time.LoadLocation(pragueZone)
		if err != nil {
			return time.Time{}, fmt.Errorf("loading time zone %s: %w", pragueZone, err)
		}
		local := filedAt.In(prague)
		filed := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
		if filed.Before(ref) {
			ref = filed
		}
	}

	if yearStart := time.Date(overviewYear+1, time.January, 1, 0, 0, 0, 0, time.UTC); ref.Before(yearStart) {
		ref = yearStart
	}
	return ref, nil
}
