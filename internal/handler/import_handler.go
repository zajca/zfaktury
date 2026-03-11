package handler

import (
	"log/slog"
	"net/http"

	"github.com/zajca/zfaktury/internal/service"
)

// ImportHandler handles HTTP requests for expense import from documents.
type ImportHandler struct {
	svc *service.ImportService
}

// NewImportHandler creates a new ImportHandler.
func NewImportHandler(svc *service.ImportService) *ImportHandler {
	return &ImportHandler{svc: svc}
}

// importResponse is the JSON response for an expense import operation.
type importResponse struct {
	Expense  expenseResponse   `json:"expense"`
	Document documentResponse  `json:"document"`
	OCR      *ocrResultResponse `json:"ocr,omitempty"`
}

// Import handles POST /api/v1/expenses/import.
// Expects a multipart form with a "file" field.
func (h *ImportHandler) Import(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.svc.ImportFromDocument(r.Context(), header.Filename, contentType, file)
	if err != nil {
		slog.Error("expense import failed", "error", err)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	resp := importResponse{
		Expense:  expenseFromDomain(&result.Expense),
		Document: documentFromDomain(&result.Document),
	}
	if result.OCR != nil {
		ocr := ocrResultFromDomain(result.OCR)
		resp.OCR = &ocr
	}

	respondJSON(w, http.StatusCreated, resp)
}
