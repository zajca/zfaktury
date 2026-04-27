package service

import (
	"bytes"
	"context"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service/ocr"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestNormalizeDeductionCategory(t *testing.T) {
	cases := map[string]string{
		domain.DeductionMortgage:      domain.DeductionMortgage,
		domain.DeductionLifeInsurance: domain.DeductionLifeInsurance,
		domain.DeductionPension:       domain.DeductionPension,
		domain.DeductionDonation:      domain.DeductionDonation,
		domain.DeductionUnionDues:     domain.DeductionUnionDues,
		"unknown":                     "",
		"":                            "",
		"random_garbage":              "",
		"MORTGAGE":                    "", // case-sensitive
	}
	for in, want := range cases {
		if got := normalizeDeductionCategory(in); got != want {
			t.Errorf("normalizeDeductionCategory(%q) = %q, want %q", in, got, want)
		}
	}
}

// taxExtractionMockProvider implements ocr.Provider returning a canned JSON
// response for ProcessWithPrompt.
type taxExtractionMockProvider struct {
	response string
	err      error
}

func (m *taxExtractionMockProvider) ProcessImage(_ context.Context, _ []byte, _ string) (*domain.OCRResult, error) {
	return nil, nil
}

func (m *taxExtractionMockProvider) ProcessWithPrompt(_ context.Context, _ []byte, _ string, _, _ string) (string, error) {
	return m.response, m.err
}

func (m *taxExtractionMockProvider) Name() string {
	return "tax-extraction-mock"
}

var _ ocr.Provider = (*taxExtractionMockProvider)(nil)

func newTaxExtractionTestService(t *testing.T, provider ocr.Provider) (*TaxDocumentExtractionService, *TaxDeductionDocumentService, repository.TaxDeductionRepo) {
	t.Helper()
	db := testutil.NewTestDB(t)
	deductRepo := repository.NewTaxDeductionRepository(db)
	docRepo := repository.NewTaxDeductionDocumentRepository(db)
	dataDir := t.TempDir()
	docSvc := NewTaxDeductionDocumentService(docRepo, deductRepo, dataDir, nil)
	extSvc := NewTaxDocumentExtractionService(provider, docSvc, deductRepo, docRepo)
	return extSvc, docSvc, deductRepo
}

const mockTaxDeductionJSON = `{
	"category": "mortgage",
	"provider_name": "Česká spořitelna",
	"provider_ico": "45244782",
	"contract_number": "12345/2025",
	"document_date": "2026-01-15",
	"period_year": 2025,
	"amount_czk": 45000.50,
	"purpose": "",
	"description_suggestion": "Úroky z hypotéky 2025",
	"confidence": 0.92,
	"raw_text": "POTVRZENI"
}`

func TestTaxDocumentExtractionService_ExtractAmount_HappyPath(t *testing.T) {
	provider := &taxExtractionMockProvider{response: mockTaxDeductionJSON}
	extSvc, docSvc, deductRepo := newTaxExtractionTestService(t, provider)
	ctx := context.Background()

	ded := &domain.TaxDeduction{
		Year:          2025,
		Category:      domain.DeductionMortgage,
		Description:   "",
		ClaimedAmount: 0,
	}
	if err := deductRepo.Create(ctx, ded); err != nil {
		t.Fatalf("seeding deduction: %v", err)
	}
	doc, err := docSvc.Upload(ctx, ded.ID, "potvrzeni.pdf", "application/pdf", bytes.NewReader(pdfMagic))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	result, err := extSvc.ExtractAmount(ctx, doc.ID)
	if err != nil {
		t.Fatalf("ExtractAmount: %v", err)
	}

	if result.Category != domain.DeductionMortgage {
		t.Errorf("Category = %q, want mortgage", result.Category)
	}
	if result.ProviderName != "Česká spořitelna" {
		t.Errorf("ProviderName = %q", result.ProviderName)
	}
	if result.ContractNumber != "12345/2025" {
		t.Errorf("ContractNumber = %q", result.ContractNumber)
	}
	if result.AmountCZK != 45000 {
		t.Errorf("AmountCZK = %d, want 45000 (truncated whole crowns)", result.AmountCZK)
	}
	if result.AmountHalere != 4500050 {
		t.Errorf("AmountHalere = %d, want 4500050", int64(result.AmountHalere))
	}
	if result.PeriodYear != 2025 {
		t.Errorf("PeriodYear = %d, want 2025 (parent's year wins)", result.PeriodYear)
	}
	if result.Confidence != 0.92 {
		t.Errorf("Confidence = %v", result.Confidence)
	}

	// The document should have its extraction fields persisted.
	got, err := docSvc.GetByID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if int64(got.ExtractedAmount) != 4500050 {
		t.Errorf("persisted ExtractedAmount = %d, want 4500050", int64(got.ExtractedAmount))
	}
	if got.Confidence != 0.92 {
		t.Errorf("persisted Confidence = %v", got.Confidence)
	}
}

func TestTaxDocumentExtractionService_ExtractAmount_RejectsZeroID(t *testing.T) {
	provider := &taxExtractionMockProvider{response: mockTaxDeductionJSON}
	extSvc, _, _ := newTaxExtractionTestService(t, provider)
	if _, err := extSvc.ExtractAmount(context.Background(), 0); err == nil {
		t.Fatal("expected error for zero document ID")
	}
}

func TestTaxDocumentExtractionService_ExtractAmount_NormalisesUnknownCategory(t *testing.T) {
	provider := &taxExtractionMockProvider{response: `{"category":"weird","amount_czk":1000,"period_year":2025,"confidence":0.5}`}
	extSvc, docSvc, deductRepo := newTaxExtractionTestService(t, provider)
	ctx := context.Background()

	ded := &domain.TaxDeduction{Year: 2025, Category: domain.DeductionDonation}
	if err := deductRepo.Create(ctx, ded); err != nil {
		t.Fatalf("seeding deduction: %v", err)
	}
	doc, err := docSvc.Upload(ctx, ded.ID, "x.pdf", "application/pdf", bytes.NewReader(pdfMagic))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	result, err := extSvc.ExtractAmount(ctx, doc.ID)
	if err != nil {
		t.Fatalf("ExtractAmount: %v", err)
	}
	if result.Category != "" {
		t.Errorf("Category = %q, want empty for unknown", result.Category)
	}
}
