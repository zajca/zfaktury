package config

import (
	"strings"
	"testing"
)

func validConfig() *Config {
	return &Config{
		Server: ServerConfig{Port: 8080},
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidate_PortZeroIsValid(t *testing.T) {
	cfg := validConfig()
	cfg.Server.Port = 0
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected port 0 to be valid (random port), got: %v", err)
	}
}

func TestValidate_InvalidPort(t *testing.T) {
	for _, port := range []int{-1, 65536, 100000} {
		cfg := validConfig()
		cfg.Server.Port = port
		err := cfg.Validate()
		if err == nil {
			t.Fatalf("expected error for port %d, got nil", port)
		}
		if !strings.Contains(err.Error(), "server.port must be 0-65535") {
			t.Fatalf("unexpected error message: %v", err)
		}
	}
}

func TestValidate_SMTPHostWithoutPort(t *testing.T) {
	cfg := validConfig()
	cfg.SMTP.Host = "smtp.example.com"
	cfg.SMTP.Port = 0
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for SMTP host without port, got nil")
	}
	if !strings.Contains(err.Error(), "smtp.port is required when smtp.host is set") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidate_SMTPHostWithPort(t *testing.T) {
	cfg := validConfig()
	cfg.SMTP.Host = "smtp.example.com"
	cfg.SMTP.Port = 587
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidate_OCRProviderWithoutAPIKey(t *testing.T) {
	cfg := validConfig()
	cfg.OCR.Provider = "openai"
	cfg.OCR.APIKey = ""
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for OCR provider without API key, got nil")
	}
	if !strings.Contains(err.Error(), "ocr.api_key is required when ocr.provider is set") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidate_OCRProviderWithAPIKey(t *testing.T) {
	cfg := validConfig()
	cfg.OCR.Provider = "openai"
	cfg.OCR.APIKey = "sk-test"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: -1},
		SMTP:   SMTPConfig{Host: "smtp.example.com", Port: 0},
		OCR:    OCRConfig{Provider: "openai", APIKey: ""},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for multiple validation failures, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "server.port must be 0-65535") {
		t.Fatalf("missing port error in: %s", msg)
	}
	if !strings.Contains(msg, "smtp.port is required") {
		t.Fatalf("missing SMTP error in: %s", msg)
	}
	if !strings.Contains(msg, "ocr.api_key is required") {
		t.Fatalf("missing OCR error in: %s", msg)
	}
}
