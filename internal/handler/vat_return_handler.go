package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// VATReturnHandler handles HTTP requests for VAT return management.
type VATReturnHandler struct {
	svc *service.VATReturnService
}

// NewVATReturnHandler creates a new VATReturnHandler.
func NewVATReturnHandler(svc *service.VATReturnService) *VATReturnHandler {
	return &VATReturnHandler{svc: svc}
}

// Routes registers VAT return routes on the given router.
func (h *VATReturnHandler) Routes() chi.Router {
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

// mapVATReturnError maps domain errors to HTTP status codes.
func mapVATReturnError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		respondError(w, http.StatusNotFound, "vat return not found")
	case errors.Is(err, domain.ErrFilingAlreadyExists):
		respondError(w, http.StatusConflict, "vat return already exists for this period")
	case errors.Is(err, domain.ErrFilingAlreadyFiled):
		respondError(w, http.StatusConflict, "vat return already filed")
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusBadRequest, err.Error())
	default:
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

// Create handles POST /api/v1/vat-returns.
func (h *VATReturnHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req vatReturnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	vr := &domain.VATReturn{
		Period: domain.TaxPeriod{
			Year:    req.Year,
			Month:   req.Month,
			Quarter: req.Quarter,
		},
		FilingType: req.FilingType,
	}

	if err := h.svc.Create(r.Context(), vr); err != nil {
		slog.Error("failed to create vat return", "error", err)
		mapVATReturnError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, vatReturnFromDomain(vr))
}

// List handles GET /api/v1/vat-returns.
func (h *VATReturnHandler) List(w http.ResponseWriter, r *http.Request) {
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
		slog.Error("failed to list vat returns", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list vat returns")
		return
	}

	items := make([]vatReturnResponse, 0, len(returns))
	for i := range returns {
		items = append(items, vatReturnFromDomain(&returns[i]))
	}

	respondJSON(w, http.StatusOK, items)
}

// GetByID handles GET /api/v1/vat-returns/{id}.
func (h *VATReturnHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid vat return ID")
		return
	}

	vr, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get vat return", "error", err, "id", id)
		mapVATReturnError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, vatReturnFromDomain(vr))
}

// Delete handles DELETE /api/v1/vat-returns/{id}.
func (h *VATReturnHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid vat return ID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		slog.Error("failed to delete vat return", "error", err, "id", id)
		mapVATReturnError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Recalculate handles POST /api/v1/vat-returns/{id}/recalculate.
func (h *VATReturnHandler) Recalculate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid vat return ID")
		return
	}

	vr, err := h.svc.Recalculate(r.Context(), id)
	if err != nil {
		slog.Error("failed to recalculate vat return", "error", err, "id", id)
		mapVATReturnError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, vatReturnFromDomain(vr))
}

// GenerateXML handles POST /api/v1/vat-returns/{id}/generate-xml.
func (h *VATReturnHandler) GenerateXML(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid vat return ID")
		return
	}

	vr, err := h.svc.GenerateXML(r.Context(), id)
	if err != nil {
		slog.Error("failed to generate vat return XML", "error", err, "id", id)
		mapVATReturnError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, vatReturnFromDomain(vr))
}

// DownloadXML handles GET /api/v1/vat-returns/{id}/xml.
func (h *VATReturnHandler) DownloadXML(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid vat return ID")
		return
	}

	xmlData, err := h.svc.GetXMLData(r.Context(), id)
	if err != nil {
		slog.Error("failed to get vat return XML", "error", err, "id", id)
		mapVATReturnError(w, err)
		return
	}

	if len(xmlData) == 0 {
		respondError(w, http.StatusNotFound, "XML not yet generated for this vat return")
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	filename := fmt.Sprintf("dph-priznani-%d.xml", id)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(xmlData)))
	w.WriteHeader(http.StatusOK)
	w.Write(xmlData)
}

// MarkFiled handles POST /api/v1/vat-returns/{id}/mark-filed.
func (h *VATReturnHandler) MarkFiled(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid vat return ID")
		return
	}

	vr, err := h.svc.MarkFiled(r.Context(), id)
	if err != nil {
		slog.Error("failed to mark vat return as filed", "error", err, "id", id)
		mapVATReturnError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, vatReturnFromDomain(vr))
}
