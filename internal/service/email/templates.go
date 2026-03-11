package email

import (
	"bytes"
	"fmt"
	"html/template"
	texttemplate "text/template"
)

// InvoiceEmailItem represents a single line item in the invoice email.
type InvoiceEmailItem struct {
	Description string
	Quantity    string
	UnitPrice   string
	Total       string
}

// InvoiceEmailData holds all data needed to render an invoice email.
type InvoiceEmailData struct {
	InvoiceNumber  string
	CustomerName   string
	IssueDate      string
	DueDate        string
	TotalFormatted string
	CurrencyCode   string
	PaymentMethod  string
	BankAccount    string
	VariableSymbol string
	SenderName     string
	Items          []InvoiceEmailItem
}

// RenderInvoiceEmail renders the invoice email templates and returns the
// subject, HTML body and plain-text body.
func RenderInvoiceEmail(data InvoiceEmailData) (subject, htmlBody, textBody string, err error) {
	subject = fmt.Sprintf("Faktura %s - %s", data.InvoiceNumber, data.SenderName)

	htmlBody, err = renderHTML(data)
	if err != nil {
		return "", "", "", fmt.Errorf("rendering HTML email template: %w", err)
	}

	textBody, err = renderText(data)
	if err != nil {
		return "", "", "", fmt.Errorf("rendering text email template: %w", err)
	}

	return subject, htmlBody, textBody, nil
}

// renderHTML renders the HTML email template.
func renderHTML(data InvoiceEmailData) (string, error) {
	tmpl, err := template.New("invoice_html").Parse(invoiceHTMLTemplate)
	if err != nil {
		return "", fmt.Errorf("parsing HTML template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing HTML template: %w", err)
	}
	return buf.String(), nil
}

// renderText renders the plain-text email template.
func renderText(data InvoiceEmailData) (string, error) {
	tmpl, err := texttemplate.New("invoice_text").Parse(invoiceTextTemplate)
	if err != nil {
		return "", fmt.Errorf("parsing text template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing text template: %w", err)
	}
	return buf.String(), nil
}

// invoiceHTMLTemplate is a clean, professional HTML invoice email with inline
// CSS and no external resources.
const invoiceHTMLTemplate = `<!DOCTYPE html>
<html lang="cs">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Faktura {{.InvoiceNumber}}</title>
</head>
<body style="margin:0;padding:0;background-color:#f4f4f4;font-family:Arial,Helvetica,sans-serif;">
<table width="100%" cellpadding="0" cellspacing="0" style="background-color:#f4f4f4;padding:20px 0;">
  <tr>
    <td align="center">
      <table width="600" cellpadding="0" cellspacing="0" style="background-color:#ffffff;border-radius:6px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.08);">

        <!-- Header -->
        <tr>
          <td style="background-color:#1a56db;padding:28px 32px;">
            <h1 style="margin:0;color:#ffffff;font-size:22px;font-weight:700;letter-spacing:0.5px;">Faktura {{.InvoiceNumber}}</h1>
            <p style="margin:4px 0 0;color:#bfdbfe;font-size:14px;">{{.SenderName}}</p>
          </td>
        </tr>

        <!-- Greeting -->
        <tr>
          <td style="padding:28px 32px 0;">
            <p style="margin:0;font-size:15px;color:#374151;">Dobrý den,</p>
            <p style="margin:12px 0 0;font-size:15px;color:#374151;">
              v příloze naleznete fakturu č. <strong>{{.InvoiceNumber}}</strong> vystavenou pro <strong>{{.CustomerName}}</strong>.
            </p>
          </td>
        </tr>

        <!-- Invoice details -->
        <tr>
          <td style="padding:24px 32px 0;">
            <table width="100%" cellpadding="0" cellspacing="0" style="background-color:#f9fafb;border-radius:4px;border:1px solid #e5e7eb;">
              <tr>
                <td style="padding:16px 20px;border-bottom:1px solid #e5e7eb;">
                  <table width="100%" cellpadding="0" cellspacing="0">
                    <tr>
                      <td style="font-size:13px;color:#6b7280;width:50%;">Datum vystavení</td>
                      <td style="font-size:13px;color:#6b7280;width:50%;">Datum splatnosti</td>
                    </tr>
                    <tr>
                      <td style="font-size:15px;color:#111827;font-weight:600;padding-top:2px;">{{.IssueDate}}</td>
                      <td style="font-size:15px;color:#111827;font-weight:600;padding-top:2px;">{{.DueDate}}</td>
                    </tr>
                  </table>
                </td>
              </tr>
              <tr>
                <td style="padding:16px 20px;">
                  <table width="100%" cellpadding="0" cellspacing="0">
                    <tr>
                      <td style="font-size:13px;color:#6b7280;">Celková částka</td>
                    </tr>
                    <tr>
                      <td style="font-size:22px;color:#1a56db;font-weight:700;padding-top:2px;">{{.TotalFormatted}} {{.CurrencyCode}}</td>
                    </tr>
                  </table>
                </td>
              </tr>
            </table>
          </td>
        </tr>

        <!-- Items table -->
        {{if .Items}}
        <tr>
          <td style="padding:24px 32px 0;">
            <table width="100%" cellpadding="0" cellspacing="0" style="border-collapse:collapse;">
              <thead>
                <tr style="background-color:#f3f4f6;">
                  <th style="text-align:left;padding:10px 12px;font-size:12px;color:#6b7280;font-weight:600;text-transform:uppercase;letter-spacing:0.5px;border-bottom:1px solid #e5e7eb;">Popis</th>
                  <th style="text-align:right;padding:10px 12px;font-size:12px;color:#6b7280;font-weight:600;text-transform:uppercase;letter-spacing:0.5px;border-bottom:1px solid #e5e7eb;">Množství</th>
                  <th style="text-align:right;padding:10px 12px;font-size:12px;color:#6b7280;font-weight:600;text-transform:uppercase;letter-spacing:0.5px;border-bottom:1px solid #e5e7eb;">Jedn. cena</th>
                  <th style="text-align:right;padding:10px 12px;font-size:12px;color:#6b7280;font-weight:600;text-transform:uppercase;letter-spacing:0.5px;border-bottom:1px solid #e5e7eb;">Celkem</th>
                </tr>
              </thead>
              <tbody>
                {{range .Items}}
                <tr>
                  <td style="padding:10px 12px;font-size:14px;color:#374151;border-bottom:1px solid #f3f4f6;">{{.Description}}</td>
                  <td style="padding:10px 12px;font-size:14px;color:#374151;text-align:right;border-bottom:1px solid #f3f4f6;">{{.Quantity}}</td>
                  <td style="padding:10px 12px;font-size:14px;color:#374151;text-align:right;border-bottom:1px solid #f3f4f6;">{{.UnitPrice}}</td>
                  <td style="padding:10px 12px;font-size:14px;color:#374151;text-align:right;border-bottom:1px solid #f3f4f6;font-weight:600;">{{.Total}}</td>
                </tr>
                {{end}}
              </tbody>
            </table>
          </td>
        </tr>
        {{end}}

        <!-- Payment info -->
        {{if .BankAccount}}
        <tr>
          <td style="padding:24px 32px 0;">
            <table width="100%" cellpadding="0" cellspacing="0" style="background-color:#eff6ff;border-left:4px solid #1a56db;border-radius:0 4px 4px 0;padding:16px 20px;">
              <tr><td style="font-size:13px;color:#1e40af;font-weight:700;text-transform:uppercase;letter-spacing:0.5px;padding-bottom:10px;">Platební údaje</td></tr>
              {{if .PaymentMethod}}<tr><td style="font-size:14px;color:#374151;padding-bottom:4px;"><strong>Způsob platby:</strong> {{.PaymentMethod}}</td></tr>{{end}}
              <tr><td style="font-size:14px;color:#374151;padding-bottom:4px;"><strong>Číslo účtu:</strong> {{.BankAccount}}</td></tr>
              {{if .VariableSymbol}}<tr><td style="font-size:14px;color:#374151;"><strong>Variabilní symbol:</strong> {{.VariableSymbol}}</td></tr>{{end}}
            </table>
          </td>
        </tr>
        {{end}}

        <!-- Footer -->
        <tr>
          <td style="padding:28px 32px;">
            <p style="margin:0;font-size:15px;color:#374151;">V případě dotazů nás neváhejte kontaktovat.</p>
            <p style="margin:16px 0 0;font-size:15px;color:#374151;">S pozdravem,<br><strong>{{.SenderName}}</strong></p>
          </td>
        </tr>
        <tr>
          <td style="background-color:#f9fafb;padding:16px 32px;border-top:1px solid #e5e7eb;">
            <p style="margin:0;font-size:12px;color:#9ca3af;text-align:center;">Tato zpráva byla vygenerována automaticky systémem ZFaktury.</p>
          </td>
        </tr>

      </table>
    </td>
  </tr>
</table>
</body>
</html>`

// invoiceTextTemplate is a plain-text fallback for the invoice email.
const invoiceTextTemplate = `Dobrý den,

v příloze naleznete fakturu č. {{.InvoiceNumber}} vystavenou pro {{.CustomerName}}.

PŘEHLED FAKTURY
---------------
Datum vystavení:  {{.IssueDate}}
Datum splatnosti: {{.DueDate}}
Celková částka:   {{.TotalFormatted}} {{.CurrencyCode}}
{{if .Items}}
POLOŽKY
-------
{{range .Items}}  {{.Description}}
    Množství: {{.Quantity}}  Jedn. cena: {{.UnitPrice}}  Celkem: {{.Total}}
{{end}}{{end}}{{if .BankAccount}}
PLATEBNÍ ÚDAJE
--------------
{{if .PaymentMethod}}Způsob platby:    {{.PaymentMethod}}
{{end}}Číslo účtu:       {{.BankAccount}}
{{if .VariableSymbol}}Variabilní symbol: {{.VariableSymbol}}
{{end}}{{end}}
V případě dotazů nás neváhejte kontaktovat.

S pozdravem,
{{.SenderName}}

---
Tato zpráva byla vygenerována automaticky systémem ZFaktury.
`
