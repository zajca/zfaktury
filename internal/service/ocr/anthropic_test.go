package ocr

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnthropicProvider_ProcessImage_Success(t *testing.T) {
	ocrResp := ocrJSONResponse{
		VendorName:    "Anthropic Vendor",
		InvoiceNumber: "FV-002",
		TotalAmount:   2000.00,
		CurrencyCode:  "CZK",
		Confidence:    0.85,
	}
	ocrJSON, _ := json.Marshal(ocrResp)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Anthropic-specific headers.
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("x-api-key header = %q, want %q", r.Header.Get("x-api-key"), "test-key")
		}
		if r.Header.Get("anthropic-version") != anthropicAPIVersion {
			t.Errorf("anthropic-version header = %q, want %q", r.Header.Get("anthropic-version"), anthropicAPIVersion)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type header = %q, want %q", r.Header.Get("Content-Type"), "application/json")
		}

		// Verify request body structure.
		body, _ := io.ReadAll(r.Body)
		var req anthropicRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to parse request body: %v", err)
		}
		if req.Model != anthropicDefaultModel {
			t.Errorf("model = %q, want %q", req.Model, anthropicDefaultModel)
		}
		if req.System == "" {
			t.Error("system prompt is empty")
		}
		if len(req.Messages) != 1 {
			t.Fatalf("messages count = %d, want 1", len(req.Messages))
		}
		if req.Messages[0].Role != "user" {
			t.Errorf("message role = %q, want %q", req.Messages[0].Role, "user")
		}
		// Should have image + text content parts.
		if len(req.Messages[0].Content) != 2 {
			t.Fatalf("content parts = %d, want 2", len(req.Messages[0].Content))
		}
		if req.Messages[0].Content[0].Type != "image" {
			t.Errorf("first content type = %q, want %q", req.Messages[0].Content[0].Type, "image")
		}
		if req.Messages[0].Content[0].Source == nil {
			t.Fatal("image source is nil")
		}
		if req.Messages[0].Content[0].Source.Type != "base64" {
			t.Errorf("source type = %q, want %q", req.Messages[0].Content[0].Source.Type, "base64")
		}

		resp := anthropicResponse{
			Content: []anthropicResponseContent{
				{Type: "text", Text: string(ocrJSON)},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewAnthropicProvider("test-key", "")
	provider.SetBaseURL(server.URL)

	result, err := provider.ProcessImage(context.Background(), []byte{0xFF, 0xD8, 0xFF}, "image/jpeg")
	if err != nil {
		t.Fatalf("ProcessImage() error: %v", err)
	}
	if result.VendorName != "Anthropic Vendor" {
		t.Errorf("VendorName = %q, want %q", result.VendorName, "Anthropic Vendor")
	}
	if result.TotalAmount != 200000 {
		t.Errorf("TotalAmount = %d, want %d", result.TotalAmount, 200000)
	}
}

func TestAnthropicProvider_ProcessImage_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"type": "error", "error": {"type": "authentication_error", "message": "invalid api key"}}`))
	}))
	defer server.Close()

	provider := NewAnthropicProvider("bad-key", "")
	provider.SetBaseURL(server.URL)

	_, err := provider.ProcessImage(context.Background(), []byte{0xFF, 0xD8, 0xFF}, "image/jpeg")
	if err == nil {
		t.Error("expected error for API error response")
	}
}

func TestAnthropicProvider_ProcessImage_UnsupportedContentType(t *testing.T) {
	provider := NewAnthropicProvider("test-key", "")
	_, err := provider.ProcessImage(context.Background(), []byte("data"), "image/webp")
	if err == nil {
		t.Error("expected error for unsupported content type")
	}
}

func TestAnthropicProvider_Name(t *testing.T) {
	provider := NewAnthropicProvider("key", "")
	if provider.Name() != "claude" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "claude")
	}
}

func TestAnthropicProvider_DefaultModel(t *testing.T) {
	provider := NewAnthropicProvider("key", "")
	if provider.model != anthropicDefaultModel {
		t.Errorf("model = %q, want %q", provider.model, anthropicDefaultModel)
	}
}

func TestAnthropicProvider_CustomModel(t *testing.T) {
	provider := NewAnthropicProvider("key", "claude-opus-4-20250514")
	if provider.model != "claude-opus-4-20250514" {
		t.Errorf("model = %q, want %q", provider.model, "claude-opus-4-20250514")
	}
}

func TestParseAnthropicResponse_NoContent(t *testing.T) {
	body := []byte(`{"content": []}`)
	_, err := parseAnthropicResponse(body)
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestParseAnthropicResponse_ErrorResponse(t *testing.T) {
	body := []byte(`{"error": {"type": "rate_limit_error", "message": "too many requests"}}`)
	_, err := parseAnthropicResponse(body)
	if err == nil {
		t.Error("expected error for error response")
	}
}
