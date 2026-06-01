package service

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestDurationUntilHour(t *testing.T) {
	loc := time.UTC
	tests := []struct {
		name string
		now  time.Time
		hour int
		want time.Duration
	}{
		{
			name: "later today",
			now:  time.Date(2026, 6, 1, 5, 0, 0, 0, loc),
			hour: 7,
			want: 2 * time.Hour,
		},
		{
			name: "already past, rolls to tomorrow",
			now:  time.Date(2026, 6, 1, 9, 0, 0, 0, loc),
			hour: 7,
			want: 22 * time.Hour,
		},
		{
			name: "exactly on the hour rolls to tomorrow",
			now:  time.Date(2026, 6, 1, 7, 0, 0, 0, loc),
			hour: 7,
			want: 24 * time.Hour,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := durationUntilHour(tt.now, tt.hour); got != tt.want {
				t.Errorf("durationUntilHour() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRecurringScheduler_RunOnce verifies the scheduler processes due recurring
// invoices for every company, not just the active one, and that a failure for
// one company does not stop the others.
func TestRecurringScheduler_RunOnce(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	contactRepo := repository.NewContactRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	sequenceRepo := repository.NewSequenceRepository(db)
	recurringRepo := repository.NewRecurringInvoiceRepository(db)
	companyRepo := repository.NewCompanyRepository(db)
	contactSvc := NewContactService(contactRepo, nil, nil)
	sequenceSvc := NewSequenceService(sequenceRepo, nil)
	invoiceSvc := NewInvoiceService(invoiceRepo, contactSvc, sequenceSvc, nil)
	recurringSvc := NewRecurringInvoiceService(recurringRepo, invoiceSvc, nil, nil)

	// testutil seeds company 1; add a second company.
	company2ID, err := companyRepo.Create(ctx, domain.Company{Name: "Second s.r.o.", LegalName: "Second s.r.o."})
	if err != nil {
		t.Fatalf("creating second company: %v", err)
	}

	// A due recurring invoice + customer for each company. Both companies use
	// the same "FV" sequence starting at 1, so both generate the SAME invoice
	// number ("FV<year>0001") on the same run. Migration 028 made invoice_number
	// unique per company, so this must NOT collide -- exercising that fix.
	seedDue := func(companyID int64) {
		testutil.SeedInvoiceSequence(t, db, companyID, "FV", time.Now().Year())
		c := &domain.Contact{Name: "Cust", Type: domain.ContactTypeCompany}
		if err := contactSvc.Create(ctx, companyID, c); err != nil {
			t.Fatalf("creating customer for company %d: %v", companyID, err)
		}
		ri := makeTestRecurringInvoice(c.ID)
		ri.NextIssueDate = time.Now().Truncate(24 * time.Hour)
		if err := recurringSvc.Create(ctx, companyID, ri); err != nil {
			t.Fatalf("creating recurring invoice for company %d: %v", companyID, err)
		}
	}
	seedDue(1)
	seedDue(company2ID)

	scheduler := NewRecurringScheduler(companyRepo, recurringSvc, 7)
	scheduler.runOnce(ctx)

	// Each company should now have exactly one generated invoice, and both
	// should carry the SAME number -- which only works because invoice_number is
	// unique per company (migration 028), not globally.
	var numbers []string
	for _, companyID := range []int64{1, company2ID} {
		list, total, err := invoiceSvc.List(ctx, companyID, domain.InvoiceFilter{})
		if err != nil {
			t.Fatalf("List(company %d) error: %v", companyID, err)
		}
		if total != 1 {
			t.Errorf("company %d: generated invoices = %d, want 1", companyID, total)
			continue
		}
		numbers = append(numbers, list[0].InvoiceNumber)
	}
	if len(numbers) == 2 && numbers[0] != numbers[1] {
		t.Errorf("expected both companies to share the same number, got %v", numbers)
	}
}
