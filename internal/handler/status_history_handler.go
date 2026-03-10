package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/service"
)

// StatusHistoryHandler handles HTTP requests for invoice status history and overdue detection.
type StatusHistoryHandler struct {
	overdueSvc *service.OverdueService
}

// NewStatusHistoryHandler creates a new StatusHistoryHandler.
func NewStatusHistoryHandler(overdueSvc *service.OverdueService) *StatusHistoryHandler {
	return &StatusHistoryHandler{overdueSvc: overdueSvc}
}

// statusChangeResponse is the JSON response for a status change record.
type statusChangeResponse struct {
	ID        int64  `json:"id"`
	InvoiceID int64  `json:"invoice_id"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
	ChangedAt string `json:"changed_at"`
	Note      string `json:"note"`
}

// checkOverdueResponse is the JSON response for the check-overdue endpoint.
type checkOverdueResponse struct {
	Marked int `json:"marked"`
}

// GetHistory returns the status change history for an invoice.
// GET /invoices/{id}/history
func (h *StatusHistoryHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	invoiceID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	changes, err := h.overdueSvc.GetHistory(r.Context(), invoiceID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get status history")
		return
	}

	resp := make([]statusChangeResponse, 0, len(changes))
	for _, c := range changes {
		resp = append(resp, statusChangeResponse{
			ID:        c.ID,
			InvoiceID: c.InvoiceID,
			OldStatus: c.OldStatus,
			NewStatus: c.NewStatus,
			ChangedAt: c.ChangedAt.Format(time.RFC3339),
			Note:      c.Note,
		})
	}

	respondJSON(w, http.StatusOK, resp)
}

// CheckOverdue triggers overdue detection for all sent invoices past their due date.
// POST /invoices/check-overdue
func (h *StatusHistoryHandler) CheckOverdue(w http.ResponseWriter, r *http.Request) {
	count, err := h.overdueSvc.CheckOverdue(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to check overdue invoices")
		return
	}

	respondJSON(w, http.StatusOK, checkOverdueResponse{Marked: count})
}
