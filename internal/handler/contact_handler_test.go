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

func setupContactRouter(t *testing.T) *chi.Mux {
	t.Helper()
	db := testutil.NewTestDB(t)
	contactRepo := repository.NewContactRepository(db)
	contactSvc := service.NewContactService(contactRepo, nil, nil)
	h := NewContactHandler(contactSvc)

	r := chi.NewRouter()
	r.Mount("/api/v1/contacts", h.Routes())
	return r
}

func createTestContact(t *testing.T, r *chi.Mux, name string) contactResponse {
	t.Helper()
	body := fmt.Sprintf(`{"type":"company","name":%q}`, name)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("creating contact: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp contactResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return resp
}

func TestContactHandler_Create(t *testing.T) {
	r := setupContactRouter(t)

	body := `{"type":"company","name":"Test Corp","ico":"12345678"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp contactResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.Name != "Test Corp" {
		t.Errorf("Name = %q, want %q", resp.Name, "Test Corp")
	}
}

func TestContactHandler_Create_InvalidBody(t *testing.T) {
	r := setupContactRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestContactHandler_Create_MissingName(t *testing.T) {
	r := setupContactRouter(t)

	body := `{"type":"company","name":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestContactHandler_List(t *testing.T) {
	r := setupContactRouter(t)

	createTestContact(t, r, "Listed Corp")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp listResponse[contactResponse]
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("Total = %d, want 1", resp.Total)
	}
	if len(resp.Data) != 1 {
		t.Errorf("len(Data) = %d, want 1", len(resp.Data))
	}
}

func TestContactHandler_GetByID(t *testing.T) {
	r := setupContactRouter(t)
	created := createTestContact(t, r, "Get Me")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/contacts/%d", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp contactResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "Get Me" {
		t.Errorf("Name = %q, want %q", resp.Name, "Get Me")
	}
}

func TestContactHandler_GetByID_NotFound(t *testing.T) {
	r := setupContactRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestContactHandler_GetByID_InvalidID(t *testing.T) {
	r := setupContactRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestContactHandler_Update(t *testing.T) {
	r := setupContactRouter(t)
	created := createTestContact(t, r, "Before")

	updateBody := `{"type":"company","name":"After","email":"after@test.cz"}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/contacts/%d", created.ID), bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp contactResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Name != "After" {
		t.Errorf("Name = %q, want %q", resp.Name, "After")
	}
}

func TestContactHandler_Delete(t *testing.T) {
	r := setupContactRouter(t)
	created := createTestContact(t, r, "To Delete")

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/contacts/%d", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Verify deleted.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/contacts/%d", created.ID), nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusNotFound {
		t.Errorf("after delete: status = %d, want %d", getW.Code, http.StatusNotFound)
	}
}

func TestContactHandler_Delete_NotFound(t *testing.T) {
	r := setupContactRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/contacts/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestContactHandler_List_WithSearch(t *testing.T) {
	r := setupContactRouter(t)

	createTestContact(t, r, "Alpha Corp")
	createTestContact(t, r, "Beta Inc")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts?search=Alpha", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp listResponse[contactResponse]
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 1 {
		t.Errorf("Total = %d, want 1", resp.Total)
	}
}

func TestContactHandler_Update_InvalidID(t *testing.T) {
	r := setupContactRouter(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/contacts/abc", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestContactHandler_Update_InvalidBody(t *testing.T) {
	r := setupContactRouter(t)
	created := createTestContact(t, r, "Update Invalid")

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/contacts/%d", created.ID), bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestContactHandler_Update_ValidationError(t *testing.T) {
	r := setupContactRouter(t)
	created := createTestContact(t, r, "Update Validation")

	// Empty name should fail validation.
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/contacts/%d", created.ID), bytes.NewBufferString(`{"type":"company","name":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

func TestContactHandler_Delete_InvalidID(t *testing.T) {
	r := setupContactRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/contacts/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestContactHandler_List_WithTypeFilter(t *testing.T) {
	r := setupContactRouter(t)

	createTestContact(t, r, "Company Contact")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts?type=company", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp listResponse[contactResponse]
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 1 {
		t.Errorf("Total = %d, want 1", resp.Total)
	}
}

func TestContactHandler_List_WithPagination(t *testing.T) {
	r := setupContactRouter(t)

	createTestContact(t, r, "Page1")
	createTestContact(t, r, "Page2")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts?limit=1&offset=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp listResponse[contactResponse]
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Data) != 1 {
		t.Errorf("len(Data) = %d, want 1", len(resp.Data))
	}
	if resp.Total != 2 {
		t.Errorf("Total = %d, want 2", resp.Total)
	}
}

func TestContactHandler_List_WithFavoriteFilter(t *testing.T) {
	r := setupContactRouter(t)

	createTestContact(t, r, "Fav Test")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts?favorite=true", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
