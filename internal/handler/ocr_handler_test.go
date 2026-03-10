package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/service/ocr"
	"github.com/zajca/zfaktury/internal/testutil"
)

// mockOCRProvider implements ocr.Provider for handler tests.
type mockOCRProvider struct {
	result *domain.OCRResult
	err    error
}

func (m *mockOCRProvider) ProcessImage(ctx context.Context, imageData []byte, contentType string) (*domain.OCRResult, error) {
	return m.result, m.err
}

func (m *mockOCRProvider) Name() string {
	return "mock"
}

// Verify mockOCRProvider implements ocr.Provider.
var _ ocr.Provider = (*mockOCRProvider)(nil)

// setupOCRRouter creates a test router with document upload and OCR endpoints.
func setupOCRRouter(t *testing.T, provider ocr.Provider) (*chi.Mux, int64) {
	t.Helper()
	db := testutil.NewTestDB(t)
	dataDir := t.TempDir()

	expense := testutil.SeedExpense(t, db, nil)

	docRepo := repository.NewDocumentRepository(db)
	docSvc := service.NewDocumentService(docRepo, dataDir)
	ocrSvc := service.NewOCRService(provider, docSvc)

	docHandler := NewDocumentHandler(docSvc)
	ocrHandler := NewOCRHandler(ocrSvc)

	r := chi.NewRouter()
	r.Route("/api/v1", func(api chi.Router) {
		api.Mount("/", docHandler.Routes())
		// OCR routes are mounted directly since chi can't mount two handlers on "/".
		api.Post("/documents/{id}/ocr", ocrHandler.ProcessDocument)
	})

	return r, expense.ID
}

// uploadTestDocument uploads a PDF document and returns its ID.
func uploadTestDocument(t *testing.T, r *chi.Mux, expenseID int64) int64 {
	t.Helper()

	url := fmt.Sprintf("/api/v1/expenses/%d/documents", expenseID)
	req := buildUploadRequestWithContentType(t, url, "receipt.pdf", "application/pdf", []byte("%PDF-1.4 test content for OCR"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("upload: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp documentResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return resp.ID
}

func TestOCRHandler_ProcessDocument_Success(t *testing.T) {
	provider := &mockOCRProvider{
		result: &domain.OCRResult{
			VendorName:     "Firma s.r.o.",
			VendorICO:      "12345678",
			VendorDIC:      "CZ12345678",
			InvoiceNumber:  "FV-001",
			IssueDate:      "2026-01-15",
			DueDate:        "2026-02-15",
			TotalAmount:    1210000,
			VATAmount:      210000,
			VATRatePercent: 21,
			CurrencyCode:   "CZK",
			Description:    "IT sluzby",
			Confidence:     0.95,
			Items: []domain.OCRItem{
				{
					Description:    "Konzultace",
					Quantity:       1000,
					UnitPrice:      100000,
					VATRatePercent: 21,
					TotalAmount:    121000,
				},
			},
		},
	}

	r, expenseID := setupOCRRouter(t, provider)
	docID := uploadTestDocument(t, r, expenseID)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/documents/%d/ocr", docID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp ocrResultResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if resp.VendorName != "Firma s.r.o." {
		t.Errorf("VendorName = %q, want %q", resp.VendorName, "Firma s.r.o.")
	}
	if resp.TotalAmount != 1210000 {
		t.Errorf("TotalAmount = %d, want %d", resp.TotalAmount, 1210000)
	}
	if resp.Confidence != 0.95 {
		t.Errorf("Confidence = %f, want %f", resp.Confidence, 0.95)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(resp.Items))
	}
	if resp.Items[0].Description != "Konzultace" {
		t.Errorf("Items[0].Description = %q, want %q", resp.Items[0].Description, "Konzultace")
	}
}

func TestOCRHandler_ProcessDocument_InvalidID(t *testing.T) {
	provider := &mockOCRProvider{}
	r, _ := setupOCRRouter(t, provider)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/abc/ocr", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOCRHandler_ProcessDocument_NotFound(t *testing.T) {
	provider := &mockOCRProvider{}
	r, _ := setupOCRRouter(t, provider)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/99999/ocr", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestOCRHandler_ProcessDocument_ProviderError(t *testing.T) {
	provider := &mockOCRProvider{
		err: fmt.Errorf("API quota exceeded"),
	}

	r, expenseID := setupOCRRouter(t, provider)
	docID := uploadTestDocument(t, r, expenseID)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/documents/%d/ocr", docID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestOCRHandler_ProcessDocument_UnsupportedType(t *testing.T) {
	provider := &mockOCRProvider{
		result: &domain.OCRResult{},
	}

	r, expenseID := setupOCRRouter(t, provider)

	// Upload a WebP image (supported for upload but not for OCR).
	url := fmt.Sprintf("/api/v1/expenses/%d/documents", expenseID)
	// WebP magic: RIFF + size + WEBP
	webpData := make([]byte, 512)
	copy(webpData, []byte("RIFF"))
	copy(webpData[8:], []byte("WEBP"))

	var buf bytes.Buffer
	mw := multipartWriter(t, &buf, "photo.webp", "image/webp", webpData)
	req := httptest.NewRequest(http.MethodPost, url, &buf)
	req.Header.Set("Content-Type", mw)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("upload webp: status = %d, body = %s", w.Code, w.Body.String())
	}

	var docResp documentResponse
	json.NewDecoder(w.Body).Decode(&docResp)

	// Now try OCR on the WebP document.
	ocrReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/documents/%d/ocr", docResp.ID), nil)
	ocrW := httptest.NewRecorder()
	r.ServeHTTP(ocrW, ocrReq)

	if ocrW.Code != http.StatusUnprocessableEntity {
		t.Errorf("OCR on webp: status = %d, want %d, body = %s", ocrW.Code, http.StatusUnprocessableEntity, ocrW.Body.String())
	}
}

// multipartWriter creates a multipart form with a file part and returns the content type.
func multipartWriter(t *testing.T, buf *bytes.Buffer, filename, contentType string, data []byte) string {
	t.Helper()
	mw := multipart.NewWriter(buf)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename)}
	h["Content-Type"] = []string{contentType}
	part, err := mw.CreatePart(h)
	if err != nil {
		t.Fatalf("creating form part: %v", err)
	}
	part.Write(data)
	mw.Close()
	return mw.FormDataContentType()
}
