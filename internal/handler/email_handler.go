package handler

import (
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/zajca/zfaktury/internal/isdoc"
	"github.com/zajca/zfaktury/internal/pdf"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/service/email"
)

// EmailHandler handles sending invoices via email.
type EmailHandler struct {
	invoiceSvc  *service.InvoiceService
	settingsSvc *service.SettingsService
	pdfGen      *pdf.InvoicePDFGenerator
	isdocGen    *isdoc.ISDOCGenerator
	sender      *email.EmailSender
}

// NewEmailHandler creates a new EmailHandler.
func NewEmailHandler(
	invoiceSvc *service.InvoiceService,
	settingsSvc *service.SettingsService,
	pdfGen *pdf.InvoicePDFGenerator,
	isdocGen *isdoc.ISDOCGenerator,
	sender *email.EmailSender,
) *EmailHandler {
	return &EmailHandler{
		invoiceSvc:  invoiceSvc,
		settingsSvc: settingsSvc,
		pdfGen:      pdfGen,
		isdocGen:    isdocGen,
		sender:      sender,
	}
}

// sendEmailRequest is the JSON request body for sending an invoice via email.
type sendEmailRequest struct {
	To          string `json:"to"`
	Subject     string `json:"subject"`
	Body        string `json:"body"`
	AttachPDF   *bool  `json:"attach_pdf"`
	AttachISDOC *bool  `json:"attach_isdoc"`
}

// emailDefaultsResponse is the response for GET /api/v1/email/defaults.
type emailDefaultsResponse struct {
	AttachPDF   bool   `json:"attach_pdf"`
	AttachISDOC bool   `json:"attach_isdoc"`
	Subject     string `json:"subject"`
	Body        string `json:"body"`
}

// GetDefaults handles GET /api/v1/email/defaults?invoice_number=XXX.
func (h *EmailHandler) GetDefaults(w http.ResponseWriter, r *http.Request) {
	invoiceNumber := r.URL.Query().Get("invoice_number")

	settings, err := h.settingsSvc.GetAll(r.Context())
	if err != nil {
		slog.Error("failed to load settings for email defaults", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load settings")
		return
	}

	attachPDF := true
	if v, ok := settings[service.SettingEmailAttachPDF]; ok {
		attachPDF = v == "true"
	}

	attachISDOC := false
	if v, ok := settings[service.SettingEmailAttachISDOC]; ok {
		attachISDOC = v == "true"
	}

	subjectTpl := settings[service.SettingEmailSubjectTpl]
	if subjectTpl == "" {
		subjectTpl = "Faktura {invoice_number}"
	}

	bodyTpl := settings[service.SettingEmailBodyTpl]
	if bodyTpl == "" {
		bodyTpl = "Dobrý den,\n\nv příloze zasíláme fakturu {invoice_number}.\n\nS pozdravem"
	}

	subject := strings.ReplaceAll(subjectTpl, "{invoice_number}", invoiceNumber)
	body := strings.ReplaceAll(bodyTpl, "{invoice_number}", invoiceNumber)

	respondJSON(w, http.StatusOK, emailDefaultsResponse{
		AttachPDF:   attachPDF,
		AttachISDOC: attachISDOC,
		Subject:     subject,
		Body:        body,
	})
}

// SendEmail handles POST /api/v1/invoices/{id}/send-email.
func (h *EmailHandler) SendEmail(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	if h.sender == nil || !h.sender.IsConfigured() {
		respondError(w, http.StatusUnprocessableEntity, "SMTP is not configured. Configure [smtp] section in config.toml")
		return
	}

	var req sendEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.To == "" {
		respondError(w, http.StatusBadRequest, "recipient email (to) is required")
		return
	}
	if _, err := mail.ParseAddress(req.To); err != nil {
		respondError(w, http.StatusBadRequest, "invalid recipient email address")
		return
	}
	if req.Subject == "" {
		respondError(w, http.StatusBadRequest, "subject is required")
		return
	}

	// Default attachment flags: PDF=true, ISDOC=false when not specified.
	attachPDF := true
	if req.AttachPDF != nil {
		attachPDF = *req.AttachPDF
	}
	attachISDOC := false
	if req.AttachISDOC != nil {
		attachISDOC = *req.AttachISDOC
	}

	if !attachPDF && !attachISDOC {
		respondError(w, http.StatusBadRequest, "at least one attachment type must be selected")
		return
	}

	invoice, err := h.invoiceSvc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get invoice for email", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "invoice not found")
		return
	}

	// Load supplier settings.
	settings, err := h.settingsSvc.GetAll(r.Context())
	if err != nil {
		slog.Error("failed to load supplier settings", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load supplier settings")
		return
	}

	var attachments []email.Attachment

	if attachPDF {
		supplier := pdf.SupplierInfo{
			Name:          settings[service.SettingCompanyName],
			ICO:           settings[service.SettingICO],
			DIC:           settings[service.SettingDIC],
			VATRegistered: settings[service.SettingVATRegistered] == "true",
			Street:        settings[service.SettingStreet],
			City:          settings[service.SettingCity],
			ZIP:           settings[service.SettingZIP],
			Email:         settings[service.SettingEmail],
			Phone:         settings[service.SettingPhone],
			BankAccount:   settings[service.SettingBankAccount],
			BankCode:      settings[service.SettingBankCode],
			IBAN:          settings[service.SettingIBAN],
			SWIFT:         settings[service.SettingSWIFT],
		}

		pdfSvcSettings, err := h.settingsSvc.GetPDFSettings(r.Context())
		if err != nil {
			slog.Error("failed to load PDF settings for email", "error", err)
			respondError(w, http.StatusInternalServerError, "failed to load PDF settings")
			return
		}
		pdfSettings := pdf.PDFSettings{
			LogoPath:        pdfSvcSettings.LogoPath,
			AccentColor:     pdfSvcSettings.AccentColor,
			FooterText:      pdfSvcSettings.FooterText,
			ShowQR:          pdfSvcSettings.ShowQR,
			ShowBankDetails: pdfSvcSettings.ShowBankDetails,
			FontSize:        pdfSvcSettings.FontSize,
		}

		pdfBytes, err := h.pdfGen.Generate(r.Context(), invoice, supplier, pdfSettings)
		if err != nil {
			slog.Error("failed to generate PDF for email", "error", err, "id", id)
			respondError(w, http.StatusInternalServerError, "failed to generate PDF")
			return
		}

		attachments = append(attachments, email.Attachment{
			Filename:    fmt.Sprintf("faktura_%s.pdf", invoice.InvoiceNumber),
			ContentType: "application/pdf",
			Data:        pdfBytes,
		})
	}

	if attachISDOC {
		isdocSupplier := isdoc.SupplierInfo{
			CompanyName: settings[service.SettingCompanyName],
			ICO:         settings[service.SettingICO],
			DIC:         settings[service.SettingDIC],
			Street:      settings[service.SettingStreet],
			City:        settings[service.SettingCity],
			ZIP:         settings[service.SettingZIP],
			Email:       settings[service.SettingEmail],
			Phone:       settings[service.SettingPhone],
			BankAccount: settings[service.SettingBankAccount],
			BankCode:    settings[service.SettingBankCode],
			IBAN:        settings[service.SettingIBAN],
			SWIFT:       settings[service.SettingSWIFT],
		}

		isdocBytes, err := h.isdocGen.Generate(r.Context(), invoice, isdocSupplier)
		if err != nil {
			slog.Error("failed to generate ISDOC for email", "error", err, "id", id)
			respondError(w, http.StatusInternalServerError, "failed to generate ISDOC")
			return
		}

		attachments = append(attachments, email.Attachment{
			Filename:    fmt.Sprintf("%s.isdoc", invoice.InvoiceNumber),
			ContentType: "application/xml",
			Data:        isdocBytes,
		})
	}

	msg := email.EmailMessage{
		To:          []string{req.To},
		Subject:     req.Subject,
		BodyText:    req.Body,
		BodyHTML:    "<p>" + strings.ReplaceAll(html.EscapeString(req.Body), "\n", "<br>") + "</p>",
		Attachments: attachments,
	}

	if err := h.sender.Send(r.Context(), msg); err != nil {
		slog.Error("failed to send invoice email", "error", err, "id", id, "to", req.To)
		respondError(w, http.StatusInternalServerError, "failed to send email")
		return
	}

	slog.Info("invoice email sent", "id", id, "to", req.To)
	respondJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}
