package service

import (
	"context"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

// mockInvoiceDocumentRepo is a minimal in-memory InvoiceDocumentRepo for unit tests.
type mockInvoiceDocumentRepo struct {
	docs   map[int64]*domain.InvoiceDocument
	nextID int64
}

func newMockInvoiceDocumentRepo() *mockInvoiceDocumentRepo {
	return &mockInvoiceDocumentRepo{
		docs:   make(map[int64]*domain.InvoiceDocument),
		nextID: 1,
	}
}

func (m *mockInvoiceDocumentRepo) Create(ctx context.Context, doc *domain.InvoiceDocument) error {
	doc.ID = m.nextID
	m.nextID++
	cp := *doc
	m.docs[doc.ID] = &cp
	return nil
}

func (m *mockInvoiceDocumentRepo) GetByID(ctx context.Context, id int64) (*domain.InvoiceDocument, error) {
	doc, ok := m.docs[id]
	if !ok {
		return nil, &notFoundError{id: id}
	}
	cp := *doc
	return &cp, nil
}

func (m *mockInvoiceDocumentRepo) ListByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.InvoiceDocument, error) {
	var result []domain.InvoiceDocument
	for _, d := range m.docs {
		if d.InvoiceID == invoiceID {
			result = append(result, *d)
		}
	}
	return result, nil
}

func (m *mockInvoiceDocumentRepo) Delete(ctx context.Context, id int64) error {
	if _, ok := m.docs[id]; !ok {
		return &notFoundError{id: id}
	}
	delete(m.docs, id)
	return nil
}

func (m *mockInvoiceDocumentRepo) CountByInvoiceID(ctx context.Context, invoiceID int64) (int, error) {
	count := 0
	for _, d := range m.docs {
		if d.InvoiceID == invoiceID {
			count++
		}
	}
	return count, nil
}

// newInvoiceDocumentTestService creates an InvoiceDocumentService with a temp dataDir and mock repo.
func newInvoiceDocumentTestService(t *testing.T) (*InvoiceDocumentService, *mockInvoiceDocumentRepo) {
	t.Helper()
	repo := newMockInvoiceDocumentRepo()
	dataDir := t.TempDir()
	svc := NewInvoiceDocumentService(repo, dataDir, nil)
	return svc, repo
}

func TestInvoiceDocumentService_Upload_ValidPDF(t *testing.T) {
	svc, _ := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	doc, err := svc.Upload(ctx, 1, "receipt.pdf", "application/pdf", pdfMagic)
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}
	if doc.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if doc.Filename != "receipt.pdf" {
		t.Errorf("Filename = %q, want %q", doc.Filename, "receipt.pdf")
	}
	if doc.Size != int64(len(pdfMagic)) {
		t.Errorf("Size = %d, want %d", doc.Size, len(pdfMagic))
	}
	if doc.ContentType != "application/pdf" {
		t.Errorf("ContentType = %q, want application/pdf", doc.ContentType)
	}
}

func TestInvoiceDocumentService_Upload_ValidImage(t *testing.T) {
	svc, _ := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	doc, err := svc.Upload(ctx, 1, "photo.jpg", "image/jpeg", jpegMagic)
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}
	if doc.ContentType != "image/jpeg" {
		t.Errorf("ContentType = %q, want image/jpeg", doc.ContentType)
	}
}

func TestInvoiceDocumentService_Upload_ZeroInvoiceID(t *testing.T) {
	svc, _ := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	_, err := svc.Upload(ctx, 0, "file.pdf", "application/pdf", pdfMagic)
	if err == nil {
		t.Error("expected error for zero invoice ID")
	}
}

func TestInvoiceDocumentService_Upload_DotFilename(t *testing.T) {
	svc, _ := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	// sanitizeFilename(".") returns "." which is a valid (if odd) filename.
	// filepath.Base("") returns "." so empty string becomes ".".
	doc, err := svc.Upload(ctx, 1, ".", "application/pdf", pdfMagic)
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}
	if doc.Filename != "." {
		t.Errorf("Filename = %q, want %q", doc.Filename, ".")
	}
}

func TestInvoiceDocumentService_Upload_SanitizesFilename(t *testing.T) {
	svc, _ := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	doc, err := svc.Upload(ctx, 1, "../../etc/passwd", "application/pdf", pdfMagic)
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}
	if doc.Filename == "../../etc/passwd" {
		t.Error("filename was not sanitized")
	}
	if doc.Filename != "passwd" {
		t.Errorf("sanitized filename = %q, want %q", doc.Filename, "passwd")
	}
}

func TestInvoiceDocumentService_Upload_TooLarge(t *testing.T) {
	svc, _ := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	oversized := make([]byte, maxDocumentSize+1)
	_, err := svc.Upload(ctx, 1, "huge.pdf", "application/pdf", oversized)
	if err == nil {
		t.Error("expected error for file exceeding size limit")
	}
}

func TestInvoiceDocumentService_Upload_TooManyDocuments(t *testing.T) {
	svc, repo := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	// Pre-populate the repo with maxDocsPerInvoice documents.
	for i := 0; i < maxDocsPerInvoice; i++ {
		repo.docs[int64(i+1)] = &domain.InvoiceDocument{
			ID:        int64(i + 1),
			InvoiceID: 42,
		}
	}
	repo.nextID = int64(maxDocsPerInvoice + 1)

	_, err := svc.Upload(ctx, 42, "extra.pdf", "application/pdf", pdfMagic)
	if err == nil {
		t.Error("expected error when document limit is reached")
	}
}

func TestInvoiceDocumentService_Upload_ContentTypeDetection(t *testing.T) {
	svc, _ := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	types := []struct {
		filename    string
		contentType string
		data        []byte
	}{
		{"file.jpg", "image/jpeg", jpegMagic},
		{"file.png", "image/png", pngMagic},
		{"file.pdf", "application/pdf", pdfMagic},
		{"file.webp", "image/webp", webpMagic},
	}

	for _, tt := range types {
		t.Run(tt.contentType, func(t *testing.T) {
			doc, err := svc.Upload(ctx, 1, tt.filename, tt.contentType, tt.data)
			if err != nil {
				t.Errorf("Upload() for %q error: %v", tt.contentType, err)
			}
			if doc != nil && doc.ContentType != tt.contentType {
				t.Errorf("ContentType = %q, want %q", doc.ContentType, tt.contentType)
			}
		})
	}
}

func TestInvoiceDocumentService_GetByID_ZeroID(t *testing.T) {
	svc, _ := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestInvoiceDocumentService_GetByID_NotFound(t *testing.T) {
	svc, _ := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent document")
	}
}

func TestInvoiceDocumentService_GetByID_Success(t *testing.T) {
	svc, _ := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	doc, err := svc.Upload(ctx, 1, "test.pdf", "application/pdf", pdfMagic)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	got, err := svc.GetByID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != doc.ID {
		t.Errorf("ID = %d, want %d", got.ID, doc.ID)
	}
	if got.Filename != "test.pdf" {
		t.Errorf("Filename = %q, want test.pdf", got.Filename)
	}
}

func TestInvoiceDocumentService_ListByInvoiceID_ZeroID(t *testing.T) {
	svc, _ := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	_, err := svc.ListByInvoiceID(ctx, 0)
	if err == nil {
		t.Error("expected error for zero invoice ID")
	}
}

func TestInvoiceDocumentService_ListByInvoiceID_Success(t *testing.T) {
	svc, _ := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	_, err := svc.Upload(ctx, 1, "test.pdf", "application/pdf", pdfMagic)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	docs, err := svc.ListByInvoiceID(ctx, 1)
	if err != nil {
		t.Fatalf("ListByInvoiceID: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("got %d docs, want 1", len(docs))
	}
}

func TestInvoiceDocumentService_ListByInvoiceID_Empty(t *testing.T) {
	svc, _ := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	docs, err := svc.ListByInvoiceID(ctx, 999)
	if err != nil {
		t.Fatalf("ListByInvoiceID: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("got %d docs, want 0", len(docs))
	}
}

func TestInvoiceDocumentService_Delete_ZeroID(t *testing.T) {
	svc, _ := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	err := svc.Delete(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestInvoiceDocumentService_Delete_RemovesFromRepo(t *testing.T) {
	svc, repo := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	doc, err := svc.Upload(ctx, 1, "todelete.pdf", "application/pdf", pdfMagic)
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	if err := svc.Delete(ctx, doc.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	if _, ok := repo.docs[doc.ID]; ok {
		t.Error("expected document to be removed from repo after delete")
	}
}

func TestInvoiceDocumentService_Delete_NotFound(t *testing.T) {
	svc, _ := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	err := svc.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent document")
	}
}

func TestInvoiceDocumentService_GetFilePath_ZeroID(t *testing.T) {
	svc, _ := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	_, _, err := svc.GetFilePath(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestInvoiceDocumentService_GetFilePath_Valid(t *testing.T) {
	svc, _ := newInvoiceDocumentTestService(t)
	ctx := context.Background()

	doc, err := svc.Upload(ctx, 1, "file.pdf", "application/pdf", pdfMagic)
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
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
