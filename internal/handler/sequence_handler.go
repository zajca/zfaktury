package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// SequenceHandler handles HTTP requests for invoice sequence management.
type SequenceHandler struct {
	svc *service.SequenceService
}

// NewSequenceHandler creates a new SequenceHandler.
func NewSequenceHandler(svc *service.SequenceService) *SequenceHandler {
	return &SequenceHandler{svc: svc}
}

// Routes registers invoice sequence routes on the given router.
func (h *SequenceHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	return r
}

// List handles GET /api/v1/invoice-sequences.
func (h *SequenceHandler) List(w http.ResponseWriter, r *http.Request) {
	sequences, err := h.svc.List(r.Context())
	if err != nil {
		slog.Error("failed to list invoice sequences", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list invoice sequences")
		return
	}

	items := make([]sequenceResponse, 0, len(sequences))
	for i := range sequences {
		items = append(items, sequenceFromDomain(&sequences[i]))
	}

	respondJSON(w, http.StatusOK, items)
}

// Create handles POST /api/v1/invoice-sequences.
func (h *SequenceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req sequenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	seq := req.toDomain()
	if err := h.svc.Create(r.Context(), seq); err != nil {
		slog.Error("failed to create invoice sequence", "error", err)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, sequenceFromDomain(seq))
}

// GetByID handles GET /api/v1/invoice-sequences/{id}.
func (h *SequenceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid sequence ID")
		return
	}

	seq, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get invoice sequence", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "invoice sequence not found")
		return
	}

	respondJSON(w, http.StatusOK, sequenceFromDomain(seq))
}

// Update handles PUT /api/v1/invoice-sequences/{id}.
func (h *SequenceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid sequence ID")
		return
	}

	var req sequenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	seq := req.toDomain()
	seq.ID = id

	if err := h.svc.Update(r.Context(), seq); err != nil {
		slog.Error("failed to update invoice sequence", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, sequenceFromDomain(seq))
}

// Delete handles DELETE /api/v1/invoice-sequences/{id}.
func (h *SequenceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid sequence ID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		slog.Error("failed to delete invoice sequence", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Sequence DTOs ---

// sequenceRequest is the JSON request body for creating/updating an invoice sequence.
type sequenceRequest struct {
	Prefix        string `json:"prefix"`
	NextNumber    int    `json:"next_number"`
	Year          int    `json:"year"`
	FormatPattern string `json:"format_pattern"`
}

// toDomain converts a sequenceRequest to a domain.InvoiceSequence.
func (r *sequenceRequest) toDomain() *domain.InvoiceSequence {
	return &domain.InvoiceSequence{
		Prefix:        r.Prefix,
		NextNumber:    r.NextNumber,
		Year:          r.Year,
		FormatPattern: r.FormatPattern,
	}
}

// sequenceResponse is the JSON response for an invoice sequence.
type sequenceResponse struct {
	ID            int64  `json:"id"`
	Prefix        string `json:"prefix"`
	NextNumber    int    `json:"next_number"`
	Year          int    `json:"year"`
	FormatPattern string `json:"format_pattern"`
	Preview       string `json:"preview"`
}

// sequenceFromDomain converts a domain.InvoiceSequence to a sequenceResponse.
func sequenceFromDomain(seq *domain.InvoiceSequence) sequenceResponse {
	return sequenceResponse{
		ID:            seq.ID,
		Prefix:        seq.Prefix,
		NextNumber:    seq.NextNumber,
		Year:          seq.Year,
		FormatPattern: seq.FormatPattern,
		Preview:       service.FormatPreview(seq),
	}
}
