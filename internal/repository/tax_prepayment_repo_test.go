package repository

import (
	"context"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestTaxPrepaymentRepository_ListByYear_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxPrepaymentRepository(db)
	ctx := context.Background()

	result, err := repo.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear on empty DB: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %d entries", len(result))
	}
}

func TestTaxPrepaymentRepository_UpsertAll_And_ListByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxPrepaymentRepository(db)
	ctx := context.Background()

	prepayments := []domain.TaxPrepayment{
		{Month: 1, TaxAmount: 100000, SocialAmount: 200000, HealthAmount: 300000},
		{Month: 2, TaxAmount: 110000, SocialAmount: 210000, HealthAmount: 310000},
		{Month: 3, TaxAmount: 120000, SocialAmount: 220000, HealthAmount: 320000},
	}

	if err := repo.UpsertAll(ctx, 2025, prepayments); err != nil {
		t.Fatalf("UpsertAll: %v", err)
	}

	result, err := repo.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 prepayments, got %d", len(result))
	}

	// Verify ordering by month and values
	for i, want := range prepayments {
		got := result[i]
		if got.Year != 2025 {
			t.Errorf("row %d: expected year 2025, got %d", i, got.Year)
		}
		if got.Month != want.Month {
			t.Errorf("row %d: expected month %d, got %d", i, want.Month, got.Month)
		}
		if got.TaxAmount != want.TaxAmount {
			t.Errorf("row %d: expected tax_amount %d, got %d", i, want.TaxAmount, got.TaxAmount)
		}
		if got.SocialAmount != want.SocialAmount {
			t.Errorf("row %d: expected social_amount %d, got %d", i, want.SocialAmount, got.SocialAmount)
		}
		if got.HealthAmount != want.HealthAmount {
			t.Errorf("row %d: expected health_amount %d, got %d", i, want.HealthAmount, got.HealthAmount)
		}
	}

	// Verify different year returns nothing
	other, err := repo.ListByYear(ctx, 2024)
	if err != nil {
		t.Fatalf("ListByYear other year: %v", err)
	}
	if len(other) != 0 {
		t.Fatalf("expected empty slice for other year, got %d", len(other))
	}
}

func TestTaxPrepaymentRepository_UpsertAll_Replaces(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxPrepaymentRepository(db)
	ctx := context.Background()

	// First upsert: 3 months
	first := []domain.TaxPrepayment{
		{Month: 1, TaxAmount: 100000, SocialAmount: 200000, HealthAmount: 300000},
		{Month: 2, TaxAmount: 110000, SocialAmount: 210000, HealthAmount: 310000},
		{Month: 3, TaxAmount: 120000, SocialAmount: 220000, HealthAmount: 320000},
	}
	if err := repo.UpsertAll(ctx, 2025, first); err != nil {
		t.Fatalf("UpsertAll first: %v", err)
	}

	// Second upsert: only 2 months with different amounts
	second := []domain.TaxPrepayment{
		{Month: 1, TaxAmount: 500000, SocialAmount: 600000, HealthAmount: 700000},
		{Month: 2, TaxAmount: 550000, SocialAmount: 650000, HealthAmount: 750000},
	}
	if err := repo.UpsertAll(ctx, 2025, second); err != nil {
		t.Fatalf("UpsertAll second: %v", err)
	}

	result, err := repo.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear after replace: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 prepayments after replace, got %d", len(result))
	}
	if result[0].TaxAmount != 500000 {
		t.Errorf("expected replaced tax_amount 500000, got %d", result[0].TaxAmount)
	}
	if result[1].TaxAmount != 550000 {
		t.Errorf("expected replaced tax_amount 550000, got %d", result[1].TaxAmount)
	}
}

func TestTaxPrepaymentRepository_SumByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxPrepaymentRepository(db)
	ctx := context.Background()

	prepayments := []domain.TaxPrepayment{
		{Month: 1, TaxAmount: 100000, SocialAmount: 200000, HealthAmount: 300000},
		{Month: 2, TaxAmount: 150000, SocialAmount: 250000, HealthAmount: 350000},
		{Month: 3, TaxAmount: 200000, SocialAmount: 300000, HealthAmount: 400000},
	}
	if err := repo.UpsertAll(ctx, 2025, prepayments); err != nil {
		t.Fatalf("UpsertAll: %v", err)
	}

	taxTotal, socialTotal, healthTotal, err := repo.SumByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("SumByYear: %v", err)
	}

	// 100000 + 150000 + 200000 = 450000
	if taxTotal != 450000 {
		t.Errorf("expected tax total 450000, got %d", taxTotal)
	}
	// 200000 + 250000 + 300000 = 750000
	if socialTotal != 750000 {
		t.Errorf("expected social total 750000, got %d", socialTotal)
	}
	// 300000 + 350000 + 400000 = 1050000
	if healthTotal != 1050000 {
		t.Errorf("expected health total 1050000, got %d", healthTotal)
	}

	// Empty year returns zeros
	taxEmpty, socialEmpty, healthEmpty, err := repo.SumByYear(ctx, 2024)
	if err != nil {
		t.Fatalf("SumByYear empty year: %v", err)
	}
	if taxEmpty != 0 || socialEmpty != 0 || healthEmpty != 0 {
		t.Errorf("expected all zeros for empty year, got tax=%d social=%d health=%d", taxEmpty, socialEmpty, healthEmpty)
	}
}
