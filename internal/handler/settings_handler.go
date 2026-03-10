package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/service"
)

// SettingsHandler handles HTTP requests for application settings.
type SettingsHandler struct {
	svc *service.SettingsService
}

// NewSettingsHandler creates a new SettingsHandler.
func NewSettingsHandler(svc *service.SettingsService) *SettingsHandler {
	return &SettingsHandler{svc: svc}
}

// Routes registers settings routes on the given router.
func (h *SettingsHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.GetAll)
	r.Put("/", h.Update)
	return r
}

// GetAll handles GET /api/v1/settings.
func (h *SettingsHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	settings, err := h.svc.GetAll(r.Context())
	if err != nil {
		slog.Error("failed to get settings", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get settings")
		return
	}

	respondJSON(w, http.StatusOK, settings)
}

// Update handles PUT /api/v1/settings.
func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.SetBulk(r.Context(), req); err != nil {
		slog.Error("failed to update settings", "error", err)
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Return all settings after update.
	settings, err := h.svc.GetAll(r.Context())
	if err != nil {
		slog.Error("failed to get settings after update", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get settings")
		return
	}

	respondJSON(w, http.StatusOK, settings)
}
