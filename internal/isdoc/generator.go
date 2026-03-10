package isdoc

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/zajca/zfaktury/internal/domain"
)

// SupplierInfo contains the supplier (OSVC) details for ISDOC generation.
// These are loaded from application settings.
type SupplierInfo struct {
	CompanyName string
	ICO         string
	DIC         string
	Street      string
	City        string
	ZIP         string
	Email       string
	Phone       string
	BankAccount string
	BankCode    string
	IBAN        string
	SWIFT       string
}

// ISDOCGenerator generates ISDOC 6.0.2 XML documents from invoices.
type ISDOCGenerator struct{}

// NewISDOCGenerator creates a new ISDOCGenerator.
func NewISDOCGenerator() *ISDOCGenerator {
	return &ISDOCGenerator{}
}

// Generate produces an ISDOC 6.0.2 XML document for the given invoice.
func (g *ISDOCGenerator) Generate(_ context.Context, invoice *domain.Invoice, supplier SupplierInfo) ([]byte, error) {
	if invoice == nil {
		return nil, fmt.Errorf("invoice is required")
	}

	doc := g.buildDocument(invoice, supplier)

	output, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling ISDOC XML: %w", err)
	}

	// Prepend XML declaration.
	xmlDecl := []byte(xml.Header)
	result := make([]byte, 0, len(xmlDecl)+len(output))
	result = append(result, xmlDecl...)
	result = append(result, output...)

	return result, nil
}

// buildDocument maps a domain.Invoice to the ISDOC XML structure.
func (g *ISDOCGenerator) buildDocument(inv *domain.Invoice, supplier SupplierInfo) Invoice {
	doc := Invoice{
		Xmlns:             ISDOCNamespace,
		Version:           "6.0.2",
		DocumentType:      mapDocumentType(inv.Type),
		ID:                inv.InvoiceNumber,
		UUID:              fmt.Sprintf("zfaktury-%d", inv.ID),
		IssuingSystem:     "ZFaktury",
		IssueDate:         inv.IssueDate.Format("2006-01-02"),
		TaxPointDate:      inv.DeliveryDate.Format("2006-01-02"),
		VATApplicable:     g.hasVAT(inv),
		Note:              inv.Notes,
		LocalCurrencyCode: "CZK",
	}

	// Foreign currency handling.
	if inv.CurrencyCode != "" && inv.CurrencyCode != domain.CurrencyCZK {
		doc.ForeignCurrencyCode = inv.CurrencyCode
		doc.CurrRate = inv.ExchangeRate.String()
		doc.RefCurrRate = "1.00"
	}

	// Supplier party.
	doc.AccountingSupplierParty = g.buildSupplierParty(supplier)

	// Customer party.
	doc.AccountingCustomerParty = g.buildCustomerParty(inv)

	// Invoice lines.
	doc.InvoiceLines = g.buildInvoiceLines(inv)

	// Tax total.
	doc.TaxTotal = g.buildTaxTotal(inv)

	// Legal monetary total.
	doc.LegalMonetaryTotal = g.buildLegalMonetaryTotal(inv)

	// Payment means.
	doc.PaymentMeans = g.buildPaymentMeans(inv)

	return doc
}

// mapDocumentType converts the domain invoice type to ISDOC document type code.
// ISDOC document types: 1=invoice, 2=credit note, 4=proforma.
func mapDocumentType(invType string) int {
	switch invType {
	case domain.InvoiceTypeRegular:
		return 1
	case domain.InvoiceTypeCreditNote:
		return 2
	case domain.InvoiceTypeProforma:
		return 4
	default:
		return 1
	}
}

// hasVAT checks if any line item has a non-zero VAT rate.
func (g *ISDOCGenerator) hasVAT(inv *domain.Invoice) bool {
	for _, item := range inv.Items {
		if item.VATRatePercent > 0 {
			return true
		}
	}
	return false
}

// buildSupplierParty creates the ISDOC supplier party from SupplierInfo.
func (g *ISDOCGenerator) buildSupplierParty(supplier SupplierInfo) AccountingParty {
	party := Party{
		PartyIdentification: PartyIdentification{
			UserID: supplier.ICO,
			ID:     supplier.ICO,
		},
		PartyName: PartyName{
			Name: supplier.CompanyName,
		},
		PostalAddress: PostalAddress{
			StreetName: supplier.Street,
			CityName:   supplier.City,
			PostalZone: supplier.ZIP,
			Country: Country{
				IdentificationCode: "CZ",
				Name:               "Ceska republika",
			},
		},
	}

	if supplier.DIC != "" {
		party.PartyTaxScheme = &PartyTaxScheme{
			CompanyID: supplier.DIC,
			TaxScheme: TaxScheme{Name: "VAT"},
		}
	}

	if supplier.Email != "" || supplier.Phone != "" {
		party.Contact = &PartyContact{
			Telephone:      supplier.Phone,
			ElectronicMail: supplier.Email,
		}
	}

	return AccountingParty{Party: party}
}

// buildCustomerParty creates the ISDOC customer party from the invoice's customer.
func (g *ISDOCGenerator) buildCustomerParty(inv *domain.Invoice) AccountingParty {
	if inv.Customer == nil {
		return AccountingParty{
			Party: Party{
				PartyIdentification: PartyIdentification{
					UserID: fmt.Sprintf("customer-%d", inv.CustomerID),
					ID:     fmt.Sprintf("customer-%d", inv.CustomerID),
				},
			},
		}
	}

	c := inv.Customer
	party := Party{
		PartyIdentification: PartyIdentification{
			UserID: c.ICO,
			ID:     c.ICO,
		},
		PartyName: PartyName{
			Name: c.Name,
		},
		PostalAddress: PostalAddress{
			StreetName: c.Street,
			CityName:   c.City,
			PostalZone: c.ZIP,
			Country: Country{
				IdentificationCode: countryCodeOrDefault(c.Country),
				Name:               c.Country,
			},
		},
	}

	if c.DIC != "" {
		party.PartyTaxScheme = &PartyTaxScheme{
			CompanyID: c.DIC,
			TaxScheme: TaxScheme{Name: "VAT"},
		}
	}

	if c.Email != "" || c.Phone != "" {
		party.Contact = &PartyContact{
			Telephone:      c.Phone,
			ElectronicMail: c.Email,
		}
	}

	return AccountingParty{Party: party}
}

// countryCodeOrDefault returns the country code or "CZ" as default.
func countryCodeOrDefault(country string) string {
	if country == "" {
		return "CZ"
	}
	return country
}

// buildInvoiceLines creates ISDOC invoice lines from domain items.
func (g *ISDOCGenerator) buildInvoiceLines(inv *domain.Invoice) InvoiceLines {
	lines := make([]InvoiceLine, 0, len(inv.Items))

	for i, item := range inv.Items {
		// Calculate item subtotal (before VAT).
		itemSubtotal := domain.Amount(int64(item.Quantity) * int64(item.UnitPrice) / 100)
		unitPriceVAT := item.UnitPrice.Multiply(float64(100+item.VATRatePercent) / 100.0)

		line := InvoiceLine{
			ID: fmt.Sprintf("%d", i+1),
			InvoicedQuantity: InvoicedQuantity{
				Value:    item.Quantity.String(),
				UnitCode: item.Unit,
			},
			LineExtensionAmount:             itemSubtotal.String(),
			LineExtensionAmountTaxInclusive: item.TotalAmount.String(),
			LineExtensionTaxAmount:          item.VATAmount.String(),
			UnitPrice:                       item.UnitPrice.String(),
			UnitPriceTaxInclusive:           unitPriceVAT.String(),
			ClassifiedTaxCategory: ClassifiedTaxCategory{
				Percent:              fmt.Sprintf("%d", item.VATRatePercent),
				VATCalculationMethod: 0, // from top (default)
			},
			Item: Item{
				Description: item.Description,
			},
		}

		lines = append(lines, line)
	}

	return InvoiceLines{InvoiceLine: lines}
}

// buildTaxTotal creates the ISDOC tax total section.
func (g *ISDOCGenerator) buildTaxTotal(inv *domain.Invoice) TaxTotal {
	// Group items by VAT rate.
	type taxGroup struct {
		taxable domain.Amount
		tax     domain.Amount
		total   domain.Amount
	}
	groups := make(map[int]*taxGroup)

	for _, item := range inv.Items {
		tg, ok := groups[item.VATRatePercent]
		if !ok {
			tg = &taxGroup{}
			groups[item.VATRatePercent] = tg
		}
		itemSubtotal := domain.Amount(int64(item.Quantity) * int64(item.UnitPrice) / 100)
		tg.taxable = tg.taxable.Add(itemSubtotal)
		tg.tax = tg.tax.Add(item.VATAmount)
		tg.total = tg.total.Add(item.TotalAmount)
	}

	subtotals := make([]TaxSubTotal, 0, len(groups))
	for rate, tg := range groups {
		subtotals = append(subtotals, TaxSubTotal{
			TaxableAmount:                    tg.taxable.String(),
			TaxAmount:                        tg.tax.String(),
			TaxInclusiveAmount:               tg.total.String(),
			AlreadyClaimedTaxableAmount:      "0.00",
			AlreadyClaimedTaxAmount:          "0.00",
			AlreadyClaimedTaxInclusiveAmount: "0.00",
			DifferenceTaxableAmount:          tg.taxable.String(),
			DifferenceTaxAmount:              tg.tax.String(),
			DifferenceTaxInclusiveAmount:     tg.total.String(),
			TaxCategory: TaxCategory{
				Percent: fmt.Sprintf("%d", rate),
			},
		})
	}

	return TaxTotal{
		TaxSubTotal: subtotals,
		TaxAmount:   inv.VATAmount.String(),
	}
}

// buildLegalMonetaryTotal creates the ISDOC monetary total section.
func (g *ISDOCGenerator) buildLegalMonetaryTotal(inv *domain.Invoice) LegalMonetaryTotal {
	payable := inv.TotalAmount.Sub(inv.PaidAmount)

	return LegalMonetaryTotal{
		TaxExclusiveAmount:               inv.SubtotalAmount.String(),
		TaxInclusiveAmount:               inv.TotalAmount.String(),
		AlreadyClaimedTaxExclusiveAmount: "0.00",
		AlreadyClaimedTaxInclusiveAmount: "0.00",
		DifferenceTaxExclusiveAmount:     inv.SubtotalAmount.String(),
		DifferenceTaxInclusiveAmount:     inv.TotalAmount.String(),
		PaidDepositsAmount:               inv.PaidAmount.String(),
		PayableAmount:                    payable.String(),
	}
}

// buildPaymentMeans creates the ISDOC payment means section.
func (g *ISDOCGenerator) buildPaymentMeans(inv *domain.Invoice) *PaymentMeans {
	// PaymentMeansCode: 42 = bank transfer, 10 = cash.
	meansCode := 42
	if inv.PaymentMethod == "cash" {
		meansCode = 10
	}

	pm := &PaymentMeans{
		PaymentDueDate: inv.DueDate.Format("2006-01-02"),
		Payment: Payment{
			PaidAmount:       inv.TotalAmount.String(),
			PaymentMeansCode: meansCode,
		},
	}

	// Add bank details if available.
	if inv.BankAccount != "" || inv.IBAN != "" {
		pm.Payment.Details = &PaymentDetails{
			PaymentDueDate: inv.DueDate.Format("2006-01-02"),
			ID:             inv.BankAccount,
			BankCode:       inv.BankCode,
			IBAN:           inv.IBAN,
			BIC:            inv.SWIFT,
			VariableSymbol: inv.VariableSymbol,
			ConstantSymbol: inv.ConstantSymbol,
		}
	}

	return pm
}
