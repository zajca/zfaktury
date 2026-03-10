package email

import (
	"strings"
	"testing"
)

func sampleInvoiceData() InvoiceEmailData {
	return InvoiceEmailData{
		InvoiceNumber:  "2024-001",
		CustomerName:   "Novák s.r.o.",
		IssueDate:      "01.01.2024",
		DueDate:        "15.01.2024",
		TotalFormatted: "12 100,00",
		CurrencyCode:   "CZK",
		PaymentMethod:  "Bankovní převod",
		BankAccount:    "123456789/0300",
		VariableSymbol: "2024001",
		SenderName:     "Jan Svoboda",
		Items: []InvoiceEmailItem{
			{
				Description: "Vývoj webové aplikace",
				Quantity:    "10",
				UnitPrice:   "1 210,00 CZK",
				Total:       "12 100,00 CZK",
			},
		},
	}
}

func TestRenderInvoiceEmail_Subject(t *testing.T) {
	data := sampleInvoiceData()
	subject, _, _, err := RenderInvoiceEmail(data)
	if err != nil {
		t.Fatalf("RenderInvoiceEmail error: %v", err)
	}

	expected := "Faktura 2024-001 - Jan Svoboda"
	if subject != expected {
		t.Errorf("subject = %q, want %q", subject, expected)
	}
}

func TestRenderInvoiceEmail_HTMLContainsKeyFields(t *testing.T) {
	data := sampleInvoiceData()
	_, htmlBody, _, err := RenderInvoiceEmail(data)
	if err != nil {
		t.Fatalf("RenderInvoiceEmail error: %v", err)
	}

	checks := []struct {
		name  string
		value string
	}{
		{"invoice number", "2024-001"},
		{"customer name", "Novák s.r.o."},
		{"issue date", "01.01.2024"},
		{"due date", "15.01.2024"},
		{"total amount", "12 100,00"},
		{"currency", "CZK"},
		{"bank account", "123456789/0300"},
		{"variable symbol", "2024001"},
		{"sender name", "Jan Svoboda"},
		{"item description", "Vývoj webové aplikace"},
		{"payment method", "Bankovní převod"},
	}

	for _, c := range checks {
		if !strings.Contains(htmlBody, c.value) {
			t.Errorf("HTML body missing %s (%q)", c.name, c.value)
		}
	}
}

func TestRenderInvoiceEmail_HTMLIsValidMarkup(t *testing.T) {
	data := sampleInvoiceData()
	_, htmlBody, _, err := RenderInvoiceEmail(data)
	if err != nil {
		t.Fatalf("RenderInvoiceEmail error: %v", err)
	}

	if !strings.Contains(htmlBody, "<!DOCTYPE html>") {
		t.Error("HTML body should contain DOCTYPE declaration")
	}
	if !strings.Contains(htmlBody, "<html") {
		t.Error("HTML body should contain <html> tag")
	}
	if !strings.Contains(htmlBody, "</html>") {
		t.Error("HTML body should contain closing </html> tag")
	}
	if !strings.Contains(htmlBody, "lang=\"cs\"") {
		t.Error("HTML body should have Czech language attribute")
	}
}

func TestRenderInvoiceEmail_TextContainsKeyFields(t *testing.T) {
	data := sampleInvoiceData()
	_, _, textBody, err := RenderInvoiceEmail(data)
	if err != nil {
		t.Fatalf("RenderInvoiceEmail error: %v", err)
	}

	checks := []struct {
		name  string
		value string
	}{
		{"invoice number", "2024-001"},
		{"customer name", "Novák s.r.o."},
		{"issue date", "01.01.2024"},
		{"due date", "15.01.2024"},
		{"total amount", "12 100,00"},
		{"currency", "CZK"},
		{"bank account", "123456789/0300"},
		{"variable symbol", "2024001"},
		{"sender name", "Jan Svoboda"},
	}

	for _, c := range checks {
		if !strings.Contains(textBody, c.value) {
			t.Errorf("text body missing %s (%q)", c.name, c.value)
		}
	}
}

func TestRenderInvoiceEmail_NoItemsSection(t *testing.T) {
	data := sampleInvoiceData()
	data.Items = nil

	_, htmlBody, textBody, err := RenderInvoiceEmail(data)
	if err != nil {
		t.Fatalf("RenderInvoiceEmail error: %v", err)
	}

	// Templates should still render without error even with no items.
	if htmlBody == "" {
		t.Error("HTML body should not be empty")
	}
	if textBody == "" {
		t.Error("text body should not be empty")
	}
}

func TestRenderInvoiceEmail_NoBankAccount(t *testing.T) {
	data := sampleInvoiceData()
	data.BankAccount = ""
	data.VariableSymbol = ""
	data.PaymentMethod = ""

	_, htmlBody, textBody, err := RenderInvoiceEmail(data)
	if err != nil {
		t.Fatalf("RenderInvoiceEmail error: %v", err)
	}

	// Payment block should be omitted when bank account is empty.
	if strings.Contains(htmlBody, "Platební údaje") {
		t.Error("HTML body should not contain payment section when BankAccount is empty")
	}
	if strings.Contains(textBody, "PLATEBNÍ ÚDAJE") {
		t.Error("text body should not contain payment section when BankAccount is empty")
	}
}

func TestRenderInvoiceEmail_SubjectFormat(t *testing.T) {
	cases := []struct {
		invoiceNumber string
		senderName    string
		expected      string
	}{
		{"2024-001", "Jan Novák", "Faktura 2024-001 - Jan Novák"},
		{"F-2025-0042", "ACME s.r.o.", "Faktura F-2025-0042 - ACME s.r.o."},
		{"INV-100", "Petra Dvořáková", "Faktura INV-100 - Petra Dvořáková"},
	}

	for _, c := range cases {
		data := InvoiceEmailData{
			InvoiceNumber: c.invoiceNumber,
			SenderName:    c.senderName,
		}
		subject, _, _, err := RenderInvoiceEmail(data)
		if err != nil {
			t.Fatalf("RenderInvoiceEmail error: %v", err)
		}
		if subject != c.expected {
			t.Errorf("subject = %q, want %q", subject, c.expected)
		}
	}
}
