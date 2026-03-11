package repository

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestInvoiceRepository_ListOverdueCandidateIDs(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Overdue Test Customer"})
	repo := NewInvoiceRepository(db)

	items := []domain.InvoiceItem{
		{Description: "Work", Quantity: 100, Unit: "hod", UnitPrice: 100000, VATRatePercent: 21},
	}

	// Create a 'sent' invoice with past due date.
	inv1 := testutil.SeedInvoice(t, db, customer.ID, items)
	_, err := db.ExecContext(ctx, `UPDATE invoices SET status = 'sent', due_date = '2026-01-01' WHERE id = ?`, inv1.ID)
	if err != nil {
		t.Fatalf("updating invoice: %v", err)
	}

	// Create a 'sent' invoice with future due date.
	inv2 := testutil.SeedInvoice(t, db, customer.ID, items)
	_, err = db.ExecContext(ctx, `UPDATE invoices SET status = 'sent', due_date = '2026-12-31' WHERE id = ?`, inv2.ID)
	if err != nil {
		t.Fatalf("updating invoice: %v", err)
	}

	// Create a 'draft' invoice with past due date (should NOT be a candidate).
	inv3 := testutil.SeedInvoice(t, db, customer.ID, items)
	_, err = db.ExecContext(ctx, `UPDATE invoices SET status = 'draft', due_date = '2026-01-01' WHERE id = ?`, inv3.ID)
	if err != nil {
		t.Fatalf("updating invoice: %v", err)
	}

	beforeDate := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	ids, err := repo.ListOverdueCandidateIDs(ctx, beforeDate)
	if err != nil {
		t.Fatalf("listing overdue candidates: %v", err)
	}

	if len(ids) != 1 {
		t.Fatalf("expected 1 overdue candidate, got %d", len(ids))
	}
	if ids[0] != inv1.ID {
		t.Errorf("expected candidate ID %d, got %d", inv1.ID, ids[0])
	}
}
