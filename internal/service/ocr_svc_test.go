package service

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service/ocr"
)

// mockOCRProvider implements ocr.Provider for testing.
type mockOCRProvider struct {
	result *domain.OCRResult
	err    error
}

func (m *mockOCRProvider) ProcessImage(ctx context.Context, imageData []byte, contentType string) (*domain.OCRResult, error) {
	return m.result, m.err
}

func (m *mockOCRProvider) ProcessWithPrompt(_ context.Context, _ []byte, _ string, _, _ string) (string, error) {
	return "", nil
}

func (m *mockOCRProvider) Name() string {
	return "mock"
}

// Verify mockOCRProvider implements ocr.Provider.
var _ ocr.Provider = (*mockOCRProvider)(nil)

func newOCRTestService(t *testing.T, provider ocr.Provider) (*OCRService, *DocumentService) {
	t.Helper()
	docSvc, _ := newDocumentTestService(t)
	ocrSvc := NewOCRService(provider, docSvc)
	return ocrSvc, docSvc
}

func TestOCRService_ProcessDocument_Success(t *testing.T) {
	expectedResult := &domain.OCRResult{
		VendorName:  "Test Vendor",
		TotalAmount: 100000,
		Confidence:  0.95,
	}
	provider := &mockOCRProvider{result: expectedResult}
	ocrSvc, docSvc := newOCRTestService(t, provider)
	ctx := context.Background()

	// Upload a valid document first.
	data := bytes.NewReader(pdfMagic)
	doc, err := docSvc.Upload(ctx, 1, "receipt.pdf", "application/pdf", data)
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	result, err := ocrSvc.ProcessDocument(ctx, doc.ID)
	if err != nil {
		t.Fatalf("ProcessDocument() error: %v", err)
	}
	if result.VendorName != "Test Vendor" {
		t.Errorf("VendorName = %q, want %q", result.VendorName, "Test Vendor")
	}
	if result.TotalAmount != 100000 {
		t.Errorf("TotalAmount = %d, want %d", result.TotalAmount, 100000)
	}
}

func TestOCRService_ProcessDocument_ZeroID(t *testing.T) {
	provider := &mockOCRProvider{}
	ocrSvc, _ := newOCRTestService(t, provider)

	_, err := ocrSvc.ProcessDocument(context.Background(), 0)
	if err == nil {
		t.Error("expected error for zero document ID")
	}
}

func TestOCRService_ProcessDocument_NotFound(t *testing.T) {
	provider := &mockOCRProvider{}
	ocrSvc, _ := newOCRTestService(t, provider)

	_, err := ocrSvc.ProcessDocument(context.Background(), 99999)
	if err == nil {
		t.Error("expected error for non-existent document")
	}
}

func TestOCRService_ProcessDocument_UnsupportedContentType(t *testing.T) {
	provider := &mockOCRProvider{}
	ocrSvc, docSvc := newOCRTestService(t, provider)
	ctx := context.Background()

	// Upload a webp file (supported for upload but not for OCR).
	data := bytes.NewReader(webpMagic)
	doc, err := docSvc.Upload(ctx, 1, "photo.webp", "image/webp", data)
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	_, err = ocrSvc.ProcessDocument(ctx, doc.ID)
	if err == nil {
		t.Error("expected error for unsupported content type")
	}
}

func TestOCRService_ProcessDocument_ProviderError(t *testing.T) {
	provider := &mockOCRProvider{err: errors.New("API quota exceeded")}
	ocrSvc, docSvc := newOCRTestService(t, provider)
	ctx := context.Background()

	data := bytes.NewReader(jpegMagic)
	doc, err := docSvc.Upload(ctx, 1, "photo.jpg", "image/jpeg", data)
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	_, err = ocrSvc.ProcessDocument(ctx, doc.ID)
	if err == nil {
		t.Error("expected error when provider fails")
	}
}

func TestOCRService_ProcessDocument_JPEG(t *testing.T) {
	expectedResult := &domain.OCRResult{
		VendorName: "JPEG Vendor",
		Confidence: 0.8,
	}
	provider := &mockOCRProvider{result: expectedResult}
	ocrSvc, docSvc := newOCRTestService(t, provider)
	ctx := context.Background()

	data := bytes.NewReader(jpegMagic)
	doc, err := docSvc.Upload(ctx, 1, "scan.jpg", "image/jpeg", data)
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	result, err := ocrSvc.ProcessDocument(ctx, doc.ID)
	if err != nil {
		t.Fatalf("ProcessDocument() error: %v", err)
	}
	if result.VendorName != "JPEG Vendor" {
		t.Errorf("VendorName = %q, want %q", result.VendorName, "JPEG Vendor")
	}
}

func TestOCRService_ProcessDocument_PNG(t *testing.T) {
	expectedResult := &domain.OCRResult{
		VendorName: "PNG Vendor",
		Confidence: 0.7,
	}
	provider := &mockOCRProvider{result: expectedResult}
	ocrSvc, docSvc := newOCRTestService(t, provider)
	ctx := context.Background()

	data := bytes.NewReader(pngMagic)
	doc, err := docSvc.Upload(ctx, 1, "scan.png", "image/png", data)
	if err != nil {
		t.Fatalf("Upload() error: %v", err)
	}

	result, err := ocrSvc.ProcessDocument(ctx, doc.ID)
	if err != nil {
		t.Fatalf("ProcessDocument() error: %v", err)
	}
	if result.VendorName != "PNG Vendor" {
		t.Errorf("VendorName = %q, want %q", result.VendorName, "PNG Vendor")
	}
}
