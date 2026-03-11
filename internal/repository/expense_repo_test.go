package repository

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestExpenseRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	e := &domain.Expense{
		Description:     "Office supplies",
		Amount:          domain.NewAmount(500, 0),
		IssueDate:       time.Now(),
		CurrencyCode:    domain.CurrencyCZK,
		Category:        "supplies",
		BusinessPercent: 100,
		PaymentMethod:   "bank_transfer",
	}

	if err := repo.Create(ctx, e); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if e.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if e.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestExpenseRepository_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	vendor := testutil.SeedContact(t, db, &domain.Contact{Name: "Vendor X"})
	seeded := testutil.SeedExpense(t, db, &domain.Expense{
		VendorID:    &vendor.ID,
		Description: "Test expense",
		Category:    "travel",
	})

	got, err := repo.GetByID(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.Description != "Test expense" {
		t.Errorf("Description = %q, want %q", got.Description, "Test expense")
	}
	if got.Category != "travel" {
		t.Errorf("Category = %q, want %q", got.Category, "travel")
	}
	if got.Vendor == nil {
		t.Fatal("expected Vendor to be populated")
	}
	if got.Vendor.Name != "Vendor X" {
		t.Errorf("Vendor.Name = %q, want %q", got.Vendor.Name, "Vendor X")
	}
}

func TestExpenseRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent expense")
	}
}

func TestExpenseRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	seeded := testutil.SeedExpense(t, db, &domain.Expense{Description: "Before"})

	seeded.Description = "After"
	seeded.Category = "marketing"
	if err := repo.Update(ctx, seeded); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Description != "After" {
		t.Errorf("Description = %q, want %q", got.Description, "After")
	}
	if got.Category != "marketing" {
		t.Errorf("Category = %q, want %q", got.Category, "marketing")
	}
}

func TestExpenseRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	seeded := testutil.SeedExpense(t, db, nil)

	if err := repo.Delete(ctx, seeded.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := repo.GetByID(ctx, seeded.ID)
	if err == nil {
		t.Error("expected error when getting deleted expense")
	}
}

func TestExpenseRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent expense")
	}
}

func TestExpenseRepository_List_All(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	testutil.SeedExpense(t, db, &domain.Expense{Description: "Expense A"})
	testutil.SeedExpense(t, db, &domain.Expense{Description: "Expense B"})

	expenses, total, err := repo.List(ctx, domain.ExpenseFilter{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(expenses) != 2 {
		t.Errorf("len = %d, want 2", len(expenses))
	}
}

func TestExpenseRepository_List_CategoryFilter(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	testutil.SeedExpense(t, db, &domain.Expense{Description: "Travel", Category: "travel"})
	testutil.SeedExpense(t, db, &domain.Expense{Description: "Office", Category: "supplies"})

	expenses, total, err := repo.List(ctx, domain.ExpenseFilter{Category: "travel"})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if expenses[0].Description != "Travel" {
		t.Errorf("Description = %q, want %q", expenses[0].Description, "Travel")
	}
}

func TestExpenseRepository_List_VendorFilter(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	vendor := testutil.SeedContact(t, db, &domain.Contact{Name: "Vendor Y"})
	testutil.SeedExpense(t, db, &domain.Expense{Description: "From vendor", VendorID: &vendor.ID})
	testutil.SeedExpense(t, db, &domain.Expense{Description: "No vendor"})

	expenses, total, err := repo.List(ctx, domain.ExpenseFilter{VendorID: &vendor.ID})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if expenses[0].Description != "From vendor" {
		t.Errorf("Description = %q, want %q", expenses[0].Description, "From vendor")
	}
}

func TestExpenseRepository_List_SearchFilter(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	testutil.SeedExpense(t, db, &domain.Expense{Description: "Cloud hosting"})
	testutil.SeedExpense(t, db, &domain.Expense{Description: "Lunch meeting"})

	expenses, total, err := repo.List(ctx, domain.ExpenseFilter{Search: "Cloud"})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if expenses[0].Description != "Cloud hosting" {
		t.Errorf("Description = %q, want %q", expenses[0].Description, "Cloud hosting")
	}
}

func TestExpenseRepository_List_DateFilter(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	now := time.Now()
	testutil.SeedExpense(t, db, &domain.Expense{Description: "Recent", IssueDate: now})
	testutil.SeedExpense(t, db, &domain.Expense{Description: "Old", IssueDate: now.AddDate(-1, 0, 0)})

	dateFrom := now.AddDate(0, -1, 0)
	expenses, total, err := repo.List(ctx, domain.ExpenseFilter{DateFrom: &dateFrom})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if expenses[0].Description != "Recent" {
		t.Errorf("Description = %q, want %q", expenses[0].Description, "Recent")
	}
}

func TestExpenseRepository_List_ExcludesSoftDeleted(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	toDelete := testutil.SeedExpense(t, db, &domain.Expense{Description: "Delete me"})
	testutil.SeedExpense(t, db, &domain.Expense{Description: "Keep me"})

	if err := repo.Delete(ctx, toDelete.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	expenses, total, err := repo.List(ctx, domain.ExpenseFilter{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if expenses[0].Description != "Keep me" {
		t.Errorf("Description = %q, want %q", expenses[0].Description, "Keep me")
	}
}
