package service

import (
	"context"
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

// validPlatforms is the set of accepted investment platform identifiers.
var validPlatforms = map[string]bool{
	domain.PlatformPortu:      true,
	domain.PlatformZonky:      true,
	domain.PlatformTrading212: true,
	domain.PlatformRevolut:    true,
	domain.PlatformOther:      true,
}

// InvestmentDocumentService handles business logic for investment document management.
type InvestmentDocumentService struct {
	repo         repository.InvestmentDocumentRepo
	capitalRepo  repository.CapitalIncomeRepo
	securityRepo repository.SecurityTransactionRepo
	dataDir      string
	audit        *AuditService
}

// NewInvestmentDocumentService creates a new InvestmentDocumentService.
func NewInvestmentDocumentService(
	repo repository.InvestmentDocumentRepo,
	capitalRepo repository.CapitalIncomeRepo,
	securityRepo repository.SecurityTransactionRepo,
	dataDir string,
	audit *AuditService,
) *InvestmentDocumentService {
	return &InvestmentDocumentService{
		repo:         repo,
		capitalRepo:  capitalRepo,
		securityRepo: securityRepo,
		dataDir:      dataDir,
		audit:        audit,
	}
}

// Upload validates, stores, and registers a new investment document (broker statement).
// data is read fully to enforce the size limit before writing to disk.
func (s *InvestmentDocumentService) Upload(ctx context.Context, year int, platform string, filename string, contentType string, data io.Reader) (*domain.InvestmentDocument, error) {
	// Validate year.
	if year < 2000 || year > 2100 {
		return nil, fmt.Errorf("year must be between 2000 and 2100, got %d", year)
	}

	// Validate platform.
	if !validPlatforms[platform] {
		return nil, fmt.Errorf("platform %q is not valid; allowed: portu, zonky, trading212, revolut, other", platform)
	}

	// Validate content type.
	if !allowedContentTypes[contentType] {
		return nil, fmt.Errorf("content type %q is not allowed; allowed types: image/jpeg, image/png, application/pdf, image/webp, image/heic", contentType)
	}

	// Sanitize filename: strip path separators, limit length.
	filename = sanitizeFilename(filename)
	if filename == "" {
		return nil, fmt.Errorf("filename is required: %w", domain.ErrInvalidInput)
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

	// Build storage path: {dataDir}/investment-documents/{year}/{uuid}_{filename}
	storageDir := filepath.Join(s.dataDir, "investment-documents", fmt.Sprintf("%d", year))
	if err := os.MkdirAll(storageDir, 0750); err != nil {
		return nil, fmt.Errorf("creating document storage directory: %w", err)
	}

	storageName := uuid.New().String() + "_" + filename
	storagePath := filepath.Join(storageDir, storageName)

	if err := os.WriteFile(storagePath, fileBytes, 0640); err != nil {
		return nil, fmt.Errorf("writing document to disk: %w", err)
	}

	doc := &domain.InvestmentDocument{
		Year:             year,
		Platform:         platform,
		Filename:         filename,
		ContentType:      contentType,
		StoragePath:      storagePath,
		Size:             int64(len(fileBytes)),
		ExtractionStatus: domain.ExtractionPending,
	}

	if err := s.repo.Create(ctx, doc); err != nil {
		// Clean up file on DB failure.
		_ = os.Remove(storagePath)
		return nil, fmt.Errorf("saving document record: %w", err)
	}

	if s.audit != nil {
		meta := map[string]any{
			"id":           doc.ID,
			"year":         doc.Year,
			"platform":     doc.Platform,
			"filename":     doc.Filename,
			"content_type": doc.ContentType,
		}
		s.audit.Log(ctx, "investment_document", doc.ID, "create", nil, meta)
	}

	return doc, nil
}

// GetByID retrieves an investment document's metadata by its ID.
func (s *InvestmentDocumentService) GetByID(ctx context.Context, id int64) (*domain.InvestmentDocument, error) {
	if id == 0 {
		return nil, fmt.Errorf("document ID is required: %w", domain.ErrInvalidInput)
	}
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching document: %w", err)
	}
	return doc, nil
}

// ListByYear retrieves all active investment documents for a given year.
func (s *InvestmentDocumentService) ListByYear(ctx context.Context, year int) ([]domain.InvestmentDocument, error) {
	if year < 2000 || year > 2100 {
		return nil, fmt.Errorf("year must be between 2000 and 2100, got %d", year)
	}
	docs, err := s.repo.ListByYear(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("listing documents for year: %w", err)
	}
	return docs, nil
}

// Delete removes an investment document and all linked capital income entries and security transactions.
func (s *InvestmentDocumentService) Delete(ctx context.Context, id int64) error {
	if id == 0 {
		return fmt.Errorf("document ID is required: %w", domain.ErrInvalidInput)
	}

	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("getting document before delete: %w", err)
	}

	// Delete linked capital income entries.
	if err := s.capitalRepo.DeleteByDocumentID(ctx, id); err != nil {
		return fmt.Errorf("deleting linked capital income entries: %w", err)
	}

	// Delete linked security transactions.
	if err := s.securityRepo.DeleteByDocumentID(ctx, id); err != nil {
		return fmt.Errorf("deleting linked security transactions: %w", err)
	}

	// Delete the document record.
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("soft-deleting document record: %w", err)
	}

	// Best-effort removal; do not fail if file is already gone.
	if doc.StoragePath != "" {
		_ = os.Remove(doc.StoragePath)
	}

	if s.audit != nil {
		s.audit.Log(ctx, "investment_document", id, "delete", nil, nil)
	}

	return nil
}

// GetFilePath returns the filesystem path and content type for serving a document.
// It validates that the stored path is within the expected data directory.
func (s *InvestmentDocumentService) GetFilePath(ctx context.Context, id int64) (string, string, error) {
	if id == 0 {
		return "", "", fmt.Errorf("document ID is required: %w", domain.ErrInvalidInput)
	}
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", "", fmt.Errorf("fetching document for file path: %w", err)
	}

	// Validate the stored path is within our data directory to prevent path
	// traversal AND symlink escape (EvalSymlinks resolves the whole chain).
	expectedPrefix := filepath.Join(s.dataDir, "investment-documents") + string(filepath.Separator)
	absPath, err := filepath.EvalSymlinks(doc.StoragePath)
	if err != nil {
		return "", "", fmt.Errorf("invalid storage path: %w", err)
	}
	if !strings.HasPrefix(absPath, expectedPrefix) {
		return "", "", fmt.Errorf("document storage path is outside allowed directory: %w", domain.ErrInvalidInput)
	}

	return absPath, doc.ContentType, nil
}
