package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestTaxYearSettingsRepository_GetByYear_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxYearSettingsRepository(db)
	ctx := context.Background()

	_, err := repo.GetByYear(ctx, 2099)
	if err == nil {
		t.Fatal("expected error for nonexistent year, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected domain.ErrNotFound, got: %v", err)
	}
}

func TestTaxYearSettingsRepository_Upsert_And_GetByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxYearSettingsRepository(db)
	ctx := context.Background()

	tys := &domain.TaxYearSettings{
		Year:            2025,
		FlatRatePercent: 60,
	}
	if err := repo.Upsert(ctx, tys); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := repo.GetByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("GetByYear: %v", err)
	}
	if got.Year != 2025 {
		t.Errorf("expected year 2025, got %d", got.Year)
	}
	if got.FlatRatePercent != 60 {
		t.Errorf("expected flat_rate_percent 60, got %d", got.FlatRatePercent)
	}
	if got.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	if got.UpdatedAt.IsZero() {
		t.Error("expected non-zero updated_at")
	}
}

func TestTaxYearSettingsRepository_Upsert_Overwrites(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxYearSettingsRepository(db)
	ctx := context.Background()

	tys := &domain.TaxYearSettings{
		Year:            2025,
		FlatRatePercent: 60,
	}
	if err := repo.Upsert(ctx, tys); err != nil {
		t.Fatalf("Upsert first: %v", err)
	}

	tys.FlatRatePercent = 80
	if err := repo.Upsert(ctx, tys); err != nil {
		t.Fatalf("Upsert second: %v", err)
	}

	got, err := repo.GetByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("GetByYear after overwrite: %v", err)
	}
	if got.FlatRatePercent != 80 {
		t.Errorf("expected flat_rate_percent 80 after overwrite, got %d", got.FlatRatePercent)
	}
}
