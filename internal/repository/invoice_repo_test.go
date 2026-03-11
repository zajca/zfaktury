package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

// setupInvoiceTestDB creates a test DB and seeds a contact and invoice sequence.
func setupInvoiceTestDB(t *testing.T) (*sql.DB, int64, int64) {
	t.Helper()
	db := testutil.NewTestDB(t)
	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Test Customer"})
	seqID := testutil.SeedInvoiceSequence(t, db, "FV", 2026)
	return db, customer.ID, seqID
}

var invoiceRepoCounter int

func makeRepoInvoice(customerID, seqID int64) *domain.Invoice {
	invoiceRepoCounter++
	inv := &domain.Invoice{
		SequenceID:    seqID,
		InvoiceNumber: fmt.Sprintf("FV2026%04d", invoiceRepoCounter),
		Type:          domain.InvoiceTypeRegular,
		Status:        domain.InvoiceStatusDraft,
		IssueDate:     time.Now(),
		DueDate:       time.Now().AddDate(0, 0, 14),
		DeliveryDate:  time.Now(),
		CustomerID:    customerID,
		CurrencyCode:  domain.CurrencyCZK,
		ExchangeRate:  100,
		PaymentMethod: "bank_transfer",
		Items: []domain.InvoiceItem{
			{
				Description:    "Web development",
				Quantity:       100,
				Unit:           "hod",
				UnitPrice:      150000,
				VATRatePercent: 21,
			},
		},
	}
	inv.CalculateTotals()
	return inv
}

func TestInvoiceRepository_Create(t *testing.T) {
	db, customerID, seqID := setupInvoiceTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	inv := makeRepoInvoice(customerID, seqID)

	if err := repo.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if inv.ID == 0 {
		t.Error("expected non-zero invoice ID")
	}
	if inv.Items[0].ID == 0 {
		t.Error("expected non-zero item ID")
	}
	if inv.Items[0].InvoiceID != inv.ID {
		t.Errorf("item.InvoiceID = %d, want %d", inv.Items[0].InvoiceID, inv.ID)
	}
}

func TestInvoiceRepository_GetByID(t *testing.T) {
	db, customerID, seqID := setupInvoiceTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	// Update customer with more details for JOIN testing.
	testutil.SeedContact(t, db, &domain.Contact{Name: "Customer B", ICO: "22222222"})

	inv := makeRepoInvoice(customerID, seqID)
	inv.Items = append(inv.Items, domain.InvoiceItem{
		Description: "Item 2", Quantity: 200, Unit: "hod", UnitPrice: 20000, VATRatePercent: 12, SortOrder: 1,
	})
	inv.CalculateTotals()

	if err := repo.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.InvoiceNumber != inv.InvoiceNumber {
		t.Errorf("InvoiceNumber = %q, want %q", got.InvoiceNumber, inv.InvoiceNumber)
	}

	// Customer should be joined.
	if got.Customer == nil {
		t.Fatal("expected Customer to be populated")
	}
	if got.Customer.Name != "Test Customer" {
		t.Errorf("Customer.Name = %q, want %q", got.Customer.Name, "Test Customer")
	}

	// Items should be loaded.
	if len(got.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(got.Items))
	}
	if got.Items[0].Description != "Web development" {
		t.Errorf("Items[0].Description = %q, want %q", got.Items[0].Description, "Web development")
	}
}

func TestInvoiceRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent invoice")
	}
}

func TestInvoiceRepository_Update(t *testing.T) {
	db, customerID, seqID := setupInvoiceTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	inv := makeRepoInvoice(customerID, seqID)
	if err := repo.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Update: replace items.
	inv.Notes = "Updated note"
	inv.Items = []domain.InvoiceItem{
		{Description: "Replaced A", Quantity: 300, Unit: "hod", UnitPrice: 50000, VATRatePercent: 21, SortOrder: 0},
		{Description: "Replaced B", Quantity: 100, Unit: "ks", UnitPrice: 20000, VATRatePercent: 0, SortOrder: 1},
	}
	inv.CalculateTotals()

	if err := repo.Update(ctx, inv); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Notes != "Updated note" {
		t.Errorf("Notes = %q, want %q", got.Notes, "Updated note")
	}
	if len(got.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(got.Items))
	}
	if got.Items[0].Description != "Replaced A" {
		t.Errorf("Items[0].Description = %q, want %q", got.Items[0].Description, "Replaced A")
	}
}

func TestInvoiceRepository_Delete(t *testing.T) {
	db, customerID, seqID := setupInvoiceTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	inv := makeRepoInvoice(customerID, seqID)
	if err := repo.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := repo.Delete(ctx, inv.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := repo.GetByID(ctx, inv.ID)
	if err == nil {
		t.Error("expected error when getting deleted invoice")
	}
}

func TestInvoiceRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent invoice")
	}
}

func TestInvoiceRepository_List_All(t *testing.T) {
	db, customerID, seqID := setupInvoiceTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	inv1 := makeRepoInvoice(customerID, seqID)
	inv2 := makeRepoInvoice(customerID, seqID)
	repo.Create(ctx, inv1)
	repo.Create(ctx, inv2)

	invoices, total, err := repo.List(ctx, domain.InvoiceFilter{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(invoices) != 2 {
		t.Errorf("len(invoices) = %d, want 2", len(invoices))
	}
}

func TestInvoiceRepository_List_StatusFilter(t *testing.T) {
	db, customerID, seqID := setupInvoiceTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	inv1 := makeRepoInvoice(customerID, seqID)
	repo.Create(ctx, inv1)

	inv2 := makeRepoInvoice(customerID, seqID)
	repo.Create(ctx, inv2)
	repo.UpdateStatus(ctx, inv2.ID, domain.InvoiceStatusSent)

	invoices, total, err := repo.List(ctx, domain.InvoiceFilter{Status: domain.InvoiceStatusSent})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(invoices) != 1 {
		t.Errorf("len = %d, want 1", len(invoices))
	}
}

func TestInvoiceRepository_List_SearchFilter(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Searchable Corp"})
	seqID := testutil.SeedInvoiceSequence(t, db, "FV", 2026)

	inv := makeRepoInvoice(customer.ID, seqID)
	repo.Create(ctx, inv)

	invoices, total, err := repo.List(ctx, domain.InvoiceFilter{Search: "Searchable"})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(invoices) != 1 {
		t.Errorf("len = %d, want 1", len(invoices))
	}
}

func TestInvoiceRepository_UpdateStatus(t *testing.T) {
	db, customerID, seqID := setupInvoiceTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	inv := makeRepoInvoice(customerID, seqID)
	repo.Create(ctx, inv)

	if err := repo.UpdateStatus(ctx, inv.ID, domain.InvoiceStatusSent); err != nil {
		t.Fatalf("UpdateStatus() error: %v", err)
	}

	got, err := repo.GetByID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Status != domain.InvoiceStatusSent {
		t.Errorf("Status = %q, want %q", got.Status, domain.InvoiceStatusSent)
	}
}

func TestInvoiceRepository_GetNextNumber(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	seqID := testutil.SeedInvoiceSequence(t, db, "FV", 2026)

	num1, err := repo.GetNextNumber(ctx, seqID)
	if err != nil {
		t.Fatalf("GetNextNumber() error: %v", err)
	}
	if !strings.HasPrefix(num1, "FV2026") {
		t.Errorf("number = %q, expected prefix FV2026", num1)
	}
	if num1 != "FV20260001" {
		t.Errorf("number = %q, want FV20260001", num1)
	}

	num2, err := repo.GetNextNumber(ctx, seqID)
	if err != nil {
		t.Fatalf("GetNextNumber() second call error: %v", err)
	}
	if num2 != "FV20260002" {
		t.Errorf("number = %q, want FV20260002", num2)
	}
}

func TestInvoiceRepository_GetNextNumber_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	_, err := repo.GetNextNumber(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent sequence")
	}
}

func TestInvoiceRepository_GetRelatedInvoices(t *testing.T) {
	db, customerID, seqID := setupInvoiceTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	// Create the parent invoice.
	parent := makeRepoInvoice(customerID, seqID)
	if err := repo.Create(ctx, parent); err != nil {
		t.Fatalf("Create() parent error: %v", err)
	}

	// Create a credit note referencing the parent.
	creditNote := makeRepoInvoice(customerID, seqID)
	creditNote.Type = domain.InvoiceTypeCreditNote
	creditNote.RelatedInvoiceID = &parent.ID
	creditNote.RelationType = domain.RelationTypeCreditNote
	if err := repo.Create(ctx, creditNote); err != nil {
		t.Fatalf("Create() credit note error: %v", err)
	}

	// Create a settlement invoice referencing the parent.
	settlement := makeRepoInvoice(customerID, seqID)
	settlement.RelatedInvoiceID = &parent.ID
	settlement.RelationType = domain.RelationTypeSettlement
	if err := repo.Create(ctx, settlement); err != nil {
		t.Fatalf("Create() settlement error: %v", err)
	}

	// Create an unrelated invoice (should NOT appear).
	unrelated := makeRepoInvoice(customerID, seqID)
	if err := repo.Create(ctx, unrelated); err != nil {
		t.Fatalf("Create() unrelated error: %v", err)
	}

	related, err := repo.GetRelatedInvoices(ctx, parent.ID)
	if err != nil {
		t.Fatalf("GetRelatedInvoices() error: %v", err)
	}
	if len(related) != 2 {
		t.Fatalf("len(related) = %d, want 2", len(related))
	}

	// Verify the related invoices have correct IDs.
	relatedIDs := map[int64]bool{}
	for _, inv := range related {
		relatedIDs[inv.ID] = true
	}
	if !relatedIDs[creditNote.ID] {
		t.Errorf("expected credit note ID %d in related invoices", creditNote.ID)
	}
	if !relatedIDs[settlement.ID] {
		t.Errorf("expected settlement ID %d in related invoices", settlement.ID)
	}
}

func TestInvoiceRepository_GetRelatedInvoices_Empty(t *testing.T) {
	db, customerID, seqID := setupInvoiceTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	inv := makeRepoInvoice(customerID, seqID)
	if err := repo.Create(ctx, inv); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	related, err := repo.GetRelatedInvoices(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetRelatedInvoices() error: %v", err)
	}
	if len(related) != 0 {
		t.Errorf("expected empty related invoices, got %d", len(related))
	}
}

func TestInvoiceRepository_GetRelatedInvoices_ExcludesDeleted(t *testing.T) {
	db, customerID, seqID := setupInvoiceTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	parent := makeRepoInvoice(customerID, seqID)
	if err := repo.Create(ctx, parent); err != nil {
		t.Fatalf("Create() parent error: %v", err)
	}

	child := makeRepoInvoice(customerID, seqID)
	child.RelatedInvoiceID = &parent.ID
	child.RelationType = domain.RelationTypeCreditNote
	if err := repo.Create(ctx, child); err != nil {
		t.Fatalf("Create() child error: %v", err)
	}

	// Soft-delete the child.
	if err := repo.Delete(ctx, child.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	related, err := repo.GetRelatedInvoices(ctx, parent.ID)
	if err != nil {
		t.Fatalf("GetRelatedInvoices() error: %v", err)
	}
	if len(related) != 0 {
		t.Errorf("expected 0 related after delete, got %d", len(related))
	}
}

func TestInvoiceRepository_FindByRelatedInvoice(t *testing.T) {
	db, customerID, seqID := setupInvoiceTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	parent := makeRepoInvoice(customerID, seqID)
	if err := repo.Create(ctx, parent); err != nil {
		t.Fatalf("Create() parent error: %v", err)
	}

	creditNote := makeRepoInvoice(customerID, seqID)
	creditNote.Type = domain.InvoiceTypeCreditNote
	creditNote.RelatedInvoiceID = &parent.ID
	creditNote.RelationType = domain.RelationTypeCreditNote
	if err := repo.Create(ctx, creditNote); err != nil {
		t.Fatalf("Create() credit note error: %v", err)
	}

	// Find credit note by relation.
	found, err := repo.FindByRelatedInvoice(ctx, parent.ID, domain.RelationTypeCreditNote)
	if err != nil {
		t.Fatalf("FindByRelatedInvoice() error: %v", err)
	}
	if found == nil {
		t.Fatal("expected non-nil result")
	}
	if found.ID != creditNote.ID {
		t.Errorf("found.ID = %d, want %d", found.ID, creditNote.ID)
	}
	if found.RelationType != domain.RelationTypeCreditNote {
		t.Errorf("RelationType = %q, want %q", found.RelationType, domain.RelationTypeCreditNote)
	}

	// Search for a relation type that does not exist.
	notFound, err := repo.FindByRelatedInvoice(ctx, parent.ID, domain.RelationTypeSettlement)
	if err != nil {
		t.Fatalf("FindByRelatedInvoice() error: %v", err)
	}
	if notFound != nil {
		t.Errorf("expected nil for non-existent relation, got ID=%d", notFound.ID)
	}
}

func TestInvoiceRepository_FindByRelatedInvoice_ExcludesDeleted(t *testing.T) {
	db, customerID, seqID := setupInvoiceTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	parent := makeRepoInvoice(customerID, seqID)
	if err := repo.Create(ctx, parent); err != nil {
		t.Fatalf("Create() parent error: %v", err)
	}

	child := makeRepoInvoice(customerID, seqID)
	child.RelatedInvoiceID = &parent.ID
	child.RelationType = domain.RelationTypeSettlement
	if err := repo.Create(ctx, child); err != nil {
		t.Fatalf("Create() child error: %v", err)
	}

	// Soft-delete the child.
	if err := repo.Delete(ctx, child.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	found, err := repo.FindByRelatedInvoice(ctx, parent.ID, domain.RelationTypeSettlement)
	if err != nil {
		t.Fatalf("FindByRelatedInvoice() error: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil for deleted invoice, got ID=%d", found.ID)
	}
}

func TestInvoiceRepository_List_DateFilter(t *testing.T) {
	db, customerID, seqID := setupInvoiceTestDB(t)
	repo := NewInvoiceRepository(db)
	ctx := context.Background()

	now := time.Now()

	// Create an old invoice directly.
	past := now.AddDate(0, -3, 0)
	db.ExecContext(ctx, `
		INSERT INTO invoices (sequence_id, invoice_number, type, status, issue_date, due_date, delivery_date,
			customer_id, currency_code, exchange_rate, payment_method,
			subtotal_amount, vat_amount, total_amount, paid_amount, created_at, updated_at)
		VALUES (?, ?, 'regular', 'draft', ?, ?, ?, ?, 'CZK', 100, 'bank_transfer', 0, 0, 0, 0, ?, ?)`,
		seqID, "FV-PAST", past, past.AddDate(0, 0, 14), past, customerID, now, now)

	// Create a recent invoice.
	inv := makeRepoInvoice(customerID, seqID)
	repo.Create(ctx, inv)

	// Filter from 1 month ago (should exclude 3-month-old one).
	dateFrom := now.AddDate(0, -1, 0)
	invoices, total, err := repo.List(ctx, domain.InvoiceFilter{DateFrom: &dateFrom})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	_ = invoices
}
