package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func seedInvestmentDocument(t *testing.T, repo *InvestmentDocumentRepository, doc *domain.InvestmentDocument) *domain.InvestmentDocument {
	t.Helper()
	if doc == nil {
		doc = &domain.InvestmentDocument{}
	}
	if doc.Year == 0 {
		doc.Year = 2025
	}
	if doc.Platform == "" {
		doc.Platform = domain.PlatformPortu
	}
	if doc.Filename == "" {
		doc.Filename = "test-statement.pdf"
	}
	if doc.ContentType == "" {
		doc.ContentType = "application/pdf"
	}
	if doc.StoragePath == "" {
		doc.StoragePath = "/data/docs/test-statement.pdf"
	}
	if doc.Size == 0 {
		doc.Size = 12345
	}
	if doc.ExtractionStatus == "" {
		doc.ExtractionStatus = domain.ExtractionPending
	}

	ctx := context.Background()
	if err := repo.Create(ctx, doc); err != nil {
		t.Fatalf("seedInvestmentDocument: %v", err)
	}
	return doc
}

func TestInvestmentDocumentRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewInvestmentDocumentRepository(db)
	ctx := context.Background()

	doc := &domain.InvestmentDocument{
		Year:             2025,
		Platform:         domain.PlatformPortu,
		Filename:         "portu-2025.pdf",
		ContentType:      "application/pdf",
		StoragePath:      "/data/docs/portu-2025.pdf",
		Size:             54321,
		ExtractionStatus: domain.ExtractionPending,
	}

	if err := repo.Create(ctx, doc); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if doc.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	if doc.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if doc.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestInvestmentDocumentRepository_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewInvestmentDocumentRepository(db)
	ctx := context.Background()

	seeded := seedInvestmentDocument(t, repo, &domain.InvestmentDocument{
		Year:     2025,
		Platform: domain.PlatformTrading212,
		Filename: "trading212-report.pdf",
		Size:     99999,
	})

	got, err := repo.GetByID(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.Year != 2025 {
		t.Errorf("Year = %d, want %d", got.Year, 2025)
	}
	if got.Platform != domain.PlatformTrading212 {
		t.Errorf("Platform = %q, want %q", got.Platform, domain.PlatformTrading212)
	}
	if got.Filename != "trading212-report.pdf" {
		t.Errorf("Filename = %q, want %q", got.Filename, "trading212-report.pdf")
	}
	if got.Size != 99999 {
		t.Errorf("Size = %d, want %d", got.Size, 99999)
	}
	if got.ExtractionStatus != domain.ExtractionPending {
		t.Errorf("ExtractionStatus = %q, want %q", got.ExtractionStatus, domain.ExtractionPending)
	}
}

func TestInvestmentDocumentRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewInvestmentDocumentRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent document")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestInvestmentDocumentRepository_ListByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewInvestmentDocumentRepository(db)
	ctx := context.Background()

	seedInvestmentDocument(t, repo, &domain.InvestmentDocument{Year: 2025, Filename: "doc1.pdf"})
	seedInvestmentDocument(t, repo, &domain.InvestmentDocument{Year: 2025, Filename: "doc2.pdf"})
	seedInvestmentDocument(t, repo, &domain.InvestmentDocument{Year: 2024, Filename: "doc3.pdf"})

	docs, err := repo.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear() error: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("len(docs) = %d, want 2", len(docs))
	}
}

func TestInvestmentDocumentRepository_ListByYear_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewInvestmentDocumentRepository(db)
	ctx := context.Background()

	docs, err := repo.ListByYear(ctx, 2099)
	if err != nil {
		t.Fatalf("ListByYear() error: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("expected empty result, got %d", len(docs))
	}
}

func TestInvestmentDocumentRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewInvestmentDocumentRepository(db)
	ctx := context.Background()

	seeded := seedInvestmentDocument(t, repo, nil)

	if err := repo.Delete(ctx, seeded.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := repo.GetByID(ctx, seeded.ID)
	if err == nil {
		t.Error("expected error when getting deleted document")
	}
}

func TestInvestmentDocumentRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewInvestmentDocumentRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent document")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestInvestmentDocumentRepository_UpdateExtraction(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewInvestmentDocumentRepository(db)
	ctx := context.Background()

	seeded := seedInvestmentDocument(t, repo, nil)

	if err := repo.UpdateExtraction(ctx, seeded.ID, domain.ExtractionExtracted, ""); err != nil {
		t.Fatalf("UpdateExtraction() error: %v", err)
	}

	got, err := repo.GetByID(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.ExtractionStatus != domain.ExtractionExtracted {
		t.Errorf("ExtractionStatus = %q, want %q", got.ExtractionStatus, domain.ExtractionExtracted)
	}
}

func TestInvestmentDocumentRepository_UpdateExtraction_WithError(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewInvestmentDocumentRepository(db)
	ctx := context.Background()

	seeded := seedInvestmentDocument(t, repo, nil)

	if err := repo.UpdateExtraction(ctx, seeded.ID, domain.ExtractionFailed, "OCR service unavailable"); err != nil {
		t.Fatalf("UpdateExtraction() error: %v", err)
	}

	got, err := repo.GetByID(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.ExtractionStatus != domain.ExtractionFailed {
		t.Errorf("ExtractionStatus = %q, want %q", got.ExtractionStatus, domain.ExtractionFailed)
	}
	if got.ExtractionError != "OCR service unavailable" {
		t.Errorf("ExtractionError = %q, want %q", got.ExtractionError, "OCR service unavailable")
	}
}

func TestInvestmentDocumentRepository_UpdateExtraction_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewInvestmentDocumentRepository(db)
	ctx := context.Background()

	err := repo.UpdateExtraction(ctx, 99999, domain.ExtractionExtracted, "")
	if err == nil {
		t.Error("expected error for non-existent document")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}
