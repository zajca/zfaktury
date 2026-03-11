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

func TestInvoiceService_Create_NoDueDate(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	inv := makeInvoice(customerID)
	inv.DueDate = time.Time{} // zero value

	err := svc.Create(ctx, inv)
	if err == nil {
		t.Fatal("expected error for missing due date, got nil")
	}
	if !strings.Contains(err.Error(), "due date") {
		t.Errorf("error should mention 'due date', got: %v", err)
	}
}

func TestInvoiceService_Update_NoDueDate(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	inv := makeInvoice(customerID)
	if err := svc.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	inv.DueDate = time.Time{} // zero value

	err := svc.Update(ctx, inv)
	if err == nil {
		t.Fatal("expected error for missing due date, got nil")
	}
	if !strings.Contains(err.Error(), "due date") {
		t.Errorf("error should mention 'due date', got: %v", err)
	}
}

func TestInvoiceService_GetByID_ZeroID(t *testing.T) {
	svc, _, _, _, _ := newInvoiceTestStack(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
	if !strings.Contains(err.Error(), "ID is required") {
		t.Errorf("error should mention 'ID is required', got: %v", err)
	}
}

func TestInvoiceService_GetByID_NotFound(t *testing.T) {
	svc, _, _, _, _ := newInvoiceTestStack(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent invoice")
	}
}

func TestInvoiceService_GetByID_Valid(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	inv := makeInvoice(customerID)
	if err := svc.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := svc.GetByID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.ID != inv.ID {
		t.Errorf("ID = %d, want %d", got.ID, inv.ID)
	}
	if got.CustomerID != customerID {
		t.Errorf("CustomerID = %d, want %d", got.CustomerID, customerID)
	}
}

func TestInvoiceService_Delete_ZeroID(t *testing.T) {
	svc, _, _, _, _ := newInvoiceTestStack(t)
	ctx := context.Background()

	err := svc.Delete(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
	if !strings.Contains(err.Error(), "ID is required") {
		t.Errorf("error should mention 'ID is required', got: %v", err)
	}
}

func TestInvoiceService_Delete_NotFound(t *testing.T) {
	svc, _, _, _, _ := newInvoiceTestStack(t)
	ctx := context.Background()

	err := svc.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent invoice")
	}
}

func TestInvoiceService_Delete_DraftInvoice(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	inv := makeInvoice(customerID)
	if err := svc.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.Delete(ctx, inv.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// Verify it's gone.
	_, err := svc.GetByID(ctx, inv.ID)
	if err == nil {
		t.Error("expected error after deleting invoice")
	}
}

func TestInvoiceService_SettleProforma_ZeroID(t *testing.T) {
	svc, _, _, _, _ := newInvoiceTestStack(t)
	ctx := context.Background()

	_, err := svc.SettleProforma(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestInvoiceService_SettleProforma_NotProforma(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	// Create a regular invoice (not proforma).
	inv := makeInvoice(customerID)
	if err := svc.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	_, err := svc.SettleProforma(ctx, inv.ID)
	if err == nil {
		t.Error("expected error for non-proforma invoice")
	}
	if !strings.Contains(err.Error(), "not a proforma") {
		t.Errorf("error should mention 'not a proforma', got: %v", err)
	}
}

func TestInvoiceService_SettleProforma_UnpaidProforma(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	// Create proforma (draft, not paid). Let auto-sequence handle the number.
	proforma := makeInvoice(customerID)
	proforma.Type = domain.InvoiceTypeProforma
	proforma.SequenceID = 0
	proforma.InvoiceNumber = ""
	if err := svc.Create(ctx, proforma); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	_, err := svc.SettleProforma(ctx, proforma.ID)
	if err == nil {
		t.Error("expected error for unpaid proforma")
	}
	if !strings.Contains(err.Error(), "paid") {
		t.Errorf("error should mention 'paid', got: %v", err)
	}
}

func TestInvoiceService_SettleProforma_Valid(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	// Create and pay a proforma.
	proforma := makeInvoice(customerID)
	proforma.Type = domain.InvoiceTypeProforma
	proforma.InvoiceNumber = "ZF20260001"
	if err := svc.Create(ctx, proforma); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.MarkAsPaid(ctx, proforma.ID, proforma.TotalAmount, time.Now()); err != nil {
		t.Fatalf("MarkAsPaid() error: %v", err)
	}

	// Settle the proforma.
	settlement, err := svc.SettleProforma(ctx, proforma.ID)
	if err != nil {
		t.Fatalf("SettleProforma() error: %v", err)
	}

	if settlement.ID == 0 {
		t.Error("expected non-zero settlement ID")
	}
	if settlement.Type != domain.InvoiceTypeRegular {
		t.Errorf("Type = %q, want %q", settlement.Type, domain.InvoiceTypeRegular)
	}
	if settlement.Status != domain.InvoiceStatusDraft {
		t.Errorf("Status = %q, want %q", settlement.Status, domain.InvoiceStatusDraft)
	}
	if settlement.RelatedInvoiceID == nil || *settlement.RelatedInvoiceID != proforma.ID {
		t.Error("settlement should reference the proforma")
	}
	if settlement.RelationType != domain.RelationTypeSettlement {
		t.Errorf("RelationType = %q, want %q", settlement.RelationType, domain.RelationTypeSettlement)
	}
	if settlement.CustomerID != customerID {
		t.Errorf("CustomerID = %d, want %d", settlement.CustomerID, customerID)
	}
	if len(settlement.Items) != len(proforma.Items) {
		t.Errorf("len(Items) = %d, want %d", len(settlement.Items), len(proforma.Items))
	}
}

func TestInvoiceService_SettleProforma_Idempotent(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	// Create and pay a proforma.
	proforma := makeInvoice(customerID)
	proforma.Type = domain.InvoiceTypeProforma
	proforma.InvoiceNumber = "ZF20260002"
	if err := svc.Create(ctx, proforma); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := svc.MarkAsPaid(ctx, proforma.ID, proforma.TotalAmount, time.Now()); err != nil {
		t.Fatalf("MarkAsPaid() error: %v", err)
	}

	// Settle twice.
	first, err := svc.SettleProforma(ctx, proforma.ID)
	if err != nil {
		t.Fatalf("first SettleProforma() error: %v", err)
	}
	second, err := svc.SettleProforma(ctx, proforma.ID)
	if err != nil {
		t.Fatalf("second SettleProforma() error: %v", err)
	}

	if first.ID != second.ID {
		t.Errorf("idempotent settle returned different IDs: %d vs %d", first.ID, second.ID)
	}
}

func TestInvoiceService_CreateCreditNote_ZeroID(t *testing.T) {
	svc, _, _, _, _ := newInvoiceTestStack(t)
	ctx := context.Background()

	_, err := svc.CreateCreditNote(ctx, 0, nil, "reason")
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestInvoiceService_CreateCreditNote_NotRegular(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	// Create a proforma invoice.
	proforma := makeInvoice(customerID)
	proforma.Type = domain.InvoiceTypeProforma
	proforma.InvoiceNumber = "ZF20260003"
	if err := svc.Create(ctx, proforma); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	_, err := svc.CreateCreditNote(ctx, proforma.ID, nil, "reason")
	if err == nil {
		t.Error("expected error for non-regular invoice")
	}
	if !strings.Contains(err.Error(), "regular invoices") {
		t.Errorf("error should mention 'regular invoices', got: %v", err)
	}
}

func TestInvoiceService_CreateCreditNote_DraftInvoice(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	// Create a draft regular invoice.
	inv := makeInvoice(customerID)
	if err := svc.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	_, err := svc.CreateCreditNote(ctx, inv.ID, nil, "reason")
	if err == nil {
		t.Error("expected error for draft invoice")
	}
	if !strings.Contains(err.Error(), "sent or paid") {
		t.Errorf("error should mention 'sent or paid', got: %v", err)
	}
}

func TestInvoiceService_CreateCreditNote_FullCreditNote(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	// Create and send a regular invoice.
	inv := makeInvoice(customerID)
	if err := svc.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := svc.MarkAsSent(ctx, inv.ID); err != nil {
		t.Fatalf("MarkAsSent() error: %v", err)
	}

	// Create full credit note (no items provided).
	cn, err := svc.CreateCreditNote(ctx, inv.ID, nil, "full refund")
	if err != nil {
		t.Fatalf("CreateCreditNote() error: %v", err)
	}

	if cn.Type != domain.InvoiceTypeCreditNote {
		t.Errorf("Type = %q, want %q", cn.Type, domain.InvoiceTypeCreditNote)
	}
	if cn.RelatedInvoiceID == nil || *cn.RelatedInvoiceID != inv.ID {
		t.Error("credit note should reference the original invoice")
	}
	if cn.RelationType != domain.RelationTypeCreditNote {
		t.Errorf("RelationType = %q, want %q", cn.RelationType, domain.RelationTypeCreditNote)
	}
	if cn.Notes != "full refund" {
		t.Errorf("Notes = %q, want %q", cn.Notes, "full refund")
	}
	if len(cn.Items) != len(inv.Items) {
		t.Fatalf("len(Items) = %d, want %d", len(cn.Items), len(inv.Items))
	}
	// Verify prices are negated.
	for i, item := range cn.Items {
		originalPrice := inv.Items[i].UnitPrice
		if item.UnitPrice != originalPrice*-1 {
			t.Errorf("item %d UnitPrice = %d, want %d (negated)", i, item.UnitPrice, originalPrice*-1)
		}
	}
}

func TestInvoiceService_CreateCreditNote_PartialCreditNote(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	// Create, send, and pay a regular invoice.
	inv := makeInvoice(customerID)
	if err := svc.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := svc.MarkAsPaid(ctx, inv.ID, inv.TotalAmount, time.Now()); err != nil {
		t.Fatalf("MarkAsPaid() error: %v", err)
	}

	// Create partial credit note with custom items.
	partialItems := []domain.InvoiceItem{
		{
			Description:    "Partial refund item",
			Quantity:       100,
			Unit:           "ks",
			UnitPrice:      50000,
			VATRatePercent: 21,
		},
	}
	cn, err := svc.CreateCreditNote(ctx, inv.ID, partialItems, "partial refund")
	if err != nil {
		t.Fatalf("CreateCreditNote() error: %v", err)
	}

	if len(cn.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(cn.Items))
	}
	// Partial credit note items should also have negated prices.
	if cn.Items[0].UnitPrice != -50000 {
		t.Errorf("item UnitPrice = %d, want -50000", cn.Items[0].UnitPrice)
	}
	if cn.Items[0].Description != "Partial refund item" {
		t.Errorf("item Description = %q, want %q", cn.Items[0].Description, "Partial refund item")
	}
}

func TestInvoiceService_GetRelatedInvoices_Empty(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	inv := makeInvoice(customerID)
	if err := svc.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	related, err := svc.GetRelatedInvoices(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetRelatedInvoices() error: %v", err)
	}
	if len(related) != 0 {
		t.Errorf("len(related) = %d, want 0", len(related))
	}
}

func TestInvoiceService_GetRelatedInvoices_WithCreditNote(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	// Create and send a regular invoice.
	inv := makeInvoice(customerID)
	if err := svc.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := svc.MarkAsSent(ctx, inv.ID); err != nil {
		t.Fatalf("MarkAsSent() error: %v", err)
	}

	// Create a credit note for it.
	_, err := svc.CreateCreditNote(ctx, inv.ID, nil, "refund")
	if err != nil {
		t.Fatalf("CreateCreditNote() error: %v", err)
	}

	related, err := svc.GetRelatedInvoices(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetRelatedInvoices() error: %v", err)
	}
	if len(related) != 1 {
		t.Fatalf("len(related) = %d, want 1", len(related))
	}
	if related[0].Type != domain.InvoiceTypeCreditNote {
		t.Errorf("related Type = %q, want %q", related[0].Type, domain.InvoiceTypeCreditNote)
	}
}

func TestInvoiceService_MarkAsSent_ZeroID(t *testing.T) {
	svc, _, _, _, _ := newInvoiceTestStack(t)
	ctx := context.Background()

	err := svc.MarkAsSent(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestInvoiceService_MarkAsPaid_ZeroID(t *testing.T) {
	svc, _, _, _, _ := newInvoiceTestStack(t)
	ctx := context.Background()

	err := svc.MarkAsPaid(ctx, 0, 100, time.Now())
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestInvoiceService_Update_PreservesInvoiceNumber(t *testing.T) {
	svc, _, _, _, createCustomer := newInvoiceTestStack(t)
	ctx := context.Background()
	customerID := createCustomer()

	inv := makeInvoice(customerID)
	if err := svc.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	originalNumber := inv.InvoiceNumber
	originalSeqID := inv.SequenceID
	originalVS := inv.VariableSymbol
	originalType := inv.Type

	// Update with empty identifying fields -- service should preserve originals.
	inv.InvoiceNumber = ""
	inv.SequenceID = 0
	inv.VariableSymbol = ""
	inv.Type = ""
	inv.Notes = "Updated"

	if err := svc.Update(ctx, inv); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	updated, err := svc.GetByID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if updated.InvoiceNumber != originalNumber {
		t.Errorf("InvoiceNumber = %q, want %q", updated.InvoiceNumber, originalNumber)
	}
	if updated.SequenceID != originalSeqID {
		t.Errorf("SequenceID = %d, want %d", updated.SequenceID, originalSeqID)
	}
	if updated.VariableSymbol != originalVS {
		t.Errorf("VariableSymbol = %q, want %q", updated.VariableSymbol, originalVS)
	}
	if updated.Type != originalType {
		t.Errorf("Type = %q, want %q", updated.Type, originalType)
	}
	if updated.Notes != "Updated" {
		t.Errorf("Notes = %q, want %q", updated.Notes, "Updated")
	}
}
