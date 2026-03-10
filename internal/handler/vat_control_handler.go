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

// VATControlStatementHandler handles HTTP requests for VAT control statements.
type VATControlStatementHandler struct {
	svc         *service.VATControlStatementService
	settingsSvc *service.SettingsService
}

// NewVATControlStatementHandler creates a new VATControlStatementHandler.
func NewVATControlStatementHandler(svc *service.VATControlStatementService, settingsSvc *service.SettingsService) *VATControlStatementHandler {
	return &VATControlStatementHandler{svc: svc, settingsSvc: settingsSvc}
}

// Routes registers VAT control statement routes on the given router.
func (h *VATControlStatementHandler) Routes() chi.Router {
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

// Create handles POST /api/v1/vat-control-statements.
func (h *VATControlStatementHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req controlStatementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cs := &domain.VATControlStatement{
		Period: domain.TaxPeriod{
			Year:  req.Year,
			Month: req.Month,
		},
		FilingType: req.FilingType,
	}

	if err := h.svc.Create(r.Context(), cs); err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			respondError(w, http.StatusBadRequest, "invalid input")
			return
		}
		if errors.Is(err, domain.ErrDuplicateNumber) {
			respondError(w, http.StatusConflict, "control statement already exists for this period")
			return
		}
		slog.Error("failed to create control statement", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to create control statement")
		return
	}

	respondJSON(w, http.StatusCreated, controlStatementFromDomain(cs, nil))
}

// List handles GET /api/v1/vat-control-statements.
func (h *VATControlStatementHandler) List(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		yearStr = strconv.Itoa(time.Now().Year())
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	statements, err := h.svc.List(r.Context(), year)
	if err != nil {
		slog.Error("failed to list control statements", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list control statements")
		return
	}

	var resp []controlStatementResponse
	for _, cs := range statements {
		resp = append(resp, controlStatementFromDomain(&cs, nil))
	}

	respondJSON(w, http.StatusOK, resp)
}

// GetByID handles GET /api/v1/vat-control-statements/{id}.
func (h *VATControlStatementHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	cs, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "control statement not found")
			return
		}
		slog.Error("failed to get control statement", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "failed to get control statement")
		return
	}

	lines, err := h.svc.GetLines(r.Context(), id)
	if err != nil {
		slog.Error("failed to get control statement lines", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "failed to get control statement lines")
		return
	}

	respondJSON(w, http.StatusOK, controlStatementFromDomain(cs, lines))
}

// Delete handles DELETE /api/v1/vat-control-statements/{id}.
func (h *VATControlStatementHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "control statement not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			respondError(w, http.StatusBadRequest, "cannot delete a filed control statement")
			return
		}
		slog.Error("failed to delete control statement", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "failed to delete control statement")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Recalculate handles POST /api/v1/vat-control-statements/{id}/recalculate.
func (h *VATControlStatementHandler) Recalculate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.svc.Recalculate(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "control statement not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			respondError(w, http.StatusBadRequest, "cannot recalculate a filed control statement")
			return
		}
		slog.Error("failed to recalculate control statement", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "failed to recalculate control statement")
		return
	}

	cs, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get control statement after recalculate", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "recalculation succeeded but failed to fetch result")
		return
	}

	lines, err := h.svc.GetLines(r.Context(), id)
	if err != nil {
		slog.Error("failed to get lines after recalculate", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "recalculation succeeded but failed to fetch lines")
		return
	}

	respondJSON(w, http.StatusOK, controlStatementFromDomain(cs, lines))
}

// GenerateXML handles POST /api/v1/vat-control-statements/{id}/generate-xml.
func (h *VATControlStatementHandler) GenerateXML(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Get DIC from settings.
	dic := ""
	if h.settingsSvc != nil {
		settings, err := h.settingsSvc.GetAll(r.Context())
		if err == nil {
			dic = settings["dic"]
		}
	}
	if dic == "" {
		respondError(w, http.StatusBadRequest, "DIC not configured in settings")
		return
	}

	xmlData, err := h.svc.GenerateXML(r.Context(), id, dic)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "control statement not found")
			return
		}
		slog.Error("failed to generate XML", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "failed to generate XML")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"xml_size": len(xmlData),
		"message":  "XML generated successfully",
	})
}

// DownloadXML handles GET /api/v1/vat-control-statements/{id}/xml.
func (h *VATControlStatementHandler) DownloadXML(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	cs, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "control statement not found")
			return
		}
		slog.Error("failed to get control statement for XML download", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "failed to get control statement")
		return
	}

	if len(cs.XMLData) == 0 {
		respondError(w, http.StatusNotFound, "XML has not been generated yet")
		return
	}

	filename := "kh_" + strconv.Itoa(cs.Period.Year) + "_" + strconv.Itoa(cs.Period.Month) + ".xml"
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Length", strconv.Itoa(len(cs.XMLData)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(cs.XMLData)
}

// MarkFiled handles POST /api/v1/vat-control-statements/{id}/mark-filed.
func (h *VATControlStatementHandler) MarkFiled(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.svc.MarkFiled(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "control statement not found")
			return
		}
		slog.Error("failed to mark control statement as filed", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "failed to mark as filed")
		return
	}

	cs, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get control statement after mark filed", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "mark filed succeeded but failed to fetch result")
		return
	}

	respondJSON(w, http.StatusOK, controlStatementFromDomain(cs, nil))
}
