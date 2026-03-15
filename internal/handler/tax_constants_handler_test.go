package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/calc"
	"github.com/zajca/zfaktury/internal/domain"
)

func TestToCZK(t *testing.T) {
	tests := []struct {
		input domain.Amount
		want  int64
	}{
		{10000, 100},
		{0, 0},
		{50, 0},
		{100, 1},
	}
	for _, tt := range tests {
		got := toCZK(tt.input)
		if got != tt.want {
			t.Errorf("toCZK(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestTaxConstantsFromService(t *testing.T) {
	c := calc.TaxYearConstants{
		BasicCredit:   3083400,
		SpouseCredit:  2477000,
		FlatRateCaps:  map[int]domain.Amount{60: 200000000},
		TimeTestYears: 3,
	}
	resp := taxConstantsFromService(2025, c)
	if resp.Year != 2025 {
		t.Errorf("Year = %d, want 2025", resp.Year)
	}
	if resp.BasicCredit != 30834 {
		t.Errorf("BasicCredit = %d, want 30834", resp.BasicCredit)
	}
	if resp.FlatRateCaps["60"] != 2000000 {
		t.Errorf("FlatRateCaps[60] = %d, want 2000000", resp.FlatRateCaps["60"])
	}
	if resp.TimeTestYears != 3 {
		t.Errorf("TimeTestYears = %d, want 3", resp.TimeTestYears)
	}
}

func TestHandleGetTaxConstants(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/tax-constants/{year}", handleGetTaxConstants)

	t.Run("valid year", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/tax-constants/2025", nil)
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp taxConstantsResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Year != 2025 {
			t.Errorf("Year = %d, want 2025", resp.Year)
		}
	})

	t.Run("invalid year", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/tax-constants/abc", nil)
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("unsupported year", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/tax-constants/1990", nil)
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})
}
