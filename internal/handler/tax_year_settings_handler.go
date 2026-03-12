package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// TaxYearSettingsHandler handles HTTP requests for per-year tax settings.
type TaxYearSettingsHandler struct {
	svc *service.TaxYearSettingsService
}

// NewTaxYearSettingsHandler creates a new TaxYearSettingsHandler.
func NewTaxYearSettingsHandler(svc *service.TaxYearSettingsService) *TaxYearSettingsHandler {
	return &TaxYearSettingsHandler{svc: svc}
}

// Routes registers tax year settings routes.
func (h *TaxYearSettingsHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{year}", h.GetByYear)
	r.Put("/{year}", h.Save)
	return r
}

type taxYearSettingsResponse struct {
	Year            int                `json:"year"`
	FlatRatePercent int                `json:"flat_rate_percent"`
	Prepayments     []taxPrepaymentDTO `json:"prepayments"`
}

type taxPrepaymentDTO struct {
	Month        int   `json:"month"`
	TaxAmount    int64 `json:"tax_amount"`
	SocialAmount int64 `json:"social_amount"`
	HealthAmount int64 `json:"health_amount"`
}

type taxYearSettingsRequest struct {
	FlatRatePercent int                `json:"flat_rate_percent"`
	Prepayments     []taxPrepaymentDTO `json:"prepayments"`
}

func parseYear(r *http.Request) (int, error) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		return 0, err
	}
	if year < 2000 || year > 2100 {
		return 0, fmt.Errorf("year out of range: %d", year)
	}
	return year, nil
}

// GetByYear handles GET /api/v1/tax-year-settings/{year}.
func (h *TaxYearSettingsHandler) GetByYear(w http.ResponseWriter, r *http.Request) {
	year, err := parseYear(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	tys, err := h.svc.GetByYear(r.Context(), year)
	if err != nil {
		slog.Error("failed to get tax year settings", "error", err, "year", year)
		respondError(w, http.StatusInternalServerError, "failed to get tax year settings")
		return
	}

	prepayments, err := h.svc.GetPrepayments(r.Context(), year)
	if err != nil {
		slog.Error("failed to get prepayments", "error", err, "year", year)
		respondError(w, http.StatusInternalServerError, "failed to get prepayments")
		return
	}

	resp := taxYearSettingsResponse{
		Year:            tys.Year,
		FlatRatePercent: tys.FlatRatePercent,
		Prepayments:     make([]taxPrepaymentDTO, 0, len(prepayments)),
	}
	for _, tp := range prepayments {
		resp.Prepayments = append(resp.Prepayments, taxPrepaymentDTO{
			Month:        tp.Month,
			TaxAmount:    int64(tp.TaxAmount),
			SocialAmount: int64(tp.SocialAmount),
			HealthAmount: int64(tp.HealthAmount),
		})
	}

	respondJSON(w, http.StatusOK, resp)
}

// Save handles PUT /api/v1/tax-year-settings/{year}.
func (h *TaxYearSettingsHandler) Save(w http.ResponseWriter, r *http.Request) {
	year, err := parseYear(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	var req taxYearSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	validRates := map[int]bool{0: true, 30: true, 40: true, 60: true, 80: true}
	if !validRates[req.FlatRatePercent] {
		respondError(w, http.StatusBadRequest, "flat_rate_percent must be one of 0, 30, 40, 60, 80")
		return
	}

	prepayments := make([]domain.TaxPrepayment, 0, len(req.Prepayments))
	for _, dto := range req.Prepayments {
		if dto.Month < 1 || dto.Month > 12 {
			respondError(w, http.StatusBadRequest, "month must be between 1 and 12")
			return
		}
		if dto.TaxAmount < 0 || dto.SocialAmount < 0 || dto.HealthAmount < 0 {
			respondError(w, http.StatusBadRequest, "prepayment amounts must not be negative")
			return
		}
		prepayments = append(prepayments, domain.TaxPrepayment{
			Year:         year,
			Month:        dto.Month,
			TaxAmount:    domain.Amount(dto.TaxAmount),
			SocialAmount: domain.Amount(dto.SocialAmount),
			HealthAmount: domain.Amount(dto.HealthAmount),
		})
	}

	if err := h.svc.Save(r.Context(), year, req.FlatRatePercent, prepayments); err != nil {
		slog.Error("failed to save tax year settings", "error", err, "year", year)
		respondError(w, http.StatusInternalServerError, "failed to save tax year settings")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
