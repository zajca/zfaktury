package repository

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func seedRecurringExpense(t *testing.T, repo *RecurringExpenseRepository, re *domain.RecurringExpense) *domain.RecurringExpense {
	t.Helper()
	if re == nil {
		re = &domain.RecurringExpense{}
	}
	if re.Name == "" {
		re.Name = "Test Recurring"
	}
	if re.Description == "" {
		re.Description = "Test recurring expense"
	}
	if re.Amount == 0 {
		re.Amount = domain.NewAmount(1000, 0)
	}
	if re.CurrencyCode == "" {
		re.CurrencyCode = domain.CurrencyCZK
	}
	if re.Frequency == "" {
		re.Frequency = "monthly"
	}
	if re.NextIssueDate.IsZero() {
		re.NextIssueDate = time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	}
	if re.BusinessPercent == 0 {
		re.BusinessPercent = 100
	}
	if re.PaymentMethod == "" {
		re.PaymentMethod = "bank_transfer"
	}
	re.IsActive = true

	ctx := context.Background()
	if err := repo.Create(ctx, re); err != nil {
		t.Fatalf("seedRecurringExpense: %v", err)
	}
	return re
}

func TestRecurringExpenseRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringExpenseRepository(db)

	re := &domain.RecurringExpense{
		Name:            "Monthly hosting",
		Description:     "Cloud hosting fee",
		Amount:          domain.NewAmount(500, 0),
		CurrencyCode:    domain.CurrencyCZK,
		Frequency:       "monthly",
		NextIssueDate:   time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		IsActive:        true,
		BusinessPercent: 100,
		PaymentMethod:   "bank_transfer",
	}

	if err := repo.Create(context.Background(), re); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if re.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if re.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestRecurringExpenseRepository_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringExpenseRepository(db)

	vendor := testutil.SeedContact(t, db, &domain.Contact{Name: "Hosting Co"})
	seeded := seedRecurringExpense(t, repo, &domain.RecurringExpense{
		Name:        "Monthly hosting",
		Description: "Cloud hosting",
		VendorID:    &vendor.ID,
	})

	got, err := repo.GetByID(context.Background(), seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Name != "Monthly hosting" {
		t.Errorf("Name = %q, want %q", got.Name, "Monthly hosting")
	}
	if got.Description != "Cloud hosting" {
		t.Errorf("Description = %q, want %q", got.Description, "Cloud hosting")
	}
	if got.Vendor == nil {
		t.Fatal("expected Vendor to be populated")
	}
	if got.Vendor.Name != "Hosting Co" {
		t.Errorf("Vendor.Name = %q, want %q", got.Vendor.Name, "Hosting Co")
	}
}

func TestRecurringExpenseRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringExpenseRepository(db)

	_, err := repo.GetByID(context.Background(), 99999)
	if err == nil {
		t.Error("expected error for non-existent recurring expense")
	}
}

func TestRecurringExpenseRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringExpenseRepository(db)

	seeded := seedRecurringExpense(t, repo, &domain.RecurringExpense{Name: "Before"})

	seeded.Name = "After"
	seeded.Frequency = "quarterly"
	if err := repo.Update(context.Background(), seeded); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(context.Background(), seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Name != "After" {
		t.Errorf("Name = %q, want %q", got.Name, "After")
	}
	if got.Frequency != "quarterly" {
		t.Errorf("Frequency = %q, want %q", got.Frequency, "quarterly")
	}
}

func TestRecurringExpenseRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringExpenseRepository(db)

	seeded := seedRecurringExpense(t, repo, nil)

	if err := repo.Delete(context.Background(), seeded.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := repo.GetByID(context.Background(), seeded.ID)
	if err == nil {
		t.Error("expected error when getting deleted recurring expense")
	}
}

func TestRecurringExpenseRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringExpenseRepository(db)

	err := repo.Delete(context.Background(), 99999)
	if err == nil {
		t.Error("expected error for non-existent recurring expense")
	}
}

func TestRecurringExpenseRepository_List(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringExpenseRepository(db)

	seedRecurringExpense(t, repo, &domain.RecurringExpense{Name: "A"})
	seedRecurringExpense(t, repo, &domain.RecurringExpense{Name: "B"})

	items, total, err := repo.List(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(items) != 2 {
		t.Errorf("len = %d, want 2", len(items))
	}
}

func TestRecurringExpenseRepository_List_ExcludesSoftDeleted(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringExpenseRepository(db)

	toDelete := seedRecurringExpense(t, repo, &domain.RecurringExpense{Name: "Delete me"})
	seedRecurringExpense(t, repo, &domain.RecurringExpense{Name: "Keep me"})

	if err := repo.Delete(context.Background(), toDelete.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	items, total, err := repo.List(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if items[0].Name != "Keep me" {
		t.Errorf("Name = %q, want %q", items[0].Name, "Keep me")
	}
}

func TestRecurringExpenseRepository_ListActive(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringExpenseRepository(db)

	active := seedRecurringExpense(t, repo, &domain.RecurringExpense{Name: "Active"})
	_ = active
	inactive := seedRecurringExpense(t, repo, &domain.RecurringExpense{Name: "Inactive"})
	if err := repo.Deactivate(context.Background(), inactive.ID); err != nil {
		t.Fatalf("Deactivate() error: %v", err)
	}

	items, err := repo.ListActive(context.Background())
	if err != nil {
		t.Fatalf("ListActive() error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("len = %d, want 1", len(items))
	}
	if items[0].Name != "Active" {
		t.Errorf("Name = %q, want %q", items[0].Name, "Active")
	}
}

func TestRecurringExpenseRepository_ListDue(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringExpenseRepository(db)

	seedRecurringExpense(t, repo, &domain.RecurringExpense{
		Name:          "Due",
		NextIssueDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	})
	seedRecurringExpense(t, repo, &domain.RecurringExpense{
		Name:          "Future",
		NextIssueDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	})

	asOf := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	items, err := repo.ListDue(context.Background(), asOf)
	if err != nil {
		t.Fatalf("ListDue() error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len = %d, want 1", len(items))
	}
	if items[0].Name != "Due" {
		t.Errorf("Name = %q, want %q", items[0].Name, "Due")
	}
}

func TestRecurringExpenseRepository_Deactivate(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringExpenseRepository(db)

	seeded := seedRecurringExpense(t, repo, nil)

	if err := repo.Deactivate(context.Background(), seeded.ID); err != nil {
		t.Fatalf("Deactivate() error: %v", err)
	}

	got, err := repo.GetByID(context.Background(), seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.IsActive {
		t.Error("expected IsActive to be false after deactivation")
	}
}

func TestRecurringExpenseRepository_Activate(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringExpenseRepository(db)

	seeded := seedRecurringExpense(t, repo, nil)
	if err := repo.Deactivate(context.Background(), seeded.ID); err != nil {
		t.Fatalf("Deactivate() error: %v", err)
	}

	if err := repo.Activate(context.Background(), seeded.ID); err != nil {
		t.Fatalf("Activate() error: %v", err)
	}

	got, err := repo.GetByID(context.Background(), seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if !got.IsActive {
		t.Error("expected IsActive to be true after activation")
	}
}

func TestRecurringExpenseRepository_WithEndDate(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringExpenseRepository(db)

	endDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	seeded := seedRecurringExpense(t, repo, &domain.RecurringExpense{
		Name:    "With end date",
		EndDate: &endDate,
	})

	got, err := repo.GetByID(context.Background(), seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.EndDate == nil {
		t.Fatal("expected EndDate to be set")
	}
	if got.EndDate.Format("2006-01-02") != "2026-12-31" {
		t.Errorf("EndDate = %s, want 2026-12-31", got.EndDate.Format("2006-01-02"))
	}
}
