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

// allowedContentTypes is the set of MIME types accepted for document uploads.
var allowedContentTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"application/pdf": true,
	"image/webp":      true,
	"image/heic":      true,
}

const (
	maxDocumentSize  = 20 << 20 // 20 MB
	maxDocsPerExpense = 10
	maxFilenameLen   = 255
)

// DocumentService handles business logic for expense document management.
type DocumentService struct {
	repo    repository.DocumentRepo
	dataDir string
}

// NewDocumentService creates a new DocumentService.
func NewDocumentService(repo repository.DocumentRepo, dataDir string) *DocumentService {
	return &DocumentService{repo: repo, dataDir: dataDir}
}

// Upload validates, stores, and registers a new document for an expense.
// data is read fully to enforce the size limit before writing to disk.
func (s *DocumentService) Upload(ctx context.Context, expenseID int64, filename string, contentType string, data io.Reader) (*domain.ExpenseDocument, error) {
	if expenseID == 0 {
		return nil, errors.New("expense ID is required")
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

	// Check per-expense document limit.
	count, err := s.repo.CountByExpenseID(ctx, expenseID)
	if err != nil {
		return nil, fmt.Errorf("checking document count: %w", err)
	}
	if count >= maxDocsPerExpense {
		return nil, fmt.Errorf("maximum of %d documents per expense reached", maxDocsPerExpense)
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

	// Build storage path: {dataDir}/documents/{expenseID}/{uuid}_{filename}
	storageDir := filepath.Join(s.dataDir, "documents", fmt.Sprintf("%d", expenseID))
	if err := os.MkdirAll(storageDir, 0750); err != nil {
		return nil, fmt.Errorf("creating document storage directory: %w", err)
	}

	storageName := uuid.New().String() + "_" + filename
	storagePath := filepath.Join(storageDir, storageName)

	if err := os.WriteFile(storagePath, fileBytes, 0640); err != nil {
		return nil, fmt.Errorf("writing document to disk: %w", err)
	}

	doc := &domain.ExpenseDocument{
		ExpenseID:   expenseID,
		Filename:    filename,
		ContentType: contentType,
		StoragePath: storagePath,
		Size:        int64(len(fileBytes)),
	}

	if err := s.repo.Create(ctx, doc); err != nil {
		// Clean up file on DB failure.
		_ = os.Remove(storagePath)
		return nil, fmt.Errorf("saving document record: %w", err)
	}

	return doc, nil
}

// GetByID retrieves a document's metadata by its ID.
func (s *DocumentService) GetByID(ctx context.Context, id int64) (*domain.ExpenseDocument, error) {
	if id == 0 {
		return nil, errors.New("document ID is required")
	}
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching document: %w", err)
	}
	return doc, nil
}

// ListByExpenseID retrieves all active documents for an expense.
func (s *DocumentService) ListByExpenseID(ctx context.Context, expenseID int64) ([]domain.ExpenseDocument, error) {
	if expenseID == 0 {
		return nil, errors.New("expense ID is required")
	}
	docs, err := s.repo.ListByExpenseID(ctx, expenseID)
	if err != nil {
		return nil, fmt.Errorf("listing documents for expense: %w", err)
	}
	return docs, nil
}

// Delete soft-deletes the document record and removes the file from disk.
func (s *DocumentService) Delete(ctx context.Context, id int64) error {
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
func (s *DocumentService) GetFilePath(ctx context.Context, id int64) (string, string, error) {
	if id == 0 {
		return "", "", errors.New("document ID is required")
	}
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", "", fmt.Errorf("fetching document for file path: %w", err)
	}

	// Validate the stored path is within our data directory to prevent path traversal.
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

// detectByMagicBytes checks file magic bytes for types that http.DetectContentType
// cannot identify. Returns the detected type or "application/octet-stream".
func detectByMagicBytes(data []byte, declaredType string) string {
	if len(data) >= 5 && string(data[:5]) == "%PDF-" {
		return "application/pdf"
	}
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	// HEIC/HEIF: ftyp box at offset 4 with brands heic, heix, mif1.
	if len(data) >= 12 && string(data[4:8]) == "ftyp" {
		brand := string(data[8:12])
		if brand == "heic" || brand == "heix" || brand == "mif1" {
			return "image/heic"
		}
	}
	return "application/octet-stream"
}

// sanitizeFilename removes path separators, null bytes, and limits the filename length.
func sanitizeFilename(name string) string {
	// Remove null bytes and directory traversal characters.
	name = strings.ReplaceAll(name, "\x00", "")
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "/", "")
	name = strings.ReplaceAll(name, "\\", "")
	name = strings.TrimSpace(name)

	if len(name) > maxFilenameLen {
		ext := filepath.Ext(name)
		base := name[:maxFilenameLen-len(ext)]
		name = base + ext
	}

	return name
}
