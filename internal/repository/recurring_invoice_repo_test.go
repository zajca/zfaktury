package repository

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func makeRecurringInvoice(customerID int64) *domain.RecurringInvoice {
	return &domain.RecurringInvoice{
		Name:          "Monthly hosting",
		CustomerID:    customerID,
		Frequency:     domain.FrequencyMonthly,
		NextIssueDate: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		CurrencyCode:  domain.CurrencyCZK,
		ExchangeRate:  100,
		PaymentMethod: "bank_transfer",
		IsActive:      true,
		Items: []domain.RecurringInvoiceItem{
			{
				Description:    "Web hosting",
				Quantity:       100,
				Unit:           "ks",
				UnitPrice:      50000,
				VATRatePercent: 21,
				SortOrder:      0,
			},
		},
	}
}

func TestRecurringInvoiceRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	ri := makeRecurringInvoice(customer.ID)

	if err := repo.Create(ctx, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if ri.ID == 0 {
		t.Error("expected non-zero recurring invoice ID")
	}
	if ri.Items[0].ID == 0 {
		t.Error("expected non-zero item ID")
	}
	if ri.Items[0].RecurringInvoiceID != ri.ID {
		t.Errorf("item.RecurringInvoiceID = %d, want %d", ri.Items[0].RecurringInvoiceID, ri.ID)
	}
}

func TestRecurringInvoiceRepository_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	ri := makeRecurringInvoice(customer.ID)
	if err := repo.Create(ctx, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, ri.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.Name != "Monthly hosting" {
		t.Errorf("Name = %q, want %q", got.Name, "Monthly hosting")
	}
	if got.Frequency != domain.FrequencyMonthly {
		t.Errorf("Frequency = %q, want %q", got.Frequency, domain.FrequencyMonthly)
	}
	if got.Customer == nil {
		t.Fatal("expected Customer to be populated")
	}
	if got.Customer.Name != "Test Customer" {
		t.Errorf("Customer.Name = %q, want %q", got.Customer.Name, "Test Customer")
	}
	if len(got.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(got.Items))
	}
	if got.Items[0].Description != "Web hosting" {
		t.Errorf("Items[0].Description = %q, want %q", got.Items[0].Description, "Web hosting")
	}
}

func TestRecurringInvoiceRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent recurring invoice")
	}
}

func TestRecurringInvoiceRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	ri := makeRecurringInvoice(customer.ID)
	if err := repo.Create(ctx, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	ri.Name = "Updated name"
	ri.Notes = "Updated notes"
	ri.Items = []domain.RecurringInvoiceItem{
		{Description: "New item A", Quantity: 200, Unit: "hod", UnitPrice: 100000, VATRatePercent: 21, SortOrder: 0},
		{Description: "New item B", Quantity: 100, Unit: "ks", UnitPrice: 30000, VATRatePercent: 0, SortOrder: 1},
	}

	if err := repo.Update(ctx, ri); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, ri.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Name != "Updated name" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated name")
	}
	if got.Notes != "Updated notes" {
		t.Errorf("Notes = %q, want %q", got.Notes, "Updated notes")
	}
	if len(got.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(got.Items))
	}
	if got.Items[0].Description != "New item A" {
		t.Errorf("Items[0].Description = %q, want %q", got.Items[0].Description, "New item A")
	}
}

func TestRecurringInvoiceRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	ri := makeRecurringInvoice(customer.ID)
	if err := repo.Create(ctx, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := repo.Delete(ctx, ri.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := repo.GetByID(ctx, ri.ID)
	if err == nil {
		t.Error("expected error when getting deleted recurring invoice")
	}
}

func TestRecurringInvoiceRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent recurring invoice")
	}
}

func TestRecurringInvoiceRepository_List(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	ri1 := makeRecurringInvoice(customer.ID)
	ri2 := makeRecurringInvoice(customer.ID)
	ri2.Name = "Quarterly service"
	ri2.Frequency = domain.FrequencyQuarterly

	repo.Create(ctx, ri1)
	repo.Create(ctx, ri2)

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("len(list) = %d, want 2", len(list))
	}
}

func TestRecurringInvoiceRepository_ListDue(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	// Due today.
	ri1 := makeRecurringInvoice(customer.ID)
	ri1.NextIssueDate = time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	repo.Create(ctx, ri1)

	// Due in the future.
	ri2 := makeRecurringInvoice(customer.ID)
	ri2.Name = "Future invoice"
	ri2.NextIssueDate = time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	repo.Create(ctx, ri2)

	// Inactive but due.
	ri3 := makeRecurringInvoice(customer.ID)
	ri3.Name = "Inactive invoice"
	ri3.NextIssueDate = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	ri3.IsActive = false
	repo.Create(ctx, ri3)

	today := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	due, err := repo.ListDue(ctx, today)
	if err != nil {
		t.Fatalf("ListDue() error: %v", err)
	}
	if len(due) != 1 {
		t.Errorf("len(due) = %d, want 1", len(due))
	}
	if len(due) > 0 && due[0].ID != ri1.ID {
		t.Errorf("due[0].ID = %d, want %d", due[0].ID, ri1.ID)
	}
	// Items should be loaded.
	if len(due) > 0 && len(due[0].Items) != 1 {
		t.Errorf("len(due[0].Items) = %d, want 1", len(due[0].Items))
	}
}

func TestRecurringInvoiceRepository_Deactivate(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	ri := makeRecurringInvoice(customer.ID)
	repo.Create(ctx, ri)

	if err := repo.Deactivate(ctx, ri.ID); err != nil {
		t.Fatalf("Deactivate() error: %v", err)
	}

	got, err := repo.GetByID(ctx, ri.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.IsActive {
		t.Error("expected IsActive to be false after deactivation")
	}
}
