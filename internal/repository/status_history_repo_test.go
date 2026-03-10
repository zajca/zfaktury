package repository

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestStatusHistoryRepository_CreateAndList(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "History Test Customer"})
	inv := testutil.SeedInvoice(t, db, customer.ID, []domain.InvoiceItem{
		{Description: "Work", Quantity: 100, Unit: "hod", UnitPrice: 100000, VATRatePercent: 21},
	})

	repo := NewStatusHistoryRepository(db)

	now := time.Now().Truncate(time.Second)

	// Create two status changes.
	change1 := &domain.InvoiceStatusChange{
		InvoiceID: inv.ID,
		OldStatus: domain.InvoiceStatusDraft,
		NewStatus: domain.InvoiceStatusSent,
		ChangedAt: now,
		Note:      "sent to customer",
	}
	if err := repo.Create(ctx, change1); err != nil {
		t.Fatalf("creating status change 1: %v", err)
	}
	if change1.ID == 0 {
		t.Fatal("expected non-zero ID after create")
	}

	change2 := &domain.InvoiceStatusChange{
		InvoiceID: inv.ID,
		OldStatus: domain.InvoiceStatusSent,
		NewStatus: domain.InvoiceStatusOverdue,
		ChangedAt: now.Add(time.Hour),
		Note:      "automatically marked as overdue",
	}
	if err := repo.Create(ctx, change2); err != nil {
		t.Fatalf("creating status change 2: %v", err)
	}

	// List changes.
	changes, err := repo.ListByInvoiceID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("listing status history: %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}

	// Verify ordering (ASC by changed_at).
	if changes[0].NewStatus != domain.InvoiceStatusSent {
		t.Errorf("first change should be 'sent', got %q", changes[0].NewStatus)
	}
	if changes[1].NewStatus != domain.InvoiceStatusOverdue {
		t.Errorf("second change should be 'overdue', got %q", changes[1].NewStatus)
	}
	if changes[0].Note != "sent to customer" {
		t.Errorf("unexpected note: %q", changes[0].Note)
	}
}

func TestStatusHistoryRepository_ListByInvoiceID_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	repo := NewStatusHistoryRepository(db)

	// Non-existent invoice should return empty slice, not error.
	changes, err := repo.ListByInvoiceID(ctx, 9999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 0 {
		t.Fatalf("expected 0 changes, got %d", len(changes))
	}
}
