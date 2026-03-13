package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

const maxDocsPerInvoice = 10

// InvoiceDocumentService handles business logic for invoice document management.
type InvoiceDocumentService struct {
	repo    repository.InvoiceDocumentRepo
	dataDir string
	audit   *AuditService
}

// NewInvoiceDocumentService creates a new InvoiceDocumentService.
func NewInvoiceDocumentService(repo repository.InvoiceDocumentRepo, dataDir string, audit *AuditService) *InvoiceDocumentService {
	return &InvoiceDocumentService{repo: repo, dataDir: dataDir, audit: audit}
}

// Upload validates, stores, and registers a new document for an invoice.
func (s *InvoiceDocumentService) Upload(ctx context.Context, invoiceID int64, filename string, contentType string, data []byte) (*domain.InvoiceDocument, error) {
	if invoiceID == 0 {
		return nil, errors.New("invoice ID is required")
	}

	filename = sanitizeFilename(filename)
	if filename == "" {
		return nil, errors.New("filename is required")
	}

	// Check per-invoice document limit.
	count, err := s.repo.CountByInvoiceID(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("checking document count: %w", err)
	}
	if count >= maxDocsPerInvoice {
		return nil, fmt.Errorf("maximum of %d documents per invoice reached", maxDocsPerInvoice)
	}

	if int64(len(data)) > maxDocumentSize {
		return nil, fmt.Errorf("file size exceeds maximum of %d MB", maxDocumentSize>>20)
	}

	// Detect content type from file bytes.
	detectedType := detectContentType(data)
	if detectedType != "" {
		contentType = detectedType
	}

	// Build storage path: {dataDir}/documents/invoices/{invoiceID}/{uuid}_{filename}
	storageDir := filepath.Join(s.dataDir, "documents", "invoices", fmt.Sprintf("%d", invoiceID))
	if err := os.MkdirAll(storageDir, 0750); err != nil {
		return nil, fmt.Errorf("creating document storage directory: %w", err)
	}

	storageName := uuid.New().String() + "_" + filename
	storagePath := filepath.Join(storageDir, storageName)

	if err := os.WriteFile(storagePath, data, 0640); err != nil {
		return nil, fmt.Errorf("writing document to disk: %w", err)
	}

	doc := &domain.InvoiceDocument{
		InvoiceID:   invoiceID,
		Filename:    filename,
		ContentType: contentType,
		StoragePath: storagePath,
		Size:        int64(len(data)),
	}

	if err := s.repo.Create(ctx, doc); err != nil {
		_ = os.Remove(storagePath)
		return nil, fmt.Errorf("saving document record: %w", err)
	}

	if s.audit != nil {
		meta := map[string]any{
			"id":           doc.ID,
			"invoice_id":   doc.InvoiceID,
			"filename":     doc.Filename,
			"content_type": doc.ContentType,
		}
		s.audit.Log(ctx, "invoice_document", doc.ID, "create", nil, meta)
	}

	return doc, nil
}

// GetByID retrieves a document's metadata by its ID.
func (s *InvoiceDocumentService) GetByID(ctx context.Context, id int64) (*domain.InvoiceDocument, error) {
	if id == 0 {
		return nil, errors.New("document ID is required")
	}
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching invoice document: %w", err)
	}
	return doc, nil
}

// ListByInvoiceID retrieves all active documents for an invoice.
func (s *InvoiceDocumentService) ListByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.InvoiceDocument, error) {
	if invoiceID == 0 {
		return nil, errors.New("invoice ID is required")
	}
	docs, err := s.repo.ListByInvoiceID(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("listing documents for invoice: %w", err)
	}
	return docs, nil
}

// Delete soft-deletes the document record and removes the file from disk.
func (s *InvoiceDocumentService) Delete(ctx context.Context, id int64) error {
	if id == 0 {
		return errors.New("document ID is required")
	}

	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("getting document before delete: %w", err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("soft-deleting document record: %w", err)
	}

	if doc.StoragePath != "" {
		_ = os.Remove(doc.StoragePath)
	}

	if s.audit != nil {
		s.audit.Log(ctx, "invoice_document", id, "delete", nil, nil)
	}

	return nil
}

// GetFilePath returns the filesystem path and content type for serving a document.
func (s *InvoiceDocumentService) GetFilePath(ctx context.Context, id int64) (string, string, error) {
	if id == 0 {
		return "", "", errors.New("document ID is required")
	}
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", "", fmt.Errorf("fetching document for file path: %w", err)
	}

	expectedPrefix := filepath.Join(s.dataDir, "documents") + string(filepath.Separator)
	absPath, err := filepath.Abs(doc.StoragePath)
	if err != nil {
		return "", "", fmt.Errorf("invalid storage path: %w", err)
	}
	if !strings.HasPrefix(absPath, expectedPrefix) {
		return "", "", errors.New("document storage path is outside allowed directory")
	}

	return absPath, doc.ContentType, nil
}

// detectContentType tries to detect a valid content type from file bytes.
// Returns empty string if detection is not confident.
func detectContentType(data []byte) string {
	if len(data) >= 5 && bytes.Equal(data[:5], []byte("%PDF-")) {
		return "application/pdf"
	}
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}
	if len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return "image/png"
	}
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	return ""
}
