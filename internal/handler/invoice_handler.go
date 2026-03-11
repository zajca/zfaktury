package handler

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/isdoc"
	"github.com/zajca/zfaktury/internal/pdf"
	"github.com/zajca/zfaktury/internal/service"
)

// InvoiceHandler handles HTTP requests for invoice management.
type InvoiceHandler struct {
	svc         *service.InvoiceService
	settingsSvc *service.SettingsService
	pdfGen      *pdf.InvoicePDFGenerator
	isdocGen    *isdoc.ISDOCGenerator
}

// NewInvoiceHandler creates a new InvoiceHandler.
func NewInvoiceHandler(svc *service.InvoiceService, settingsSvc *service.SettingsService, pdfGen *pdf.InvoicePDFGenerator, isdocGen *isdoc.ISDOCGenerator) *InvoiceHandler {
	return &InvoiceHandler{
		svc:         svc,
		settingsSvc: settingsSvc,
		pdfGen:      pdfGen,
		isdocGen:    isdocGen,
	}
}

// Routes registers invoice routes on the given router.
func (h *InvoiceHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/send", h.MarkAsSent)
	r.Post("/{id}/mark-paid", h.MarkAsPaid)
	r.Post("/{id}/duplicate", h.Duplicate)
	r.Post("/{id}/settle", h.SettleProforma)
	r.Post("/{id}/credit-note", h.CreateCreditNote)
	r.Get("/{id}/pdf", h.DownloadPDF)
	r.Get("/{id}/qr", h.QRPayment)
	r.Get("/{id}/isdoc", h.ExportISDOC)
	r.Post("/export/isdoc", h.ExportISDOCBatch)
	return r
}

// Create handles POST /api/v1/invoices.
func (h *InvoiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req invoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	invoice, err := req.toDomain()
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.svc.Create(r.Context(), invoice); err != nil {
		slog.Error("failed to create invoice", "error", err)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, invoiceFromDomain(invoice))
}

// List handles GET /api/v1/invoices.
func (h *InvoiceHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	filter := domain.InvoiceFilter{
		Status:     r.URL.Query().Get("status"),
		CustomerID: parseOptionalInt64(r, "customer_id"),
		DateFrom:   parseOptionalTime(r, "date_from"),
		DateTo:     parseOptionalTime(r, "date_to"),
		Search:     r.URL.Query().Get("search"),
		Limit:      limit,
		Offset:     offset,
	}

	invoices, total, err := h.svc.List(r.Context(), filter)
	if err != nil {
		slog.Error("failed to list invoices", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list invoices")
		return
	}

	items := make([]invoiceResponse, 0, len(invoices))
	for i := range invoices {
		items = append(items, invoiceFromDomain(&invoices[i]))
	}

	respondJSON(w, http.StatusOK, listResponse[invoiceResponse]{
		Data:   items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// GetByID handles GET /api/v1/invoices/{id}.
func (h *InvoiceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	invoice, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get invoice", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "invoice not found")
		return
	}

	respondJSON(w, http.StatusOK, invoiceFromDomain(invoice))
}

// Update handles PUT /api/v1/invoices/{id}.
func (h *InvoiceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	var req invoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	invoice, err := req.toDomain()
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	invoice.ID = id

	if err := h.svc.Update(r.Context(), invoice); err != nil {
		slog.Error("failed to update invoice", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	updated, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch updated invoice")
		return
	}

	respondJSON(w, http.StatusOK, invoiceFromDomain(updated))
}

// Delete handles DELETE /api/v1/invoices/{id}.
func (h *InvoiceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		slog.Error("failed to delete invoice", "error", err, "id", id)
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// MarkAsSent handles POST /api/v1/invoices/{id}/send.
func (h *InvoiceHandler) MarkAsSent(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	if err := h.svc.MarkAsSent(r.Context(), id); err != nil {
		slog.Error("failed to mark invoice as sent", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// Return the updated invoice.
	invoice, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch updated invoice")
		return
	}

	respondJSON(w, http.StatusOK, invoiceFromDomain(invoice))
}

// MarkAsPaid handles POST /api/v1/invoices/{id}/mark-paid.
func (h *InvoiceHandler) MarkAsPaid(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	var req markPaidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	paidAt := time.Now()
	if req.PaidAt != "" {
		parsed, err := time.Parse("2006-01-02", req.PaidAt)
		if err != nil {
			// Try RFC3339 as fallback.
			parsed, err = time.Parse(time.RFC3339, req.PaidAt)
			if err != nil {
				respondError(w, http.StatusBadRequest, "invalid paid_at date format")
				return
			}
		}
		paidAt = parsed
	}

	if err := h.svc.MarkAsPaid(r.Context(), id, domain.Amount(req.Amount), paidAt); err != nil {
		slog.Error("failed to mark invoice as paid", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// Return the updated invoice.
	invoice, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch updated invoice")
		return
	}

	respondJSON(w, http.StatusOK, invoiceFromDomain(invoice))
}

// Duplicate handles POST /api/v1/invoices/{id}/duplicate.
func (h *InvoiceHandler) Duplicate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	invoice, err := h.svc.Duplicate(r.Context(), id)
	if err != nil {
		slog.Error("failed to duplicate invoice", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, invoiceFromDomain(invoice))
}

// SettleProforma handles POST /api/v1/invoices/{id}/settle.
// Creates a regular settlement invoice from a paid proforma.
func (h *InvoiceHandler) SettleProforma(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	invoice, err := h.svc.SettleProforma(r.Context(), id)
	if err != nil {
		slog.Error("failed to settle proforma", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, invoiceFromDomain(invoice))
}

// creditNoteRequest is the JSON request body for creating a credit note.
type creditNoteRequest struct {
	Items  []invoiceItemRequest `json:"items"`
	Reason string               `json:"reason"`
}

// CreateCreditNote handles POST /api/v1/invoices/{id}/credit-note.
func (h *InvoiceHandler) CreateCreditNote(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	var req creditNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Convert request items to domain items.
	var items []domain.InvoiceItem
	for _, ri := range req.Items {
		items = append(items, domain.InvoiceItem{
			Description:    ri.Description,
			Quantity:       domain.Amount(ri.Quantity),
			Unit:           ri.Unit,
			UnitPrice:      domain.Amount(ri.UnitPrice),
			VATRatePercent: ri.VATRatePercent,
			SortOrder:      ri.SortOrder,
		})
	}

	invoice, err := h.svc.CreateCreditNote(r.Context(), id, items, req.Reason)
	if err != nil {
		slog.Error("failed to create credit note", "error", err, "id", id)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, invoiceFromDomain(invoice))
}

// DownloadPDF handles GET /api/v1/invoices/{id}/pdf.
func (h *InvoiceHandler) DownloadPDF(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	invoice, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get invoice for PDF", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "invoice not found")
		return
	}

	supplier, err := h.loadPDFSupplierInfo(r)
	if err != nil {
		slog.Error("failed to load supplier settings for PDF", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load supplier settings")
		return
	}

	pdfBytes, err := h.pdfGen.Generate(r.Context(), invoice, supplier)
	if err != nil {
		slog.Error("failed to generate PDF", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "failed to generate PDF")
		return
	}

	filename := fmt.Sprintf("faktura_%s.pdf", invoice.InvoiceNumber)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}

// QRPayment handles GET /api/v1/invoices/{id}/qr.
func (h *InvoiceHandler) QRPayment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	invoice, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get invoice for QR", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "invoice not found")
		return
	}

	// Determine IBAN and SWIFT: prefer invoice values, fall back to settings.
	iban := invoice.IBAN
	swift := invoice.SWIFT
	if iban == "" {
		supplier, err := h.loadPDFSupplierInfo(r)
		if err != nil {
			slog.Error("failed to load supplier settings for QR", "error", err)
			respondError(w, http.StatusInternalServerError, "failed to load supplier settings")
			return
		}
		iban = supplier.IBAN
		swift = supplier.SWIFT
	}

	if iban == "" {
		respondError(w, http.StatusUnprocessableEntity, "IBAN is required for QR payment generation")
		return
	}

	qrBytes, err := pdf.GenerateQRPayment(invoice, iban, swift)
	if err != nil {
		slog.Error("failed to generate QR payment", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "failed to generate QR payment code")
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(qrBytes)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(qrBytes)
}

// loadPDFSupplierInfo reads supplier information from application settings for PDF generation.
func (h *InvoiceHandler) loadPDFSupplierInfo(r *http.Request) (pdf.SupplierInfo, error) {
	settings, err := h.settingsSvc.GetAll(r.Context())
	if err != nil {
		return pdf.SupplierInfo{}, fmt.Errorf("loading settings: %w", err)
	}

	return pdf.SupplierInfo{
		Name:          settings[service.SettingCompanyName],
		ICO:           settings[service.SettingICO],
		DIC:           settings[service.SettingDIC],
		VATRegistered: settings[service.SettingVATRegistered] == "true",
		Street:        settings[service.SettingStreet],
		City:          settings[service.SettingCity],
		ZIP:           settings[service.SettingZIP],
		Email:         settings[service.SettingEmail],
		Phone:         settings[service.SettingPhone],
		BankAccount:   settings[service.SettingBankAccount],
		BankCode:      settings[service.SettingBankCode],
		IBAN:          settings[service.SettingIBAN],
		SWIFT:         settings[service.SettingSWIFT],
	}, nil
}

// loadSupplierInfo reads supplier information from application settings.
func (h *InvoiceHandler) loadSupplierInfo(r *http.Request) (isdoc.SupplierInfo, error) {
	settings, err := h.settingsSvc.GetAll(r.Context())
	if err != nil {
		return isdoc.SupplierInfo{}, fmt.Errorf("loading settings: %w", err)
	}

	return isdoc.SupplierInfo{
		CompanyName: settings[service.SettingCompanyName],
		ICO:         settings[service.SettingICO],
		DIC:         settings[service.SettingDIC],
		Street:      settings[service.SettingStreet],
		City:        settings[service.SettingCity],
		ZIP:         settings[service.SettingZIP],
		Email:       settings[service.SettingEmail],
		Phone:       settings[service.SettingPhone],
		BankAccount: settings[service.SettingBankAccount],
		BankCode:    settings[service.SettingBankCode],
		IBAN:        settings[service.SettingIBAN],
		SWIFT:       settings[service.SettingSWIFT],
	}, nil
}

// ExportISDOC handles GET /api/v1/invoices/{id}/isdoc.
func (h *InvoiceHandler) ExportISDOC(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid invoice ID")
		return
	}

	invoice, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get invoice for ISDOC export", "error", err, "id", id)
		respondError(w, http.StatusNotFound, "invoice not found")
		return
	}

	supplier, err := h.loadSupplierInfo(r)
	if err != nil {
		slog.Error("failed to load supplier info", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load supplier settings")
		return
	}

	xmlData, err := h.isdocGen.Generate(r.Context(), invoice, supplier)
	if err != nil {
		slog.Error("failed to generate ISDOC", "error", err, "id", id)
		respondError(w, http.StatusInternalServerError, "failed to generate ISDOC document")
		return
	}

	filename := fmt.Sprintf("%s.isdoc", invoice.InvoiceNumber)
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(xmlData)
}

// isdocBatchRequest is the JSON request body for batch ISDOC export.
type isdocBatchRequest struct {
	InvoiceIDs []int64 `json:"invoice_ids"`
}

// ExportISDOCBatch handles POST /api/v1/invoices/export/isdoc.
func (h *InvoiceHandler) ExportISDOCBatch(w http.ResponseWriter, r *http.Request) {
	var req isdocBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.InvoiceIDs) == 0 {
		respondError(w, http.StatusBadRequest, "invoice_ids is required")
		return
	}

	if len(req.InvoiceIDs) > 500 {
		respondError(w, http.StatusBadRequest, "maximum 500 invoices per batch export")
		return
	}

	supplier, err := h.loadSupplierInfo(r)
	if err != nil {
		slog.Error("failed to load supplier info", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load supplier settings")
		return
	}

	// Buffer the ZIP in memory so partial failures can be reported as errors.
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	for _, id := range req.InvoiceIDs {
		invoice, err := h.svc.GetByID(r.Context(), id)
		if err != nil {
			slog.Error("failed to get invoice for batch ISDOC export", "error", err, "id", id)
			continue
		}

		xmlData, err := h.isdocGen.Generate(r.Context(), invoice, supplier)
		if err != nil {
			slog.Error("failed to generate ISDOC for batch export", "error", err, "id", id)
			continue
		}

		// Sanitize filename to prevent zip slip attacks.
		safeNumber := filepath.Base(invoice.InvoiceNumber)
		filename := fmt.Sprintf("%s.isdoc", safeNumber)
		f, err := zipWriter.Create(filename)
		if err != nil {
			slog.Error("failed to create zip entry", "error", err, "filename", filename)
			continue
		}
		if _, err := f.Write(xmlData); err != nil {
			slog.Error("failed to write zip entry data", "error", err, "filename", filename)
			continue
		}
	}

	if err := zipWriter.Close(); err != nil {
		slog.Error("failed to finalize zip archive", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to create zip archive")
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="isdoc-export.zip"`)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
