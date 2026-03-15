package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/pdf"
	"github.com/zajca/zfaktury/internal/service"
)

// PDFSettingsHandler handles HTTP requests for PDF template settings.
type PDFSettingsHandler struct {
	settingsSvc *service.SettingsService
	invoiceSvc  *service.InvoiceService
	pdfGen      *pdf.InvoicePDFGenerator
	dataDir     string
}

// NewPDFSettingsHandler creates a new PDFSettingsHandler.
func NewPDFSettingsHandler(
	settingsSvc *service.SettingsService,
	invoiceSvc *service.InvoiceService,
	pdfGen *pdf.InvoicePDFGenerator,
	dataDir string,
) *PDFSettingsHandler {
	return &PDFSettingsHandler{
		settingsSvc: settingsSvc,
		invoiceSvc:  invoiceSvc,
		pdfGen:      pdfGen,
		dataDir:     dataDir,
	}
}

// pdfSettingsResponse is the JSON response for PDF settings.
type pdfSettingsResponse struct {
	LogoPath        string `json:"logo_path"`
	AccentColor     string `json:"accent_color"`
	FooterText      string `json:"footer_text"`
	ShowQR          bool   `json:"show_qr"`
	ShowBankDetails bool   `json:"show_bank_details"`
	FontSize        string `json:"font_size"`
	HasLogo         bool   `json:"has_logo"`
}

// pdfSettingsRequest is the JSON request body for updating PDF settings.
type pdfSettingsRequest struct {
	AccentColor     string `json:"accent_color"`
	FooterText      string `json:"footer_text"`
	ShowQR          bool   `json:"show_qr"`
	ShowBankDetails bool   `json:"show_bank_details"`
	FontSize        string `json:"font_size"`
}

// GetPDFSettings handles GET /api/v1/settings/pdf.
func (h *PDFSettingsHandler) GetPDFSettings(w http.ResponseWriter, r *http.Request) {
	ps, err := h.settingsSvc.GetPDFSettings(r.Context())
	if err != nil {
		slog.Error("failed to get PDF settings", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get PDF settings")
		return
	}

	hasLogo := false
	if ps.LogoPath != "" {
		if _, err := os.Stat(ps.LogoPath); err == nil {
			hasLogo = true
		}
	}

	respondJSON(w, http.StatusOK, pdfSettingsResponse{
		LogoPath:        ps.LogoPath,
		AccentColor:     ps.AccentColor,
		FooterText:      ps.FooterText,
		ShowQR:          ps.ShowQR,
		ShowBankDetails: ps.ShowBankDetails,
		FontSize:        ps.FontSize,
		HasLogo:         hasLogo,
	})
}

// UpdatePDFSettings handles PUT /api/v1/settings/pdf.
func (h *PDFSettingsHandler) UpdatePDFSettings(w http.ResponseWriter, r *http.Request) {
	var req pdfSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate font size.
	switch req.FontSize {
	case "small", "normal", "large":
	default:
		respondError(w, http.StatusBadRequest, "font_size must be one of: small, normal, large")
		return
	}

	// Validate accent color (basic hex check).
	color := strings.TrimPrefix(req.AccentColor, "#")
	if len(color) != 6 {
		respondError(w, http.StatusBadRequest, "accent_color must be a 6-digit hex color (e.g. #2563eb)")
		return
	}

	// Preserve existing logo_path (not settable via this endpoint).
	existing, err := h.settingsSvc.GetPDFSettings(r.Context())
	if err != nil {
		slog.Error("failed to get existing PDF settings", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load existing settings")
		return
	}

	ps := service.PDFSettings{
		LogoPath:        existing.LogoPath,
		AccentColor:     req.AccentColor,
		FooterText:      req.FooterText,
		ShowQR:          req.ShowQR,
		ShowBankDetails: req.ShowBankDetails,
		FontSize:        req.FontSize,
	}

	if err := h.settingsSvc.SavePDFSettings(r.Context(), ps); err != nil {
		slog.Error("failed to save PDF settings", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to save PDF settings")
		return
	}

	hasLogo := false
	if ps.LogoPath != "" {
		if _, err := os.Stat(ps.LogoPath); err == nil {
			hasLogo = true
		}
	}

	respondJSON(w, http.StatusOK, pdfSettingsResponse{
		LogoPath:        ps.LogoPath,
		AccentColor:     ps.AccentColor,
		FooterText:      ps.FooterText,
		ShowQR:          ps.ShowQR,
		ShowBankDetails: ps.ShowBankDetails,
		FontSize:        ps.FontSize,
		HasLogo:         hasLogo,
	})
}

// allowedLogoTypes maps Content-Type to file extension.
var allowedLogoTypes = map[string]string{
	"image/png":     ".png",
	"image/jpeg":    ".jpg",
	"image/svg+xml": ".svg",
}

// maxLogoSize is the maximum logo file size (2MB).
const maxLogoSize = 2 << 20

// UploadLogo handles POST /api/v1/settings/logo.
func (h *PDFSettingsHandler) UploadLogo(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxLogoSize); err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse multipart form or file exceeds 2MB limit")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "missing file field in form")
		return
	}
	defer func() { _ = file.Close() }()

	if header.Size > maxLogoSize {
		respondError(w, http.StatusBadRequest, "file exceeds 2MB size limit")
		return
	}

	contentType := header.Header.Get("Content-Type")
	ext, ok := allowedLogoTypes[contentType]
	if !ok {
		respondError(w, http.StatusBadRequest, "unsupported file type, accepted: PNG, JPEG, SVG")
		return
	}

	// Save logo to {DataDir}/documents/logo.{ext}
	docsDir := filepath.Join(h.dataDir, "documents")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		slog.Error("failed to create documents directory", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to save logo")
		return
	}

	// Remove existing logo files first.
	h.removeLogoFiles(docsDir)

	logoPath := filepath.Join(docsDir, "logo"+ext)
	dst, err := os.Create(logoPath)
	if err != nil {
		slog.Error("failed to create logo file", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to save logo")
		return
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, file); err != nil {
		slog.Error("failed to write logo file", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to save logo")
		return
	}

	// Update the pdf.logo_path setting.
	ps, err := h.settingsSvc.GetPDFSettings(r.Context())
	if err != nil {
		slog.Error("failed to get PDF settings", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to update settings")
		return
	}
	ps.LogoPath = logoPath
	if err := h.settingsSvc.SavePDFSettings(r.Context(), ps); err != nil {
		slog.Error("failed to save logo path setting", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to update settings")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"logo_path": logoPath,
	})
}

// GetLogo handles GET /api/v1/settings/logo.
func (h *PDFSettingsHandler) GetLogo(w http.ResponseWriter, r *http.Request) {
	ps, err := h.settingsSvc.GetPDFSettings(r.Context())
	if err != nil {
		slog.Error("failed to get PDF settings", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get settings")
		return
	}

	if ps.LogoPath == "" {
		respondError(w, http.StatusNotFound, "no logo configured")
		return
	}

	if _, err := os.Stat(ps.LogoPath); err != nil {
		respondError(w, http.StatusNotFound, "logo file not found")
		return
	}

	ext := strings.ToLower(filepath.Ext(ps.LogoPath))
	contentType := "application/octet-stream"
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".svg":
		contentType = "image/svg+xml"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": "logo" + ext}))
	http.ServeFile(w, r, ps.LogoPath)
}

// DeleteLogo handles DELETE /api/v1/settings/logo.
func (h *PDFSettingsHandler) DeleteLogo(w http.ResponseWriter, r *http.Request) {
	ps, err := h.settingsSvc.GetPDFSettings(r.Context())
	if err != nil {
		slog.Error("failed to get PDF settings", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get settings")
		return
	}

	if ps.LogoPath != "" {
		_ = os.Remove(ps.LogoPath)
	}

	// Clear the logo_path setting.
	ps.LogoPath = ""
	if err := h.settingsSvc.SavePDFSettings(r.Context(), ps); err != nil {
		slog.Error("failed to clear logo path setting", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to update settings")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// PreviewPDF handles GET /api/v1/settings/pdf-preview.
func (h *PDFSettingsHandler) PreviewPDF(w http.ResponseWriter, r *http.Request) {
	ps, err := h.settingsSvc.GetPDFSettings(r.Context())
	if err != nil {
		slog.Error("failed to get PDF settings for preview", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load PDF settings")
		return
	}

	pdfSettings := pdf.PDFSettings{
		LogoPath:        ps.LogoPath,
		AccentColor:     ps.AccentColor,
		FooterText:      ps.FooterText,
		ShowQR:          ps.ShowQR,
		ShowBankDetails: ps.ShowBankDetails,
		FontSize:        ps.FontSize,
	}

	// Load supplier info.
	settings, err := h.settingsSvc.GetAll(r.Context())
	if err != nil {
		slog.Error("failed to load supplier settings for preview", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to load supplier settings")
		return
	}

	supplier := pdf.SupplierInfo{
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
	}

	// Try to use the most recent invoice.
	invoice := h.getPreviewInvoice(r)

	pdfBytes, err := h.pdfGen.Generate(r.Context(), invoice, supplier, pdfSettings)
	if err != nil {
		slog.Error("failed to generate preview PDF", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to generate preview PDF")
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": "preview.pdf"}))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}

// getPreviewInvoice returns the most recent invoice, or generates sample data.
func (h *PDFSettingsHandler) getPreviewInvoice(r *http.Request) *domain.Invoice {
	if h.invoiceSvc != nil {
		filter := domain.InvoiceFilter{Limit: 1, Offset: 0}
		invoices, _, err := h.invoiceSvc.List(r.Context(), filter)
		if err == nil && len(invoices) > 0 {
			return &invoices[0]
		}
	}

	// Generate sample invoice for preview.
	now := time.Now()
	return &domain.Invoice{
		InvoiceNumber: "2026-0001",
		Type:          domain.InvoiceTypeRegular,
		Status:        domain.InvoiceStatusDraft,
		IssueDate:     now,
		DueDate:       now.AddDate(0, 0, 14),
		DeliveryDate:  now,
		CurrencyCode:  "CZK",
		Items: []domain.InvoiceItem{
			{
				Description:    "Webovy vyvoj - brezen 2026",
				Quantity:       domain.Amount(10000),
				Unit:           "hod",
				UnitPrice:      domain.Amount(150000),
				VATRatePercent: 21,
				VATAmount:      domain.Amount(315000),
				TotalAmount:    domain.Amount(1815000),
			},
			{
				Description:    "Hosting a domena",
				Quantity:       domain.Amount(100),
				Unit:           "ks",
				UnitPrice:      domain.Amount(50000),
				VATRatePercent: 21,
				VATAmount:      domain.Amount(10500),
				TotalAmount:    domain.Amount(60500),
			},
		},
		SubtotalAmount: domain.Amount(1550000),
		VATAmount:      domain.Amount(325500),
		TotalAmount:    domain.Amount(1875500),
		VariableSymbol: "20260001",
		Customer: &domain.Contact{
			Name:   "Vzorova firma s.r.o.",
			ICO:    "12345678",
			DIC:    "CZ12345678",
			Street: "Vzorova 123",
			City:   "Praha",
			ZIP:    "11000",
		},
	}
}

// removeLogoFiles removes all logo.* files from the given directory.
func (h *PDFSettingsHandler) removeLogoFiles(dir string) {
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".svg"} {
		_ = os.Remove(filepath.Join(dir, "logo"+ext))
	}
}
