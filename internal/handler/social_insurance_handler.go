package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/filename"
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
	r.Put("/{id}/activity-months", h.SetActivityMonths)
	r.Put("/{id}/secondary-activity", h.SetSecondaryActivity)
	r.Put("/{id}/employment-base", h.SetEmploymentBase)
	r.Post("/{id}/generate-xml", h.GenerateXML)
	r.Get("/{id}/xml", h.DownloadXML)
	r.Post("/{id}/mark-filed", h.MarkFiled)
	return r
}

// --- DTOs ---

// socialInsuranceRequest is the JSON request body for creating a social insurance overview.
type socialInsuranceRequest struct {
	Year           int    `json:"year"`
	FilingType     string `json:"filing_type"`
	ActivityMonths []int  `json:"activity_months"`
}

// socialInsuranceActivityMonthsRequest is the JSON request body for updating the months of activity.
type socialInsuranceActivityMonthsRequest struct {
	ActivityMonths []int `json:"activity_months"`
}

// socialInsuranceSecondaryActivityRequest is the JSON request body for the months
// of secondary activity; empty months with an empty reason clear them.
type socialInsuranceSecondaryActivityRequest struct {
	Months []int  `json:"months"`
	Reason string `json:"reason"`
}

// socialInsuranceEmploymentBaseRequest is the JSON request body for the
// employment assessment base in halere; null removes it.
type socialInsuranceEmploymentBaseRequest struct {
	EmploymentAssessmentBase *int64 `json:"employment_assessment_base"`
}

// socialInsuranceResponse is the JSON response for a social insurance overview.
type socialInsuranceResponse struct {
	ID                  int64              `json:"id"`
	Year                int                `json:"year"`
	FilingType          string             `json:"filing_type"`
	ActivityMonths      []int              `json:"activity_months"`
	SecondaryMonths     []int              `json:"secondary_months"`
	SecondaryReason     string             `json:"secondary_reason"`
	EmploymentBase      *int64             `json:"employment_assessment_base"`
	ReportedEmployment  *int64             `json:"reported_employment_base"`
	MainTaxBaseShare    int64              `json:"main_tax_base_share"`
	SecondaryTaxShare   int64              `json:"secondary_tax_base_share"`
	SecondaryAssessment int64              `json:"secondary_assessment_base"`
	MainPartialBase     int64              `json:"main_partial_base"`
	SecondaryPartial    int64              `json:"secondary_partial_base"`
	SecondaryDecisive   int64              `json:"secondary_decisive_amount"`
	SumBase             int64              `json:"sum_base"`
	InsuredBase         int64              `json:"insured_base"`
	TotalRevenue        int64              `json:"total_revenue"`
	TotalExpenses       int64              `json:"total_expenses"`
	TaxBase             int64              `json:"tax_base"`
	AssessmentBase      int64              `json:"assessment_base"`
	MinAssessmentBase   int64              `json:"min_assessment_base"`
	FinalAssessmentBase int64              `json:"final_assessment_base"`
	InsuranceRate       int                `json:"insurance_rate"`
	TotalInsurance      int64              `json:"total_insurance"`
	Prepayments         int64              `json:"prepayments"`
	Difference          int64              `json:"difference"`
	NewMonthlyPrepay    int64              `json:"new_monthly_prepay"`
	NewMonthlyBase      *int64             `json:"new_monthly_base"`
	FilingMethod        string             `json:"filing_method"`
	AdvanceYear         int                `json:"advance_year"`
	AdvanceStartMonth   int                `json:"advance_start_month"`
	AdvancePeriods      []advancePeriodDTO `json:"advance_periods"`
	Warnings            []string           `json:"warnings"`
	HasXML              bool               `json:"has_xml"`
	Status              string             `json:"status"`
	FiledAt             *string            `json:"filed_at,omitempty"`
	CreatedAt           string             `json:"created_at"`
	UpdatedAt           string             `json:"updated_at"`
}

// advancePeriodDTO is the JSON form of a domain.AdvancePeriod.
type advancePeriodDTO struct {
	FromMonth     int   `json:"from_month"`
	MonthlyBase   int64 `json:"monthly_base"`
	MonthlyPrepay int64 `json:"monthly_prepay"`
}

// advancePeriodsToDTO converts periods to DTOs; no periods give an empty slice.
func advancePeriodsToDTO(periods []domain.AdvancePeriod) []advancePeriodDTO {
	dtos := make([]advancePeriodDTO, 0, len(periods))
	for _, p := range periods {
		dtos = append(dtos, advancePeriodDTO{
			FromMonth:     p.FromMonth,
			MonthlyBase:   int64(p.MonthlyBase),
			MonthlyPrepay: int64(p.MonthlyPrepay),
		})
	}
	return dtos
}

// socialInsuranceFromDomain converts a domain.SocialInsuranceOverview to a socialInsuranceResponse.
func socialInsuranceFromDomain(sio *domain.SocialInsuranceOverview) socialInsuranceResponse {
	var newMonthlyBase *int64
	if sio.NewMonthlyBase != nil {
		base := int64(*sio.NewMonthlyBase)
		newMonthlyBase = &base
	}
	warnings := sio.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	return socialInsuranceResponse{
		SecondaryMonths:     sio.SecondaryMonths.Months(),
		SecondaryReason:     string(sio.SecondaryReason),
		EmploymentBase:      optionalAmountToInt64(sio.EmploymentAssessmentBase),
		ReportedEmployment:  optionalAmountToInt64(sio.ReportedEmploymentBase),
		MainTaxBaseShare:    int64(sio.MainTaxBaseShare),
		SecondaryTaxShare:   int64(sio.SecondaryTaxBaseShare),
		SecondaryAssessment: int64(sio.SecondaryAssessmentBase),
		MainPartialBase:     int64(sio.MainPartialBase),
		SecondaryPartial:    int64(sio.SecondaryPartialBase),
		SecondaryDecisive:   int64(sio.SecondaryDecisiveAmount),
		SumBase:             int64(sio.SumBase),
		InsuredBase:         int64(sio.InsuredBase),
		NewMonthlyBase:      newMonthlyBase,
		FilingMethod:        string(sio.FilingMethod),
		AdvanceYear:         sio.Year + 1,
		AdvanceStartMonth:   sio.AdvanceStartMonth,
		AdvancePeriods:      advancePeriodsToDTO(sio.AdvancePeriods),
		Warnings:            warnings,
		ID:                  sio.ID,
		Year:                sio.Year,
		FilingType:          sio.FilingType,
		ActivityMonths:      sio.ActivityMonths.Months(),
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

	months, err := domain.NewActivityMonths(req.ActivityMonths)
	if err != nil {
		mapDomainError(w, err)
		return
	}
	sio.ActivityMonths = months

	if err := h.svc.Create(r.Context(), sio); err != nil {
		slog.Error("failed to create social insurance overview", "error", err)
		mapDomainError(w, err)
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
		mapDomainError(w, err)
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
		mapDomainError(w, err)
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
		mapDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, socialInsuranceFromDomain(sio))
}

// SetActivityMonths handles PUT /api/v1/social-insurance/{id}/activity-months.
func (h *SocialInsuranceHandler) SetActivityMonths(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid social insurance overview ID")
		return
	}

	var req socialInsuranceActivityMonthsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	months, err := domain.NewActivityMonths(req.ActivityMonths)
	if err != nil {
		mapDomainError(w, err)
		return
	}

	sio, err := h.svc.SetActivityMonths(r.Context(), id, months)
	if err != nil {
		slog.Error("failed to set social insurance activity months", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, socialInsuranceFromDomain(sio))
}

// optionalAmountToInt64 converts an optional amount into a JSON-friendly pointer.
func optionalAmountToInt64(amount *domain.Amount) *int64 {
	if amount == nil {
		return nil
	}
	value := int64(*amount)
	return &value
}

// SetSecondaryActivity handles PUT /api/v1/social-insurance/{id}/secondary-activity.
func (h *SocialInsuranceHandler) SetSecondaryActivity(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid social insurance overview ID")
		return
	}

	var req socialInsuranceSecondaryActivityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var months domain.ActivityMonths
	if len(req.Months) > 0 {
		months, err = domain.NewActivityMonths(req.Months)
		if err != nil {
			mapDomainError(w, err)
			return
		}
	}

	sio, err := h.svc.SetSecondaryActivity(r.Context(), id, months, domain.SecondaryActivityReason(req.Reason))
	if err != nil {
		slog.Error("failed to set social insurance secondary activity", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, socialInsuranceFromDomain(sio))
}

// SetEmploymentBase handles PUT /api/v1/social-insurance/{id}/employment-base.
func (h *SocialInsuranceHandler) SetEmploymentBase(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid social insurance overview ID")
		return
	}

	var req socialInsuranceEmploymentBaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var base *domain.Amount
	if req.EmploymentAssessmentBase != nil {
		amount := domain.Amount(*req.EmploymentAssessmentBase)
		base = &amount
	}

	sio, err := h.svc.SetEmploymentBase(r.Context(), id, base)
	if err != nil {
		slog.Error("failed to set social insurance employment base", "error", err, "id", id)
		mapDomainError(w, err)
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
		mapDomainError(w, err)
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

	sio, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get social insurance XML", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}

	if len(sio.XMLData) == 0 {
		respondError(w, http.StatusNotFound, "XML not yet generated for this social insurance overview")
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", filename.ContentDisposition(filename.SocialInsurance(sio.Year, sio.FilingType)))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(sio.XMLData)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(sio.XMLData)
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
		mapDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, socialInsuranceFromDomain(sio))
}
