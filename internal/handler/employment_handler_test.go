package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
)

// --- Mock service ---

type mockEmploymentService struct {
	uploadResp *domain.EmploymentDocument
	uploadErr  error
	uploadGot  struct {
		year        int
		kind        string
		filename    string
		contentType string
		size        int
	}

	extractResp *domain.EmploymentCertificate
	extractErr  error
	extractGot  int64

	listDocsResp []*domain.EmploymentDocument
	listDocsErr  error
	listDocsYear int

	deleteDocErr error
	deleteDocID  int64

	createErr  error
	createGot  *domain.EmploymentCertificate
	updateErr  error
	updateGot  *domain.EmploymentCertificate
	confirmErr error
	confirmID  int64

	getResp *domain.EmploymentCertificate
	getErr  error
	getID   int64

	listResp     []*domain.EmploymentCertificate
	listErr      error
	listYearGot  int
	deleteErr    error
	deleteCertID int64
}

func (m *mockEmploymentService) UploadDocument(_ context.Context, year int, kind, filename, contentType string, content io.Reader) (*domain.EmploymentDocument, error) {
	m.uploadGot.year = year
	m.uploadGot.kind = kind
	m.uploadGot.filename = filename
	m.uploadGot.contentType = contentType
	if content != nil {
		buf, _ := io.ReadAll(content)
		m.uploadGot.size = len(buf)
	}
	if m.uploadErr != nil {
		return nil, m.uploadErr
	}
	return m.uploadResp, nil
}

func (m *mockEmploymentService) ExtractDocument(_ context.Context, docID int64) (*domain.EmploymentCertificate, error) {
	m.extractGot = docID
	if m.extractErr != nil {
		return nil, m.extractErr
	}
	return m.extractResp, nil
}

func (m *mockEmploymentService) ListDocumentsByYear(_ context.Context, year int) ([]*domain.EmploymentDocument, error) {
	m.listDocsYear = year
	return m.listDocsResp, m.listDocsErr
}

func (m *mockEmploymentService) DeleteDocument(_ context.Context, id int64) error {
	m.deleteDocID = id
	return m.deleteDocErr
}

func (m *mockEmploymentService) Create(_ context.Context, cert *domain.EmploymentCertificate) error {
	if m.createErr != nil {
		return m.createErr
	}
	cert.ID = 42
	cert.Status = "draft"
	cert.CreatedAt = time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC)
	cert.UpdatedAt = cert.CreatedAt
	m.createGot = cert
	return nil
}

func (m *mockEmploymentService) Update(_ context.Context, cert *domain.EmploymentCertificate) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	cert.UpdatedAt = time.Date(2025, 5, 2, 12, 0, 0, 0, time.UTC)
	m.updateGot = cert
	return nil
}

func (m *mockEmploymentService) Confirm(_ context.Context, certID int64) error {
	m.confirmID = certID
	return m.confirmErr
}

func (m *mockEmploymentService) Get(_ context.Context, certID int64) (*domain.EmploymentCertificate, error) {
	m.getID = certID
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.getResp, nil
}

func (m *mockEmploymentService) ListByYear(_ context.Context, year int) ([]*domain.EmploymentCertificate, error) {
	m.listYearGot = year
	return m.listResp, m.listErr
}

func (m *mockEmploymentService) Delete(_ context.Context, certID int64) error {
	m.deleteCertID = certID
	return m.deleteErr
}

// --- Test helpers ---

func mountEmployment(svc employmentService) *chi.Mux {
	h := NewEmploymentHandler(svc)
	r := chi.NewRouter()
	r.Route("/api/v1/tax/employment", func(api chi.Router) {
		api.Mount("/", h.Routes())
	})
	return r
}

func sampleEmploymentDoc() *domain.EmploymentDocument {
	return &domain.EmploymentDocument{
		ID:               1,
		Year:             2025,
		Kind:             domain.EmploymentDocAdvance,
		Filename:         "potvrzeni.pdf",
		ContentType:      "application/pdf",
		Size:             512,
		ExtractionStatus: "pending",
		CreatedAt:        time.Date(2025, 4, 1, 9, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2025, 4, 1, 9, 0, 0, 0, time.UTC),
	}
}

func sampleEmploymentCert() *domain.EmploymentCertificate {
	return &domain.EmploymentCertificate{
		ID:                 10,
		Year:               2025,
		CertificateType:    domain.CertificateAdvance,
		EmployerName:       "ACME s.r.o.",
		EmployerICO:        "12345678",
		EmployerAddress:    "Prague",
		ContractType:       domain.ContractHPP,
		PeriodFrom:         time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodTo:           time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		GrossIncome:        12000000, // 120 000 CZK
		AdvanceTaxWithheld: 1800000,  // 18 000 CZK
		Status:             "draft",
		CreatedAt:          time.Date(2025, 4, 1, 9, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2025, 4, 1, 9, 0, 0, 0, time.UTC),
	}
}

// buildEmploymentUploadRequest creates a multipart upload request for the
// employment documents endpoint.
func buildEmploymentUploadRequest(t *testing.T, filename, fileContentType string, content []byte, query string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	h := make(map[string][]string)
	h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename)}
	h["Content-Type"] = []string{fileContentType}
	part, err := mw.CreatePart(h)
	if err != nil {
		t.Fatalf("creating form part: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
		t.Fatalf("copying file content: %v", err)
	}
	mw.Close()

	url := "/api/v1/tax/employment/documents"
	if query != "" {
		url += "?" + query
	}
	req := httptest.NewRequest(http.MethodPost, url, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

// --- Upload tests ---

func TestEmploymentHandler_UploadDocument_OK(t *testing.T) {
	svc := &mockEmploymentService{uploadResp: sampleEmploymentDoc()}
	r := mountEmployment(svc)

	pdfContent := append([]byte("%PDF-1.4 "), bytes.Repeat([]byte{0x00}, 503)...)
	req := buildEmploymentUploadRequest(t, "potvrzeni.pdf", "application/pdf", pdfContent, "year=2025&kind=advance")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}
	var resp employmentDocumentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if resp.ID != 1 || resp.Filename != "potvrzeni.pdf" {
		t.Errorf("unexpected response: %+v", resp)
	}
	if svc.uploadGot.year != 2025 || svc.uploadGot.kind != "advance" {
		t.Errorf("svc args = %+v", svc.uploadGot)
	}
	if svc.uploadGot.contentType != "application/pdf" {
		t.Errorf("contentType = %q, want application/pdf", svc.uploadGot.contentType)
	}
	if svc.uploadGot.size != len(pdfContent) {
		t.Errorf("size = %d, want %d", svc.uploadGot.size, len(pdfContent))
	}
}

func TestEmploymentHandler_UploadDocument_DefaultsKindToAdvance(t *testing.T) {
	svc := &mockEmploymentService{uploadResp: sampleEmploymentDoc()}
	r := mountEmployment(svc)

	req := buildEmploymentUploadRequest(t, "doc.pdf", "application/pdf", []byte("%PDF-1.4 "), "year=2025")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if svc.uploadGot.kind != "advance" {
		t.Errorf("kind = %q, want advance (default)", svc.uploadGot.kind)
	}
}

func TestEmploymentHandler_UploadDocument_MissingYear(t *testing.T) {
	svc := &mockEmploymentService{}
	r := mountEmployment(svc)

	req := buildEmploymentUploadRequest(t, "doc.pdf", "application/pdf", []byte("%PDF-1.4"), "")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestEmploymentHandler_UploadDocument_InvalidYear(t *testing.T) {
	svc := &mockEmploymentService{}
	r := mountEmployment(svc)

	req := buildEmploymentUploadRequest(t, "doc.pdf", "application/pdf", []byte("%PDF-1.4"), "year=abc")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestEmploymentHandler_UploadDocument_MissingFile(t *testing.T) {
	svc := &mockEmploymentService{}
	r := mountEmployment(svc)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("year", "2025")
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax/employment/documents?year=2025", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400, body = %s", w.Code, w.Body.String())
	}
}

func TestEmploymentHandler_UploadDocument_InvalidMultipart(t *testing.T) {
	svc := &mockEmploymentService{}
	r := mountEmployment(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax/employment/documents?year=2025", strings.NewReader("not a multipart body"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xyz")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// MIME rejection: service rejects with ErrInvalidInput → 400.
func TestEmploymentHandler_UploadDocument_MIMERejection(t *testing.T) {
	svc := &mockEmploymentService{
		uploadErr: fmt.Errorf("unsupported MIME type: %w", domain.ErrInvalidInput),
	}
	r := mountEmployment(svc)

	req := buildEmploymentUploadRequest(t, "evil.txt", "text/plain", []byte("hello"), "year=2025&kind=advance")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (invalid input)", w.Code)
	}
}

// Oversize rejection: client tries to send > 10 MB body. The MaxBytesReader
// kicks in during ParseMultipartForm and the handler should answer 400.
func TestEmploymentHandler_UploadDocument_OversizeRejection(t *testing.T) {
	svc := &mockEmploymentService{}
	r := mountEmployment(svc)

	// 11 MB to exceed the 10 MB cap.
	oversize := bytes.Repeat([]byte{0x42}, (10<<20)+(1<<20))
	req := buildEmploymentUploadRequest(t, "big.pdf", "application/pdf", oversize, "year=2025&kind=advance")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (oversize)", w.Code)
	}
	if svc.uploadGot.year != 0 {
		t.Errorf("svc.UploadDocument should not have been called for oversize body, got year=%d", svc.uploadGot.year)
	}
}

// --- Extract test ---

func TestEmploymentHandler_ExtractDocument_OK(t *testing.T) {
	svc := &mockEmploymentService{extractResp: sampleEmploymentCert()}
	r := mountEmployment(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax/employment/documents/7/extract", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if svc.extractGot != 7 {
		t.Errorf("extractGot = %d, want 7", svc.extractGot)
	}
	var resp employmentCertificateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if resp.EmployerICO != "12345678" {
		t.Errorf("EmployerICO = %q", resp.EmployerICO)
	}
	if resp.GrossIncomeCZK != 120000 {
		t.Errorf("GrossIncomeCZK = %f, want 120000", resp.GrossIncomeCZK)
	}
}

func TestEmploymentHandler_ExtractDocument_InvalidID(t *testing.T) {
	svc := &mockEmploymentService{}
	r := mountEmployment(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax/employment/documents/abc/extract", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestEmploymentHandler_ExtractDocument_NotFound(t *testing.T) {
	svc := &mockEmploymentService{
		extractErr: fmt.Errorf("loading document: %w", domain.ErrNotFound),
	}
	r := mountEmployment(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax/employment/documents/9999/extract", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// --- Document list/delete ---

func TestEmploymentHandler_ListDocuments_OK(t *testing.T) {
	svc := &mockEmploymentService{listDocsResp: []*domain.EmploymentDocument{sampleEmploymentDoc()}}
	r := mountEmployment(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tax/employment/documents?year=2025", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var items []employmentDocumentResponse
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("len = %d, want 1", len(items))
	}
	if svc.listDocsYear != 2025 {
		t.Errorf("listDocsYear = %d", svc.listDocsYear)
	}
}

func TestEmploymentHandler_ListDocuments_MissingYear(t *testing.T) {
	r := mountEmployment(&mockEmploymentService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tax/employment/documents", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestEmploymentHandler_DeleteDocument_OK(t *testing.T) {
	svc := &mockEmploymentService{}
	r := mountEmployment(svc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tax/employment/documents/3", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if svc.deleteDocID != 3 {
		t.Errorf("deleteDocID = %d, want 3", svc.deleteDocID)
	}
}

func TestEmploymentHandler_DeleteDocument_InvalidID(t *testing.T) {
	r := mountEmployment(&mockEmploymentService{})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tax/employment/documents/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestEmploymentHandler_DeleteDocument_NotFound(t *testing.T) {
	svc := &mockEmploymentService{deleteDocErr: fmt.Errorf("delete: %w", domain.ErrNotFound)}
	r := mountEmployment(svc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tax/employment/documents/99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// --- Certificate CRUD ---

func TestEmploymentHandler_CreateCertificate_OK(t *testing.T) {
	svc := &mockEmploymentService{}
	r := mountEmployment(svc)

	body := `{
		"year": 2025,
		"certificate_type": "advance",
		"employer_name": "ACME s.r.o.",
		"employer_ico": "12345678",
		"contract_type": "hpp",
		"period_from": "2025-01-01",
		"period_to": "2025-12-31",
		"gross_income_czk": 120000,
		"advance_tax_withheld_czk": 18000
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax/employment/certificates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if svc.createGot == nil {
		t.Fatal("Create not called")
	}
	if svc.createGot.GrossIncome != 12000000 {
		t.Errorf("GrossIncome (halere) = %d, want 12000000", svc.createGot.GrossIncome)
	}
	if svc.createGot.AdvanceTaxWithheld != 1800000 {
		t.Errorf("AdvanceTaxWithheld (halere) = %d, want 1800000", svc.createGot.AdvanceTaxWithheld)
	}
	if svc.createGot.CertificateType != domain.CertificateAdvance {
		t.Errorf("CertificateType = %q", svc.createGot.CertificateType)
	}
	if !svc.createGot.PeriodFrom.Equal(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("PeriodFrom = %v", svc.createGot.PeriodFrom)
	}
}

func TestEmploymentHandler_CreateCertificate_InvalidJSON(t *testing.T) {
	r := mountEmployment(&mockEmploymentService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax/employment/certificates", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestEmploymentHandler_CreateCertificate_InvalidDate(t *testing.T) {
	r := mountEmployment(&mockEmploymentService{})

	body := `{"year":2025,"certificate_type":"advance","period_from":"01-01-2025","period_to":"2025-12-31"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax/employment/certificates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestEmploymentHandler_CreateCertificate_ValidationError(t *testing.T) {
	svc := &mockEmploymentService{
		createErr: fmt.Errorf("invalid IČO: %w", domain.ErrInvalidInput),
	}
	r := mountEmployment(svc)

	body := `{
		"year": 2025,
		"certificate_type": "advance",
		"period_from": "2025-01-01",
		"period_to": "2025-12-31"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax/employment/certificates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (invalid input)", w.Code)
	}
}

func TestEmploymentHandler_UpdateCertificate_OK(t *testing.T) {
	svc := &mockEmploymentService{}
	r := mountEmployment(svc)

	body := `{
		"year": 2025,
		"certificate_type": "withholding",
		"contract_type": "dpp",
		"period_from": "2025-06-01",
		"period_to": "2025-06-30",
		"gross_income_czk": 9500,
		"withheld_final_tax_czk": 1425,
		"include_withholding_in_dap": true
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax/employment/certificates/55", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if svc.updateGot == nil {
		t.Fatal("Update not called")
	}
	if svc.updateGot.ID != 55 {
		t.Errorf("Update ID = %d, want 55", svc.updateGot.ID)
	}
	if svc.updateGot.WithheldFinalTax != 142500 {
		t.Errorf("WithheldFinalTax = %d, want 142500", svc.updateGot.WithheldFinalTax)
	}
	if !svc.updateGot.IncludeWithholdingInDAP {
		t.Errorf("IncludeWithholdingInDAP = false, want true")
	}
}

func TestEmploymentHandler_UpdateCertificate_InvalidID(t *testing.T) {
	r := mountEmployment(&mockEmploymentService{})

	body := `{"year":2025,"certificate_type":"advance","period_from":"2025-01-01","period_to":"2025-12-31"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax/employment/certificates/abc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestEmploymentHandler_UpdateCertificate_NotFound(t *testing.T) {
	svc := &mockEmploymentService{updateErr: fmt.Errorf("update: %w", domain.ErrNotFound)}
	r := mountEmployment(svc)

	body := `{"year":2025,"certificate_type":"advance","period_from":"2025-01-01","period_to":"2025-12-31"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax/employment/certificates/9999", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestEmploymentHandler_GetCertificate_OK(t *testing.T) {
	svc := &mockEmploymentService{getResp: sampleEmploymentCert()}
	r := mountEmployment(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tax/employment/certificates/10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if svc.getID != 10 {
		t.Errorf("getID = %d, want 10", svc.getID)
	}
	var resp employmentCertificateResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID != 10 {
		t.Errorf("ID = %d, want 10", resp.ID)
	}
}

func TestEmploymentHandler_GetCertificate_NotFound(t *testing.T) {
	svc := &mockEmploymentService{getErr: fmt.Errorf("get: %w", domain.ErrNotFound)}
	r := mountEmployment(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tax/employment/certificates/99", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestEmploymentHandler_ListCertificates_OK(t *testing.T) {
	svc := &mockEmploymentService{
		listResp: []*domain.EmploymentCertificate{sampleEmploymentCert()},
	}
	r := mountEmployment(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tax/employment/certificates?year=2025", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp []employmentCertificateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("len = %d, want 1", len(resp))
	}
	if svc.listYearGot != 2025 {
		t.Errorf("listYearGot = %d", svc.listYearGot)
	}
}

func TestEmploymentHandler_ListCertificates_MissingYear(t *testing.T) {
	r := mountEmployment(&mockEmploymentService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tax/employment/certificates", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestEmploymentHandler_ConfirmCertificate_OK(t *testing.T) {
	svc := &mockEmploymentService{getResp: sampleEmploymentCert()}
	r := mountEmployment(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax/employment/certificates/10/confirm", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if svc.confirmID != 10 {
		t.Errorf("confirmID = %d, want 10", svc.confirmID)
	}
	if svc.getID != 10 {
		t.Errorf("expected Get to be called after confirm; getID = %d", svc.getID)
	}
}

func TestEmploymentHandler_ConfirmCertificate_InvalidID(t *testing.T) {
	r := mountEmployment(&mockEmploymentService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax/employment/certificates/abc/confirm", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestEmploymentHandler_ConfirmCertificate_ValidationError(t *testing.T) {
	svc := &mockEmploymentService{
		confirmErr: fmt.Errorf("confirm: %w", domain.ErrInvalidInput),
	}
	r := mountEmployment(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax/employment/certificates/10/confirm", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestEmploymentHandler_DeleteCertificate_OK(t *testing.T) {
	svc := &mockEmploymentService{}
	r := mountEmployment(svc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tax/employment/certificates/10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if svc.deleteCertID != 10 {
		t.Errorf("deleteCertID = %d, want 10", svc.deleteCertID)
	}
}

func TestEmploymentHandler_DeleteCertificate_InvalidID(t *testing.T) {
	r := mountEmployment(&mockEmploymentService{})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tax/employment/certificates/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestEmploymentHandler_DeleteCertificate_NotFound(t *testing.T) {
	svc := &mockEmploymentService{deleteErr: fmt.Errorf("delete: %w", domain.ErrNotFound)}
	r := mountEmployment(svc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tax/employment/certificates/9999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// --- Internal: amount conversion ---

func TestEmploymentHandler_AmountFromCZK_Rounding(t *testing.T) {
	cases := []struct {
		in   float64
		want domain.Amount
	}{
		{0, 0},
		{1, 100},
		{1.5, 150},
		{12345.67, 1234567},
		{0.005, 1}, // half-away-from-zero
		{-1.5, -150},
		{-12345.67, -1234567},
	}
	for _, c := range cases {
		got := amountFromCZK(c.in)
		if got != c.want {
			t.Errorf("amountFromCZK(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

// Ensure the mock satisfies the handler-side interface.
var _ employmentService = (*mockEmploymentService)(nil)

// errors.Is sanity for the wrapped errors used in tests above.
func TestEmploymentHandler_ErrorMappingSanity(t *testing.T) {
	wrapped := fmt.Errorf("ctx: %w", domain.ErrNotFound)
	if !errors.Is(wrapped, domain.ErrNotFound) {
		t.Fatal("expected wrapped error to match ErrNotFound via errors.Is")
	}
}
