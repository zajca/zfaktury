package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestSocialInsuranceOverviewRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}

	if err := repo.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if sio.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	if sio.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if sio.Status != domain.FilingStatusDraft {
		t.Errorf("Status = %q, want %q", sio.Status, domain.FilingStatusDraft)
	}
}

func TestSocialInsuranceOverviewRepository_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		TotalRevenue:   domain.NewAmount(800000, 0),
		TotalExpenses:  domain.NewAmount(300000, 0),
		TotalInsurance: domain.NewAmount(145000, 0),
	}
	if err := repo.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.Year != 2025 {
		t.Errorf("Year = %d, want 2025", got.Year)
	}
	if got.TotalRevenue != domain.NewAmount(800000, 0) {
		t.Errorf("TotalRevenue = %d, want %d", got.TotalRevenue, domain.NewAmount(800000, 0))
	}
	if got.TotalInsurance != domain.NewAmount(145000, 0) {
		t.Errorf("TotalInsurance = %d, want %d", got.TotalInsurance, domain.NewAmount(145000, 0))
	}
}

func TestSocialInsuranceOverviewRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent social_insurance_overview")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestSocialInsuranceOverviewRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	sio.TotalRevenue = domain.NewAmount(1200000, 0)
	sio.TotalInsurance = domain.NewAmount(175000, 0)
	sio.Difference = domain.NewAmount(50000, 0)

	if err := repo.Update(ctx, sio); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.TotalRevenue != domain.NewAmount(1200000, 0) {
		t.Errorf("TotalRevenue = %d, want %d", got.TotalRevenue, domain.NewAmount(1200000, 0))
	}
	if got.TotalInsurance != domain.NewAmount(175000, 0) {
		t.Errorf("TotalInsurance = %d, want %d", got.TotalInsurance, domain.NewAmount(175000, 0))
	}
}

func TestSocialInsuranceOverviewRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := repo.Delete(ctx, sio.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := repo.GetByID(ctx, sio.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestSocialInsuranceOverviewRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent delete")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestSocialInsuranceOverviewRepository_List(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	for i, ft := range []string{domain.FilingTypeRegular, domain.FilingTypeCorrective} {
		sio := &domain.SocialInsuranceOverview{
			Year:       2025,
			FilingType: ft,
		}
		if err := repo.Create(ctx, sio); err != nil {
			t.Fatalf("Create() %d error: %v", i, err)
		}
	}

	// Different year.
	sio2024 := &domain.SocialInsuranceOverview{
		Year:       2024,
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, sio2024); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	returns, err := repo.List(ctx, 2025)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(returns) != 2 {
		t.Errorf("List(2025) returned %d items, want 2", len(returns))
	}

	returns2024, err := repo.List(ctx, 2024)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(returns2024) != 1 {
		t.Errorf("List(2024) returned %d items, want 1", len(returns2024))
	}
}

func TestSocialInsuranceOverviewRepository_GetByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByYear(ctx, 2025, domain.FilingTypeRegular)
	if err != nil {
		t.Fatalf("GetByYear() error: %v", err)
	}
	if got.ID != sio.ID {
		t.Errorf("GetByYear() ID = %d, want %d", got.ID, sio.ID)
	}

	// Non-existent.
	_, err = repo.GetByYear(ctx, 2099, domain.FilingTypeRegular)
	if err == nil {
		t.Error("expected error for non-existent year")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}
