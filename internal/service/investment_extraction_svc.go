package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service/ocr"
)

// ocrSupportedInvestmentContentTypes defines which document types can be processed for extraction.
var ocrSupportedInvestmentContentTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"application/pdf": true,
}

// InvestmentExtractionService extracts investment data from broker statements using AI.
type InvestmentExtractionService struct {
	provider     ocr.Provider
	docSvc       *InvestmentDocumentService
	capitalRepo  repository.CapitalIncomeRepo
	securityRepo repository.SecurityTransactionRepo
	docRepo      repository.InvestmentDocumentRepo
}

// NewInvestmentExtractionService creates a new InvestmentExtractionService.
func NewInvestmentExtractionService(
	provider ocr.Provider,
	docSvc *InvestmentDocumentService,
	capitalRepo repository.CapitalIncomeRepo,
	securityRepo repository.SecurityTransactionRepo,
	docRepo repository.InvestmentDocumentRepo,
) *InvestmentExtractionService {
	return &InvestmentExtractionService{
		provider:     provider,
		docSvc:       docSvc,
		capitalRepo:  capitalRepo,
		securityRepo: securityRepo,
		docRepo:      docRepo,
	}
}

// ExtractFromDocument reads a broker statement, sends it through AI, and extracts
// investment data (capital income entries and security transactions).
func (s *InvestmentExtractionService) ExtractFromDocument(ctx context.Context, documentID int64) (*domain.InvestmentExtractionResult, error) {
	if documentID == 0 {
		return nil, fmt.Errorf("document ID is required: %w", domain.ErrInvalidInput)
	}

	doc, err := s.docSvc.GetByID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("getting document: %w", err)
	}

	if !ocrSupportedInvestmentContentTypes[doc.ContentType] {
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

	// Build prompts with platform-specific hints.
	sysPrompt := ocr.InvestmentSystemPrompt()
	usrPrompt := ocr.InvestmentUserPrompt(doc.Platform)

	rawResponse, err := s.provider.ProcessWithPrompt(ctx, fileData, contentType, sysPrompt, usrPrompt)
	if err != nil {
		// Mark extraction as failed.
		_ = s.docRepo.UpdateExtraction(ctx, documentID, domain.ExtractionFailed, err.Error())
		return nil, fmt.Errorf("AI processing failed: %w", err)
	}

	parsed, err := ocr.ParseInvestmentJSON(rawResponse)
	if err != nil {
		slog.Warn("failed to parse investment AI response",
			"document_id", documentID,
			"error", err,
			"response_length", len(rawResponse),
			"response_preview", truncate(rawResponse, 1000),
		)
		_ = s.docRepo.UpdateExtraction(ctx, documentID, domain.ExtractionFailed, err.Error())
		return nil, fmt.Errorf("parsing AI response: %w", err)
	}

	result := &domain.InvestmentExtractionResult{
		Platform:    doc.Platform,
		Confidence:  parsed.Confidence,
		RawResponse: rawResponse,
	}

	// Convert and persist capital income entries.
	for _, entry := range parsed.CapitalEntries {
		incomeDate, parseErr := time.Parse("2006-01-02", entry.IncomeDate)
		if parseErr != nil {
			slog.Warn("AI returned unparseable income date, using Jan 1 fallback", "raw_date", entry.IncomeDate, "document_id", documentID)
			incomeDate = time.Date(doc.Year, 1, 1, 0, 0, 0, 0, time.UTC)
		}

		grossAmount := ocr.CzkToHalere(entry.GrossAmount)
		withheldTaxCZ := ocr.CzkToHalere(entry.WithheldTaxCZ)
		withheldTaxForeign := ocr.CzkToHalere(entry.WithheldTaxForeign)
		netAmount := grossAmount - withheldTaxCZ - withheldTaxForeign

		docID := doc.ID
		capitalEntry := &domain.CapitalIncomeEntry{
			Year:               doc.Year,
			DocumentID:         &docID,
			Category:           entry.Category,
			Description:        entry.Description,
			IncomeDate:         incomeDate,
			GrossAmount:        domain.Amount(grossAmount),
			WithheldTaxCZ:      domain.Amount(withheldTaxCZ),
			WithheldTaxForeign: domain.Amount(withheldTaxForeign),
			CountryCode:        entry.CountryCode,
			NeedsDeclaring:     entry.NeedsDeclaring,
			NetAmount:          domain.Amount(netAmount),
		}

		if err := s.capitalRepo.Create(ctx, capitalEntry); err != nil {
			_ = s.docRepo.UpdateExtraction(ctx, documentID, domain.ExtractionFailed, fmt.Sprintf("creating capital entry: %v", err))
			return nil, fmt.Errorf("creating capital income entry: %w", err)
		}

		result.CapitalEntries = append(result.CapitalEntries, *capitalEntry)
	}

	// Convert and persist security transactions.
	for _, tx := range parsed.Transactions {
		txDate, parseErr := time.Parse("2006-01-02", tx.TransactionDate)
		if parseErr != nil {
			slog.Warn("AI returned unparseable transaction date, using Jan 1 fallback", "raw_date", tx.TransactionDate, "document_id", documentID)
			txDate = time.Date(doc.Year, 1, 1, 0, 0, 0, 0, time.UTC)
		}

		docID := doc.ID
		secTx := &domain.SecurityTransaction{
			Year:            doc.Year,
			DocumentID:      &docID,
			AssetType:       tx.AssetType,
			AssetName:       tx.AssetName,
			ISIN:            tx.ISIN,
			TransactionType: tx.TransactionType,
			TransactionDate: txDate,
			Quantity:        ocr.QuantityToInt(tx.Quantity),
			UnitPrice:       domain.Amount(ocr.CzkToHalere(tx.UnitPrice)),
			TotalAmount:     domain.Amount(ocr.CzkToHalere(tx.TotalAmount)),
			Fees:            domain.Amount(ocr.CzkToHalere(tx.Fees)),
			CurrencyCode:    tx.CurrencyCode,
			ExchangeRate:    ocr.ExchangeRateToInt(tx.ExchangeRate),
		}

		if err := s.securityRepo.Create(ctx, secTx); err != nil {
			_ = s.docRepo.UpdateExtraction(ctx, documentID, domain.ExtractionFailed, fmt.Sprintf("creating transaction: %v", err))
			return nil, fmt.Errorf("creating security transaction: %w", err)
		}

		result.Transactions = append(result.Transactions, *secTx)
	}

	// Mark extraction as successful.
	if err := s.docRepo.UpdateExtraction(ctx, documentID, domain.ExtractionExtracted, ""); err != nil {
		return nil, fmt.Errorf("updating document extraction status: %w", err)
	}

	return result, nil
}
