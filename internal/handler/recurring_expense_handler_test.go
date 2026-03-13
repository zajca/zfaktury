package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/testutil"
)

func setupRecurringExpenseRouter(t *testing.T) *chi.Mux {
	db := testutil.NewTestDB(t)
	expenseRepo := repository.NewExpenseRepository(db)
	recurringRepo := repository.NewRecurringExpenseRepository(db)

	expenseSvc := service.NewExpenseService(expenseRepo, nil)
	recurringExpenseSvc := service.NewRecurringExpenseService(recurringRepo, expenseSvc, nil)

	h := NewRecurringExpenseHandler(recurringExpenseSvc)

	r := chi.NewRouter()
	r.Mount("/api/v1/recurring-expenses", h.Routes())
	return r
}

func validRecurringExpenseBody() map[string]any {
	return map[string]any{
		"name":             "Monthly rent",
		"description":      "Office rent",
		"amount":           1500000,
		"frequency":        "monthly",
		"next_issue_date":  "2026-04-01",
		"currency_code":    "CZK",
		"business_percent": 100,
		"payment_method":   "bank_transfer",
		"is_active":        true,
	}
}

func createRecurringExpense(t *testing.T, r *chi.Mux) int64 {
	t.Helper()
	body, _ := json.Marshal(validRecurringExpenseBody())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return int64(resp["id"].(float64))
}

func TestRecurringExpenseCreate(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	body, _ := json.Marshal(validRecurringExpenseBody())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp["name"] != "Monthly rent" {
		t.Errorf("expected name 'Monthly rent', got %v", resp["name"])
	}
	if resp["next_issue_date"] != "2026-04-01" {
		t.Errorf("expected next_issue_date '2026-04-01', got %v", resp["next_issue_date"])
	}
	if int64(resp["amount"].(float64)) != 1500000 {
		t.Errorf("expected amount 1500000, got %v", resp["amount"])
	}
}

func TestRecurringExpenseCreateInvalidJSON(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses", bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestRecurringExpenseCreateMissingName(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	body := validRecurringExpenseBody()
	delete(body, "name")
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRecurringExpenseList(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	// Create two recurring expenses.
	createRecurringExpense(t, r)
	createRecurringExpense(t, r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recurring-expenses", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	data, ok := resp["data"].([]any)
	if !ok {
		t.Fatalf("expected 'data' array in response")
	}
	if len(data) < 2 {
		t.Errorf("expected at least 2 items, got %d", len(data))
	}
	if resp["total"] == nil {
		t.Error("expected 'total' in response")
	}
	if resp["limit"] == nil {
		t.Error("expected 'limit' in response")
	}
	if resp["offset"] == nil {
		t.Error("expected 'offset' in response")
	}
}

func TestRecurringExpenseGetByID(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	id := createRecurringExpense(t, r)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/recurring-expenses/%d", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if int64(resp["id"].(float64)) != id {
		t.Errorf("expected id %d, got %v", id, resp["id"])
	}
}

func TestRecurringExpenseGetByIDNotFound(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recurring-expenses/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestRecurringExpenseUpdate(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	id := createRecurringExpense(t, r)

	updated := validRecurringExpenseBody()
	updated["name"] = "Updated rent"
	updated["amount"] = 2000000
	data, _ := json.Marshal(updated)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/recurring-expenses/%d", id), bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp["name"] != "Updated rent" {
		t.Errorf("expected name 'Updated rent', got %v", resp["name"])
	}
	if int64(resp["amount"].(float64)) != 2000000 {
		t.Errorf("expected amount 2000000, got %v", resp["amount"])
	}
}

func TestRecurringExpenseDelete(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	id := createRecurringExpense(t, r)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/recurring-expenses/%d", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRecurringExpenseDeleteNotFound(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/recurring-expenses/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRecurringExpenseActivate(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	id := createRecurringExpense(t, r)

	// Deactivate first, then activate.
	deactReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/recurring-expenses/%d/deactivate", id), nil)
	deactW := httptest.NewRecorder()
	r.ServeHTTP(deactW, deactReq)
	if deactW.Code != http.StatusNoContent {
		t.Fatalf("deactivate: expected 204, got %d", deactW.Code)
	}

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/recurring-expenses/%d/activate", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRecurringExpenseDeactivate(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	id := createRecurringExpense(t, r)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/recurring-expenses/%d/deactivate", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRecurringExpenseGeneratePending(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	// Create a recurring expense with next_issue_date in the past (today).
	body := validRecurringExpenseBody()
	body["next_issue_date"] = time.Now().Format("2006-01-02")
	data, _ := json.Marshal(body)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses", bytes.NewReader(data))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", createW.Code, createW.Body.String())
	}

	// Generate pending expenses as of today.
	genBody, _ := json.Marshal(map[string]string{
		"as_of_date": time.Now().Format("2006-01-02"),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses/generate", bytes.NewReader(genBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	generated := int(resp["generated"].(float64))
	if generated < 1 {
		t.Errorf("expected at least 1 generated expense, got %d", generated)
	}
}

func TestRecurringExpenseUpdateAllFields(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	id := createRecurringExpense(t, r)

	endDate := "2027-12-31"
	updated := map[string]any{
		"name":              "Updated rent",
		"description":       "Updated office rent",
		"amount":            2500000,
		"currency_code":     "EUR",
		"exchange_rate":     2500,
		"vat_rate_percent":  21,
		"vat_amount":        525000,
		"is_tax_deductible": true,
		"business_percent":  80,
		"payment_method":    "cash",
		"notes":             "Some notes",
		"frequency":         "quarterly",
		"next_issue_date":   "2026-07-01",
		"end_date":          endDate,
		"is_active":         true,
		"category":          "office",
	}
	data, _ := json.Marshal(updated)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/recurring-expenses/%d", id), bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if resp["name"] != "Updated rent" {
		t.Errorf("expected name 'Updated rent', got %v", resp["name"])
	}
	if int64(resp["amount"].(float64)) != 2500000 {
		t.Errorf("expected amount 2500000, got %v", resp["amount"])
	}
	if resp["currency_code"] != "EUR" {
		t.Errorf("expected currency_code 'EUR', got %v", resp["currency_code"])
	}
	if int64(resp["exchange_rate"].(float64)) != 2500 {
		t.Errorf("expected exchange_rate 2500, got %v", resp["exchange_rate"])
	}
	if int(resp["vat_rate_percent"].(float64)) != 21 {
		t.Errorf("expected vat_rate_percent 21, got %v", resp["vat_rate_percent"])
	}
	if int64(resp["vat_amount"].(float64)) != 525000 {
		t.Errorf("expected vat_amount 525000, got %v", resp["vat_amount"])
	}
	if resp["is_tax_deductible"] != true {
		t.Errorf("expected is_tax_deductible true, got %v", resp["is_tax_deductible"])
	}
	if int(resp["business_percent"].(float64)) != 80 {
		t.Errorf("expected business_percent 80, got %v", resp["business_percent"])
	}
	if resp["payment_method"] != "cash" {
		t.Errorf("expected payment_method 'cash', got %v", resp["payment_method"])
	}
	if resp["notes"] != "Some notes" {
		t.Errorf("expected notes 'Some notes', got %v", resp["notes"])
	}
	if resp["frequency"] != "quarterly" {
		t.Errorf("expected frequency 'quarterly', got %v", resp["frequency"])
	}
	if resp["next_issue_date"] != "2026-07-01" {
		t.Errorf("expected next_issue_date '2026-07-01', got %v", resp["next_issue_date"])
	}
	if resp["end_date"] != "2027-12-31" {
		t.Errorf("expected end_date '2027-12-31', got %v", resp["end_date"])
	}
	if resp["category"] != "office" {
		t.Errorf("expected category 'office', got %v", resp["category"])
	}

	// Verify by GET that update persisted.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/recurring-expenses/%d", id), nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET after update: expected 200, got %d", getW.Code)
	}
	var getResp map[string]any
	json.Unmarshal(getW.Body.Bytes(), &getResp)
	if getResp["name"] != "Updated rent" {
		t.Errorf("GET: expected name 'Updated rent', got %v", getResp["name"])
	}
}

func TestRecurringExpenseUpdateInvalidJSON(t *testing.T) {
	r := setupRecurringExpenseRouter(t)
	id := createRecurringExpense(t, r)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/recurring-expenses/%d", id), bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestRecurringExpenseUpdateInvalidID(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	body, _ := json.Marshal(validRecurringExpenseBody())
	req := httptest.NewRequest(http.MethodPut, "/api/v1/recurring-expenses/abc", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestRecurringExpenseUpdateInvalidNextIssueDate(t *testing.T) {
	r := setupRecurringExpenseRouter(t)
	id := createRecurringExpense(t, r)

	body := validRecurringExpenseBody()
	body["next_issue_date"] = "not-a-date"
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/recurring-expenses/%d", id), bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRecurringExpenseActivateNotFound(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	// Activate on non-existent ID -- service does not return error, returns 204.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses/99999/activate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRecurringExpenseActivateInvalidID(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses/abc/activate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestRecurringExpenseDeactivateNotFound(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses/99999/deactivate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRecurringExpenseDeactivateInvalidID(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses/abc/deactivate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestRecurringExpenseActivateVerifyState(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	id := createRecurringExpense(t, r)

	// Deactivate.
	deactReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/recurring-expenses/%d/deactivate", id), nil)
	deactW := httptest.NewRecorder()
	r.ServeHTTP(deactW, deactReq)
	if deactW.Code != http.StatusNoContent {
		t.Fatalf("deactivate: expected 204, got %d", deactW.Code)
	}

	// Verify deactivated.
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/recurring-expenses/%d", id), nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	var resp map[string]any
	json.Unmarshal(getW.Body.Bytes(), &resp)
	if resp["is_active"] != false {
		t.Errorf("expected is_active false after deactivate, got %v", resp["is_active"])
	}

	// Activate.
	actReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/recurring-expenses/%d/activate", id), nil)
	actW := httptest.NewRecorder()
	r.ServeHTTP(actW, actReq)
	if actW.Code != http.StatusNoContent {
		t.Fatalf("activate: expected 204, got %d: %s", actW.Code, actW.Body.String())
	}

	// Verify activated.
	getReq2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/recurring-expenses/%d", id), nil)
	getW2 := httptest.NewRecorder()
	r.ServeHTTP(getW2, getReq2)
	var resp2 map[string]any
	json.Unmarshal(getW2.Body.Bytes(), &resp2)
	if resp2["is_active"] != true {
		t.Errorf("expected is_active true after activate, got %v", resp2["is_active"])
	}
}

func TestRecurringExpenseGeneratePendingInvalidJSON(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses/generate", bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestRecurringExpenseGeneratePendingInvalidDate(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	genBody, _ := json.Marshal(map[string]string{
		"as_of_date": "not-a-date",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses/generate", bytes.NewReader(genBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestRecurringExpenseGeneratePendingFutureDate(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	futureDate := time.Now().AddDate(0, 0, 10).Format("2006-01-02")
	genBody, _ := json.Marshal(map[string]string{
		"as_of_date": futureDate,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses/generate", bytes.NewReader(genBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRecurringExpenseGeneratePendingOldDate(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	oldDate := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	genBody, _ := json.Marshal(map[string]string{
		"as_of_date": oldDate,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses/generate", bytes.NewReader(genBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRecurringExpenseGeneratePendingEmptyDate(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	// Create a recurring expense with next_issue_date = today so generation will find it.
	body := validRecurringExpenseBody()
	body["next_issue_date"] = time.Now().Format("2006-01-02")
	data, _ := json.Marshal(body)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses", bytes.NewReader(data))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	r.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", createW.Code, createW.Body.String())
	}

	// Send empty as_of_date -- should default to today.
	genBody, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses/generate", bytes.NewReader(genBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["generated"] == nil {
		t.Error("expected 'generated' field in response")
	}
}

func TestRecurringExpenseCreateWithEndDate(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	body := validRecurringExpenseBody()
	endDate := "2027-12-31"
	body["end_date"] = endDate
	body["vat_rate_percent"] = 21
	body["vat_amount"] = 315000
	body["is_tax_deductible"] = true
	body["category"] = "office"
	body["notes"] = "Test notes"
	body["exchange_rate"] = 100

	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["end_date"] != "2027-12-31" {
		t.Errorf("expected end_date '2027-12-31', got %v", resp["end_date"])
	}
	if resp["category"] != "office" {
		t.Errorf("expected category 'office', got %v", resp["category"])
	}
	if resp["notes"] != "Test notes" {
		t.Errorf("expected notes 'Test notes', got %v", resp["notes"])
	}
	if int(resp["vat_rate_percent"].(float64)) != 21 {
		t.Errorf("expected vat_rate_percent 21, got %v", resp["vat_rate_percent"])
	}
	if resp["is_tax_deductible"] != true {
		t.Errorf("expected is_tax_deductible true, got %v", resp["is_tax_deductible"])
	}
}

func TestRecurringExpenseCreateInvalidEndDate(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	body := validRecurringExpenseBody()
	body["end_date"] = "not-a-date"
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRecurringExpenseCreateMissingNextIssueDate(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	body := validRecurringExpenseBody()
	delete(body, "next_issue_date")
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRecurringExpenseCreateInvalidNextIssueDate(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	body := validRecurringExpenseBody()
	body["next_issue_date"] = "invalid-date"
	data, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recurring-expenses", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRecurringExpenseGetByIDInvalidID(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recurring-expenses/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestRecurringExpenseDeleteInvalidID(t *testing.T) {
	r := setupRecurringExpenseRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/recurring-expenses/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}
