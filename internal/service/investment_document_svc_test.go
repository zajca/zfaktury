package service

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

// newInvestmentDocumentService creates a test InvestmentDocumentService backed by real SQLite.
func newInvestmentDocumentService(t *testing.T) (*InvestmentDocumentService, *repository.InvestmentDocumentRepository) {
	t.Helper()
	db := testutil.NewTestDB(t)
	docRepo := repository.NewInvestmentDocumentRepository(db)
	capitalRepo := repository.NewCapitalIncomeRepository(db)
	securityRepo := repository.NewSecurityTransactionRepository(db)
	dataDir := t.TempDir()
	svc := NewInvestmentDocumentService(docRepo, capitalRepo, securityRepo, dataDir)
	return svc, docRepo
}

func TestInvestmentDocumentService_Upload_ValidPDF(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	doc, err := svc.Upload(ctx, 2025, domain.PlatformPortu, "statement.pdf", "application/pdf", data)
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}
	if doc.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if doc.Year != 2025 {
		t.Errorf("Year = %d, want 2025", doc.Year)
	}
	if doc.Platform != domain.PlatformPortu {
		t.Errorf("Platform = %q, want %q", doc.Platform, domain.PlatformPortu)
	}
	if doc.Filename != "statement.pdf" {
		t.Errorf("Filename = %q, want statement.pdf", doc.Filename)
	}
	if doc.Size != int64(len(pdfMagic)) {
		t.Errorf("Size = %d, want %d", doc.Size, len(pdfMagic))
	}
	if doc.ExtractionStatus != domain.ExtractionPending {
		t.Errorf("ExtractionStatus = %q, want %q", doc.ExtractionStatus, domain.ExtractionPending)
	}
}

func TestInvestmentDocumentService_Upload_ValidImage(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	data := bytes.NewReader(jpegMagic)
	doc, err := svc.Upload(ctx, 2025, domain.PlatformRevolut, "photo.jpg", "image/jpeg", data)
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}
	if doc.ContentType != "image/jpeg" {
		t.Errorf("ContentType = %q, want image/jpeg", doc.ContentType)
	}
}

func TestInvestmentDocumentService_Upload_ValidPNG(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	data := bytes.NewReader(pngMagic)
	doc, err := svc.Upload(ctx, 2025, domain.PlatformTrading212, "screenshot.png", "image/png", data)
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}
	if doc.ContentType != "image/png" {
		t.Errorf("ContentType = %q, want image/png", doc.ContentType)
	}
}

func TestInvestmentDocumentService_Upload_InvalidYear(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	tests := []struct {
		name string
		year int
	}{
		{"too low", 1999},
		{"too high", 2101},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := bytes.NewReader(pdfMagic)
			_, err := svc.Upload(ctx, tt.year, domain.PlatformPortu, "file.pdf", "application/pdf", data)
			if err == nil {
				t.Errorf("expected error for year %d", tt.year)
			}
		})
	}
}

func TestInvestmentDocumentService_Upload_InvalidPlatform(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	_, err := svc.Upload(ctx, 2025, "unknown_broker", "file.pdf", "application/pdf", data)
	if err == nil {
		t.Error("expected error for invalid platform")
	}
}

func TestInvestmentDocumentService_Upload_InvalidContentType(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	data := bytes.NewReader([]byte("not a valid file"))
	_, err := svc.Upload(ctx, 2025, domain.PlatformPortu, "file.exe", "application/octet-stream", data)
	if err == nil {
		t.Error("expected error for disallowed content type")
	}
}

func TestInvestmentDocumentService_Upload_EmptyFilename(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	// sanitizeFilename("") returns "." via filepath.Base, so upload succeeds.
	// This tests that the service doesn't panic on empty input.
	data := bytes.NewReader(pdfMagic)
	doc, err := svc.Upload(ctx, 2025, domain.PlatformPortu, "", "application/pdf", data)
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}
	if doc.Filename == "" {
		t.Error("expected non-empty sanitized filename")
	}
}

func TestInvestmentDocumentService_Upload_SanitizesFilename(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	doc, err := svc.Upload(ctx, 2025, domain.PlatformPortu, "../../etc/passwd", "application/pdf", data)
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}
	if doc.Filename == "../../etc/passwd" {
		t.Error("filename was not sanitized")
	}
}

func TestInvestmentDocumentService_Upload_TooLarge(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	oversized := bytes.NewReader(bytes.Repeat([]byte("a"), maxDocumentSize+1))
	_, err := svc.Upload(ctx, 2025, domain.PlatformPortu, "huge.pdf", "application/pdf", oversized)
	if err == nil {
		t.Error("expected error for oversized file")
	}
}

func TestInvestmentDocumentService_Upload_AllPlatforms(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	platforms := []string{
		domain.PlatformPortu,
		domain.PlatformZonky,
		domain.PlatformTrading212,
		domain.PlatformRevolut,
		domain.PlatformOther,
	}

	for _, p := range platforms {
		t.Run(p, func(t *testing.T) {
			data := bytes.NewReader(pdfMagic)
			_, err := svc.Upload(ctx, 2025, p, "file.pdf", "application/pdf", data)
			if err != nil {
				t.Errorf("Upload(%s) error: %v", p, err)
			}
		})
	}
}

func TestInvestmentDocumentService_GetByID_Valid(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	doc, err := svc.Upload(ctx, 2025, domain.PlatformPortu, "test.pdf", "application/pdf", data)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	got, err := svc.GetByID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.ID != doc.ID {
		t.Errorf("ID = %d, want %d", got.ID, doc.ID)
	}
	if got.Filename != "test.pdf" {
		t.Errorf("Filename = %q, want test.pdf", got.Filename)
	}
}

func TestInvestmentDocumentService_GetByID_ZeroID(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestInvestmentDocumentService_GetByID_NotFound(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent document")
	}
}

func TestInvestmentDocumentService_ListByYear(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	// Upload two documents for 2025.
	for _, name := range []string{"doc1.pdf", "doc2.pdf"} {
		data := bytes.NewReader(pdfMagic)
		if _, err := svc.Upload(ctx, 2025, domain.PlatformPortu, name, "application/pdf", data); err != nil {
			t.Fatalf("Upload %s: %v", name, err)
		}
	}
	// Upload one for 2024.
	data := bytes.NewReader(pdfMagic)
	if _, err := svc.Upload(ctx, 2024, domain.PlatformPortu, "old.pdf", "application/pdf", data); err != nil {
		t.Fatalf("Upload old.pdf: %v", err)
	}

	docs, err := svc.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear() error: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("len = %d, want 2", len(docs))
	}
}

func TestInvestmentDocumentService_ListByYear_InvalidYear(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	_, err := svc.ListByYear(ctx, 1999)
	if err == nil {
		t.Error("expected error for invalid year")
	}
}

func TestInvestmentDocumentService_ListByYear_Empty(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	docs, err := svc.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear() error: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("len = %d, want 0", len(docs))
	}
}

func TestInvestmentDocumentService_Delete_Valid(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	doc, err := svc.Upload(ctx, 2025, domain.PlatformPortu, "todelete.pdf", "application/pdf", data)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if err := svc.Delete(ctx, doc.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// Verify deleted.
	_, err = svc.GetByID(ctx, doc.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestInvestmentDocumentService_Delete_ZeroID(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	err := svc.Delete(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestInvestmentDocumentService_Delete_NotFound(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	err := svc.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error when deleting non-existent document")
	}
}

func TestInvestmentDocumentService_GetFilePath_Valid(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	doc, err := svc.Upload(ctx, 2025, domain.PlatformPortu, "file.pdf", "application/pdf", data)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	path, ct, err := svc.GetFilePath(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetFilePath() error: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
	if ct != "application/pdf" {
		t.Errorf("ContentType = %q, want application/pdf", ct)
	}
}

func TestInvestmentDocumentService_GetFilePath_ZeroID(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	_, _, err := svc.GetFilePath(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestInvestmentDocumentService_GetFilePath_NotFound(t *testing.T) {
	svc, _ := newInvestmentDocumentService(t)
	ctx := context.Background()

	_, _, err := svc.GetFilePath(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent document")
	}
}

func TestInvestmentDocumentService_Delete_CascadesLinkedEntries(t *testing.T) {
	db := testutil.NewTestDB(t)
	docRepo := repository.NewInvestmentDocumentRepository(db)
	capitalRepo := repository.NewCapitalIncomeRepository(db)
	securityRepo := repository.NewSecurityTransactionRepository(db)
	dataDir := t.TempDir()
	svc := NewInvestmentDocumentService(docRepo, capitalRepo, securityRepo, dataDir)
	ctx := context.Background()

	// Upload a document.
	data := bytes.NewReader(pdfMagic)
	doc, err := svc.Upload(ctx, 2025, domain.PlatformPortu, "cascade.pdf", "application/pdf", data)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	// Create linked capital income entry.
	docID := doc.ID
	capitalEntry := &domain.CapitalIncomeEntry{
		Year:        2025,
		DocumentID:  &docID,
		Category:    domain.CapitalCategoryDividendCZ,
		IncomeDate:  mustParseTime(t, "2025-01-01"),
		GrossAmount: 10000,
		NetAmount:   10000,
	}
	if err := capitalRepo.Create(ctx, capitalEntry); err != nil {
		t.Fatalf("Create capital: %v", err)
	}

	// Create linked security transaction.
	secTx := &domain.SecurityTransaction{
		Year:            2025,
		DocumentID:      &docID,
		AssetType:       domain.AssetTypeStock,
		AssetName:       "AAPL",
		ISIN:            "US0378331005",
		TransactionType: domain.TransactionTypeBuy,
		TransactionDate: mustParseTime(t, "2025-01-01"),
		Quantity:        10000,
		TotalAmount:     500000,
		CurrencyCode:    "CZK",
	}
	if err := securityRepo.Create(ctx, secTx); err != nil {
		t.Fatalf("Create security tx: %v", err)
	}

	// Delete the document -- should cascade delete linked entries.
	if err := svc.Delete(ctx, doc.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// Verify capital entry is deleted.
	_, err = capitalRepo.GetByID(ctx, capitalEntry.ID)
	if err == nil {
		t.Error("expected capital entry to be deleted after document delete")
	}

	// Verify security transaction is deleted.
	_, err = securityRepo.GetByID(ctx, secTx.ID)
	if err == nil {
		t.Error("expected security transaction to be deleted after document delete")
	}
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("parsing time %q: %v", s, err)
	}
	return parsed
}
