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

func TestRecurringInvoiceRepository_Deactivate_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	err := repo.Deactivate(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent recurring invoice")
	}
}

func TestRecurringInvoiceRepository_Deactivate_AlreadyDeleted(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	ri := makeRecurringInvoice(customer.ID)
	repo.Create(ctx, ri)
	repo.Delete(ctx, ri.ID)

	err := repo.Deactivate(ctx, ri.ID)
	if err == nil {
		t.Error("expected error when deactivating a deleted recurring invoice")
	}
}

func TestRecurringInvoiceRepository_Delete_AlreadyDeleted(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	ri := makeRecurringInvoice(customer.ID)
	repo.Create(ctx, ri)
	repo.Delete(ctx, ri.ID)

	err := repo.Delete(ctx, ri.ID)
	if err == nil {
		t.Error("expected error when deleting already deleted recurring invoice")
	}
}

func TestRecurringInvoiceRepository_Create_WithEndDate(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	endDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	ri := makeRecurringInvoice(customer.ID)
	ri.EndDate = &endDate

	if err := repo.Create(ctx, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, ri.ID)
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

func TestRecurringInvoiceRepository_Update_WithEndDate(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	ri := makeRecurringInvoice(customer.ID)
	repo.Create(ctx, ri)

	endDate := time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC)
	ri.EndDate = &endDate
	ri.Name = "Updated with end date"

	if err := repo.Update(ctx, ri); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, ri.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.EndDate == nil {
		t.Fatal("expected EndDate to be set after update")
	}
	if got.EndDate.Format("2006-01-02") != "2027-06-30" {
		t.Errorf("EndDate = %s, want 2027-06-30", got.EndDate.Format("2006-01-02"))
	}
}

func TestRecurringInvoiceRepository_Update_MultipleItems(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	ri := makeRecurringInvoice(customer.ID)
	repo.Create(ctx, ri)

	// Update with multiple items to ensure all items get IDs assigned.
	ri.Items = []domain.RecurringInvoiceItem{
		{Description: "Item A", Quantity: 100, Unit: "ks", UnitPrice: 10000, VATRatePercent: 21, SortOrder: 0},
		{Description: "Item B", Quantity: 200, Unit: "hod", UnitPrice: 20000, VATRatePercent: 15, SortOrder: 1},
		{Description: "Item C", Quantity: 300, Unit: "ks", UnitPrice: 30000, VATRatePercent: 0, SortOrder: 2},
	}

	if err := repo.Update(ctx, ri); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	for i, item := range ri.Items {
		if item.ID == 0 {
			t.Errorf("Items[%d].ID = 0, expected non-zero", i)
		}
		if item.RecurringInvoiceID != ri.ID {
			t.Errorf("Items[%d].RecurringInvoiceID = %d, want %d", i, item.RecurringInvoiceID, ri.ID)
		}
	}

	got, err := repo.GetByID(ctx, ri.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if len(got.Items) != 3 {
		t.Errorf("len(Items) = %d, want 3", len(got.Items))
	}
}

func TestRecurringInvoiceRepository_Create_MultipleItems(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	ri := makeRecurringInvoice(customer.ID)
	ri.Items = []domain.RecurringInvoiceItem{
		{Description: "Item 1", Quantity: 100, Unit: "ks", UnitPrice: 10000, VATRatePercent: 21, SortOrder: 0},
		{Description: "Item 2", Quantity: 200, Unit: "hod", UnitPrice: 20000, VATRatePercent: 15, SortOrder: 1},
	}

	if err := repo.Create(ctx, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	for i, item := range ri.Items {
		if item.ID == 0 {
			t.Errorf("Items[%d].ID = 0, expected non-zero", i)
		}
		if item.RecurringInvoiceID != ri.ID {
			t.Errorf("Items[%d].RecurringInvoiceID = %d, want %d", i, item.RecurringInvoiceID, ri.ID)
		}
	}
}

func TestRecurringInvoiceRepository_List_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if list != nil {
		t.Errorf("expected nil list for empty database, got %d items", len(list))
	}
}

func TestRecurringInvoiceRepository_List_ExcludesSoftDeleted(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	ri1 := makeRecurringInvoice(customer.ID)
	ri1.Name = "Keep me"
	repo.Create(ctx, ri1)

	ri2 := makeRecurringInvoice(customer.ID)
	ri2.Name = "Delete me"
	repo.Create(ctx, ri2)

	repo.Delete(ctx, ri2.ID)

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	if list[0].Name != "Keep me" {
		t.Errorf("Name = %q, want %q", list[0].Name, "Keep me")
	}
}

func TestRecurringInvoiceRepository_List_WithCustomerData(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Acme Corp"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	ri := makeRecurringInvoice(customer.ID)
	repo.Create(ctx, ri)

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	if list[0].Customer == nil {
		t.Fatal("expected Customer to be populated in list")
	}
	if list[0].Customer.Name != "Acme Corp" {
		t.Errorf("Customer.Name = %q, want %q", list[0].Customer.Name, "Acme Corp")
	}
}

func TestRecurringInvoiceRepository_List_WithEndDate(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	endDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	ri := makeRecurringInvoice(customer.ID)
	ri.EndDate = &endDate
	repo.Create(ctx, ri)

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	if list[0].EndDate == nil {
		t.Fatal("expected EndDate to be set in list result")
	}
	if list[0].EndDate.Format("2006-01-02") != "2026-12-31" {
		t.Errorf("EndDate = %s, want 2026-12-31", list[0].EndDate.Format("2006-01-02"))
	}
}

func TestRecurringInvoiceRepository_ListDue_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	today := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	due, err := repo.ListDue(ctx, today)
	if err != nil {
		t.Fatalf("ListDue() error: %v", err)
	}
	if due != nil {
		t.Errorf("expected nil list for empty database, got %d items", len(due))
	}
}

func TestRecurringInvoiceRepository_ListDue_WithEndDate(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	endDate := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	ri := makeRecurringInvoice(customer.ID)
	ri.NextIssueDate = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	ri.EndDate = &endDate
	repo.Create(ctx, ri)

	today := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	due, err := repo.ListDue(ctx, today)
	if err != nil {
		t.Fatalf("ListDue() error: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("len(due) = %d, want 1", len(due))
	}
	if due[0].EndDate == nil {
		t.Fatal("expected EndDate to be set in due result")
	}
}

func TestRecurringInvoiceRepository_ListDue_MultipleWithItems(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	ri1 := makeRecurringInvoice(customer.ID)
	ri1.Name = "Due 1"
	ri1.NextIssueDate = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	ri1.Items = []domain.RecurringInvoiceItem{
		{Description: "Item A", Quantity: 100, Unit: "ks", UnitPrice: 10000, VATRatePercent: 21, SortOrder: 0},
		{Description: "Item B", Quantity: 200, Unit: "hod", UnitPrice: 20000, VATRatePercent: 15, SortOrder: 1},
	}
	repo.Create(ctx, ri1)

	ri2 := makeRecurringInvoice(customer.ID)
	ri2.Name = "Due 2"
	ri2.NextIssueDate = time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	repo.Create(ctx, ri2)

	today := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	due, err := repo.ListDue(ctx, today)
	if err != nil {
		t.Fatalf("ListDue() error: %v", err)
	}
	if len(due) != 2 {
		t.Fatalf("len(due) = %d, want 2", len(due))
	}
	// First one should have 2 items.
	if len(due[0].Items) != 2 {
		t.Errorf("due[0] items = %d, want 2", len(due[0].Items))
	}
	// Second one should have 1 item.
	if len(due[1].Items) != 1 {
		t.Errorf("due[1] items = %d, want 1", len(due[1].Items))
	}
}

func TestRecurringInvoiceRepository_List_NoCustomerMatch(t *testing.T) {
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	repo := NewRecurringInvoiceRepository(db)
	ctx := context.Background()

	ri := makeRecurringInvoice(customer.ID)
	repo.Create(ctx, ri)

	// Verify List returns customer data populated when customer exists.
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	if list[0].Customer == nil {
		t.Fatal("expected Customer to be populated")
	}
	if list[0].Customer.ID != customer.ID {
		t.Errorf("Customer.ID = %d, want %d", list[0].Customer.ID, customer.ID)
	}
}
