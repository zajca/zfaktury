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

// SocialInsuranceHandler handles HTTP requests for social insurance overview management.
type SocialInsuranceHandler struct {
	svc *service.SocialInsuranceService
}

// NewSocialInsuranceHandler creates a new SocialInsuranceHandler.
func NewSocialInsuranceHandler(svc *service.SocialInsuranceService) *SocialInsuranceHandler {
	return &SocialInsuranceHandler{svc: svc}
}

// Routes registers social insurance overview routes on the given router.
func (h *SocialInsuranceHandler) Routes() chi.Router {
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

// --- DTOs ---

// socialInsuranceRequest is the JSON request body for creating a social insurance overview.
type socialInsuranceRequest struct {
	Year       int    `json:"year"`
	FilingType string `json:"filing_type"`
}

// socialInsuranceResponse is the JSON response for a social insurance overview.
type socialInsuranceResponse struct {
	ID                  int64   `json:"id"`
	Year                int     `json:"year"`
	FilingType          string  `json:"filing_type"`
	TotalRevenue        int64   `json:"total_revenue"`
	TotalExpenses       int64   `json:"total_expenses"`
	TaxBase             int64   `json:"tax_base"`
	AssessmentBase      int64   `json:"assessment_base"`
	MinAssessmentBase   int64   `json:"min_assessment_base"`
	FinalAssessmentBase int64   `json:"final_assessment_base"`
	InsuranceRate       int     `json:"insurance_rate"`
	TotalInsurance      int64   `json:"total_insurance"`
	Prepayments         int64   `json:"prepayments"`
	Difference          int64   `json:"difference"`
	NewMonthlyPrepay    int64   `json:"new_monthly_prepay"`
	HasXML              bool    `json:"has_xml"`
	Status              string  `json:"status"`
	FiledAt             *string `json:"filed_at,omitempty"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

// socialInsuranceFromDomain converts a domain.SocialInsuranceOverview to a socialInsuranceResponse.
func socialInsuranceFromDomain(sio *domain.SocialInsuranceOverview) socialInsuranceResponse {
	return socialInsuranceResponse{
		ID:                  sio.ID,
		Year:                sio.Year,
		FilingType:          sio.FilingType,
		TotalRevenue:        int64(sio.TotalRevenue),
		TotalExpenses:       int64(sio.TotalExpenses),
		TaxBase:             int64(sio.TaxBase),
		AssessmentBase:      int64(sio.AssessmentBase),
		MinAssessmentBase:   int64(sio.MinAssessmentBase),
		FinalAssessmentBase: int64(sio.FinalAssessmentBase),
		InsuranceRate:       sio.InsuranceRate,
		TotalInsurance:      int64(sio.TotalInsurance),
		Prepayments:         int64(sio.Prepayments),
		Difference:          int64(sio.Difference),
		NewMonthlyPrepay:    int64(sio.NewMonthlyPrepay),
		HasXML:              len(sio.XMLData) > 0,
		Status:              sio.Status,
		FiledAt:             formatOptionalTime(sio.FiledAt),
		CreatedAt:           sio.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           sio.UpdatedAt.Format(time.RFC3339),
	}
}

// mapSocialInsuranceError maps domain errors to HTTP status codes.
func mapSocialInsuranceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		respondError(w, http.StatusNotFound, "social insurance overview not found")
	case errors.Is(err, domain.ErrFilingAlreadyExists):
		respondError(w, http.StatusConflict, "social insurance overview already exists for this year")
	case errors.Is(err, domain.ErrFilingAlreadyFiled):
		respondError(w, http.StatusConflict, "social insurance overview already filed")
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusBadRequest, err.Error())
	default:
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

// Create handles POST /api/v1/social-insurance.
func (h *SocialInsuranceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req socialInsuranceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sio := &domain.SocialInsuranceOverview{
		Year:       req.Year,
		FilingType: req.FilingType,
	}

	if err := h.svc.Create(r.Context(), sio); err != nil {
		slog.Error("failed to create social insurance overview", "error", err)
		mapSocialInsuranceError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, socialInsuranceFromDomain(sio))
}

// List handles GET /api/v1/social-insurance.
func (h *SocialInsuranceHandler) List(w http.ResponseWriter, r *http.Request) {
	year := 0
	if v := r.URL.Query().Get("year"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid year parameter")
			return
		}
		year = parsed
	}

	overviews, err := h.svc.List(r.Context(), year)
	if err != nil {
		slog.Error("failed to list social insurance overviews", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list social insurance overviews")
		return
	}

	items := make([]socialInsuranceResponse, 0, len(overviews))
	for i := range overviews {
		items = append(items, socialInsuranceFromDomain(&overviews[i]))
	}

	respondJSON(w, http.StatusOK, items)
}

// GetByID handles GET /api/v1/social-insurance/{id}.
func (h *SocialInsuranceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid social insurance overview ID")
		return
	}

	sio, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get social insurance overview", "error", err, "id", id)
		mapSocialInsuranceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, socialInsuranceFromDomain(sio))
}

// Delete handles DELETE /api/v1/social-insurance/{id}.
func (h *SocialInsuranceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid social insurance overview ID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		slog.Error("failed to delete social insurance overview", "error", err, "id", id)
		mapSocialInsuranceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Recalculate handles POST /api/v1/social-insurance/{id}/recalculate.
func (h *SocialInsuranceHandler) Recalculate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid social insurance overview ID")
		return
	}

	sio, err := h.svc.Recalculate(r.Context(), id)
	if err != nil {
		slog.Error("failed to recalculate social insurance overview", "error", err, "id", id)
		mapSocialInsuranceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, socialInsuranceFromDomain(sio))
}

// GenerateXML handles POST /api/v1/social-insurance/{id}/generate-xml.
func (h *SocialInsuranceHandler) GenerateXML(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid social insurance overview ID")
		return
	}

	sio, err := h.svc.GenerateXML(r.Context(), id)
	if err != nil {
		slog.Error("failed to generate social insurance XML", "error", err, "id", id)
		mapSocialInsuranceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, socialInsuranceFromDomain(sio))
}

// DownloadXML handles GET /api/v1/social-insurance/{id}/xml.
func (h *SocialInsuranceHandler) DownloadXML(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid social insurance overview ID")
		return
	}

	xmlData, err := h.svc.GetXMLData(r.Context(), id)
	if err != nil {
		slog.Error("failed to get social insurance XML", "error", err, "id", id)
		mapSocialInsuranceError(w, err)
		return
	}

	if len(xmlData) == 0 {
		respondError(w, http.StatusNotFound, "XML not yet generated for this social insurance overview")
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	filename := fmt.Sprintf("cssz-prehled-%d.xml", id)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(xmlData)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(xmlData)
}

// MarkFiled handles POST /api/v1/social-insurance/{id}/mark-filed.
func (h *SocialInsuranceHandler) MarkFiled(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid social insurance overview ID")
		return
	}

	sio, err := h.svc.MarkFiled(r.Context(), id)
	if err != nil {
		slog.Error("failed to mark social insurance overview as filed", "error", err, "id", id)
		mapSocialInsuranceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, socialInsuranceFromDomain(sio))
}
