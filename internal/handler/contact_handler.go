package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// ContactHandler handles HTTP requests for contact management.
type ContactHandler struct {
	svc *service.ContactService
}

// NewContactHandler creates a new ContactHandler.
func NewContactHandler(svc *service.ContactService) *ContactHandler {
	return &ContactHandler{svc: svc}
}

// Routes registers contact routes on the given router.
func (h *ContactHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/ares/{ico}", h.LookupARES)
	return r
}

// Create handles POST /api/v1/contacts.
func (h *ContactHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req contactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	contact := req.toDomain()
	if err := h.svc.Create(r.Context(), contact); err != nil {
		slog.Error("failed to create contact", "error", err)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, contactFromDomain(contact))
}

// List handles GET /api/v1/contacts.
func (h *ContactHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	filter := domain.ContactFilter{
		Search:   r.URL.Query().Get("search"),
		Type:     r.URL.Query().Get("type"),
		Favorite: parseOptionalBool(r, "favorite"),
		Limit:    limit,
		Offset:   offset,
	}

	contacts, total, err := h.svc.List(r.Context(), filter)
	if err != nil {
		slog.Error("failed to list contacts", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list contacts")
		return
	}

	items := make([]contactResponse, 0, len(contacts))
	for i := range contacts {
		items = append(items, contactFromDomain(&contacts[i]))
	}

	respondJSON(w, http.StatusOK, listResponse[contactResponse]{
		Data:   items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// GetByID handles GET /api/v1/contacts/{id}.
func (h *ContactHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid contact ID")
		return
	}

	contact, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get contact", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "contact not found")
		return
	}

	respondJSON(w, http.StatusOK, contactFromDomain(contact))
}

// Update handles PUT /api/v1/contacts/{id}.
func (h *ContactHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid contact ID")
		return
	}

	var req contactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	contact := req.toDomain()
	contact.ID = id

	if err := h.svc.Update(r.Context(), contact); err != nil {
		slog.Error("failed to update contact", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, contactFromDomain(contact))
}

// Delete handles DELETE /api/v1/contacts/{id}.
func (h *ContactHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid contact ID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		slog.Error("failed to delete contact", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "contact not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// LookupARES handles GET /api/v1/contacts/ares/{ico}.
func (h *ContactHandler) LookupARES(w http.ResponseWriter, r *http.Request) {
	ico := chi.URLParam(r, "ico")
	if ico == "" {
		respondError(w, http.StatusBadRequest, "ICO is required")
		return
	}

	contact, err := h.svc.LookupARES(r.Context(), ico)
	if err != nil {
		slog.Error("failed to lookup ARES", "error", err, "ico", ico)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, contactFromDomain(contact))
}
