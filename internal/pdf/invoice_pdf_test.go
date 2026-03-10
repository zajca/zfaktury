package pdf

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

func TestInvoicePDFGenerator_Generate(t *testing.T) {
	gen := NewInvoicePDFGenerator()

	invoice := &domain.Invoice{
		ID:             1,
		InvoiceNumber:  "FV2026001",
		Type:           domain.InvoiceTypeRegular,
		Status:         domain.InvoiceStatusSent,
		IssueDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		DueDate:        time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		DeliveryDate:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		VariableSymbol: "2026001",
		ConstantSymbol: "0308",
		CurrencyCode:   domain.CurrencyCZK,
		PaymentMethod:  "bank_transfer",
		BankAccount:    "1265098001",
		BankCode:       "5500",
		IBAN:           "CZ5855000000001265098001",
		SWIFT:          "RZBCCZPP",
		Customer: &domain.Contact{
			ID:      1,
			Name:    "Test Customer s.r.o.",
			ICO:     "12345678",
			DIC:     "CZ12345678",
			Street:  "Testovaci 123",
			City:    "Praha",
			ZIP:     "11000",
			Country: "CZ",
		},
		Items: []domain.InvoiceItem{
			{
				ID:             1,
				Description:    "Web development",
				Quantity:       domain.NewAmount(10, 0), // 10.00
				Unit:           "hod",
				UnitPrice:      domain.NewAmount(1500, 0), // 1500.00 CZK
				VATRatePercent: 21,
				VATAmount:      domain.NewAmount(3150, 0),  // 3150.00 CZK
				TotalAmount:    domain.NewAmount(18150, 0), // 18150.00 CZK
			},
			{
				ID:             2,
				Description:    "Hosting",
				Quantity:       domain.NewAmount(1, 0), // 1.00
				Unit:           "ks",
				UnitPrice:      domain.NewAmount(500, 0), // 500.00 CZK
				VATRatePercent: 21,
				VATAmount:      domain.NewAmount(105, 0), // 105.00 CZK
				TotalAmount:    domain.NewAmount(605, 0), // 605.00 CZK
			},
		},
		SubtotalAmount: domain.NewAmount(15500, 0),
		VATAmount:      domain.NewAmount(3255, 0),
		TotalAmount:    domain.NewAmount(18755, 0),
	}

	supplier := SupplierInfo{
		Name:          "Test Dodavatel",
		ICO:           "87654321",
		DIC:           "CZ87654321",
		VATRegistered: true,
		Street:        "Dodavatelska 456",
		City:          "Brno",
		ZIP:           "60200",
		Email:         "info@dodavatel.cz",
		Phone:         "+420 123 456 789",
		BankAccount:   "1265098001",
		BankCode:      "5500",
		IBAN:          "CZ5855000000001265098001",
		SWIFT:         "RZBCCZPP",
	}

	pdfBytes, err := gen.Generate(context.Background(), invoice, supplier)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Error("Generate() returned empty bytes")
	}

	// PDF files start with %PDF.
	if len(pdfBytes) < 4 {
		t.Fatal("Generate() returned too few bytes to be a valid PDF")
	}
	if string(pdfBytes[:4]) != "%PDF" {
		t.Errorf("Generate() output does not start with %%PDF magic bytes, got %q", string(pdfBytes[:4]))
	}
}

func TestInvoicePDFGenerator_Generate_NonVATRegistered(t *testing.T) {
	gen := NewInvoicePDFGenerator()

	invoice := &domain.Invoice{
		ID:            1,
		InvoiceNumber: "FV2026002",
		Type:          domain.InvoiceTypeRegular,
		Status:        domain.InvoiceStatusDraft,
		IssueDate:     time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		DeliveryDate:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		CurrencyCode:  domain.CurrencyCZK,
		Items: []domain.InvoiceItem{
			{
				Description:    "Service",
				Quantity:       domain.NewAmount(1, 0),
				Unit:           "ks",
				UnitPrice:      domain.NewAmount(1000, 0),
				VATRatePercent: 0,
				VATAmount:      0,
				TotalAmount:    domain.NewAmount(1000, 0),
			},
		},
		SubtotalAmount: domain.NewAmount(1000, 0),
		VATAmount:      0,
		TotalAmount:    domain.NewAmount(1000, 0),
	}

	supplier := SupplierInfo{
		Name:          "OSVC Tester",
		ICO:           "11111111",
		VATRegistered: false,
		City:          "Praha",
		ZIP:           "11000",
	}

	pdfBytes, err := gen.Generate(context.Background(), invoice, supplier)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Error("Generate() returned empty bytes")
	}
}

func TestInvoicePDFGenerator_Generate_NoCustomer(t *testing.T) {
	gen := NewInvoicePDFGenerator()

	invoice := &domain.Invoice{
		ID:            1,
		InvoiceNumber: "FV2026003",
		Type:          domain.InvoiceTypeProforma,
		Status:        domain.InvoiceStatusDraft,
		IssueDate:     time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		DueDate:       time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		DeliveryDate:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		CurrencyCode:  domain.CurrencyCZK,
		Customer:      nil,
		Items: []domain.InvoiceItem{
			{
				Description:    "Consulting",
				Quantity:       domain.NewAmount(5, 0),
				Unit:           "hod",
				UnitPrice:      domain.NewAmount(2000, 0),
				VATRatePercent: 0,
			},
		},
		SubtotalAmount: domain.NewAmount(10000, 0),
		TotalAmount:    domain.NewAmount(10000, 0),
	}

	supplier := SupplierInfo{
		Name: "Test OSVC",
		ICO:  "99999999",
	}

	pdfBytes, err := gen.Generate(context.Background(), invoice, supplier)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Error("Generate() returned empty bytes")
	}
}
