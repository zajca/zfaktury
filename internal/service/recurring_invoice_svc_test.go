package service

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func newRecurringInvoiceTestStack(t *testing.T) (*RecurringInvoiceService, *InvoiceService, func() int64) {
	t.Helper()
	db := testutil.NewTestDB(t)
	contactRepo := repository.NewContactRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	sequenceRepo := repository.NewSequenceRepository(db)
	recurringRepo := repository.NewRecurringInvoiceRepository(db)
	contactSvc := NewContactService(contactRepo, nil, nil)
	sequenceSvc := NewSequenceService(sequenceRepo, nil)
	invoiceSvc := NewInvoiceService(invoiceRepo, contactSvc, sequenceSvc, nil)
	recurringSvc := NewRecurringInvoiceService(recurringRepo, invoiceSvc, nil)

	// Seed a default invoice sequence.
	testutil.SeedInvoiceSequence(t, db, "FV", 2026)

	createCustomer := func() int64 {
		c := &domain.Contact{Name: "Test Customer", Type: domain.ContactTypeCompany}
		if err := contactSvc.Create(context.Background(), c); err != nil {
			t.Fatalf("creating customer: %v", err)
		}
		return c.ID
	}

	return recurringSvc, invoiceSvc, createCustomer
}

func makeTestRecurringInvoice(customerID int64) *domain.RecurringInvoice {
	return &domain.RecurringInvoice{
		Name:          "Monthly hosting",
		CustomerID:    customerID,
		Frequency:     domain.FrequencyMonthly,
		NextIssueDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
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

func TestRecurringInvoiceService_Create_Valid(t *testing.T) {
	svc, _, createCustomer := newRecurringInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	ri := makeTestRecurringInvoice(customerID)
	if err := svc.Create(ctx, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if ri.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestRecurringInvoiceService_Create_NoName(t *testing.T) {
	svc, _, createCustomer := newRecurringInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	ri := makeTestRecurringInvoice(customerID)
	ri.Name = ""
	err := svc.Create(ctx, ri)
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestRecurringInvoiceService_Create_NoCustomer(t *testing.T) {
	svc, _, _ := newRecurringInvoiceTestStack(t)
	ctx := context.Background()

	ri := makeTestRecurringInvoice(0)
	err := svc.Create(ctx, ri)
	if err == nil {
		t.Error("expected error for missing customer")
	}
}

func TestRecurringInvoiceService_Create_NoItems(t *testing.T) {
	svc, _, createCustomer := newRecurringInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	ri := makeTestRecurringInvoice(customerID)
	ri.Items = nil
	err := svc.Create(ctx, ri)
	if err == nil {
		t.Error("expected error for empty items")
	}
}

func TestRecurringInvoiceService_GenerateInvoice(t *testing.T) {
	svc, _, createCustomer := newRecurringInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	ri := makeTestRecurringInvoice(customerID)
	if err := svc.Create(ctx, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	inv, err := svc.GenerateInvoice(ctx, ri.ID)
	if err != nil {
		t.Fatalf("GenerateInvoice() error: %v", err)
	}
	if inv.ID == 0 {
		t.Error("expected non-zero invoice ID")
	}
	if inv.Status != domain.InvoiceStatusDraft {
		t.Errorf("Status = %q, want %q", inv.Status, domain.InvoiceStatusDraft)
	}
	if inv.CustomerID != customerID {
		t.Errorf("CustomerID = %d, want %d", inv.CustomerID, customerID)
	}
	if len(inv.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(inv.Items))
	}
	if inv.Items[0].Description != "Web hosting" {
		t.Errorf("Items[0].Description = %q, want %q", inv.Items[0].Description, "Web hosting")
	}
	if inv.TotalAmount == 0 {
		t.Error("expected non-zero TotalAmount")
	}
}

func TestRecurringInvoiceService_ProcessDue(t *testing.T) {
	svc, invoiceSvc, createCustomer := newRecurringInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	// Create a recurring invoice due today (March 10, 2026).
	ri := makeTestRecurringInvoice(customerID)
	ri.NextIssueDate = time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	if err := svc.Create(ctx, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	count, err := svc.ProcessDue(ctx)
	if err != nil {
		t.Fatalf("ProcessDue() error: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	// Verify the recurring invoice's next_issue_date was advanced.
	updated, err := svc.GetByID(ctx, ri.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	expectedNext := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	if !updated.NextIssueDate.Equal(expectedNext) {
		t.Errorf("NextIssueDate = %v, want %v", updated.NextIssueDate, expectedNext)
	}

	// Verify an invoice was created.
	invoices, total, err := invoiceSvc.List(ctx, domain.InvoiceFilter{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 {
		t.Errorf("total invoices = %d, want 1", total)
	}
	if len(invoices) > 0 && invoices[0].CustomerID != customerID {
		t.Errorf("invoice CustomerID = %d, want %d", invoices[0].CustomerID, customerID)
	}
}

func TestRecurringInvoiceService_ProcessDue_PastEndDate(t *testing.T) {
	svc, _, createCustomer := newRecurringInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	endDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	ri := makeTestRecurringInvoice(customerID)
	ri.NextIssueDate = time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
	ri.EndDate = &endDate
	if err := svc.Create(ctx, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	count, err := svc.ProcessDue(ctx)
	if err != nil {
		t.Fatalf("ProcessDue() error: %v", err)
	}
	// Should deactivate, not generate.
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}

	updated, err := svc.GetByID(ctx, ri.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if updated.IsActive {
		t.Error("expected IsActive to be false after past end_date")
	}
}

func TestRecurringInvoiceService_ProcessDue_DeactivatesAfterLastCycle(t *testing.T) {
	svc, _, createCustomer := newRecurringInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	// End date is before next cycle would be.
	endDate := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	ri := makeTestRecurringInvoice(customerID)
	ri.NextIssueDate = time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	ri.EndDate = &endDate
	if err := svc.Create(ctx, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	count, err := svc.ProcessDue(ctx)
	if err != nil {
		t.Fatalf("ProcessDue() error: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	updated, err := svc.GetByID(ctx, ri.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	// Next issue date (April 10) > end date (March 20), should be deactivated.
	if updated.IsActive {
		t.Error("expected IsActive to be false after last cycle")
	}
}

// --- Update tests ---

func TestRecurringInvoiceService_Update_Valid(t *testing.T) {
	svc, _, createCustomer := newRecurringInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	ri := makeTestRecurringInvoice(customerID)
	if err := svc.Create(ctx, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	ri.Name = "Updated hosting"
	ri.Frequency = domain.FrequencyQuarterly
	ri.Items[0].Description = "Premium hosting"

	if err := svc.Update(ctx, ri); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := svc.GetByID(ctx, ri.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Name != "Updated hosting" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated hosting")
	}
	if got.Frequency != domain.FrequencyQuarterly {
		t.Errorf("Frequency = %q, want %q", got.Frequency, domain.FrequencyQuarterly)
	}
}

func TestRecurringInvoiceService_Update_ZeroID(t *testing.T) {
	svc, _, _ := newRecurringInvoiceTestStack(t)
	ctx := context.Background()

	ri := makeTestRecurringInvoice(1)
	ri.ID = 0
	err := svc.Update(ctx, ri)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestRecurringInvoiceService_Update_EmptyName(t *testing.T) {
	svc, _, createCustomer := newRecurringInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	ri := makeTestRecurringInvoice(customerID)
	if err := svc.Create(ctx, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	ri.Name = ""
	err := svc.Update(ctx, ri)
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestRecurringInvoiceService_Update_NoCustomer(t *testing.T) {
	svc, _, createCustomer := newRecurringInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	ri := makeTestRecurringInvoice(customerID)
	if err := svc.Create(ctx, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	ri.CustomerID = 0
	err := svc.Update(ctx, ri)
	if err == nil {
		t.Error("expected error for zero customer ID")
	}
}

func TestRecurringInvoiceService_Update_NoItems(t *testing.T) {
	svc, _, createCustomer := newRecurringInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	ri := makeTestRecurringInvoice(customerID)
	if err := svc.Create(ctx, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	ri.Items = nil
	err := svc.Update(ctx, ri)
	if err == nil {
		t.Error("expected error for empty items")
	}
}

func TestRecurringInvoiceService_Update_NotFound(t *testing.T) {
	svc, _, createCustomer := newRecurringInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	ri := makeTestRecurringInvoice(customerID)
	ri.ID = 99999
	err := svc.Update(ctx, ri)
	if err == nil {
		t.Error("expected error for non-existent recurring invoice")
	}
}

// --- Delete tests ---

func TestRecurringInvoiceService_Delete_Valid(t *testing.T) {
	svc, _, createCustomer := newRecurringInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	ri := makeTestRecurringInvoice(customerID)
	if err := svc.Create(ctx, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.Delete(ctx, ri.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// Verify it's soft-deleted (GetByID should fail).
	_, err := svc.GetByID(ctx, ri.ID)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestRecurringInvoiceService_Delete_ZeroID(t *testing.T) {
	svc, _, _ := newRecurringInvoiceTestStack(t)
	ctx := context.Background()

	err := svc.Delete(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestRecurringInvoiceService_Delete_NotFound(t *testing.T) {
	svc, _, _ := newRecurringInvoiceTestStack(t)
	ctx := context.Background()

	err := svc.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent recurring invoice")
	}
}

// --- List tests ---

func TestRecurringInvoiceService_List_Empty(t *testing.T) {
	svc, _, _ := newRecurringInvoiceTestStack(t)
	ctx := context.Background()

	items, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("List() returned %d items, want 0", len(items))
	}
}

func TestRecurringInvoiceService_List_MultipleItems(t *testing.T) {
	svc, _, createCustomer := newRecurringInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	for i := 0; i < 3; i++ {
		ri := makeTestRecurringInvoice(customerID)
		ri.Name = "Recurring " + string(rune('A'+i))
		if err := svc.Create(ctx, ri); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	items, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("List() returned %d items, want 3", len(items))
	}
}

func TestRecurringInvoiceService_List_ExcludesDeleted(t *testing.T) {
	svc, _, createCustomer := newRecurringInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	ri1 := makeTestRecurringInvoice(customerID)
	ri1.Name = "Keep"
	if err := svc.Create(ctx, ri1); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	ri2 := makeTestRecurringInvoice(customerID)
	ri2.Name = "Delete me"
	if err := svc.Create(ctx, ri2); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.Delete(ctx, ri2.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	items, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("List() returned %d items, want 1 (deleted should be excluded)", len(items))
	}
	if items[0].Name != "Keep" {
		t.Errorf("remaining item Name = %q, want %q", items[0].Name, "Keep")
	}
}

func TestRecurringInvoiceService_GetByID_Valid(t *testing.T) {
	svc, _, createCustomer := newRecurringInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	ri := makeTestRecurringInvoice(customerID)
	if err := svc.Create(ctx, ri); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := svc.GetByID(ctx, ri.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Name != "Monthly hosting" {
		t.Errorf("Name = %q, want %q", got.Name, "Monthly hosting")
	}
	if got.CustomerID != customerID {
		t.Errorf("CustomerID = %d, want %d", got.CustomerID, customerID)
	}
}

func TestRecurringInvoiceService_GetByID_ZeroID(t *testing.T) {
	svc, _, _ := newRecurringInvoiceTestStack(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestRecurringInvoiceService_GetByID_NotFound(t *testing.T) {
	svc, _, _ := newRecurringInvoiceTestStack(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent ID")
	}
}
