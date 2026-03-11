package ocr

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseOCRJSON_ValidResponse(t *testing.T) {
	input := `{
		"vendor_name": "Firma s.r.o.",
		"vendor_ico": "12345678",
		"vendor_dic": "CZ12345678",
		"invoice_number": "FV-2026-001",
		"issue_date": "2026-01-15",
		"due_date": "2026-02-15",
		"total_amount": 12100.00,
		"vat_amount": 2100.00,
		"vat_rate_percent": 21,
		"currency_code": "CZK",
		"description": "IT sluzby",
		"items": [
			{
				"description": "Konzultace",
				"quantity": 10.0,
				"unit_price": 1000.00,
				"vat_rate_percent": 21,
				"total_amount": 12100.00
			}
		],
		"raw_text": "Faktura FV-2026-001...",
		"confidence": 0.95
	}`

	result, err := ParseOCRJSON(input)
	if err != nil {
		t.Fatalf("ParseOCRJSON() error: %v", err)
	}

	if result.VendorName != "Firma s.r.o." {
		t.Errorf("VendorName = %q, want %q", result.VendorName, "Firma s.r.o.")
	}
	if result.VendorICO != "12345678" {
		t.Errorf("VendorICO = %q, want %q", result.VendorICO, "12345678")
	}
	if result.VendorDIC != "CZ12345678" {
		t.Errorf("VendorDIC = %q, want %q", result.VendorDIC, "CZ12345678")
	}
	if result.InvoiceNumber != "FV-2026-001" {
		t.Errorf("InvoiceNumber = %q, want %q", result.InvoiceNumber, "FV-2026-001")
	}
	if result.IssueDate != "2026-01-15" {
		t.Errorf("IssueDate = %q, want %q", result.IssueDate, "2026-01-15")
	}
	if result.DueDate != "2026-02-15" {
		t.Errorf("DueDate = %q, want %q", result.DueDate, "2026-02-15")
	}
	if result.TotalAmount != 1210000 {
		t.Errorf("TotalAmount = %d, want %d", result.TotalAmount, 1210000)
	}
	if result.VATAmount != 210000 {
		t.Errorf("VATAmount = %d, want %d", result.VATAmount, 210000)
	}
	if result.VATRatePercent != 21 {
		t.Errorf("VATRatePercent = %d, want %d", result.VATRatePercent, 21)
	}
	if result.CurrencyCode != "CZK" {
		t.Errorf("CurrencyCode = %q, want %q", result.CurrencyCode, "CZK")
	}
	if result.Confidence != 0.95 {
		t.Errorf("Confidence = %f, want %f", result.Confidence, 0.95)
	}

	if len(result.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(result.Items))
	}
	item := result.Items[0]
	if item.Description != "Konzultace" {
		t.Errorf("Items[0].Description = %q, want %q", item.Description, "Konzultace")
	}
	if item.Quantity != 1000 {
		t.Errorf("Items[0].Quantity = %d, want %d", item.Quantity, 1000)
	}
	if item.UnitPrice != 100000 {
		t.Errorf("Items[0].UnitPrice = %d, want %d", item.UnitPrice, 100000)
	}
}

func TestParseOCRJSON_WithCodeFences(t *testing.T) {
	input := "```json\n{\"vendor_name\": \"Test\", \"vendor_ico\": \"\", \"vendor_dic\": \"\", \"invoice_number\": \"\", \"issue_date\": \"\", \"due_date\": \"\", \"total_amount\": 0, \"vat_amount\": 0, \"vat_rate_percent\": 0, \"currency_code\": \"CZK\", \"description\": \"\", \"items\": [], \"raw_text\": \"\", \"confidence\": 0.5}\n```"

	result, err := ParseOCRJSON(input)
	if err != nil {
		t.Fatalf("ParseOCRJSON() with code fences error: %v", err)
	}
	if result.VendorName != "Test" {
		t.Errorf("VendorName = %q, want %q", result.VendorName, "Test")
	}
}

func TestParseOCRJSON_InvalidJSON(t *testing.T) {
	_, err := ParseOCRJSON("not json at all")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseOCRJSON_EmptyItems(t *testing.T) {
	input := `{"vendor_name": "", "vendor_ico": "", "vendor_dic": "", "invoice_number": "", "issue_date": "", "due_date": "", "total_amount": 500.50, "vat_amount": 0, "vat_rate_percent": 0, "currency_code": "CZK", "description": "", "items": [], "raw_text": "", "confidence": 0.3}`

	result, err := ParseOCRJSON(input)
	if err != nil {
		t.Fatalf("ParseOCRJSON() error: %v", err)
	}
	if result.TotalAmount != 50050 {
		t.Errorf("TotalAmount = %d, want %d", result.TotalAmount, 50050)
	}
	if len(result.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(result.Items))
	}
}

func TestOpenAICompatibleProvider_ProcessImage_Success(t *testing.T) {
	ocrResp := ocrJSONResponse{
		VendorName:    "Mock Vendor",
		InvoiceNumber: "FV-001",
		TotalAmount:   1000.00,
		CurrencyCode:  "CZK",
		Confidence:    0.9,
	}
	ocrJSON, _ := json.Marshal(ocrResp)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization header = %q, want %q", r.Header.Get("Authorization"), "Bearer test-key")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type header = %q, want %q", r.Header.Get("Content-Type"), "application/json")
		}

		resp := chatResponse{
			Choices: []chatChoice{
				{Message: chatResponseMessage{Content: string(ocrJSON)}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOpenAIProvider("test-key", "")
	provider.SetBaseURL(server.URL)

	result, err := provider.ProcessImage(context.Background(), []byte{0xFF, 0xD8, 0xFF}, "image/jpeg")
	if err != nil {
		t.Fatalf("ProcessImage() error: %v", err)
	}
	if result.VendorName != "Mock Vendor" {
		t.Errorf("VendorName = %q, want %q", result.VendorName, "Mock Vendor")
	}
	if result.TotalAmount != 100000 {
		t.Errorf("TotalAmount = %d, want %d", result.TotalAmount, 100000)
	}
}

func TestOpenAICompatibleProvider_ProcessImage_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": {"message": "rate limit exceeded"}}`))
	}))
	defer server.Close()

	provider := NewOpenAIProvider("test-key", "")
	provider.SetBaseURL(server.URL)

	_, err := provider.ProcessImage(context.Background(), []byte{0xFF, 0xD8, 0xFF}, "image/jpeg")
	if err == nil {
		t.Error("expected error for API error response")
	}
}

func TestOpenAICompatibleProvider_ProcessImage_UnsupportedContentType(t *testing.T) {
	provider := NewOpenAIProvider("test-key", "")
	_, err := provider.ProcessImage(context.Background(), []byte("data"), "image/webp")
	if err == nil {
		t.Error("expected error for unsupported content type")
	}
}

func TestOpenAICompatibleProvider_Name(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		want     string
	}{
		{"openai", NewOpenAIProvider("key", ""), "openai"},
		{"openrouter", NewOpenRouterProvider("key", ""), "openrouter"},
		{"mistral", NewMistralProvider("key", ""), "mistral"},
		{"gemini", NewGeminiProvider("key", ""), "gemini"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.provider.Name(); got != tt.want {
				t.Errorf("Name() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOpenAICompatibleProvider_DefaultModels(t *testing.T) {
	tests := []struct {
		name  string
		prov  *OpenAICompatibleProvider
		model string
	}{
		{"openai", NewOpenAIProvider("key", ""), "gpt-4o"},
		{"openrouter", NewOpenRouterProvider("key", ""), "google/gemini-2.0-flash-001"},
		{"mistral", NewMistralProvider("key", ""), "pixtral-large-latest"},
		{"gemini", NewGeminiProvider("key", ""), "gemini-2.0-flash"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prov.model != tt.model {
				t.Errorf("model = %q, want %q", tt.prov.model, tt.model)
			}
		})
	}
}

func TestOpenAICompatibleProvider_CustomModel(t *testing.T) {
	p := NewOpenAIProvider("key", "gpt-4o-mini")
	if p.model != "gpt-4o-mini" {
		t.Errorf("model = %q, want %q", p.model, "gpt-4o-mini")
	}
}
