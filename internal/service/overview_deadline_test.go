package service

import (
	"errors"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestOverviewDeadline(t *testing.T) {
	tests := []struct {
		name   string
		year   int
		method domain.FilingMethod
		want   time.Time
	}{
		{"2025 standard", 2025, domain.FilingMethodStandard, date(2026, time.May, 4)},
		{"2025 electronic", 2025, domain.FilingMethodElectronic, date(2026, time.June, 4)},
		{"2025 advisor", 2025, domain.FilingMethodAdvisor, date(2026, time.August, 3)},
		{"2025 unset", 2025, domain.FilingMethodUnset, date(2026, time.May, 4)},
		{"2024 standard", 2024, domain.FilingMethodStandard, date(2025, time.May, 2)},
		{"2024 electronic", 2024, domain.FilingMethodElectronic, date(2025, time.June, 2)},
		{"2024 advisor", 2024, domain.FilingMethodAdvisor, date(2025, time.August, 1)},
		{"2023 standard, Easter Monday return date", 2023, domain.FilingMethodStandard, date(2024, time.May, 2)},
		{"2026 standard", 2026, domain.FilingMethodStandard, date(2027, time.May, 3)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := OverviewDeadline(tt.year, tt.method)
			if err != nil {
				t.Fatalf("OverviewDeadline: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("OverviewDeadline(%d, %q) = %s, want %s", tt.year, tt.method, got.Format("2006-01-02"), tt.want.Format("2006-01-02"))
			}
		})
	}
}

func TestOverviewDeadline_InvalidMethod(t *testing.T) {
	_, err := OverviewDeadline(2025, domain.FilingMethod("paper"))
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

type advanceStartCase struct {
	name       string
	deadline   time.Time
	filedAt    *time.Time
	wantSocial int
	wantHealth int
}

func advanceStartCases() []advanceStartCase {
	at := func(year int, month time.Month, day, hour, min int) *time.Time {
		v := time.Date(year, month, day, hour, min, 0, 0, time.UTC)
		return &v
	}
	return []advanceStartCase{
		{"deadline 4 May", date(2026, time.May, 4), nil, 6, 5},
		{"deadline 4 June", date(2026, time.June, 4), nil, 7, 6},
		{"deadline 3 August", date(2026, time.August, 3), nil, 9, 8},
		{"filed early in March", date(2026, time.June, 4), at(2026, time.March, 15, 12, 0), 4, 3},
		{"filed late", date(2026, time.May, 4), at(2026, time.July, 10, 12, 0), 6, 5},
		{"filed on the deadline day", date(2026, time.May, 4), at(2026, time.May, 4, 12, 0), 6, 5},
		{"filed before the year ended", date(2026, time.May, 4), at(2025, time.December, 31, 12, 0), 2, 1},
		{"filed 00:30 on 1 June in Prague", date(2026, time.June, 4), at(2026, time.May, 31, 22, 30), 7, 6},
	}
}

func TestSocialAdvanceStartMonth(t *testing.T) {
	for _, tt := range advanceStartCases() {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SocialAdvanceStartMonth(2025, tt.deadline, tt.filedAt)
			if err != nil {
				t.Fatalf("SocialAdvanceStartMonth: %v", err)
			}
			if got != tt.wantSocial {
				t.Errorf("SocialAdvanceStartMonth = %d, want %d", got, tt.wantSocial)
			}
		})
	}
}

func TestHealthAdvanceStartMonth(t *testing.T) {
	for _, tt := range advanceStartCases() {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HealthAdvanceStartMonth(2025, tt.deadline, tt.filedAt)
			if err != nil {
				t.Fatalf("HealthAdvanceStartMonth: %v", err)
			}
			if got != tt.wantHealth {
				t.Errorf("HealthAdvanceStartMonth = %d, want %d", got, tt.wantHealth)
			}
		})
	}
}
