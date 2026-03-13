package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func newVIESSvc(t *testing.T) (*VIESSummaryService, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	viesRepo := repository.NewVIESSummaryRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	contactRepo := repository.NewContactRepository(db)
	svc := NewVIESSummaryService(viesRepo, invoiceRepo, contactRepo, nil)
	return svc, db
}

func validVIESSummary() *domain.VIESSummary {
	return &domain.VIESSummary{
		Period: domain.TaxPeriod{
			Year:    2025,
			Quarter: 1,
		},
		FilingType: domain.FilingTypeRegular,
	}
}

func TestVIES_Create_Valid(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	vs := validVIESSummary()
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if vs.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	if vs.Status != domain.FilingStatusDraft {
		t.Errorf("Status = %q, want %q", vs.Status, domain.FilingStatusDraft)
	}
}

func TestVIES_Create_InvalidYear(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	vs := validVIESSummary()
	vs.Period.Year = 1999
	err := svc.Create(ctx, vs)
	if err == nil {
		t.Fatal("expected error for year 1999, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestVIES_Create_InvalidQuarter(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	vs := validVIESSummary()
	vs.Period.Quarter = 5
	err := svc.Create(ctx, vs)
	if err == nil {
		t.Fatal("expected error for quarter 5, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestVIES_Create_InvalidQuarterZero(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	vs := validVIESSummary()
	vs.Period.Quarter = 0
	err := svc.Create(ctx, vs)
	if err == nil {
		t.Fatal("expected error for quarter 0, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestVIES_Create_InvalidFilingType(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	vs := validVIESSummary()
	vs.FilingType = "bogus"
	err := svc.Create(ctx, vs)
	if err == nil {
		t.Fatal("expected error for invalid filing_type, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestVIES_Create_DefaultFilingType(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	vs := validVIESSummary()
	vs.FilingType = "" // should default to regular
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if vs.FilingType != domain.FilingTypeRegular {
		t.Errorf("FilingType = %q, want %q", vs.FilingType, domain.FilingTypeRegular)
	}
}

func TestVIES_Create_DuplicatePeriod(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	vs1 := validVIESSummary()
	if err := svc.Create(ctx, vs1); err != nil {
		t.Fatalf("first Create() error: %v", err)
	}

	vs2 := validVIESSummary()
	err := svc.Create(ctx, vs2)
	if err == nil {
		t.Fatal("expected error for duplicate period, got nil")
	}
	if !errors.Is(err, domain.ErrDuplicateNumber) {
		t.Errorf("expected ErrDuplicateNumber, got: %v", err)
	}
}

func TestVIES_GetByID(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	vs := validVIESSummary()
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := svc.GetByID(ctx, vs.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.ID != vs.ID {
		t.Errorf("ID = %d, want %d", got.ID, vs.ID)
	}
	if got.Period.Year != 2025 || got.Period.Quarter != 1 {
		t.Errorf("Period = %+v, want Year=2025 Quarter=1", got.Period)
	}
	if got.Status != domain.FilingStatusDraft {
		t.Errorf("Status = %q, want %q", got.Status, domain.FilingStatusDraft)
	}
}

func TestVIES_GetByID_NotFound(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 99999)
	if err == nil {
		t.Fatal("expected error for non-existent ID, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestVIES_List(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	// Create two summaries for 2025 and one for 2024.
	vs1 := &domain.VIESSummary{
		Period:     domain.TaxPeriod{Year: 2025, Quarter: 1},
		FilingType: domain.FilingTypeRegular,
	}
	vs2 := &domain.VIESSummary{
		Period:     domain.TaxPeriod{Year: 2025, Quarter: 2},
		FilingType: domain.FilingTypeRegular,
	}
	vs3 := &domain.VIESSummary{
		Period:     domain.TaxPeriod{Year: 2024, Quarter: 4},
		FilingType: domain.FilingTypeRegular,
	}
	for _, vs := range []*domain.VIESSummary{vs1, vs2, vs3} {
		if err := svc.Create(ctx, vs); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	list, err := svc.List(ctx, 2025)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("List(2025) returned %d items, want 2", len(list))
	}

	list2024, err := svc.List(ctx, 2024)
	if err != nil {
		t.Fatalf("List(2024) error: %v", err)
	}
	if len(list2024) != 1 {
		t.Errorf("List(2024) returned %d items, want 1", len(list2024))
	}
}

func TestVIES_Delete_Draft(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	vs := validVIESSummary()
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.Delete(ctx, vs.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// Verify it's gone.
	_, err := svc.GetByID(ctx, vs.ID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got: %v", err)
	}
}

func TestVIES_Delete_Filed(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	vs := validVIESSummary()
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := svc.MarkFiled(ctx, vs.ID); err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}

	err := svc.Delete(ctx, vs.ID)
	if err == nil {
		t.Fatal("expected error when deleting filed summary, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestVIES_Delete_NotFound(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	err := svc.Delete(ctx, 99999)
	if err == nil {
		t.Fatal("expected error when deleting non-existent summary, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestVIES_MarkFiled(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	vs := validVIESSummary()
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.MarkFiled(ctx, vs.ID); err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}

	got, err := svc.GetByID(ctx, vs.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Status != domain.FilingStatusFiled {
		t.Errorf("Status = %q, want %q", got.Status, domain.FilingStatusFiled)
	}
	if got.FiledAt == nil {
		t.Error("FiledAt should be set after MarkFiled")
	}
}

func TestVIES_MarkFiled_AlreadyFiled(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	vs := validVIESSummary()
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := svc.MarkFiled(ctx, vs.ID); err != nil {
		t.Fatalf("first MarkFiled() error: %v", err)
	}

	err := svc.MarkFiled(ctx, vs.ID)
	if err == nil {
		t.Fatal("expected error when marking already filed summary, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestVIES_GetLines_Empty(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	vs := validVIESSummary()
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	lines, err := svc.GetLines(ctx, vs.ID)
	if err != nil {
		t.Fatalf("GetLines() error: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("GetLines() returned %d lines, want 0", len(lines))
	}
}

// --- quarterDateRange tests ---

func TestQuarterDateRange_Q1(t *testing.T) {
	start, end := quarterDateRange(2025, 1)
	wantStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	wantEndApprox := time.Date(2025, 3, 31, 23, 59, 59, 0, time.UTC)

	if !start.Equal(wantStart) {
		t.Errorf("Q1 start = %v, want %v", start, wantStart)
	}
	if end.Before(wantEndApprox) {
		t.Errorf("Q1 end = %v, should be >= %v", end, wantEndApprox)
	}
	// End should be the last nanosecond of March 31.
	if end.Month() != time.March || end.Day() != 31 {
		t.Errorf("Q1 end month/day = %v/%d, want March/31", end.Month(), end.Day())
	}
}

func TestQuarterDateRange_Q2(t *testing.T) {
	start, end := quarterDateRange(2025, 2)
	wantStart := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)

	if !start.Equal(wantStart) {
		t.Errorf("Q2 start = %v, want %v", start, wantStart)
	}
	if end.Month() != time.June || end.Day() != 30 {
		t.Errorf("Q2 end month/day = %v/%d, want June/30", end.Month(), end.Day())
	}
}

func TestQuarterDateRange_Q3(t *testing.T) {
	start, end := quarterDateRange(2025, 3)
	wantStart := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)

	if !start.Equal(wantStart) {
		t.Errorf("Q3 start = %v, want %v", start, wantStart)
	}
	if end.Month() != time.September || end.Day() != 30 {
		t.Errorf("Q3 end month/day = %v/%d, want September/30", end.Month(), end.Day())
	}
}

func TestQuarterDateRange_Q4(t *testing.T) {
	start, end := quarterDateRange(2025, 4)
	wantStart := time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC)

	if !start.Equal(wantStart) {
		t.Errorf("Q4 start = %v, want %v", start, wantStart)
	}
	if end.Month() != time.December || end.Day() != 31 {
		t.Errorf("Q4 end month/day = %v/%d, want December/31", end.Month(), end.Day())
	}
}

// --- Recalculate tests ---

// seedInvoiceWithDates creates an invoice with specific dates and status via direct SQL.
func seedInvoiceWithDates(t *testing.T, db *sql.DB, customerID int64, issueDate, deliveryDate time.Time, status string, invType string, items []domain.InvoiceItem) *domain.Invoice {
	t.Helper()

	inv := testutil.SeedInvoice(t, db, customerID, items)

	// Update dates, status, and type via raw SQL to bypass service layer.
	_, err := db.Exec(`
		UPDATE invoices SET
			issue_date = ?, delivery_date = ?, status = ?, type = ?
		WHERE id = ?`,
		issueDate.Format("2006-01-02"),
		deliveryDate.Format("2006-01-02"),
		status,
		invType,
		inv.ID,
	)
	if err != nil {
		t.Fatalf("updating invoice dates/status: %v", err)
	}

	inv.IssueDate = issueDate
	inv.DeliveryDate = deliveryDate
	inv.Status = status
	inv.Type = invType
	return inv
}

func TestVIES_Recalculate_BasicEUPartner(t *testing.T) {
	svc, db := newVIESSvc(t)
	ctx := context.Background()

	// Create an EU partner contact (German company).
	euContact := testutil.SeedContact(t, db, &domain.Contact{
		Name:    "German GmbH",
		DIC:     "DE123456789",
		Country: "DE",
	})

	// Create a VIES summary for Q1 2025.
	vs := validVIESSummary()
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Create a sent invoice in Q1 2025 for the EU partner.
	items := []domain.InvoiceItem{
		{Description: "Consulting", Quantity: 100, Unit: "hod", UnitPrice: 200000, VATRatePercent: 0},
	}
	seedInvoiceWithDates(t, db, euContact.ID,
		time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC), // issue date
		time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC), // delivery date
		domain.InvoiceStatusSent,
		domain.InvoiceTypeRegular,
		items,
	)

	if err := svc.Recalculate(ctx, vs.ID); err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	lines, err := svc.GetLines(ctx, vs.ID)
	if err != nil {
		t.Fatalf("GetLines() error: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].PartnerDIC != "DE123456789" {
		t.Errorf("PartnerDIC = %q, want %q", lines[0].PartnerDIC, "DE123456789")
	}
	if lines[0].CountryCode != "DE" {
		t.Errorf("CountryCode = %q, want %q", lines[0].CountryCode, "DE")
	}
	if lines[0].ServiceCode != "3" {
		t.Errorf("ServiceCode = %q, want %q", lines[0].ServiceCode, "3")
	}
	// Subtotal: quantity(100) * unit_price(200000) / 100 = 200000
	if lines[0].TotalAmount != 200000 {
		t.Errorf("TotalAmount = %d, want 200000", lines[0].TotalAmount)
	}
}

func TestVIES_Recalculate_MultipleEUPartners(t *testing.T) {
	svc, db := newVIESSvc(t)
	ctx := context.Background()

	deContact := testutil.SeedContact(t, db, &domain.Contact{
		Name: "German GmbH", DIC: "DE111111111", Country: "DE",
	})
	skContact := testutil.SeedContact(t, db, &domain.Contact{
		Name: "Slovak s.r.o.", DIC: "SK222222222", Country: "SK",
	})

	vs := validVIESSummary()
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	items := []domain.InvoiceItem{
		{Description: "Service", Quantity: 100, Unit: "ks", UnitPrice: 100000, VATRatePercent: 0},
	}
	// Two invoices for DE partner.
	seedInvoiceWithDates(t, db, deContact.ID,
		time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
		domain.InvoiceStatusSent, domain.InvoiceTypeRegular, items,
	)
	seedInvoiceWithDates(t, db, deContact.ID,
		time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC),
		domain.InvoiceStatusPaid, domain.InvoiceTypeRegular, items,
	)
	// One invoice for SK partner.
	seedInvoiceWithDates(t, db, skContact.ID,
		time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC),
		domain.InvoiceStatusOverdue, domain.InvoiceTypeRegular, items,
	)

	if err := svc.Recalculate(ctx, vs.ID); err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	lines, err := svc.GetLines(ctx, vs.ID)
	if err != nil {
		t.Fatalf("GetLines() error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	// Build a map for easier assertions.
	lineMap := make(map[string]domain.VIESSummaryLine)
	for _, l := range lines {
		lineMap[l.CountryCode] = l
	}

	deLine, ok := lineMap["DE"]
	if !ok {
		t.Fatal("missing DE line")
	}
	// Two invoices: each subtotal = 100*100000/100 = 100000, total = 200000
	if deLine.TotalAmount != 200000 {
		t.Errorf("DE TotalAmount = %d, want 200000", deLine.TotalAmount)
	}

	skLine, ok := lineMap["SK"]
	if !ok {
		t.Fatal("missing SK line")
	}
	if skLine.TotalAmount != 100000 {
		t.Errorf("SK TotalAmount = %d, want 100000", skLine.TotalAmount)
	}
}

func TestVIES_Recalculate_CreditNoteReducesAmount(t *testing.T) {
	svc, db := newVIESSvc(t)
	ctx := context.Background()

	euContact := testutil.SeedContact(t, db, &domain.Contact{
		Name: "German GmbH", DIC: "DE999999999", Country: "DE",
	})

	vs := validVIESSummary()
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Regular invoice: subtotal = 100*500000/100 = 500000
	regularItems := []domain.InvoiceItem{
		{Description: "Main service", Quantity: 100, Unit: "ks", UnitPrice: 500000, VATRatePercent: 0},
	}
	seedInvoiceWithDates(t, db, euContact.ID,
		time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		domain.InvoiceStatusSent, domain.InvoiceTypeRegular, regularItems,
	)

	// Credit note: subtotal = 100*100000/100 = 100000 (reduces by this amount)
	creditItems := []domain.InvoiceItem{
		{Description: "Correction", Quantity: 100, Unit: "ks", UnitPrice: 100000, VATRatePercent: 0},
	}
	seedInvoiceWithDates(t, db, euContact.ID,
		time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		domain.InvoiceStatusSent, domain.InvoiceTypeCreditNote, creditItems,
	)

	if err := svc.Recalculate(ctx, vs.ID); err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	lines, err := svc.GetLines(ctx, vs.ID)
	if err != nil {
		t.Fatalf("GetLines() error: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	// 500000 - 100000 = 400000
	if lines[0].TotalAmount != 400000 {
		t.Errorf("TotalAmount = %d, want 400000", lines[0].TotalAmount)
	}
}

func TestVIES_Recalculate_IgnoresDraftAndCancelledInvoices(t *testing.T) {
	svc, db := newVIESSvc(t)
	ctx := context.Background()

	euContact := testutil.SeedContact(t, db, &domain.Contact{
		Name: "German GmbH", DIC: "DE555555555", Country: "DE",
	})

	vs := validVIESSummary()
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	items := []domain.InvoiceItem{
		{Description: "Service", Quantity: 100, Unit: "ks", UnitPrice: 100000, VATRatePercent: 0},
	}

	// Draft invoice -- should be ignored.
	seedInvoiceWithDates(t, db, euContact.ID,
		time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
		domain.InvoiceStatusDraft, domain.InvoiceTypeRegular, items,
	)

	// Cancelled invoice -- should be ignored.
	seedInvoiceWithDates(t, db, euContact.ID,
		time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC),
		domain.InvoiceStatusCancelled, domain.InvoiceTypeRegular, items,
	)

	if err := svc.Recalculate(ctx, vs.ID); err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	lines, err := svc.GetLines(ctx, vs.ID)
	if err != nil {
		t.Fatalf("GetLines() error: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines (draft/cancelled ignored), got %d", len(lines))
	}
}

func TestVIES_Recalculate_IgnoresNonEUContact(t *testing.T) {
	svc, db := newVIESSvc(t)
	ctx := context.Background()

	// Czech contact -- not EU partner.
	czContact := testutil.SeedContact(t, db, &domain.Contact{
		Name: "Czech s.r.o.", DIC: "CZ12345678", Country: "CZ",
	})

	vs := validVIESSummary()
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	items := []domain.InvoiceItem{
		{Description: "Service", Quantity: 100, Unit: "ks", UnitPrice: 100000, VATRatePercent: 21},
	}
	seedInvoiceWithDates(t, db, czContact.ID,
		time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC),
		domain.InvoiceStatusSent, domain.InvoiceTypeRegular, items,
	)

	if err := svc.Recalculate(ctx, vs.ID); err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	lines, err := svc.GetLines(ctx, vs.ID)
	if err != nil {
		t.Fatalf("GetLines() error: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines (CZ not EU partner), got %d", len(lines))
	}
}

func TestVIES_Recalculate_IgnoresDeliveryDateOutsideQuarter(t *testing.T) {
	svc, db := newVIESSvc(t)
	ctx := context.Background()

	euContact := testutil.SeedContact(t, db, &domain.Contact{
		Name: "German GmbH", DIC: "DE777777777", Country: "DE",
	})

	vs := validVIESSummary() // Q1 2025
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	items := []domain.InvoiceItem{
		{Description: "Service", Quantity: 100, Unit: "ks", UnitPrice: 100000, VATRatePercent: 0},
	}

	// Issue date in Q1 but delivery date in Q2.
	seedInvoiceWithDates(t, db, euContact.ID,
		time.Date(2025, 3, 28, 0, 0, 0, 0, time.UTC), // issue in Q1
		time.Date(2025, 4, 2, 0, 0, 0, 0, time.UTC),  // delivery in Q2
		domain.InvoiceStatusSent, domain.InvoiceTypeRegular, items,
	)

	if err := svc.Recalculate(ctx, vs.ID); err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	lines, err := svc.GetLines(ctx, vs.ID)
	if err != nil {
		t.Fatalf("GetLines() error: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines (delivery date outside Q1), got %d", len(lines))
	}
}

func TestVIES_Recalculate_Filed_Error(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	vs := validVIESSummary()
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if err := svc.MarkFiled(ctx, vs.ID); err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}

	err := svc.Recalculate(ctx, vs.ID)
	if err == nil {
		t.Fatal("expected error when recalculating filed summary, got nil")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestVIES_Recalculate_ReplacesOldLines(t *testing.T) {
	svc, db := newVIESSvc(t)
	ctx := context.Background()

	euContact := testutil.SeedContact(t, db, &domain.Contact{
		Name: "German GmbH", DIC: "DE888888888", Country: "DE",
	})

	vs := validVIESSummary()
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	items := []domain.InvoiceItem{
		{Description: "Service", Quantity: 100, Unit: "ks", UnitPrice: 100000, VATRatePercent: 0},
	}
	seedInvoiceWithDates(t, db, euContact.ID,
		time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		domain.InvoiceStatusSent, domain.InvoiceTypeRegular, items,
	)

	// First recalculate.
	if err := svc.Recalculate(ctx, vs.ID); err != nil {
		t.Fatalf("first Recalculate() error: %v", err)
	}
	lines1, _ := svc.GetLines(ctx, vs.ID)
	if len(lines1) != 1 {
		t.Fatalf("expected 1 line after first recalculate, got %d", len(lines1))
	}

	// Second recalculate should replace, not duplicate.
	if err := svc.Recalculate(ctx, vs.ID); err != nil {
		t.Fatalf("second Recalculate() error: %v", err)
	}
	lines2, _ := svc.GetLines(ctx, vs.ID)
	if len(lines2) != 1 {
		t.Errorf("expected 1 line after second recalculate, got %d (lines were duplicated)", len(lines2))
	}
}

func TestVIES_Recalculate_ZeroAmountPartnerSkipped(t *testing.T) {
	svc, db := newVIESSvc(t)
	ctx := context.Background()

	euContact := testutil.SeedContact(t, db, &domain.Contact{
		Name: "German GmbH", DIC: "DE666666666", Country: "DE",
	})

	vs := validVIESSummary()
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	items := []domain.InvoiceItem{
		{Description: "Service", Quantity: 100, Unit: "ks", UnitPrice: 100000, VATRatePercent: 0},
	}

	// Regular invoice + credit note of the same amount = zero net.
	seedInvoiceWithDates(t, db, euContact.ID,
		time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
		domain.InvoiceStatusSent, domain.InvoiceTypeRegular, items,
	)
	seedInvoiceWithDates(t, db, euContact.ID,
		time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC),
		domain.InvoiceStatusSent, domain.InvoiceTypeCreditNote, items,
	)

	if err := svc.Recalculate(ctx, vs.ID); err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	lines, err := svc.GetLines(ctx, vs.ID)
	if err != nil {
		t.Fatalf("GetLines() error: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines (zero net amount), got %d", len(lines))
	}
}

// --- GenerateXML tests ---

func TestVIES_GenerateXML_Valid(t *testing.T) {
	svc, db := newVIESSvc(t)
	ctx := context.Background()

	euContact := testutil.SeedContact(t, db, &domain.Contact{
		Name: "German GmbH", DIC: "DE444444444", Country: "DE",
	})

	vs := validVIESSummary()
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	items := []domain.InvoiceItem{
		{Description: "Consulting", Quantity: 100, Unit: "hod", UnitPrice: 300000, VATRatePercent: 0},
	}
	seedInvoiceWithDates(t, db, euContact.ID,
		time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		domain.InvoiceStatusPaid, domain.InvoiceTypeRegular, items,
	)

	// Recalculate first to create lines.
	if err := svc.Recalculate(ctx, vs.ID); err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	// Generate XML.
	if err := svc.GenerateXML(ctx, vs.ID, "CZ12345678"); err != nil {
		t.Fatalf("GenerateXML() error: %v", err)
	}

	// Verify XML was stored.
	updated, err := svc.GetByID(ctx, vs.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if len(updated.XMLData) == 0 {
		t.Fatal("expected non-empty XMLData after GenerateXML")
	}

	xmlStr := string(updated.XMLData)
	if !strings.Contains(xmlStr, "<?xml") {
		t.Error("XML data should contain XML declaration")
	}
	if !strings.Contains(xmlStr, "12345678") {
		t.Error("XML data should contain filer DIC (without CZ prefix)")
	}
	if !strings.Contains(xmlStr, "444444444") {
		t.Error("XML data should contain partner DIC")
	}
}

func TestVIES_GenerateXML_NoLines(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	vs := validVIESSummary()
	if err := svc.Create(ctx, vs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// GenerateXML with no lines should succeed (empty summary is valid).
	if err := svc.GenerateXML(ctx, vs.ID, "CZ87654321"); err != nil {
		t.Fatalf("GenerateXML() error: %v", err)
	}

	updated, err := svc.GetByID(ctx, vs.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if len(updated.XMLData) == 0 {
		t.Fatal("expected non-empty XMLData even with no lines")
	}
}

func TestVIES_GenerateXML_NotFound(t *testing.T) {
	svc, _ := newVIESSvc(t)
	ctx := context.Background()

	err := svc.GenerateXML(ctx, 99999, "CZ12345678")
	if err == nil {
		t.Fatal("expected error for non-existent ID, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}
