package service

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service/ocr"
	"github.com/zajca/zfaktury/internal/testutil"
)

// investmentMockOCRProvider implements ocr.Provider for investment extraction tests.
type investmentMockOCRProvider struct {
	promptResponse string
	promptErr      error
}

func (m *investmentMockOCRProvider) ProcessImage(_ context.Context, _ []byte, _ string) (*domain.OCRResult, error) {
	return nil, nil
}

func (m *investmentMockOCRProvider) ProcessWithPrompt(_ context.Context, _ []byte, _ string, _, _ string) (string, error) {
	return m.promptResponse, m.promptErr
}

func (m *investmentMockOCRProvider) Name() string {
	return "investment-mock"
}

var _ ocr.Provider = (*investmentMockOCRProvider)(nil)

// newInvestmentExtractionTestService creates a test setup with real SQLite and a mock OCR provider.
func newInvestmentExtractionTestService(t *testing.T, provider ocr.Provider) (*InvestmentExtractionService, *InvestmentDocumentService, *repository.CapitalIncomeRepository, *repository.SecurityTransactionRepository) {
	t.Helper()
	db := testutil.NewTestDB(t)
	docRepo := repository.NewInvestmentDocumentRepository(db)
	capitalRepo := repository.NewCapitalIncomeRepository(db)
	securityRepo := repository.NewSecurityTransactionRepository(db)
	dataDir := t.TempDir()
	docSvc := NewInvestmentDocumentService(docRepo, capitalRepo, securityRepo, dataDir, nil)
	extractionSvc := NewInvestmentExtractionService(provider, docSvc, capitalRepo, securityRepo, docRepo)
	return extractionSvc, docSvc, capitalRepo, securityRepo
}

// mockInvestmentJSON is a valid AI response JSON for testing.
const mockInvestmentJSON = `{
	"platform": "portu",
	"capital_entries": [
		{
			"category": "dividend_cz",
			"description": "Dividenda z VWCE",
			"income_date": "2025-06-15",
			"gross_amount": 1000.50,
			"withheld_tax_cz": 150.07,
			"withheld_tax_foreign": 0,
			"country_code": "CZ",
			"needs_declaring": false
		}
	],
	"transactions": [
		{
			"asset_type": "etf",
			"asset_name": "VWCE",
			"isin": "IE00BK5BQT80",
			"transaction_type": "buy",
			"transaction_date": "2025-03-15",
			"quantity": 2.5,
			"unit_price": 2500.0,
			"total_amount": 6250.0,
			"fees": 15.0,
			"currency_code": "CZK",
			"exchange_rate": 1.0
		}
	],
	"confidence": 0.92
}`

func TestInvestmentExtractionService_ExtractFromDocument_Success(t *testing.T) {
	provider := &investmentMockOCRProvider{promptResponse: mockInvestmentJSON}
	extractionSvc, docSvc, _, _ := newInvestmentExtractionTestService(t, provider)
	ctx := context.Background()

	// Upload a document first.
	data := bytes.NewReader(pdfMagic)
	doc, err := docSvc.Upload(ctx, 2025, domain.PlatformPortu, "statement.pdf", "application/pdf", data)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	result, err := extractionSvc.ExtractFromDocument(ctx, doc.ID)
	if err != nil {
		t.Fatalf("ExtractFromDocument() error: %v", err)
	}

	if result.Platform != domain.PlatformPortu {
		t.Errorf("Platform = %q, want %q", result.Platform, domain.PlatformPortu)
	}
	if result.Confidence != 0.92 {
		t.Errorf("Confidence = %f, want 0.92", result.Confidence)
	}
	if len(result.CapitalEntries) != 1 {
		t.Fatalf("CapitalEntries len = %d, want 1", len(result.CapitalEntries))
	}
	if result.CapitalEntries[0].Category != domain.CapitalCategoryDividendCZ {
		t.Errorf("CapitalEntry Category = %q, want %q", result.CapitalEntries[0].Category, domain.CapitalCategoryDividendCZ)
	}
	if len(result.Transactions) != 1 {
		t.Fatalf("Transactions len = %d, want 1", len(result.Transactions))
	}
	if result.Transactions[0].AssetName != "VWCE" {
		t.Errorf("Transaction AssetName = %q, want VWCE", result.Transactions[0].AssetName)
	}

	// Verify document extraction status is updated.
	updatedDoc, err := docSvc.GetByID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetByID after extract: %v", err)
	}
	if updatedDoc.ExtractionStatus != domain.ExtractionExtracted {
		t.Errorf("ExtractionStatus = %q, want %q", updatedDoc.ExtractionStatus, domain.ExtractionExtracted)
	}
}

func TestInvestmentExtractionService_ExtractFromDocument_ZeroID(t *testing.T) {
	provider := &investmentMockOCRProvider{}
	extractionSvc, _, _, _ := newInvestmentExtractionTestService(t, provider)
	ctx := context.Background()

	_, err := extractionSvc.ExtractFromDocument(ctx, 0)
	if err == nil {
		t.Error("expected error for zero document ID")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestInvestmentExtractionService_ExtractFromDocument_NotFound(t *testing.T) {
	provider := &investmentMockOCRProvider{}
	extractionSvc, _, _, _ := newInvestmentExtractionTestService(t, provider)
	ctx := context.Background()

	_, err := extractionSvc.ExtractFromDocument(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent document")
	}
}

func TestInvestmentExtractionService_ExtractFromDocument_UnsupportedContentType(t *testing.T) {
	provider := &investmentMockOCRProvider{}
	extractionSvc, docSvc, _, _ := newInvestmentExtractionTestService(t, provider)
	ctx := context.Background()

	// Upload a WebP image (not supported for extraction).
	data := bytes.NewReader(webpMagic)
	doc, err := docSvc.Upload(ctx, 2025, domain.PlatformPortu, "file.webp", "image/webp", data)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	_, err = extractionSvc.ExtractFromDocument(ctx, doc.ID)
	if err == nil {
		t.Error("expected error for unsupported content type")
	}
}

func TestInvestmentExtractionService_ExtractFromDocument_ProviderError(t *testing.T) {
	provider := &investmentMockOCRProvider{
		promptErr: errors.New("AI service unavailable"),
	}
	extractionSvc, docSvc, _, _ := newInvestmentExtractionTestService(t, provider)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	doc, err := docSvc.Upload(ctx, 2025, domain.PlatformPortu, "statement.pdf", "application/pdf", data)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	_, err = extractionSvc.ExtractFromDocument(ctx, doc.ID)
	if err == nil {
		t.Error("expected error when provider fails")
	}

	// Verify extraction status is marked as failed.
	updatedDoc, err := docSvc.GetByID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updatedDoc.ExtractionStatus != domain.ExtractionFailed {
		t.Errorf("ExtractionStatus = %q, want %q", updatedDoc.ExtractionStatus, domain.ExtractionFailed)
	}
}

func TestInvestmentExtractionService_ExtractFromDocument_InvalidJSON(t *testing.T) {
	provider := &investmentMockOCRProvider{
		promptResponse: "this is not valid JSON at all",
	}
	extractionSvc, docSvc, _, _ := newInvestmentExtractionTestService(t, provider)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	doc, err := docSvc.Upload(ctx, 2025, domain.PlatformPortu, "statement.pdf", "application/pdf", data)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	_, err = extractionSvc.ExtractFromDocument(ctx, doc.ID)
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}

	// Verify extraction is marked failed.
	updatedDoc, err := docSvc.GetByID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updatedDoc.ExtractionStatus != domain.ExtractionFailed {
		t.Errorf("ExtractionStatus = %q, want %q", updatedDoc.ExtractionStatus, domain.ExtractionFailed)
	}
}

func TestInvestmentExtractionService_ExtractFromDocument_EmptyResult(t *testing.T) {
	emptyResp := `{"platform":"portu","capital_entries":[],"transactions":[],"confidence":0.5}`
	provider := &investmentMockOCRProvider{promptResponse: emptyResp}
	extractionSvc, docSvc, _, _ := newInvestmentExtractionTestService(t, provider)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	doc, err := docSvc.Upload(ctx, 2025, domain.PlatformPortu, "empty.pdf", "application/pdf", data)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	result, err := extractionSvc.ExtractFromDocument(ctx, doc.ID)
	if err != nil {
		t.Fatalf("ExtractFromDocument() error: %v", err)
	}

	if len(result.CapitalEntries) != 0 {
		t.Errorf("CapitalEntries len = %d, want 0", len(result.CapitalEntries))
	}
	if len(result.Transactions) != 0 {
		t.Errorf("Transactions len = %d, want 0", len(result.Transactions))
	}
	if result.Confidence != 0.5 {
		t.Errorf("Confidence = %f, want 0.5", result.Confidence)
	}
}

func TestInvestmentExtractionService_ExtractFromDocument_PersistsEntries(t *testing.T) {
	provider := &investmentMockOCRProvider{promptResponse: mockInvestmentJSON}
	extractionSvc, docSvc, capitalRepo, securityRepo := newInvestmentExtractionTestService(t, provider)
	ctx := context.Background()

	data := bytes.NewReader(pdfMagic)
	doc, err := docSvc.Upload(ctx, 2025, domain.PlatformPortu, "statement.pdf", "application/pdf", data)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	result, err := extractionSvc.ExtractFromDocument(ctx, doc.ID)
	if err != nil {
		t.Fatalf("ExtractFromDocument: %v", err)
	}

	// Verify capital entries are persisted in the DB.
	if len(result.CapitalEntries) == 0 {
		t.Fatal("expected at least one capital entry")
	}
	capitalEntries, err := capitalRepo.ListByDocumentID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("ListByDocumentID capital: %v", err)
	}
	if len(capitalEntries) != 1 {
		t.Errorf("persisted capital entries = %d, want 1", len(capitalEntries))
	}

	// Verify security transactions are persisted in the DB.
	if len(result.Transactions) == 0 {
		t.Fatal("expected at least one transaction")
	}
	secTxs, err := securityRepo.ListByDocumentID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("ListByDocumentID security: %v", err)
	}
	if len(secTxs) != 1 {
		t.Errorf("persisted security txs = %d, want 1", len(secTxs))
	}
}
