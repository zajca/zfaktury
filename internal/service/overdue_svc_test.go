package service

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestOverdueService_CheckOverdue(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Overdue Svc Customer"})
	items := []domain.InvoiceItem{
		{Description: "Work", Quantity: 100, Unit: "hod", UnitPrice: 100000, VATRatePercent: 21},
	}

	// Create a 'sent' invoice with past due date.
	inv1 := testutil.SeedInvoice(t, db, customer.ID, items)
	_, err := db.ExecContext(ctx, `UPDATE invoices SET status = 'sent', due_date = '2026-01-01' WHERE id = ?`, inv1.ID)
	if err != nil {
		t.Fatalf("updating invoice: %v", err)
	}

	// Create a 'sent' invoice with future due date (should NOT be marked).
	inv2 := testutil.SeedInvoice(t, db, customer.ID, items)
	_, err = db.ExecContext(ctx, `UPDATE invoices SET status = 'sent', due_date = '2099-12-31' WHERE id = ?`, inv2.ID)
	if err != nil {
		t.Fatalf("updating invoice: %v", err)
	}

	invoiceRepo := repository.NewInvoiceRepository(db)
	historyRepo := repository.NewStatusHistoryRepository(db)
	svc := NewOverdueService(invoiceRepo, historyRepo)

	count, err := svc.CheckOverdue(ctx)
	if err != nil {
		t.Fatalf("CheckOverdue: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 overdue, got %d", count)
	}

	// Verify status changed.
	var status string
	db.QueryRowContext(ctx, `SELECT status FROM invoices WHERE id = ?`, inv1.ID).Scan(&status)
	if status != domain.InvoiceStatusOverdue {
		t.Errorf("invoice status = %q, want %q", status, domain.InvoiceStatusOverdue)
	}

	// Verify history was recorded.
	history, err := svc.GetHistory(ctx, inv1.ID)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history))
	}
	if history[0].OldStatus != domain.InvoiceStatusSent {
		t.Errorf("old_status = %q, want %q", history[0].OldStatus, domain.InvoiceStatusSent)
	}
	if history[0].NewStatus != domain.InvoiceStatusOverdue {
		t.Errorf("new_status = %q, want %q", history[0].NewStatus, domain.InvoiceStatusOverdue)
	}
	if history[0].Note != "automatically marked as overdue" {
		t.Errorf("note = %q", history[0].Note)
	}
}

func TestOverdueService_CheckOverdue_NoCandidates(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	invoiceRepo := repository.NewInvoiceRepository(db)
	historyRepo := repository.NewStatusHistoryRepository(db)
	svc := NewOverdueService(invoiceRepo, historyRepo)

	count, err := svc.CheckOverdue(ctx)
	if err != nil {
		t.Fatalf("CheckOverdue: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}
}

func TestOverdueService_GetHistory(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "History Svc Customer"})
	items := []domain.InvoiceItem{
		{Description: "Work", Quantity: 100, Unit: "hod", UnitPrice: 100000, VATRatePercent: 21},
	}
	inv := testutil.SeedInvoice(t, db, customer.ID, items)

	historyRepo := repository.NewStatusHistoryRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	svc := NewOverdueService(invoiceRepo, historyRepo)

	// Manually create a history entry.
	change := &domain.InvoiceStatusChange{
		InvoiceID: inv.ID,
		OldStatus: domain.InvoiceStatusDraft,
		NewStatus: domain.InvoiceStatusSent,
		ChangedAt: time.Now(),
		Note:      "test",
	}
	if err := historyRepo.Create(ctx, change); err != nil {
		t.Fatalf("creating history: %v", err)
	}

	history, err := svc.GetHistory(ctx, inv.ID)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1, got %d", len(history))
	}
	if history[0].NewStatus != domain.InvoiceStatusSent {
		t.Errorf("new_status = %q, want %q", history[0].NewStatus, domain.InvoiceStatusSent)
	}
}
