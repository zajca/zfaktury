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

func setupVATControlRouter(t *testing.T) (*chi.Mux, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	vatControlRepo := repository.NewVATControlStatementRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	contactRepo := repository.NewContactRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)

	vatControlSvc := service.NewVATControlStatementService(vatControlRepo, invoiceRepo, expenseRepo, contactRepo)
	settingsSvc := service.NewSettingsService(settingsRepo)
	h := NewVATControlStatementHandler(vatControlSvc, settingsSvc)

	r := chi.NewRouter()
	r.Mount("/api/v1/vat-control-statements", h.Routes())
	return r, db
}

func createVATControlStatement(t *testing.T, r *chi.Mux, year, month int) int64 {
	t.Helper()
	body := fmt.Sprintf(`{"year":%d,"month":%d,"filing_type":"regular"}`, year, month)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vat-control-statements", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("createVATControlStatement: status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("createVATControlStatement: decode error: %v", err)
	}
	id, ok := resp["id"].(float64)
	if !ok {
		t.Fatalf("createVATControlStatement: id not found in response: %v", resp)
	}
	return int64(id)
}

func TestVATControl_Create_Valid(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	body := `{"year":2025,"month":3,"filing_type":"regular"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vat-control-statements", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp controlStatementResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.Period.Year != 2025 {
		t.Errorf("Period.Year = %d, want 2025", resp.Period.Year)
	}
	if resp.Period.Month != 3 {
		t.Errorf("Period.Month = %d, want 3", resp.Period.Month)
	}
}

func TestVATControl_Create_InvalidJSON(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vat-control-statements", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestVATControl_Create_DuplicatePeriod(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	body := `{"year":2025,"month":5,"filing_type":"regular"}`
	// First create should succeed.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vat-control-statements", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("first create: status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// Second create for the same period should fail with 409.
	req = httptest.NewRequest(http.MethodPost, "/api/v1/vat-control-statements", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("duplicate create: status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestVATControl_List_WithYear(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	createVATControlStatement(t, r, 2025, 1)
	createVATControlStatement(t, r, 2025, 2)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vat-control-statements?year=2025", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []controlStatementResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(resp) < 2 {
		t.Errorf("expected at least 2 statements, got %d", len(resp))
	}
}

func TestVATControl_List_NoYear(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vat-control-statements", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestVATControl_GetByID(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	id := createVATControlStatement(t, r, 2025, 6)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vat-control-statements/%d", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp controlStatementResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.ID != id {
		t.Errorf("ID = %d, want %d", resp.ID, id)
	}
}

func TestVATControl_GetByID_NotFound(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vat-control-statements/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestVATControl_Delete(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	id := createVATControlStatement(t, r, 2025, 7)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/vat-control-statements/%d", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}
}

func TestVATControl_Delete_NotFound(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/vat-control-statements/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestVATControl_MarkFiled(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	id := createVATControlStatement(t, r, 2025, 8)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-control-statements/%d/mark-filed", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp controlStatementResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.FiledAt == nil {
		t.Error("expected FiledAt to be set after mark-filed")
	}
}

func TestVATControl_GenerateXML_NoDIC(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	id := createVATControlStatement(t, r, 2025, 9)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-control-statements/%d/generate-xml", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestVATControl_GenerateXML_WithDIC(t *testing.T) {
	r, db := setupVATControlRouter(t)

	// Insert DIC setting directly into the database.
	_, err := db.ExecContext(context.Background(), "INSERT INTO settings (key, value, updated_at) VALUES ('dic', 'CZ12345678', datetime('now'))")
	if err != nil {
		t.Fatalf("failed to insert DIC setting: %v", err)
	}

	id := createVATControlStatement(t, r, 2025, 10)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-control-statements/%d/generate-xml", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestVATControl_GenerateXML_InvalidID(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vat-control-statements/abc/generate-xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestVATControl_GenerateXML_NotFound(t *testing.T) {
	r, db := setupVATControlRouter(t)

	// Insert DIC to get past the DIC check.
	_, err := db.ExecContext(context.Background(), "INSERT INTO settings (key, value, updated_at) VALUES ('dic', 'CZ12345678', datetime('now'))")
	if err != nil {
		t.Fatalf("failed to insert DIC setting: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vat-control-statements/99999/generate-xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestVATControl_Recalculate_Success(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	id := createVATControlStatement(t, r, 2025, 11)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-control-statements/%d/recalculate", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp controlStatementResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.ID != id {
		t.Errorf("ID = %d, want %d", resp.ID, id)
	}
}

func TestVATControl_Recalculate_InvalidID(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vat-control-statements/abc/recalculate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestVATControl_Recalculate_NotFound(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vat-control-statements/99999/recalculate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestVATControl_Recalculate_Filed(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	id := createVATControlStatement(t, r, 2025, 12)

	// Mark as filed first.
	markReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-control-statements/%d/mark-filed", id), nil)
	markW := httptest.NewRecorder()
	r.ServeHTTP(markW, markReq)
	if markW.Code != http.StatusOK {
		t.Fatalf("mark-filed: status = %d, want %d", markW.Code, http.StatusOK)
	}

	// Recalculate a filed statement should fail.
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-control-statements/%d/recalculate", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestVATControl_DownloadXML_InvalidID(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vat-control-statements/abc/xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestVATControl_DownloadXML_NotFound(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vat-control-statements/99999/xml", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestVATControl_DownloadXML_NoXMLGenerated(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	id := createVATControlStatement(t, r, 2026, 1)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vat-control-statements/%d/xml", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestVATControl_DownloadXML_Success(t *testing.T) {
	r, db := setupVATControlRouter(t)

	// Insert DIC setting.
	_, err := db.ExecContext(context.Background(), "INSERT INTO settings (key, value, updated_at) VALUES ('dic', 'CZ12345678', datetime('now'))")
	if err != nil {
		t.Fatalf("failed to insert DIC setting: %v", err)
	}

	id := createVATControlStatement(t, r, 2026, 2)

	// Generate XML first.
	genReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-control-statements/%d/generate-xml", id), nil)
	genW := httptest.NewRecorder()
	r.ServeHTTP(genW, genReq)
	if genW.Code != http.StatusOK {
		t.Fatalf("generate-xml: status = %d, want %d, body: %s", genW.Code, http.StatusOK, genW.Body.String())
	}

	// Now download the XML.
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vat-control-statements/%d/xml", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/xml" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/xml")
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty XML body")
	}
}

func TestVATControl_GetByID_InvalidID(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vat-control-statements/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestVATControl_Delete_InvalidID(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/vat-control-statements/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestVATControl_Delete_Filed(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	id := createVATControlStatement(t, r, 2026, 3)

	// Mark as filed.
	markReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-control-statements/%d/mark-filed", id), nil)
	markW := httptest.NewRecorder()
	r.ServeHTTP(markW, markReq)
	if markW.Code != http.StatusOK {
		t.Fatalf("mark-filed: status = %d", markW.Code)
	}

	// Delete a filed statement should fail.
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/vat-control-statements/%d", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestVATControl_MarkFiled_InvalidID(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vat-control-statements/abc/mark-filed", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestVATControl_MarkFiled_NotFound(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vat-control-statements/99999/mark-filed", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestVATControl_List_InvalidYear(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vat-control-statements?year=notanumber", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestVATControl_Create_InvalidInput(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	// Month 0 should be invalid input.
	body := `{"year":2025,"month":0,"filing_type":"regular"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vat-control-statements", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestVATControl_List_EmptyResult(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vat-control-statements?year=1999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp []controlStatementResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	// nil or empty list is fine for no results.
}

func TestVATControl_Recalculate_SuccessWithLines(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	id := createVATControlStatement(t, r, 2026, 4)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-control-statements/%d/recalculate", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp controlStatementResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.ID != id {
		t.Errorf("ID = %d, want %d", resp.ID, id)
	}
}

func TestVATControl_MarkFiled_Verify(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	id := createVATControlStatement(t, r, 2026, 5)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-control-statements/%d/mark-filed", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp controlStatementResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Status != "filed" {
		t.Errorf("Status = %q, want %q", resp.Status, "filed")
	}
	if resp.FiledAt == nil {
		t.Error("expected FiledAt to be set")
	}

	// Verify GetByID after filing shows correct status.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vat-control-statements/%d", id), nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("get after mark-filed: status = %d", getW.Code)
	}
	var getResp controlStatementResponse
	json.NewDecoder(getW.Body).Decode(&getResp)
	if getResp.Status != "filed" {
		t.Errorf("after mark-filed GetByID: Status = %q, want %q", getResp.Status, "filed")
	}
}

func TestVATControl_GetByID_WithLines(t *testing.T) {
	r, _ := setupVATControlRouter(t)

	id := createVATControlStatement(t, r, 2026, 6)

	// Recalculate to populate lines.
	recalcReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-control-statements/%d/recalculate", id), nil)
	recalcW := httptest.NewRecorder()
	r.ServeHTTP(recalcW, recalcReq)
	if recalcW.Code != http.StatusOK {
		t.Fatalf("recalculate: status = %d", recalcW.Code)
	}

	// GetByID should return the statement with lines.
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vat-control-statements/%d", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp controlStatementResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.ID != id {
		t.Errorf("ID = %d, want %d", resp.ID, id)
	}
}

func TestVATControl_GenerateXML_WithDIC_VerifyResponse(t *testing.T) {
	r, db := setupVATControlRouter(t)

	_, err := db.ExecContext(context.Background(), "INSERT INTO settings (key, value, updated_at) VALUES ('dic', 'CZ99999999', datetime('now'))")
	if err != nil {
		t.Fatalf("failed to insert DIC setting: %v", err)
	}

	id := createVATControlStatement(t, r, 2026, 7)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/vat-control-statements/%d/generate-xml", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify XML was generated by downloading.
	dlReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/vat-control-statements/%d/xml", id), nil)
	dlW := httptest.NewRecorder()
	r.ServeHTTP(dlW, dlReq)

	if dlW.Code != http.StatusOK {
		t.Fatalf("download: status = %d, want %d", dlW.Code, http.StatusOK)
	}
	if dlW.Body.Len() == 0 {
		t.Error("expected non-empty XML body")
	}
}
