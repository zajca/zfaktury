package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func seedCapitalIncomeEntry(t *testing.T, repo *CapitalIncomeRepository, entry *domain.CapitalIncomeEntry) *domain.CapitalIncomeEntry {
	t.Helper()
	if entry == nil {
		entry = &domain.CapitalIncomeEntry{}
	}
	if entry.Year == 0 {
		entry.Year = 2025
	}
	if entry.Category == "" {
		entry.Category = domain.CapitalCategoryDividendForeign
	}
	if entry.Description == "" {
		entry.Description = "Test dividend"
	}
	if entry.IncomeDate.IsZero() {
		entry.IncomeDate = time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	}
	if entry.GrossAmount == 0 {
		entry.GrossAmount = domain.NewAmount(1000, 0)
	}
	if entry.NetAmount == 0 {
		entry.NetAmount = domain.NewAmount(850, 0)
	}
	if entry.CountryCode == "" {
		entry.CountryCode = "US"
	}

	ctx := context.Background()
	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("seedCapitalIncomeEntry: %v", err)
	}
	return entry
}

func TestCapitalIncomeRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCapitalIncomeRepository(db)
	ctx := context.Background()

	entry := &domain.CapitalIncomeEntry{
		Year:               2025,
		Category:           domain.CapitalCategoryDividendForeign,
		Description:        "Apple Inc. dividend",
		IncomeDate:         time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
		GrossAmount:        domain.NewAmount(500, 0),
		WithheldTaxCZ:      0,
		WithheldTaxForeign: domain.NewAmount(75, 0),
		CountryCode:        "US",
		NeedsDeclaring:     true,
		NetAmount:          domain.NewAmount(425, 0),
	}

	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if entry.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	if entry.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestCapitalIncomeRepository_Create_WithDocumentID(t *testing.T) {
	db := testutil.NewTestDB(t)
	docRepo := NewInvestmentDocumentRepository(db)
	repo := NewCapitalIncomeRepository(db)
	ctx := context.Background()

	doc := seedInvestmentDocument(t, docRepo, nil)
	docID := doc.ID

	entry := &domain.CapitalIncomeEntry{
		Year:        2025,
		DocumentID:  &docID,
		Category:    domain.CapitalCategoryInterest,
		Description: "Interest payment",
		IncomeDate:  time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC),
		GrossAmount: domain.NewAmount(200, 0),
		NetAmount:   domain.NewAmount(200, 0),
		CountryCode: "CZ",
	}

	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.DocumentID == nil || *got.DocumentID != docID {
		t.Errorf("DocumentID = %v, want %d", got.DocumentID, docID)
	}
}

func TestCapitalIncomeRepository_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCapitalIncomeRepository(db)
	ctx := context.Background()

	seeded := seedCapitalIncomeEntry(t, repo, &domain.CapitalIncomeEntry{
		Year:        2025,
		Category:    domain.CapitalCategoryDividendCZ,
		Description: "CEZ dividenda",
		CountryCode: "CZ",
		GrossAmount: domain.NewAmount(300, 0),
		NetAmount:   domain.NewAmount(255, 0),
	})

	got, err := repo.GetByID(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.Category != domain.CapitalCategoryDividendCZ {
		t.Errorf("Category = %q, want %q", got.Category, domain.CapitalCategoryDividendCZ)
	}
	if got.Description != "CEZ dividenda" {
		t.Errorf("Description = %q, want %q", got.Description, "CEZ dividenda")
	}
	if got.GrossAmount != domain.NewAmount(300, 0) {
		t.Errorf("GrossAmount = %d, want %d", got.GrossAmount, domain.NewAmount(300, 0))
	}
}

func TestCapitalIncomeRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCapitalIncomeRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent entry")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestCapitalIncomeRepository_ListByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCapitalIncomeRepository(db)
	ctx := context.Background()

	seedCapitalIncomeEntry(t, repo, &domain.CapitalIncomeEntry{Year: 2025, IncomeDate: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)})
	seedCapitalIncomeEntry(t, repo, &domain.CapitalIncomeEntry{Year: 2025, IncomeDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)})
	seedCapitalIncomeEntry(t, repo, &domain.CapitalIncomeEntry{Year: 2024, IncomeDate: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)})

	entries, err := repo.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear() error: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("len(entries) = %d, want 2", len(entries))
	}

	// Verify ordered by income_date ASC.
	if len(entries) >= 2 && entries[0].IncomeDate.After(entries[1].IncomeDate) {
		t.Error("expected entries ordered by income_date ASC")
	}
}

func TestCapitalIncomeRepository_ListByYear_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCapitalIncomeRepository(db)
	ctx := context.Background()

	entries, err := repo.ListByYear(ctx, 2099)
	if err != nil {
		t.Fatalf("ListByYear() error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty result, got %d", len(entries))
	}
}

func TestCapitalIncomeRepository_ListByDocumentID(t *testing.T) {
	db := testutil.NewTestDB(t)
	docRepo := NewInvestmentDocumentRepository(db)
	repo := NewCapitalIncomeRepository(db)
	ctx := context.Background()

	doc := seedInvestmentDocument(t, docRepo, nil)
	docID := doc.ID

	seedCapitalIncomeEntry(t, repo, &domain.CapitalIncomeEntry{DocumentID: &docID})
	seedCapitalIncomeEntry(t, repo, &domain.CapitalIncomeEntry{DocumentID: &docID})
	seedCapitalIncomeEntry(t, repo, nil) // no document_id

	entries, err := repo.ListByDocumentID(ctx, docID)
	if err != nil {
		t.Fatalf("ListByDocumentID() error: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("len(entries) = %d, want 2", len(entries))
	}
}

func TestCapitalIncomeRepository_SumByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCapitalIncomeRepository(db)
	ctx := context.Background()

	// Entry that needs declaring.
	seedCapitalIncomeEntry(t, repo, &domain.CapitalIncomeEntry{
		Year:               2025,
		GrossAmount:        domain.NewAmount(1000, 0),
		WithheldTaxCZ:      domain.NewAmount(50, 0),
		WithheldTaxForeign: domain.NewAmount(100, 0),
		NetAmount:          domain.NewAmount(850, 0),
		NeedsDeclaring:     true,
	})
	// Another entry that needs declaring.
	seedCapitalIncomeEntry(t, repo, &domain.CapitalIncomeEntry{
		Year:               2025,
		GrossAmount:        domain.NewAmount(500, 0),
		WithheldTaxCZ:      domain.NewAmount(25, 0),
		WithheldTaxForeign: domain.NewAmount(50, 0),
		NetAmount:          domain.NewAmount(425, 0),
		NeedsDeclaring:     true,
	})
	// Entry that does NOT need declaring -- should be excluded from sum.
	seedCapitalIncomeEntry(t, repo, &domain.CapitalIncomeEntry{
		Year:           2025,
		GrossAmount:    domain.NewAmount(200, 0),
		NetAmount:      domain.NewAmount(200, 0),
		NeedsDeclaring: false,
	})

	grossTotal, taxTotal, netTotal, err := repo.SumByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("SumByYear() error: %v", err)
	}

	// Only the two declaring entries should be summed.
	expectedGross := domain.NewAmount(1500, 0)
	expectedTax := domain.NewAmount(225, 0)
	expectedNet := domain.NewAmount(1275, 0)

	if grossTotal != expectedGross {
		t.Errorf("grossTotal = %d, want %d", grossTotal, expectedGross)
	}
	if taxTotal != expectedTax {
		t.Errorf("taxTotal = %d, want %d", taxTotal, expectedTax)
	}
	if netTotal != expectedNet {
		t.Errorf("netTotal = %d, want %d", netTotal, expectedNet)
	}
}

func TestCapitalIncomeRepository_SumByYear_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCapitalIncomeRepository(db)
	ctx := context.Background()

	grossTotal, taxTotal, netTotal, err := repo.SumByYear(ctx, 2099)
	if err != nil {
		t.Fatalf("SumByYear() error: %v", err)
	}
	if grossTotal != 0 || taxTotal != 0 || netTotal != 0 {
		t.Errorf("expected all zeros for empty year, got gross=%d tax=%d net=%d", grossTotal, taxTotal, netTotal)
	}
}

func TestCapitalIncomeRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCapitalIncomeRepository(db)
	ctx := context.Background()

	seeded := seedCapitalIncomeEntry(t, repo, nil)

	seeded.Description = "Updated description"
	seeded.GrossAmount = domain.NewAmount(2000, 0)
	seeded.NeedsDeclaring = true
	if err := repo.Update(ctx, seeded); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Description != "Updated description" {
		t.Errorf("Description = %q, want %q", got.Description, "Updated description")
	}
	if got.GrossAmount != domain.NewAmount(2000, 0) {
		t.Errorf("GrossAmount = %d, want %d", got.GrossAmount, domain.NewAmount(2000, 0))
	}
	if !got.NeedsDeclaring {
		t.Error("expected NeedsDeclaring to be true")
	}
}

func TestCapitalIncomeRepository_Update_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCapitalIncomeRepository(db)
	ctx := context.Background()

	entry := &domain.CapitalIncomeEntry{
		ID:          99999,
		Year:        2025,
		Category:    domain.CapitalCategoryInterest,
		Description: "Ghost",
		IncomeDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	err := repo.Update(ctx, entry)
	if err == nil {
		t.Error("expected error for non-existent entry")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestCapitalIncomeRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCapitalIncomeRepository(db)
	ctx := context.Background()

	seeded := seedCapitalIncomeEntry(t, repo, nil)

	if err := repo.Delete(ctx, seeded.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := repo.GetByID(ctx, seeded.ID)
	if err == nil {
		t.Error("expected error when getting deleted entry")
	}
}

func TestCapitalIncomeRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCapitalIncomeRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent entry")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestCapitalIncomeRepository_DeleteByDocumentID(t *testing.T) {
	db := testutil.NewTestDB(t)
	docRepo := NewInvestmentDocumentRepository(db)
	repo := NewCapitalIncomeRepository(db)
	ctx := context.Background()

	doc := seedInvestmentDocument(t, docRepo, nil)
	docID := doc.ID

	seedCapitalIncomeEntry(t, repo, &domain.CapitalIncomeEntry{DocumentID: &docID})
	seedCapitalIncomeEntry(t, repo, &domain.CapitalIncomeEntry{DocumentID: &docID})

	if err := repo.DeleteByDocumentID(ctx, docID); err != nil {
		t.Fatalf("DeleteByDocumentID() error: %v", err)
	}

	entries, err := repo.ListByDocumentID(ctx, docID)
	if err != nil {
		t.Fatalf("ListByDocumentID() error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after delete, got %d", len(entries))
	}
}

func TestCapitalIncomeRepository_DeleteByDocumentID_NoEntries(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewCapitalIncomeRepository(db)
	ctx := context.Background()

	// Should not error even when no entries exist for the document.
	if err := repo.DeleteByDocumentID(ctx, 99999); err != nil {
		t.Fatalf("DeleteByDocumentID() should not error for non-existent document, got: %v", err)
	}
}
