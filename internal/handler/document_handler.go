package handler

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// DocumentHandler handles HTTP requests for expense document management.
type DocumentHandler struct {
	svc *service.DocumentService
}

// NewDocumentHandler creates a new DocumentHandler.
func NewDocumentHandler(svc *service.DocumentService) *DocumentHandler {
	return &DocumentHandler{svc: svc}
}

// Routes returns a router for document endpoints.
// The lead mounts expense sub-routes at /expenses/{id}/documents and
// standalone routes at /documents/{id}.
func (h *DocumentHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/expenses/{id}/documents", h.Upload)
	r.Get("/expenses/{id}/documents", h.ListByExpense)
	r.Get("/documents/{id}", h.GetByID)
	r.Get("/documents/{id}/download", h.Download)
	r.Delete("/documents/{id}", h.Delete)
	return r
}

// documentResponse is the JSON representation of an expense document.
type documentResponse struct {
	ID          int64  `json:"id"`
	ExpenseID   int64  `json:"expense_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	CreatedAt   string `json:"created_at"`
}

// Upload handles POST /expenses/{id}/documents.
// Expects a multipart form with a "file" field.
func (h *DocumentHandler) Upload(w http.ResponseWriter, r *http.Request) {
	expenseID, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid expense ID")
		return
	}

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "missing file field in form")
		return
	}
	defer func() { _ = file.Close() }()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	doc, err := h.svc.Upload(r.Context(), expenseID, header.Filename, contentType, file)
	if err != nil {
		slog.Error("failed to upload document", "error", err, "expense_id", expenseID)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, documentFromDomain(doc))
}

// ListByExpense handles GET /expenses/{id}/documents.
func (h *DocumentHandler) ListByExpense(w http.ResponseWriter, r *http.Request) {
	expenseID, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid expense ID")
		return
	}

	docs, err := h.svc.ListByExpenseID(r.Context(), expenseID)
	if err != nil {
		slog.Error("failed to list documents", "error", err, "expense_id", expenseID)
		respondError(w, http.StatusInternalServerError, "failed to list documents")
		return
	}

	items := make([]documentResponse, 0, len(docs))
	for i := range docs {
		items = append(items, documentFromDomain(&docs[i]))
	}

	respondJSON(w, http.StatusOK, items)
}

// GetByID handles GET /documents/{id}.
func (h *DocumentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	doc, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get document", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "document not found")
		return
	}

	respondJSON(w, http.StatusOK, documentFromDomain(doc))
}

// Download handles GET /documents/{id}/download.
// Serves the file with appropriate Content-Type and Content-Disposition headers.
func (h *DocumentHandler) Download(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	doc, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get document", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "document not found")
		return
	}

	filePath := doc.StoragePath

	// Verify the file exists on disk before attempting to serve.
	if _, err := os.Stat(filePath); err != nil {
		slog.Error("document file missing from disk", "error", err, "path", filePath, "id", id)
		respondError(w, http.StatusNotFound, "document file not found")
		return
	}

	// Sanitize filename for Content-Disposition header: strip quotes and control chars.
	safeFilename := strings.Map(func(r rune) rune {
		if r == '"' || r == '\\' || r < 32 {
			return '_'
		}
		return r
	}, doc.Filename)

	w.Header().Set("Content-Type", doc.ContentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+safeFilename+`"`)
	http.ServeFile(w, r, filePath)
}

// Delete handles DELETE /documents/{id}.
func (h *DocumentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		slog.Error("failed to delete document", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "document not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// documentFromDomain converts a domain.ExpenseDocument to a documentResponse.
func documentFromDomain(doc *domain.ExpenseDocument) documentResponse {
	_ = doc.StoragePath // StoragePath is intentionally not exposed in the response.
	return documentResponse{
		ID:          doc.ID,
		ExpenseID:   doc.ExpenseID,
		Filename:    doc.Filename,
		ContentType: doc.ContentType,
		Size:        doc.Size,
		CreatedAt:   doc.CreatedAt.Format(time.RFC3339),
	}
}
