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

func setupTaxCreditsRouter(t *testing.T) *chi.Mux {
	t.Helper()
	db := testutil.NewTestDB(t)

	spouseRepo := repository.NewTaxSpouseCreditRepository(db)
	childRepo := repository.NewTaxChildCreditRepository(db)
	personalRepo := repository.NewTaxPersonalCreditsRepository(db)
	deductionRepo := repository.NewTaxDeductionRepository(db)

	svc := service.NewTaxCreditsService(spouseRepo, childRepo, personalRepo, deductionRepo, nil)
	h := NewTaxCreditsHandler(svc)

	r := chi.NewRouter()
	r.Mount("/api/v1/tax-credits", h.Routes())
	return r
}

// --- GET /{year} (Summary) ---

func TestTaxCredits_GetSummary_Empty(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tax-credits/2025", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp taxCreditsSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Year != 2025 {
		t.Errorf("Year = %d, want 2025", resp.Year)
	}
	if resp.Spouse != nil {
		t.Error("expected Spouse to be nil for empty year")
	}
	if len(resp.Children) != 0 {
		t.Errorf("expected 0 children, got %d", len(resp.Children))
	}
	if resp.Personal != nil {
		t.Error("expected Personal to be nil for empty year")
	}
	if resp.TotalCredits != 0 {
		t.Errorf("TotalCredits = %d, want 0", resp.TotalCredits)
	}
	if resp.TotalChildBenefit != 0 {
		t.Errorf("TotalChildBenefit = %d, want 0", resp.TotalChildBenefit)
	}
}

func TestTaxCredits_GetSummary_InvalidYear(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tax-credits/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxCredits_GetSummary_WithData(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	// Create spouse credit.
	spouseBody := `{"spouse_name":"Jana","spouse_birth_number":"8551011234","spouse_income":5000000,"spouse_ztp":false,"months_claimed":12}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/2025/spouse", bytes.NewBufferString(spouseBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upsert spouse: status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Create a child.
	childBody := `{"child_name":"Petr","birth_number":"1201011234","child_order":1,"months_claimed":12,"ztp":false}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tax-credits/2025/children", bytes.NewBufferString(childBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create child: status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// Create personal credits.
	personalBody := `{"is_student":true,"student_months":10,"disability_level":0}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/2025/personal", bytes.NewBufferString(personalBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upsert personal: status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Now get the summary.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tax-credits/2025", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("summary: status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp taxCreditsSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Year != 2025 {
		t.Errorf("Year = %d, want 2025", resp.Year)
	}
	if resp.Spouse == nil {
		t.Fatal("expected Spouse to be non-nil")
	}
	if resp.Spouse.SpouseName != "Jana" {
		t.Errorf("Spouse.SpouseName = %q, want %q", resp.Spouse.SpouseName, "Jana")
	}
	if len(resp.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(resp.Children))
	}
	if resp.Children[0].ChildName != "Petr" {
		t.Errorf("Children[0].ChildName = %q, want %q", resp.Children[0].ChildName, "Petr")
	}
	if resp.Personal == nil {
		t.Fatal("expected Personal to be non-nil")
	}
	if !resp.Personal.IsStudent {
		t.Error("expected Personal.IsStudent to be true")
	}
	if resp.TotalCredits <= 0 {
		t.Errorf("expected TotalCredits > 0, got %d", resp.TotalCredits)
	}
	if resp.TotalChildBenefit <= 0 {
		t.Errorf("expected TotalChildBenefit > 0, got %d", resp.TotalChildBenefit)
	}
}

// --- PUT /{year}/spouse (Upsert) ---

func TestTaxCredits_UpsertSpouse(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	body := `{"spouse_name":"Jana","spouse_birth_number":"8551011234","spouse_income":5000000,"spouse_ztp":false,"months_claimed":12}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/2025/spouse", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp taxSpouseCreditResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.SpouseName != "Jana" {
		t.Errorf("SpouseName = %q, want %q", resp.SpouseName, "Jana")
	}
	if resp.Year != 2025 {
		t.Errorf("Year = %d, want 2025", resp.Year)
	}
	if resp.MonthsClaimed != 12 {
		t.Errorf("MonthsClaimed = %d, want 12", resp.MonthsClaimed)
	}
	if resp.SpouseIncome != 5000000 {
		t.Errorf("SpouseIncome = %d, want 5000000", resp.SpouseIncome)
	}
}

func TestTaxCredits_UpsertSpouse_Update(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	// Create first.
	body := `{"spouse_name":"Jana","spouse_birth_number":"8551011234","spouse_income":5000000,"spouse_ztp":false,"months_claimed":12}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/2025/spouse", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first upsert: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Update with different name.
	body = `{"spouse_name":"Marie","spouse_birth_number":"8551011234","spouse_income":3000000,"spouse_ztp":true,"months_claimed":6}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/2025/spouse", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("second upsert: status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp taxSpouseCreditResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.SpouseName != "Marie" {
		t.Errorf("SpouseName = %q, want %q", resp.SpouseName, "Marie")
	}
	if resp.MonthsClaimed != 6 {
		t.Errorf("MonthsClaimed = %d, want 6", resp.MonthsClaimed)
	}
	if !resp.SpouseZTP {
		t.Error("expected SpouseZTP to be true")
	}
}

func TestTaxCredits_UpsertSpouse_InvalidYear(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	body := `{"spouse_name":"Jana","spouse_birth_number":"8551011234","spouse_income":5000000,"spouse_ztp":false,"months_claimed":12}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/abc/spouse", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxCredits_UpsertSpouse_MissingName(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	body := `{"spouse_name":"","spouse_birth_number":"8551011234","spouse_income":5000000,"spouse_ztp":false,"months_claimed":12}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/2025/spouse", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestTaxCredits_UpsertSpouse_MonthsOutOfRange(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	tests := []struct {
		name   string
		months int
	}{
		{"zero months", 0},
		{"13 months", 13},
		{"negative months", -1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := fmt.Sprintf(`{"spouse_name":"Jana","spouse_birth_number":"8551011234","spouse_income":5000000,"spouse_ztp":false,"months_claimed":%d}`, tc.months)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/2025/spouse", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("months=%d: status = %d, want %d, body: %s", tc.months, w.Code, http.StatusBadRequest, w.Body.String())
			}
		})
	}
}

func TestTaxCredits_UpsertSpouse_InvalidJSON(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/2025/spouse", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- DELETE /{year}/spouse ---

func TestTaxCredits_DeleteSpouse(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	// Create spouse first.
	body := `{"spouse_name":"Jana","spouse_birth_number":"8551011234","spouse_income":5000000,"spouse_ztp":false,"months_claimed":12}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/2025/spouse", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upsert: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Delete.
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/tax-credits/2025/spouse", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: status = %d, want %d, body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify gone via summary.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tax-credits/2025", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("summary after delete: status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp taxCreditsSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Spouse != nil {
		t.Error("expected Spouse to be nil after delete")
	}
}

func TestTaxCredits_DeleteSpouse_InvalidYear(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tax-credits/abc/spouse", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- POST /{year}/children (Create) ---

func TestTaxCredits_CreateChild(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	body := `{"child_name":"Petr","birth_number":"1201011234","child_order":1,"months_claimed":12,"ztp":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-credits/2025/children", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp taxChildCreditResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.ChildName != "Petr" {
		t.Errorf("ChildName = %q, want %q", resp.ChildName, "Petr")
	}
	if resp.Year != 2025 {
		t.Errorf("Year = %d, want 2025", resp.Year)
	}
	if resp.ChildOrder != 1 {
		t.Errorf("ChildOrder = %d, want 1", resp.ChildOrder)
	}
	if resp.MonthsClaimed != 12 {
		t.Errorf("MonthsClaimed = %d, want 12", resp.MonthsClaimed)
	}
}

func TestTaxCredits_CreateChild_InvalidYear(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	body := `{"child_name":"Petr","birth_number":"1201011234","child_order":1,"months_claimed":12,"ztp":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-credits/abc/children", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxCredits_CreateChild_InvalidJSON(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-credits/2025/children", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxCredits_CreateChild_MonthsOutOfRange(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	tests := []struct {
		name   string
		months int
	}{
		{"zero months", 0},
		{"13 months", 13},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := fmt.Sprintf(`{"child_name":"Petr","birth_number":"1201011234","child_order":1,"months_claimed":%d,"ztp":false}`, tc.months)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-credits/2025/children", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("months=%d: status = %d, want %d, body: %s", tc.months, w.Code, http.StatusBadRequest, w.Body.String())
			}
		})
	}
}

func TestTaxCredits_CreateChild_InvalidChildOrder(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	tests := []struct {
		name  string
		order int
	}{
		{"order 0", 0},
		{"order 4", 4},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := fmt.Sprintf(`{"child_name":"Petr","birth_number":"1201011234","child_order":%d,"months_claimed":12,"ztp":false}`, tc.order)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-credits/2025/children", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("order=%d: status = %d, want %d, body: %s", tc.order, w.Code, http.StatusBadRequest, w.Body.String())
			}
		})
	}
}

// --- PUT /{year}/children/{id} (Update) ---

func TestTaxCredits_UpdateChild(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	// Create child first.
	body := `{"child_name":"Petr","birth_number":"1201011234","child_order":1,"months_claimed":12,"ztp":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-credits/2025/children", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var created taxChildCreditResponse
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	// Update the child.
	updateBody := `{"child_name":"Petr Updated","birth_number":"1201011234","child_order":2,"months_claimed":6,"ztp":true}`
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/tax-credits/2025/children/%d", created.ID), bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update: status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var updated taxChildCreditResponse
	if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if updated.ChildName != "Petr Updated" {
		t.Errorf("ChildName = %q, want %q", updated.ChildName, "Petr Updated")
	}
	if updated.ChildOrder != 2 {
		t.Errorf("ChildOrder = %d, want 2", updated.ChildOrder)
	}
	if updated.MonthsClaimed != 6 {
		t.Errorf("MonthsClaimed = %d, want 6", updated.MonthsClaimed)
	}
	if !updated.ZTP {
		t.Error("expected ZTP to be true")
	}
}

func TestTaxCredits_UpdateChild_InvalidID(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	body := `{"child_name":"Petr","birth_number":"1201011234","child_order":1,"months_claimed":12,"ztp":false}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/2025/children/abc", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxCredits_UpdateChild_InvalidJSON(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/2025/children/1", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxCredits_UpdateChild_InvalidYear(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	body := `{"child_name":"Petr","birth_number":"1201011234","child_order":1,"months_claimed":12,"ztp":false}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/abc/children/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- DELETE /{year}/children/{id} ---

func TestTaxCredits_DeleteChild(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	// Create child first.
	body := `{"child_name":"Petr","birth_number":"1201011234","child_order":1,"months_claimed":12,"ztp":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-credits/2025/children", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, want %d", w.Code, http.StatusCreated)
	}

	var created taxChildCreditResponse
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	// Delete.
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/tax-credits/2025/children/%d", created.ID), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: status = %d, want %d, body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify gone via list.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tax-credits/2025/children", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("list after delete: status = %d, want %d", w.Code, http.StatusOK)
	}

	var children []taxChildCreditResponse
	if err := json.NewDecoder(w.Body).Decode(&children); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(children) != 0 {
		t.Errorf("expected 0 children after delete, got %d", len(children))
	}
}

func TestTaxCredits_DeleteChild_InvalidID(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tax-credits/2025/children/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- PUT /{year}/personal ---

func TestTaxCredits_UpsertPersonal(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	body := `{"is_student":true,"student_months":10,"disability_level":1}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/2025/personal", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp taxPersonalCreditsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Year != 2025 {
		t.Errorf("Year = %d, want 2025", resp.Year)
	}
	if !resp.IsStudent {
		t.Error("expected IsStudent to be true")
	}
	if resp.StudentMonths != 10 {
		t.Errorf("StudentMonths = %d, want 10", resp.StudentMonths)
	}
	if resp.DisabilityLevel != 1 {
		t.Errorf("DisabilityLevel = %d, want 1", resp.DisabilityLevel)
	}
}

func TestTaxCredits_UpsertPersonal_Update(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	// Create first.
	body := `{"is_student":true,"student_months":10,"disability_level":0}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/2025/personal", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first upsert: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Update.
	body = `{"is_student":false,"student_months":0,"disability_level":2}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/2025/personal", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("second upsert: status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp taxPersonalCreditsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.IsStudent {
		t.Error("expected IsStudent to be false after update")
	}
	if resp.DisabilityLevel != 2 {
		t.Errorf("DisabilityLevel = %d, want 2", resp.DisabilityLevel)
	}
}

func TestTaxCredits_UpsertPersonal_InvalidYear(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	body := `{"is_student":true,"student_months":10,"disability_level":0}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/abc/personal", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxCredits_UpsertPersonal_InvalidJSON(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/2025/personal", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxCredits_UpsertPersonal_StudentMonthsOutOfRange(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	body := `{"is_student":true,"student_months":13,"disability_level":0}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/2025/personal", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestTaxCredits_UpsertPersonal_DisabilityLevelOutOfRange(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	body := `{"is_student":false,"student_months":0,"disability_level":4}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/2025/personal", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// --- GET /{year}/children (List) ---

func TestTaxCredits_ListChildren_Empty(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tax-credits/2025/children", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var children []taxChildCreditResponse
	if err := json.NewDecoder(w.Body).Decode(&children); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(children) != 0 {
		t.Errorf("expected 0 children, got %d", len(children))
	}
}

func TestTaxCredits_ListChildren_Multiple(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	// Create two children.
	for i, name := range []string{"Petr", "Anna"} {
		body := fmt.Sprintf(`{"child_name":%q,"birth_number":"120101%04d","child_order":%d,"months_claimed":12,"ztp":false}`, name, i, i+1)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-credits/2025/children", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create child %d: status = %d, want %d", i, w.Code, http.StatusCreated)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tax-credits/2025/children", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var children []taxChildCreditResponse
	if err := json.NewDecoder(w.Body).Decode(&children); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(children) != 2 {
		t.Errorf("expected 2 children, got %d", len(children))
	}
}

func TestTaxCredits_ListChildren_InvalidYear(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tax-credits/abc/children", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- POST /{year}/copy-from/{sourceYear} ---

func TestTaxCredits_CopyFromYear(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	// Set up source year data.
	spouseBody := `{"spouse_name":"Jana","spouse_birth_number":"8551011234","spouse_income":5000000,"spouse_ztp":false,"months_claimed":10}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-credits/2024/spouse", bytes.NewBufferString(spouseBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upsert spouse 2024: status = %d, body: %s", w.Code, w.Body.String())
	}

	childBody := `{"child_name":"Petr","birth_number":"1201011234","child_order":1,"months_claimed":8,"ztp":false}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tax-credits/2024/children", bytes.NewBufferString(childBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create child 2024: status = %d, body: %s", w.Code, w.Body.String())
	}

	// Copy from 2024 to 2025.
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tax-credits/2025/copy-from/2024", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("copy: status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify target year has data.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tax-credits/2025", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("summary 2025: status = %d, body: %s", w.Code, w.Body.String())
	}

	var resp taxCreditsSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Spouse == nil {
		t.Fatal("expected Spouse to be copied")
	}
	if resp.Spouse.SpouseName != "Jana" {
		t.Errorf("Spouse.SpouseName = %q, want %q", resp.Spouse.SpouseName, "Jana")
	}
	// Copied entries should have months_claimed reset to 12.
	if resp.Spouse.MonthsClaimed != 12 {
		t.Errorf("Spouse.MonthsClaimed = %d, want 12 (reset on copy)", resp.Spouse.MonthsClaimed)
	}
	if len(resp.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(resp.Children))
	}
	if resp.Children[0].ChildName != "Petr" {
		t.Errorf("Children[0].ChildName = %q, want %q", resp.Children[0].ChildName, "Petr")
	}
	if resp.Children[0].MonthsClaimed != 12 {
		t.Errorf("Children[0].MonthsClaimed = %d, want 12 (reset on copy)", resp.Children[0].MonthsClaimed)
	}
}

func TestTaxCredits_CopyFromYear_SameYear(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-credits/2025/copy-from/2025", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestTaxCredits_CopyFromYear_InvalidTargetYear(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-credits/abc/copy-from/2024", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxCredits_CopyFromYear_InvalidSourceYear(t *testing.T) {
	r := setupTaxCreditsRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tax-credits/2025/copy-from/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
