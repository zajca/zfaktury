package handler

import (
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/zajca/zfaktury/internal/pdf"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/service/email"
)

// EmailHandler handles sending invoices via email.
type EmailHandler struct {
	invoiceSvc  *service.InvoiceService
	settingsSvc *service.SettingsService
	pdfGen      *pdf.InvoicePDFGenerator
	sender      *email.EmailSender
}

// NewEmailHandler creates a new EmailHandler.
func NewEmailHandler(
	invoiceSvc *service.InvoiceService,
	settingsSvc *service.SettingsService,
	pdfGen *pdf.InvoicePDFGenerator,
	sender *email.EmailSender,
) *EmailHandler {
	return &EmailHandler{
		invoiceSvc:  invoiceSvc,
		settingsSvc: settingsSvc,
		pdfGen:      pdfGen,
		sender:      sender,
	}
}

// sendEmailRequest is the JSON request body for sending an invoice via email.
type sendEmailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// SendEmail handles POST /api/v1/invoices/{id}/send-email.
func (h *EmailHandler) SendEmail(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	if !h.sender.IsConfigured() {
		respondError(w, http.StatusUnprocessableEntity, "SMTP is not configured")
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

	invoice, err := h.invoiceSvc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get invoice for email", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "invoice not found")
		return
	}

	// Load supplier settings for PDF generation.
	settings, err := h.settingsSvc.GetAll(r.Context())
	if err != nil {
		slog.Error("failed to load supplier settings", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load supplier settings")
		return
	}

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

	pdfBytes, err := h.pdfGen.Generate(r.Context(), invoice, supplier)
	if err != nil {
		slog.Error("failed to generate PDF for email", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "failed to generate PDF")
		return
	}

	filename := fmt.Sprintf("faktura_%s.pdf", invoice.InvoiceNumber)

	msg := email.EmailMessage{
		To:       []string{req.To},
		Subject:  req.Subject,
		BodyText: req.Body,
		BodyHTML: "<p>" + strings.ReplaceAll(html.EscapeString(req.Body), "\n", "<br>") + "</p>",
		Attachments: []email.Attachment{
			{
				Filename:    filename,
				ContentType: "application/pdf",
				Data:        pdfBytes,
			},
		},
	}

	if err := h.sender.Send(r.Context(), msg); err != nil {
		slog.Error("failed to send invoice email", "error", err, "id", id, "to", req.To)
		respondError(w, http.StatusInternalServerError, "failed to send email")
		return
	}

	slog.Info("invoice email sent", "id", id, "to", req.To)
	respondJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}
