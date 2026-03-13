package handler

import (
	"log/slog"
	"mime"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// InvoiceDocumentHandler handles HTTP requests for invoice document management.
type InvoiceDocumentHandler struct {
	svc *service.InvoiceDocumentService
}

// NewInvoiceDocumentHandler creates a new InvoiceDocumentHandler.
func NewInvoiceDocumentHandler(svc *service.InvoiceDocumentService) *InvoiceDocumentHandler {
	return &InvoiceDocumentHandler{svc: svc}
}

// Routes returns a router for standalone invoice document endpoints.
func (h *InvoiceDocumentHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/invoice-documents/{id}", h.GetByID)
	r.Get("/invoice-documents/{id}/download", h.Download)
	r.Delete("/invoice-documents/{id}", h.Delete)
	return r
}

// invoiceDocumentResponse is the JSON representation of an invoice document.
type invoiceDocumentResponse struct {
	ID          int64  `json:"id"`
	InvoiceID   int64  `json:"invoice_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	CreatedAt   string `json:"created_at"`
}

// ListByInvoice handles GET /invoices/{id}/documents.
func (h *InvoiceDocumentHandler) ListByInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	docs, err := h.svc.ListByInvoiceID(r.Context(), invoiceID)
	if err != nil {
		slog.Error("failed to list invoice documents", "error", err, "invoice_id", invoiceID)
		respondError(w, http.StatusInternalServerError, "failed to list documents")
		return
	}

	items := make([]invoiceDocumentResponse, 0, len(docs))
	for i := range docs {
		items = append(items, invoiceDocFromDomain(&docs[i]))
	}

	respondJSON(w, http.StatusOK, items)
}

// GetByID handles GET /invoice-documents/{id}.
func (h *InvoiceDocumentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	doc, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get invoice document", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "document not found")
		return
	}

	respondJSON(w, http.StatusOK, invoiceDocFromDomain(doc))
}

// Download handles GET /invoice-documents/{id}/download.
func (h *InvoiceDocumentHandler) Download(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	doc, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get invoice document", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "document not found")
		return
	}

	filePath := doc.StoragePath
	if _, err := os.Stat(filePath); err != nil {
		slog.Error("invoice document file missing from disk", "error", err, "path", filePath, "id", id)
		respondError(w, http.StatusNotFound, "document file not found")
		return
	}

	w.Header().Set("Content-Type", doc.ContentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": doc.Filename}))
	http.ServeFile(w, r, filePath)
}

// Delete handles DELETE /invoice-documents/{id}.
func (h *InvoiceDocumentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		slog.Error("failed to delete invoice document", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "document not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// invoiceDocFromDomain converts a domain.InvoiceDocument to an invoiceDocumentResponse.
func invoiceDocFromDomain(doc *domain.InvoiceDocument) invoiceDocumentResponse {
	return invoiceDocumentResponse{
		ID:          doc.ID,
		InvoiceID:   doc.InvoiceID,
		Filename:    doc.Filename,
		ContentType: doc.ContentType,
		Size:        doc.Size,
		CreatedAt:   doc.CreatedAt.Format(time.RFC3339),
	}
}
