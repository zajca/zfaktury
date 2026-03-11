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

func setupVIESRouter(t *testing.T) (*chi.Mux, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	viesRepo := repository.NewVIESSummaryRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	contactRepo := repository.NewContactRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	viesSvc := service.NewVIESSummaryService(viesRepo, invoiceRepo, contactRepo)
	settingsSvc := service.NewSettingsService(settingsRepo)
	h := NewVIESHandler(viesSvc, settingsSvc)

	r := chi.NewRouter()
	r.Mount("/api/v1/vies-summaries", h.Routes())
	return r, db
}

func createTestVIESSummary(t *testing.T, r *chi.Mux, year, quarter int) viesSummaryResponse {
	t.Helper()
	body := fmt.Sprintf(`{"year":%d,"quarter":%d,"filing_type":"regular"}`, year, quarter)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vies-summaries", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("creating VIES summary: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp viesSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return resp
}

func TestVIESHandler_Create(t *testing.T) {
	r, _ := setupVIESRouter(t)

	body := `{"year":2025,"quarter":1,"filing_type":"regular"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vies-summaries", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp viesSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.Period.Year != 2025 {
		t.Errorf("Period.Year = %d, want 2025", resp.Period.Year)
	}
	if resp.Period.Quarter != 1 {
		t.Errorf("Period.Quarter = %d, want 1", resp.Period.Quarter)
	}
	if resp.FilingType != "regular" {
		t.Errorf("FilingType = %q, want %q", resp.FilingType, "regular")
	}
}

func TestVIESHandler_Create_InvalidJSON(t *testing.T) {
	r, _ := setupVIESRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vies-summaries", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestVIESHandler_Create_Duplicate(t *testing.T) {
	r, _ := setupVIESRouter(t)

	// Create first summary.
	createTestVIESSummary(t, r, 2025, 1)

	// Attempt duplicate.
	body := `{"year":2025,"quarter":1,"filing_type":"regular"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vies-summaries", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestVIESHandler_List(t *testing.T) {
	r, _ := setupVIESRouter(t)

	createTestVIESSummary(t, r, 2025, 1)
	createTestVIESSummary(t, r, 2025, 2)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vies-summaries?year=2025", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []viesSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("len(resp) = %d, want 2", len(resp))
	}
}

func TestVIESHandler_GetByID(t *testing.T) {
	r, _ := setupVIESRouter(t)

	created := createTestVIESSummary(t, r, 2025, 1)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vies-summaries/%d", created.ID), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp viesSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.ID != created.ID {
		t.Errorf("ID = %d, want %d", resp.ID, created.ID)
	}
}

func TestVIESHandler_GetByID_NotFound(t *testing.T) {
	r, _ := setupVIESRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vies-summaries/99999", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestVIESHandler_Delete(t *testing.T) {
	r, _ := setupVIESRouter(t)

	created := createTestVIESSummary(t, r, 2025, 1)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/vies-summaries/%d", created.ID), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusNoContent, w.Body.String())
	}
}

func TestVIESHandler_Delete_NotFound(t *testing.T) {
	r, _ := setupVIESRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/vies-summaries/99999", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestVIESHandler_MarkFiled(t *testing.T) {
	r, _ := setupVIESRouter(t)

	created := createTestVIESSummary(t, r, 2025, 1)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vies-summaries/%d/mark-filed", created.ID), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp viesSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.Status != "filed" {
		t.Errorf("Status = %q, want %q", resp.Status, "filed")
	}
	if resp.FiledAt == nil {
		t.Error("expected FiledAt to be set")
	}
}

func TestVIESHandler_MarkFiled_AlreadyFiled(t *testing.T) {
	r, _ := setupVIESRouter(t)

	created := createTestVIESSummary(t, r, 2025, 1)

	// Mark filed first time.
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vies-summaries/%d/mark-filed", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("first mark-filed: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Mark filed second time.
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vies-summaries/%d/mark-filed", created.ID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestVIESHandler_GenerateXML_NoDIC(t *testing.T) {
	r, _ := setupVIESRouter(t)

	created := createTestVIESSummary(t, r, 2025, 1)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vies-summaries/%d/generate-xml", created.ID), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestVIESHandler_GenerateXML_WithDIC(t *testing.T) {
	r, db := setupVIESRouter(t)

	// Insert DIC setting.
	_, err := db.ExecContext(context.Background(), "INSERT INTO settings (key, value, updated_at) VALUES ('dic', 'CZ12345678', datetime('now'))")
	if err != nil {
		t.Fatalf("inserting DIC setting: %v", err)
	}

	created := createTestVIESSummary(t, r, 2025, 1)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vies-summaries/%d/generate-xml", created.ID), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp viesSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if !resp.HasXML {
		t.Error("expected HasXML to be true after generation")
	}
}

func TestVIESHandler_DownloadXML_NoXMLGenerated(t *testing.T) {
	r, _ := setupVIESRouter(t)

	created := createTestVIESSummary(t, r, 2025, 1)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vies-summaries/%d/xml", created.ID), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestVIESHandler_DownloadXML_InvalidID(t *testing.T) {
	r, _ := setupVIESRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vies-summaries/abc/xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestVIESHandler_DownloadXML_NotFound(t *testing.T) {
	r, _ := setupVIESRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vies-summaries/99999/xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestVIESHandler_DownloadXML_Success(t *testing.T) {
	r, db := setupVIESRouter(t)

	// Insert DIC setting.
	_, err := db.ExecContext(context.Background(), "INSERT INTO settings (key, value, updated_at) VALUES ('dic', 'CZ12345678', datetime('now'))")
	if err != nil {
		t.Fatalf("inserting DIC setting: %v", err)
	}

	created := createTestVIESSummary(t, r, 2025, 2)

	// Generate XML first.
	genReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vies-summaries/%d/generate-xml", created.ID), nil)
	genW := httptest.NewRecorder()
	r.ServeHTTP(genW, genReq)
	if genW.Code != http.StatusOK {
		t.Fatalf("generate-xml: status = %d, want %d, body = %s", genW.Code, http.StatusOK, genW.Body.String())
	}

	// Download the XML.
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vies-summaries/%d/xml", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/xml" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/xml")
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty XML body")
	}
}

func TestVIESHandler_Recalculate_Success(t *testing.T) {
	r, _ := setupVIESRouter(t)

	created := createTestVIESSummary(t, r, 2025, 3)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vies-summaries/%d/recalculate", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp viesSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.ID != created.ID {
		t.Errorf("ID = %d, want %d", resp.ID, created.ID)
	}
}

func TestVIESHandler_Recalculate_InvalidID(t *testing.T) {
	r, _ := setupVIESRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vies-summaries/abc/recalculate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestVIESHandler_Recalculate_NotFound(t *testing.T) {
	r, _ := setupVIESRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vies-summaries/99999/recalculate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestVIESHandler_Recalculate_Filed(t *testing.T) {
	r, _ := setupVIESRouter(t)

	created := createTestVIESSummary(t, r, 2025, 4)

	// Mark as filed first.
	markReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vies-summaries/%d/mark-filed", created.ID), nil)
	markW := httptest.NewRecorder()
	r.ServeHTTP(markW, markReq)
	if markW.Code != http.StatusOK {
		t.Fatalf("mark-filed: status = %d, want %d", markW.Code, http.StatusOK)
	}

	// Recalculate a filed summary should fail.
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vies-summaries/%d/recalculate", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestVIESHandler_GenerateXML_InvalidID(t *testing.T) {
	r, _ := setupVIESRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vies-summaries/abc/generate-xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestVIESHandler_GenerateXML_NotFound(t *testing.T) {
	r, db := setupVIESRouter(t)

	// Insert DIC to get past the DIC check.
	_, err := db.ExecContext(context.Background(), "INSERT INTO settings (key, value, updated_at) VALUES ('dic', 'CZ12345678', datetime('now'))")
	if err != nil {
		t.Fatalf("inserting DIC setting: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vies-summaries/99999/generate-xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestVIESHandler_Delete_InvalidID(t *testing.T) {
	r, _ := setupVIESRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/vies-summaries/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestVIESHandler_Delete_Filed(t *testing.T) {
	r, _ := setupVIESRouter(t)

	created := createTestVIESSummary(t, r, 2026, 1)

	// Mark as filed.
	markReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vies-summaries/%d/mark-filed", created.ID), nil)
	markW := httptest.NewRecorder()
	r.ServeHTTP(markW, markReq)
	if markW.Code != http.StatusOK {
		t.Fatalf("mark-filed: status = %d", markW.Code)
	}

	// Delete a filed summary should fail.
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/vies-summaries/%d", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestVIESHandler_MarkFiled_InvalidID(t *testing.T) {
	r, _ := setupVIESRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vies-summaries/abc/mark-filed", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestVIESHandler_MarkFiled_NotFound(t *testing.T) {
	r, _ := setupVIESRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vies-summaries/99999/mark-filed", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestVIESHandler_GetByID_InvalidID(t *testing.T) {
	r, _ := setupVIESRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vies-summaries/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestVIESHandler_List_InvalidYear(t *testing.T) {
	r, _ := setupVIESRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vies-summaries?year=notanumber", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestVIESHandler_List_NoYear(t *testing.T) {
	r, _ := setupVIESRouter(t)

	// Without year param, should default to current year.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vies-summaries", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestVIESHandler_Create_InvalidInput(t *testing.T) {
	r, _ := setupVIESRouter(t)

	// Quarter 0 should be invalid input.
	body := `{"year":2025,"quarter":0,"filing_type":"regular"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vies-summaries", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestVIESHandler_List_EmptyResult(t *testing.T) {
	r, _ := setupVIESRouter(t)

	// Query a year with no summaries.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vies-summaries?year=1999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp []viesSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("len(resp) = %d, want 0", len(resp))
	}
}

func TestVIESHandler_Recalculate_SuccessWithLines(t *testing.T) {
	r, _ := setupVIESRouter(t)

	created := createTestVIESSummary(t, r, 2026, 2)

	// Recalculate and verify lines field is present in response.
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vies-summaries/%d/recalculate", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp viesSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.ID != created.ID {
		t.Errorf("ID = %d, want %d", resp.ID, created.ID)
	}
	// Lines should be an empty slice (no invoices for this period).
	if resp.Lines != nil && len(resp.Lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(resp.Lines))
	}
}

func TestVIESHandler_MarkFiled_NotFoundID(t *testing.T) {
	r, _ := setupVIESRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vies-summaries/88888/mark-filed", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestVIESHandler_GenerateXML_WithDIC_VerifyHasXML(t *testing.T) {
	r, db := setupVIESRouter(t)

	// Insert DIC setting.
	_, err := db.ExecContext(context.Background(), "INSERT INTO settings (key, value, updated_at) VALUES ('dic', 'CZ99999999', datetime('now'))")
	if err != nil {
		t.Fatalf("inserting DIC setting: %v", err)
	}

	created := createTestVIESSummary(t, r, 2026, 3)

	// Generate XML.
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vies-summaries/%d/generate-xml", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("generate-xml: status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp viesSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if !resp.HasXML {
		t.Error("expected HasXML to be true after generation")
	}

	// Verify GetByID also returns the summary with HasXML=true.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vies-summaries/%d", created.ID), nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("get after generate: status = %d", getW.Code)
	}
	var getResp viesSummaryResponse
	json.NewDecoder(getW.Body).Decode(&getResp)
	if !getResp.HasXML {
		t.Error("GetByID: expected HasXML to be true")
	}
}
