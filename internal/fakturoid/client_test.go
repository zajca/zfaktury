package fakturoid

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// newTestClient creates a Client with a pre-set access token for testing.
func newTestClient(baseURL string) *Client {
	c := NewClient("slug", "user@test.cz", "cid", "csecret", WithBaseURL(baseURL))
	c.accessToken = "test-access-token"
	return c
}

func TestListSubjects_Paginated(t *testing.T) {
	var requestCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)

		// Verify auth and headers.
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-access-token" {
			t.Errorf("unexpected Authorization header: %q", auth)
		}
		ua := r.Header.Get("User-Agent")
		if !strings.Contains(ua, "ZFaktury") {
			t.Errorf("unexpected User-Agent: %q", ua)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("unexpected Accept header: %q", r.Header.Get("Accept"))
		}

		page := r.URL.Query().Get("page")
		switch page {
		case "1":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]Subject{
				{ID: 1, Name: "Alpha s.r.o.", RegistrationNo: "12345678"},
				{ID: 2, Name: "Beta a.s.", RegistrationNo: "87654321"},
			})
		case "2":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]Subject{
				{ID: 3, Name: "Gamma k.s.", RegistrationNo: "11223344"},
			})
		case "3":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]Subject{})
		default:
			t.Errorf("unexpected page: %s", page)
		}
	}))
	defer srv.Close()

	client := NewClient("test-slug", "test@example.com", "cid", "csecret",
		WithBaseURL(srv.URL),
		WithTimeout(0), // use default for test server
	)
	client.accessToken = "test-access-token"

	subjects, err := client.ListSubjects(context.Background())
	if err != nil {
		t.Fatalf("ListSubjects failed: %v", err)
	}

	if len(subjects) != 3 {
		t.Fatalf("expected 3 subjects, got %d", len(subjects))
	}
	if subjects[0].Name != "Alpha s.r.o." {
		t.Errorf("expected first subject name 'Alpha s.r.o.', got %q", subjects[0].Name)
	}
	if subjects[2].ID != 3 {
		t.Errorf("expected third subject ID 3, got %d", subjects[2].ID)
	}

	// 3 requests: page 1, page 2, page 3 (empty).
	if got := requestCount.Load(); got != 3 {
		t.Errorf("expected 3 requests, got %d", got)
	}
}

func TestListInvoices_SinglePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "invoices.json") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		page := r.URL.Query().Get("page")
		switch page {
		case "1":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]Invoice{
				{
					ID:       101,
					Number:   "2026-0001",
					Status:   "paid",
					IssuedOn: "2026-01-15",
					Total:    12100.0,
					Lines: []InvoiceLine{
						{Name: "Consulting", Quantity: 10, UnitPrice: 1000, VatRate: 21},
					},
					Payments: []Payment{
						{PaidOn: "2026-01-20"},
					},
				},
			})
		case "2":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]Invoice{})
		}
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)

	invoices, err := client.ListInvoices(context.Background())
	if err != nil {
		t.Fatalf("ListInvoices failed: %v", err)
	}

	if len(invoices) != 1 {
		t.Fatalf("expected 1 invoice, got %d", len(invoices))
	}
	inv := invoices[0]
	if inv.Number != "2026-0001" {
		t.Errorf("expected number '2026-0001', got %q", inv.Number)
	}
	if inv.Total != 12100.0 {
		t.Errorf("expected total 12100, got %f", inv.Total)
	}
	if len(inv.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(inv.Lines))
	}
	if inv.Lines[0].Name != "Consulting" {
		t.Errorf("expected line name 'Consulting', got %q", inv.Lines[0].Name)
	}
	if len(inv.Payments) != 1 || inv.Payments[0].PaidOn != "2026-01-20" {
		t.Errorf("unexpected payments: %+v", inv.Payments)
	}
}

func TestListExpenses_SinglePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "expenses.json") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		page := r.URL.Query().Get("page")
		switch page {
		case "1":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]Expense{
				{
					ID:             201,
					OriginalNumber: "FV-2026-100",
					IssuedOn:       "2026-02-01",
					Total:          5000.0,
					PaymentMethod:  "bank",
					Lines: []ExpenseLine{
						{Name: "Office supplies", Quantity: 1, UnitPrice: 5000, VatRate: 21},
					},
				},
			})
		case "2":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]Expense{})
		}
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)

	expenses, err := client.ListExpenses(context.Background())
	if err != nil {
		t.Fatalf("ListExpenses failed: %v", err)
	}

	if len(expenses) != 1 {
		t.Fatalf("expected 1 expense, got %d", len(expenses))
	}
	exp := expenses[0]
	if exp.OriginalNumber != "FV-2026-100" {
		t.Errorf("expected original number 'FV-2026-100', got %q", exp.OriginalNumber)
	}
	if exp.PaymentMethod != "bank" {
		t.Errorf("expected payment method 'bank', got %q", exp.PaymentMethod)
	}
	if len(exp.Lines) != 1 || exp.Lines[0].Name != "Office supplies" {
		t.Errorf("unexpected lines: %+v", exp.Lines)
	}
}

func TestListSubjects_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Unauthorized"}`))
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)

	_, err := client.ListSubjects(context.Background())
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "HTTP 401") {
		t.Errorf("expected HTTP 401 in error, got: %v", err)
	}
}

func TestListInvoices_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Internal Server Error"}`))
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)

	_, err := client.ListInvoices(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("expected HTTP 500 in error, got: %v", err)
	}
}

func TestBearerAuthAndUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer my-access-token" {
			t.Errorf("expected Bearer auth, got %q", auth)
		}

		ua := r.Header.Get("User-Agent")
		if ua != "ZFaktury (info@firma.cz)" {
			t.Errorf("expected User-Agent 'ZFaktury (info@firma.cz)', got %q", ua)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Subject{})
	}))
	defer srv.Close()

	client := NewClient("my-slug", "info@firma.cz", "cid", "csecret", WithBaseURL(srv.URL))
	client.accessToken = "my-access-token"

	_, err := client.ListSubjects(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Subject{})
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	client := newTestClient(srv.URL)

	_, err := client.ListSubjects(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}
