package handler

import (
	"encoding/json"
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

// HealthInsuranceHandler handles HTTP requests for health insurance overview management.
type HealthInsuranceHandler struct {
	svc *service.HealthInsuranceService
}

// NewHealthInsuranceHandler creates a new HealthInsuranceHandler.
func NewHealthInsuranceHandler(svc *service.HealthInsuranceService) *HealthInsuranceHandler {
	return &HealthInsuranceHandler{svc: svc}
}

// Routes registers health insurance overview routes on the given router.
func (h *HealthInsuranceHandler) Routes() chi.Router {
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

// healthInsuranceRequest is the JSON request body for creating a health insurance overview.
type healthInsuranceRequest struct {
	Year       int    `json:"year"`
	FilingType string `json:"filing_type"`
}

// healthInsuranceResponse is the JSON response for a health insurance overview.
type healthInsuranceResponse struct {
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

// healthInsuranceFromDomain converts a domain.HealthInsuranceOverview to a healthInsuranceResponse.
func healthInsuranceFromDomain(hi *domain.HealthInsuranceOverview) healthInsuranceResponse {
	return healthInsuranceResponse{
		ID:                  hi.ID,
		Year:                hi.Year,
		FilingType:          hi.FilingType,
		TotalRevenue:        int64(hi.TotalRevenue),
		TotalExpenses:       int64(hi.TotalExpenses),
		TaxBase:             int64(hi.TaxBase),
		AssessmentBase:      int64(hi.AssessmentBase),
		MinAssessmentBase:   int64(hi.MinAssessmentBase),
		FinalAssessmentBase: int64(hi.FinalAssessmentBase),
		InsuranceRate:       hi.InsuranceRate,
		TotalInsurance:      int64(hi.TotalInsurance),
		Prepayments:         int64(hi.Prepayments),
		Difference:          int64(hi.Difference),
		NewMonthlyPrepay:    int64(hi.NewMonthlyPrepay),
		HasXML:              len(hi.XMLData) > 0,
		Status:              hi.Status,
		FiledAt:             formatOptionalTime(hi.FiledAt),
		CreatedAt:           hi.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           hi.UpdatedAt.Format(time.RFC3339),
	}
}

// Create handles POST /api/v1/health-insurance.
func (h *HealthInsuranceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req healthInsuranceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	hi := &domain.HealthInsuranceOverview{
		Year:       req.Year,
		FilingType: req.FilingType,
	}

	if err := h.svc.Create(r.Context(), hi); err != nil {
		slog.Error("failed to create health insurance overview", "error", err)
		mapDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, healthInsuranceFromDomain(hi))
}

// List handles GET /api/v1/health-insurance.
func (h *HealthInsuranceHandler) List(w http.ResponseWriter, r *http.Request) {
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
		slog.Error("failed to list health insurance overviews", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list health insurance overviews")
		return
	}

	items := make([]healthInsuranceResponse, 0, len(overviews))
	for i := range overviews {
		items = append(items, healthInsuranceFromDomain(&overviews[i]))
	}

	respondJSON(w, http.StatusOK, items)
}

// GetByID handles GET /api/v1/health-insurance/{id}.
func (h *HealthInsuranceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid health insurance overview ID")
		return
	}

	hi, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get health insurance overview", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, healthInsuranceFromDomain(hi))
}

// Delete handles DELETE /api/v1/health-insurance/{id}.
func (h *HealthInsuranceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid health insurance overview ID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		slog.Error("failed to delete health insurance overview", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Recalculate handles POST /api/v1/health-insurance/{id}/recalculate.
func (h *HealthInsuranceHandler) Recalculate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid health insurance overview ID")
		return
	}

	hi, err := h.svc.Recalculate(r.Context(), id)
	if err != nil {
		slog.Error("failed to recalculate health insurance overview", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, healthInsuranceFromDomain(hi))
}

// GenerateXML handles POST /api/v1/health-insurance/{id}/generate-xml.
func (h *HealthInsuranceHandler) GenerateXML(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid health insurance overview ID")
		return
	}

	hi, err := h.svc.GenerateXML(r.Context(), id)
	if err != nil {
		slog.Error("failed to generate health insurance overview XML", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, healthInsuranceFromDomain(hi))
}

// DownloadXML handles GET /api/v1/health-insurance/{id}/xml.
func (h *HealthInsuranceHandler) DownloadXML(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid health insurance overview ID")
		return
	}

	xmlData, err := h.svc.GetXMLData(r.Context(), id)
	if err != nil {
		slog.Error("failed to get health insurance overview XML", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}

	if len(xmlData) == 0 {
		respondError(w, http.StatusNotFound, "XML not yet generated for this health insurance overview")
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	filename := fmt.Sprintf("zp-prehled-%d.xml", id)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(xmlData)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(xmlData)
}

// MarkFiled handles POST /api/v1/health-insurance/{id}/mark-filed.
func (h *HealthInsuranceHandler) MarkFiled(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid health insurance overview ID")
		return
	}

	hi, err := h.svc.MarkFiled(r.Context(), id)
	if err != nil {
		slog.Error("failed to mark health insurance overview as filed", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, healthInsuranceFromDomain(hi))
}
