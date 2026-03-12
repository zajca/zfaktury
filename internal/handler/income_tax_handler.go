package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// IncomeTaxHandler handles HTTP requests for income tax return management.
type IncomeTaxHandler struct {
	svc *service.IncomeTaxReturnService
}

// NewIncomeTaxHandler creates a new IncomeTaxHandler.
func NewIncomeTaxHandler(svc *service.IncomeTaxReturnService) *IncomeTaxHandler {
	return &IncomeTaxHandler{svc: svc}
}

// Routes registers income tax return routes on the given router.
func (h *IncomeTaxHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/recalculate", h.Recalculate)
	r.Post("/{id}/generate-xml", h.GenerateXML)
	r.Get("/{id}/xml", h.DownloadXML)
	r.Post("/{id}/mark-filed", h.MarkFiled)
	return r
}

// incomeTaxRequest is the JSON request body for creating an income tax return.
type incomeTaxRequest struct {
	Year       int    `json:"year"`
	FilingType string `json:"filing_type"`
}

// incomeTaxResponse is the JSON response for an income tax return.
type incomeTaxResponse struct {
	ID         int64  `json:"id"`
	Year       int    `json:"year"`
	FilingType string `json:"filing_type"`

	TotalRevenue    int64 `json:"total_revenue"`
	ActualExpenses  int64 `json:"actual_expenses"`
	FlatRatePercent int   `json:"flat_rate_percent"`
	FlatRateAmount  int64 `json:"flat_rate_amount"`
	UsedExpenses    int64 `json:"used_expenses"`

	TaxBase         int64 `json:"tax_base"`
	TotalDeductions int64 `json:"total_deductions"`
	TaxBaseRounded  int64 `json:"tax_base_rounded"`
	TaxAt15        int64 `json:"tax_at_15"`
	TaxAt23        int64 `json:"tax_at_23"`
	TotalTax       int64 `json:"total_tax"`

	CreditBasic      int64 `json:"credit_basic"`
	CreditSpouse     int64 `json:"credit_spouse"`
	CreditDisability int64 `json:"credit_disability"`
	CreditStudent    int64 `json:"credit_student"`
	TotalCredits     int64 `json:"total_credits"`

	TaxAfterCredits int64 `json:"tax_after_credits"`
	ChildBenefit    int64 `json:"child_benefit"`
	TaxAfterBenefit int64 `json:"tax_after_benefit"`

	Prepayments int64 `json:"prepayments"`
	TaxDue      int64 `json:"tax_due"`

	CapitalIncomeGross int64 `json:"capital_income_gross"`
	CapitalIncomeTax   int64 `json:"capital_income_tax"`
	CapitalIncomeNet   int64 `json:"capital_income_net"`

	OtherIncomeGross    int64 `json:"other_income_gross"`
	OtherIncomeExpenses int64 `json:"other_income_expenses"`
	OtherIncomeExempt   int64 `json:"other_income_exempt"`
	OtherIncomeNet      int64 `json:"other_income_net"`

	HasXML    bool    `json:"has_xml"`
	Status    string  `json:"status"`
	FiledAt   *string `json:"filed_at,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// incomeTaxFromDomain converts a domain.IncomeTaxReturn to an incomeTaxResponse.
func incomeTaxFromDomain(itr *domain.IncomeTaxReturn) incomeTaxResponse {
	return incomeTaxResponse{
		ID:         itr.ID,
		Year:       itr.Year,
		FilingType: itr.FilingType,

		TotalRevenue:    int64(itr.TotalRevenue),
		ActualExpenses:  int64(itr.ActualExpenses),
		FlatRatePercent: itr.FlatRatePercent,
		FlatRateAmount:  int64(itr.FlatRateAmount),
		UsedExpenses:    int64(itr.UsedExpenses),

		TaxBase:         int64(itr.TaxBase),
		TotalDeductions: int64(itr.TotalDeductions),
		TaxBaseRounded:  int64(itr.TaxBaseRounded),
		TaxAt15:        int64(itr.TaxAt15),
		TaxAt23:        int64(itr.TaxAt23),
		TotalTax:       int64(itr.TotalTax),

		CreditBasic:      int64(itr.CreditBasic),
		CreditSpouse:     int64(itr.CreditSpouse),
		CreditDisability: int64(itr.CreditDisability),
		CreditStudent:    int64(itr.CreditStudent),
		TotalCredits:     int64(itr.TotalCredits),

		TaxAfterCredits: int64(itr.TaxAfterCredits),
		ChildBenefit:    int64(itr.ChildBenefit),
		TaxAfterBenefit: int64(itr.TaxAfterBenefit),

		Prepayments: int64(itr.Prepayments),
		TaxDue:      int64(itr.TaxDue),

		CapitalIncomeGross: int64(itr.CapitalIncomeGross),
		CapitalIncomeTax:   int64(itr.CapitalIncomeTax),
		CapitalIncomeNet:   int64(itr.CapitalIncomeNet),

		OtherIncomeGross:    int64(itr.OtherIncomeGross),
		OtherIncomeExpenses: int64(itr.OtherIncomeExpenses),
		OtherIncomeExempt:   int64(itr.OtherIncomeExempt),
		OtherIncomeNet:      int64(itr.OtherIncomeNet),

		HasXML:    len(itr.XMLData) > 0,
		Status:    itr.Status,
		FiledAt:   formatOptionalTime(itr.FiledAt),
		CreatedAt: itr.CreatedAt.Format(time.RFC3339),
		UpdatedAt: itr.UpdatedAt.Format(time.RFC3339),
	}
}

// mapIncomeTaxError maps domain errors to HTTP status codes.
func mapIncomeTaxError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		respondError(w, http.StatusNotFound, "income tax return not found")
	case errors.Is(err, domain.ErrFilingAlreadyExists):
		respondError(w, http.StatusConflict, "income tax return already exists for this year")
	case errors.Is(err, domain.ErrFilingAlreadyFiled):
		respondError(w, http.StatusConflict, "income tax return already filed")
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusBadRequest, err.Error())
	default:
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

// Create handles POST /api/v1/income-tax-returns.
func (h *IncomeTaxHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req incomeTaxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	itr := &domain.IncomeTaxReturn{
		Year:       req.Year,
		FilingType: req.FilingType,
	}

	if err := h.svc.Create(r.Context(), itr); err != nil {
		slog.Error("failed to create income tax return", "error", err)
		mapIncomeTaxError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, incomeTaxFromDomain(itr))
}

// List handles GET /api/v1/income-tax-returns.
func (h *IncomeTaxHandler) List(w http.ResponseWriter, r *http.Request) {
	year := 0
	if v := r.URL.Query().Get("year"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid year parameter")
			return
		}
		year = parsed
	}

	returns, err := h.svc.List(r.Context(), year)
	if err != nil {
		slog.Error("failed to list income tax returns", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list income tax returns")
		return
	}

	items := make([]incomeTaxResponse, 0, len(returns))
	for i := range returns {
		items = append(items, incomeTaxFromDomain(&returns[i]))
	}

	respondJSON(w, http.StatusOK, items)
}

// GetByID handles GET /api/v1/income-tax-returns/{id}.
func (h *IncomeTaxHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid income tax return ID")
		return
	}

	itr, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get income tax return", "error", err, "id", id)
		mapIncomeTaxError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, incomeTaxFromDomain(itr))
}

// Delete handles DELETE /api/v1/income-tax-returns/{id}.
func (h *IncomeTaxHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid income tax return ID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		slog.Error("failed to delete income tax return", "error", err, "id", id)
		mapIncomeTaxError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Recalculate handles POST /api/v1/income-tax-returns/{id}/recalculate.
func (h *IncomeTaxHandler) Recalculate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid income tax return ID")
		return
	}

	itr, err := h.svc.Recalculate(r.Context(), id)
	if err != nil {
		slog.Error("failed to recalculate income tax return", "error", err, "id", id)
		mapIncomeTaxError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, incomeTaxFromDomain(itr))
}

// GenerateXML handles POST /api/v1/income-tax-returns/{id}/generate-xml.
func (h *IncomeTaxHandler) GenerateXML(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid income tax return ID")
		return
	}

	itr, err := h.svc.GenerateXML(r.Context(), id)
	if err != nil {
		slog.Error("failed to generate income tax return XML", "error", err, "id", id)
		mapIncomeTaxError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, incomeTaxFromDomain(itr))
}

// DownloadXML handles GET /api/v1/income-tax-returns/{id}/xml.
func (h *IncomeTaxHandler) DownloadXML(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid income tax return ID")
		return
	}

	xmlData, err := h.svc.GetXMLData(r.Context(), id)
	if err != nil {
		slog.Error("failed to get income tax return XML", "error", err, "id", id)
		mapIncomeTaxError(w, err)
		return
	}

	if len(xmlData) == 0 {
		respondError(w, http.StatusNotFound, "XML not yet generated for this income tax return")
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	filename := fmt.Sprintf("dpfo-%d.xml", id)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(xmlData)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(xmlData)
}

// MarkFiled handles POST /api/v1/income-tax-returns/{id}/mark-filed.
func (h *IncomeTaxHandler) MarkFiled(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid income tax return ID")
		return
	}

	itr, err := h.svc.MarkFiled(r.Context(), id)
	if err != nil {
		slog.Error("failed to mark income tax return as filed", "error", err, "id", id)
		mapIncomeTaxError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, incomeTaxFromDomain(itr))
}
