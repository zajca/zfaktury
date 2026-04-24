package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

// TaxDocumentExtractionService extracts structured data from tax deduction proof
// documents using AI/OCR. It produces all fields required to pre-fill a
// TaxDeduction entry for the tax return.
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

// ExtractAmount reads a tax deduction document, sends it through AI using a
// deduction-specific prompt, and extracts structured data. The extraction
// result is persisted on the document record (amount + confidence).
//
// When the document is already linked to a TaxDeduction (deduction_id > 0) the
// deduction's year is used; otherwise the period_year reported by the AI is
// used. The caller can use the full result to create a new TaxDeduction or
// update an existing one.
func (s *TaxDocumentExtractionService) ExtractAmount(ctx context.Context, documentID int64) (*domain.TaxExtractionResult, error) {
	if documentID == 0 {
		return nil, fmt.Errorf("document ID is required: %w", domain.ErrInvalidInput)
	}

	doc, err := s.docSvc.GetByID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("getting document: %w", err)
	}

	if !ocrSupportedTaxContentTypes[doc.ContentType] {
		return nil, fmt.Errorf("document content type %q is not supported for extraction; supported: image/jpeg, image/png, application/pdf", doc.ContentType)
	}

	filePath, contentType, err := s.docSvc.GetFilePath(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("getting document file path: %w", err)
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("document file unavailable")
	}

	sysPrompt := ocr.DeductionSystemPrompt()
	usrPrompt := ocr.DeductionUserPrompt()

	rawResponse, err := s.provider.ProcessWithPrompt(ctx, fileData, contentType, sysPrompt, usrPrompt)
	if err != nil {
		return nil, fmt.Errorf("OCR processing failed: %w", err)
	}

	parsed, err := ocr.ParseDeductionJSON(rawResponse)
	if err != nil {
		return nil, fmt.Errorf("parsing deduction extraction response: %w", err)
	}

	amountHalere := domain.Amount(ocr.CzkToHalere(parsed.AmountCZK))
	amountCZK := int(amountHalere / 100)
	confidence := parsed.Confidence
	if confidence == 0 && amountCZK > 0 {
		confidence = 0.8
	}

	// Determine the tax year: prefer parent deduction's year, fall back to
	// model-reported period_year, then document creation year.
	year := parsed.PeriodYear
	if doc.TaxDeductionID > 0 {
		parent, derr := s.deductRepo.GetByID(ctx, doc.TaxDeductionID)
		if derr != nil && !errors.Is(derr, domain.ErrNotFound) {
			slog.Warn("fetching parent deduction for extraction failed; using model-reported year", "error", derr, "document_id", documentID)
		} else if parent != nil {
			year = parent.Year
		}
	}
	if year == 0 {
		year = doc.CreatedAt.Year()
	}

	result := &domain.TaxExtractionResult{
		Category:              normalizeDeductionCategory(parsed.Category),
		ProviderName:          parsed.ProviderName,
		ProviderICO:           parsed.ProviderICO,
		ContractNumber:        parsed.ContractNumber,
		DocumentDate:          parsed.DocumentDate,
		PeriodYear:            year,
		AmountCZK:             amountCZK,
		AmountHalere:          amountHalere,
		Purpose:               parsed.Purpose,
		DescriptionSuggestion: parsed.DescriptionSuggestion,
		Confidence:            confidence,
	}

	// Persist amount and confidence on the document record so the UI can show
	// the extraction badge next to the file.
	if err := s.docRepo.UpdateExtraction(ctx, documentID, amountHalere, confidence); err != nil {
		return nil, fmt.Errorf("updating document extraction: %w", err)
	}

	return result, nil
}

// normalizeDeductionCategory maps model-reported categories to domain constants,
// treating anything unknown as empty so callers can prompt the user to choose.
func normalizeDeductionCategory(raw string) string {
	switch raw {
	case domain.DeductionMortgage,
		domain.DeductionLifeInsurance,
		domain.DeductionPension,
		domain.DeductionDonation,
		domain.DeductionUnionDues:
		return raw
	default:
		return ""
	}
}
