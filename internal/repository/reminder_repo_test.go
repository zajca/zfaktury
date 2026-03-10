package repository

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestReminderRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReminderRepository(db)
	ctx := context.Background()

	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Reminder Customer", Email: "test@example.com"})
	inv := testutil.SeedInvoice(t, db, customer.ID, []domain.InvoiceItem{
		{Description: "Service", Quantity: 100, Unit: "hod", UnitPrice: 100000, VATRatePercent: 21},
	})

	rem := &domain.PaymentReminder{
		InvoiceID:      inv.ID,
		ReminderNumber: 1,
		SentAt:         time.Now(),
		SentTo:         "test@example.com",
		Subject:        "Reminder subject",
		BodyPreview:    "Preview text",
	}

	if err := repo.Create(ctx, rem); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if rem.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	if rem.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestReminderRepository_ListByInvoiceID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReminderRepository(db)
	ctx := context.Background()

	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "List Customer", Email: "list@example.com"})
	inv := testutil.SeedInvoice(t, db, customer.ID, []domain.InvoiceItem{
		{Description: "Service", Quantity: 100, Unit: "hod", UnitPrice: 100000, VATRatePercent: 21},
	})

	// Create two reminders.
	now := time.Now()
	for i := 1; i <= 2; i++ {
		rem := &domain.PaymentReminder{
			InvoiceID:      inv.ID,
			ReminderNumber: i,
			SentAt:         now.Add(time.Duration(i) * time.Hour),
			SentTo:         "list@example.com",
			Subject:        "Subject",
			BodyPreview:    "Preview",
		}
		if err := repo.Create(ctx, rem); err != nil {
			t.Fatalf("Create() reminder %d error: %v", i, err)
		}
	}

	reminders, err := repo.ListByInvoiceID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("ListByInvoiceID() error: %v", err)
	}

	if len(reminders) != 2 {
		t.Fatalf("ListByInvoiceID() returned %d reminders, want 2", len(reminders))
	}

	if reminders[0].ReminderNumber != 1 {
		t.Errorf("first reminder number = %d, want 1", reminders[0].ReminderNumber)
	}
	if reminders[1].ReminderNumber != 2 {
		t.Errorf("second reminder number = %d, want 2", reminders[1].ReminderNumber)
	}
}

func TestReminderRepository_CountByInvoiceID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReminderRepository(db)
	ctx := context.Background()

	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Count Customer", Email: "count@example.com"})
	inv := testutil.SeedInvoice(t, db, customer.ID, []domain.InvoiceItem{
		{Description: "Service", Quantity: 100, Unit: "hod", UnitPrice: 100000, VATRatePercent: 21},
	})

	// Initially zero.
	count, err := repo.CountByInvoiceID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("CountByInvoiceID() error: %v", err)
	}
	if count != 0 {
		t.Errorf("CountByInvoiceID() = %d, want 0", count)
	}

	// Add a reminder.
	rem := &domain.PaymentReminder{
		InvoiceID:      inv.ID,
		ReminderNumber: 1,
		SentAt:         time.Now(),
		SentTo:         "count@example.com",
		Subject:        "Subject",
		BodyPreview:    "Preview",
	}
	if err := repo.Create(ctx, rem); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	count, err = repo.CountByInvoiceID(ctx, inv.ID)
	if err != nil {
		t.Fatalf("CountByInvoiceID() error: %v", err)
	}
	if count != 1 {
		t.Errorf("CountByInvoiceID() = %d, want 1", count)
	}
}
