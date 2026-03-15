package calc

import (
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

func TestGetTaxConstants_KnownYears(t *testing.T) {
	for _, year := range []int{2024, 2025, 2026} {
		c, err := GetTaxConstants(year)
		if err != nil {
			t.Fatalf("GetTaxConstants(%d) error: %v", year, err)
		}
		if c.ProgressiveThreshold == 0 {
			t.Errorf("GetTaxConstants(%d): ProgressiveThreshold should not be 0", year)
		}
		if c.BasicCredit == 0 {
			t.Errorf("GetTaxConstants(%d): BasicCredit should not be 0", year)
		}
		if c.SocialRate != 292 {
			t.Errorf("GetTaxConstants(%d): SocialRate = %d, want 292", year, c.SocialRate)
		}
		if c.HealthRate != 135 {
			t.Errorf("GetTaxConstants(%d): HealthRate = %d, want 135", year, c.HealthRate)
		}
		if len(c.FlatRateCaps) == 0 {
			t.Errorf("GetTaxConstants(%d): FlatRateCaps should not be empty", year)
		}
	}
}

func TestGetTaxConstants_2024Values(t *testing.T) {
	c, _ := GetTaxConstants(2024)
	if c.BasicCredit != domain.NewAmount(30_840, 0) {
		t.Errorf("2024 BasicCredit = %d, want %d", c.BasicCredit, domain.NewAmount(30_840, 0))
	}
	if c.SocialMinMonthly != domain.NewAmount(11_024, 0) {
		t.Errorf("2024 SocialMinMonthly = %d, want %d", c.SocialMinMonthly, domain.NewAmount(11_024, 0))
	}
	if c.SecurityExemptionLimit != 0 {
		t.Errorf("2024 SecurityExemptionLimit = %d, want 0 (no limit)", c.SecurityExemptionLimit)
	}
}

func TestGetTaxConstants_2025Values(t *testing.T) {
	c, _ := GetTaxConstants(2025)
	if c.SocialMinMonthly != domain.NewAmount(11_584, 0) {
		t.Errorf("2025 SocialMinMonthly = %d, want %d", c.SocialMinMonthly, domain.NewAmount(11_584, 0))
	}
	if c.HealthMinMonthly != domain.NewAmount(10_874, 0) {
		t.Errorf("2025 HealthMinMonthly = %d, want %d", c.HealthMinMonthly, domain.NewAmount(10_874, 0))
	}
	if c.SecurityExemptionLimit != domain.NewAmount(100_000_000, 0) {
		t.Errorf("2025 SecurityExemptionLimit = %d, want %d", c.SecurityExemptionLimit, domain.NewAmount(100_000_000, 0))
	}
}

func TestGetTaxConstants_2026Values(t *testing.T) {
	c, _ := GetTaxConstants(2026)
	if c.SocialMinMonthly != domain.NewAmount(12_139, 0) {
		t.Errorf("2026 SocialMinMonthly = %d, want %d", c.SocialMinMonthly, domain.NewAmount(12_139, 0))
	}
	if c.HealthMinMonthly != domain.NewAmount(11_396, 0) {
		t.Errorf("2026 HealthMinMonthly = %d, want %d", c.HealthMinMonthly, domain.NewAmount(11_396, 0))
	}
}

func TestGetTaxConstants_UnknownYear(t *testing.T) {
	_, err := GetTaxConstants(1999)
	if err == nil {
		t.Fatal("expected error for year 1999")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}

	_, err = GetTaxConstants(9999)
	if err == nil {
		t.Fatal("expected error for year 9999")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetTaxConstants_FlatRateCaps(t *testing.T) {
	c, _ := GetTaxConstants(2025)
	expected := map[int]domain.Amount{
		80: domain.NewAmount(1_600_000, 0),
		60: domain.NewAmount(1_200_000, 0),
		40: domain.NewAmount(800_000, 0),
		30: domain.NewAmount(600_000, 0),
	}
	for pct, want := range expected {
		got, ok := c.FlatRateCaps[pct]
		if !ok {
			t.Errorf("FlatRateCaps[%d] not found", pct)
			continue
		}
		if got != want {
			t.Errorf("FlatRateCaps[%d] = %d, want %d", pct, got, want)
		}
	}
}
