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

// VIESHandler handles HTTP requests for VIES summary management.
type VIESHandler struct {
	svc         *service.VIESSummaryService
	settingsSvc *service.SettingsService
}

// NewVIESHandler creates a new VIESHandler.
func NewVIESHandler(svc *service.VIESSummaryService, settingsSvc *service.SettingsService) *VIESHandler {
	return &VIESHandler{svc: svc, settingsSvc: settingsSvc}
}

// Routes registers VIES summary routes on the given router.
func (h *VIESHandler) Routes() chi.Router {
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

// Create handles POST /api/v1/vies-summaries.
func (h *VIESHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req viesSummaryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	vs := &domain.VIESSummary{
		Period: domain.TaxPeriod{
			Year:    req.Year,
			Quarter: req.Quarter,
		},
		FilingType: req.FilingType,
	}

	if err := h.svc.Create(r.Context(), vs); err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			respondError(w, http.StatusBadRequest, "invalid input")
			return
		}
		if errors.Is(err, domain.ErrDuplicateNumber) {
			respondError(w, http.StatusConflict, "VIES summary already exists for this period")
			return
		}
		slog.Error("failed to create VIES summary", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to create VIES summary")
		return
	}

	respondJSON(w, http.StatusCreated, viesSummaryFromDomain(vs, nil))
}

// List handles GET /api/v1/vies-summaries?year=YYYY.
func (h *VIESHandler) List(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		yearStr = strconv.Itoa(time.Now().Year())
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	summaries, err := h.svc.List(r.Context(), year)
	if err != nil {
		slog.Error("failed to list VIES summaries", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list VIES summaries")
		return
	}

	resp := make([]viesSummaryResponse, 0, len(summaries))
	for i := range summaries {
		resp = append(resp, viesSummaryFromDomain(&summaries[i], nil))
	}

	respondJSON(w, http.StatusOK, resp)
}

// GetByID handles GET /api/v1/vies-summaries/{id}.
func (h *VIESHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid ID")
		return
	}

	vs, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get VIES summary", "error", err)
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "VIES summary not found")
		} else {
			respondError(w, http.StatusInternalServerError, "failed to get VIES summary")
		}
		return
	}

	lines, err := h.svc.GetLines(r.Context(), id)
	if err != nil {
		slog.Error("failed to get VIES summary lines", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load lines")
		return
	}

	respondJSON(w, http.StatusOK, viesSummaryFromDomain(vs, lines))
}

// Delete handles DELETE /api/v1/vies-summaries/{id}.
func (h *VIESHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid ID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "VIES summary not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			respondError(w, http.StatusBadRequest, "cannot delete a filed VIES summary")
			return
		}
		slog.Error("failed to delete VIES summary", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to delete VIES summary")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Recalculate handles POST /api/v1/vies-summaries/{id}/recalculate.
func (h *VIESHandler) Recalculate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid ID")
		return
	}

	if err := h.svc.Recalculate(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "VIES summary not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			respondError(w, http.StatusBadRequest, "cannot recalculate a filed VIES summary")
			return
		}
		slog.Error("failed to recalculate VIES summary", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to recalculate VIES summary")
		return
	}

	vs, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get recalculated VIES summary", "error", err)
		respondError(w, http.StatusInternalServerError, "recalculation succeeded but failed to load result")
		return
	}

	lines, err := h.svc.GetLines(r.Context(), id)
	if err != nil {
		slog.Error("failed to get VIES summary lines", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load lines")
		return
	}

	respondJSON(w, http.StatusOK, viesSummaryFromDomain(vs, lines))
}

// GenerateXML handles POST /api/v1/vies-summaries/{id}/generate-xml.
func (h *VIESHandler) GenerateXML(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid ID")
		return
	}

	settings, err := h.settingsSvc.GetAll(r.Context())
	if err != nil {
		slog.Error("failed to load settings for XML generation", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load settings")
		return
	}
	dic, ok := settings["dic"]
	if !ok || dic == "" {
		respondError(w, http.StatusUnprocessableEntity, "DIC not configured in settings")
		return
	}

	if err := h.svc.GenerateXML(r.Context(), id, dic); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "VIES summary not found")
			return
		}
		slog.Error("failed to generate VIES XML", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to generate XML")
		return
	}

	vs, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get VIES summary after XML generation", "error", err)
		respondError(w, http.StatusInternalServerError, "XML generated but failed to load result")
		return
	}

	respondJSON(w, http.StatusOK, viesSummaryFromDomain(vs, nil))
}

// DownloadXML handles GET /api/v1/vies-summaries/{id}/xml.
func (h *VIESHandler) DownloadXML(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid ID")
		return
	}

	vs, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get VIES summary for XML download", "error", err)
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "VIES summary not found")
		} else {
			respondError(w, http.StatusInternalServerError, "failed to get VIES summary")
		}
		return
	}

	if len(vs.XMLData) == 0 {
		respondError(w, http.StatusNotFound, "XML not yet generated")
		return
	}

	filename := "vies_" + strconv.Itoa(vs.Period.Year) + "_Q" + strconv.Itoa(vs.Period.Quarter) + ".xml"
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	if _, writeErr := w.Write(vs.XMLData); writeErr != nil {
		slog.Error("failed to write XML response", "error", writeErr)
	}
}

// MarkFiled handles POST /api/v1/vies-summaries/{id}/mark-filed.
func (h *VIESHandler) MarkFiled(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid ID")
		return
	}

	if err := h.svc.MarkFiled(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "VIES summary not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			respondError(w, http.StatusBadRequest, "VIES summary already filed")
			return
		}
		slog.Error("failed to mark VIES summary as filed", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to mark as filed")
		return
	}

	vs, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get VIES summary after marking filed", "error", err)
		respondError(w, http.StatusInternalServerError, "marked as filed but failed to load result")
		return
	}

	respondJSON(w, http.StatusOK, viesSummaryFromDomain(vs, nil))
}
