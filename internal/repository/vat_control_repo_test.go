package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestVATControlStatementRepository_CreateAndGetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	cs := &domain.VATControlStatement{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 3,
		},
		FilingType: domain.FilingTypeRegular,
		XMLData:    []byte("<xml>test</xml>"),
		Status:     domain.FilingStatusDraft,
	}

	if err := repo.Create(ctx, cs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if cs.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	if cs.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if cs.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}

	got, err := repo.GetByID(ctx, cs.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.Period.Year != 2025 {
		t.Errorf("Year = %d, want 2025", got.Period.Year)
	}
	if got.Period.Month != 3 {
		t.Errorf("Month = %d, want 3", got.Period.Month)
	}
	if got.FilingType != domain.FilingTypeRegular {
		t.Errorf("FilingType = %q, want %q", got.FilingType, domain.FilingTypeRegular)
	}
	if string(got.XMLData) != "<xml>test</xml>" {
		t.Errorf("XMLData = %q, want %q", got.XMLData, "<xml>test</xml>")
	}
	if got.Status != domain.FilingStatusDraft {
		t.Errorf("Status = %q, want %q", got.Status, domain.FilingStatusDraft)
	}
	if got.FiledAt != nil {
		t.Errorf("FiledAt = %v, want nil", got.FiledAt)
	}
}

func TestVATControlStatementRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	cs := &domain.VATControlStatement{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 5,
		},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusDraft,
	}
	if err := repo.Create(ctx, cs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	cs.Status = domain.FilingStatusReady
	cs.XMLData = []byte("<xml>updated</xml>")
	cs.FilingType = domain.FilingTypeCorrective

	if err := repo.Update(ctx, cs); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, cs.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.Status != domain.FilingStatusReady {
		t.Errorf("Status = %q, want %q", got.Status, domain.FilingStatusReady)
	}
	if string(got.XMLData) != "<xml>updated</xml>" {
		t.Errorf("XMLData = %q, want %q", got.XMLData, "<xml>updated</xml>")
	}
	if got.FilingType != domain.FilingTypeCorrective {
		t.Errorf("FilingType = %q, want %q", got.FilingType, domain.FilingTypeCorrective)
	}
}

func TestVATControlStatementRepository_Delete_Existing(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	cs := &domain.VATControlStatement{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 1,
		},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusDraft,
	}
	if err := repo.Create(ctx, cs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := repo.Delete(ctx, cs.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := repo.GetByID(ctx, cs.ID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetByID after delete: got err %v, want domain.ErrNotFound", err)
	}
}

func TestVATControlStatementRepository_Delete_WithLines(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	cs := &domain.VATControlStatement{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 2,
		},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusDraft,
	}
	if err := repo.Create(ctx, cs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	lines := []domain.VATControlStatementLine{
		{
			ControlStatementID: cs.ID,
			Section:            domain.ControlSectionA4,
			PartnerDIC:         "CZ12345678",
			DocumentNumber:     "FV001",
			DPPD:               "2025-02-15",
			Base:               100000,
			VAT:                21000,
			VATRatePercent:     21,
		},
	}
	if err := repo.CreateLines(ctx, lines); err != nil {
		t.Fatalf("CreateLines() error: %v", err)
	}

	// Delete should remove both statement and lines.
	if err := repo.Delete(ctx, cs.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := repo.GetByID(ctx, cs.ID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetByID after delete: got err %v, want domain.ErrNotFound", err)
	}

	gotLines, err := repo.GetLines(ctx, cs.ID)
	if err != nil {
		t.Fatalf("GetLines() error: %v", err)
	}
	if len(gotLines) != 0 {
		t.Errorf("expected 0 lines after delete, got %d", len(gotLines))
	}
}

func TestVATControlStatementRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Delete non-existent: got err %v, want domain.ErrNotFound", err)
	}
}

func TestVATControlStatementRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetByID non-existent: got err %v, want domain.ErrNotFound", err)
	}
}

func TestVATControlStatementRepository_List(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	// Create statements for 2025.
	for _, month := range []int{3, 1, 2} {
		cs := &domain.VATControlStatement{
			Period: domain.TaxPeriod{
				Year:  2025,
				Month: month,
			},
			FilingType: domain.FilingTypeRegular,
			Status:     domain.FilingStatusDraft,
		}
		if err := repo.Create(ctx, cs); err != nil {
			t.Fatalf("Create() error for month %d: %v", month, err)
		}
	}

	// Create one for 2024.
	cs2024 := &domain.VATControlStatement{
		Period: domain.TaxPeriod{
			Year:  2024,
			Month: 12,
		},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusDraft,
	}
	if err := repo.Create(ctx, cs2024); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// List 2025 should return 3 items ordered by month ASC.
	results, err := repo.List(ctx, 2025)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("List(2025) returned %d items, want 3", len(results))
	}
	if results[0].Period.Month != 1 {
		t.Errorf("first result month = %d, want 1", results[0].Period.Month)
	}
	if results[1].Period.Month != 2 {
		t.Errorf("second result month = %d, want 2", results[1].Period.Month)
	}
	if results[2].Period.Month != 3 {
		t.Errorf("third result month = %d, want 3", results[2].Period.Month)
	}

	// List 2024 should return 1.
	results2024, err := repo.List(ctx, 2024)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(results2024) != 1 {
		t.Errorf("List(2024) returned %d items, want 1", len(results2024))
	}

	// List year with no data should return empty.
	results2023, err := repo.List(ctx, 2023)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(results2023) != 0 {
		t.Errorf("List(2023) returned %d items, want 0", len(results2023))
	}
}

func TestVATControlStatementRepository_GetByPeriod(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	cs := &domain.VATControlStatement{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 7,
		},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusDraft,
	}
	if err := repo.Create(ctx, cs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByPeriod(ctx, 2025, 7, domain.FilingTypeRegular)
	if err != nil {
		t.Fatalf("GetByPeriod() error: %v", err)
	}
	if got.ID != cs.ID {
		t.Errorf("GetByPeriod() ID = %d, want %d", got.ID, cs.ID)
	}
	if got.Period.Year != 2025 || got.Period.Month != 7 {
		t.Errorf("GetByPeriod() period = %d/%d, want 2025/7", got.Period.Year, got.Period.Month)
	}

	// Non-existent period.
	_, err = repo.GetByPeriod(ctx, 2025, 8, domain.FilingTypeRegular)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetByPeriod non-existent: got err %v, want domain.ErrNotFound", err)
	}

	// Same period but different filing type.
	_, err = repo.GetByPeriod(ctx, 2025, 7, domain.FilingTypeCorrective)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetByPeriod wrong filing type: got err %v, want domain.ErrNotFound", err)
	}
}

func TestVATControlStatementRepository_CreateAndGetLines_Nullable(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	cs := &domain.VATControlStatement{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 4,
		},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusDraft,
	}
	if err := repo.Create(ctx, cs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	lines := []domain.VATControlStatementLine{
		{
			ControlStatementID: cs.ID,
			Section:            domain.ControlSectionB2,
			PartnerDIC:         "CZ87654321",
			DocumentNumber:     "FV100",
			DPPD:               "2025-04-10",
			Base:               200000,
			VAT:                42000,
			VATRatePercent:     21,
			// InvoiceID and ExpenseID are nil.
		},
		{
			ControlStatementID: cs.ID,
			Section:            domain.ControlSectionA4,
			PartnerDIC:         "CZ11111111",
			DocumentNumber:     "FV101",
			DPPD:               "2025-04-15",
			Base:               500000,
			VAT:                105000,
			VATRatePercent:     21,
		},
	}

	if err := repo.CreateLines(ctx, lines); err != nil {
		t.Fatalf("CreateLines() error: %v", err)
	}

	// Verify IDs were set on the slice.
	for i, line := range lines {
		if line.ID == 0 {
			t.Errorf("line[%d].ID = 0, expected non-zero", i)
		}
	}

	gotLines, err := repo.GetLines(ctx, cs.ID)
	if err != nil {
		t.Fatalf("GetLines() error: %v", err)
	}
	if len(gotLines) != 2 {
		t.Fatalf("GetLines() returned %d lines, want 2", len(gotLines))
	}

	// Lines should be ordered by section ASC, id ASC.
	// A4 comes before B2.
	if gotLines[0].Section != domain.ControlSectionA4 {
		t.Errorf("first line section = %q, want %q", gotLines[0].Section, domain.ControlSectionA4)
	}
	if gotLines[1].Section != domain.ControlSectionB2 {
		t.Errorf("second line section = %q, want %q", gotLines[1].Section, domain.ControlSectionB2)
	}

	// Verify nullable fields are nil.
	if gotLines[0].InvoiceID != nil {
		t.Errorf("line[0].InvoiceID = %v, want nil", gotLines[0].InvoiceID)
	}
	if gotLines[0].ExpenseID != nil {
		t.Errorf("line[0].ExpenseID = %v, want nil", gotLines[0].ExpenseID)
	}

	// Verify round-trip values.
	if gotLines[0].PartnerDIC != "CZ11111111" {
		t.Errorf("line[0].PartnerDIC = %q, want %q", gotLines[0].PartnerDIC, "CZ11111111")
	}
	if gotLines[0].Base != 500000 {
		t.Errorf("line[0].Base = %d, want 500000", gotLines[0].Base)
	}
	if gotLines[0].VAT != 105000 {
		t.Errorf("line[0].VAT = %d, want 105000", gotLines[0].VAT)
	}
	if gotLines[0].VATRatePercent != 21 {
		t.Errorf("line[0].VATRatePercent = %d, want 21", gotLines[0].VATRatePercent)
	}
}

func TestVATControlStatementRepository_CreateLines_WithInvoiceID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	// Seed a contact and invoice for the foreign key.
	contact := testutil.SeedContact(t, db, nil)
	inv := testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Service", Quantity: 100, Unit: "ks", UnitPrice: 10000, VATRatePercent: 21},
	})

	cs := &domain.VATControlStatement{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 6,
		},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusDraft,
	}
	if err := repo.Create(ctx, cs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	invoiceID := inv.ID
	lines := []domain.VATControlStatementLine{
		{
			ControlStatementID: cs.ID,
			Section:            domain.ControlSectionA4,
			PartnerDIC:         "CZ12345678",
			DocumentNumber:     inv.InvoiceNumber,
			DPPD:               "2025-06-01",
			Base:               10000,
			VAT:                2100,
			VATRatePercent:     21,
			InvoiceID:          &invoiceID,
		},
	}

	if err := repo.CreateLines(ctx, lines); err != nil {
		t.Fatalf("CreateLines() error: %v", err)
	}

	gotLines, err := repo.GetLines(ctx, cs.ID)
	if err != nil {
		t.Fatalf("GetLines() error: %v", err)
	}
	if len(gotLines) != 1 {
		t.Fatalf("GetLines() returned %d lines, want 1", len(gotLines))
	}

	if gotLines[0].InvoiceID == nil {
		t.Fatal("line.InvoiceID = nil, want non-nil")
	}
	if *gotLines[0].InvoiceID != inv.ID {
		t.Errorf("line.InvoiceID = %d, want %d", *gotLines[0].InvoiceID, inv.ID)
	}
	if gotLines[0].ExpenseID != nil {
		t.Errorf("line.ExpenseID = %v, want nil", gotLines[0].ExpenseID)
	}
}

func TestVATControlStatementRepository_CreateLines_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	// Empty slice should be a no-op.
	if err := repo.CreateLines(ctx, []domain.VATControlStatementLine{}); err != nil {
		t.Fatalf("CreateLines(empty) error: %v", err)
	}

	// Nil slice should also be a no-op.
	if err := repo.CreateLines(ctx, nil); err != nil {
		t.Fatalf("CreateLines(nil) error: %v", err)
	}
}

func TestVATControlStatementRepository_CreateLines_WithExpenseID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	// Seed an expense for the foreign key.
	expense := testutil.SeedExpense(t, db, nil)

	cs := &domain.VATControlStatement{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 9,
		},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusDraft,
	}
	if err := repo.Create(ctx, cs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	expenseID := expense.ID
	lines := []domain.VATControlStatementLine{
		{
			ControlStatementID: cs.ID,
			Section:            domain.ControlSectionB2,
			PartnerDIC:         "CZ55555555",
			DocumentNumber:     "FP001",
			DPPD:               "2025-09-01",
			Base:               50000,
			VAT:                10500,
			VATRatePercent:     21,
			ExpenseID:          &expenseID,
		},
	}

	if err := repo.CreateLines(ctx, lines); err != nil {
		t.Fatalf("CreateLines() error: %v", err)
	}

	gotLines, err := repo.GetLines(ctx, cs.ID)
	if err != nil {
		t.Fatalf("GetLines() error: %v", err)
	}
	if len(gotLines) != 1 {
		t.Fatalf("GetLines() returned %d lines, want 1", len(gotLines))
	}
	if gotLines[0].ExpenseID == nil {
		t.Fatal("line.ExpenseID = nil, want non-nil")
	}
	if *gotLines[0].ExpenseID != expense.ID {
		t.Errorf("line.ExpenseID = %d, want %d", *gotLines[0].ExpenseID, expense.ID)
	}
	if gotLines[0].InvoiceID != nil {
		t.Errorf("line.InvoiceID = %v, want nil", gotLines[0].InvoiceID)
	}
}

func TestVATControlStatementRepository_Create_WithFiledAt(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	filedAt := time.Date(2025, 3, 25, 10, 30, 0, 0, time.UTC)
	cs := &domain.VATControlStatement{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 3,
		},
		FilingType: domain.FilingTypeRegular,
		XMLData:    []byte("<xml>filed</xml>"),
		Status:     domain.FilingStatusFiled,
		FiledAt:    &filedAt,
	}

	if err := repo.Create(ctx, cs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, cs.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.FiledAt == nil {
		t.Fatal("expected FiledAt to be set")
	}
	if got.Status != domain.FilingStatusFiled {
		t.Errorf("Status = %q, want %q", got.Status, domain.FilingStatusFiled)
	}
}

func TestVATControlStatementRepository_Update_WithFiledAt(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	cs := &domain.VATControlStatement{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 10,
		},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusDraft,
	}
	if err := repo.Create(ctx, cs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	filedAt := time.Date(2025, 10, 25, 14, 0, 0, 0, time.UTC)
	cs.FiledAt = &filedAt
	cs.Status = domain.FilingStatusFiled

	if err := repo.Update(ctx, cs); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, cs.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.FiledAt == nil {
		t.Fatal("expected FiledAt to be set after update")
	}
	if got.Status != domain.FilingStatusFiled {
		t.Errorf("Status = %q, want %q", got.Status, domain.FilingStatusFiled)
	}
}

func TestVATControlStatementRepository_GetLines_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	cs := &domain.VATControlStatement{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 11,
		},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusDraft,
	}
	if err := repo.Create(ctx, cs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Statement exists but has no lines.
	gotLines, err := repo.GetLines(ctx, cs.ID)
	if err != nil {
		t.Fatalf("GetLines() error: %v", err)
	}
	if gotLines != nil {
		t.Errorf("expected nil lines for statement with no lines, got %d", len(gotLines))
	}
}

func TestVATControlStatementRepository_GetLines_NonExistentStatement(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	gotLines, err := repo.GetLines(ctx, 99999)
	if err != nil {
		t.Fatalf("GetLines() error: %v", err)
	}
	if gotLines != nil {
		t.Errorf("expected nil lines for non-existent statement, got %d", len(gotLines))
	}
}

func TestVATControlStatementRepository_DeleteLines_NonExistentStatement(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	// Should not error even if no lines exist.
	if err := repo.DeleteLines(ctx, 99999); err != nil {
		t.Fatalf("DeleteLines() error: %v", err)
	}
}

func TestVATControlStatementRepository_List_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	results, err := repo.List(ctx, 2020)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil list for empty year, got %d", len(results))
	}
}

func TestVATControlStatementRepository_CreateLines_MultipleLines(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	cs := &domain.VATControlStatement{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 12,
		},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusDraft,
	}
	if err := repo.Create(ctx, cs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	lines := []domain.VATControlStatementLine{
		{
			ControlStatementID: cs.ID,
			Section:            domain.ControlSectionA4,
			PartnerDIC:         "CZ11111111",
			DocumentNumber:     "FV300",
			DPPD:               "2025-12-01",
			Base:               100000,
			VAT:                21000,
			VATRatePercent:     21,
		},
		{
			ControlStatementID: cs.ID,
			Section:            domain.ControlSectionA5,
			PartnerDIC:         "CZ22222222",
			DocumentNumber:     "FV301",
			DPPD:               "2025-12-05",
			Base:               200000,
			VAT:                42000,
			VATRatePercent:     21,
		},
		{
			ControlStatementID: cs.ID,
			Section:            domain.ControlSectionB2,
			PartnerDIC:         "CZ33333333",
			DocumentNumber:     "FP100",
			DPPD:               "2025-12-10",
			Base:               50000,
			VAT:                10500,
			VATRatePercent:     21,
		},
	}

	if err := repo.CreateLines(ctx, lines); err != nil {
		t.Fatalf("CreateLines() error: %v", err)
	}

	for i, line := range lines {
		if line.ID == 0 {
			t.Errorf("line[%d].ID = 0, expected non-zero", i)
		}
	}

	gotLines, err := repo.GetLines(ctx, cs.ID)
	if err != nil {
		t.Fatalf("GetLines() error: %v", err)
	}
	if len(gotLines) != 3 {
		t.Fatalf("GetLines() returned %d lines, want 3", len(gotLines))
	}

	// Verify ordering: A4, A5, B2 (section ASC, id ASC).
	if gotLines[0].Section != domain.ControlSectionA4 {
		t.Errorf("line[0].Section = %q, want %q", gotLines[0].Section, domain.ControlSectionA4)
	}
	if gotLines[1].Section != domain.ControlSectionA5 {
		t.Errorf("line[1].Section = %q, want %q", gotLines[1].Section, domain.ControlSectionA5)
	}
	if gotLines[2].Section != domain.ControlSectionB2 {
		t.Errorf("line[2].Section = %q, want %q", gotLines[2].Section, domain.ControlSectionB2)
	}
}

func TestVATControlStatementRepository_DeleteLines(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATControlStatementRepository(db)
	ctx := context.Background()

	cs := &domain.VATControlStatement{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 8,
		},
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusDraft,
	}
	if err := repo.Create(ctx, cs); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	lines := []domain.VATControlStatementLine{
		{
			ControlStatementID: cs.ID,
			Section:            domain.ControlSectionA5,
			PartnerDIC:         "CZ99999999",
			DocumentNumber:     "FV200",
			DPPD:               "2025-08-01",
			Base:               300000,
			VAT:                63000,
			VATRatePercent:     21,
		},
		{
			ControlStatementID: cs.ID,
			Section:            domain.ControlSectionB3,
			PartnerDIC:         "CZ88888888",
			DocumentNumber:     "FV201",
			DPPD:               "2025-08-05",
			Base:               150000,
			VAT:                31500,
			VATRatePercent:     21,
		},
	}
	if err := repo.CreateLines(ctx, lines); err != nil {
		t.Fatalf("CreateLines() error: %v", err)
	}

	// Verify lines exist.
	gotLines, err := repo.GetLines(ctx, cs.ID)
	if err != nil {
		t.Fatalf("GetLines() error: %v", err)
	}
	if len(gotLines) != 2 {
		t.Fatalf("GetLines() returned %d lines, want 2", len(gotLines))
	}

	// Delete all lines.
	if err := repo.DeleteLines(ctx, cs.ID); err != nil {
		t.Fatalf("DeleteLines() error: %v", err)
	}

	// Verify lines are gone.
	gotLines, err = repo.GetLines(ctx, cs.ID)
	if err != nil {
		t.Fatalf("GetLines() after delete error: %v", err)
	}
	if len(gotLines) != 0 {
		t.Errorf("expected 0 lines after DeleteLines, got %d", len(gotLines))
	}

	// Statement itself should still exist.
	_, err = repo.GetByID(ctx, cs.ID)
	if err != nil {
		t.Errorf("statement should still exist after DeleteLines, got err: %v", err)
	}
}
