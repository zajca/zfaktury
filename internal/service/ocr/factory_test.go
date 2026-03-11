package ocr

import (
	"testing"
)

func TestNewProvider_Defaults(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		wantName     string
	}{
		{"openai explicit", "openai", "openai"},
		{"openai empty", "", "openai"},
		{"openrouter", "openrouter", "openrouter"},
		{"mistral", "mistral", "mistral"},
		{"gemini", "gemini", "gemini"},
		{"claude", "claude", "claude"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProvider(tt.providerName, "test-key", "", "")
			if err != nil {
				t.Fatalf("NewProvider() error: %v", err)
			}
			if p.Name() != tt.wantName {
				t.Errorf("Name() = %q, want %q", p.Name(), tt.wantName)
			}
		})
	}
}

func TestNewProvider_UnknownProvider(t *testing.T) {
	_, err := NewProvider("unknown", "key", "", "")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestNewProvider_BaseURLOverride(t *testing.T) {
	customURL := "https://custom.example.com/v1/chat/completions"

	p, err := NewProvider("openai", "key", "", customURL)
	if err != nil {
		t.Fatalf("NewProvider() error: %v", err)
	}
	oai, ok := p.(*OpenAICompatibleProvider)
	if !ok {
		t.Fatal("expected *OpenAICompatibleProvider")
	}
	if oai.baseURL != customURL {
		t.Errorf("baseURL = %q, want %q", oai.baseURL, customURL)
	}
}

func TestNewProvider_BaseURLOverride_Claude(t *testing.T) {
	customURL := "https://custom.example.com/v1/messages"

	p, err := NewProvider("claude", "key", "", customURL)
	if err != nil {
		t.Fatalf("NewProvider() error: %v", err)
	}
	ap, ok := p.(*AnthropicProvider)
	if !ok {
		t.Fatal("expected *AnthropicProvider")
	}
	if ap.baseURL != customURL {
		t.Errorf("baseURL = %q, want %q", ap.baseURL, customURL)
	}
}

func TestNewProvider_ModelOverride(t *testing.T) {
	p, err := NewProvider("openai", "key", "gpt-4o-mini", "")
	if err != nil {
		t.Fatalf("NewProvider() error: %v", err)
	}
	oai := p.(*OpenAICompatibleProvider)
	if oai.model != "gpt-4o-mini" {
		t.Errorf("model = %q, want %q", oai.model, "gpt-4o-mini")
	}
}
