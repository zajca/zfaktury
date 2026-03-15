package isdoc

import (
	"context"
	"encoding/xml"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func testInvoice() *domain.Invoice {
	inv := &domain.Invoice{
		ID:             1,
		InvoiceNumber:  "FV20260001",
		Type:           domain.InvoiceTypeRegular,
		Status:         domain.InvoiceStatusSent,
		IssueDate:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		DueDate:        time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		DeliveryDate:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		VariableSymbol: "20260001",
		ConstantSymbol: "0308",
		CustomerID:     10,
		CurrencyCode:   "CZK",
		PaymentMethod:  "bank_transfer",
		BankAccount:    "1234567890",
		BankCode:       "0100",
		IBAN:           "CZ6508000000001234567890",
		SWIFT:          "KOMBCZPP",
		Notes:          "Test invoice",
		Customer: &domain.Contact{
			ID:      10,
			Type:    domain.ContactTypeCompany,
			Name:    "Test Customer s.r.o.",
			ICO:     "12345678",
			DIC:     "CZ12345678",
			Street:  "Testovaci 123",
			City:    "Praha",
			ZIP:     "11000",
			Country: "CZ",
			Email:   "customer@test.cz",
			Phone:   "+420123456789",
		},
		Items: []domain.InvoiceItem{
			{
				ID:             1,
				InvoiceID:      1,
				Description:    "Web development",
				Quantity:       domain.NewAmount(10, 0), // 10.00
				Unit:           "hod",
				UnitPrice:      domain.NewAmount(1500, 0), // 1500.00 CZK
				VATRatePercent: 21,
				SortOrder:      1,
			},
			{
				ID:             2,
				InvoiceID:      1,
				Description:    "Hosting",
				Quantity:       domain.NewAmount(1, 0), // 1.00
				Unit:           "ks",
				UnitPrice:      domain.NewAmount(500, 0), // 500.00 CZK
				VATRatePercent: 21,
				SortOrder:      2,
			},
		},
	}
	inv.CalculateTotals()
	return inv
}

func testSupplier() SupplierInfo {
	return SupplierInfo{
		CompanyName: "Jan Novak",
		ICO:         "87654321",
		DIC:         "CZ87654321",
		Street:      "Dodavatelska 456",
		City:        "Brno",
		ZIP:         "60200",
		Email:       "jan@novak.cz",
		Phone:       "+420987654321",
		BankAccount: "9876543210",
		BankCode:    "0100",
		IBAN:        "CZ6508000000009876543210",
		SWIFT:       "KOMBCZPP",
	}
}

func TestGenerate_XMLDeclaration(t *testing.T) {
	gen := NewISDOCGenerator()
	data, err := gen.Generate(context.Background(), testInvoice(), testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	xml := string(data)
	if !strings.HasPrefix(xml, `<?xml version="1.0" encoding="UTF-8"?>`) {
		t.Error("XML output should start with XML declaration")
	}
}

func TestGenerate_Namespace(t *testing.T) {
	gen := NewISDOCGenerator()
	data, err := gen.Generate(context.Background(), testInvoice(), testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(string(data), ISDOCNamespace) {
		t.Errorf("XML should contain ISDOC namespace %q", ISDOCNamespace)
	}
}

func TestGenerate_ValidXML(t *testing.T) {
	gen := NewISDOCGenerator()
	data, err := gen.Generate(context.Background(), testInvoice(), testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify the XML can be parsed back.
	var doc Invoice
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Generated XML is not valid: %v", err)
	}
}

func TestGenerate_DocumentType(t *testing.T) {
	tests := []struct {
		invType  string
		expected int
	}{
		{domain.InvoiceTypeRegular, 1},
		{domain.InvoiceTypeCreditNote, 2},
		{domain.InvoiceTypeProforma, 4},
	}

	gen := NewISDOCGenerator()
	for _, tt := range tests {
		t.Run(tt.invType, func(t *testing.T) {
			inv := testInvoice()
			inv.Type = tt.invType

			data, err := gen.Generate(context.Background(), inv, testSupplier())
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			var doc Invoice
			if err := xml.Unmarshal(data, &doc); err != nil {
				t.Fatalf("Failed to parse XML: %v", err)
			}

			if doc.DocumentType != tt.expected {
				t.Errorf("DocumentType = %d, want %d", doc.DocumentType, tt.expected)
			}
		})
	}
}

func TestGenerate_InvoiceID(t *testing.T) {
	gen := NewISDOCGenerator()
	inv := testInvoice()

	data, err := gen.Generate(context.Background(), inv, testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var doc Invoice
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	if doc.ID != "FV20260001" {
		t.Errorf("ID = %q, want %q", doc.ID, "FV20260001")
	}
}

func TestGenerate_Dates(t *testing.T) {
	gen := NewISDOCGenerator()
	inv := testInvoice()

	data, err := gen.Generate(context.Background(), inv, testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var doc Invoice
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	if doc.IssueDate != "2026-03-01" {
		t.Errorf("IssueDate = %q, want %q", doc.IssueDate, "2026-03-01")
	}
	if doc.TaxPointDate != "2026-03-01" {
		t.Errorf("TaxPointDate = %q, want %q", doc.TaxPointDate, "2026-03-01")
	}
}

func TestGenerate_SupplierParty(t *testing.T) {
	gen := NewISDOCGenerator()
	data, err := gen.Generate(context.Background(), testInvoice(), testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var doc Invoice
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	sp := doc.AccountingSupplierParty.Party
	if sp.PartyName.Name != "Jan Novak" {
		t.Errorf("Supplier name = %q, want %q", sp.PartyName.Name, "Jan Novak")
	}
	if sp.PartyIdentification.ID != "87654321" {
		t.Errorf("Supplier ICO = %q, want %q", sp.PartyIdentification.ID, "87654321")
	}
	if sp.PartyTaxScheme == nil || sp.PartyTaxScheme.CompanyID != "CZ87654321" {
		t.Error("Supplier DIC should be CZ87654321")
	}
	if sp.PostalAddress.CityName != "Brno" {
		t.Errorf("Supplier city = %q, want %q", sp.PostalAddress.CityName, "Brno")
	}
}

func TestGenerate_CustomerParty(t *testing.T) {
	gen := NewISDOCGenerator()
	data, err := gen.Generate(context.Background(), testInvoice(), testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var doc Invoice
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	cp := doc.AccountingCustomerParty.Party
	if cp.PartyName.Name != "Test Customer s.r.o." {
		t.Errorf("Customer name = %q, want %q", cp.PartyName.Name, "Test Customer s.r.o.")
	}
	if cp.PartyIdentification.ID != "12345678" {
		t.Errorf("Customer ICO = %q, want %q", cp.PartyIdentification.ID, "12345678")
	}
}

func TestGenerate_InvoiceLines(t *testing.T) {
	gen := NewISDOCGenerator()
	data, err := gen.Generate(context.Background(), testInvoice(), testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var doc Invoice
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	if len(doc.InvoiceLines.InvoiceLine) != 2 {
		t.Fatalf("Expected 2 invoice lines, got %d", len(doc.InvoiceLines.InvoiceLine))
	}

	line1 := doc.InvoiceLines.InvoiceLine[0]
	if line1.Item.Description != "Web development" {
		t.Errorf("Line 1 description = %q, want %q", line1.Item.Description, "Web development")
	}
	if line1.ID != "1" {
		t.Errorf("Line 1 ID = %q, want %q", line1.ID, "1")
	}
}

func TestGenerate_TaxTotal(t *testing.T) {
	gen := NewISDOCGenerator()
	inv := testInvoice()

	data, err := gen.Generate(context.Background(), inv, testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var doc Invoice
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	if doc.TaxTotal.TaxAmount != inv.VATAmount.String() {
		t.Errorf("TaxAmount = %q, want %q", doc.TaxTotal.TaxAmount, inv.VATAmount.String())
	}

	if len(doc.TaxTotal.TaxSubTotal) == 0 {
		t.Error("Expected at least one TaxSubTotal")
	}
}

func TestGenerate_LegalMonetaryTotal(t *testing.T) {
	gen := NewISDOCGenerator()
	inv := testInvoice()

	data, err := gen.Generate(context.Background(), inv, testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var doc Invoice
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	if doc.LegalMonetaryTotal.TaxExclusiveAmount != inv.SubtotalAmount.String() {
		t.Errorf("TaxExclusiveAmount = %q, want %q", doc.LegalMonetaryTotal.TaxExclusiveAmount, inv.SubtotalAmount.String())
	}
	if doc.LegalMonetaryTotal.TaxInclusiveAmount != inv.TotalAmount.String() {
		t.Errorf("TaxInclusiveAmount = %q, want %q", doc.LegalMonetaryTotal.TaxInclusiveAmount, inv.TotalAmount.String())
	}
}

func TestGenerate_PaymentMeans(t *testing.T) {
	gen := NewISDOCGenerator()
	inv := testInvoice()

	data, err := gen.Generate(context.Background(), inv, testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var doc Invoice
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	if doc.PaymentMeans == nil {
		t.Fatal("PaymentMeans should not be nil")
	}
	if doc.PaymentMeans.Payment.PaymentMeansCode != 42 {
		t.Errorf("PaymentMeansCode = %d, want 42 (bank transfer)", doc.PaymentMeans.Payment.PaymentMeansCode)
	}
	if doc.PaymentMeans.Payment.Details == nil {
		t.Fatal("Payment.Details should not be nil")
	}
	if doc.PaymentMeans.Payment.Details.VariableSymbol != "20260001" {
		t.Errorf("VariableSymbol = %q, want %q", doc.PaymentMeans.Payment.Details.VariableSymbol, "20260001")
	}
	if doc.PaymentMeans.Payment.Details.IBAN != "CZ6508000000001234567890" {
		t.Errorf("IBAN = %q, want %q", doc.PaymentMeans.Payment.Details.IBAN, "CZ6508000000001234567890")
	}
}

func TestGenerate_CashPayment(t *testing.T) {
	gen := NewISDOCGenerator()
	inv := testInvoice()
	inv.PaymentMethod = "cash"
	inv.BankAccount = ""
	inv.IBAN = ""

	data, err := gen.Generate(context.Background(), inv, testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var doc Invoice
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	if doc.PaymentMeans.Payment.PaymentMeansCode != 10 {
		t.Errorf("PaymentMeansCode = %d, want 10 (cash)", doc.PaymentMeans.Payment.PaymentMeansCode)
	}
}

func TestGenerate_NilInvoice(t *testing.T) {
	gen := NewISDOCGenerator()
	_, err := gen.Generate(context.Background(), nil, testSupplier())
	if err == nil {
		t.Error("Expected error for nil invoice")
	}
}

func TestGenerate_VATApplicable(t *testing.T) {
	gen := NewISDOCGenerator()

	// Invoice with VAT.
	inv := testInvoice()
	data, err := gen.Generate(context.Background(), inv, testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	var doc Invoice
	xml.Unmarshal(data, &doc)
	if !doc.VATApplicable {
		t.Error("VATApplicable should be true when items have VAT")
	}

	// Invoice without VAT.
	inv2 := testInvoice()
	for i := range inv2.Items {
		inv2.Items[i].VATRatePercent = 0
	}
	inv2.CalculateTotals()
	data2, err := gen.Generate(context.Background(), inv2, testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	var doc2 Invoice
	xml.Unmarshal(data2, &doc2)
	if doc2.VATApplicable {
		t.Error("VATApplicable should be false when no items have VAT")
	}
}

func TestGenerate_ForeignCurrency(t *testing.T) {
	gen := NewISDOCGenerator()
	inv := testInvoice()
	inv.CurrencyCode = "EUR"
	inv.ExchangeRate = domain.NewAmount(25, 34) // 25.34

	data, err := gen.Generate(context.Background(), inv, testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var doc Invoice
	xml.Unmarshal(data, &doc)

	if doc.LocalCurrencyCode != "CZK" {
		t.Errorf("LocalCurrencyCode = %q, want %q", doc.LocalCurrencyCode, "CZK")
	}
	if doc.ForeignCurrencyCode != "EUR" {
		t.Errorf("ForeignCurrencyCode = %q, want %q", doc.ForeignCurrencyCode, "EUR")
	}
	if doc.CurrRate != "25.34" {
		t.Errorf("CurrRate = %q, want %q", doc.CurrRate, "25.34")
	}
}

func TestGenerate_SupplierWithoutDIC(t *testing.T) {
	gen := NewISDOCGenerator()
	supplier := testSupplier()
	supplier.DIC = ""

	data, err := gen.Generate(context.Background(), testInvoice(), supplier)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var doc Invoice
	xml.Unmarshal(data, &doc)

	if doc.AccountingSupplierParty.Party.PartyTaxScheme != nil {
		t.Error("PartyTaxScheme should be nil when DIC is empty")
	}
}

func TestGenerate_CustomerWithoutContact(t *testing.T) {
	gen := NewISDOCGenerator()
	inv := testInvoice()
	inv.Customer = nil

	data, err := gen.Generate(context.Background(), inv, testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var doc Invoice
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Generated XML is not valid: %v", err)
	}

	// Should still have customer party with fallback ID.
	cp := doc.AccountingCustomerParty.Party
	if cp.PartyIdentification.ID != "customer-10" {
		t.Errorf("Customer ID = %q, want %q", cp.PartyIdentification.ID, "customer-10")
	}
}

func goldenPath(name string) string {
	return filepath.Join("testdata", name+".golden.xml")
}

func TestISDOC_Golden_Regular(t *testing.T) {
	gen := NewISDOCGenerator()
	inv := &domain.Invoice{
		ID:             1,
		InvoiceNumber:  "FV20250001",
		Type:           domain.InvoiceTypeRegular,
		Status:         domain.InvoiceStatusSent,
		IssueDate:      time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
		DueDate:        time.Date(2025, 3, 29, 0, 0, 0, 0, time.UTC),
		DeliveryDate:   time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
		VariableSymbol: "20250001",
		ConstantSymbol: "0308",
		CustomerID:     10,
		CurrencyCode:   "CZK",
		PaymentMethod:  "bank_transfer",
		BankAccount:    "1234567890",
		BankCode:       "0100",
		IBAN:           "CZ6508000000001234567890",
		SWIFT:          "KOMBCZPP",
		Notes:          "Fakturujeme Vam za provedene prace.",
		Customer: &domain.Contact{
			ID:      10,
			Type:    domain.ContactTypeCompany,
			Name:    "Acme s.r.o.",
			ICO:     "12345678",
			DIC:     "CZ12345678",
			Street:  "Hlavni 100",
			City:    "Praha",
			ZIP:     "11000",
			Country: "CZ",
			Email:   "info@acme.cz",
			Phone:   "+420111222333",
		},
		Items: []domain.InvoiceItem{
			{
				ID:             1,
				InvoiceID:      1,
				Description:    "Web application development",
				Quantity:       domain.NewAmount(80, 0),
				Unit:           "hod",
				UnitPrice:      domain.NewAmount(1200, 0),
				VATRatePercent: 21,
				SortOrder:      1,
			},
			{
				ID:             2,
				InvoiceID:      1,
				Description:    "Domain registration",
				Quantity:       domain.NewAmount(1, 0),
				Unit:           "ks",
				UnitPrice:      domain.NewAmount(350, 0),
				VATRatePercent: 21,
				SortOrder:      2,
			},
		},
	}
	inv.CalculateTotals()

	data, err := gen.Generate(context.Background(), inv, testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	testutil.AssertGolden(t, goldenPath("regular_invoice"), data)
}

func TestISDOC_Golden_CreditNote(t *testing.T) {
	gen := NewISDOCGenerator()
	relatedID := int64(1)
	inv := &domain.Invoice{
		ID:               2,
		InvoiceNumber:    "DC20250001",
		Type:             domain.InvoiceTypeCreditNote,
		Status:           domain.InvoiceStatusSent,
		IssueDate:        time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC),
		DueDate:          time.Date(2025, 4, 3, 0, 0, 0, 0, time.UTC),
		DeliveryDate:     time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC),
		VariableSymbol:   "20250002",
		ConstantSymbol:   "0308",
		CustomerID:       10,
		CurrencyCode:     "CZK",
		PaymentMethod:    "bank_transfer",
		BankAccount:      "1234567890",
		BankCode:         "0100",
		IBAN:             "CZ6508000000001234567890",
		SWIFT:            "KOMBCZPP",
		Notes:            "Credit note for invoice FV20250001.",
		RelatedInvoiceID: &relatedID,
		RelationType:     domain.RelationTypeCreditNote,
		Customer: &domain.Contact{
			ID:      10,
			Type:    domain.ContactTypeCompany,
			Name:    "Acme s.r.o.",
			ICO:     "12345678",
			DIC:     "CZ12345678",
			Street:  "Hlavni 100",
			City:    "Praha",
			ZIP:     "11000",
			Country: "CZ",
			Email:   "info@acme.cz",
			Phone:   "+420111222333",
		},
		Items: []domain.InvoiceItem{
			{
				ID:             3,
				InvoiceID:      2,
				Description:    "Web application development - correction",
				Quantity:       domain.NewAmount(10, 0),
				Unit:           "hod",
				UnitPrice:      domain.NewAmount(1200, 0),
				VATRatePercent: 21,
				SortOrder:      1,
			},
		},
	}
	inv.CalculateTotals()

	data, err := gen.Generate(context.Background(), inv, testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	testutil.AssertGolden(t, goldenPath("credit_note"), data)
}

func TestISDOC_Golden_ForeignCurrency(t *testing.T) {
	gen := NewISDOCGenerator()
	inv := &domain.Invoice{
		ID:             3,
		InvoiceNumber:  "FV20250003",
		Type:           domain.InvoiceTypeRegular,
		Status:         domain.InvoiceStatusSent,
		IssueDate:      time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
		DueDate:        time.Date(2025, 3, 29, 0, 0, 0, 0, time.UTC),
		DeliveryDate:   time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
		VariableSymbol: "20250003",
		ConstantSymbol: "0308",
		CustomerID:     20,
		CurrencyCode:   "EUR",
		ExchangeRate:   domain.NewAmount(25, 34),
		PaymentMethod:  "bank_transfer",
		BankAccount:    "1234567890",
		BankCode:       "0100",
		IBAN:           "CZ6508000000001234567890",
		SWIFT:          "KOMBCZPP",
		Notes:          "Invoice in EUR.",
		Customer: &domain.Contact{
			ID:      20,
			Type:    domain.ContactTypeCompany,
			Name:    "EuroTech GmbH",
			ICO:     "DE987654",
			DIC:     "DE123456789",
			Street:  "Berliner Str. 42",
			City:    "Berlin",
			ZIP:     "10115",
			Country: "DE",
			Email:   "billing@eurotech.de",
			Phone:   "+49301234567",
		},
		Items: []domain.InvoiceItem{
			{
				ID:             4,
				InvoiceID:      3,
				Description:    "Consulting services",
				Quantity:       domain.NewAmount(40, 0),
				Unit:           "hod",
				UnitPrice:      domain.NewAmount(75, 0),
				VATRatePercent: 0,
				SortOrder:      1,
			},
		},
	}
	inv.CalculateTotals()

	data, err := gen.Generate(context.Background(), inv, testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	testutil.AssertGolden(t, goldenPath("foreign_currency"), data)
}

func TestISDOC_Golden_CashPayment(t *testing.T) {
	gen := NewISDOCGenerator()
	inv := &domain.Invoice{
		ID:             4,
		InvoiceNumber:  "FV20250004",
		Type:           domain.InvoiceTypeRegular,
		Status:         domain.InvoiceStatusPaid,
		IssueDate:      time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
		DueDate:        time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
		DeliveryDate:   time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
		VariableSymbol: "20250004",
		ConstantSymbol: "0308",
		CustomerID:     30,
		CurrencyCode:   "CZK",
		PaymentMethod:  "cash",
		Notes:          "Paid in cash.",
		Customer: &domain.Contact{
			ID:      30,
			Type:    domain.ContactTypeIndividual,
			Name:    "Petr Svoboda",
			ICO:     "98765432",
			Street:  "Namesti Miru 5",
			City:    "Olomouc",
			ZIP:     "77900",
			Country: "CZ",
		},
		Items: []domain.InvoiceItem{
			{
				ID:             5,
				InvoiceID:      4,
				Description:    "IT equipment repair",
				Quantity:       domain.NewAmount(1, 0),
				Unit:           "ks",
				UnitPrice:      domain.NewAmount(2500, 0),
				VATRatePercent: 21,
				SortOrder:      1,
			},
		},
	}
	inv.CalculateTotals()

	data, err := gen.Generate(context.Background(), inv, testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	testutil.AssertGolden(t, goldenPath("cash_payment"), data)
}

func TestGenerate_CustomerWithEmptyCountry(t *testing.T) {
	gen := NewISDOCGenerator()
	inv := testInvoice()
	inv.Customer.Country = ""

	data, err := gen.Generate(context.Background(), inv, testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var doc Invoice
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	cp := doc.AccountingCustomerParty.Party
	if cp.PostalAddress.Country.IdentificationCode != "CZ" {
		t.Errorf("Country code = %q, want %q (default)", cp.PostalAddress.Country.IdentificationCode, "CZ")
	}
}

func TestGenerate_CustomerWithoutDIC(t *testing.T) {
	gen := NewISDOCGenerator()
	inv := testInvoice()
	inv.Customer.DIC = ""

	data, err := gen.Generate(context.Background(), inv, testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var doc Invoice
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	cp := doc.AccountingCustomerParty.Party
	if cp.PartyTaxScheme != nil {
		t.Error("PartyTaxScheme should be nil when customer DIC is empty")
	}
}

func TestGenerate_CustomerWithoutContactInfo(t *testing.T) {
	gen := NewISDOCGenerator()
	inv := testInvoice()
	inv.Customer.Email = ""
	inv.Customer.Phone = ""

	data, err := gen.Generate(context.Background(), inv, testSupplier())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var doc Invoice
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	cp := doc.AccountingCustomerParty.Party
	if cp.Contact != nil {
		t.Error("Contact should be nil when customer has no email or phone")
	}
}

func TestGenerate_SupplierWithoutContactInfo(t *testing.T) {
	gen := NewISDOCGenerator()
	supplier := testSupplier()
	supplier.Email = ""
	supplier.Phone = ""

	data, err := gen.Generate(context.Background(), testInvoice(), supplier)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var doc Invoice
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	sp := doc.AccountingSupplierParty.Party
	if sp.Contact != nil {
		t.Error("Contact should be nil when supplier has no email or phone")
	}
}

func TestMapDocumentType(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"regular", 1},
		{"credit_note", 2},
		{"proforma", 4},
		{"unknown", 1}, // defaults to 1
		{"", 1},
	}

	for _, tt := range tests {
		got := mapDocumentType(tt.input)
		if got != tt.expected {
			t.Errorf("mapDocumentType(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}
