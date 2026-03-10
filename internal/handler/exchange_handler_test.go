package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/service/cnb"
)

const testRates = `10.03.2026 #049
země|měna|množství|kód|kurz
EMU|euro|1|EUR|25,340
USA|dolar|1|USD|23,456
Japonsko|jen|100|JPY|15,871
`

// setupExchangeHandler creates an ExchangeHandler with pre-cached rates for testing.
func setupExchangeHandler(t *testing.T) (*ExchangeHandler, chi.Router) {
	t.Helper()

	client := cnb.NewClient()

	// Pre-populate cache using exported method isn't available,
	// so we test through the handler with a real test CNB server.
	// Instead, use a test HTTP server that serves our rates.
	cnbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testRates))
	}))
	t.Cleanup(cnbServer.Close)

	// We can't override the CNB base URL directly, so pre-populate the cache
	// by using the internal cache. Since cnb.Client fields are unexported,
	// we'll use a different approach: create a real integration through
	// pre-caching via GetRate with a mock.
	// For test simplicity, we'll set up via cnb.NewClientWithHTTPClient if available.
	// Since it's not, we populate cache by calling parseRates indirectly.

	// The simplest approach: use cnb.NewClient and populate cache via exported test helper.
	// Since no such helper exists, we test the handler's HTTP layer by checking
	// validation and response format, and use a separate approach for full integration.

	h := NewExchangeHandler(client)
	r := chi.NewRouter()
	r.Mount("/api/v1/exchange-rate", h.Routes())
	return h, r
}

func TestExchangeHandlerMissingCurrency(t *testing.T) {
	_, router := setupExchangeHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/exchange-rate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp errorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error != "currency parameter is required" {
		t.Errorf("error = %q, want %q", resp.Error, "currency parameter is required")
	}
}

func TestExchangeHandlerInvalidCurrencyCode(t *testing.T) {
	_, router := setupExchangeHandler(t)

	tests := []struct {
		name     string
		currency string
	}{
		{"lowercase", "eur"},
		{"too short", "EU"},
		{"too long", "EURO"},
		{"numbers", "123"},
		{"mixed", "E1R"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/exchange-rate?currency="+tt.currency, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d for currency %q", w.Code, http.StatusBadRequest, tt.currency)
			}
		})
	}
}

func TestExchangeHandlerInvalidDate(t *testing.T) {
	_, router := setupExchangeHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/exchange-rate?currency=EUR&date=2026/03/10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp errorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp.Error, "YYYY-MM-DD") {
		t.Errorf("error = %q, want to contain YYYY-MM-DD", resp.Error)
	}
}

func TestExchangeHandlerValidRequest(t *testing.T) {
	client := cnb.NewClient()
	// Pre-populate cache via exported NewClient (cache is unexported, so we use
	// a test server approach)

	// Create a test CNB server
	cnbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testRates))
	}))
	defer cnbServer.Close()

	// Since we can't override baseURL, use SetBaseURL if it exists.
	// Instead, we'll inject test data by making the client fetch from our server.
	// The cleanest way: expose a test helper on Client. Since we can't modify
	// the Client API in this task, we'll use a workaround.

	// Workaround: Use the client with pre-populated cache by calling internal method
	// through reflection or by providing a NewClientForTest constructor.
	// For now, let's add a SetBaseURL method to the client.

	// Actually, let's just add a test-friendly constructor. But the task says
	// don't modify existing files beyond what's specified. The cnb package is new
	// code we're creating, so we can add whatever we need.

	// Let's add SetBaseURL to the client and use it in tests.
	client.SetBaseURL(cnbServer.URL + "/denni_kurz.txt")

	h := NewExchangeHandler(client)
	router := chi.NewRouter()
	router.Mount("/api/v1/exchange-rate", h.Routes())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/exchange-rate?currency=EUR&date=2026-03-10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp exchangeRateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.CurrencyCode != "EUR" {
		t.Errorf("currency_code = %q, want %q", resp.CurrencyCode, "EUR")
	}
	if resp.Date != "2026-03-10" {
		t.Errorf("date = %q, want %q", resp.Date, "2026-03-10")
	}
	// EUR rate is 25.340 CZK per 1 EUR = 2534 halere
	if resp.Rate != 2534 {
		t.Errorf("rate = %d, want 2534", resp.Rate)
	}
}

func TestExchangeHandlerDefaultDate(t *testing.T) {
	cnbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testRates))
	}))
	defer cnbServer.Close()

	client := cnb.NewClient()
	client.SetBaseURL(cnbServer.URL + "/denni_kurz.txt")

	h := NewExchangeHandler(client)
	router := chi.NewRouter()
	router.Mount("/api/v1/exchange-rate", h.Routes())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/exchange-rate?currency=EUR", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp exchangeRateResponse
	json.NewDecoder(w.Body).Decode(&resp)

	// Date should default to today
	today := time.Now().Format("2006-01-02")
	if resp.Date != today {
		t.Errorf("date = %q, want %q", resp.Date, today)
	}
}

func TestExchangeHandlerJPYMultiUnit(t *testing.T) {
	cnbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testRates))
	}))
	defer cnbServer.Close()

	client := cnb.NewClient()
	client.SetBaseURL(cnbServer.URL + "/denni_kurz.txt")

	h := NewExchangeHandler(client)
	router := chi.NewRouter()
	router.Mount("/api/v1/exchange-rate", h.Routes())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/exchange-rate?currency=JPY&date=2026-03-10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp exchangeRateResponse
	json.NewDecoder(w.Body).Decode(&resp)

	// JPY: 15.871 CZK per 100 JPY = 0.15871 CZK per 1 JPY = 16 halere (rounded)
	if resp.Rate != 16 {
		t.Errorf("rate = %d, want 16 (0.15871 CZK per 1 JPY in halere)", resp.Rate)
	}
}
