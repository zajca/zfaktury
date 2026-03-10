package cnb

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const sampleRates = `10.03.2026 #049
země|měna|množství|kód|kurz
Austrálie|dolar|1|AUD|15,432
EMU|euro|1|EUR|25,340
Japonsko|jen|100|JPY|15,871
USA|dolar|1|USD|23,456
Velká Británie|libra|1|GBP|29,123
`

func almostEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

func TestParseRates(t *testing.T) {
	rates, err := parseRates(strings.NewReader(sampleRates))
	if err != nil {
		t.Fatalf("parseRates failed: %v", err)
	}

	if len(rates) != 5 {
		t.Fatalf("expected 5 rates, got %d", len(rates))
	}

	eur, ok := rates["EUR"]
	if !ok {
		t.Fatal("EUR not found")
	}
	if eur.Country != "EMU" {
		t.Errorf("EUR country = %q, want %q", eur.Country, "EMU")
	}
	if eur.Currency != "euro" {
		t.Errorf("EUR currency = %q, want %q", eur.Currency, "euro")
	}
	if eur.Amount != 1 {
		t.Errorf("EUR amount = %d, want 1", eur.Amount)
	}
	if !almostEqual(eur.Rate, 25.340, 0.001) {
		t.Errorf("EUR rate = %f, want 25.340", eur.Rate)
	}

	jpy, ok := rates["JPY"]
	if !ok {
		t.Fatal("JPY not found")
	}
	if jpy.Amount != 100 {
		t.Errorf("JPY amount = %d, want 100", jpy.Amount)
	}
	if !almostEqual(jpy.Rate, 15.871, 0.001) {
		t.Errorf("JPY rate = %f, want 15.871", jpy.Rate)
	}
}

func TestParseRatesEmptyInput(t *testing.T) {
	_, err := parseRates(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseRatesHeaderOnly(t *testing.T) {
	input := "10.03.2026 #049\nzemě|měna|množství|kód|kurz\n"
	_, err := parseRates(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for header-only input")
	}
}

func TestGetRate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, sampleRates)
	}))
	defer server.Close()

	client := NewClient()
	client.SetBaseURL(server.URL)

	ctx := context.Background()

	// Test EUR (amount=1)
	rate, err := client.GetRate(ctx, "EUR", time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GetRate EUR failed: %v", err)
	}
	if !almostEqual(rate, 25.340, 0.001) {
		t.Errorf("EUR rate = %f, want 25.340", rate)
	}

	// Test JPY (amount=100, should divide by 100)
	rate, err = client.GetRate(ctx, "JPY", time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GetRate JPY failed: %v", err)
	}
	if !almostEqual(rate, 0.15871, 0.0001) {
		t.Errorf("JPY rate = %f, want 0.15871", rate)
	}

	// Test unknown currency
	_, err = client.GetRate(ctx, "XYZ", time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected error for unknown currency XYZ")
	}
}

func TestGetRateCaseInsensitive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, sampleRates)
	}))
	defer server.Close()

	client := NewClient()
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	rate, err := client.GetRate(ctx, "eur", time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GetRate eur (lowercase) failed: %v", err)
	}
	if !almostEqual(rate, 25.340, 0.001) {
		t.Errorf("EUR rate = %f, want 25.340", rate)
	}
}

func TestGetRateInvalidCurrencyCode(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	_, err := client.GetRate(ctx, "EU", time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected error for 2-letter currency code")
	}
}

func TestGetRateWeekendFallback(t *testing.T) {
	// Server returns data only for specific dates
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		date := r.URL.Query().Get("date")
		// Only serve data for Friday March 6, return empty for Saturday/Sunday
		if date == "06.03.2026" {
			fmt.Fprint(w, sampleRates)
		} else {
			// Return header-only response (no data rows) to simulate no rates available
			fmt.Fprint(w, "08.03.2026 #049\nzemě|měna|množství|kód|kurz\n")
		}
	}))
	defer server.Close()

	client := NewClient()
	client.SetBaseURL(server.URL)

	ctx := context.Background()

	// Request Sunday (March 8) - should fall back to Friday (March 6)
	rate, err := client.GetRate(ctx, "EUR", time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GetRate with weekend fallback failed: %v", err)
	}
	if !almostEqual(rate, 25.340, 0.001) {
		t.Errorf("EUR rate = %f, want 25.340", rate)
	}
}

func TestCacheTTLExpiry(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		fmt.Fprint(w, sampleRates)
	}))
	defer server.Close()

	client := NewClient()
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	date := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	// First call - should fetch from server
	_, err := client.GetRate(ctx, "EUR", date)
	if err != nil {
		t.Fatalf("first GetRate failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 server call, got %d", callCount)
	}

	// Second call - should use cache
	_, err = client.GetRate(ctx, "EUR", date)
	if err != nil {
		t.Fatalf("second GetRate failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 server call (cached), got %d", callCount)
	}

	// Manually expire cache
	dateKey := date.Format("02.01.2006")
	client.mu.Lock()
	entry := client.cache[dateKey]
	entry.fetchedAt = time.Now().Add(-2 * time.Hour)
	client.cache[dateKey] = entry
	client.mu.Unlock()

	// Third call - should refetch
	_, err = client.GetRate(ctx, "EUR", date)
	if err != nil {
		t.Fatalf("third GetRate failed: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 server calls after cache expiry, got %d", callCount)
	}
}

func TestFetchRatesServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient()
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	_, err := client.GetRate(ctx, "EUR", time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.cache == nil {
		t.Fatal("cache not initialized")
	}
	if client.httpClient == nil {
		t.Fatal("httpClient not initialized")
	}
}
