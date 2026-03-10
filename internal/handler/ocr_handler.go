package handler

import (
	"log/slog"
	"net/http"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// OCRHandler handles HTTP requests for OCR processing of documents.
type OCRHandler struct {
	svc *service.OCRService
}

// NewOCRHandler creates a new OCRHandler.
// OCR routes are registered directly in router.go (not via Routes()) because
// chi cannot mount two handlers on "/" in the same route group.
func NewOCRHandler(svc *service.OCRService) *OCRHandler {
	return &OCRHandler{svc: svc}
}

// ocrItemResponse is the JSON response for a single OCR-extracted line item.
type ocrItemResponse struct {
	Description    string `json:"description"`
	Quantity       int64  `json:"quantity"`
	UnitPrice      int64  `json:"unit_price"`
	VATRatePercent int    `json:"vat_rate_percent"`
	TotalAmount    int64  `json:"total_amount"`
}

// ocrResultResponse is the JSON response for OCR extraction results.
type ocrResultResponse struct {
	VendorName     string            `json:"vendor_name"`
	VendorICO      string            `json:"vendor_ico"`
	VendorDIC      string            `json:"vendor_dic"`
	InvoiceNumber  string            `json:"invoice_number"`
	IssueDate      string            `json:"issue_date"`
	DueDate        string            `json:"due_date"`
	TotalAmount    int64             `json:"total_amount"`
	VATAmount      int64             `json:"vat_amount"`
	VATRatePercent int               `json:"vat_rate_percent"`
	CurrencyCode   string            `json:"currency_code"`
	Description    string            `json:"description"`
	Items          []ocrItemResponse `json:"items"`
	Confidence     float64           `json:"confidence"`
}

// ProcessDocument handles POST /documents/{id}/ocr.
func (h *OCRHandler) ProcessDocument(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	result, err := h.svc.ProcessDocument(r.Context(), id)
	if err != nil {
		slog.Error("OCR processing failed", "error", err, "document_id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, ocrResultFromDomain(result))
}

// ocrResultFromDomain converts a domain.OCRResult to an ocrResultResponse.
func ocrResultFromDomain(r *domain.OCRResult) ocrResultResponse {
	resp := ocrResultResponse{
		VendorName:     r.VendorName,
		VendorICO:      r.VendorICO,
		VendorDIC:      r.VendorDIC,
		InvoiceNumber:  r.InvoiceNumber,
		IssueDate:      r.IssueDate,
		DueDate:        r.DueDate,
		TotalAmount:    int64(r.TotalAmount),
		VATAmount:      int64(r.VATAmount),
		VATRatePercent: r.VATRatePercent,
		CurrencyCode:   r.CurrencyCode,
		Description:    r.Description,
		Confidence:     r.Confidence,
	}

	resp.Items = make([]ocrItemResponse, 0, len(r.Items))
	for _, item := range r.Items {
		resp.Items = append(resp.Items, ocrItemResponse{
			Description:    item.Description,
			Quantity:       int64(item.Quantity),
			UnitPrice:      int64(item.UnitPrice),
			VATRatePercent: item.VATRatePercent,
			TotalAmount:    int64(item.TotalAmount),
		})
	}

	return resp
}
