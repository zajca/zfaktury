package ocr

import (
	"strings"
	"testing"
)

func TestParseInvestmentJSON_Valid(t *testing.T) {
	input := `{
		"platform": "portu",
		"capital_entries": [
			{
				"category": "dividend_foreign",
				"description": "Vanguard S&P 500 ETF dividend",
				"income_date": "2025-06-15",
				"gross_amount": 1234.56,
				"withheld_tax_cz": 0,
				"withheld_tax_foreign": 185.18,
				"country_code": "US",
				"needs_declaring": true
			}
		],
		"transactions": [
			{
				"asset_type": "etf",
				"asset_name": "Vanguard S&P 500 ETF",
				"isin": "US9229083632",
				"transaction_type": "buy",
				"transaction_date": "2025-03-10",
				"quantity": 2.5,
				"unit_price": 450.00,
				"total_amount": 1125.00,
				"fees": 1.50,
				"currency_code": "USD",
				"exchange_rate": 23.50
			}
		],
		"confidence": 0.95
	}`

	resp, err := ParseInvestmentJSON(input)
	if err != nil {
		t.Fatalf("ParseInvestmentJSON() error: %v", err)
	}

	if resp.Platform != "portu" {
		t.Errorf("Platform = %q, want %q", resp.Platform, "portu")
	}
	if resp.Confidence != 0.95 {
		t.Errorf("Confidence = %f, want %f", resp.Confidence, 0.95)
	}
	if len(resp.CapitalEntries) != 1 {
		t.Fatalf("len(CapitalEntries) = %d, want 1", len(resp.CapitalEntries))
	}
	ce := resp.CapitalEntries[0]
	if ce.Category != "dividend_foreign" {
		t.Errorf("CapitalEntries[0].Category = %q, want %q", ce.Category, "dividend_foreign")
	}
	if ce.GrossAmount != 1234.56 {
		t.Errorf("CapitalEntries[0].GrossAmount = %f, want %f", ce.GrossAmount, 1234.56)
	}
	if !ce.NeedsDeclaring {
		t.Error("expected CapitalEntries[0].NeedsDeclaring to be true")
	}

	if len(resp.Transactions) != 1 {
		t.Fatalf("len(Transactions) = %d, want 1", len(resp.Transactions))
	}
	tx := resp.Transactions[0]
	if tx.AssetType != "etf" {
		t.Errorf("Transactions[0].AssetType = %q, want %q", tx.AssetType, "etf")
	}
	if tx.Quantity != 2.5 {
		t.Errorf("Transactions[0].Quantity = %f, want %f", tx.Quantity, 2.5)
	}
	if tx.ExchangeRate != 23.50 {
		t.Errorf("Transactions[0].ExchangeRate = %f, want %f", tx.ExchangeRate, 23.50)
	}
}

func TestParseInvestmentJSON_WithCodeFences(t *testing.T) {
	input := "```json\n{\"platform\": \"zonky\", \"capital_entries\": [], \"transactions\": [], \"confidence\": 0.8}\n```"

	resp, err := ParseInvestmentJSON(input)
	if err != nil {
		t.Fatalf("ParseInvestmentJSON() error: %v", err)
	}

	if resp.Platform != "zonky" {
		t.Errorf("Platform = %q, want %q", resp.Platform, "zonky")
	}
}

func TestParseInvestmentJSON_InvalidJSON(t *testing.T) {
	_, err := ParseInvestmentJSON("not valid json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseInvestmentJSON_EmptyEntries(t *testing.T) {
	input := `{"platform": "other", "capital_entries": [], "transactions": [], "confidence": 0.5}`

	resp, err := ParseInvestmentJSON(input)
	if err != nil {
		t.Fatalf("ParseInvestmentJSON() error: %v", err)
	}

	if len(resp.CapitalEntries) != 0 {
		t.Errorf("len(CapitalEntries) = %d, want 0", len(resp.CapitalEntries))
	}
	if len(resp.Transactions) != 0 {
		t.Errorf("len(Transactions) = %d, want 0", len(resp.Transactions))
	}
}

func TestCzkToHalere(t *testing.T) {
	tests := []struct {
		name string
		czk  float64
		want int64
	}{
		{"zero", 0, 0},
		{"whole", 100.0, 10000},
		{"with halere", 1234.56, 123456},
		{"small", 0.01, 1},
		{"negative", -50.25, -5024}, // rounding: -50.25*100+0.5 = -5024.5 -> -5024
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CzkToHalere(tt.czk)
			if got != tt.want {
				t.Errorf("CzkToHalere(%f) = %d, want %d", tt.czk, got, tt.want)
			}
		})
	}
}

func TestQuantityToInt(t *testing.T) {
	tests := []struct {
		name string
		qty  float64
		want int64
	}{
		{"one share", 1.0, 10000},
		{"fractional", 2.5, 25000},
		{"zero", 0, 0},
		{"small fraction", 0.0001, 1},
		{"large", 100.0, 1000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := QuantityToInt(tt.qty)
			if got != tt.want {
				t.Errorf("QuantityToInt(%f) = %d, want %d", tt.qty, got, tt.want)
			}
		})
	}
}

func TestExchangeRateToInt(t *testing.T) {
	tests := []struct {
		name string
		rate float64
		want int64
	}{
		{"czk/eur", 25.12, 251200},
		{"czk/usd", 23.50, 235000},
		{"one to one", 1.0, 10000},
		{"zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExchangeRateToInt(tt.rate)
			if got != tt.want {
				t.Errorf("ExchangeRateToInt(%f) = %d, want %d", tt.rate, got, tt.want)
			}
		})
	}
}

func TestInvestmentSystemPrompt(t *testing.T) {
	prompt := InvestmentSystemPrompt()
	if prompt == "" {
		t.Error("expected non-empty system prompt")
	}
	if !strings.Contains(prompt, "JSON") {
		t.Error("expected system prompt to mention JSON")
	}
	if !strings.Contains(prompt, "capital_entries") {
		t.Error("expected system prompt to mention capital_entries")
	}
	if !strings.Contains(prompt, "transactions") {
		t.Error("expected system prompt to mention transactions")
	}
}

func TestInvestmentUserPrompt(t *testing.T) {
	tests := []struct {
		platform    string
		shouldMatch string
	}{
		{"portu", "Portu"},
		{"zonky", "Zonky"},
		{"trading212", "Trading212"},
		{"revolut", "Revolut"},
		{"other", "Analyzuj"},
		{"unknown", "Analyzuj"},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			prompt := InvestmentUserPrompt(tt.platform)
			if prompt == "" {
				t.Error("expected non-empty user prompt")
			}
			if !strings.Contains(prompt, tt.shouldMatch) {
				t.Errorf("expected prompt for %q to contain %q, got: %q", tt.platform, tt.shouldMatch, prompt)
			}
		})
	}
}
