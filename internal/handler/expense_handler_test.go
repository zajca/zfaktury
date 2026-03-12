package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/testutil"
)

func setupExpenseRouter(t *testing.T) *chi.Mux {
	t.Helper()
	db := testutil.NewTestDB(t)
	expenseRepo := repository.NewExpenseRepository(db)
	expenseSvc := service.NewExpenseService(expenseRepo, nil)
	h := NewExpenseHandler(expenseSvc)

	r := chi.NewRouter()
	r.Mount("/api/v1/expenses", h.Routes())
	return r
}

func expenseBody() string {
	return `{
		"description": "Office supplies",
		"amount": 50000,
		"issue_date": "2026-03-01",
		"currency_code": "CZK",
		"category": "supplies",
		"business_percent": 100,
		"payment_method": "bank_transfer",
		"is_tax_deductible": true
	}`
}

func TestExpenseHandler_Create(t *testing.T) {
	r := setupExpenseRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewBufferString(expenseBody()))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp expenseResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.Description != "Office supplies" {
		t.Errorf("Description = %q, want %q", resp.Description, "Office supplies")
	}
}

func TestExpenseHandler_Create_InvalidBody(t *testing.T) {
	r := setupExpenseRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestExpenseHandler_Create_MissingDescription(t *testing.T) {
	r := setupExpenseRouter(t)

	body := `{"description":"","amount":50000,"issue_date":"2026-03-01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestExpenseHandler_Create_ZeroAmount(t *testing.T) {
	r := setupExpenseRouter(t)

	body := `{"description":"No amount","amount":0,"issue_date":"2026-03-01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func createTestExpense(t *testing.T, r *chi.Mux) expenseResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewBufferString(expenseBody()))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("creating expense: status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp expenseResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return resp
}

func TestExpenseHandler_GetByID(t *testing.T) {
	r := setupExpenseRouter(t)
	created := createTestExpense(t, r)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/expenses/%d", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp expenseResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Description != "Office supplies" {
		t.Errorf("Description = %q, want %q", resp.Description, "Office supplies")
	}
}

func TestExpenseHandler_GetByID_NotFound(t *testing.T) {
	r := setupExpenseRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/expenses/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestExpenseHandler_GetByID_InvalidID(t *testing.T) {
	r := setupExpenseRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/expenses/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestExpenseHandler_List(t *testing.T) {
	r := setupExpenseRouter(t)

	createTestExpense(t, r)
	createTestExpense(t, r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/expenses", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp listResponse[expenseResponse]
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 2 {
		t.Errorf("Total = %d, want 2", resp.Total)
	}
}

func TestExpenseHandler_Update(t *testing.T) {
	r := setupExpenseRouter(t)
	created := createTestExpense(t, r)

	updateBody := `{
		"description": "Updated supplies",
		"amount": 75000,
		"issue_date": "2026-03-05",
		"business_percent": 80,
		"payment_method": "cash"
	}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/expenses/%d", created.ID), bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp expenseResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Description != "Updated supplies" {
		t.Errorf("Description = %q, want %q", resp.Description, "Updated supplies")
	}
}

func TestExpenseHandler_Delete(t *testing.T) {
	r := setupExpenseRouter(t)
	created := createTestExpense(t, r)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/expenses/%d", created.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Verify deleted.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/expenses/%d", created.ID), nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusNotFound {
		t.Errorf("after delete: status = %d, want %d", getW.Code, http.StatusNotFound)
	}
}

func TestExpenseHandler_Delete_NotFound(t *testing.T) {
	r := setupExpenseRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/expenses/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestExpenseHandler_Create_MissingIssueDate(t *testing.T) {
	r := setupExpenseRouter(t)

	body := `{"description":"Test","amount":50000,"issue_date":"","category":"supplies","business_percent":100,"payment_method":"cash","is_tax_deductible":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	var resp errorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp.Error, "issue_date is required") {
		t.Errorf("error = %q, want to contain %q", resp.Error, "issue_date is required")
	}
}

func TestExpenseHandler_MarkTaxReviewed(t *testing.T) {
	r := setupExpenseRouter(t)

	e1 := createTestExpense(t, r)
	e2 := createTestExpense(t, r)

	body := fmt.Sprintf(`{"ids":[%d,%d]}`, e1.ID, e2.ID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses/review", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify that expenses are now marked as tax reviewed.
	for _, id := range []int64{e1.ID, e2.ID} {
		getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/expenses/%d", id), nil)
		getW := httptest.NewRecorder()
		r.ServeHTTP(getW, getReq)

		var resp expenseResponse
		json.NewDecoder(getW.Body).Decode(&resp)
		if resp.TaxReviewedAt == nil {
			t.Errorf("expense %d: TaxReviewedAt is nil, expected non-nil after marking", id)
		}
	}
}

func TestExpenseHandler_MarkTaxReviewed_InvalidBody(t *testing.T) {
	r := setupExpenseRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses/review", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestExpenseHandler_UnmarkTaxReviewed(t *testing.T) {
	r := setupExpenseRouter(t)

	e1 := createTestExpense(t, r)
	e2 := createTestExpense(t, r)

	// First mark them as reviewed.
	markBody := fmt.Sprintf(`{"ids":[%d,%d]}`, e1.ID, e2.ID)
	markReq := httptest.NewRequest(http.MethodPost, "/api/v1/expenses/review", bytes.NewBufferString(markBody))
	markReq.Header.Set("Content-Type", "application/json")
	markW := httptest.NewRecorder()
	r.ServeHTTP(markW, markReq)

	if markW.Code != http.StatusNoContent {
		t.Fatalf("mark: status = %d, want %d, body = %s", markW.Code, http.StatusNoContent, markW.Body.String())
	}

	// Now unmark them.
	unmarkBody := fmt.Sprintf(`{"ids":[%d,%d]}`, e1.ID, e2.ID)
	unmarkReq := httptest.NewRequest(http.MethodPost, "/api/v1/expenses/unreview", bytes.NewBufferString(unmarkBody))
	unmarkReq.Header.Set("Content-Type", "application/json")
	unmarkW := httptest.NewRecorder()
	r.ServeHTTP(unmarkW, unmarkReq)

	if unmarkW.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d, body = %s", unmarkW.Code, http.StatusNoContent, unmarkW.Body.String())
	}

	// Verify that expenses are no longer marked as tax reviewed.
	for _, id := range []int64{e1.ID, e2.ID} {
		getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/expenses/%d", id), nil)
		getW := httptest.NewRecorder()
		r.ServeHTTP(getW, getReq)

		var resp expenseResponse
		json.NewDecoder(getW.Body).Decode(&resp)
		if resp.TaxReviewedAt != nil {
			t.Errorf("expense %d: TaxReviewedAt = %v, expected nil after unmarking", id, *resp.TaxReviewedAt)
		}
	}
}

func TestExpenseHandler_UnmarkTaxReviewed_InvalidBody(t *testing.T) {
	r := setupExpenseRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses/unreview", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestExpenseHandler_Update_InvalidID(t *testing.T) {
	r := setupExpenseRouter(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/expenses/abc", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestExpenseHandler_Update_InvalidBody(t *testing.T) {
	r := setupExpenseRouter(t)
	created := createTestExpense(t, r)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/expenses/%d", created.ID), bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestExpenseHandler_Delete_InvalidID(t *testing.T) {
	r := setupExpenseRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/expenses/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestExpenseHandler_List_WithFilters(t *testing.T) {
	r := setupExpenseRouter(t)
	createTestExpense(t, r)

	// Filter by category.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/expenses?category=supplies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("category filter: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Filter by date range.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/expenses?date_from=2026-01-01&date_to=2026-12-31", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("date filter: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Filter by search.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/expenses?search=supplies", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("search filter: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Filter by tax_reviewed.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/expenses?tax_reviewed=false", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("tax_reviewed filter: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Pagination.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/expenses?limit=5&offset=0", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("pagination: status = %d, want %d", w.Code, http.StatusOK)
	}
}
