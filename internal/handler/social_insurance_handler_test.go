package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/testutil"
)

func setupSocialInsuranceRouter(t *testing.T) (*chi.Mux, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	socialRepo := repository.NewSocialInsuranceOverviewRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	taxYearSettingsRepo := repository.NewTaxYearSettingsRepository(db)
	taxPrepaymentRepo := repository.NewTaxPrepaymentRepository(db)

	svc := service.NewSocialInsuranceService(
		socialRepo, invoiceRepo, expenseRepo,
		settingsRepo, taxYearSettingsRepo, taxPrepaymentRepo, nil,
	)
	h := NewSocialInsuranceHandler(svc)

	r := chi.NewRouter()
	r.Mount("/api/v1/social-insurance", h.Routes())
	return r, db
}

func createSocialInsurance(t *testing.T, r *chi.Mux, year int) int64 {
	t.Helper()
	body := fmt.Sprintf(`{"year":%d,"filing_type":"regular"}`, year)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/social-insurance", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("createSocialInsurance: status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp socialInsuranceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("createSocialInsurance: decode error: %v", err)
	}
	return resp.ID
}

func TestSocialInsurance_Create(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)

	body := `{"year":2025,"filing_type":"regular"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/social-insurance", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp socialInsuranceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.Year != 2025 {
		t.Errorf("Year = %d, want 2025", resp.Year)
	}
	if resp.FilingType != "regular" {
		t.Errorf("FilingType = %q, want %q", resp.FilingType, "regular")
	}
	if resp.Status != "draft" {
		t.Errorf("Status = %q, want %q", resp.Status, "draft")
	}
}

func TestSocialInsurance_Create_InvalidJSON(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/social-insurance", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSocialInsurance_Create_InvalidYear(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)

	body := `{"year":0,"filing_type":"regular"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/social-insurance", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestSocialInsurance_Create_Duplicate(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)

	createSocialInsurance(t, r, 2025)

	body := `{"year":2025,"filing_type":"regular"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/social-insurance", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestSocialInsurance_List(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	createSocialInsurance(t, r, 2025)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/social-insurance?year=2025", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var items []socialInsuranceResponse
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

func TestSocialInsurance_List_InvalidYear(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/social-insurance?year=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSocialInsurance_GetByID(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/social-insurance/%d", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp socialInsuranceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.ID != id {
		t.Errorf("ID = %d, want %d", resp.ID, id)
	}
}

func TestSocialInsurance_GetByID_NotFound(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/social-insurance/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestSocialInsurance_GetByID_InvalidID(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/social-insurance/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSocialInsurance_Delete(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/social-insurance/%d", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify it's gone.
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/social-insurance/%d", id), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("after delete: status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestSocialInsurance_Delete_NotFound(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/social-insurance/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestSocialInsurance_Delete_Filed_Conflict(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)

	// Mark filed first.
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/social-insurance/%d/mark-filed", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("mark-filed: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Attempt to delete a filed overview -> 409.
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/social-insurance/%d", id), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("delete filed: status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestSocialInsurance_Recalculate(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/social-insurance/%d/recalculate", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp socialInsuranceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.ID != id {
		t.Errorf("ID = %d, want %d", resp.ID, id)
	}
}

func TestSocialInsurance_Recalculate_NotFound(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/social-insurance/99999/recalculate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestSocialInsurance_Recalculate_Filed_Conflict(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)

	// Mark filed.
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/social-insurance/%d/mark-filed", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("mark-filed: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Recalculate filed overview -> 409.
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/social-insurance/%d/recalculate", id), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestSocialInsurance_MarkFiled(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/social-insurance/%d/mark-filed", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp socialInsuranceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Status != "filed" {
		t.Errorf("Status = %q, want %q", resp.Status, "filed")
	}
	if resp.FiledAt == nil {
		t.Error("expected FiledAt to be set")
	}
}

func TestSocialInsurance_MarkFiled_AlreadyFiled(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)

	// Mark filed first time.
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/social-insurance/%d/mark-filed", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("first mark-filed: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Mark filed second time -> 409.
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/social-insurance/%d/mark-filed", id), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("second mark-filed: status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestSocialInsurance_MarkFiled_NotFound(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/social-insurance/99999/mark-filed", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestSocialInsurance_GenerateXML(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/social-insurance/%d/generate-xml", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp socialInsuranceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if !resp.HasXML {
		t.Error("expected HasXML to be true after generation")
	}
}

func TestSocialInsurance_GenerateXML_NotFound(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/social-insurance/99999/generate-xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestSocialInsurance_DownloadXML(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)

	// Generate XML first.
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/social-insurance/%d/generate-xml", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("generate-xml: status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Download XML.
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/social-insurance/%d/xml", id), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("download xml: status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/xml" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/xml")
	}

	if w.Body.Len() == 0 {
		t.Error("expected non-empty XML body")
	}
}

func TestSocialInsurance_DownloadXML_NotFound(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/social-insurance/99999/xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestSocialInsurance_DownloadXML_NoXMLGenerated(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/social-insurance/%d/xml", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}
