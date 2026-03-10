package pdf

import (
	"context"
	"fmt"

	maroto "github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/consts/extension"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"github.com/zajca/zfaktury/internal/domain"
)

// InvoicePDFGenerator generates PDF documents for invoices.
type InvoicePDFGenerator struct{}

// NewInvoicePDFGenerator creates a new InvoicePDFGenerator.
func NewInvoicePDFGenerator() *InvoicePDFGenerator {
	return &InvoicePDFGenerator{}
}

// Generate creates a PDF document for the given invoice and supplier info.
func (g *InvoicePDFGenerator) Generate(_ context.Context, invoice *domain.Invoice, supplier SupplierInfo) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageSize(pagesize.A4).
		WithLeftMargin(15).
		WithRightMargin(15).
		WithTopMargin(15).
		WithBottomMargin(15).
		WithDefaultFont(&props.Font{
			Size:   9,
			Family: "arial",
			Style:  fontstyle.Normal,
		}).
		Build()

	m := maroto.New(cfg)

	// Header: Invoice number, type, dates.
	g.addHeader(m, invoice)

	// Separator.
	m.AddRows(line.NewRow(2))

	// Two-column: supplier (left) | customer (right).
	g.addParties(m, invoice, supplier)

	// Separator.
	m.AddRows(line.NewRow(2))

	// Line items table.
	g.addItemsTable(m, invoice)

	// VAT summary.
	g.addVATSummary(m, invoice)

	// Totals.
	g.addTotals(m, invoice)

	// Separator.
	m.AddRows(line.NewRow(2))

	// Payment info + QR code.
	g.addPaymentSection(m, invoice, supplier)

	// Footer: VAT note.
	if !supplier.VATRegistered {
		m.AddRows(
			row.New(8).Add(
				col.New(12).Add(
					text.New("Subjekt neni platce DPH.", props.Text{
						Size:  8,
						Style: fontstyle.Italic,
						Align: align.Center,
					}),
				),
			),
		)
	}

	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("generating PDF: %w", err)
	}

	return doc.GetBytes(), nil
}

func (g *InvoicePDFGenerator) addHeader(m core.Maroto, invoice *domain.Invoice) {
	typeLabel := invoiceTypeLabel(invoice.Type)

	m.AddRows(
		row.New(12).Add(
			col.New(8).Add(
				text.New(fmt.Sprintf("%s %s", typeLabel, invoice.InvoiceNumber), props.Text{
					Size:  16,
					Style: fontstyle.Bold,
				}),
			),
			col.New(4).Add(
				text.New(statusLabel(invoice.Status), props.Text{
					Size:  10,
					Align: align.Right,
					Style: fontstyle.Bold,
				}),
			),
		),
	)

	m.AddRows(
		row.New(6).Add(
			col.New(4).Add(
				text.New(fmt.Sprintf("Datum vystaveni: %s", invoice.IssueDate.Format("02.01.2006")), props.Text{Size: 8}),
			),
			col.New(4).Add(
				text.New(fmt.Sprintf("Datum splatnosti: %s", invoice.DueDate.Format("02.01.2006")), props.Text{Size: 8}),
			),
			col.New(4).Add(
				text.New(fmt.Sprintf("DUZP: %s", invoice.DeliveryDate.Format("02.01.2006")), props.Text{Size: 8, Align: align.Right}),
			),
		),
	)
}

func (g *InvoicePDFGenerator) addParties(m core.Maroto, invoice *domain.Invoice, supplier SupplierInfo) {
	// Supplier column content.
	supplierRows := []string{supplier.Name}
	if supplier.Street != "" {
		supplierRows = append(supplierRows, supplier.Street)
	}
	if supplier.City != "" || supplier.ZIP != "" {
		supplierRows = append(supplierRows, fmt.Sprintf("%s %s", supplier.ZIP, supplier.City))
	}
	if supplier.ICO != "" {
		supplierRows = append(supplierRows, fmt.Sprintf("ICO: %s", supplier.ICO))
	}
	if supplier.DIC != "" {
		supplierRows = append(supplierRows, fmt.Sprintf("DIC: %s", supplier.DIC))
	}

	// Customer column content.
	var customerRows []string
	if invoice.Customer != nil {
		c := invoice.Customer
		customerRows = append(customerRows, c.Name)
		if c.Street != "" {
			customerRows = append(customerRows, c.Street)
		}
		if c.City != "" || c.ZIP != "" {
			customerRows = append(customerRows, fmt.Sprintf("%s %s", c.ZIP, c.City))
		}
		if c.ICO != "" {
			customerRows = append(customerRows, fmt.Sprintf("ICO: %s", c.ICO))
		}
		if c.DIC != "" {
			customerRows = append(customerRows, fmt.Sprintf("DIC: %s", c.DIC))
		}
	}

	// Headers.
	m.AddRows(
		row.New(8).Add(
			col.New(6).Add(
				text.New("Dodavatel", props.Text{
					Size:  10,
					Style: fontstyle.Bold,
				}),
			),
			col.New(6).Add(
				text.New("Odberatel", props.Text{
					Size:  10,
					Style: fontstyle.Bold,
				}),
			),
		),
	)

	// Determine max rows needed.
	maxRows := len(supplierRows)
	if len(customerRows) > maxRows {
		maxRows = len(customerRows)
	}

	for i := 0; i < maxRows; i++ {
		sText := ""
		if i < len(supplierRows) {
			sText = supplierRows[i]
		}
		cText := ""
		if i < len(customerRows) {
			cText = customerRows[i]
		}

		sStyle := props.Text{Size: 8}
		cStyle := props.Text{Size: 8}
		// Bold the first line (company names).
		if i == 0 {
			sStyle.Style = fontstyle.Bold
			cStyle.Style = fontstyle.Bold
		}

		m.AddRows(
			row.New(5).Add(
				col.New(6).Add(text.New(sText, sStyle)),
				col.New(6).Add(text.New(cText, cStyle)),
			),
		)
	}
}

func (g *InvoicePDFGenerator) addItemsTable(m core.Maroto, invoice *domain.Invoice) {
	// Table header.
	headerColor := &props.Color{Red: 240, Green: 240, Blue: 240}
	headerStyle := props.Text{Size: 7, Style: fontstyle.Bold}
	headerStyleRight := props.Text{Size: 7, Style: fontstyle.Bold, Align: align.Right}
	cellStyle := &props.Cell{
		BackgroundColor: headerColor,
		BorderType:      border.Bottom,
		BorderThickness: 0.3,
	}

	m.AddRows(
		row.New(7).Add(
			col.New(1).Add(text.New("#", headerStyle)).WithStyle(cellStyle),
			col.New(3).Add(text.New("Popis", headerStyle)).WithStyle(cellStyle),
			col.New(1).Add(text.New("Mn.", headerStyleRight)).WithStyle(cellStyle),
			col.New(1).Add(text.New("Jedn.", headerStyle)).WithStyle(cellStyle),
			col.New(2).Add(text.New("Cena/ks", headerStyleRight)).WithStyle(cellStyle),
			col.New(1).Add(text.New("DPH %", headerStyleRight)).WithStyle(cellStyle),
			col.New(1).Add(text.New("DPH", headerStyleRight)).WithStyle(cellStyle),
			col.New(2).Add(text.New("Celkem", headerStyleRight)).WithStyle(cellStyle),
		),
	)

	// Table rows.
	rowStyle := props.Text{Size: 8}
	rowStyleRight := props.Text{Size: 8, Align: align.Right}
	rowCellStyle := &props.Cell{
		BorderType:      border.Bottom,
		BorderThickness: 0.1,
	}

	for i, item := range invoice.Items {
		// Calculate subtotal (before VAT) for this item.
		itemSubtotal := domain.Amount(int64(item.Quantity) * int64(item.UnitPrice) / 100)

		m.AddRows(
			row.New(6).Add(
				col.New(1).Add(text.New(fmt.Sprintf("%d", i+1), rowStyle)).WithStyle(rowCellStyle),
				col.New(3).Add(text.New(item.Description, rowStyle)).WithStyle(rowCellStyle),
				col.New(1).Add(text.New(formatQuantity(item.Quantity), rowStyleRight)).WithStyle(rowCellStyle),
				col.New(1).Add(text.New(item.Unit, rowStyle)).WithStyle(rowCellStyle),
				col.New(2).Add(text.New(formatAmount(item.UnitPrice), rowStyleRight)).WithStyle(rowCellStyle),
				col.New(1).Add(text.New(fmt.Sprintf("%d%%", item.VATRatePercent), rowStyleRight)).WithStyle(rowCellStyle),
				col.New(1).Add(text.New(formatAmount(item.VATAmount), rowStyleRight)).WithStyle(rowCellStyle),
				col.New(2).Add(text.New(formatAmount(itemSubtotal.Add(item.VATAmount)), rowStyleRight)).WithStyle(rowCellStyle),
			),
		)
	}
}

func (g *InvoicePDFGenerator) addVATSummary(m core.Maroto, invoice *domain.Invoice) {
	// Group items by VAT rate.
	vatGroups := make(map[int]struct {
		base domain.Amount
		vat  domain.Amount
	})

	for _, item := range invoice.Items {
		group := vatGroups[item.VATRatePercent]
		itemSubtotal := domain.Amount(int64(item.Quantity) * int64(item.UnitPrice) / 100)
		group.base = group.base.Add(itemSubtotal)
		group.vat = group.vat.Add(item.VATAmount)
		vatGroups[item.VATRatePercent] = group
	}

	if len(vatGroups) > 0 {
		m.AddRows(row.New(4)) // Spacer.

		m.AddRows(
			row.New(6).Add(
				col.New(6), // Empty left side.
				col.New(2).Add(text.New("Sazba DPH", props.Text{Size: 7, Style: fontstyle.Bold})),
				col.New(2).Add(text.New("Zaklad", props.Text{Size: 7, Style: fontstyle.Bold, Align: align.Right})),
				col.New(2).Add(text.New("DPH", props.Text{Size: 7, Style: fontstyle.Bold, Align: align.Right})),
			),
		)

		for rate, group := range vatGroups {
			m.AddRows(
				row.New(5).Add(
					col.New(6),
					col.New(2).Add(text.New(fmt.Sprintf("%d%%", rate), props.Text{Size: 8})),
					col.New(2).Add(text.New(formatAmount(group.base), props.Text{Size: 8, Align: align.Right})),
					col.New(2).Add(text.New(formatAmount(group.vat), props.Text{Size: 8, Align: align.Right})),
				),
			)
		}
	}
}

func (g *InvoicePDFGenerator) addTotals(m core.Maroto, invoice *domain.Invoice) {
	m.AddRows(row.New(4)) // Spacer.

	rightStyle := props.Text{Size: 9, Align: align.Right}
	rightBoldStyle := props.Text{Size: 11, Style: fontstyle.Bold, Align: align.Right}

	m.AddRows(
		row.New(6).Add(
			col.New(6),
			col.New(3).Add(text.New("Zaklad celkem:", rightStyle)),
			col.New(3).Add(text.New(formatAmountCZK(invoice.SubtotalAmount), rightStyle)),
		),
		row.New(6).Add(
			col.New(6),
			col.New(3).Add(text.New("DPH celkem:", rightStyle)),
			col.New(3).Add(text.New(formatAmountCZK(invoice.VATAmount), rightStyle)),
		),
		row.New(8).Add(
			col.New(6),
			col.New(3).Add(text.New("Celkem k uhrade:", rightBoldStyle)),
			col.New(3).Add(text.New(formatAmountCZK(invoice.TotalAmount), rightBoldStyle)),
		),
	)
}

func (g *InvoicePDFGenerator) addPaymentSection(m core.Maroto, invoice *domain.Invoice, supplier SupplierInfo) {
	m.AddRows(
		row.New(8).Add(
			col.New(12).Add(
				text.New("Platebni udaje", props.Text{
					Size:  10,
					Style: fontstyle.Bold,
				}),
			),
		),
	)

	// Determine IBAN and SWIFT to use (invoice overrides supplier).
	iban := invoice.IBAN
	if iban == "" {
		iban = supplier.IBAN
	}
	swift := invoice.SWIFT
	if swift == "" {
		swift = supplier.SWIFT
	}
	bankAccount := invoice.BankAccount
	if bankAccount == "" {
		bankAccount = supplier.BankAccount
	}
	bankCode := invoice.BankCode
	if bankCode == "" {
		bankCode = supplier.BankCode
	}

	// Try to generate QR code.
	var qrBytes []byte
	if iban != "" {
		qr, err := GenerateQRPayment(invoice, iban, swift)
		if err == nil {
			qrBytes = qr
		}
	}

	// Payment details on the left, QR on the right.
	paymentInfoSize := 8
	if qrBytes != nil {
		paymentInfoSize = 8
	}

	labelStyle := props.Text{Size: 8, Style: fontstyle.Bold}
	valueStyle := props.Text{Size: 8}

	if bankAccount != "" {
		accountStr := bankAccount
		if bankCode != "" {
			accountStr += "/" + bankCode
		}
		m.AddRows(
			row.New(5).Add(
				col.New(paymentInfoSize).Add(
					text.New(fmt.Sprintf("Cislo uctu: %s", accountStr), valueStyle),
				),
			),
		)
	}

	if iban != "" {
		m.AddRows(
			row.New(5).Add(
				col.New(paymentInfoSize).Add(
					text.New(fmt.Sprintf("IBAN: %s", iban), valueStyle),
				),
			),
		)
	}

	if invoice.VariableSymbol != "" {
		m.AddRows(
			row.New(5).Add(
				col.New(paymentInfoSize).Add(
					text.New(fmt.Sprintf("Variabilni symbol: %s", invoice.VariableSymbol), valueStyle),
				),
			),
		)
	}

	if invoice.ConstantSymbol != "" {
		m.AddRows(
			row.New(5).Add(
				col.New(paymentInfoSize).Add(
					text.New(fmt.Sprintf("Konstantni symbol: %s", invoice.ConstantSymbol), valueStyle),
				),
			),
		)
	}

	m.AddRows(
		row.New(5).Add(
			col.New(paymentInfoSize).Add(
				text.New(fmt.Sprintf("Datum splatnosti: %s", invoice.DueDate.Format("02.01.2006")), labelStyle),
			),
		),
	)

	// QR code image.
	if qrBytes != nil {
		m.AddRows(
			row.New(8).Add(
				col.New(12).Add(
					text.New("QR platba:", props.Text{Size: 8, Style: fontstyle.Bold}),
				),
			),
		)
		m.AddRows(
			row.New(40).Add(
				col.New(4).Add(
					image.NewFromBytes(qrBytes, extension.Png, props.Rect{
						Percent: 100,
						Center:  true,
					}),
				),
				col.New(8),
			),
		)
	}
}

// invoiceTypeLabel returns a Czech label for the invoice type.
func invoiceTypeLabel(t string) string {
	switch t {
	case domain.InvoiceTypeRegular:
		return "Faktura"
	case domain.InvoiceTypeProforma:
		return "Proforma faktura"
	case domain.InvoiceTypeCreditNote:
		return "Dobropis"
	default:
		return "Faktura"
	}
}

// statusLabel returns a Czech label for the invoice status.
func statusLabel(s string) string {
	switch s {
	case domain.InvoiceStatusDraft:
		return "Koncept"
	case domain.InvoiceStatusSent:
		return "Odeslana"
	case domain.InvoiceStatusPaid:
		return "Uhrazena"
	case domain.InvoiceStatusOverdue:
		return "Po splatnosti"
	case domain.InvoiceStatusCancelled:
		return "Stornovana"
	default:
		return s
	}
}

// formatAmount formats a domain.Amount for display in a PDF (e.g. "1 234,56").
func formatAmount(a domain.Amount) string {
	return a.String()
}

// formatAmountCZK formats a domain.Amount with CZK suffix.
func formatAmountCZK(a domain.Amount) string {
	return a.String() + " CZK"
}

// formatQuantity formats a quantity Amount (stored as cents) as a decimal string.
func formatQuantity(a domain.Amount) string {
	return a.String()
}
