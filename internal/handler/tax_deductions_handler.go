package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// TaxDeductionsHandler handles HTTP requests for tax deduction management.
type TaxDeductionsHandler struct {
	creditsSvc *service.TaxCreditsService
	docSvc     *service.TaxDeductionDocumentService
	extractSvc *service.TaxDocumentExtractionService // may be nil if OCR not configured
}

// NewTaxDeductionsHandler creates a new TaxDeductionsHandler.
func NewTaxDeductionsHandler(
	creditsSvc *service.TaxCreditsService,
	docSvc *service.TaxDeductionDocumentService,
	extractSvc *service.TaxDocumentExtractionService,
) *TaxDeductionsHandler {
	return &TaxDeductionsHandler{
		creditsSvc: creditsSvc,
		docSvc:     docSvc,
		extractSvc: extractSvc,
	}
}

// Routes returns a router for deduction CRUD and deduction-scoped document operations.
func (h *TaxDeductionsHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{year}", h.List)
	r.Post("/{year}", h.Create)
	r.Put("/{year}/{id}", h.Update)
	r.Delete("/{year}/{id}", h.Delete)
	r.Post("/{year}/{id}/documents", h.UploadDocument)
	r.Get("/{year}/{id}/documents", h.ListDocuments)
	return r
}

// DocumentRoutes returns routes for document-level operations (mounted separately).
func (h *TaxDeductionsHandler) DocumentRoutes() chi.Router {
	r := chi.NewRouter()
	r.Delete("/{id}", h.DeleteDocument)
	r.Get("/{id}/file", h.DownloadDocument)
	r.Post("/{id}/extract", h.ExtractDocument)
	return r
}

// --- DTOs ---

type taxDeductionRequest struct {
	Category      string `json:"category"`
	Description   string `json:"description"`
	ClaimedAmount int64  `json:"claimed_amount"` // halere
}

type taxDeductionResponse struct {
	ID            int64                     `json:"id"`
	Year          int                       `json:"year"`
	Category      string                    `json:"category"`
	Description   string                    `json:"description"`
	ClaimedAmount int64                     `json:"claimed_amount"`
	MaxAmount     int64                     `json:"max_amount"`
	AllowedAmount int64                     `json:"allowed_amount"`
	Documents     []taxDeductionDocResponse `json:"documents,omitempty"`
	CreatedAt     string                    `json:"created_at"`
	UpdatedAt     string                    `json:"updated_at"`
}

type taxDeductionDocResponse struct {
	ID              int64   `json:"id"`
	TaxDeductionID  int64   `json:"tax_deduction_id"`
	Filename        string  `json:"filename"`
	ContentType     string  `json:"content_type"`
	Size            int64   `json:"size"`
	ExtractedAmount int64   `json:"extracted_amount"`
	Confidence      float64 `json:"confidence"`
	CreatedAt       string  `json:"created_at"`
}

type taxExtractionResponse struct {
	AmountCZK  int     `json:"amount_czk"`
	Year       int     `json:"year"`
	Confidence float64 `json:"confidence"`
}

// --- Conversion helpers ---

func taxDeductionFromDomain(ded *domain.TaxDeduction, docs []domain.TaxDeductionDocument) taxDeductionResponse {
	resp := taxDeductionResponse{
		ID:            ded.ID,
		Year:          ded.Year,
		Category:      ded.Category,
		Description:   ded.Description,
		ClaimedAmount: int64(ded.ClaimedAmount),
		MaxAmount:     int64(ded.MaxAmount),
		AllowedAmount: int64(ded.AllowedAmount),
		CreatedAt:     ded.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     ded.UpdatedAt.Format(time.RFC3339),
	}
	if len(docs) > 0 {
		resp.Documents = make([]taxDeductionDocResponse, 0, len(docs))
		for i := range docs {
			resp.Documents = append(resp.Documents, taxDeductionDocFromDomain(&docs[i]))
		}
	}
	return resp
}

func taxDeductionDocFromDomain(doc *domain.TaxDeductionDocument) taxDeductionDocResponse {
	return taxDeductionDocResponse{
		ID:              doc.ID,
		TaxDeductionID:  doc.TaxDeductionID,
		Filename:        doc.Filename,
		ContentType:     doc.ContentType,
		Size:            doc.Size,
		ExtractedAmount: int64(doc.ExtractedAmount),
		Confidence:      doc.Confidence,
		CreatedAt:       doc.CreatedAt.Format(time.RFC3339),
	}
}

// --- Error mapping ---

func mapTaxDeductionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		respondError(w, http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrInvalidInput):
		respondError(w, http.StatusBadRequest, err.Error())
	default:
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

// --- Handler methods ---

// List handles GET /{year}.
func (h *TaxDeductionsHandler) List(w http.ResponseWriter, r *http.Request) {
	year, err := parseYear(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year")
		return
	}

	deductions, err := h.creditsSvc.ListDeductions(r.Context(), year)
	if err != nil {
		slog.Error("failed to list tax deductions", "error", err, "year", year)
		mapTaxDeductionError(w, err)
		return
	}

	results := make([]taxDeductionResponse, 0, len(deductions))
	for i := range deductions {
		docs, err := h.docSvc.ListByDeductionID(r.Context(), deductions[i].ID)
		if err != nil {
			slog.Error("failed to list documents for deduction", "error", err, "deduction_id", deductions[i].ID)
			mapTaxDeductionError(w, err)
			return
		}
		results = append(results, taxDeductionFromDomain(&deductions[i], docs))
	}

	respondJSON(w, http.StatusOK, results)
}

// Create handles POST /{year}.
func (h *TaxDeductionsHandler) Create(w http.ResponseWriter, r *http.Request) {
	year, err := parseYear(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year")
		return
	}

	var req taxDeductionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ded := &domain.TaxDeduction{
		Year:          year,
		Category:      req.Category,
		Description:   req.Description,
		ClaimedAmount: domain.Amount(req.ClaimedAmount),
	}

	if err := h.creditsSvc.CreateDeduction(r.Context(), ded); err != nil {
		slog.Error("failed to create tax deduction", "error", err, "year", year)
		mapTaxDeductionError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, taxDeductionFromDomain(ded, nil))
}

// Update handles PUT /{year}/{id}.
func (h *TaxDeductionsHandler) Update(w http.ResponseWriter, r *http.Request) {
	year, err := parseYear(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year")
		return
	}

	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid deduction ID")
		return
	}

	var req taxDeductionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ded := &domain.TaxDeduction{
		ID:            id,
		Year:          year,
		Category:      req.Category,
		Description:   req.Description,
		ClaimedAmount: domain.Amount(req.ClaimedAmount),
	}

	if err := h.creditsSvc.UpdateDeduction(r.Context(), ded); err != nil {
		slog.Error("failed to update tax deduction", "error", err, "id", id, "year", year)
		mapTaxDeductionError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, taxDeductionFromDomain(ded, nil))
}

// Delete handles DELETE /{year}/{id}.
func (h *TaxDeductionsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid deduction ID")
		return
	}

	if err := h.creditsSvc.DeleteDeduction(r.Context(), id); err != nil {
		slog.Error("failed to delete tax deduction", "error", err, "id", id)
		mapTaxDeductionError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UploadDocument handles POST /{year}/{id}/documents.
func (h *TaxDeductionsHandler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	deductionID, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid deduction ID")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 20<<20+512)
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

	doc, err := h.docSvc.Upload(r.Context(), deductionID, header.Filename, contentType, file)
	if err != nil {
		slog.Error("failed to upload tax deduction document", "error", err, "deduction_id", deductionID)
		mapTaxDeductionError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, taxDeductionDocFromDomain(doc))
}

// ListDocuments handles GET /{year}/{id}/documents.
func (h *TaxDeductionsHandler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	deductionID, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid deduction ID")
		return
	}

	docs, err := h.docSvc.ListByDeductionID(r.Context(), deductionID)
	if err != nil {
		slog.Error("failed to list tax deduction documents", "error", err, "deduction_id", deductionID)
		mapTaxDeductionError(w, err)
		return
	}

	items := make([]taxDeductionDocResponse, 0, len(docs))
	for i := range docs {
		items = append(items, taxDeductionDocFromDomain(&docs[i]))
	}

	respondJSON(w, http.StatusOK, items)
}

// DeleteDocument handles DELETE /{id} on DocumentRoutes.
func (h *TaxDeductionsHandler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	if err := h.docSvc.Delete(r.Context(), id); err != nil {
		slog.Error("failed to delete tax deduction document", "error", err, "id", id)
		mapTaxDeductionError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DownloadDocument handles GET /{id}/file on DocumentRoutes.
func (h *TaxDeductionsHandler) DownloadDocument(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	filePath, contentType, err := h.docSvc.GetFilePath(r.Context(), id)
	if err != nil {
		slog.Error("failed to get tax deduction document file path", "error", err, "id", id)
		mapTaxDeductionError(w, err)
		return
	}

	// Verify the file exists on disk before attempting to serve.
	if _, err := os.Stat(filePath); err != nil {
		slog.Error("tax deduction document file missing from disk", "error", err, "path", filePath, "id", id)
		respondError(w, http.StatusNotFound, "document file not found")
		return
	}

	// Get the document metadata for the filename.
	doc, err := h.docSvc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get tax deduction document metadata", "error", err, "id", id)
		mapTaxDeductionError(w, err)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
		"filename": filepath.Base(doc.Filename),
	}))
	http.ServeFile(w, r, filePath)
}

// ExtractDocument handles POST /{id}/extract on DocumentRoutes.
func (h *TaxDeductionsHandler) ExtractDocument(w http.ResponseWriter, r *http.Request) {
	if h.extractSvc == nil {
		respondError(w, http.StatusNotImplemented, "document extraction is not configured")
		return
	}

	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	result, err := h.extractSvc.ExtractAmount(r.Context(), id)
	if err != nil {
		slog.Error("failed to extract amount from tax deduction document", "error", err, "id", id)
		mapTaxDeductionError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, taxExtractionResponse{
		AmountCZK:  result.AmountCZK,
		Year:       result.Year,
		Confidence: result.Confidence,
	})
}
