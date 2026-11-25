package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

// stubTaxYearSettingsRepo is a small in-test TaxYearSettingsRepo.
type stubTaxYearSettingsRepo struct {
	getByYear func(ctx context.Context, year int) (*domain.TaxYearSettings, error)
}

func (r stubTaxYearSettingsRepo) GetByYear(ctx context.Context, year int) (*domain.TaxYearSettings, error) {
	return r.getByYear(ctx, year)
}

func (r stubTaxYearSettingsRepo) Upsert(ctx context.Context, tys *domain.TaxYearSettings) error {
	return nil
}

// notFoundTaxYearSettingsRepo simulates a year with no stored settings row.
func notFoundTaxYearSettingsRepo() stubTaxYearSettingsRepo {
	return stubTaxYearSettingsRepo{getByYear: func(ctx context.Context, year int) (*domain.TaxYearSettings, error) {
		return nil, domain.ErrNotFound
	}}
}

// calendarTestRepo returns a real TaxYearSettingsRepository on in-memory SQLite.
func calendarTestRepo(t *testing.T) repository.TaxYearSettingsRepo {
	t.Helper()
	db := testutil.NewTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	return repository.NewTaxYearSettingsRepository(db)
}

func findDeadlineByType(deadlines []TaxDeadline, typ string) *TaxDeadline {
	for i := range deadlines {
		if deadlines[i].Type == typ {
			return &deadlines[i]
		}
	}
	return nil
}

func TestTaxCalendarService_GetDeadlines_Count(t *testing.T) {
	svc := NewTaxCalendarService(notFoundTaxYearSettingsRepo())

	deadlines, err := svc.GetDeadlines(context.Background(), 2026)
	if err != nil {
		t.Fatalf("GetDeadlines: %v", err)
	}

	// 12 VAT + 1 income tax + 1 social + 1 health + 4 quarterly advances = 19
	expected := 19
	if len(deadlines) != expected {
		t.Errorf("GetDeadlines() returned %d deadlines, want %d", len(deadlines), expected)
	}
}

func TestTaxCalendarService_GetDeadlines_Sorted(t *testing.T) {
	svc := NewTaxCalendarService(notFoundTaxYearSettingsRepo())

	deadlines, err := svc.GetDeadlines(context.Background(), 2026)
	if err != nil {
		t.Fatalf("GetDeadlines: %v", err)
	}

	for i := 1; i < len(deadlines); i++ {
		if deadlines[i].Date.Before(deadlines[i-1].Date) {
			t.Errorf("deadlines not sorted: [%d] %s (%s) is before [%d] %s (%s)",
				i, deadlines[i].Name, deadlines[i].Date.Format("2006-01-02"),
				i-1, deadlines[i-1].Name, deadlines[i-1].Date.Format("2006-01-02"))
		}
	}
}

func TestTaxCalendarService_GetDeadlines_WeekendShift(t *testing.T) {
	svc := NewTaxCalendarService(notFoundTaxYearSettingsRepo())

	deadlines, err := svc.GetDeadlines(context.Background(), 2026)
	if err != nil {
		t.Fatalf("GetDeadlines: %v", err)
	}

	for _, d := range deadlines {
		wd := d.Date.Weekday()
		if wd == time.Saturday || wd == time.Sunday {
			t.Errorf("deadline %q falls on %s (%s), expected business day",
				d.Name, wd, d.Date.Format("2006-01-02"))
		}
	}
}

func TestTaxCalendarService_GetDeadlines_Types(t *testing.T) {
	svc := NewTaxCalendarService(notFoundTaxYearSettingsRepo())

	deadlines, err := svc.GetDeadlines(context.Background(), 2026)
	if err != nil {
		t.Fatalf("GetDeadlines: %v", err)
	}

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
		time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),    // Restoration Day
		time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC),        // Labour Day
		time.Date(2026, time.May, 8, 0, 0, 0, 0, time.UTC),        // Liberation Day
		time.Date(2026, time.July, 5, 0, 0, 0, 0, time.UTC),       // SS. Cyril and Methodius
		time.Date(2026, time.July, 6, 0, 0, 0, 0, time.UTC),       // Jan Hus Day
		time.Date(2026, time.September, 28, 0, 0, 0, 0, time.UTC), // Czech Statehood Day
		time.Date(2026, time.October, 28, 0, 0, 0, 0, time.UTC),   // Independent Czechoslovak State Day
		time.Date(2026, time.November, 17, 0, 0, 0, 0, time.UTC),  // Struggle for Freedom and Democracy Day
		time.Date(2026, time.December, 24, 0, 0, 0, 0, time.UTC),  // Christmas Eve
		time.Date(2026, time.December, 25, 0, 0, 0, 0, time.UTC),  // Christmas Day
		time.Date(2026, time.December, 26, 0, 0, 0, 0, time.UTC),  // St. Stephen's Day
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

// TestTaxCalendarService_GetDeadlines_FilingMethod checks that for calendar
// year 2026 the return and overview deadlines follow the filing method stored
// for tax year 2025.
func TestTaxCalendarService_GetDeadlines_FilingMethod(t *testing.T) {
	tests := []struct {
		name         string
		seed         *domain.TaxYearSettings // nil = no settings row for 2025
		wantReturn   time.Time
		wantOverview time.Time
		wantNote     string
	}{
		{"missing settings row", nil, date(2026, time.April, 1), date(2026, time.May, 4), "bez daňového poradce"},
		{"unset method", &domain.TaxYearSettings{Year: 2025, FilingMethod: domain.FilingMethodUnset}, date(2026, time.April, 1), date(2026, time.May, 4), "bez daňového poradce"},
		{"standard", &domain.TaxYearSettings{Year: 2025, FilingMethod: domain.FilingMethodStandard}, date(2026, time.April, 1), date(2026, time.May, 4), "způsob podání přiznání za rok 2025: do 1. dubna (papírově nebo elektronicky)"},
		{"electronic, 1 May is a holiday", &domain.TaxYearSettings{Year: 2025, FilingMethod: domain.FilingMethodElectronic}, date(2026, time.May, 4), date(2026, time.June, 4), "způsob podání přiznání za rok 2025: elektronicky po 1. dubnu"},
		{"advisor", &domain.TaxYearSettings{Year: 2025, FilingMethod: domain.FilingMethodAdvisor}, date(2026, time.July, 1), date(2026, time.August, 3), "způsob podání přiznání za rok 2025: daňový poradce po 1. dubnu"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := calendarTestRepo(t)
			if tt.seed != nil {
				if err := repo.Upsert(context.Background(), tt.seed); err != nil {
					t.Fatalf("seeding tax year settings: %v", err)
				}
			}

			svc := NewTaxCalendarService(repo)
			deadlines, err := svc.GetDeadlines(context.Background(), 2026)
			if err != nil {
				t.Fatalf("GetDeadlines: %v", err)
			}

			returnDeadline := findDeadlineByType(deadlines, "income_tax")
			social := findDeadlineByType(deadlines, "social")
			health := findDeadlineByType(deadlines, "health")
			if returnDeadline == nil || social == nil || health == nil {
				t.Fatalf("missing income_tax/social/health entries in %v", deadlines)
			}

			if !returnDeadline.Date.Equal(tt.wantReturn) {
				t.Errorf("return date = %s, want %s", returnDeadline.Date.Format("2006-01-02"), tt.wantReturn.Format("2006-01-02"))
			}
			if !social.Date.Equal(tt.wantOverview) {
				t.Errorf("social overview date = %s, want %s", social.Date.Format("2006-01-02"), tt.wantOverview.Format("2006-01-02"))
			}
			if !health.Date.Equal(tt.wantOverview) {
				t.Errorf("health overview date = %s, want %s", health.Date.Format("2006-01-02"), tt.wantOverview.Format("2006-01-02"))
			}
			for _, d := range []*TaxDeadline{returnDeadline, social, health} {
				if !strings.Contains(d.Description, tt.wantNote) {
					t.Errorf("%s description = %q, want it to contain %q", d.Name, d.Description, tt.wantNote)
				}
				// The raw stored filing method value must never appear in the text.
				if tt.seed != nil && tt.seed.FilingMethod != domain.FilingMethodUnset {
					if raw := string(tt.seed.FilingMethod); strings.Contains(d.Description, raw) {
						t.Errorf("%s description = %q, must not contain the raw filing method %q", d.Name, d.Description, raw)
					}
				}
			}
		})
	}
}

// TestTaxCalendarService_GetDeadlines_CzechTexts checks that every deadline
// name and description produced for a calendar year is Czech.
func TestTaxCalendarService_GetDeadlines_CzechTexts(t *testing.T) {
	svc := NewTaxCalendarService(notFoundTaxYearSettingsRepo())

	deadlines, err := svc.GetDeadlines(context.Background(), 2026)
	if err != nil {
		t.Fatalf("GetDeadlines: %v", err)
	}

	monthNames := []string{
		"leden", "únor", "březen", "duben", "květen", "červen",
		"červenec", "srpen", "září", "říjen", "listopad", "prosinec",
	}
	wantNames := make(map[string]bool, 19)
	for _, m := range monthNames {
		wantNames[fmt.Sprintf("Přiznání k DPH za %s 2026", m)] = true
	}
	wantNames["Přiznání k dani z příjmů"] = true
	wantNames["Přehled o příjmech a výdajích pro ČSSZ"] = true
	wantNames["Přehled pro zdravotní pojišťovnu"] = true
	for _, q := range []string{"1. čtvrtletí", "2. čtvrtletí", "3. čtvrtletí", "4. čtvrtletí"} {
		wantNames["Záloha na daň z příjmů – "+q] = true
	}

	gotNames := make(map[string]bool, len(deadlines))
	for _, d := range deadlines {
		gotNames[d.Name] = true
	}
	for want := range wantNames {
		if !gotNames[want] {
			t.Errorf("missing deadline name %q", want)
		}
	}
	for got := range gotNames {
		if !wantNames[got] {
			t.Errorf("unexpected deadline name %q", got)
		}
	}

	// Descriptions: Czech wording, month in genitive with the year.
	vat := findDeadlineByName(deadlines, "Přiznání k DPH za leden 2026")
	if vat == nil {
		t.Fatalf("missing January VAT deadline")
	}
	if want := "Měsíční přiznání k DPH za leden 2026"; vat.Description != want {
		t.Errorf("VAT description = %q, want %q", vat.Description, want)
	}
	if want := "Termín podání přiznání k dani z příjmů za rok 2025"; !strings.Contains(findDeadlineByType(deadlines, "income_tax").Description, want) {
		t.Errorf("income tax description = %q, want it to contain %q", findDeadlineByType(deadlines, "income_tax").Description, want)
	}
	if want := "Termín podání přehledu o příjmech a výdajích pro ČSSZ za rok 2025"; !strings.Contains(findDeadlineByType(deadlines, "social").Description, want) {
		t.Errorf("social description = %q, want it to contain %q", findDeadlineByType(deadlines, "social").Description, want)
	}
	if want := "Termín podání přehledu pro zdravotní pojišťovnu za rok 2025"; !strings.Contains(findDeadlineByType(deadlines, "health").Description, want) {
		t.Errorf("health description = %q, want it to contain %q", findDeadlineByType(deadlines, "health").Description, want)
	}
	advance := findDeadlineByName(deadlines, "Záloha na daň z příjmů – 1. čtvrtletí")
	if advance == nil {
		t.Fatalf("missing Q1 advance deadline")
	}
	if want := "Čtvrtletní záloha na daň z příjmů za 1. čtvrtletí 2026"; advance.Description != want {
		t.Errorf("advance description = %q, want %q", advance.Description, want)
	}

	// No English leftovers anywhere.
	for _, d := range deadlines {
		for _, en := range []string{"VAT return", "Income tax", "Social insurance", "Health insurance", "Monthly VAT", "filing method", "Quarterly income"} {
			if strings.Contains(d.Name, en) || strings.Contains(d.Description, en) {
				t.Errorf("deadline %q still contains English %q", d.Name, en)
			}
		}
	}
}

func findDeadlineByName(deadlines []TaxDeadline, name string) *TaxDeadline {
	for i := range deadlines {
		if deadlines[i].Name == name {
			return &deadlines[i]
		}
	}
	return nil
}

// TestTaxCalendarService_GetDeadlines_UnsetNoteWhenDateMoved checks the note
// when no filing method is set for a tax year whose 1 April was shifted (2023
// tax year: 1 April 2024 was Easter Monday, the shown date is 2 April 2024).
func TestTaxCalendarService_GetDeadlines_UnsetNoteWhenDateMoved(t *testing.T) {
	svc := NewTaxCalendarService(notFoundTaxYearSettingsRepo())

	deadlines, err := svc.GetDeadlines(context.Background(), 2024)
	if err != nil {
		t.Fatalf("GetDeadlines: %v", err)
	}

	returnDeadline := findDeadlineByType(deadlines, "income_tax")
	if returnDeadline == nil {
		t.Fatalf("missing income_tax deadline in %v", deadlines)
	}
	if want := date(2024, time.April, 2); !returnDeadline.Date.Equal(want) {
		t.Errorf("return date = %s, want %s (1 April 2024 was Easter Monday)", returnDeadline.Date.Format("2006-01-02"), want.Format("2006-01-02"))
	}

	for _, d := range []*TaxDeadline{returnDeadline, findDeadlineByType(deadlines, "social"), findDeadlineByType(deadlines, "health")} {
		if !strings.Contains(d.Description, "papírově nebo elektronicky bez daňového poradce") {
			t.Errorf("%s description = %q, want it to state the paper/electronic-without-advisor assumption", d.Name, d.Description)
		}
		if !strings.Contains(d.Description, "první následující pracovní den") {
			t.Errorf("%s description = %q, want it to mention the shift to the next business day instead of hardcoding 1 April", d.Name, d.Description)
		}
		if !strings.Contains(d.Description, "Daňové zálohy a nastavení") {
			t.Errorf("%s description = %q, want it to name the settings page „Daňové zálohy a nastavení“", d.Name, d.Description)
		}
		if strings.Contains(d.Description, "filing by 1 April") {
			t.Errorf("%s description = %q, must not claim the dates assume filing by 1 April", d.Name, d.Description)
		}
	}
}

// TestTaxCalendarService_GetDeadlines_UnknownFilingMethod checks that an
// unknown stored filing method fails with ErrInvalidInput and never leaks the
// raw value.
func TestTaxCalendarService_GetDeadlines_UnknownFilingMethod(t *testing.T) {
	svc := NewTaxCalendarService(stubTaxYearSettingsRepo{getByYear: func(ctx context.Context, year int) (*domain.TaxYearSettings, error) {
		return &domain.TaxYearSettings{Year: year, FilingMethod: domain.FilingMethod("paper")}, nil
	}})

	_, err := svc.GetDeadlines(context.Background(), 2026)
	if err == nil {
		t.Fatal("GetDeadlines returned nil error, want ErrInvalidInput for an unknown filing method")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("error = %v, want it to wrap domain.ErrInvalidInput", err)
	}
	if strings.Contains(err.Error(), "paper") {
		t.Errorf("error = %q, must not leak the raw filing method value", err)
	}
}

// TestTaxCalendarService_GetDeadlines_RepoError checks that a repo error other
// than not-found is returned wrapped.
func TestTaxCalendarService_GetDeadlines_RepoError(t *testing.T) {
	boom := errors.New("database is down")
	svc := NewTaxCalendarService(stubTaxYearSettingsRepo{getByYear: func(ctx context.Context, year int) (*domain.TaxYearSettings, error) {
		return nil, boom
	}})

	_, err := svc.GetDeadlines(context.Background(), 2026)
	if err == nil {
		t.Fatal("GetDeadlines returned nil error, want the wrapped repo error")
	}
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want it to wrap %v", err, boom)
	}
	if !strings.Contains(err.Error(), "listing tax deadlines") {
		t.Errorf("error = %q, want it wrapped with a verb", err)
	}
}

// TestIncomeTaxReturnDeadline covers the extracted return-deadline helper.
func TestIncomeTaxReturnDeadline(t *testing.T) {
	tests := []struct {
		name   string
		year   int
		method domain.FilingMethod
		want   time.Time
	}{
		{"2025 standard", 2025, domain.FilingMethodStandard, date(2026, time.April, 1)},
		{"2025 electronic, 1 May is a holiday", 2025, domain.FilingMethodElectronic, date(2026, time.May, 4)},
		{"2025 advisor", 2025, domain.FilingMethodAdvisor, date(2026, time.July, 1)},
		{"2025 unset", 2025, domain.FilingMethodUnset, date(2026, time.April, 1)},
		{"2023 standard, 1 April is Easter Monday", 2023, domain.FilingMethodStandard, date(2024, time.April, 2)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IncomeTaxReturnDeadline(tt.year, tt.method)
			if err != nil {
				t.Fatalf("IncomeTaxReturnDeadline: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("IncomeTaxReturnDeadline(%d, %q) = %s, want %s", tt.year, tt.method, got.Format("2006-01-02"), tt.want.Format("2006-01-02"))
			}
		})
	}
}

func TestNextBusinessDay_Weekend(t *testing.T) {
	svc := NewTaxCalendarService(notFoundTaxYearSettingsRepo())
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
	svc := NewTaxCalendarService(notFoundTaxYearSettingsRepo())

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
	svc := NewTaxCalendarService(notFoundTaxYearSettingsRepo())

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
	svc := NewTaxCalendarService(notFoundTaxYearSettingsRepo())
	holidays := make(map[time.Time]bool)

	// 2026-01-05 is a Monday (not a holiday).
	monday := time.Date(2026, time.January, 5, 0, 0, 0, 0, time.UTC)
	got := svc.nextBusinessDay(monday, holidays)
	if !got.Equal(monday) {
		t.Errorf("nextBusinessDay(%s) = %s, want same date", monday.Format("2006-01-02"), got.Format("2006-01-02"))
	}
}
