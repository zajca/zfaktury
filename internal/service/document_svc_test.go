package service

import (
	"bytes"
	"context"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
)

// mockDocumentRepo is a minimal in-memory DocumentRepo for unit tests.
type mockDocumentRepo struct {
	docs   map[int64]*domain.ExpenseDocument
	nextID int64
}

func newMockDocumentRepo() *mockDocumentRepo {
	return &mockDocumentRepo{
		docs:   make(map[int64]*domain.ExpenseDocument),
		nextID: 1,
	}
}

func (m *mockDocumentRepo) Create(ctx context.Context, doc *domain.ExpenseDocument) error {
	doc.ID = m.nextID
	m.nextID++
	cp := *doc
	m.docs[doc.ID] = &cp
	return nil
}

func (m *mockDocumentRepo) GetByID(ctx context.Context, id int64) (*domain.ExpenseDocument, error) {
	doc, ok := m.docs[id]
	if !ok {
		return nil, errNotFound(id)
	}
	cp := *doc
	return &cp, nil
}

func (m *mockDocumentRepo) ListByExpenseID(ctx context.Context, expenseID int64) ([]domain.ExpenseDocument, error) {
	var result []domain.ExpenseDocument
	for _, d := range m.docs {
		if d.ExpenseID == expenseID {
			result = append(result, *d)
		}
	}
	return result, nil
}

func (m *mockDocumentRepo) Delete(ctx context.Context, id int64) error {
	if _, ok := m.docs[id]; !ok {
		return errNotFound(id)
	}
	delete(m.docs, id)
	return nil
}

func (m *mockDocumentRepo) CountByExpenseID(ctx context.Context, expenseID int64) (int, error) {
	count := 0
	for _, d := range m.docs {
		if d.ExpenseID == expenseID {
			count++
		}
	}
	return count, nil
}

func errNotFound(id int64) error {
	return &notFoundError{id: id}
}

type notFoundError struct{ id int64 }

func (e *notFoundError) Error() string {
	return "document not found"
}

// Test data with proper magic bytes for content type detection.
var (
	// Minimal JPEG: SOI marker + padding.
	jpegMagic = append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x00}, 508)...)
	// Minimal PNG: 8-byte signature + padding.
	pngMagic = append([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, bytes.Repeat([]byte{0x00}, 504)...)
	// Minimal PDF: magic header.
	pdfMagic = append([]byte("%PDF-1.4 "), bytes.Repeat([]byte{0x00}, 503)...)
	// WebP: RIFF + size + WEBP.
	webpMagic = func() []byte {
		b := make([]byte, 512)
		copy(b, []byte("RIFF"))
		copy(b[8:], []byte("WEBP"))
		return b
	}()
)

// newDocumentTestService creates a DocumentService with a temp dataDir and mock repo.
func newDocumentTestService(t *testing.T) (*DocumentService, *mockDocumentRepo) {
	t.Helper()
	repo := newMockDocumentRepo()
	dataDir := t.TempDir()
	svc := NewDocumentService(repo, dataDir)
	return svc, repo
}

func TestDocumentService_Upload_ValidPDF(t *testing.T) {
	svc, _ := newDocumentTestService(t)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	doc, err := svc.Upload(ctx, 1, "receipt.pdf", "application/pdf", data)
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
}

func TestDocumentService_Upload_ValidImage(t *testing.T) {
	svc, _ := newDocumentTestService(t)
	ctx := context.Background()

	data := bytes.NewReader(jpegMagic)
	doc, err := svc.Upload(ctx, 1, "photo.jpg", "image/jpeg", data)
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}
	if doc.ContentType != "image/jpeg" {
		t.Errorf("ContentType = %q, want image/jpeg", doc.ContentType)
	}
}

func TestDocumentService_Upload_InvalidContentType(t *testing.T) {
	svc, _ := newDocumentTestService(t)
	ctx := context.Background()

	data := bytes.NewReader([]byte("not a valid file"))
	_, err := svc.Upload(ctx, 1, "file.exe", "application/octet-stream", data)
	if err == nil {
		t.Error("expected error for disallowed content type")
	}
}

func TestDocumentService_Upload_TooLarge(t *testing.T) {
	svc, _ := newDocumentTestService(t)
	ctx := context.Background()

	oversized := bytes.NewReader(bytes.Repeat([]byte("a"), maxDocumentSize+1))
	_, err := svc.Upload(ctx, 1, "huge.pdf", "application/pdf", oversized)
	if err == nil {
		t.Error("expected error for file exceeding size limit")
	}
}

func TestDocumentService_Upload_TooManyDocuments(t *testing.T) {
	svc, repo := newDocumentTestService(t)
	ctx := context.Background()

	// Pre-populate the repo with maxDocsPerExpense documents.
	for i := 0; i < maxDocsPerExpense; i++ {
		repo.docs[int64(i+1)] = &domain.ExpenseDocument{
			ID:        int64(i + 1),
			ExpenseID: 42,
		}
	}
	repo.nextID = int64(maxDocsPerExpense + 1)

	data := bytes.NewReader(pdfMagic)
	_, err := svc.Upload(ctx, 42, "extra.pdf", "application/pdf", data)
	if err == nil {
		t.Error("expected error when document limit is reached")
	}
}

func TestDocumentService_Upload_ZeroExpenseID(t *testing.T) {
	svc, _ := newDocumentTestService(t)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	_, err := svc.Upload(ctx, 0, "file.pdf", "application/pdf", data)
	if err == nil {
		t.Error("expected error for zero expense ID")
	}
}

func TestDocumentService_Upload_SanitizesFilename(t *testing.T) {
	svc, _ := newDocumentTestService(t)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	doc, err := svc.Upload(ctx, 1, "../../etc/passwd", "application/pdf", data)
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

func TestDocumentService_GetByID_ZeroID(t *testing.T) {
	svc, _ := newDocumentTestService(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestDocumentService_GetByID_NotFound(t *testing.T) {
	svc, _ := newDocumentTestService(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent document")
	}
}

func TestDocumentService_ListByExpenseID_ZeroID(t *testing.T) {
	svc, _ := newDocumentTestService(t)
	ctx := context.Background()

	_, err := svc.ListByExpenseID(ctx, 0)
	if err == nil {
		t.Error("expected error for zero expense ID")
	}
}

func TestDocumentService_ListByExpenseID_Success(t *testing.T) {
	svc, _ := newDocumentTestService(t)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	_, err := svc.Upload(ctx, 1, "test.pdf", "application/pdf", data)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	docs, err := svc.ListByExpenseID(ctx, 1)
	if err != nil {
		t.Fatalf("ListByExpenseID: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("got %d docs, want 1", len(docs))
	}
}

func TestDocumentService_ListByExpenseID_Empty(t *testing.T) {
	svc, _ := newDocumentTestService(t)
	ctx := context.Background()

	docs, err := svc.ListByExpenseID(ctx, 999)
	if err != nil {
		t.Fatalf("ListByExpenseID: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("got %d docs, want 0", len(docs))
	}
}

func TestDocumentService_Delete_ZeroID(t *testing.T) {
	svc, _ := newDocumentTestService(t)
	ctx := context.Background()

	err := svc.Delete(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestDocumentService_Delete_RemovesFromRepo(t *testing.T) {
	svc, repo := newDocumentTestService(t)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	doc, err := svc.Upload(ctx, 1, "todelete.pdf", "application/pdf", data)
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

func TestDocumentService_GetFilePath_ZeroID(t *testing.T) {
	svc, _ := newDocumentTestService(t)
	ctx := context.Background()

	_, _, err := svc.GetFilePath(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
}

func TestDocumentService_GetFilePath_Valid(t *testing.T) {
	svc, _ := newDocumentTestService(t)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	doc, err := svc.Upload(ctx, 1, "file.pdf", "application/pdf", data)
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

func TestDocumentService_AllowedContentTypes(t *testing.T) {
	svc, _ := newDocumentTestService(t)
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
		// heic is detected as application/octet-stream by Go, skip in content detection test
	}

	for _, tt := range types {
		t.Run(tt.contentType, func(t *testing.T) {
			data := bytes.NewReader(tt.data)
			_, err := svc.Upload(ctx, 1, tt.filename, tt.contentType, data)
			if err != nil {
				t.Errorf("Upload() for %q error: %v", tt.contentType, err)
			}
		})
	}
}
