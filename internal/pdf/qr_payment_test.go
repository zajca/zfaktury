package pdf

import (
	"strings"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

func TestGenerateSPDString(t *testing.T) {
	invoice := &domain.Invoice{
		InvoiceNumber:  "FV2026001",
		VariableSymbol: "2026001",
		ConstantSymbol: "0308",
		TotalAmount:    domain.NewAmount(15000, 0), // 15000.00 CZK
		DueDate:        time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
	}

	iban := "CZ5855000000001265098001"
	swift := "RZBCCZPP"

	spd, err := GenerateSPDString(invoice, iban, swift)
	if err != nil {
		t.Fatalf("GenerateSPDString() error = %v", err)
	}

	// SPD must start with header.
	if !strings.HasPrefix(spd, "SPD*1.0*") {
		t.Errorf("SPD string should start with SPD*1.0*, got %q", spd)
	}

	// Check IBAN is present.
	if !strings.Contains(spd, "ACC:"+iban) {
		t.Errorf("SPD should contain ACC:%s, got %q", iban, spd)
	}

	// Check amount.
	if !strings.Contains(spd, "AM:15000.00") {
		t.Errorf("SPD should contain AM:15000.00, got %q", spd)
	}

	// Check currency.
	if !strings.Contains(spd, "CC:CZK") {
		t.Errorf("SPD should contain CC:CZK, got %q", spd)
	}

	// Check variable symbol.
	if !strings.Contains(spd, "X-VS:2026001") {
		t.Errorf("SPD should contain X-VS:2026001, got %q", spd)
	}

	// Check constant symbol.
	if !strings.Contains(spd, "X-KS:0308") {
		t.Errorf("SPD should contain X-KS:0308, got %q", spd)
	}

	// Check message contains invoice number.
	if !strings.Contains(spd, "MSG:FV2026001") {
		t.Errorf("SPD should contain MSG:FV2026001, got %q", spd)
	}
}

func TestGenerateSPDString_NoIBAN(t *testing.T) {
	invoice := &domain.Invoice{
		TotalAmount: domain.NewAmount(100, 0),
	}

	_, err := GenerateSPDString(invoice, "", "")
	if err == nil {
		t.Error("GenerateSPDString() should return error when IBAN is empty")
	}
}

func TestGenerateSPDString_MinimalFields(t *testing.T) {
	invoice := &domain.Invoice{
		TotalAmount: domain.NewAmount(500, 50), // 500.50 CZK
	}

	spd, err := GenerateSPDString(invoice, "CZ5855000000001265098001", "")
	if err != nil {
		t.Fatalf("GenerateSPDString() error = %v", err)
	}

	if !strings.HasPrefix(spd, "SPD*1.0*") {
		t.Errorf("SPD should start with SPD*1.0*, got %q", spd)
	}

	if !strings.Contains(spd, "AM:500.50") {
		t.Errorf("SPD should contain AM:500.50, got %q", spd)
	}

	// No variable symbol expected.
	if strings.Contains(spd, "X-VS:") {
		t.Errorf("SPD should not contain X-VS when variable symbol is empty, got %q", spd)
	}
}

func TestGenerateQRPayment(t *testing.T) {
	invoice := &domain.Invoice{
		InvoiceNumber:  "FV2026001",
		VariableSymbol: "2026001",
		TotalAmount:    domain.NewAmount(1000, 0),
		DueDate:        time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC),
	}

	pngBytes, err := GenerateQRPayment(invoice, "CZ5855000000001265098001", "RZBCCZPP")
	if err != nil {
		t.Fatalf("GenerateQRPayment() error = %v", err)
	}

	if len(pngBytes) == 0 {
		t.Error("GenerateQRPayment() returned empty bytes")
	}

	// PNG files start with specific magic bytes.
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47}
	if len(pngBytes) < 4 {
		t.Fatal("GenerateQRPayment() returned too few bytes to be a valid PNG")
	}
	for i, b := range pngMagic {
		if pngBytes[i] != b {
			t.Errorf("GenerateQRPayment() byte %d = %x, want %x (not a valid PNG)", i, pngBytes[i], b)
		}
	}
}

func TestGenerateQRPayment_NoIBAN(t *testing.T) {
	invoice := &domain.Invoice{
		TotalAmount: domain.NewAmount(100, 0),
	}

	_, err := GenerateQRPayment(invoice, "", "")
	if err == nil {
		t.Error("GenerateQRPayment() should return error when IBAN is empty")
	}
}
