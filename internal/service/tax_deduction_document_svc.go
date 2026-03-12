package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// TaxDeductionDocumentService handles business logic for tax deduction document management.
type TaxDeductionDocumentService struct {
	repo       repository.TaxDeductionDocumentRepo
	deductRepo repository.TaxDeductionRepo
	dataDir    string
}

// NewTaxDeductionDocumentService creates a new TaxDeductionDocumentService.
func NewTaxDeductionDocumentService(
	repo repository.TaxDeductionDocumentRepo,
	deductRepo repository.TaxDeductionRepo,
	dataDir string,
) *TaxDeductionDocumentService {
	return &TaxDeductionDocumentService{
		repo:       repo,
		deductRepo: deductRepo,
		dataDir:    dataDir,
	}
}

// Upload validates, stores, and registers a new proof document for a tax deduction.
// data is read fully to enforce the size limit before writing to disk.
func (s *TaxDeductionDocumentService) Upload(ctx context.Context, deductionID int64, filename string, contentType string, data io.Reader) (*domain.TaxDeductionDocument, error) {
	if deductionID == 0 {
		return nil, errors.New("deduction ID is required")
	}

	// Validate the deduction exists.
	if _, err := s.deductRepo.GetByID(ctx, deductionID); err != nil {
		return nil, fmt.Errorf("fetching deduction: %w", err)
	}

	// Validate content type.
	if !allowedContentTypes[contentType] {
		return nil, fmt.Errorf("content type %q is not allowed; allowed types: image/jpeg, image/png, application/pdf, image/webp, image/heic", contentType)
	}

	// Sanitize filename: strip path separators, limit length.
	filename = sanitizeFilename(filename)
	if filename == "" {
		return nil, errors.New("filename is required")
	}

	// Read file data, enforcing the size limit (read one extra byte to detect overflow).
	limited := io.LimitReader(data, maxDocumentSize+1)
	fileBytes, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("reading uploaded file: %w", err)
	}
	if int64(len(fileBytes)) > maxDocumentSize {
		return nil, fmt.Errorf("file size exceeds maximum of %d MB", maxDocumentSize>>20)
	}

	// Detect actual content type from file bytes to prevent MIME spoofing.
	detectedType := http.DetectContentType(fileBytes)
	// Normalize detected type (may include params like charset).
	if idx := strings.IndexByte(detectedType, ';'); idx != -1 {
		detectedType = strings.TrimSpace(detectedType[:idx])
	}
	// http.DetectContentType has limited signatures. For types it can't detect,
	// verify magic bytes manually and fall back to the declared content type.
	if detectedType == "application/octet-stream" {
		detectedType = detectByMagicBytes(fileBytes, contentType)
	}
	if !allowedContentTypes[detectedType] {
		return nil, fmt.Errorf("detected content type %q is not allowed", detectedType)
	}
	// Use detected type as the canonical type.
	contentType = detectedType

	// Build storage path: {dataDir}/tax-documents/{deductionID}/{uuid}_{filename}
	storageDir := filepath.Join(s.dataDir, "tax-documents", fmt.Sprintf("%d", deductionID))
	if err := os.MkdirAll(storageDir, 0750); err != nil {
		return nil, fmt.Errorf("creating document storage directory: %w", err)
	}

	storageName := uuid.New().String() + "_" + filename
	storagePath := filepath.Join(storageDir, storageName)

	if err := os.WriteFile(storagePath, fileBytes, 0640); err != nil {
		return nil, fmt.Errorf("writing document to disk: %w", err)
	}

	doc := &domain.TaxDeductionDocument{
		TaxDeductionID: deductionID,
		Filename:       filename,
		ContentType:    contentType,
		StoragePath:    storagePath,
		Size:           int64(len(fileBytes)),
	}

	if err := s.repo.Create(ctx, doc); err != nil {
		// Clean up file on DB failure.
		_ = os.Remove(storagePath)
		return nil, fmt.Errorf("saving document record: %w", err)
	}

	return doc, nil
}

// GetByID retrieves a tax deduction document's metadata by its ID.
func (s *TaxDeductionDocumentService) GetByID(ctx context.Context, id int64) (*domain.TaxDeductionDocument, error) {
	if id == 0 {
		return nil, errors.New("document ID is required")
	}
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching document: %w", err)
	}
	return doc, nil
}

// ListByDeductionID retrieves all active documents for a tax deduction.
func (s *TaxDeductionDocumentService) ListByDeductionID(ctx context.Context, deductionID int64) ([]domain.TaxDeductionDocument, error) {
	if deductionID == 0 {
		return nil, errors.New("deduction ID is required")
	}
	docs, err := s.repo.ListByDeductionID(ctx, deductionID)
	if err != nil {
		return nil, fmt.Errorf("listing documents for deduction: %w", err)
	}
	return docs, nil
}

// Delete soft-deletes the document record and removes the file from disk.
func (s *TaxDeductionDocumentService) Delete(ctx context.Context, id int64) error {
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

	// Best-effort removal; do not fail if file is already gone.
	if doc.StoragePath != "" {
		_ = os.Remove(doc.StoragePath)
	}

	return nil
}

// GetFilePath returns the filesystem path and content type for serving a document.
// It validates that the stored path is within the expected data directory.
func (s *TaxDeductionDocumentService) GetFilePath(ctx context.Context, id int64) (string, string, error) {
	if id == 0 {
		return "", "", errors.New("document ID is required")
	}
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", "", fmt.Errorf("fetching document for file path: %w", err)
	}

	// Validate the stored path is within our data directory to prevent path traversal.
	expectedPrefix := filepath.Join(s.dataDir, "tax-documents") + string(filepath.Separator)
	absPath, err := filepath.Abs(doc.StoragePath)
	if err != nil {
		return "", "", fmt.Errorf("invalid storage path: %w", err)
	}
	if !strings.HasPrefix(absPath, expectedPrefix) {
		return "", "", errors.New("document storage path is outside allowed directory")
	}

	return absPath, doc.ContentType, nil
}
