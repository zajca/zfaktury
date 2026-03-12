package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestIncomeTaxReturnRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewIncomeTaxReturnRepository(db)
	ctx := context.Background()

	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}

	if err := repo.Create(ctx, itr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if itr.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	if itr.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if itr.Status != domain.FilingStatusDraft {
		t.Errorf("Status = %q, want %q", itr.Status, domain.FilingStatusDraft)
	}
}

func TestIncomeTaxReturnRepository_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewIncomeTaxReturnRepository(db)
	ctx := context.Background()

	itr := &domain.IncomeTaxReturn{
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		TotalRevenue:   domain.NewAmount(500000, 0),
		ActualExpenses: domain.NewAmount(200000, 0),
	}
	if err := repo.Create(ctx, itr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, itr.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.Year != 2025 {
		t.Errorf("Year = %d, want 2025", got.Year)
	}
	if got.TotalRevenue != domain.NewAmount(500000, 0) {
		t.Errorf("TotalRevenue = %d, want %d", got.TotalRevenue, domain.NewAmount(500000, 0))
	}
	if got.ActualExpenses != domain.NewAmount(200000, 0) {
		t.Errorf("ActualExpenses = %d, want %d", got.ActualExpenses, domain.NewAmount(200000, 0))
	}
}

func TestIncomeTaxReturnRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewIncomeTaxReturnRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent income_tax_return")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestIncomeTaxReturnRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewIncomeTaxReturnRepository(db)
	ctx := context.Background()

	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, itr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	itr.TotalRevenue = domain.NewAmount(1000000, 0)
	itr.TaxBase = domain.NewAmount(800000, 0)
	itr.TotalTax = domain.NewAmount(120000, 0)

	if err := repo.Update(ctx, itr); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, itr.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.TotalRevenue != domain.NewAmount(1000000, 0) {
		t.Errorf("TotalRevenue = %d, want %d", got.TotalRevenue, domain.NewAmount(1000000, 0))
	}
	if got.TaxBase != domain.NewAmount(800000, 0) {
		t.Errorf("TaxBase = %d, want %d", got.TaxBase, domain.NewAmount(800000, 0))
	}
}

func TestIncomeTaxReturnRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewIncomeTaxReturnRepository(db)
	ctx := context.Background()

	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, itr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := repo.Delete(ctx, itr.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := repo.GetByID(ctx, itr.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestIncomeTaxReturnRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewIncomeTaxReturnRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent delete")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestIncomeTaxReturnRepository_List(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewIncomeTaxReturnRepository(db)
	ctx := context.Background()

	// Create two returns for 2025 (different filing types) and one for 2024.
	filings := []struct {
		year       int
		filingType string
	}{
		{2025, domain.FilingTypeRegular},
		{2025, domain.FilingTypeCorrective},
		{2024, domain.FilingTypeRegular},
	}
	for _, f := range filings {
		itr := &domain.IncomeTaxReturn{
			Year:       f.year,
			FilingType: f.filingType,
		}
		if err := repo.Create(ctx, itr); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	returns, err := repo.List(ctx, 2025)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(returns) != 2 {
		t.Errorf("List(2025) returned %d items, want 2", len(returns))
	}

	returns2024, err := repo.List(ctx, 2024)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(returns2024) != 1 {
		t.Errorf("List(2024) returned %d items, want 1", len(returns2024))
	}
}

func TestIncomeTaxReturnRepository_GetByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewIncomeTaxReturnRepository(db)
	ctx := context.Background()

	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, itr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByYear(ctx, 2025, domain.FilingTypeRegular)
	if err != nil {
		t.Fatalf("GetByYear() error: %v", err)
	}
	if got.ID != itr.ID {
		t.Errorf("GetByYear() ID = %d, want %d", got.ID, itr.ID)
	}

	// Non-existent year.
	_, err = repo.GetByYear(ctx, 2099, domain.FilingTypeRegular)
	if err == nil {
		t.Error("expected error for non-existent year")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestIncomeTaxReturnRepository_LinkInvoices(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewIncomeTaxReturnRepository(db)
	ctx := context.Background()

	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, itr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Create a contact and invoice for linking.
	contact := testutil.SeedContact(t, db, nil)
	inv := testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Test", Quantity: 100, Unit: "ks", UnitPrice: 10000, VATRatePercent: 21},
	})

	// Link invoices.
	if err := repo.LinkInvoices(ctx, itr.ID, []int64{inv.ID}); err != nil {
		t.Fatalf("LinkInvoices() error: %v", err)
	}

	ids, err := repo.GetLinkedInvoiceIDs(ctx, itr.ID)
	if err != nil {
		t.Fatalf("GetLinkedInvoiceIDs() error: %v", err)
	}
	if len(ids) != 1 || ids[0] != inv.ID {
		t.Errorf("GetLinkedInvoiceIDs() = %v, want [%d]", ids, inv.ID)
	}

	// Re-link with empty list should clear.
	if err := repo.LinkInvoices(ctx, itr.ID, []int64{}); err != nil {
		t.Fatalf("LinkInvoices() clear error: %v", err)
	}
	ids, err = repo.GetLinkedInvoiceIDs(ctx, itr.ID)
	if err != nil {
		t.Fatalf("GetLinkedInvoiceIDs() error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected empty linked invoices after clear, got %v", ids)
	}
}

func TestIncomeTaxReturnRepository_LinkExpenses(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewIncomeTaxReturnRepository(db)
	ctx := context.Background()

	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, itr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	exp1 := testutil.SeedExpense(t, db, &domain.Expense{Description: "Expense 1"})
	exp2 := testutil.SeedExpense(t, db, &domain.Expense{Description: "Expense 2"})

	if err := repo.LinkExpenses(ctx, itr.ID, []int64{exp1.ID, exp2.ID}); err != nil {
		t.Fatalf("LinkExpenses() error: %v", err)
	}

	ids, err := repo.GetLinkedExpenseIDs(ctx, itr.ID)
	if err != nil {
		t.Fatalf("GetLinkedExpenseIDs() error: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("GetLinkedExpenseIDs() returned %d items, want 2", len(ids))
	}

	// Re-link with empty list should clear.
	if err := repo.LinkExpenses(ctx, itr.ID, []int64{}); err != nil {
		t.Fatalf("LinkExpenses() clear error: %v", err)
	}
	ids, err = repo.GetLinkedExpenseIDs(ctx, itr.ID)
	if err != nil {
		t.Fatalf("GetLinkedExpenseIDs() error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected empty linked expenses after clear, got %v", ids)
	}
}
