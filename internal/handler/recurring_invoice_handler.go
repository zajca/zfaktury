package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// --- Recurring Invoice DTOs (local to handler, not in helpers.go) ---

type recurringInvoiceItemRequest struct {
	Description    string `json:"description"`
	Quantity       int64  `json:"quantity"` // in smallest unit (cents)
	Unit           string `json:"unit"`
	UnitPrice      int64  `json:"unit_price"` // in halere
	VATRatePercent int    `json:"vat_rate_percent"`
	SortOrder      int    `json:"sort_order"`
}

type recurringInvoiceRequest struct {
	Name           string                        `json:"name"`
	CustomerID     int64                         `json:"customer_id"`
	Frequency      string                        `json:"frequency"`
	NextIssueDate  string                        `json:"next_issue_date"`
	EndDate        *string                       `json:"end_date"`
	CurrencyCode   string                        `json:"currency_code"`
	ExchangeRate   int64                         `json:"exchange_rate"`
	PaymentMethod  string                        `json:"payment_method"`
	BankAccount    string                        `json:"bank_account"`
	BankCode       string                        `json:"bank_code"`
	IBAN           string                        `json:"iban"`
	SWIFT          string                        `json:"swift"`
	ConstantSymbol string                        `json:"constant_symbol"`
	Notes          string                        `json:"notes"`
	IsActive       bool                          `json:"is_active"`
	Items          []recurringInvoiceItemRequest `json:"items"`
}

func (r *recurringInvoiceRequest) toDomain() (*domain.RecurringInvoice, error) {
	if r.NextIssueDate == "" {
		return nil, errors.New("next_issue_date is required")
	}

	ri := &domain.RecurringInvoice{
		Name:           r.Name,
		CustomerID:     r.CustomerID,
		Frequency:      r.Frequency,
		CurrencyCode:   r.CurrencyCode,
		ExchangeRate:   domain.Amount(r.ExchangeRate),
		PaymentMethod:  r.PaymentMethod,
		BankAccount:    r.BankAccount,
		BankCode:       r.BankCode,
		IBAN:           r.IBAN,
		SWIFT:          r.SWIFT,
		ConstantSymbol: r.ConstantSymbol,
		Notes:          r.Notes,
		IsActive:       r.IsActive,
	}

	nextIssueDate, err := time.Parse("2006-01-02", r.NextIssueDate)
	if err != nil {
		return nil, errors.New("invalid next_issue_date format, expected YYYY-MM-DD")
	}
	ri.NextIssueDate = nextIssueDate

	if r.EndDate != nil && *r.EndDate != "" {
		t, err := time.Parse("2006-01-02", *r.EndDate)
		if err != nil {
			return nil, err
		}
		ri.EndDate = &t
	}

	for _, item := range r.Items {
		ri.Items = append(ri.Items, domain.RecurringInvoiceItem{
			Description:    item.Description,
			Quantity:       domain.Amount(item.Quantity),
			Unit:           item.Unit,
			UnitPrice:      domain.Amount(item.UnitPrice),
			VATRatePercent: item.VATRatePercent,
			SortOrder:      item.SortOrder,
		})
	}

	return ri, nil
}

type recurringInvoiceItemResponse struct {
	ID                 int64  `json:"id"`
	RecurringInvoiceID int64  `json:"recurring_invoice_id"`
	Description        string `json:"description"`
	Quantity           int64  `json:"quantity"`
	Unit               string `json:"unit"`
	UnitPrice          int64  `json:"unit_price"`
	VATRatePercent     int    `json:"vat_rate_percent"`
	SortOrder          int    `json:"sort_order"`
}

type recurringInvoiceResponse struct {
	ID             int64                          `json:"id"`
	Name           string                         `json:"name"`
	CustomerID     int64                          `json:"customer_id"`
	Customer       *contactResponse               `json:"customer,omitempty"`
	Frequency      string                         `json:"frequency"`
	NextIssueDate  string                         `json:"next_issue_date"`
	EndDate        *string                        `json:"end_date,omitempty"`
	CurrencyCode   string                         `json:"currency_code"`
	ExchangeRate   int64                          `json:"exchange_rate"`
	PaymentMethod  string                         `json:"payment_method"`
	BankAccount    string                         `json:"bank_account"`
	BankCode       string                         `json:"bank_code"`
	IBAN           string                         `json:"iban"`
	SWIFT          string                         `json:"swift"`
	ConstantSymbol string                         `json:"constant_symbol"`
	Notes          string                         `json:"notes"`
	IsActive       bool                           `json:"is_active"`
	Items          []recurringInvoiceItemResponse `json:"items"`
	CreatedAt      string                         `json:"created_at"`
	UpdatedAt      string                         `json:"updated_at"`
}

func recurringInvoiceFromDomain(ri *domain.RecurringInvoice) recurringInvoiceResponse {
	resp := recurringInvoiceResponse{
		ID:             ri.ID,
		Name:           ri.Name,
		CustomerID:     ri.CustomerID,
		Frequency:      ri.Frequency,
		NextIssueDate:  ri.NextIssueDate.Format("2006-01-02"),
		CurrencyCode:   ri.CurrencyCode,
		ExchangeRate:   int64(ri.ExchangeRate),
		PaymentMethod:  ri.PaymentMethod,
		BankAccount:    ri.BankAccount,
		BankCode:       ri.BankCode,
		IBAN:           ri.IBAN,
		SWIFT:          ri.SWIFT,
		ConstantSymbol: ri.ConstantSymbol,
		Notes:          ri.Notes,
		IsActive:       ri.IsActive,
		CreatedAt:      ri.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      ri.UpdatedAt.Format(time.RFC3339),
	}

	if ri.EndDate != nil {
		s := ri.EndDate.Format("2006-01-02")
		resp.EndDate = &s
	}

	if ri.Customer != nil {
		c := contactFromDomain(ri.Customer)
		resp.Customer = &c
	}

	for _, item := range ri.Items {
		resp.Items = append(resp.Items, recurringInvoiceItemResponse{
			ID:                 item.ID,
			RecurringInvoiceID: item.RecurringInvoiceID,
			Description:        item.Description,
			Quantity:           int64(item.Quantity),
			Unit:               item.Unit,
			UnitPrice:          int64(item.UnitPrice),
			VATRatePercent:     item.VATRatePercent,
			SortOrder:          item.SortOrder,
		})
	}

	return resp
}

// --- Handler ---

// RecurringInvoiceHandler handles HTTP requests for recurring invoice management.
type RecurringInvoiceHandler struct {
	svc *service.RecurringInvoiceService
}

// NewRecurringInvoiceHandler creates a new RecurringInvoiceHandler.
func NewRecurringInvoiceHandler(svc *service.RecurringInvoiceService) *RecurringInvoiceHandler {
	return &RecurringInvoiceHandler{svc: svc}
}

// Routes registers recurring invoice routes on the given router.
func (h *RecurringInvoiceHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/generate", h.GenerateInvoice)
	r.Post("/process-due", h.ProcessDue)
	return r
}

// Create handles POST /api/v1/recurring-invoices.
func (h *RecurringInvoiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req recurringInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ri, err := req.toDomain()
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.svc.Create(r.Context(), ri); err != nil {
		slog.Error("failed to create recurring invoice", "error", err)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, recurringInvoiceFromDomain(ri))
}

// List handles GET /api/v1/recurring-invoices.
func (h *RecurringInvoiceHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		slog.Error("failed to list recurring invoices", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list recurring invoices")
		return
	}

	items := make([]recurringInvoiceResponse, 0, len(list))
	for i := range list {
		items = append(items, recurringInvoiceFromDomain(&list[i]))
	}

	respondJSON(w, http.StatusOK, items)
}

// GetByID handles GET /api/v1/recurring-invoices/{id}.
func (h *RecurringInvoiceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid recurring invoice ID")
		return
	}

	ri, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get recurring invoice", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "recurring invoice not found")
		return
	}

	respondJSON(w, http.StatusOK, recurringInvoiceFromDomain(ri))
}

// Update handles PUT /api/v1/recurring-invoices/{id}.
func (h *RecurringInvoiceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid recurring invoice ID")
		return
	}

	var req recurringInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ri, err := req.toDomain()
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	ri.ID = id

	if err := h.svc.Update(r.Context(), ri); err != nil {
		slog.Error("failed to update recurring invoice", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	updated, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch updated recurring invoice")
		return
	}

	respondJSON(w, http.StatusOK, recurringInvoiceFromDomain(updated))
}

// Delete handles DELETE /api/v1/recurring-invoices/{id}.
func (h *RecurringInvoiceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid recurring invoice ID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		slog.Error("failed to delete recurring invoice", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "recurring invoice not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GenerateInvoice handles POST /api/v1/recurring-invoices/{id}/generate.
func (h *RecurringInvoiceHandler) GenerateInvoice(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid recurring invoice ID")
		return
	}

	invoice, err := h.svc.GenerateInvoice(r.Context(), id)
	if err != nil {
		slog.Error("failed to generate invoice from recurring", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, invoiceFromDomain(invoice))
}

// processDueResponse is the JSON response for ProcessDue.
type processDueResponse struct {
	GeneratedCount int `json:"generated_count"`
}

// ProcessDue handles POST /api/v1/recurring-invoices/process-due.
func (h *RecurringInvoiceHandler) ProcessDue(w http.ResponseWriter, r *http.Request) {
	count, err := h.svc.ProcessDue(r.Context())
	if err != nil {
		slog.Error("failed to process due recurring invoices", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to process due recurring invoices")
		return
	}

	respondJSON(w, http.StatusOK, processDueResponse{GeneratedCount: count})
}
