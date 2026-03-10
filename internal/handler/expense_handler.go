package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// ExpenseHandler handles HTTP requests for expense management.
type ExpenseHandler struct {
	svc *service.ExpenseService
}

// NewExpenseHandler creates a new ExpenseHandler.
func NewExpenseHandler(svc *service.ExpenseService) *ExpenseHandler {
	return &ExpenseHandler{svc: svc}
}

// Routes registers expense routes on the given router.
func (h *ExpenseHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Post("/review", h.MarkTaxReviewed)
	r.Post("/unreview", h.UnmarkTaxReviewed)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	return r
}

// bulkIDsRequest is the JSON request body for bulk ID operations.
type bulkIDsRequest struct {
	IDs []int64 `json:"ids"`
}

// Create handles POST /api/v1/expenses.
func (h *ExpenseHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req expenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	expense, err := req.toDomain()
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.svc.Create(r.Context(), expense); err != nil {
		slog.Error("failed to create expense", "error", err)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, expenseFromDomain(expense))
}

// List handles GET /api/v1/expenses.
func (h *ExpenseHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	filter := domain.ExpenseFilter{
		Category:    r.URL.Query().Get("category"),
		VendorID:    parseOptionalInt64(r, "vendor_id"),
		DateFrom:    parseOptionalTime(r, "date_from"),
		DateTo:      parseOptionalTime(r, "date_to"),
		Search:      r.URL.Query().Get("search"),
		TaxReviewed: parseOptionalBool(r, "tax_reviewed"),
		Limit:       limit,
		Offset:      offset,
	}

	expenses, total, err := h.svc.List(r.Context(), filter)
	if err != nil {
		slog.Error("failed to list expenses", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list expenses")
		return
	}

	items := make([]expenseResponse, 0, len(expenses))
	for i := range expenses {
		items = append(items, expenseFromDomain(&expenses[i]))
	}

	respondJSON(w, http.StatusOK, listResponse[expenseResponse]{
		Data:   items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// GetByID handles GET /api/v1/expenses/{id}.
func (h *ExpenseHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid expense ID")
		return
	}

	expense, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get expense", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "expense not found")
		return
	}

	respondJSON(w, http.StatusOK, expenseFromDomain(expense))
}

// Update handles PUT /api/v1/expenses/{id}.
func (h *ExpenseHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid expense ID")
		return
	}

	var req expenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	expense, err := req.toDomain()
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	expense.ID = id

	if err := h.svc.Update(r.Context(), expense); err != nil {
		slog.Error("failed to update expense", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, expenseFromDomain(expense))
}

// Delete handles DELETE /api/v1/expenses/{id}.
func (h *ExpenseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid expense ID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		slog.Error("failed to delete expense", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "expense not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// MarkTaxReviewed handles POST /api/v1/expenses/review.
func (h *ExpenseHandler) MarkTaxReviewed(w http.ResponseWriter, r *http.Request) {
	var req bulkIDsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.svc.MarkTaxReviewed(r.Context(), req.IDs); err != nil {
		slog.Error("failed to mark expenses as tax reviewed", "error", err)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UnmarkTaxReviewed handles POST /api/v1/expenses/unreview.
func (h *ExpenseHandler) UnmarkTaxReviewed(w http.ResponseWriter, r *http.Request) {
	var req bulkIDsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.svc.UnmarkTaxReviewed(r.Context(), req.IDs); err != nil {
		slog.Error("failed to unmark expenses as tax reviewed", "error", err)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
