package pdf

import (
	"fmt"

	"github.com/dundee/qrpay"
	"github.com/zajca/zfaktury/internal/domain"
)

// GenerateQRPayment creates a QR code PNG image with Czech SPD (Short Payment Descriptor)
// payment information encoded for the given invoice.
func GenerateQRPayment(invoice *domain.Invoice, iban string, swift string) ([]byte, error) {
	if iban == "" {
		return nil, fmt.Errorf("IBAN is required for QR payment generation")
	}

	p := qrpay.NewSpaydPayment()

	if err := p.SetIBAN(iban); err != nil {
		return nil, fmt.Errorf("invalid IBAN %q: %w", iban, err)
	}

	if swift != "" {
		if err := p.SetBIC(swift); err != nil {
			return nil, fmt.Errorf("invalid SWIFT/BIC %q: %w", swift, err)
		}
	}

	// Amount in CZK (whole units with decimals).
	amount := fmt.Sprintf("%.2f", invoice.TotalAmount.ToCZK())
	if err := p.SetAmount(amount); err != nil {
		return nil, fmt.Errorf("setting amount: %w", err)
	}

	if err := p.SetCurrency("CZK"); err != nil {
		return nil, fmt.Errorf("setting currency: %w", err)
	}

	// Variable symbol as extended attribute.
	if invoice.VariableSymbol != "" {
		p.SetExtendedAttribute("VS", invoice.VariableSymbol)
	}

	// Constant symbol as extended attribute.
	if invoice.ConstantSymbol != "" {
		p.SetExtendedAttribute("KS", invoice.ConstantSymbol)
	}

	// Due date.
	if !invoice.DueDate.IsZero() {
		p.SetDate(invoice.DueDate)
	}

	// Message: invoice number.
	if invoice.InvoiceNumber != "" {
		if err := p.SetMessage(invoice.InvoiceNumber); err != nil {
			return nil, fmt.Errorf("setting message: %w", err)
		}
	}

	return qrpay.GetQRCodeImage(p)
}

// GenerateSPDString creates the SPD string for a given invoice without generating a QR image.
// Useful for testing the SPD format.
func GenerateSPDString(invoice *domain.Invoice, iban string, swift string) (string, error) {
	if iban == "" {
		return "", fmt.Errorf("IBAN is required for SPD generation")
	}

	p := qrpay.NewSpaydPayment()

	if err := p.SetIBAN(iban); err != nil {
		return "", fmt.Errorf("invalid IBAN %q: %w", iban, err)
	}

	if swift != "" {
		if err := p.SetBIC(swift); err != nil {
			return "", fmt.Errorf("invalid SWIFT/BIC %q: %w", swift, err)
		}
	}

	amount := fmt.Sprintf("%.2f", invoice.TotalAmount.ToCZK())
	if err := p.SetAmount(amount); err != nil {
		return "", fmt.Errorf("setting amount: %w", err)
	}

	if err := p.SetCurrency("CZK"); err != nil {
		return "", fmt.Errorf("setting currency: %w", err)
	}

	if invoice.VariableSymbol != "" {
		p.SetExtendedAttribute("VS", invoice.VariableSymbol)
	}

	if invoice.ConstantSymbol != "" {
		p.SetExtendedAttribute("KS", invoice.ConstantSymbol)
	}

	if !invoice.DueDate.IsZero() {
		p.SetDate(invoice.DueDate)
	}

	if invoice.InvoiceNumber != "" {
		if err := p.SetMessage(invoice.InvoiceNumber); err != nil {
			return "", fmt.Errorf("setting message: %w", err)
		}
	}

	return p.GenerateString()
}
