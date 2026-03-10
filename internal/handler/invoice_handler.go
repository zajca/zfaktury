package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// InvoiceHandler handles HTTP requests for invoice management.
type InvoiceHandler struct {
	svc *service.InvoiceService
}

// NewInvoiceHandler creates a new InvoiceHandler.
func NewInvoiceHandler(svc *service.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{svc: svc}
}

// Routes registers invoice routes on the given router.
func (h *InvoiceHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/send", h.MarkAsSent)
	r.Post("/{id}/mark-paid", h.MarkAsPaid)
	r.Post("/{id}/duplicate", h.Duplicate)
	return r
}

// Create handles POST /api/v1/invoices.
func (h *InvoiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req invoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	invoice, err := req.toDomain()
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid date format, expected YYYY-MM-DD")
		return
	}

	if err := h.svc.Create(r.Context(), invoice); err != nil {
		slog.Error("failed to create invoice", "error", err)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, invoiceFromDomain(invoice))
}

// List handles GET /api/v1/invoices.
func (h *InvoiceHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	filter := domain.InvoiceFilter{
		Status:     r.URL.Query().Get("status"),
		CustomerID: parseOptionalInt64(r, "customer_id"),
		DateFrom:   parseOptionalTime(r, "date_from"),
		DateTo:     parseOptionalTime(r, "date_to"),
		Search:     r.URL.Query().Get("search"),
		Limit:      limit,
		Offset:     offset,
	}

	invoices, total, err := h.svc.List(r.Context(), filter)
	if err != nil {
		slog.Error("failed to list invoices", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list invoices")
		return
	}

	items := make([]invoiceResponse, 0, len(invoices))
	for i := range invoices {
		items = append(items, invoiceFromDomain(&invoices[i]))
	}

	respondJSON(w, http.StatusOK, listResponse[invoiceResponse]{
		Data:   items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// GetByID handles GET /api/v1/invoices/{id}.
func (h *InvoiceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	invoice, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get invoice", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "invoice not found")
		return
	}

	respondJSON(w, http.StatusOK, invoiceFromDomain(invoice))
}

// Update handles PUT /api/v1/invoices/{id}.
func (h *InvoiceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	var req invoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	invoice, err := req.toDomain()
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid date format, expected YYYY-MM-DD")
		return
	}
	invoice.ID = id

	if err := h.svc.Update(r.Context(), invoice); err != nil {
		slog.Error("failed to update invoice", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, invoiceFromDomain(invoice))
}

// Delete handles DELETE /api/v1/invoices/{id}.
func (h *InvoiceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		slog.Error("failed to delete invoice", "error", err, "id", id)
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// MarkAsSent handles POST /api/v1/invoices/{id}/send.
func (h *InvoiceHandler) MarkAsSent(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	if err := h.svc.MarkAsSent(r.Context(), id); err != nil {
		slog.Error("failed to mark invoice as sent", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// Return the updated invoice.
	invoice, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch updated invoice")
		return
	}

	respondJSON(w, http.StatusOK, invoiceFromDomain(invoice))
}

// MarkAsPaid handles POST /api/v1/invoices/{id}/pay.
func (h *InvoiceHandler) MarkAsPaid(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	var req markPaidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	paidAt := time.Now()
	if req.PaidAt != "" {
		parsed, err := time.Parse("2006-01-02", req.PaidAt)
		if err != nil {
			// Try RFC3339 as fallback.
			parsed, err = time.Parse(time.RFC3339, req.PaidAt)
			if err != nil {
				respondError(w, http.StatusBadRequest, "invalid paid_at date format")
				return
			}
		}
		paidAt = parsed
	}

	if err := h.svc.MarkAsPaid(r.Context(), id, domain.Amount(req.Amount), paidAt); err != nil {
		slog.Error("failed to mark invoice as paid", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// Return the updated invoice.
	invoice, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch updated invoice")
		return
	}

	respondJSON(w, http.StatusOK, invoiceFromDomain(invoice))
}

// Duplicate handles POST /api/v1/invoices/{id}/duplicate.
func (h *InvoiceHandler) Duplicate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	invoice, err := h.svc.Duplicate(r.Context(), id)
	if err != nil {
		slog.Error("failed to duplicate invoice", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, invoiceFromDomain(invoice))
}
