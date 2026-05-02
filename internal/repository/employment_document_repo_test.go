package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func seedEmploymentDocument(t *testing.T, repo *EmploymentDocumentRepository, doc *domain.EmploymentDocument) *domain.EmploymentDocument {
	t.Helper()
	if doc == nil {
		doc = &domain.EmploymentDocument{}
	}
	if doc.Year == 0 {
		doc.Year = 2025
	}
	if doc.Filename == "" {
		doc.Filename = "potvrzeni.pdf"
	}
	if doc.ContentType == "" {
		doc.ContentType = "application/pdf"
	}
	if doc.StoragePath == "" {
		doc.StoragePath = "/data/employment_docs/2025/potvrzeni.pdf"
	}
	if doc.Size == 0 {
		doc.Size = 12345
	}

	ctx := context.Background()
	if err := repo.Create(ctx, doc); err != nil {
		t.Fatalf("seedEmploymentDocument: %v", err)
	}
	return doc
}

func TestEmploymentDocumentRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentDocumentRepository(db)
	ctx := context.Background()

	doc := &domain.EmploymentDocument{
		Year:        2025,
		Kind:        domain.EmploymentDocAdvance,
		Filename:    "potvrzeni-vzor33.pdf",
		ContentType: "application/pdf",
		StoragePath: "/data/employment_docs/2025/potvrzeni-vzor33.pdf",
		Size:        54321,
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
	if doc.ExtractionStatus != domain.ExtractionPending {
		t.Errorf("default ExtractionStatus = %q, want %q", doc.ExtractionStatus, domain.ExtractionPending)
	}
}

func TestEmploymentDocumentRepository_CreateDefaultsKind(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentDocumentRepository(db)
	ctx := context.Background()

	doc := &domain.EmploymentDocument{
		Year:        2025,
		Filename:    "no-kind.pdf",
		ContentType: "application/pdf",
		StoragePath: "/data/employment_docs/2025/no-kind.pdf",
	}
	if err := repo.Create(ctx, doc); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if doc.Kind != domain.EmploymentDocAdvance {
		t.Errorf("default Kind = %q, want %q", doc.Kind, domain.EmploymentDocAdvance)
	}
}

func TestEmploymentDocumentRepository_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentDocumentRepository(db)
	ctx := context.Background()

	seeded := seedEmploymentDocument(t, repo, &domain.EmploymentDocument{
		Year:     2025,
		Kind:     domain.EmploymentDocWithholding,
		Filename: "vzor12.pdf",
		Size:     99999,
	})

	got, err := repo.GetByID(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Year != 2025 {
		t.Errorf("Year = %d, want %d", got.Year, 2025)
	}
	if got.Kind != domain.EmploymentDocWithholding {
		t.Errorf("Kind = %q, want %q", got.Kind, domain.EmploymentDocWithholding)
	}
	if got.Filename != "vzor12.pdf" {
		t.Errorf("Filename = %q, want %q", got.Filename, "vzor12.pdf")
	}
	if got.Size != 99999 {
		t.Errorf("Size = %d, want %d", got.Size, 99999)
	}
}

func TestEmploymentDocumentRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentDocumentRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent document")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestEmploymentDocumentRepository_ListByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentDocumentRepository(db)
	ctx := context.Background()

	seedEmploymentDocument(t, repo, &domain.EmploymentDocument{Year: 2025, Filename: "doc1.pdf"})
	seedEmploymentDocument(t, repo, &domain.EmploymentDocument{Year: 2025, Filename: "doc2.pdf"})
	seedEmploymentDocument(t, repo, &domain.EmploymentDocument{Year: 2024, Filename: "doc3.pdf"})

	docs, err := repo.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear() error: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("len(docs) = %d, want 2", len(docs))
	}
}

func TestEmploymentDocumentRepository_ListByYear_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentDocumentRepository(db)
	ctx := context.Background()

	docs, err := repo.ListByYear(ctx, 2099)
	if err != nil {
		t.Fatalf("ListByYear() error: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("expected empty result, got %d", len(docs))
	}
}

func TestEmploymentDocumentRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentDocumentRepository(db)
	ctx := context.Background()

	seeded := seedEmploymentDocument(t, repo, nil)

	if err := repo.Delete(ctx, seeded.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := repo.GetByID(ctx, seeded.ID)
	if err == nil {
		t.Error("expected error when getting deleted document")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestEmploymentDocumentRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentDocumentRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent document")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestEmploymentDocumentRepository_UpdateExtraction(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentDocumentRepository(db)
	ctx := context.Background()

	seeded := seedEmploymentDocument(t, repo, nil)

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

func TestEmploymentDocumentRepository_UpdateExtraction_WithError(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentDocumentRepository(db)
	ctx := context.Background()

	seeded := seedEmploymentDocument(t, repo, nil)

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

func TestEmploymentDocumentRepository_UpdateExtraction_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewEmploymentDocumentRepository(db)
	ctx := context.Background()

	err := repo.UpdateExtraction(ctx, 99999, domain.ExtractionExtracted, "")
	if err == nil {
		t.Error("expected error for non-existent document")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}
