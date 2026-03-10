package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// ReminderHandler handles HTTP requests for payment reminders.
type ReminderHandler struct {
	svc *service.ReminderService
}

// NewReminderHandler creates a new ReminderHandler.
func NewReminderHandler(svc *service.ReminderService) *ReminderHandler {
	return &ReminderHandler{svc: svc}
}

// Routes registers reminder routes on a chi router.
// These routes are intended to be mounted under /invoices/{invoiceID}.
func (h *ReminderHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/{id}/remind", h.SendReminder)
	r.Get("/{id}/reminders", h.ListReminders)
	return r
}

// reminderResponse is the JSON response DTO for a payment reminder.
type reminderResponse struct {
	ID             int64  `json:"id"`
	InvoiceID      int64  `json:"invoice_id"`
	ReminderNumber int    `json:"reminder_number"`
	SentAt         string `json:"sent_at"`
	SentTo         string `json:"sent_to"`
	Subject        string `json:"subject"`
	BodyPreview    string `json:"body_preview"`
	CreatedAt      string `json:"created_at"`
}

func reminderFromDomain(r *domain.PaymentReminder) reminderResponse {
	return reminderResponse{
		ID:             r.ID,
		InvoiceID:      r.InvoiceID,
		ReminderNumber: r.ReminderNumber,
		SentAt:         r.SentAt.Format(time.RFC3339),
		SentTo:         r.SentTo,
		Subject:        r.Subject,
		BodyPreview:    r.BodyPreview,
		CreatedAt:      r.CreatedAt.Format(time.RFC3339),
	}
}

func parseInvoiceID(r *http.Request) (int64, error) {
	idStr := chi.URLParam(r, "id")
	return strconv.ParseInt(idStr, 10, 64)
}

// SendReminder handles POST /{invoiceID}/remind.
func (h *ReminderHandler) SendReminder(w http.ResponseWriter, r *http.Request) {
	invoiceID, err := parseInvoiceID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	reminder, err := h.svc.SendReminder(r.Context(), invoiceID)
	if err != nil {
		slog.Error("failed to send payment reminder", "invoice_id", invoiceID, "error", err)
		switch {
		case errors.Is(err, service.ErrInvoiceNotOverdue):
			respondError(w, http.StatusUnprocessableEntity, "invoice is not overdue")
		case errors.Is(err, service.ErrNoCustomerEmail):
			respondError(w, http.StatusUnprocessableEntity, "customer has no email address")
		default:
			respondError(w, http.StatusInternalServerError, "failed to send reminder")
		}
		return
	}

	respondJSON(w, http.StatusCreated, reminderFromDomain(reminder))
}

// ListReminders handles GET /{invoiceID}/reminders.
func (h *ReminderHandler) ListReminders(w http.ResponseWriter, r *http.Request) {
	invoiceID, err := parseInvoiceID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	reminders, err := h.svc.GetReminders(r.Context(), invoiceID)
	if err != nil {
		slog.Error("failed to list payment reminders", "invoice_id", invoiceID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list reminders")
		return
	}

	var resp []reminderResponse
	for i := range reminders {
		resp = append(resp, reminderFromDomain(&reminders[i]))
	}
	if resp == nil {
		resp = []reminderResponse{}
	}

	respondJSON(w, http.StatusOK, resp)
}
