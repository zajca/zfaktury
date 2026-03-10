package service

import (
	"context"
	"fmt"
	"os"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service/ocr"
)

// ocrSupportedContentTypes defines which document types can be processed by OCR.
var ocrSupportedContentTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"application/pdf": true,
}

// OCRService handles OCR processing of expense documents.
type OCRService struct {
	provider  ocr.Provider
	documents *DocumentService
}

// NewOCRService creates a new OCRService.
func NewOCRService(provider ocr.Provider, documents *DocumentService) *OCRService {
	return &OCRService{
		provider:  provider,
		documents: documents,
	}
}

// ProcessDocument retrieves a document by ID, reads its file data,
// and sends it to the OCR provider for extraction.
func (s *OCRService) ProcessDocument(ctx context.Context, documentID int64) (*domain.OCRResult, error) {
	if documentID == 0 {
		return nil, fmt.Errorf("document ID is required")
	}

	doc, err := s.documents.GetByID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("getting document: %w", err)
	}

	if !ocrSupportedContentTypes[doc.ContentType] {
		return nil, fmt.Errorf("document content type %q is not supported for OCR; supported: image/jpeg, image/png, application/pdf", doc.ContentType)
	}

	filePath, contentType, err := s.documents.GetFilePath(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("getting document file path: %w", err)
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("document file unavailable")
	}

	result, err := s.provider.ProcessImage(ctx, fileData, contentType)
	if err != nil {
		return nil, fmt.Errorf("OCR processing failed: %w", err)
	}

	return result, nil
}
