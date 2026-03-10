package email

import (
	"context"
	"strings"
	"testing"

	"github.com/zajca/zfaktury/internal/config"
)

func TestIsConfigured(t *testing.T) {
	t.Run("returns false when host is empty", func(t *testing.T) {
		s := NewEmailSender(config.SMTPConfig{})
		if s.IsConfigured() {
			t.Error("expected IsConfigured() == false for empty host")
		}
	})

	t.Run("returns true when host is set", func(t *testing.T) {
		s := NewEmailSender(config.SMTPConfig{Host: "smtp.example.com", Port: 587})
		if !s.IsConfigured() {
			t.Error("expected IsConfigured() == true when host is set")
		}
	})
}

func TestSend_NotConfigured(t *testing.T) {
	s := NewEmailSender(config.SMTPConfig{})
	err := s.Send(context.Background(), EmailMessage{
		To:      []string{"test@example.com"},
		Subject: "Test",
	})
	if err == nil {
		t.Fatal("expected error when SMTP not configured")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestSend_ContextCancelled(t *testing.T) {
	s := NewEmailSender(config.SMTPConfig{Host: "smtp.example.com", Port: 587})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := s.Send(ctx, EmailMessage{
		To:      []string{"test@example.com"},
		Subject: "Test",
	})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestCollectRecipients(t *testing.T) {
	msg := EmailMessage{
		To:  []string{"a@example.com", "b@example.com"},
		Cc:  []string{"c@example.com", "a@example.com"}, // duplicate of To
		Bcc: []string{"d@example.com"},
	}
	recipients := collectRecipients(msg)

	// Expect 4 unique addresses.
	if len(recipients) != 4 {
		t.Errorf("expected 4 recipients, got %d: %v", len(recipients), recipients)
	}

	seen := make(map[string]bool)
	for _, r := range recipients {
		if seen[r] {
			t.Errorf("duplicate recipient: %s", r)
		}
		seen[r] = true
	}
}

func TestBuildRawMessage_Headers(t *testing.T) {
	msg := EmailMessage{
		To:       []string{"to@example.com"},
		Cc:       []string{"cc@example.com"},
		Subject:  "Hello World",
		BodyText: "Plain text body",
		BodyHTML: "<p>HTML body</p>",
	}

	raw, err := buildRawMessage("from@example.com", msg)
	if err != nil {
		t.Fatalf("buildRawMessage error: %v", err)
	}

	rawStr := string(raw)

	checks := []struct {
		name    string
		contain string
	}{
		{"From header", "From: from@example.com"},
		{"To header", "To: to@example.com"},
		{"Cc header", "Cc: cc@example.com"},
		{"Subject header", "Subject: Hello World"},
		{"MIME-Version header", "MIME-Version: 1.0"},
		{"multipart/alternative", "multipart/alternative"},
		{"text/plain part", "text/plain"},
		{"text/html part", "text/html"},
	}

	for _, c := range checks {
		if !strings.Contains(rawStr, c.contain) {
			t.Errorf("%s: expected to find %q in message", c.name, c.contain)
		}
	}
}

func TestBuildRawMessage_CzechSubject(t *testing.T) {
	msg := EmailMessage{
		To:       []string{"to@example.com"},
		Subject:  "Faktura 2024-001 - Novák s.r.o.",
		BodyText: "Text",
		BodyHTML: "<p>HTML</p>",
	}

	raw, err := buildRawMessage("from@example.com", msg)
	if err != nil {
		t.Fatalf("buildRawMessage error: %v", err)
	}

	rawStr := string(raw)
	// Czech characters require RFC 2047 encoding.
	if !strings.Contains(rawStr, "=?UTF-8?B?") {
		t.Error("expected RFC 2047 encoded subject for Czech characters")
	}
}

func TestBuildRawMessage_WithAttachment(t *testing.T) {
	msg := EmailMessage{
		To:       []string{"to@example.com"},
		Subject:  "Invoice with PDF",
		BodyText: "See attachment",
		BodyHTML: "<p>See attachment</p>",
		Attachments: []Attachment{
			{
				Filename:    "invoice.pdf",
				ContentType: "application/pdf",
				Data:        []byte("%PDF-1.4 fake pdf data"),
			},
		},
	}

	raw, err := buildRawMessage("from@example.com", msg)
	if err != nil {
		t.Fatalf("buildRawMessage error: %v", err)
	}

	rawStr := string(raw)

	checks := []struct {
		name    string
		contain string
	}{
		{"multipart/mixed", "multipart/mixed"},
		{"attachment disposition", `attachment; filename="invoice.pdf"`},
		{"pdf content type", "application/pdf"},
		{"base64 encoding", "base64"},
	}

	for _, c := range checks {
		if !strings.Contains(rawStr, c.contain) {
			t.Errorf("%s: expected to find %q", c.name, c.contain)
		}
	}
}

func TestBuildRawMessage_NoCcWhenEmpty(t *testing.T) {
	msg := EmailMessage{
		To:       []string{"to@example.com"},
		Subject:  "Test",
		BodyText: "Text",
		BodyHTML: "<p>HTML</p>",
	}

	raw, err := buildRawMessage("from@example.com", msg)
	if err != nil {
		t.Fatalf("buildRawMessage error: %v", err)
	}

	rawStr := string(raw)
	if strings.Contains(rawStr, "Cc:") {
		t.Error("Cc header should not be present when Cc is empty")
	}
}

func TestEncodeHeader(t *testing.T) {
	cases := []struct {
		input     string
		wantEnc   bool // whether RFC 2047 encoding is expected
	}{
		{"Hello World", false},
		{"Faktura Novák", true},
		{"Invoice 2024-001", false},
		{"Dvořák účetnictví", true},
	}

	for _, c := range cases {
		result := encodeHeader(c.input)
		hasEnc := strings.HasPrefix(result, "=?UTF-8?B?")
		if hasEnc != c.wantEnc {
			t.Errorf("encodeHeader(%q): wantEncoded=%v, got=%q", c.input, c.wantEnc, result)
		}
	}
}
