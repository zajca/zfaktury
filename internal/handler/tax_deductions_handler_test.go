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

func setupTaxDeductionsRouter(t *testing.T) *chi.Mux {
	t.Helper()
	db := testutil.NewTestDB(t)

	spouseRepo := repository.NewTaxSpouseCreditRepository(db)
	childRepo := repository.NewTaxChildCreditRepository(db)
	personalRepo := repository.NewTaxPersonalCreditsRepository(db)
	deductionRepo := repository.NewTaxDeductionRepository(db)
	docRepo := repository.NewTaxDeductionDocumentRepository(db)

	creditsSvc := service.NewTaxCreditsService(spouseRepo, childRepo, personalRepo, deductionRepo, nil)
	docSvc := service.NewTaxDeductionDocumentService(docRepo, deductionRepo, t.TempDir(), nil)
	h := NewTaxDeductionsHandler(creditsSvc, docSvc, nil)

	r := chi.NewRouter()
	r.Mount("/api/v1/tax-deductions", h.Routes())
	return r
}

// --- GET /{year} (List) ---

func TestTaxDeductions_List_Empty(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tax-deductions/2025", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []taxDeductionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected 0 deductions, got %d", len(resp))
	}
}

func TestTaxDeductions_List_InvalidYear(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tax-deductions/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxDeductions_List_WithData(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	// Create two deductions.
	bodies := []string{
		`{"category":"mortgage","description":"Hypoteka","claimed_amount":500000}`,
		`{"category":"pension","description":"Penzijni sporeni","claimed_amount":200000}`,
	}
	for i, body := range bodies {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-deductions/2025", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create deduction %d: status = %d, want %d, body: %s", i, w.Code, http.StatusCreated, w.Body.String())
		}
	}

	// List.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tax-deductions/2025", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []taxDeductionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 deductions, got %d", len(resp))
	}
}

func TestTaxDeductions_List_DifferentYearsIsolated(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	// Create deduction in 2024.
	body := `{"category":"mortgage","description":"Hypoteka","claimed_amount":500000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-deductions/2024", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// List 2025 -- should be empty.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tax-deductions/2025", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp []taxDeductionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected 0 deductions for 2025, got %d", len(resp))
	}
}

// --- POST /{year} (Create) ---

func TestTaxDeductions_Create(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	body := `{"category":"mortgage","description":"Hypoteka","claimed_amount":500000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-deductions/2025", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp taxDeductionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.Year != 2025 {
		t.Errorf("Year = %d, want 2025", resp.Year)
	}
	if resp.Category != "mortgage" {
		t.Errorf("Category = %q, want %q", resp.Category, "mortgage")
	}
	if resp.Description != "Hypoteka" {
		t.Errorf("Description = %q, want %q", resp.Description, "Hypoteka")
	}
	if resp.ClaimedAmount != 500000 {
		t.Errorf("ClaimedAmount = %d, want 500000", resp.ClaimedAmount)
	}
	if resp.CreatedAt == "" {
		t.Error("expected non-empty CreatedAt")
	}
	if resp.UpdatedAt == "" {
		t.Error("expected non-empty UpdatedAt")
	}
}

func TestTaxDeductions_Create_AllCategories(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	categories := []string{"mortgage", "life_insurance", "pension", "donation", "union_dues"}

	for _, cat := range categories {
		t.Run(cat, func(t *testing.T) {
			body := fmt.Sprintf(`{"category":%q,"description":"Test","claimed_amount":100000}`, cat)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-deductions/2025", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				t.Errorf("category %q: status = %d, want %d, body: %s", cat, w.Code, http.StatusCreated, w.Body.String())
			}
		})
	}
}

func TestTaxDeductions_Create_InvalidYear(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	body := `{"category":"mortgage","description":"Hypoteka","claimed_amount":500000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-deductions/abc", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxDeductions_Create_InvalidCategory(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	body := `{"category":"invalid_category","description":"Test","claimed_amount":100000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-deductions/2025", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestTaxDeductions_Create_NegativeAmount(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	body := `{"category":"mortgage","description":"Hypoteka","claimed_amount":-100}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-deductions/2025", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestTaxDeductions_Create_InvalidJSON(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-deductions/2025", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxDeductions_Create_EmptyCategory(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	body := `{"category":"","description":"Test","claimed_amount":100000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-deductions/2025", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// --- PUT /{year}/{id} (Update) ---

func TestTaxDeductions_Update(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	// Create first.
	body := `{"category":"mortgage","description":"Hypoteka","claimed_amount":500000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-deductions/2025", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var created taxDeductionResponse
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	// Update.
	updateBody := `{"category":"pension","description":"Penzijni sporeni updated","claimed_amount":300000}`
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/tax-deductions/2025/%d", created.ID), bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update: status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var updated taxDeductionResponse
	if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if updated.ID != created.ID {
		t.Errorf("ID = %d, want %d", updated.ID, created.ID)
	}
	if updated.Category != "pension" {
		t.Errorf("Category = %q, want %q", updated.Category, "pension")
	}
	if updated.Description != "Penzijni sporeni updated" {
		t.Errorf("Description = %q, want %q", updated.Description, "Penzijni sporeni updated")
	}
	if updated.ClaimedAmount != 300000 {
		t.Errorf("ClaimedAmount = %d, want 300000", updated.ClaimedAmount)
	}
}

func TestTaxDeductions_Update_InvalidYear(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	body := `{"category":"mortgage","description":"Test","claimed_amount":100000}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-deductions/abc/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxDeductions_Update_InvalidID(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	body := `{"category":"mortgage","description":"Test","claimed_amount":100000}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-deductions/2025/abc", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxDeductions_Update_InvalidJSON(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-deductions/2025/1", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxDeductions_Update_InvalidCategory(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	// Create a valid deduction first.
	body := `{"category":"mortgage","description":"Test","claimed_amount":100000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-deductions/2025", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, want %d", w.Code, http.StatusCreated)
	}

	var created taxDeductionResponse
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	// Update with invalid category.
	updateBody := `{"category":"bogus","description":"Test","claimed_amount":100000}`
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/tax-deductions/2025/%d", created.ID), bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestTaxDeductions_Update_NegativeAmount(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	// Create a valid deduction first.
	body := `{"category":"mortgage","description":"Test","claimed_amount":100000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-deductions/2025", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, want %d", w.Code, http.StatusCreated)
	}

	var created taxDeductionResponse
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	// Update with negative amount.
	updateBody := `{"category":"mortgage","description":"Test","claimed_amount":-500}`
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/tax-deductions/2025/%d", created.ID), bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// --- DELETE /{year}/{id} ---

func TestTaxDeductions_Delete(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	// Create first.
	body := `{"category":"mortgage","description":"Hypoteka","claimed_amount":500000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-deductions/2025", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var created taxDeductionResponse
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	// Delete.
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/tax-deductions/2025/%d", created.ID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: status = %d, want %d, body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify gone via list.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tax-deductions/2025", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list after delete: status = %d, want %d", w.Code, http.StatusOK)
	}

	var deductions []taxDeductionResponse
	if err := json.NewDecoder(w.Body).Decode(&deductions); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(deductions) != 0 {
		t.Errorf("expected 0 deductions after delete, got %d", len(deductions))
	}
}

func TestTaxDeductions_Delete_InvalidID(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tax-deductions/2025/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxDeductions_Delete_NonExistent(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tax-deductions/2025/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

// --- Create + List round trip ---

func TestTaxDeductions_CreateAndList_RoundTrip(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	// Create a deduction.
	body := `{"category":"life_insurance","description":"Zivotni pojisteni","claimed_amount":240000}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-deductions/2025", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var created taxDeductionResponse
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	// List and verify the deduction appears.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tax-deductions/2025", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list: status = %d, want %d", w.Code, http.StatusOK)
	}

	var deductions []taxDeductionResponse
	if err := json.NewDecoder(w.Body).Decode(&deductions); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(deductions) != 1 {
		t.Fatalf("expected 1 deduction, got %d", len(deductions))
	}

	d := deductions[0]
	if d.ID != created.ID {
		t.Errorf("ID = %d, want %d", d.ID, created.ID)
	}
	if d.Category != "life_insurance" {
		t.Errorf("Category = %q, want %q", d.Category, "life_insurance")
	}
	if d.Description != "Zivotni pojisteni" {
		t.Errorf("Description = %q, want %q", d.Description, "Zivotni pojisteni")
	}
	if d.ClaimedAmount != 240000 {
		t.Errorf("ClaimedAmount = %d, want 240000", d.ClaimedAmount)
	}
}

// --- Zero amount is valid ---

func TestTaxDeductions_Create_ZeroAmount(t *testing.T) {
	r := setupTaxDeductionsRouter(t)

	body := `{"category":"mortgage","description":"Hypoteka","claimed_amount":0}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-deductions/2025", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp taxDeductionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.ClaimedAmount != 0 {
		t.Errorf("ClaimedAmount = %d, want 0", resp.ClaimedAmount)
	}
}
