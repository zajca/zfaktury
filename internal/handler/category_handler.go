package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// CategoryHandler handles HTTP requests for expense category management.
type CategoryHandler struct {
	svc *service.CategoryService
}

// NewCategoryHandler creates a new CategoryHandler.
func NewCategoryHandler(svc *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

// Routes registers expense category routes on the given router.
func (h *CategoryHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	return r
}

// List handles GET /api/v1/expense-categories.
func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.svc.List(r.Context())
	if err != nil {
		slog.Error("failed to list expense categories", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list categories")
		return
	}

	items := make([]categoryResponse, 0, len(categories))
	for i := range categories {
		items = append(items, categoryFromDomain(&categories[i]))
	}

	respondJSON(w, http.StatusOK, items)
}

// Create handles POST /api/v1/expense-categories.
func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req categoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cat := req.toDomain()
	if err := h.svc.Create(r.Context(), cat); err != nil {
		slog.Error("failed to create expense category", "error", err)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, categoryFromDomain(cat))
}

// Update handles PUT /api/v1/expense-categories/{id}.
func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid category ID")
		return
	}

	var req categoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cat := req.toDomain()
	cat.ID = id
	if err := h.svc.Update(r.Context(), cat); err != nil {
		slog.Error("failed to update expense category", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, categoryFromDomain(cat))
}

// Delete handles DELETE /api/v1/expense-categories/{id}.
func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid category ID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		slog.Error("failed to delete expense category", "error", err, "id", id)
		if err.Error() == "default categories cannot be deleted" {
			respondError(w, http.StatusForbidden, err.Error())
			return
		}
		respondError(w, http.StatusNotFound, "category not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Category DTOs ---

// categoryRequest is the JSON request body for creating/updating an expense category.
type categoryRequest struct {
	Key       string `json:"key"`
	LabelCS   string `json:"label_cs"`
	LabelEN   string `json:"label_en"`
	Color     string `json:"color"`
	SortOrder int    `json:"sort_order"`
}

// toDomain converts a categoryRequest to a domain.ExpenseCategory.
func (r *categoryRequest) toDomain() *domain.ExpenseCategory {
	return &domain.ExpenseCategory{
		Key:       r.Key,
		LabelCS:   r.LabelCS,
		LabelEN:   r.LabelEN,
		Color:     r.Color,
		SortOrder: r.SortOrder,
	}
}

// categoryResponse is the JSON response for an expense category.
type categoryResponse struct {
	ID        int64  `json:"id"`
	Key       string `json:"key"`
	LabelCS   string `json:"label_cs"`
	LabelEN   string `json:"label_en"`
	Color     string `json:"color"`
	SortOrder int    `json:"sort_order"`
	IsDefault bool   `json:"is_default"`
	CreatedAt string `json:"created_at"`
}

// categoryFromDomain converts a domain.ExpenseCategory to a categoryResponse.
func categoryFromDomain(cat *domain.ExpenseCategory) categoryResponse {
	return categoryResponse{
		ID:        cat.ID,
		Key:       cat.Key,
		LabelCS:   cat.LabelCS,
		LabelEN:   cat.LabelEN,
		Color:     cat.Color,
		SortOrder: cat.SortOrder,
		IsDefault: cat.IsDefault,
		CreatedAt: cat.CreatedAt.Format(time.RFC3339),
	}
}
