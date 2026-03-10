package repository

import (
	"context"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestVATReturnRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATReturnRepository(db)
	ctx := context.Background()

	vr := &domain.VATReturn{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 3,
		},
		FilingType: domain.FilingTypeRegular,
	}

	if err := repo.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if vr.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	if vr.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if vr.Status != "draft" {
		t.Errorf("Status = %q, want %q", vr.Status, "draft")
	}
}

func TestVATReturnRepository_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATReturnRepository(db)
	ctx := context.Background()

	vr := &domain.VATReturn{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 6,
		},
		FilingType:        domain.FilingTypeRegular,
		OutputVATBase21:   1000000,
		OutputVATAmount21: 210000,
	}
	if err := repo.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, vr.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.Period.Year != 2025 {
		t.Errorf("Year = %d, want 2025", got.Period.Year)
	}
	if got.Period.Month != 6 {
		t.Errorf("Month = %d, want 6", got.Period.Month)
	}
	if got.OutputVATBase21 != 1000000 {
		t.Errorf("OutputVATBase21 = %d, want 1000000", got.OutputVATBase21)
	}
	if got.OutputVATAmount21 != 210000 {
		t.Errorf("OutputVATAmount21 = %d, want 210000", got.OutputVATAmount21)
	}
}

func TestVATReturnRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATReturnRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent vat_return")
	}
}

func TestVATReturnRepository_List(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATReturnRepository(db)
	ctx := context.Background()

	// Create returns for 2025.
	for _, month := range []int{1, 2, 3} {
		vr := &domain.VATReturn{
			Period: domain.TaxPeriod{
				Year:  2025,
				Month: month,
			},
			FilingType: domain.FilingTypeRegular,
		}
		if err := repo.Create(ctx, vr); err != nil {
			t.Fatalf("Create() error for month %d: %v", month, err)
		}
	}

	// Create one for different year.
	vr2024 := &domain.VATReturn{
		Period: domain.TaxPeriod{
			Year:  2024,
			Month: 12,
		},
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, vr2024); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// List 2025.
	returns, err := repo.List(ctx, 2025)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(returns) != 3 {
		t.Errorf("List(2025) returned %d items, want 3", len(returns))
	}

	// List 2024.
	returns2024, err := repo.List(ctx, 2024)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(returns2024) != 1 {
		t.Errorf("List(2024) returned %d items, want 1", len(returns2024))
	}
}

func TestVATReturnRepository_GetByPeriod(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATReturnRepository(db)
	ctx := context.Background()

	vr := &domain.VATReturn{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 7,
		},
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByPeriod(ctx, 2025, 7, 0, domain.FilingTypeRegular)
	if err != nil {
		t.Fatalf("GetByPeriod() error: %v", err)
	}
	if got.ID != vr.ID {
		t.Errorf("GetByPeriod() ID = %d, want %d", got.ID, vr.ID)
	}

	// Non-existent period.
	_, err = repo.GetByPeriod(ctx, 2025, 8, 0, domain.FilingTypeRegular)
	if err == nil {
		t.Error("expected error for non-existent period")
	}
}

func TestVATReturnRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATReturnRepository(db)
	ctx := context.Background()

	vr := &domain.VATReturn{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 1,
		},
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := repo.Delete(ctx, vr.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := repo.GetByID(ctx, vr.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestVATReturnRepository_LinkInvoices(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATReturnRepository(db)
	ctx := context.Background()

	// Create a VAT return.
	vr := &domain.VATReturn{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 1,
		},
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Create a contact and invoice for linking.
	contact := testutil.SeedContact(t, db, nil)
	inv := testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Test", Quantity: 100, Unit: "ks", UnitPrice: 10000, VATRatePercent: 21},
	})

	// Link invoices.
	if err := repo.LinkInvoices(ctx, vr.ID, []int64{inv.ID}); err != nil {
		t.Fatalf("LinkInvoices() error: %v", err)
	}

	// Verify linked IDs.
	ids, err := repo.GetLinkedInvoiceIDs(ctx, vr.ID)
	if err != nil {
		t.Fatalf("GetLinkedInvoiceIDs() error: %v", err)
	}
	if len(ids) != 1 || ids[0] != inv.ID {
		t.Errorf("GetLinkedInvoiceIDs() = %v, want [%d]", ids, inv.ID)
	}

	// Re-link with empty list should clear.
	if err := repo.LinkInvoices(ctx, vr.ID, []int64{}); err != nil {
		t.Fatalf("LinkInvoices() clear error: %v", err)
	}
	ids, err = repo.GetLinkedInvoiceIDs(ctx, vr.ID)
	if err != nil {
		t.Fatalf("GetLinkedInvoiceIDs() error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected empty linked invoices after clear, got %v", ids)
	}
}

func TestVATReturnRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewVATReturnRepository(db)
	ctx := context.Background()

	vr := &domain.VATReturn{
		Period: domain.TaxPeriod{
			Year:  2025,
			Month: 5,
		},
		FilingType: domain.FilingTypeRegular,
	}
	if err := repo.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Update amounts.
	vr.OutputVATBase21 = 500000
	vr.OutputVATAmount21 = 105000
	vr.TotalOutputVAT = 105000
	vr.NetVAT = 105000

	if err := repo.Update(ctx, vr); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, vr.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.OutputVATBase21 != 500000 {
		t.Errorf("OutputVATBase21 = %d, want 500000", got.OutputVATBase21)
	}
	if got.NetVAT != 105000 {
		t.Errorf("NetVAT = %d, want 105000", got.NetVAT)
	}
}
