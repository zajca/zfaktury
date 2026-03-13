package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/testutil"
)

func setupTaxYearSettingsRouter(t *testing.T) *chi.Mux {
	t.Helper()
	db := testutil.NewTestDB(t)
	settingsRepo := repository.NewTaxYearSettingsRepository(db)
	prepaymentRepo := repository.NewTaxPrepaymentRepository(db)
	svc := service.NewTaxYearSettingsService(settingsRepo, prepaymentRepo, nil)
	h := NewTaxYearSettingsHandler(svc)

	r := chi.NewRouter()
	r.Mount("/api/v1/tax-year-settings", h.Routes())
	return r
}

func TestTaxYearSettings_GetByYear_Default(t *testing.T) {
	r := setupTaxYearSettingsRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tax-year-settings/2025", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp taxYearSettingsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Year != 2025 {
		t.Errorf("Year = %d, want 2025", resp.Year)
	}
	if resp.FlatRatePercent != 0 {
		t.Errorf("FlatRatePercent = %d, want 0", resp.FlatRatePercent)
	}
	if len(resp.Prepayments) != 12 {
		t.Errorf("len(Prepayments) = %d, want 12", len(resp.Prepayments))
	}
}

func TestTaxYearSettings_GetByYear_InvalidYear(t *testing.T) {
	r := setupTaxYearSettingsRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tax-year-settings/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxYearSettings_GetByYear_YearOutOfRange(t *testing.T) {
	r := setupTaxYearSettingsRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tax-year-settings/1999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestTaxYearSettings_Save_Valid(t *testing.T) {
	r := setupTaxYearSettingsRouter(t)

	body := `{"flat_rate_percent": 60, "prepayments": [
		{"month":1,"tax_amount":1000,"social_amount":2000,"health_amount":3000}
	]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-year-settings/2025", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}

	// Verify the settings were saved by GET.
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/tax-year-settings/2025", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d, body: %s", getW.Code, http.StatusOK, getW.Body.String())
	}

	var resp taxYearSettingsResponse
	if err := json.NewDecoder(getW.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.FlatRatePercent != 60 {
		t.Errorf("FlatRatePercent = %d, want 60", resp.FlatRatePercent)
	}

	// Check prepayment for month 1 was saved.
	found := false
	for _, p := range resp.Prepayments {
		if p.Month == 1 {
			found = true
			if p.TaxAmount != 1000 {
				t.Errorf("month 1 TaxAmount = %d, want 1000", p.TaxAmount)
			}
			if p.SocialAmount != 2000 {
				t.Errorf("month 1 SocialAmount = %d, want 2000", p.SocialAmount)
			}
			if p.HealthAmount != 3000 {
				t.Errorf("month 1 HealthAmount = %d, want 3000", p.HealthAmount)
			}
		}
	}
	if !found {
		t.Error("expected prepayment for month 1")
	}
}

func TestTaxYearSettings_Save_InvalidYear(t *testing.T) {
	r := setupTaxYearSettingsRouter(t)

	body := `{"flat_rate_percent": 60}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-year-settings/abc", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxYearSettings_Save_InvalidJSON(t *testing.T) {
	r := setupTaxYearSettingsRouter(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-year-settings/2025", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxYearSettings_Save_InvalidFlatRate(t *testing.T) {
	r := setupTaxYearSettingsRouter(t)

	body := `{"flat_rate_percent": 50}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-year-settings/2025", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestTaxYearSettings_Save_InvalidMonth(t *testing.T) {
	r := setupTaxYearSettingsRouter(t)

	body := `{"flat_rate_percent": 60, "prepayments": [{"month":13,"tax_amount":100,"social_amount":200,"health_amount":300}]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-year-settings/2025", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestTaxYearSettings_Save_InvalidMonthZero(t *testing.T) {
	r := setupTaxYearSettingsRouter(t)

	body := `{"flat_rate_percent": 60, "prepayments": [{"month":0,"tax_amount":100,"social_amount":200,"health_amount":300}]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-year-settings/2025", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTaxYearSettings_Save_NegativeAmount(t *testing.T) {
	r := setupTaxYearSettingsRouter(t)

	body := `{"flat_rate_percent": 60, "prepayments": [{"month":1,"tax_amount":-100,"social_amount":200,"health_amount":300}]}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-year-settings/2025", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestTaxYearSettings_Save_AllValidRates(t *testing.T) {
	r := setupTaxYearSettingsRouter(t)

	validRates := []int{0, 30, 40, 60, 80}
	for _, rate := range validRates {
		body, _ := json.Marshal(map[string]any{"flat_rate_percent": rate})
		req := httptest.NewRequest(http.MethodPut, "/api/v1/tax-year-settings/2025", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("rate %d: status = %d, want %d, body: %s", rate, w.Code, http.StatusNoContent, w.Body.String())
		}
	}
}
