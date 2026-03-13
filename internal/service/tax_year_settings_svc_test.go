package service

import (
	"context"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func newTaxYearSettingsSvc(t *testing.T) (*TaxYearSettingsService, *repository.TaxYearSettingsRepository, *repository.TaxPrepaymentRepository) {
	t.Helper()
	db := testutil.NewTestDB(t)
	settingsRepo := repository.NewTaxYearSettingsRepository(db)
	prepaymentRepo := repository.NewTaxPrepaymentRepository(db)
	svc := NewTaxYearSettingsService(settingsRepo, prepaymentRepo, nil)
	return svc, settingsRepo, prepaymentRepo
}

func TestTaxYearSettingsService_GetByYear_Defaults(t *testing.T) {
	svc, _, _ := newTaxYearSettingsSvc(t)
	ctx := context.Background()

	tys, err := svc.GetByYear(ctx, 2099)
	if err != nil {
		t.Fatalf("GetByYear: %v", err)
	}
	if tys.Year != 2099 {
		t.Errorf("expected year 2099, got %d", tys.Year)
	}
	if tys.FlatRatePercent != 0 {
		t.Errorf("expected flat_rate_percent 0, got %d", tys.FlatRatePercent)
	}
}

func TestTaxYearSettingsService_GetByYear_Existing(t *testing.T) {
	svc, settingsRepo, _ := newTaxYearSettingsSvc(t)
	ctx := context.Background()

	if err := settingsRepo.Upsert(ctx, &domain.TaxYearSettings{Year: 2025, FlatRatePercent: 60}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	tys, err := svc.GetByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("GetByYear: %v", err)
	}
	if tys.FlatRatePercent != 60 {
		t.Errorf("expected flat_rate_percent 60, got %d", tys.FlatRatePercent)
	}
}

func TestTaxYearSettingsService_GetPrepayments_FillsMissing(t *testing.T) {
	svc, _, prepaymentRepo := newTaxYearSettingsSvc(t)
	ctx := context.Background()

	// Insert only 3 months
	partial := []domain.TaxPrepayment{
		{Month: 1, TaxAmount: 100000, SocialAmount: 200000, HealthAmount: 300000},
		{Month: 6, TaxAmount: 150000, SocialAmount: 250000, HealthAmount: 350000},
		{Month: 12, TaxAmount: 200000, SocialAmount: 300000, HealthAmount: 400000},
	}
	if err := prepaymentRepo.UpsertAll(ctx, 2025, partial); err != nil {
		t.Fatalf("UpsertAll: %v", err)
	}

	result, err := svc.GetPrepayments(ctx, 2025)
	if err != nil {
		t.Fatalf("GetPrepayments: %v", err)
	}
	if len(result) != 12 {
		t.Fatalf("expected 12 months, got %d", len(result))
	}

	// Check filled months have correct values
	if result[0].TaxAmount != 100000 {
		t.Errorf("month 1: expected tax 100000, got %d", result[0].TaxAmount)
	}
	if result[5].TaxAmount != 150000 {
		t.Errorf("month 6: expected tax 150000, got %d", result[5].TaxAmount)
	}
	if result[11].TaxAmount != 200000 {
		t.Errorf("month 12: expected tax 200000, got %d", result[11].TaxAmount)
	}

	// Check unfilled months are zero
	if result[1].TaxAmount != 0 {
		t.Errorf("month 2: expected tax 0, got %d", result[1].TaxAmount)
	}
	if result[1].Month != 2 {
		t.Errorf("month 2: expected month field 2, got %d", result[1].Month)
	}
	if result[1].Year != 2025 {
		t.Errorf("month 2: expected year 2025, got %d", result[1].Year)
	}
}

func TestTaxYearSettingsService_Save(t *testing.T) {
	svc, settingsRepo, prepaymentRepo := newTaxYearSettingsSvc(t)
	ctx := context.Background()

	prepayments := []domain.TaxPrepayment{
		{Month: 1, TaxAmount: 100000, SocialAmount: 200000, HealthAmount: 300000},
		{Month: 2, TaxAmount: 110000, SocialAmount: 210000, HealthAmount: 310000},
	}
	if err := svc.Save(ctx, 2025, 60, prepayments); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify settings were saved
	tys, err := settingsRepo.GetByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("GetByYear: %v", err)
	}
	if tys.FlatRatePercent != 60 {
		t.Errorf("expected flat_rate_percent 60, got %d", tys.FlatRatePercent)
	}

	// Verify prepayments were saved
	result, err := prepaymentRepo.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 prepayments, got %d", len(result))
	}
}

func TestTaxYearSettingsService_GetPrepaymentSums(t *testing.T) {
	svc, _, prepaymentRepo := newTaxYearSettingsSvc(t)
	ctx := context.Background()

	prepayments := []domain.TaxPrepayment{
		{Month: 1, TaxAmount: 100000, SocialAmount: 200000, HealthAmount: 300000},
		{Month: 2, TaxAmount: 150000, SocialAmount: 250000, HealthAmount: 350000},
	}
	if err := prepaymentRepo.UpsertAll(ctx, 2025, prepayments); err != nil {
		t.Fatalf("UpsertAll: %v", err)
	}

	taxTotal, socialTotal, healthTotal, err := svc.GetPrepaymentSums(ctx, 2025)
	if err != nil {
		t.Fatalf("GetPrepaymentSums: %v", err)
	}
	if taxTotal != 250000 {
		t.Errorf("expected tax total 250000, got %d", taxTotal)
	}
	if socialTotal != 450000 {
		t.Errorf("expected social total 450000, got %d", socialTotal)
	}
	if healthTotal != 650000 {
		t.Errorf("expected health total 650000, got %d", healthTotal)
	}
}
