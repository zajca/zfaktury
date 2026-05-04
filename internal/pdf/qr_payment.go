package pdf

import (
	"fmt"
	"strings"
	"time"

	"github.com/dundee/qrpay"
	"github.com/skip2/go-qrcode"
	"github.com/zajca/zfaktury/internal/domain"
)

// digitsOnly strips non-digit characters and trims to maxLen runes.
// Used for variable/constant symbols which must be numeric per Czech SPAYD spec
// (banks reject payments with non-numeric VS/KS).
func digitsOnly(s string, maxLen int) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if maxLen > 0 && len(out) > maxLen {
		out = out[len(out)-maxLen:]
	}
	return out
}

// formatSpaydDate formats a date for SPAYD DT field as YYYYMMDD.
// Workaround for github.com/dundee/qrpay v0.0.4 bug where SetDate produces
// "YYYY[M][D]" without zero-padding, breaking single-digit months/days.
func formatSpaydDate(t time.Time) string {
	y, m, d := t.Date()
	return fmt.Sprintf("%04d%02d%02d", y, int(m), d)
}

// buildSpayd assembles a SPAYD string for the given invoice.
// Avoids qrpay.SpaydPayment.SetDate (buggy formatting) and ensures VS/KS are
// numeric only.
func buildSpayd(invoice *domain.Invoice, iban string, swift string) (string, error) {
	if iban == "" {
		return "", fmt.Errorf("IBAN is required for QR payment generation")
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

	if vs := digitsOnly(invoice.VariableSymbol, 10); vs != "" {
		p.SetExtendedAttribute("VS", vs)
	}

	if ks := digitsOnly(invoice.ConstantSymbol, 10); ks != "" {
		p.SetExtendedAttribute("KS", ks)
	}

	if invoice.InvoiceNumber != "" {
		if err := p.SetMessage(invoice.InvoiceNumber); err != nil {
			return "", fmt.Errorf("setting message: %w", err)
		}
	}

	spd, err := p.GenerateString()
	if err != nil {
		return "", err
	}

	// Insert correctly-formatted DT (qrpay's SetDate would produce DT:YYYYMD
	// without zero-padding for single-digit month/day).
	if !invoice.DueDate.IsZero() {
		dtField := "DT:" + formatSpaydDate(invoice.DueDate) + "*"
		// SPAYD spec places DT before MSG; insert there if MSG exists,
		// otherwise append before the first X- attribute or at the end.
		switch {
		case strings.Contains(spd, "*MSG:"):
			spd = strings.Replace(spd, "*MSG:", "*"+dtField+"MSG:", 1)
		case strings.Contains(spd, "*X-"):
			spd = strings.Replace(spd, "*X-", "*"+dtField+"X-", 1)
		default:
			spd += dtField
		}
	}

	return spd, nil
}

// GenerateQRPayment creates a QR code PNG image with Czech SPD (Short Payment Descriptor)
// payment information encoded for the given invoice.
func GenerateQRPayment(invoice *domain.Invoice, iban string, swift string) ([]byte, error) {
	spd, err := buildSpayd(invoice, iban, swift)
	if err != nil {
		return nil, err
	}
	return qrcode.Encode(spd, qrcode.Medium, 400)
}

// GenerateSPDString creates the SPD string for a given invoice without generating a QR image.
// Useful for testing the SPD format.
func GenerateSPDString(invoice *domain.Invoice, iban string, swift string) (string, error) {
	return buildSpayd(invoice, iban, swift)
}
