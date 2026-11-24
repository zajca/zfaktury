package service

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func supplyItems() []domain.InvoiceItem {
	return []domain.InvoiceItem{
		{Description: "Consulting", Quantity: 100, Unit: "hod", UnitPrice: 100000, VATRatePercent: 21},
	}
}

func TestVATReturnService_Recalculate_SelectsInvoicesByDeliveryDate(t *testing.T) {
	svc, db := setupVATReturnSvc(t)
	vatRepo := repository.NewVATReturnRepository(db)
	ctx := context.Background()

	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Customer a.s.", DIC: "CZ11223344"})
	// DUZP in October, issued in November.
	inv := seedInvoiceWithDates(t, db, customer.ID,
		time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC),
		domain.InvoiceStatusSent, domain.InvoiceTypeRegular, supplyItems())

	linkedIn := func(month int) []int64 {
		vr := &domain.VATReturn{Period: domain.TaxPeriod{Year: 2026, Month: month}, FilingType: domain.FilingTypeRegular}
		if err := svc.Create(ctx, vr); err != nil {
			t.Fatalf("Create(month=%d) error: %v", month, err)
		}
		if _, err := svc.Recalculate(ctx, vr.ID); err != nil {
			t.Fatalf("Recalculate(month=%d) error: %v", month, err)
		}
		ids, err := vatRepo.GetLinkedInvoiceIDs(ctx, vr.ID)
		if err != nil {
			t.Fatalf("GetLinkedInvoiceIDs(month=%d) error: %v", month, err)
		}
		return ids
	}

	if ids := linkedIn(10); !slices.Contains(ids, inv.ID) {
		t.Errorf("October return linked %v, want it to contain invoice %d", ids, inv.ID)
	}
	if ids := linkedIn(11); slices.Contains(ids, inv.ID) {
		t.Errorf("November return linked %v, must not contain invoice %d", ids, inv.ID)
	}
}

func TestVATReturnService_Recalculate_InvoiceWithoutDeliveryDateUsesIssueDate(t *testing.T) {
	svc, db := setupVATReturnSvc(t)
	vatRepo := repository.NewVATReturnRepository(db)
	ctx := context.Background()

	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Customer a.s.", DIC: "CZ11223344"})
	inv := seedInvoiceWithDates(t, db, customer.ID,
		time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		domain.InvoiceStatusSent, domain.InvoiceTypeRegular, supplyItems())
	if _, err := db.ExecContext(ctx, `UPDATE invoices SET delivery_date = NULL WHERE id = ?`, inv.ID); err != nil {
		t.Fatalf("clearing delivery_date: %v", err)
	}

	vr := &domain.VATReturn{Period: domain.TaxPeriod{Year: 2026, Month: 6}, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if _, err := svc.Recalculate(ctx, vr.ID); err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}
	ids, err := vatRepo.GetLinkedInvoiceIDs(ctx, vr.ID)
	if err != nil {
		t.Fatalf("GetLinkedInvoiceIDs() error: %v", err)
	}
	if !slices.Contains(ids, inv.ID) {
		t.Errorf("June return linked %v, want it to contain invoice %d", ids, inv.ID)
	}
}

func TestVATControlStatementService_Recalculate_SelectsInvoicesByDeliveryDate(t *testing.T) {
	svc, db := setupVATControlSvc(t)
	ctx := context.Background()

	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Customer a.s.", DIC: "CZ11223344"})
	seedInvoiceWithDates(t, db, customer.ID,
		time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC),
		domain.InvoiceStatusSent, domain.InvoiceTypeRegular, supplyItems())

	linesIn := func(month int) int {
		cs := &domain.VATControlStatement{Period: domain.TaxPeriod{Year: 2026, Month: month}, FilingType: domain.FilingTypeRegular}
		if err := svc.Create(ctx, cs); err != nil {
			t.Fatalf("Create(month=%d) error: %v", month, err)
		}
		if err := svc.Recalculate(ctx, cs.ID); err != nil {
			t.Fatalf("Recalculate(month=%d) error: %v", month, err)
		}
		lines, err := svc.GetLines(ctx, cs.ID)
		if err != nil {
			t.Fatalf("GetLines(month=%d) error: %v", month, err)
		}
		return len(lines)
	}

	if n := linesIn(10); n == 0 {
		t.Error("October control statement has no lines, want the invoice with DUZP 2026-10-31")
	}
	if n := linesIn(11); n != 0 {
		t.Errorf("November control statement has %d lines, want 0", n)
	}
}

func TestVIES_Recalculate_SelectsInvoicesByDeliveryDate(t *testing.T) {
	svc, db := newVIESSvc(t)
	ctx := context.Background()

	partner := testutil.SeedContact(t, db, &domain.Contact{Name: "German GmbH", DIC: "DE123456789", Country: "DE"})
	// DUZP in Q4 2026, issued in Q1 2027.
	seedInvoiceWithDates(t, db, partner.ID,
		time.Date(2027, 1, 3, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		domain.InvoiceStatusSent, domain.InvoiceTypeRegular, supplyItems())

	linesIn := func(year, quarter int) int {
		vs := &domain.VIESSummary{Period: domain.TaxPeriod{Year: year, Quarter: quarter}, FilingType: domain.FilingTypeRegular}
		if err := svc.Create(ctx, vs); err != nil {
			t.Fatalf("Create(%d Q%d) error: %v", year, quarter, err)
		}
		if err := svc.Recalculate(ctx, vs.ID); err != nil {
			t.Fatalf("Recalculate(%d Q%d) error: %v", year, quarter, err)
		}
		lines, err := svc.GetLines(ctx, vs.ID)
		if err != nil {
			t.Fatalf("GetLines(%d Q%d) error: %v", year, quarter, err)
		}
		return len(lines)
	}

	if n := linesIn(2026, 4); n != 1 {
		t.Errorf("Q4 2026 summary has %d lines, want 1", n)
	}
	if n := linesIn(2027, 1); n != 0 {
		t.Errorf("Q1 2027 summary has %d lines, want 0", n)
	}
}

func TestCalculateAnnualBase_SelectsInvoicesByDeliveryDate(t *testing.T) {
	db := testutil.NewTestDB(t)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	ctx := context.Background()

	customer := testutil.SeedContact(t, db, &domain.Contact{Name: "Customer a.s."})
	// DUZP in 2025, issued in 2026.
	inv := seedInvoiceWithDates(t, db, customer.ID,
		time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		domain.InvoiceStatusSent, domain.InvoiceTypeRegular, supplyItems())

	base2025, err := CalculateAnnualBase(ctx, invRepo, expRepo, 2025)
	if err != nil {
		t.Fatalf("CalculateAnnualBase(2025) error: %v", err)
	}
	if !slices.Contains(base2025.InvoiceIDs, inv.ID) {
		t.Errorf("2025 base invoice IDs = %v, want it to contain %d", base2025.InvoiceIDs, inv.ID)
	}

	base2026, err := CalculateAnnualBase(ctx, invRepo, expRepo, 2026)
	if err != nil {
		t.Fatalf("CalculateAnnualBase(2026) error: %v", err)
	}
	if slices.Contains(base2026.InvoiceIDs, inv.ID) {
		t.Errorf("2026 base invoice IDs = %v, must not contain %d", base2026.InvoiceIDs, inv.ID)
	}
}
