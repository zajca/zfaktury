package handler

import (
	"bytes"
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

func setupSequenceRouter(t *testing.T) *chi.Mux {
	t.Helper()
	db := testutil.NewTestDB(t)
	sequenceRepo := repository.NewSequenceRepository(db)
	sequenceSvc := service.NewSequenceService(sequenceRepo, nil)
	sequenceHandler := NewSequenceHandler(sequenceSvc)

	r := chi.NewRouter()
	r.Mount("/api/v1/invoice-sequences", sequenceHandler.Routes())
	return r
}

func TestSequenceHandler_Create(t *testing.T) {
	r := setupSequenceRouter(t)

	body := `{"prefix":"FV","year":2026,"next_number":1,"format_pattern":"{prefix}{year}{number:04d}"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoice-sequences", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp sequenceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.Prefix != "FV" {
		t.Errorf("Prefix = %q, want %q", resp.Prefix, "FV")
	}
	if resp.Preview == "" {
		t.Error("expected non-empty Preview")
	}
	if resp.Preview != "FV20260001" {
		t.Errorf("Preview = %q, want %q", resp.Preview, "FV20260001")
	}
}

func TestSequenceHandler_Create_InvalidBody(t *testing.T) {
	r := setupSequenceRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoice-sequences", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSequenceHandler_Create_MissingPrefix(t *testing.T) {
	r := setupSequenceRouter(t)

	body := `{"year":2026}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoice-sequences", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func createTestSequence(t *testing.T, r *chi.Mux, prefix string, year int) sequenceResponse {
	t.Helper()
	body := fmt.Sprintf(`{"prefix":"%s","year":%d,"next_number":1}`, prefix, year)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoice-sequences", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("creating sequence: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp sequenceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return resp
}

func TestSequenceHandler_List(t *testing.T) {
	r := setupSequenceRouter(t)

	createTestSequence(t, r, "FV", 2026)
	createTestSequence(t, r, "ZF", 2026)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoice-sequences", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp []sequenceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 2 {
		t.Errorf("len(sequences) = %d, want 2", len(resp))
	}
}

func TestSequenceHandler_GetByID(t *testing.T) {
	r := setupSequenceRouter(t)
	created := createTestSequence(t, r, "FV", 2026)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/invoice-sequences/%d", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp sequenceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID != created.ID {
		t.Errorf("ID = %d, want %d", resp.ID, created.ID)
	}
}

func TestSequenceHandler_GetByID_NotFound(t *testing.T) {
	r := setupSequenceRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoice-sequences/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestSequenceHandler_GetByID_InvalidID(t *testing.T) {
	r := setupSequenceRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoice-sequences/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSequenceHandler_Update(t *testing.T) {
	r := setupSequenceRouter(t)
	created := createTestSequence(t, r, "FV", 2026)

	updateBody := fmt.Sprintf(`{"prefix":"FV","year":2026,"next_number":5,"format_pattern":"%s"}`, created.FormatPattern)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/invoice-sequences/%d", created.ID), bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp sequenceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.NextNumber != 5 {
		t.Errorf("NextNumber = %d, want 5", resp.NextNumber)
	}
}

func TestSequenceHandler_Delete(t *testing.T) {
	r := setupSequenceRouter(t)
	created := createTestSequence(t, r, "FV", 2026)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/invoice-sequences/%d", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusNoContent, w.Body.String())
	}
}

func TestSequenceHandler_Delete_InvalidID(t *testing.T) {
	r := setupSequenceRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/invoice-sequences/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSequenceHandler_Update_InvalidID(t *testing.T) {
	r := setupSequenceRouter(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/invoice-sequences/abc", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSequenceHandler_Update_InvalidBody(t *testing.T) {
	r := setupSequenceRouter(t)
	created := createTestSequence(t, r, "FV", 2026)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/invoice-sequences/%d", created.ID), bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSequenceHandler_Delete_NotFound(t *testing.T) {
	r := setupSequenceRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/invoice-sequences/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestSequenceHandler_List_WithMultiple(t *testing.T) {
	r := setupSequenceRouter(t)

	createTestSequence(t, r, "FV", 2026)
	createTestSequence(t, r, "PF", 2026)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/invoice-sequences", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp []sequenceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) < 2 {
		t.Errorf("expected at least 2 sequences, got %d", len(resp))
	}
}
