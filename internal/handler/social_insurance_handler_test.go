package handler

import (
	"bytes"
	"context"
	"database/sql"
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

func setupSocialInsuranceRouter(t *testing.T) (*chi.Mux, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	socialRepo := repository.NewSocialInsuranceOverviewRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	if err := settingsRepo.SetBulk(context.Background(), map[string]string{
		service.SettingCSSZCode:        "775",
		service.SettingFirstName:       "Jan",
		service.SettingLastName:        "Novak",
		service.SettingBirthNumber:     "8001011234",
		service.SettingBirthDate:       "1980-01-01",
		service.SettingStreet:          "Hlavni",
		service.SettingHouseNumber:     "42",
		service.SettingCity:            "Praha",
		service.SettingZIP:             "11000",
		service.SettingEmail:           "jan@example.test",
		service.SettingPhone:           "777123456",
		service.SettingCSSZVariableSym: "12345678",
	}); err != nil {
		t.Fatalf("seed settings: %v", err)
	}
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
	body := fmt.Sprintf(`{"year":%d,"filing_type":"regular","activity_months":[1,2,3,4,5,6,7,8,9,10,11,12]}`, year)
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

	body := `{"year":2025,"filing_type":"regular","activity_months":[1,2,3,4,5,6,7,8,9,10,11,12]}`
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
	wantMonths := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	if len(resp.ActivityMonths) != len(wantMonths) {
		t.Fatalf("ActivityMonths = %v, want %v", resp.ActivityMonths, wantMonths)
	}
	for i := range wantMonths {
		if resp.ActivityMonths[i] != wantMonths[i] {
			t.Errorf("ActivityMonths = %v, want %v", resp.ActivityMonths, wantMonths)
			break
		}
	}
}

func TestSocialInsurance_Create_MissingActivityMonths(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)

	body := `{"year":2025,"filing_type":"regular"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/social-insurance", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestSocialInsurance_Create_InvalidActivityMonth(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)

	body := `{"year":2025,"filing_type":"regular","activity_months":[0]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/social-insurance", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
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

	body := `{"year":0,"filing_type":"regular","activity_months":[1,2,3,4,5,6,7,8,9,10,11,12]}`
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

	body := `{"year":2025,"filing_type":"regular","activity_months":[1,2,3,4,5,6,7,8,9,10,11,12]}`
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

// recalculateSocialInsurance recalculates the overview, which stores the advance
// start month the XML export requires.
func recalculateSocialInsurance(t *testing.T, r *chi.Mux, id int64) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/social-insurance/%d/recalculate", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("recalculate: status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestSocialInsurance_GetByID_AdvanceFields(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)
	recalculateSocialInsurance(t, r, id)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/social-insurance/%d", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	for _, key := range []string{"advance_year", "advance_start_month", "advance_periods", "warnings", "filing_method", "new_monthly_base"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("response has no %q: %s", key, w.Body.String())
		}
	}

	var resp socialInsuranceResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.AdvanceYear != 2026 || resp.AdvanceStartMonth != 6 {
		t.Errorf("advance year/start = %d/%d, want 2026/6", resp.AdvanceYear, resp.AdvanceStartMonth)
	}
	if len(resp.AdvancePeriods) != 2 {
		t.Errorf("AdvancePeriods = %+v, want 2 periods", resp.AdvancePeriods)
	}
	if resp.NewMonthlyBase == nil || *resp.NewMonthlyBase != 1_958_700 {
		t.Errorf("NewMonthlyBase = %v, want 1958700 halere", resp.NewMonthlyBase)
	}
	if string(raw["warnings"]) != `["filing_method_missing"]` {
		t.Errorf("warnings = %s, want [\"filing_method_missing\"]", raw["warnings"])
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
	recalculateSocialInsurance(t, r, id)

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

func TestSocialInsurance_GenerateXML_NotCalculated(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/social-insurance/%d/generate-xml", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	var resp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if want := "přehled je potřeba nejdřív přepočítat"; resp.Error != want {
		t.Errorf("error = %q, want %q", resp.Error, want)
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
	recalculateSocialInsurance(t, r, id)

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

func TestSocialInsurance_SetActivityMonths(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)

	body := `{"activity_months":[7,8,9,10,11,12]}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/social-insurance/%d/activity-months", id), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp socialInsuranceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	wantMonths := []int{7, 8, 9, 10, 11, 12}
	if len(resp.ActivityMonths) != len(wantMonths) {
		t.Fatalf("ActivityMonths = %v, want %v", resp.ActivityMonths, wantMonths)
	}
	for i := range wantMonths {
		if resp.ActivityMonths[i] != wantMonths[i] {
			t.Errorf("ActivityMonths = %v, want %v", resp.ActivityMonths, wantMonths)
			break
		}
	}
	if resp.MinAssessmentBase != 9777000 {
		t.Errorf("MinAssessmentBase = %d, want 9777000", resp.MinAssessmentBase)
	}
}

func TestSocialInsurance_SetActivityMonths_Filed_Conflict(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/social-insurance/%d/mark-filed", id), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("mark-filed: status = %d, want %d", w.Code, http.StatusOK)
	}

	body := `{"activity_months":[7,8,9,10,11,12]}`
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/social-insurance/%d/activity-months", id), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func putSocialInsurance(t *testing.T, r *chi.Mux, id int64, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/social-insurance/%d/%s", id, path), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeSocialInsurance(t *testing.T, w *httptest.ResponseRecorder) socialInsuranceResponse {
	t.Helper()
	var resp socialInsuranceResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error: %v, body: %s", err, w.Body.String())
	}
	return resp
}

func TestSocialInsurance_SetSecondaryActivity(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)

	w := putSocialInsurance(t, r, id, "secondary-activity", `{"months":[7,8,9,10,11,12],"reason":"employment"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	resp := decodeSocialInsurance(t, w)

	if len(resp.SecondaryMonths) != 6 || resp.SecondaryReason != "employment" {
		t.Errorf("secondary = %v/%q, want 6 months/employment", resp.SecondaryMonths, resp.SecondaryReason)
	}
	// No income: the secondary part is below the decisive amount, so only the
	// main months carry a minimum base.
	if resp.SecondaryDecisive != 5_586_400 {
		t.Errorf("SecondaryDecisive = %d, want 5586400 halere (111 736 - 6 x 9 312 CZK)", resp.SecondaryDecisive)
	}
	if resp.SumBase != resp.FinalAssessmentBase || resp.InsuredBase != resp.FinalAssessmentBase {
		t.Errorf("sum/insured = %d/%d, want the final base %d", resp.SumBase, resp.InsuredBase, resp.FinalAssessmentBase)
	}
	if resp.EmploymentBase != nil || resp.ReportedEmployment != nil {
		t.Errorf("employment = %v/%v, want nil", resp.EmploymentBase, resp.ReportedEmployment)
	}
	if !strings.Contains(w.Body.String(), `"secondary_months":[7,8,9,10,11,12]`) {
		t.Errorf("secondary_months not in the response: %s", w.Body.String())
	}
}

func TestSocialInsurance_SetSecondaryActivity_Clear(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)
	if w := putSocialInsurance(t, r, id, "secondary-activity", `{"months":[1],"reason":"student"}`); w.Code != http.StatusOK {
		t.Fatalf("set: status = %d, body: %s", w.Code, w.Body.String())
	}

	w := putSocialInsurance(t, r, id, "secondary-activity", `{"months":[],"reason":""}`)
	if w.Code != http.StatusOK {
		t.Fatalf("clear: status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	resp := decodeSocialInsurance(t, w)
	if len(resp.SecondaryMonths) != 0 || resp.SecondaryReason != "" {
		t.Errorf("secondary = %v/%q, want cleared", resp.SecondaryMonths, resp.SecondaryReason)
	}
	if !strings.Contains(w.Body.String(), `"secondary_months":[]`) {
		t.Errorf("secondary_months is not an empty array: %s", w.Body.String())
	}
}

func TestSocialInsurance_SetSecondaryActivity_Invalid(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{"months without a reason", `{"months":[1],"reason":""}`, http.StatusBadRequest},
		{"reason without months", `{"months":[],"reason":"care"}`, http.StatusBadRequest},
		{"unknown reason", `{"months":[1],"reason":"hobby"}`, http.StatusBadRequest},
		{"month out of range", `{"months":[13],"reason":"care"}`, http.StatusBadRequest},
		{"broken JSON", `{`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := setupSocialInsuranceRouter(t)
			id := createSocialInsurance(t, r, 2025)
			if w := putSocialInsurance(t, r, id, "secondary-activity", tt.body); w.Code != tt.want {
				t.Errorf("status = %d, want %d, body: %s", w.Code, tt.want, w.Body.String())
			}
		})
	}
}

func TestSocialInsurance_SetSecondaryActivity_Filed_Conflict(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/social-insurance/%d/mark-filed", id), nil)
	filed := httptest.NewRecorder()
	r.ServeHTTP(filed, req)
	if filed.Code != http.StatusOK {
		t.Fatalf("mark-filed: status = %d, want %d", filed.Code, http.StatusOK)
	}

	if w := putSocialInsurance(t, r, id, "secondary-activity", `{"months":[1],"reason":"care"}`); w.Code != http.StatusConflict {
		t.Errorf("secondary: status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
	if w := putSocialInsurance(t, r, id, "employment-base", `{"employment_assessment_base":100000}`); w.Code != http.StatusConflict {
		t.Errorf("employment: status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestSocialInsurance_SetEmploymentBase(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)

	// 30 000 CZK fits below the maximum with the minimum base, so it is stored
	// but not reported.
	w := putSocialInsurance(t, r, id, "employment-base", `{"employment_assessment_base":3000000}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	resp := decodeSocialInsurance(t, w)
	if resp.EmploymentBase == nil || *resp.EmploymentBase != 3_000_000 {
		t.Errorf("EmploymentBase = %v, want 3000000", resp.EmploymentBase)
	}
	if resp.ReportedEmployment != nil {
		t.Errorf("ReportedEmployment = %d, want nil", *resp.ReportedEmployment)
	}
	if !strings.Contains(w.Body.String(), `"reported_employment_base":null`) {
		t.Errorf("reported_employment_base is not null: %s", w.Body.String())
	}

	// 2 300 000 CZK reaches the maximum, so it is reported and nothing is insured.
	w = putSocialInsurance(t, r, id, "employment-base", `{"employment_assessment_base":230000000}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	resp = decodeSocialInsurance(t, w)
	if resp.ReportedEmployment == nil || *resp.ReportedEmployment != 230_000_000 || resp.InsuredBase != 0 || resp.TotalInsurance != 0 {
		t.Errorf("reported/insured/total = %v/%d/%d, want 230000000/0/0", resp.ReportedEmployment, resp.InsuredBase, resp.TotalInsurance)
	}

	w = putSocialInsurance(t, r, id, "employment-base", `{"employment_assessment_base":null}`)
	if w.Code != http.StatusOK {
		t.Fatalf("remove: status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	resp = decodeSocialInsurance(t, w)
	if resp.EmploymentBase != nil || resp.ReportedEmployment != nil || resp.InsuredBase != resp.FinalAssessmentBase {
		t.Errorf("after removal = %v/%v/%d, want nil/nil/%d", resp.EmploymentBase, resp.ReportedEmployment, resp.InsuredBase, resp.FinalAssessmentBase)
	}
}

func TestSocialInsurance_SetEmploymentBase_Invalid(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"negative", `{"employment_assessment_base":-100}`},
		{"halere", `{"employment_assessment_base":1234567}`},
		{"broken JSON", `{`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := setupSocialInsuranceRouter(t)
			id := createSocialInsurance(t, r, 2025)
			if w := putSocialInsurance(t, r, id, "employment-base", tt.body); w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
			}
		})
	}
}

func TestSocialInsurance_SetSecondaryActivity_NotFound(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)
	if w := putSocialInsurance(t, r, 99999, "secondary-activity", `{"months":[1],"reason":"care"}`); w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
	if w := putSocialInsurance(t, r, 99999, "employment-base", `{"employment_assessment_base":null}`); w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestSocialInsurance_SetActivityMonths_NotFound(t *testing.T) {
	r, _ := setupSocialInsuranceRouter(t)

	body := `{"activity_months":[7,8,9,10,11,12]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/social-insurance/99999/activity-months", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}
