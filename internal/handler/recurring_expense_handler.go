package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// RecurringExpenseHandler handles HTTP requests for recurring expense management.
type RecurringExpenseHandler struct {
	svc *service.RecurringExpenseService
}

// NewRecurringExpenseHandler creates a new RecurringExpenseHandler.
func NewRecurringExpenseHandler(svc *service.RecurringExpenseService) *RecurringExpenseHandler {
	return &RecurringExpenseHandler{svc: svc}
}

// Routes registers recurring expense routes on the given router.
func (h *RecurringExpenseHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Post("/generate", h.GeneratePending)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/activate", h.Activate)
	r.Post("/{id}/deactivate", h.Deactivate)
	return r
}

// --- DTOs ---

// recurringExpenseRequest is the JSON request body for creating/updating a recurring expense.
type recurringExpenseRequest struct {
	Name            string  `json:"name"`
	VendorID        *int64  `json:"vendor_id"`
	Category        string  `json:"category"`
	Description     string  `json:"description"`
	Amount          int64   `json:"amount"`
	CurrencyCode    string  `json:"currency_code"`
	ExchangeRate    int64   `json:"exchange_rate"`
	VATRatePercent  int     `json:"vat_rate_percent"`
	VATAmount       int64   `json:"vat_amount"`
	IsTaxDeductible bool    `json:"is_tax_deductible"`
	BusinessPercent int     `json:"business_percent"`
	PaymentMethod   string  `json:"payment_method"`
	Notes           string  `json:"notes"`
	Frequency       string  `json:"frequency"`
	NextIssueDate   string  `json:"next_issue_date"`
	EndDate         *string `json:"end_date"`
	IsActive        bool    `json:"is_active"`
}

// toDomain converts a recurringExpenseRequest to a domain.RecurringExpense.
func (r *recurringExpenseRequest) toDomain() (*domain.RecurringExpense, error) {
	if r.NextIssueDate == "" {
		return nil, fmt.Errorf("next_issue_date is required: %w", domain.ErrInvalidInput)
	}

	re := &domain.RecurringExpense{
		Name:            r.Name,
		VendorID:        r.VendorID,
		Category:        r.Category,
		Description:     r.Description,
		Amount:          domain.Amount(r.Amount),
		CurrencyCode:    r.CurrencyCode,
		ExchangeRate:    domain.Amount(r.ExchangeRate),
		VATRatePercent:  r.VATRatePercent,
		VATAmount:       domain.Amount(r.VATAmount),
		IsTaxDeductible: r.IsTaxDeductible,
		BusinessPercent: r.BusinessPercent,
		PaymentMethod:   r.PaymentMethod,
		Notes:           r.Notes,
		Frequency:       r.Frequency,
		IsActive:        r.IsActive,
	}

	nextIssueDate, err := time.Parse("2006-01-02", r.NextIssueDate)
	if err != nil {
		return nil, fmt.Errorf("invalid next_issue_date format, expected YYYY-MM-DD: %w", domain.ErrInvalidInput)
	}
	re.NextIssueDate = nextIssueDate

	if r.EndDate != nil && *r.EndDate != "" {
		t, err := time.Parse("2006-01-02", *r.EndDate)
		if err != nil {
			return nil, err
		}
		re.EndDate = &t
	}

	return re, nil
}

// recurringExpenseResponse is the JSON response for a recurring expense.
type recurringExpenseResponse struct {
	ID              int64            `json:"id"`
	Name            string           `json:"name"`
	VendorID        *int64           `json:"vendor_id,omitempty"`
	Vendor          *contactResponse `json:"vendor,omitempty"`
	Category        string           `json:"category"`
	Description     string           `json:"description"`
	Amount          int64            `json:"amount"`
	CurrencyCode    string           `json:"currency_code"`
	ExchangeRate    int64            `json:"exchange_rate"`
	VATRatePercent  int              `json:"vat_rate_percent"`
	VATAmount       int64            `json:"vat_amount"`
	IsTaxDeductible bool             `json:"is_tax_deductible"`
	BusinessPercent int              `json:"business_percent"`
	PaymentMethod   string           `json:"payment_method"`
	Notes           string           `json:"notes"`
	Frequency       string           `json:"frequency"`
	NextIssueDate   string           `json:"next_issue_date"`
	EndDate         *string          `json:"end_date,omitempty"`
	IsActive        bool             `json:"is_active"`
	CreatedAt       string           `json:"created_at"`
	UpdatedAt       string           `json:"updated_at"`
}

// recurringExpenseFromDomain converts a domain.RecurringExpense to a recurringExpenseResponse.
func recurringExpenseFromDomain(re *domain.RecurringExpense) recurringExpenseResponse {
	resp := recurringExpenseResponse{
		ID:              re.ID,
		Name:            re.Name,
		VendorID:        re.VendorID,
		Category:        re.Category,
		Description:     re.Description,
		Amount:          int64(re.Amount),
		CurrencyCode:    re.CurrencyCode,
		ExchangeRate:    int64(re.ExchangeRate),
		VATRatePercent:  re.VATRatePercent,
		VATAmount:       int64(re.VATAmount),
		IsTaxDeductible: re.IsTaxDeductible,
		BusinessPercent: re.BusinessPercent,
		PaymentMethod:   re.PaymentMethod,
		Notes:           re.Notes,
		Frequency:       re.Frequency,
		NextIssueDate:   re.NextIssueDate.Format("2006-01-02"),
		IsActive:        re.IsActive,
		CreatedAt:       re.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       re.UpdatedAt.Format(time.RFC3339),
	}

	if re.EndDate != nil {
		s := re.EndDate.Format("2006-01-02")
		resp.EndDate = &s
	}

	if re.Vendor != nil {
		v := contactFromDomain(re.Vendor)
		resp.Vendor = &v
	}

	return resp
}

// generateResponse is the JSON response for the generate endpoint.
type generateResponse struct {
	Generated int `json:"generated"`
}

// generateRequest is the JSON request body for the generate endpoint.
type generateRequest struct {
	AsOfDate string `json:"as_of_date"`
}

// --- Handlers ---

// Create handles POST /api/v1/recurring-expenses.
func (h *RecurringExpenseHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req recurringExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	re, err := req.toDomain()
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.svc.Create(r.Context(), re); err != nil {
		slog.Error("failed to create recurring expense", "error", err)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, recurringExpenseFromDomain(re))
}

// List handles GET /api/v1/recurring-expenses.
func (h *RecurringExpenseHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	items, total, err := h.svc.List(r.Context(), limit, offset)
	if err != nil {
		slog.Error("failed to list recurring expenses", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list recurring expenses")
		return
	}

	data := make([]recurringExpenseResponse, 0, len(items))
	for i := range items {
		data = append(data, recurringExpenseFromDomain(&items[i]))
	}

	respondJSON(w, http.StatusOK, listResponse[recurringExpenseResponse]{
		Data:   data,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// GetByID handles GET /api/v1/recurring-expenses/{id}.
func (h *RecurringExpenseHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid recurring expense ID")
		return
	}

	re, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get recurring expense", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "recurring expense not found")
		return
	}

	respondJSON(w, http.StatusOK, recurringExpenseFromDomain(re))
}

// Update handles PUT /api/v1/recurring-expenses/{id}.
func (h *RecurringExpenseHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid recurring expense ID")
		return
	}

	var req recurringExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	re, err := req.toDomain()
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	re.ID = id

	if err := h.svc.Update(r.Context(), re); err != nil {
		slog.Error("failed to update recurring expense", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, recurringExpenseFromDomain(re))
}

// Delete handles DELETE /api/v1/recurring-expenses/{id}.
func (h *RecurringExpenseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid recurring expense ID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		slog.Error("failed to delete recurring expense", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "recurring expense not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Activate handles POST /api/v1/recurring-expenses/{id}/activate.
func (h *RecurringExpenseHandler) Activate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid recurring expense ID")
		return
	}

	if err := h.svc.Activate(r.Context(), id); err != nil {
		slog.Error("failed to activate recurring expense", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Deactivate handles POST /api/v1/recurring-expenses/{id}/deactivate.
func (h *RecurringExpenseHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid recurring expense ID")
		return
	}

	if err := h.svc.Deactivate(r.Context(), id); err != nil {
		slog.Error("failed to deactivate recurring expense", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GeneratePending handles POST /api/v1/recurring-expenses/generate.
func (h *RecurringExpenseHandler) GeneratePending(w http.ResponseWriter, r *http.Request) {
	var req generateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var asOfDate time.Time
	if req.AsOfDate != "" {
		var err error
		asOfDate, err = time.Parse("2006-01-02", req.AsOfDate)
		if err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
	} else {
		asOfDate = time.Now()
	}

	// Restrict as_of_date to at most 7 days in the past and no future dates.
	today := time.Now().Truncate(24 * time.Hour)
	earliest := today.AddDate(0, 0, -7)
	if asOfDate.Before(earliest) {
		respondError(w, http.StatusBadRequest, "as_of_date cannot be more than 7 days in the past")
		return
	}
	if asOfDate.After(today.AddDate(0, 0, 1)) {
		respondError(w, http.StatusBadRequest, "as_of_date cannot be in the future")
		return
	}

	count, err := h.svc.GeneratePending(r.Context(), asOfDate)
	if err != nil {
		slog.Error("failed to generate pending expenses", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to generate pending expenses")
		return
	}

	respondJSON(w, http.StatusOK, generateResponse{Generated: count})
}
