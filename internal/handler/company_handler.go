package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/zajca/zfaktury/internal/domain"
)

// CompanyHandlerService is the surface CompanyHandler needs from the service layer.
// Declared as an interface so tests can stub it without spinning up a real DB.
type CompanyHandlerService interface {
	Create(ctx context.Context, c domain.Company) (int64, error)
	Get(ctx context.Context, id int64) (domain.Company, error)
	List(ctx context.Context) ([]domain.Company, error)
	Update(ctx context.Context, c domain.Company) error
	Delete(ctx context.Context, id int64) error
}

// CompanyHandler exposes the global-tier company CRUD endpoints.
// Mount at /api/v1/companies — these routes do NOT live behind the
// per-company middleware (companies are managed across the whole app).
type CompanyHandler struct {
	svc CompanyHandlerService
}

// NewCompanyHandler creates a new CompanyHandler.
func NewCompanyHandler(svc CompanyHandlerService) *CompanyHandler {
	return &CompanyHandler{svc: svc}
}

// Routes returns the chi sub-router for company management.
func (h *CompanyHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	return r
}

// List handles GET /api/v1/companies — returns all active companies.
func (h *CompanyHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		slog.Error("failed to list companies", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list companies")
		return
	}
	dtos := make([]CompanyDTO, 0, len(list))
	for _, c := range list {
		dtos = append(dtos, companyToDTO(c))
	}
	respondJSON(w, http.StatusOK, dtos)
}

// Create handles POST /api/v1/companies.
// Returns 201 Created with a Location header on success.
func (h *CompanyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto CompanyDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	id, err := h.svc.Create(r.Context(), dtoToCompany(dto))
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		slog.Error("failed to create company", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to create company")
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/api/v1/companies/%d", id))
	// Re-fetch so the response carries the canonical timestamps written by the repo.
	created, err := h.svc.Get(r.Context(), id)
	if err != nil {
		slog.Error("failed to fetch created company", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "failed to fetch created company")
		return
	}
	respondJSON(w, http.StatusCreated, companyToDTO(created))
}

// Get handles GET /api/v1/companies/{id}.
func (h *CompanyHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid company ID")
		return
	}
	c, err := h.svc.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "company not found")
			return
		}
		slog.Error("failed to get company", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "failed to get company")
		return
	}
	respondJSON(w, http.StatusOK, companyToDTO(c))
}

// Update handles PUT /api/v1/companies/{id}.
// Returns 204 No Content on success.
func (h *CompanyHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid company ID")
		return
	}
	var dto CompanyDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// Path ID is authoritative — never trust the body's id for a PUT.
	dto.ID = id
	if err := h.svc.Update(r.Context(), dtoToCompany(dto)); err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			respondError(w, http.StatusNotFound, "company not found")
		case errors.Is(err, domain.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			slog.Error("failed to update company", "error", err, "id", id)
			respondError(w, http.StatusInternalServerError, "failed to update company")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Delete handles DELETE /api/v1/companies/{id}.
// Returns 204 on success, 404 if missing, 409 when the company is the last
// remaining one (ErrLastCompany) or still has non-deleted children (ErrInUse).
func (h *CompanyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid company ID")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			respondError(w, http.StatusNotFound, "company not found")
		case errors.Is(err, domain.ErrLastCompany), errors.Is(err, domain.ErrInUse):
			respondError(w, http.StatusConflict, err.Error())
		default:
			slog.Error("failed to delete company", "error", err, "id", id)
			respondError(w, http.StatusInternalServerError, "failed to delete company")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
