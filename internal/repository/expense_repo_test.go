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

func TestExpenseRepository_MarkTaxReviewed(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	exp1 := testutil.SeedExpense(t, db, &domain.Expense{Description: "Expense 1"})
	exp2 := testutil.SeedExpense(t, db, &domain.Expense{Description: "Expense 2"})
	exp3 := testutil.SeedExpense(t, db, &domain.Expense{Description: "Expense 3"})

	// Mark exp1 and exp2 as tax reviewed.
	if err := repo.MarkTaxReviewed(ctx, []int64{exp1.ID, exp2.ID}); err != nil {
		t.Fatalf("MarkTaxReviewed() error: %v", err)
	}

	// Verify exp1 is reviewed.
	got1, err := repo.GetByID(ctx, exp1.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got1.TaxReviewedAt == nil {
		t.Error("expected exp1.TaxReviewedAt to be set")
	}

	// Verify exp2 is reviewed.
	got2, err := repo.GetByID(ctx, exp2.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got2.TaxReviewedAt == nil {
		t.Error("expected exp2.TaxReviewedAt to be set")
	}

	// Verify exp3 is NOT reviewed.
	got3, err := repo.GetByID(ctx, exp3.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got3.TaxReviewedAt != nil {
		t.Error("expected exp3.TaxReviewedAt to be nil")
	}
}

func TestExpenseRepository_MarkTaxReviewed_EmptySlice(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	// Should be a no-op, not an error.
	if err := repo.MarkTaxReviewed(ctx, []int64{}); err != nil {
		t.Fatalf("MarkTaxReviewed(empty) error: %v", err)
	}
}

func TestExpenseRepository_MarkTaxReviewed_SkipsDeleted(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	exp := testutil.SeedExpense(t, db, &domain.Expense{Description: "Deleted"})
	if err := repo.Delete(ctx, exp.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// MarkTaxReviewed on a deleted expense should not error.
	if err := repo.MarkTaxReviewed(ctx, []int64{exp.ID}); err != nil {
		t.Fatalf("MarkTaxReviewed() error: %v", err)
	}
}

func TestExpenseRepository_UnmarkTaxReviewed(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	exp1 := testutil.SeedExpense(t, db, &domain.Expense{Description: "Expense A"})
	exp2 := testutil.SeedExpense(t, db, &domain.Expense{Description: "Expense B"})

	// First mark both as reviewed.
	if err := repo.MarkTaxReviewed(ctx, []int64{exp1.ID, exp2.ID}); err != nil {
		t.Fatalf("MarkTaxReviewed() error: %v", err)
	}

	// Unmark only exp1.
	if err := repo.UnmarkTaxReviewed(ctx, []int64{exp1.ID}); err != nil {
		t.Fatalf("UnmarkTaxReviewed() error: %v", err)
	}

	// Verify exp1 is no longer reviewed.
	got1, err := repo.GetByID(ctx, exp1.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got1.TaxReviewedAt != nil {
		t.Error("expected exp1.TaxReviewedAt to be nil after unmark")
	}

	// Verify exp2 is still reviewed.
	got2, err := repo.GetByID(ctx, exp2.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got2.TaxReviewedAt == nil {
		t.Error("expected exp2.TaxReviewedAt to still be set")
	}
}

func TestExpenseRepository_UnmarkTaxReviewed_EmptySlice(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	// Should be a no-op, not an error.
	if err := repo.UnmarkTaxReviewed(ctx, []int64{}); err != nil {
		t.Fatalf("UnmarkTaxReviewed(empty) error: %v", err)
	}
}

func TestExpenseRepository_List_TaxReviewedFilter(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	exp1 := testutil.SeedExpense(t, db, &domain.Expense{Description: "Reviewed"})
	testutil.SeedExpense(t, db, &domain.Expense{Description: "Not reviewed"})

	// Mark exp1 as reviewed.
	if err := repo.MarkTaxReviewed(ctx, []int64{exp1.ID}); err != nil {
		t.Fatalf("MarkTaxReviewed() error: %v", err)
	}

	// Filter for reviewed expenses.
	reviewed := true
	expenses, total, err := repo.List(ctx, domain.ExpenseFilter{TaxReviewed: &reviewed})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if expenses[0].Description != "Reviewed" {
		t.Errorf("Description = %q, want %q", expenses[0].Description, "Reviewed")
	}

	// Filter for unreviewed expenses.
	notReviewed := false
	expenses, total, err = repo.List(ctx, domain.ExpenseFilter{TaxReviewed: &notReviewed})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if expenses[0].Description != "Not reviewed" {
		t.Errorf("Description = %q, want %q", expenses[0].Description, "Not reviewed")
	}
}

func TestExpenseRepository_Create_WithItems(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	e := &domain.Expense{
		Description:     "Itemized expense",
		IssueDate:       time.Now(),
		CurrencyCode:    domain.CurrencyCZK,
		BusinessPercent: 100,
		PaymentMethod:   "bank_transfer",
		Items: []domain.ExpenseItem{
			{
				Description:    "Item A",
				Quantity:       100,
				Unit:           "ks",
				UnitPrice:      10000,
				VATRatePercent: 21,
				VATAmount:      2100,
				TotalAmount:    12100,
				SortOrder:      1,
			},
			{
				Description:    "Item B",
				Quantity:       200,
				Unit:           "ks",
				UnitPrice:      5000,
				VATRatePercent: 21,
				VATAmount:      2100,
				TotalAmount:    12100,
				SortOrder:      2,
			},
		},
		Amount:    24200,
		VATAmount: 4200,
	}

	if err := repo.Create(ctx, e); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if e.ID == 0 {
		t.Error("expected non-zero ID")
	}

	got, err := repo.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("Items count = %d, want 2", len(got.Items))
	}
	if got.Items[0].Description != "Item A" {
		t.Errorf("Items[0].Description = %q, want %q", got.Items[0].Description, "Item A")
	}
	if got.Items[0].VATAmount != 2100 {
		t.Errorf("Items[0].VATAmount = %d, want 2100", got.Items[0].VATAmount)
	}
	if got.Items[1].SortOrder != 2 {
		t.Errorf("Items[1].SortOrder = %d, want 2", got.Items[1].SortOrder)
	}
}

func TestExpenseRepository_Update_WithItems(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewExpenseRepository(db)
	ctx := context.Background()

	e := &domain.Expense{
		Description:     "Flat expense",
		Amount:          50000,
		IssueDate:       time.Now(),
		CurrencyCode:    domain.CurrencyCZK,
		BusinessPercent: 100,
		PaymentMethod:   "bank_transfer",
	}
	if err := repo.Create(ctx, e); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	e.Items = []domain.ExpenseItem{
		{
			Description:    "New item",
			Quantity:       100,
			Unit:           "ks",
			UnitPrice:      20000,
			VATRatePercent: 21,
			VATAmount:      4200,
			TotalAmount:    24200,
			SortOrder:      1,
		},
	}
	e.Amount = 24200
	e.VATAmount = 4200

	if err := repo.Update(ctx, e); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("Items count = %d, want 1", len(got.Items))
	}
	if got.Items[0].Description != "New item" {
		t.Errorf("Items[0].Description = %q, want %q", got.Items[0].Description, "New item")
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
