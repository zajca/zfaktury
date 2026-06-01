package service

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"net/mail"
	"strings"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/isdoc"
	"github.com/zajca/zfaktury/internal/pdf"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service/email"
)

// InvoiceEmailService builds invoice attachments (PDF/ISDOC) and sends an
// invoice by email, marking draft invoices as sent afterwards. It is the single
// implementation shared by the HTTP send-email handler and the recurring
// auto-send path, so the two never drift.
type InvoiceEmailService struct {
	invoices *InvoiceService
	settings *SettingsService
	company  repository.CompanyRepo
	pdfGen   *pdf.InvoicePDFGenerator
	isdocGen *isdoc.ISDOCGenerator
	sender   *email.EmailSender
}

// NewInvoiceEmailService creates a new InvoiceEmailService.
func NewInvoiceEmailService(
	invoices *InvoiceService,
	settings *SettingsService,
	company repository.CompanyRepo,
	pdfGen *pdf.InvoicePDFGenerator,
	isdocGen *isdoc.ISDOCGenerator,
	sender *email.EmailSender,
) *InvoiceEmailService {
	return &InvoiceEmailService{
		invoices: invoices,
		settings: settings,
		company:  company,
		pdfGen:   pdfGen,
		isdocGen: isdocGen,
		sender:   sender,
	}
}

// EmailOptions describes one outgoing invoice email.
type EmailOptions struct {
	To          string
	Subject     string
	Body        string
	AttachPDF   bool
	AttachISDOC bool
}

// EmailDefaults holds the company-configured email defaults for an invoice,
// with the {invoice_number} placeholder already substituted.
type EmailDefaults struct {
	AttachPDF   bool
	AttachISDOC bool
	Subject     string
	Body        string
}

// IsConfigured reports whether an SMTP sender is available.
func (s *InvoiceEmailService) IsConfigured() bool {
	return s.sender != nil && s.sender.IsConfigured()
}

// Defaults resolves the per-company email defaults (attachment flags + subject
// and body templates) for the given invoice number. Templates fall back to the
// Czech defaults used by the manual send dialog.
func (s *InvoiceEmailService) Defaults(ctx context.Context, companyID int64, invoiceNumber string) (EmailDefaults, error) {
	settings, err := s.settings.GetAll(ctx, companyID)
	if err != nil {
		return EmailDefaults{}, fmt.Errorf("loading email defaults: %w", err)
	}

	attachPDF := true
	if v, ok := settings[SettingEmailAttachPDF]; ok {
		attachPDF = v == "true"
	}
	attachISDOC := false
	if v, ok := settings[SettingEmailAttachISDOC]; ok {
		attachISDOC = v == "true"
	}

	subjectTpl := settings[SettingEmailSubjectTpl]
	if subjectTpl == "" {
		subjectTpl = "Faktura {invoice_number}"
	}
	bodyTpl := settings[SettingEmailBodyTpl]
	if bodyTpl == "" {
		bodyTpl = "Dobrý den,\n\nv příloze zasíláme fakturu {invoice_number}.\n\nS pozdravem"
	}

	return EmailDefaults{
		AttachPDF:   attachPDF,
		AttachISDOC: attachISDOC,
		Subject:     strings.ReplaceAll(subjectTpl, "{invoice_number}", invoiceNumber),
		Body:        strings.ReplaceAll(bodyTpl, "{invoice_number}", invoiceNumber),
	}, nil
}

// Send generates the requested attachments and emails the invoice on behalf of
// the company, then marks a draft invoice as sent. The caller supplies the
// already-loaded company and invoice (the HTTP handler has both from the request
// context); SendByID is the convenience entry point that loads them by ID.
func (s *InvoiceEmailService) Send(ctx context.Context, company *domain.Company, invoice *domain.Invoice, opts EmailOptions) error {
	if !s.IsConfigured() {
		return fmt.Errorf("SMTP is not configured: %w", domain.ErrInvalidInput)
	}
	if company == nil || invoice == nil {
		return fmt.Errorf("company and invoice are required: %w", domain.ErrInvalidInput)
	}
	if opts.To == "" {
		return fmt.Errorf("recipient email is required: %w", domain.ErrInvalidInput)
	}
	if _, err := mail.ParseAddress(opts.To); err != nil {
		return fmt.Errorf("invalid recipient email %q: %w", opts.To, domain.ErrInvalidInput)
	}
	if !opts.AttachPDF && !opts.AttachISDOC {
		return fmt.Errorf("at least one attachment type must be selected: %w", domain.ErrInvalidInput)
	}

	var attachments []email.Attachment

	if opts.AttachPDF {
		pdfSvcSettings, err := s.settings.GetPDFSettings(ctx, company.ID)
		if err != nil {
			return fmt.Errorf("loading PDF settings: %w", err)
		}
		pdfSettings := pdf.PDFSettings{
			LogoPath:        pdfSvcSettings.LogoPath,
			AccentColor:     pdfSvcSettings.AccentColor,
			FooterText:      pdfSvcSettings.FooterText,
			ShowQR:          pdfSvcSettings.ShowQR,
			ShowBankDetails: pdfSvcSettings.ShowBankDetails,
			FontSize:        pdfSvcSettings.FontSize,
		}
		pdfBytes, err := s.pdfGen.Generate(ctx, invoice, pdf.SupplierFromCompany(company), pdfSettings)
		if err != nil {
			return fmt.Errorf("generating invoice PDF: %w", err)
		}
		attachments = append(attachments, email.Attachment{
			Filename:    fmt.Sprintf("faktura_%s.pdf", invoice.InvoiceNumber),
			ContentType: "application/pdf",
			Data:        pdfBytes,
		})
	}

	if opts.AttachISDOC {
		isdocBytes, err := s.isdocGen.Generate(ctx, invoice, isdoc.SupplierFromCompany(company))
		if err != nil {
			return fmt.Errorf("generating invoice ISDOC: %w", err)
		}
		attachments = append(attachments, email.Attachment{
			Filename:    fmt.Sprintf("%s.isdoc", invoice.InvoiceNumber),
			ContentType: "application/xml",
			Data:        isdocBytes,
		})
	}

	msg := email.EmailMessage{
		To:          []string{opts.To},
		Subject:     opts.Subject,
		BodyText:    opts.Body,
		BodyHTML:    "<p>" + strings.ReplaceAll(html.EscapeString(opts.Body), "\n", "<br>") + "</p>",
		Attachments: attachments,
	}

	if err := s.sender.Send(ctx, msg); err != nil {
		return fmt.Errorf("sending invoice email: %w", err)
	}

	// Advance draft invoices to "sent" once the email actually goes out.
	// Non-draft invoices keep their status -- users may legitimately re-email a
	// sent or paid invoice.
	if invoice.Status == domain.InvoiceStatusDraft {
		if err := s.invoices.MarkAsSent(ctx, company.ID, invoice.ID); err != nil {
			slog.Warn("email sent but failed to mark invoice as sent", "error", err, "id", invoice.ID)
		}
	}

	return nil
}

// SendByID loads the company and invoice by ID and then sends the email. Used by
// the recurring auto-send path, which only has identifiers.
func (s *InvoiceEmailService) SendByID(ctx context.Context, companyID, invoiceID int64, opts EmailOptions) error {
	company, err := s.company.GetByID(ctx, companyID)
	if err != nil {
		return fmt.Errorf("loading company %d for email: %w", companyID, err)
	}
	invoice, err := s.invoices.GetByID(ctx, companyID, invoiceID)
	if err != nil {
		return fmt.Errorf("loading invoice %d for email: %w", invoiceID, err)
	}
	return s.Send(ctx, &company, invoice, opts)
}
