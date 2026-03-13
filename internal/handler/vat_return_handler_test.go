package handler

import (
	"bytes"
	"context"
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

func setupVATReturnRouter(t *testing.T) (*chi.Mux, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	vatReturnRepo := repository.NewVATReturnRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	vatReturnSvc := service.NewVATReturnService(vatReturnRepo, invoiceRepo, expenseRepo, settingsRepo, nil)
	h := NewVATReturnHandler(vatReturnSvc)

	r := chi.NewRouter()
	r.Mount("/api/v1/vat-returns", h.Routes())
	return r, db
}

// createVATReturn is a helper that POSTs a valid VAT return and returns its ID.
func createVATReturn(t *testing.T, r *chi.Mux) int64 {
	t.Helper()
	body := `{"year":2025,"month":1,"filing_type":"regular"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vat-returns", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("createVATReturn: status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp vatReturnResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("createVATReturn: decode error: %v", err)
	}
	return resp.ID
}

func TestVATReturn_Create(t *testing.T) {
	r, _ := setupVATReturnRouter(t)

	body := `{"year":2025,"month":1,"filing_type":"regular"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vat-returns", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp vatReturnResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.Period.Year != 2025 {
		t.Errorf("Period.Year = %d, want 2025", resp.Period.Year)
	}
	if resp.Period.Month != 1 {
		t.Errorf("Period.Month = %d, want 1", resp.Period.Month)
	}
	if resp.FilingType != "regular" {
		t.Errorf("FilingType = %q, want %q", resp.FilingType, "regular")
	}
}

func TestVATReturn_Create_InvalidJSON(t *testing.T) {
	r, _ := setupVATReturnRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vat-returns", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestVATReturn_Create_InvalidYear(t *testing.T) {
	r, _ := setupVATReturnRouter(t)

	body := `{"year":0,"month":1,"filing_type":"regular"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vat-returns", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Year 0 is out of valid range (2000-2100) in the service, which wraps ErrInvalidInput.
	// The handler maps ErrInvalidInput to 400.
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestVATReturn_List_WithYear(t *testing.T) {
	r, _ := setupVATReturnRouter(t)

	// Create a VAT return first.
	createVATReturn(t, r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vat-returns?year=2025", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var items []vatReturnResponse
	if err := json.NewDecoder(w.Body).Decode(&items); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

func TestVATReturn_List_InvalidYear(t *testing.T) {
	r, _ := setupVATReturnRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vat-returns?year=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestVATReturn_GetByID(t *testing.T) {
	r, _ := setupVATReturnRouter(t)
	id := createVATReturn(t, r)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vat-returns/%d", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp vatReturnResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.ID != id {
		t.Errorf("ID = %d, want %d", resp.ID, id)
	}
}

func TestVATReturn_GetByID_NotFound(t *testing.T) {
	r, _ := setupVATReturnRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vat-returns/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestVATReturn_GetByID_InvalidID(t *testing.T) {
	r, _ := setupVATReturnRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vat-returns/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestVATReturn_Delete(t *testing.T) {
	r, _ := setupVATReturnRouter(t)
	id := createVATReturn(t, r)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/vat-returns/%d", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify it's gone.
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vat-returns/%d", id), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("after delete: status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestVATReturn_Delete_NotFound(t *testing.T) {
	r, _ := setupVATReturnRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/vat-returns/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestVATReturn_MarkFiled(t *testing.T) {
	r, _ := setupVATReturnRouter(t)
	id := createVATReturn(t, r)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-returns/%d/mark-filed", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp vatReturnResponse
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

func TestVATReturn_MarkFiled_AlreadyFiled(t *testing.T) {
	r, _ := setupVATReturnRouter(t)
	id := createVATReturn(t, r)

	// Mark filed the first time.
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-returns/%d/mark-filed", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("first mark-filed: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Mark filed the second time -> 409 conflict.
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-returns/%d/mark-filed", id), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("second mark-filed: status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestVATReturn_Delete_Filed_Conflict(t *testing.T) {
	r, _ := setupVATReturnRouter(t)
	id := createVATReturn(t, r)

	// Mark filed first.
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-returns/%d/mark-filed", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("mark-filed: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Attempt to delete a filed return -> 409.
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/vat-returns/%d", id), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("delete filed: status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestVATReturn_Recalculate(t *testing.T) {
	r, _ := setupVATReturnRouter(t)
	id := createVATReturn(t, r)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-returns/%d/recalculate", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp vatReturnResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.ID != id {
		t.Errorf("ID = %d, want %d", resp.ID, id)
	}
}

func TestVATReturn_Recalculate_NotFound(t *testing.T) {
	r, _ := setupVATReturnRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vat-returns/99999/recalculate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestVATReturn_GenerateXML(t *testing.T) {
	r, db := setupVATReturnRouter(t)

	// Insert "dic" setting required for XML generation.
	_, err := db.ExecContext(context.Background(),
		"INSERT INTO settings (key, value, updated_at) VALUES ('dic', 'CZ12345678', datetime('now'))")
	if err != nil {
		t.Fatalf("inserting dic setting: %v", err)
	}

	id := createVATReturn(t, r)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-returns/%d/generate-xml", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp vatReturnResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if !resp.HasXML {
		t.Error("expected HasXML to be true after generation")
	}
}

func TestVATReturn_GenerateXML_NotFound(t *testing.T) {
	r, _ := setupVATReturnRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vat-returns/99999/generate-xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestVATReturn_DownloadXML(t *testing.T) {
	r, db := setupVATReturnRouter(t)

	// Insert "dic" setting required for XML generation.
	_, err := db.ExecContext(context.Background(),
		"INSERT INTO settings (key, value, updated_at) VALUES ('dic', 'CZ12345678', datetime('now'))")
	if err != nil {
		t.Fatalf("inserting dic setting: %v", err)
	}

	id := createVATReturn(t, r)

	// Generate XML first.
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-returns/%d/generate-xml", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("generate-xml: status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Download XML.
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vat-returns/%d/xml", id), nil)
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

func TestVATReturn_DownloadXML_NotFound(t *testing.T) {
	r, _ := setupVATReturnRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vat-returns/99999/xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestVATReturn_DownloadXML_NoXMLGenerated(t *testing.T) {
	r, _ := setupVATReturnRouter(t)
	id := createVATReturn(t, r)

	// Try to download XML without generating it first.
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vat-returns/%d/xml", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}
