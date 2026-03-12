package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestTaxDeductionRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxDeductionRepository(db)
	ctx := context.Background()

	ded := &domain.TaxDeduction{
		Year:          2025,
		Category:      domain.DeductionMortgage,
		Description:   "Uroky z hypoteky 2025",
		ClaimedAmount: domain.NewAmount(150000, 0),
		MaxAmount:     domain.NewAmount(300000, 0),
		AllowedAmount: domain.NewAmount(150000, 0),
	}

	if err := repo.Create(ctx, ded); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if ded.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	if ded.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if ded.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestTaxDeductionRepository_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxDeductionRepository(db)
	ctx := context.Background()

	ded := &domain.TaxDeduction{
		Year:          2025,
		Category:      domain.DeductionLifeInsurance,
		Description:   "Zivotni pojisteni 2025",
		ClaimedAmount: domain.NewAmount(24000, 0),
		MaxAmount:     domain.NewAmount(24000, 0),
		AllowedAmount: domain.NewAmount(24000, 0),
	}
	if err := repo.Create(ctx, ded); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, ded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.ID != ded.ID {
		t.Errorf("ID = %d, want %d", got.ID, ded.ID)
	}
	if got.Year != 2025 {
		t.Errorf("Year = %d, want 2025", got.Year)
	}
	if got.Category != domain.DeductionLifeInsurance {
		t.Errorf("Category = %q, want %q", got.Category, domain.DeductionLifeInsurance)
	}
	if got.Description != "Zivotni pojisteni 2025" {
		t.Errorf("Description = %q, want %q", got.Description, "Zivotni pojisteni 2025")
	}
	if got.ClaimedAmount != domain.NewAmount(24000, 0) {
		t.Errorf("ClaimedAmount = %d, want %d", got.ClaimedAmount, domain.NewAmount(24000, 0))
	}
	if got.MaxAmount != domain.NewAmount(24000, 0) {
		t.Errorf("MaxAmount = %d, want %d", got.MaxAmount, domain.NewAmount(24000, 0))
	}
	if got.AllowedAmount != domain.NewAmount(24000, 0) {
		t.Errorf("AllowedAmount = %d, want %d", got.AllowedAmount, domain.NewAmount(24000, 0))
	}
}

func TestTaxDeductionRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxDeductionRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent ID")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestTaxDeductionRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxDeductionRepository(db)
	ctx := context.Background()

	ded := &domain.TaxDeduction{
		Year:          2025,
		Category:      domain.DeductionPension,
		Description:   "Penzijni sporeni 2025",
		ClaimedAmount: domain.NewAmount(12000, 0),
		MaxAmount:     domain.NewAmount(24000, 0),
		AllowedAmount: domain.NewAmount(12000, 0),
	}
	if err := repo.Create(ctx, ded); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	ded.Description = "Penzijni sporeni 2025 - upraveno"
	ded.ClaimedAmount = domain.NewAmount(20000, 0)
	ded.AllowedAmount = domain.NewAmount(20000, 0)

	if err := repo.Update(ctx, ded); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, ded.ID)
	if err != nil {
		t.Fatalf("GetByID() after update error: %v", err)
	}
	if got.Description != "Penzijni sporeni 2025 - upraveno" {
		t.Errorf("Description = %q, want %q", got.Description, "Penzijni sporeni 2025 - upraveno")
	}
	if got.ClaimedAmount != domain.NewAmount(20000, 0) {
		t.Errorf("ClaimedAmount = %d, want %d", got.ClaimedAmount, domain.NewAmount(20000, 0))
	}
	if got.AllowedAmount != domain.NewAmount(20000, 0) {
		t.Errorf("AllowedAmount = %d, want %d", got.AllowedAmount, domain.NewAmount(20000, 0))
	}
}

func TestTaxDeductionRepository_Update_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxDeductionRepository(db)
	ctx := context.Background()

	ded := &domain.TaxDeduction{
		ID:            99999,
		Year:          2025,
		Category:      domain.DeductionDonation,
		Description:   "Nonexistent",
		ClaimedAmount: domain.NewAmount(1000, 0),
		MaxAmount:     domain.NewAmount(1000, 0),
		AllowedAmount: domain.NewAmount(1000, 0),
	}

	err := repo.Update(ctx, ded)
	if err == nil {
		t.Error("expected error for non-existent update")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestTaxDeductionRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxDeductionRepository(db)
	ctx := context.Background()

	ded := &domain.TaxDeduction{
		Year:          2025,
		Category:      domain.DeductionUnionDues,
		Description:   "Odborove prispevky 2025",
		ClaimedAmount: domain.NewAmount(3000, 0),
		MaxAmount:     domain.NewAmount(5000, 0),
		AllowedAmount: domain.NewAmount(3000, 0),
	}
	if err := repo.Create(ctx, ded); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := repo.Delete(ctx, ded.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := repo.GetByID(ctx, ded.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got: %v", err)
	}
}

func TestTaxDeductionRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxDeductionRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent delete")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestTaxDeductionRepository_ListByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewTaxDeductionRepository(db)
	ctx := context.Background()

	// Create multiple deductions for 2025 with different categories.
	deductions := []domain.TaxDeduction{
		{Year: 2025, Category: domain.DeductionPension, Description: "Penzijni sporeni", ClaimedAmount: domain.NewAmount(12000, 0), MaxAmount: domain.NewAmount(24000, 0), AllowedAmount: domain.NewAmount(12000, 0)},
		{Year: 2025, Category: domain.DeductionMortgage, Description: "Uroky z hypoteky", ClaimedAmount: domain.NewAmount(100000, 0), MaxAmount: domain.NewAmount(300000, 0), AllowedAmount: domain.NewAmount(100000, 0)},
		{Year: 2025, Category: domain.DeductionDonation, Description: "Dar nemocnici", ClaimedAmount: domain.NewAmount(5000, 0), MaxAmount: domain.NewAmount(50000, 0), AllowedAmount: domain.NewAmount(5000, 0)},
	}
	for i := range deductions {
		if err := repo.Create(ctx, &deductions[i]); err != nil {
			t.Fatalf("Create() deduction %d error: %v", i+1, err)
		}
	}

	// Create a deduction for a different year.
	otherYear := &domain.TaxDeduction{
		Year: 2024, Category: domain.DeductionMortgage, Description: "Uroky 2024",
		ClaimedAmount: domain.NewAmount(80000, 0), MaxAmount: domain.NewAmount(300000, 0), AllowedAmount: domain.NewAmount(80000, 0),
	}
	if err := repo.Create(ctx, otherYear); err != nil {
		t.Fatalf("Create() other year error: %v", err)
	}

	// List for 2025 should return 3 deductions ordered by category, id.
	result, err := repo.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear(2025) error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("ListByYear(2025) returned %d items, want 3", len(result))
	}

	// Verify ordering by category (donation < mortgage < pension alphabetically).
	if result[0].Category != domain.DeductionDonation {
		t.Errorf("result[0].Category = %q, want %q", result[0].Category, domain.DeductionDonation)
	}
	if result[1].Category != domain.DeductionMortgage {
		t.Errorf("result[1].Category = %q, want %q", result[1].Category, domain.DeductionMortgage)
	}
	if result[2].Category != domain.DeductionPension {
		t.Errorf("result[2].Category = %q, want %q", result[2].Category, domain.DeductionPension)
	}

	// List for 2024 should return 1.
	result2024, err := repo.ListByYear(ctx, 2024)
	if err != nil {
		t.Fatalf("ListByYear(2024) error: %v", err)
	}
	if len(result2024) != 1 {
		t.Errorf("ListByYear(2024) returned %d items, want 1", len(result2024))
	}

	// List for non-existent year should return empty slice.
	resultEmpty, err := repo.ListByYear(ctx, 2099)
	if err != nil {
		t.Fatalf("ListByYear(2099) error: %v", err)
	}
	if len(resultEmpty) != 0 {
		t.Errorf("ListByYear(2099) returned %d items, want 0", len(resultEmpty))
	}
}
