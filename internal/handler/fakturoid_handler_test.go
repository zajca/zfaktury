package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/service"
)

func TestFakturoidImport_MissingCredentials(t *testing.T) {
	svc := &service.FakturoidImportService{}
	h := NewFakturoidHandler(svc)
	r := chi.NewRouter()
	r.Mount("/api/v1/import/fakturoid", h.Routes())

	body := `{"slug":"","email":"","client_id":"","client_secret":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/import/fakturoid/import", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestFakturoidImport_InvalidBody(t *testing.T) {
	svc := &service.FakturoidImportService{}
	h := NewFakturoidHandler(svc)
	r := chi.NewRouter()
	r.Mount("/api/v1/import/fakturoid", h.Routes())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/import/fakturoid/import", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp["error"] != "invalid request body" {
		t.Errorf("error = %q, want %q", resp["error"], "invalid request body")
	}
}

func TestFakturoidImport_PartialCredentials(t *testing.T) {
	svc := &service.FakturoidImportService{}
	h := NewFakturoidHandler(svc)
	r := chi.NewRouter()
	r.Mount("/api/v1/import/fakturoid", h.Routes())

	tests := []struct {
		name string
		body string
	}{
		{"missing slug", `{"slug":"","email":"test@test.cz","client_id":"cid","client_secret":"csecret"}`},
		{"missing email", `{"slug":"test","email":"","client_id":"cid","client_secret":"csecret"}`},
		{"missing client_id", `{"slug":"test","email":"test@test.cz","client_id":"","client_secret":"csecret"}`},
		{"missing client_secret", `{"slug":"test","email":"test@test.cz","client_id":"cid","client_secret":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/import/fakturoid/import", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
			}
		})
	}
}
