package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/mail"

	"github.com/zajca/zfaktury/internal/service"
)

// EmailHandler handles sending invoices via email. The PDF/ISDOC generation,
// SMTP send, and draft-to-sent transition live in service.InvoiceEmailService,
// which the recurring auto-send path reuses; this handler keeps only the HTTP
// concerns (validation, status codes).
type EmailHandler struct {
	invoiceSvc *service.InvoiceService
	emailSvc   *service.InvoiceEmailService
}

// NewEmailHandler creates a new EmailHandler.
func NewEmailHandler(invoiceSvc *service.InvoiceService, emailSvc *service.InvoiceEmailService) *EmailHandler {
	return &EmailHandler{
		invoiceSvc: invoiceSvc,
		emailSvc:   emailSvc,
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

// GetDefaults handles GET /api/v1/companies/{companyID}/email/defaults?invoice_number=XXX.
func (h *EmailHandler) GetDefaults(w http.ResponseWriter, r *http.Request) {
	company, err := CompanyFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "company context missing")
		return
	}

	invoiceNumber := r.URL.Query().Get("invoice_number")

	defaults, err := h.emailSvc.Defaults(r.Context(), company.ID, invoiceNumber)
	if err != nil {
		slog.Error("failed to load settings for email defaults", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load settings")
		return
	}

	respondJSON(w, http.StatusOK, emailDefaultsResponse{
		AttachPDF:   defaults.AttachPDF,
		AttachISDOC: defaults.AttachISDOC,
		Subject:     defaults.Subject,
		Body:        defaults.Body,
	})
}

// SendEmail handles POST /api/v1/companies/{companyID}/invoices/{id}/send-email.
func (h *EmailHandler) SendEmail(w http.ResponseWriter, r *http.Request) {
	company, err := CompanyFromContext(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "no company in context")
		return
	}

	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	if !h.emailSvc.IsConfigured() {
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

	invoice, err := h.invoiceSvc.GetByID(r.Context(), company.ID, id)
	if err != nil {
		slog.Error("failed to get invoice for email", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "invoice not found")
		return
	}

	opts := service.EmailOptions{
		To:          req.To,
		Subject:     req.Subject,
		Body:        req.Body,
		AttachPDF:   attachPDF,
		AttachISDOC: attachISDOC,
	}

	if err := h.emailSvc.Send(r.Context(), company, invoice, opts); err != nil {
		slog.Error("failed to send invoice email", "error", err, "id", id, "to", req.To)
		respondError(w, http.StatusInternalServerError, "failed to send email")
		return
	}

	slog.Info("invoice email sent", "id", id, "to", req.To)
	respondJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}
