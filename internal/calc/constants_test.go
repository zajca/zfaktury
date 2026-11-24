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
	// SP HV 2024: 30 % × 43 967 = 13 191 Kč.
	if c.SocialMinMonthly != domain.NewAmount(13_191, 0) {
		t.Errorf("2024 SocialMinMonthly = %d, want %d", c.SocialMinMonthly, domain.NewAmount(13_191, 0))
	}
	// CSSZ max annual assessment base 2024: 48 × 43 967 = 2 110 416 Kč.
	if c.SocialMaxAnnualBase != domain.NewAmount(2_110_416, 0) {
		t.Errorf("2024 SocialMaxAnnualBase = %d, want %d", c.SocialMaxAnnualBase, domain.NewAmount(2_110_416, 0))
	}
	// ZP 2024: 50 % × 43 967 = 21 983,50 Kč (stored in halere).
	if c.HealthMinMonthly != domain.NewAmount(21_983, 50) {
		t.Errorf("2024 HealthMinMonthly = %d, want %d", c.HealthMinMonthly, domain.NewAmount(21_983, 50))
	}
	// Sleva na studenta zrušena konsolidačním balíčkem od 1. 1. 2024.
	if c.StudentCredit != 0 {
		t.Errorf("2024 StudentCredit = %d, want 0 (zrušena od 2024)", c.StudentCredit)
	}
	if c.DeductionCapSavingsCombined != domain.NewAmount(48_000, 0) {
		t.Errorf("2024 DeductionCapSavingsCombined = %d, want 48000", c.DeductionCapSavingsCombined)
	}
	if c.SecurityExemptionLimit != 0 {
		t.Errorf("2024 SecurityExemptionLimit = %d, want 0 (no limit)", c.SecurityExemptionLimit)
	}
}

func TestGetTaxConstants_2025Values(t *testing.T) {
	c, _ := GetTaxConstants(2025)
	// SP HV 2025: 35 % × 46 557 = 16 295 Kč.
	if c.SocialMinMonthly != domain.NewAmount(16_295, 0) {
		t.Errorf("2025 SocialMinMonthly = %d, want %d", c.SocialMinMonthly, domain.NewAmount(16_295, 0))
	}
	// CSSZ max annual assessment base 2025: 48 × 46 557 = 2 234 736 Kč.
	if c.SocialMaxAnnualBase != domain.NewAmount(2_234_736, 0) {
		t.Errorf("2025 SocialMaxAnnualBase = %d, want %d", c.SocialMaxAnnualBase, domain.NewAmount(2_234_736, 0))
	}
	// ZP 2025: 50 % × 46 557 = 23 278,50 Kč (stored in halere).
	if c.HealthMinMonthly != domain.NewAmount(23_278, 50) {
		t.Errorf("2025 HealthMinMonthly = %d, want %d", c.HealthMinMonthly, domain.NewAmount(23_278, 50))
	}
	if c.StudentCredit != 0 {
		t.Errorf("2025 StudentCredit = %d, want 0 (zrušena od 2024)", c.StudentCredit)
	}
	if c.SecurityExemptionLimit != domain.NewAmount(40_000_000, 0) {
		t.Errorf("2025 SecurityExemptionLimit = %d, want %d", c.SecurityExemptionLimit, domain.NewAmount(40_000_000, 0))
	}
}

func TestGetTaxConstants_2026Values(t *testing.T) {
	c, _ := GetTaxConstants(2026)
	if c.ProgressiveThreshold != domain.NewAmount(1_762_812, 0) {
		t.Errorf("2026 ProgressiveThreshold = %d, want %d", c.ProgressiveThreshold, domain.NewAmount(1_762_812, 0))
	}
	// Annual overview minimum: 35 % × 48 967 = 17 138,45 → 17 139 Kč for all
	// twelve months (CSSZ recalculates January-June).
	if c.SocialMinMonthly != domain.NewAmount(17_139, 0) {
		t.Errorf("2026 SocialMinMonthly = %d, want %d", c.SocialMinMonthly, domain.NewAmount(17_139, 0))
	}
	// CSSZ max annual assessment base 2026: 48 × 48 967 = 2 350 416 Kč.
	if c.SocialMaxAnnualBase != domain.NewAmount(2_350_416, 0) {
		t.Errorf("2026 SocialMaxAnnualBase = %d, want %d", c.SocialMaxAnnualBase, domain.NewAmount(2_350_416, 0))
	}
	// ZP 2026: 50 % × 48 967 = 24 483,50 Kč (stored in halere).
	if c.HealthMinMonthly != domain.NewAmount(24_483, 50) {
		t.Errorf("2026 HealthMinMonthly = %d, want %d", c.HealthMinMonthly, domain.NewAmount(24_483, 50))
	}
	if c.StudentCredit != 0 {
		t.Errorf("2026 StudentCredit = %d, want 0 (zrušena od 2024)", c.StudentCredit)
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
