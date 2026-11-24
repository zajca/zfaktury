package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// listTotal returns the total of a list endpoint response for the given URL.
func listTotal(t *testing.T, r *chi.Mux, url string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s: status = %d, body = %s", url, w.Code, w.Body.String())
	}
	var resp struct {
		Total int `json:"total"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("GET %s: decoding response: %v", url, err)
	}
	return resp.Total
}

// The seeded invoice and expense are both issued on 2026-03-01.
func TestInvoiceHandler_List_DateFromIncludesSelectedDay(t *testing.T) {
	r, seqID := setupInvoiceRouter(t)
	customerID := createInvoiceCustomer(t, r)
	createTestInvoice(t, r, customerID, seqID)

	tests := []struct {
		url  string
		want int
	}{
		{"/api/v1/invoices?date_from=2026-03-01", 1},
		{"/api/v1/invoices?date_from=2026-03-01&date_to=2026-03-01", 1},
		{"/api/v1/invoices?date_from=2026-03-02", 0},
		{"/api/v1/invoices?date_to=2026-02-28", 0},
	}
	for _, tc := range tests {
		if got := listTotal(t, r, tc.url); got != tc.want {
			t.Errorf("GET %s: total = %d, want %d", tc.url, got, tc.want)
		}
	}
}

func TestExpenseHandler_List_DateFromIncludesSelectedDay(t *testing.T) {
	r := setupExpenseRouter(t)
	createTestExpense(t, r)

	tests := []struct {
		url  string
		want int
	}{
		{"/api/v1/expenses?date_from=2026-03-01", 1},
		{"/api/v1/expenses?date_from=2026-03-01&date_to=2026-03-01", 1},
		{"/api/v1/expenses?date_from=2026-03-02", 0},
		{"/api/v1/expenses?date_to=2026-02-28", 0},
	}
	for _, tc := range tests {
		if got := listTotal(t, r, tc.url); got != tc.want {
			t.Errorf("GET %s: total = %d, want %d", tc.url, got, tc.want)
		}
	}
}
