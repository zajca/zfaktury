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

func setupSettingsRouter(t *testing.T) *chi.Mux {
	t.Helper()
	db := testutil.NewTestDB(t)
	settingsRepo := repository.NewSettingsRepository(db)
	settingsSvc := service.NewSettingsService(settingsRepo, nil)
	h := NewSettingsHandler(settingsSvc)

	r := chi.NewRouter()
	r.Mount("/api/v1/settings", h.Routes())
	return r
}

func TestSettingsHandler_GetAll_Empty(t *testing.T) {
	r := setupSettingsRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if len(resp) != 0 {
		t.Errorf("expected empty map, got %v", resp)
	}
}

func TestSettingsHandler_Update_Valid(t *testing.T) {
	r := setupSettingsRouter(t)

	settings := map[string]string{
		"company_name": "Test Company",
		"ico":          "12345678",
		"city":         "Praha",
	}
	body, _ := json.Marshal(settings)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	for key, want := range settings {
		if got := resp[key]; got != want {
			t.Errorf("setting %q = %q, want %q", key, got, want)
		}
	}
}

func TestSettingsHandler_Update_InvalidJSON(t *testing.T) {
	r := setupSettingsRouter(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBufferString(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestSettingsHandler_Update_UnknownKey(t *testing.T) {
	r := setupSettingsRouter(t)

	settings := map[string]string{
		"unknown_key": "some value",
	}
	body, _ := json.Marshal(settings)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestSettingsHandler_GetAll_AfterUpdate(t *testing.T) {
	r := setupSettingsRouter(t)

	// First, set some values.
	settings := map[string]string{
		"company_name": "My Company",
		"email":        "info@example.com",
	}
	body, _ := json.Marshal(settings)
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	r.ServeHTTP(putW, putReq)

	if putW.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d, body = %s", putW.Code, http.StatusOK, putW.Body.String())
	}

	// Then, GET all settings and verify.
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Errorf("GET status = %d, want %d, body = %s", getW.Code, http.StatusOK, getW.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(getW.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	for key, want := range settings {
		if got := resp[key]; got != want {
			t.Errorf("setting %q = %q, want %q", key, got, want)
		}
	}
}
