package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/service"
)

func setupTaxCalendarHandler(t *testing.T) (*TaxCalendarHandler, *chi.Mux) {
	t.Helper()
	svc := service.NewTaxCalendarService()
	h := NewTaxCalendarHandler(svc)

	r := chi.NewRouter()
	r.Get("/", h.GetCalendar)
	return h, r
}

func TestTaxCalendarHandler_GetCalendar_DefaultYear(t *testing.T) {
	_, r := setupTaxCalendarHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp taxCalendarResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	currentYear := time.Now().Year()
	if resp.Year != currentYear {
		t.Errorf("Year = %d, want %d", resp.Year, currentYear)
	}
	if len(resp.Deadlines) == 0 {
		t.Error("expected non-empty deadlines list")
	}

	// Verify each deadline has required fields.
	for i, d := range resp.Deadlines {
		if d.Name == "" {
			t.Errorf("deadline[%d].Name is empty", i)
		}
		if d.Date == "" {
			t.Errorf("deadline[%d].Date is empty", i)
		}
		if d.Type == "" {
			t.Errorf("deadline[%d].Type is empty", i)
		}
	}
}

func TestTaxCalendarHandler_GetCalendar_WithYear(t *testing.T) {
	_, r := setupTaxCalendarHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/?year=2026", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp taxCalendarResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Year != 2026 {
		t.Errorf("Year = %d, want 2026", resp.Year)
	}
	if len(resp.Deadlines) == 0 {
		t.Error("expected non-empty deadlines for year 2026")
	}

	// Dates should be in 2026 or early 2027 (some deadlines for a tax year
	// fall in January/February of the following year).
	for i, d := range resp.Deadlines {
		parsed, err := time.Parse("2006-01-02", d.Date)
		if err != nil {
			t.Errorf("deadline[%d].Date %q is not valid date: %v", i, d.Date, err)
			continue
		}
		if parsed.Year() != 2026 && parsed.Year() != 2027 {
			t.Errorf("deadline[%d].Date year = %d, want 2026 or 2027", i, parsed.Year())
		}
	}
}

func TestTaxCalendarHandler_GetCalendar_InvalidYear(t *testing.T) {
	_, r := setupTaxCalendarHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/?year=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestTaxCalendarHandler_GetCalendar_OutOfRange(t *testing.T) {
	_, r := setupTaxCalendarHandler(t)

	// Year 1999 is below the 2000 minimum.
	req := httptest.NewRequest(http.MethodGet, "/?year=1999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d for year=1999", w.Code, http.StatusBadRequest)
	}

	// Year 2101 is above the 2100 maximum.
	req = httptest.NewRequest(http.MethodGet, "/?year=2101", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d for year=2101", w.Code, http.StatusBadRequest)
	}
}
