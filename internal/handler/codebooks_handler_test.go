package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zajca/zfaktury/internal/codebooks"
)

func TestCodebooksHandler_FinancialOffices(t *testing.T) {
	h := NewCodebooksHandler()
	req := httptest.NewRequest(http.MethodGet, "/financial-offices", nil)
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got []codebooks.FinancialOffice
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 15 {
		t.Errorf("len(offices) = %d, want 15", len(got))
	}
	var has464 bool
	for _, o := range got {
		if o.Code == "464" {
			has464 = true
			if !strings.Contains(o.Name, "Zlínský") {
				t.Errorf("464.Name = %q, want substring 'Zlínský'", o.Name)
			}
		}
	}
	if !has464 {
		t.Errorf("missing code 464 (Zlínský) in response")
	}
}

func TestCodebooksHandler_NACE(t *testing.T) {
	h := NewCodebooksHandler()
	req := httptest.NewRequest(http.MethodGet, "/nace", nil)
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got []codebooks.NACEEntry
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) < 700 {
		t.Errorf("len(nace) = %d, want at least 700 leaf subclasses", len(got))
	}
	var has62109 bool
	for _, e := range got {
		if e.Code == "62109" {
			has62109 = true
		}
	}
	if !has62109 {
		t.Errorf("missing code 62109 (Ostatní počítačové programování)")
	}
}
