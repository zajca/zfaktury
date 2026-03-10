package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func newInvoiceTestStack(t *testing.T) (*InvoiceService, *ContactService, *repository.InvoiceRepository, *repository.ContactRepository, func() int64) {
	t.Helper()
	db := testutil.NewTestDB(t)
	contactRepo := repository.NewContactRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	sequenceRepo := repository.NewSequenceRepository(db)
	contactSvc := NewContactService(contactRepo, nil)
	sequenceSvc := NewSequenceService(sequenceRepo)
	invoiceSvc := NewInvoiceService(invoiceRepo, contactSvc, sequenceSvc)

	// Seed a default invoice sequence so SequenceID references are valid.
	testutil.SeedInvoiceSequence(t, db, "FV", 2026)

	// Helper to create a customer quickly.
	createCustomer := func() int64 {
		c := &domain.Contact{Name: "Test Customer", Type: domain.ContactTypeCompany}
		if err := contactSvc.Create(context.Background(), c); err != nil {
			t.Fatalf("creating customer: %v", err)
		}
		return c.ID
	}

	return invoiceSvc, contactSvc, invoiceRepo, contactRepo, createCustomer
}

var invoiceCounter int

func makeInvoice(customerID int64) *domain.Invoice {
	invoiceCounter++
	return &domain.Invoice{
		CustomerID:    customerID,
		SequenceID:    1, // references the seeded sequence
		Type:          domain.InvoiceTypeRegular,
		IssueDate:     time.Now(),
		DueDate:       time.Now().AddDate(0, 0, 14),
		DeliveryDate:  time.Now(),
		CurrencyCode:  domain.CurrencyCZK,
		PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{
				Description:    "Test service",
				Quantity:       100,
				Unit:           "hod",
				UnitPrice:      100000,
				VATRatePercent: 21,
			},
		},
	}
}

func TestInvoiceService_Create_Valid(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	inv := makeInvoice(customerID)
	if err := svc.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if inv.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if inv.Status != domain.InvoiceStatusDraft {
		t.Errorf("Status = %q, want %q", inv.Status, domain.InvoiceStatusDraft)
	}
	if inv.SubtotalAmount == 0 {
		t.Error("expected non-zero SubtotalAmount after CalculateTotals")
	}
}

func TestInvoiceService_Create_NoCustomer(t *testing.T) {
	svc, _, _, _, _ := newInvoiceTestStack(t)
	ctx := context.Background()

	inv := makeInvoice(0)
	err := svc.Create(ctx, inv)
	if err == nil {
		t.Error("expected error for missing customer")
	}
}

func TestInvoiceService_Create_CustomerNotFound(t *testing.T) {
	svc, _, _, _, _ := newInvoiceTestStack(t)
	ctx := context.Background()

	inv := makeInvoice(99999)
	err := svc.Create(ctx, inv)
	if err == nil {
		t.Error("expected error for non-existent customer")
	}
}

func TestInvoiceService_Create_NoItems(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	inv := &domain.Invoice{
		CustomerID: customerID,
		Items:      nil,
	}
	err := svc.Create(ctx, inv)
	if err == nil {
		t.Error("expected error for empty items")
	}
}

func TestInvoiceService_Update_Valid(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	inv := makeInvoice(customerID)
	if err := svc.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	inv.Notes = "Updated notes"
	if err := svc.Update(ctx, inv); err != nil {
		t.Fatalf("Update() error: %v", err)
	}
}

func TestInvoiceService_Update_PaidInvoice(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	inv := makeInvoice(customerID)
	if err := svc.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Mark as paid.
	if err := svc.MarkAsPaid(ctx, inv.ID, inv.TotalAmount, time.Now()); err != nil {
		t.Fatalf("MarkAsPaid() error: %v", err)
	}

	// Try to update -- should fail.
	inv.Notes = "Can't do this"
	err := svc.Update(ctx, inv)
	if err == nil {
		t.Error("expected error when updating paid invoice")
	}
	if !strings.Contains(err.Error(), "paid") {
		t.Errorf("error should mention 'paid', got: %v", err)
	}
}

func TestInvoiceService_Delete_PaidInvoice(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	inv := makeInvoice(customerID)
	if err := svc.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.MarkAsPaid(ctx, inv.ID, inv.TotalAmount, time.Now()); err != nil {
		t.Fatalf("MarkAsPaid() error: %v", err)
	}

	err := svc.Delete(ctx, inv.ID)
	if err == nil {
		t.Error("expected error when deleting paid invoice")
	}
}

func TestInvoiceService_MarkAsSent_DraftOnly(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	inv := makeInvoice(customerID)
	if err := svc.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.MarkAsSent(ctx, inv.ID); err != nil {
		t.Fatalf("MarkAsSent() error: %v", err)
	}

	// Try to mark as sent again -- should fail (already sent, not draft).
	err := svc.MarkAsSent(ctx, inv.ID)
	if err == nil {
		t.Error("expected error when marking non-draft invoice as sent")
	}
}

func TestInvoiceService_MarkAsPaid_AlreadyPaid(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	inv := makeInvoice(customerID)
	if err := svc.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.MarkAsPaid(ctx, inv.ID, inv.TotalAmount, time.Now()); err != nil {
		t.Fatalf("MarkAsPaid() error: %v", err)
	}

	err := svc.MarkAsPaid(ctx, inv.ID, inv.TotalAmount, time.Now())
	if err == nil {
		t.Error("expected error when marking already-paid invoice as paid")
	}
}

func TestInvoiceService_MarkAsPaid_CancelledInvoice(t *testing.T) {
	svc, _, invoiceRepo, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	inv := makeInvoice(customerID)
	if err := svc.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Set status to cancelled directly.
	if err := invoiceRepo.UpdateStatus(ctx, inv.ID, domain.InvoiceStatusCancelled); err != nil {
		t.Fatalf("UpdateStatus() error: %v", err)
	}

	err := svc.MarkAsPaid(ctx, inv.ID, inv.TotalAmount, time.Now())
	if err == nil {
		t.Error("expected error when paying cancelled invoice")
	}
}

func TestInvoiceService_Duplicate(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	original := makeInvoice(customerID)
	original.InvoiceNumber = "FV-ORIG"
	original.Notes = "Original notes"
	if err := svc.Create(ctx, original); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	dup, err := svc.Duplicate(ctx, original.ID)
	if err != nil {
		t.Fatalf("Duplicate() error: %v", err)
	}

	if dup.ID == original.ID {
		t.Error("duplicate should have a different ID")
	}
	if dup.Status != domain.InvoiceStatusDraft {
		t.Errorf("Status = %q, want %q", dup.Status, domain.InvoiceStatusDraft)
	}
	if dup.CustomerID != original.CustomerID {
		t.Errorf("CustomerID = %d, want %d", dup.CustomerID, original.CustomerID)
	}
	if len(dup.Items) != len(original.Items) {
		t.Errorf("len(Items) = %d, want %d", len(dup.Items), len(original.Items))
	}
	if dup.Notes != original.Notes {
		t.Errorf("Notes = %q, want %q", dup.Notes, original.Notes)
	}
}

func TestInvoiceService_Duplicate_ZeroID(t *testing.T) {
	svc, _, _, _, _ := newInvoiceTestStack(t)
	ctx := context.Background()

	_, err := svc.Duplicate(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestInvoiceService_List_DefaultLimit(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	for i := 0; i < 3; i++ {
		inv := makeInvoice(customerID)
		inv.InvoiceNumber = "FV-LIST-" + string(rune('A'+i))
		if err := svc.Create(ctx, inv); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	invoices, total, err := svc.List(ctx, domain.InvoiceFilter{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(invoices) != 3 {
		t.Errorf("len = %d, want 3", len(invoices))
	}
}
