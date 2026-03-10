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
	contactSvc := NewContactService(contactRepo, nil)
	sequenceSvc := NewSequenceService(sequenceRepo)
	invoiceSvc := NewInvoiceService(invoiceRepo, contactSvc, sequenceSvc)
	recurringSvc := NewRecurringInvoiceService(recurringRepo, invoiceSvc)

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
