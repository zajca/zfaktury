package service

import (
	"context"
	"fmt"
	"os"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service/ocr"
)

// ocrSupportedTaxContentTypes defines which document types can be processed for extraction.
var ocrSupportedTaxContentTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"application/pdf": true,
}

// TaxDocumentExtractionService extracts amounts from tax deduction proof documents using AI/OCR.
type TaxDocumentExtractionService struct {
	provider   ocr.Provider
	docSvc     *TaxDeductionDocumentService
	deductRepo repository.TaxDeductionRepo
	docRepo    repository.TaxDeductionDocumentRepo
}

// NewTaxDocumentExtractionService creates a new TaxDocumentExtractionService.
func NewTaxDocumentExtractionService(
	provider ocr.Provider,
	docSvc *TaxDeductionDocumentService,
	deductRepo repository.TaxDeductionRepo,
	docRepo repository.TaxDeductionDocumentRepo,
) *TaxDocumentExtractionService {
	return &TaxDocumentExtractionService{
		provider:   provider,
		docSvc:     docSvc,
		deductRepo: deductRepo,
		docRepo:    docRepo,
	}
}

// ExtractAmount reads a tax deduction document, sends it through OCR, and extracts
// the monetary amount. The extraction result is persisted on the document record.
func (s *TaxDocumentExtractionService) ExtractAmount(ctx context.Context, documentID int64) (*domain.TaxExtractionResult, error) {
	if documentID == 0 {
		return nil, fmt.Errorf("document ID is required")
	}

	doc, err := s.docSvc.GetByID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("getting document: %w", err)
	}

	if !ocrSupportedTaxContentTypes[doc.ContentType] {
		return nil, fmt.Errorf("document content type %q is not supported for extraction; supported: image/jpeg, image/png, application/pdf", doc.ContentType)
	}

	// Fetch the parent deduction to get the year.
	deduction, err := s.deductRepo.GetByID(ctx, doc.TaxDeductionID)
	if err != nil {
		return nil, fmt.Errorf("fetching deduction: %w", err)
	}

	filePath, contentType, err := s.docSvc.GetFilePath(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("getting document file path: %w", err)
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("document file unavailable")
	}

	ocrResult, err := s.provider.ProcessImage(ctx, fileData, contentType)
	if err != nil {
		return nil, fmt.Errorf("OCR processing failed: %w", err)
	}

	// Build extraction result from OCR output.
	// TotalAmount is in halere (domain.Amount), convert to CZK (whole crowns).
	var amountCZK int
	var confidence float64
	if ocrResult.TotalAmount != 0 {
		amountCZK = int(ocrResult.TotalAmount / 100)
		confidence = ocrResult.Confidence
		if confidence == 0 {
			confidence = 0.8
		}
	}

	result := &domain.TaxExtractionResult{
		AmountCZK:  amountCZK,
		Year:       deduction.Year,
		Confidence: confidence,
	}

	// Persist extraction data on the document record.
	if err := s.docRepo.UpdateExtraction(ctx, documentID, ocrResult.TotalAmount, confidence); err != nil {
		return nil, fmt.Errorf("updating document extraction: %w", err)
	}

	return result, nil
}
