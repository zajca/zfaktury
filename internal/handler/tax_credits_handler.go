package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// TaxCreditsHandler handles HTTP requests for tax credits management.
type TaxCreditsHandler struct {
	svc *service.TaxCreditsService
}

// NewTaxCreditsHandler creates a new TaxCreditsHandler.
func NewTaxCreditsHandler(svc *service.TaxCreditsService) *TaxCreditsHandler {
	return &TaxCreditsHandler{svc: svc}
}

// Routes registers tax credits routes on a chi router.
func (h *TaxCreditsHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{year}", h.GetSummary)
	r.Put("/{year}/spouse", h.UpsertSpouse)
	r.Delete("/{year}/spouse", h.DeleteSpouse)
	r.Get("/{year}/children", h.ListChildren)
	r.Post("/{year}/children", h.CreateChild)
	r.Put("/{year}/children/{id}", h.UpdateChild)
	r.Delete("/{year}/children/{id}", h.DeleteChild)
	r.Put("/{year}/personal", h.UpsertPersonal)
	r.Post("/{year}/copy-from/{sourceYear}", h.CopyFromYear)
	return r
}

// --- DTOs ---

type taxCreditsSummaryResponse struct {
	Year              int                          `json:"year"`
	Spouse            *taxSpouseCreditResponse     `json:"spouse"`
	Children          []taxChildCreditResponse     `json:"children"`
	Personal          *taxPersonalCreditsResponse  `json:"personal"`
	TotalCredits      int64                        `json:"total_credits"`
	TotalChildBenefit int64                        `json:"total_child_benefit"`
}

type taxSpouseCreditRequest struct {
	SpouseName        string `json:"spouse_name"`
	SpouseBirthNumber string `json:"spouse_birth_number"`
	SpouseIncome      int64  `json:"spouse_income"`
	SpouseZTP         bool   `json:"spouse_ztp"`
	MonthsClaimed     int    `json:"months_claimed"`
}

type taxSpouseCreditResponse struct {
	ID                int64  `json:"id"`
	Year              int    `json:"year"`
	SpouseName        string `json:"spouse_name"`
	SpouseBirthNumber string `json:"spouse_birth_number"`
	SpouseIncome      int64  `json:"spouse_income"`
	SpouseZTP         bool   `json:"spouse_ztp"`
	MonthsClaimed     int    `json:"months_claimed"`
	CreditAmount      int64  `json:"credit_amount"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

type taxChildCreditRequest struct {
	ChildName     string `json:"child_name"`
	BirthNumber   string `json:"birth_number"`
	ChildOrder    int    `json:"child_order"`
	MonthsClaimed int    `json:"months_claimed"`
	ZTP           bool   `json:"ztp"`
}

type taxChildCreditResponse struct {
	ID            int64  `json:"id"`
	Year          int    `json:"year"`
	ChildName     string `json:"child_name"`
	BirthNumber   string `json:"birth_number"`
	ChildOrder    int    `json:"child_order"`
	MonthsClaimed int    `json:"months_claimed"`
	ZTP           bool   `json:"ztp"`
	CreditAmount  int64  `json:"credit_amount"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type taxPersonalCreditsRequest struct {
	IsStudent       bool `json:"is_student"`
	StudentMonths   int  `json:"student_months"`
	DisabilityLevel int  `json:"disability_level"`
}

type taxPersonalCreditsResponse struct {
	Year             int    `json:"year"`
	IsStudent        bool   `json:"is_student"`
	StudentMonths    int    `json:"student_months"`
	DisabilityLevel  int    `json:"disability_level"`
	CreditStudent    int64  `json:"credit_student"`
	CreditDisability int64  `json:"credit_disability"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// --- Domain conversion helpers ---

func spouseCreditFromDomain(sc *domain.TaxSpouseCredit) taxSpouseCreditResponse {
	return taxSpouseCreditResponse{
		ID:                sc.ID,
		Year:              sc.Year,
		SpouseName:        sc.SpouseName,
		SpouseBirthNumber: sc.SpouseBirthNumber,
		SpouseIncome:      int64(sc.SpouseIncome),
		SpouseZTP:         sc.SpouseZTP,
		MonthsClaimed:     sc.MonthsClaimed,
		CreditAmount:      int64(sc.CreditAmount),
		CreatedAt:         sc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         sc.UpdatedAt.Format(time.RFC3339),
	}
}

func childCreditFromDomain(cc *domain.TaxChildCredit) taxChildCreditResponse {
	return taxChildCreditResponse{
		ID:            cc.ID,
		Year:          cc.Year,
		ChildName:     cc.ChildName,
		BirthNumber:   cc.BirthNumber,
		ChildOrder:    cc.ChildOrder,
		MonthsClaimed: cc.MonthsClaimed,
		ZTP:           cc.ZTP,
		CreditAmount:  int64(cc.CreditAmount),
		CreatedAt:     cc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     cc.UpdatedAt.Format(time.RFC3339),
	}
}

func personalCreditsFromDomain(pc *domain.TaxPersonalCredits) taxPersonalCreditsResponse {
	return taxPersonalCreditsResponse{
		Year:             pc.Year,
		IsStudent:        pc.IsStudent,
		StudentMonths:    pc.StudentMonths,
		DisabilityLevel:  pc.DisabilityLevel,
		CreditStudent:    int64(pc.CreditStudent),
		CreditDisability: int64(pc.CreditDisability),
		CreatedAt:        pc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        pc.UpdatedAt.Format(time.RFC3339),
	}
}

// --- Error mapping ---

func mapTaxCreditsError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		respondError(w, http.StatusNotFound, "tax credit not found")
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusBadRequest, err.Error())
	default:
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

// --- Handler methods ---

// GetSummary handles GET /{year}.
func (h *TaxCreditsHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	ctx := r.Context()

	// Load spouse (may be nil).
	spouse, err := h.svc.GetSpouse(ctx, year)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		slog.Error("failed to get spouse credit", "error", err, "year", year)
		mapTaxCreditsError(w, err)
		return
	}

	// Load children.
	children, err := h.svc.ListChildren(ctx, year)
	if err != nil {
		slog.Error("failed to list child credits", "error", err, "year", year)
		mapTaxCreditsError(w, err)
		return
	}

	// Load personal credits (may be nil).
	personal, err := h.svc.GetPersonal(ctx, year)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		slog.Error("failed to get personal credits", "error", err, "year", year)
		mapTaxCreditsError(w, err)
		return
	}

	// Compute totals.
	spouseCredit, disabilityCredit, studentCredit, credErr := h.svc.ComputeCredits(ctx, year)
	if credErr != nil {
		slog.Error("failed to compute credits", "error", credErr, "year", year)
		mapTaxCreditsError(w, credErr)
		return
	}
	totalCredits := spouseCredit + disabilityCredit + studentCredit

	totalChildBenefit, err := h.svc.ComputeChildBenefit(ctx, year)
	if err != nil {
		slog.Error("failed to compute child benefit", "error", err, "year", year)
		mapTaxCreditsError(w, err)
		return
	}

	// Build response.
	resp := taxCreditsSummaryResponse{
		Year:              year,
		TotalCredits:      int64(totalCredits),
		TotalChildBenefit: int64(totalChildBenefit),
	}

	if spouse != nil {
		sc := spouseCreditFromDomain(spouse)
		resp.Spouse = &sc
	}

	resp.Children = make([]taxChildCreditResponse, 0, len(children))
	for i := range children {
		resp.Children = append(resp.Children, childCreditFromDomain(&children[i]))
	}

	if personal != nil {
		pc := personalCreditsFromDomain(personal)
		resp.Personal = &pc
	}

	respondJSON(w, http.StatusOK, resp)
}

// UpsertSpouse handles PUT /{year}/spouse.
func (h *TaxCreditsHandler) UpsertSpouse(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	var req taxSpouseCreditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sc := &domain.TaxSpouseCredit{
		Year:              year,
		SpouseName:        req.SpouseName,
		SpouseBirthNumber: req.SpouseBirthNumber,
		SpouseIncome:      domain.Amount(req.SpouseIncome),
		SpouseZTP:         req.SpouseZTP,
		MonthsClaimed:     req.MonthsClaimed,
	}

	if err := h.svc.UpsertSpouse(r.Context(), sc); err != nil {
		slog.Error("failed to upsert spouse credit", "error", err, "year", year)
		mapTaxCreditsError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, spouseCreditFromDomain(sc))
}

// DeleteSpouse handles DELETE /{year}/spouse.
func (h *TaxCreditsHandler) DeleteSpouse(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	if err := h.svc.DeleteSpouse(r.Context(), year); err != nil {
		slog.Error("failed to delete spouse credit", "error", err, "year", year)
		mapTaxCreditsError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListChildren handles GET /{year}/children.
func (h *TaxCreditsHandler) ListChildren(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	children, err := h.svc.ListChildren(r.Context(), year)
	if err != nil {
		slog.Error("failed to list child credits", "error", err, "year", year)
		mapTaxCreditsError(w, err)
		return
	}

	items := make([]taxChildCreditResponse, 0, len(children))
	for i := range children {
		items = append(items, childCreditFromDomain(&children[i]))
	}

	respondJSON(w, http.StatusOK, items)
}

// CreateChild handles POST /{year}/children.
func (h *TaxCreditsHandler) CreateChild(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	var req taxChildCreditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cc := &domain.TaxChildCredit{
		Year:          year,
		ChildName:     req.ChildName,
		BirthNumber:   req.BirthNumber,
		ChildOrder:    req.ChildOrder,
		MonthsClaimed: req.MonthsClaimed,
		ZTP:           req.ZTP,
	}

	if err := h.svc.CreateChild(r.Context(), cc); err != nil {
		slog.Error("failed to create child credit", "error", err, "year", year)
		mapTaxCreditsError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, childCreditFromDomain(cc))
}

// UpdateChild handles PUT /{year}/children/{id}.
func (h *TaxCreditsHandler) UpdateChild(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid child credit ID")
		return
	}

	var req taxChildCreditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cc := &domain.TaxChildCredit{
		ID:            id,
		Year:          year,
		ChildName:     req.ChildName,
		BirthNumber:   req.BirthNumber,
		ChildOrder:    req.ChildOrder,
		MonthsClaimed: req.MonthsClaimed,
		ZTP:           req.ZTP,
	}

	if err := h.svc.UpdateChild(r.Context(), cc); err != nil {
		slog.Error("failed to update child credit", "error", err, "id", id, "year", year)
		mapTaxCreditsError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, childCreditFromDomain(cc))
}

// DeleteChild handles DELETE /{year}/children/{id}.
func (h *TaxCreditsHandler) DeleteChild(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid child credit ID")
		return
	}

	if err := h.svc.DeleteChild(r.Context(), id); err != nil {
		slog.Error("failed to delete child credit", "error", err, "id", id)
		mapTaxCreditsError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpsertPersonal handles PUT /{year}/personal.
func (h *TaxCreditsHandler) UpsertPersonal(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	var req taxPersonalCreditsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	pc := &domain.TaxPersonalCredits{
		Year:            year,
		IsStudent:       req.IsStudent,
		StudentMonths:   req.StudentMonths,
		DisabilityLevel: req.DisabilityLevel,
	}

	if err := h.svc.UpsertPersonal(r.Context(), pc); err != nil {
		slog.Error("failed to upsert personal credits", "error", err, "year", year)
		mapTaxCreditsError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, personalCreditsFromDomain(pc))
}

// CopyFromYear handles POST /{year}/copy-from/{sourceYear}.
func (h *TaxCreditsHandler) CopyFromYear(w http.ResponseWriter, r *http.Request) {
	targetYear, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid target year parameter")
		return
	}

	sourceYear, err := strconv.Atoi(chi.URLParam(r, "sourceYear"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid source year parameter")
		return
	}

	if err := h.svc.CopyFromYear(r.Context(), sourceYear, targetYear); err != nil {
		slog.Error("failed to copy tax credits", "error", err, "sourceYear", sourceYear, "targetYear", targetYear)
		mapTaxCreditsError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "tax credits copied successfully",
	})
}
