package handler

import (
	"encoding/json"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service"
)

// InvestmentIncomeHandler handles HTTP requests for investment income management.
type InvestmentIncomeHandler struct {
	investmentSvc *service.InvestmentIncomeService
	docSvc        *service.InvestmentDocumentService
	extractionSvc *service.InvestmentExtractionService // nullable
}

// NewInvestmentIncomeHandler creates a new InvestmentIncomeHandler.
func NewInvestmentIncomeHandler(
	investmentSvc *service.InvestmentIncomeService,
	docSvc *service.InvestmentDocumentService,
	extractionSvc *service.InvestmentExtractionService,
) *InvestmentIncomeHandler {
	return &InvestmentIncomeHandler{
		investmentSvc: investmentSvc,
		docSvc:        docSvc,
		extractionSvc: extractionSvc,
	}
}

// Routes registers investment income routes on the given router.
func (h *InvestmentIncomeHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Documents
	r.Post("/documents", h.UploadDocument)
	r.Get("/documents", h.ListDocuments)
	r.Delete("/documents/{id}", h.DeleteDocument)
	r.Post("/documents/{id}/extract", h.ExtractDocument)
	r.Get("/documents/{id}/download", h.DownloadDocument)

	// Capital income (§8)
	r.Get("/capital-income", h.ListCapitalIncome)
	r.Post("/capital-income", h.CreateCapitalIncome)
	r.Put("/capital-income/{id}", h.UpdateCapitalIncome)
	r.Delete("/capital-income/{id}", h.DeleteCapitalIncome)

	// Security transactions (§10)
	r.Get("/security-transactions", h.ListSecurityTransactions)
	r.Post("/security-transactions", h.CreateSecurityTransaction)
	r.Put("/security-transactions/{id}", h.UpdateSecurityTransaction)
	r.Delete("/security-transactions/{id}", h.DeleteSecurityTransaction)

	// Computation
	r.Get("/summary/{year}", h.GetYearSummary)
	r.Post("/recalculate-fifo/{year}", h.RecalculateFIFO)

	return r
}

// --- DTOs ---

// investmentDocumentResponse is the JSON response for an investment document.
type investmentDocumentResponse struct {
	ID               int64  `json:"id"`
	Year             int    `json:"year"`
	Platform         string `json:"platform"`
	Kind             string `json:"kind"`
	Filename         string `json:"filename"`
	ContentType      string `json:"content_type"`
	Size             int64  `json:"size"`
	ExtractionStatus string `json:"extraction_status"`
	ExtractionError  string `json:"extraction_error,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// capitalIncomeRequest is the JSON request body for creating/updating a capital income entry.
type capitalIncomeRequest struct {
	Year               int    `json:"year"`
	Category           string `json:"category"`
	Description        string `json:"description"`
	IncomeDate         string `json:"income_date"`
	GrossAmount        int64  `json:"gross_amount"`
	WithheldTaxCZ      int64  `json:"withheld_tax_cz"`
	WithheldTaxForeign int64  `json:"withheld_tax_foreign"`
	CountryCode        string `json:"country_code"`
	NeedsDeclaring     bool   `json:"needs_declaring"`
}

// capitalIncomeResponse is the JSON response for a capital income entry.
type capitalIncomeResponse struct {
	ID                 int64  `json:"id"`
	Year               int    `json:"year"`
	DocumentID         *int64 `json:"document_id,omitempty"`
	Category           string `json:"category"`
	Description        string `json:"description"`
	IncomeDate         string `json:"income_date"`
	GrossAmount        int64  `json:"gross_amount"`
	WithheldTaxCZ      int64  `json:"withheld_tax_cz"`
	WithheldTaxForeign int64  `json:"withheld_tax_foreign"`
	CountryCode        string `json:"country_code"`
	NeedsDeclaring     bool   `json:"needs_declaring"`
	NetAmount          int64  `json:"net_amount"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

// securityTransactionRequest is the JSON request body for creating/updating a security transaction.
type securityTransactionRequest struct {
	Year            int    `json:"year"`
	AssetType       string `json:"asset_type"`
	AssetName       string `json:"asset_name"`
	ISIN            string `json:"isin"`
	TransactionType string `json:"transaction_type"`
	TransactionDate string `json:"transaction_date"`
	Quantity        int64  `json:"quantity"`
	UnitPrice       int64  `json:"unit_price"`
	TotalAmount     int64  `json:"total_amount"`
	Fees            int64  `json:"fees"`
	CurrencyCode    string `json:"currency_code"`
	ExchangeRate    int64  `json:"exchange_rate"`
}

// securityTransactionResponse is the JSON response for a security transaction.
type securityTransactionResponse struct {
	ID              int64  `json:"id"`
	Year            int    `json:"year"`
	DocumentID      *int64 `json:"document_id,omitempty"`
	AssetType       string `json:"asset_type"`
	AssetName       string `json:"asset_name"`
	ISIN            string `json:"isin"`
	TransactionType string `json:"transaction_type"`
	TransactionDate string `json:"transaction_date"`
	Quantity        int64  `json:"quantity"`
	UnitPrice       int64  `json:"unit_price"`
	TotalAmount     int64  `json:"total_amount"`
	Fees            int64  `json:"fees"`
	CurrencyCode    string `json:"currency_code"`
	ExchangeRate    int64  `json:"exchange_rate"`
	CostBasis       int64  `json:"cost_basis"`
	ComputedGain    int64  `json:"computed_gain"`
	TimeTestExempt  bool   `json:"time_test_exempt"`
	ExemptAmount    int64  `json:"exempt_amount"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// investmentYearSummaryResponse is the JSON response for a year summary.
type investmentYearSummaryResponse struct {
	Year                int   `json:"year"`
	CapitalIncomeGross  int64 `json:"capital_income_gross"`
	CapitalIncomeTax    int64 `json:"capital_income_tax"`
	CapitalIncomeNet    int64 `json:"capital_income_net"`
	OtherIncomeGross    int64 `json:"other_income_gross"`
	OtherIncomeExpenses int64 `json:"other_income_expenses"`
	OtherIncomeExempt   int64 `json:"other_income_exempt"`
	OtherIncomeNet      int64 `json:"other_income_net"`
}

// --- Domain conversion helpers ---

// investmentDocFromDomain converts a domain.InvestmentDocument to an investmentDocumentResponse.
func investmentDocFromDomain(doc *domain.InvestmentDocument) investmentDocumentResponse {
	return investmentDocumentResponse{
		ID:               doc.ID,
		Year:             doc.Year,
		Platform:         doc.Platform,
		Kind:             doc.Kind,
		Filename:         doc.Filename,
		ContentType:      doc.ContentType,
		Size:             doc.Size,
		ExtractionStatus: doc.ExtractionStatus,
		ExtractionError:  doc.ExtractionError,
		CreatedAt:        doc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        doc.UpdatedAt.Format(time.RFC3339),
	}
}

// capitalIncomeFromDomain converts a domain.CapitalIncomeEntry to a capitalIncomeResponse.
func capitalIncomeFromDomain(e *domain.CapitalIncomeEntry) capitalIncomeResponse {
	return capitalIncomeResponse{
		ID:                 e.ID,
		Year:               e.Year,
		DocumentID:         e.DocumentID,
		Category:           e.Category,
		Description:        e.Description,
		IncomeDate:         e.IncomeDate.Format("2006-01-02"),
		GrossAmount:        int64(e.GrossAmount),
		WithheldTaxCZ:      int64(e.WithheldTaxCZ),
		WithheldTaxForeign: int64(e.WithheldTaxForeign),
		CountryCode:        e.CountryCode,
		NeedsDeclaring:     e.NeedsDeclaring,
		NetAmount:          int64(e.NetAmount),
		CreatedAt:          e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          e.UpdatedAt.Format(time.RFC3339),
	}
}

// securityTxFromDomain converts a domain.SecurityTransaction to a securityTransactionResponse.
func securityTxFromDomain(tx *domain.SecurityTransaction) securityTransactionResponse {
	return securityTransactionResponse{
		ID:              tx.ID,
		Year:            tx.Year,
		DocumentID:      tx.DocumentID,
		AssetType:       tx.AssetType,
		AssetName:       tx.AssetName,
		ISIN:            tx.ISIN,
		TransactionType: tx.TransactionType,
		TransactionDate: tx.TransactionDate.Format("2006-01-02"),
		Quantity:        tx.Quantity,
		UnitPrice:       int64(tx.UnitPrice),
		TotalAmount:     int64(tx.TotalAmount),
		Fees:            int64(tx.Fees),
		CurrencyCode:    tx.CurrencyCode,
		ExchangeRate:    tx.ExchangeRate,
		CostBasis:       int64(tx.CostBasis),
		ComputedGain:    int64(tx.ComputedGain),
		TimeTestExempt:  tx.TimeTestExempt,
		ExemptAmount:    int64(tx.ExemptAmount),
		CreatedAt:       tx.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       tx.UpdatedAt.Format(time.RFC3339),
	}
}

// --- Document handlers ---

// UploadDocument handles POST /documents (multipart form).
func (h *InvestmentIncomeHandler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	// Hard cap on request body to prevent memory exhaustion.
	r.Body = http.MaxBytesReader(w, r.Body, 34<<20) // 34 MB (32 MB file + overhead)
	// 32 MB max form size.
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		respondError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer func() { _ = file.Close() }()

	yearStr := r.FormValue("year")
	if yearStr == "" {
		respondError(w, http.StatusBadRequest, "year is required")
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	platform := r.FormValue("platform")
	if platform == "" {
		respondError(w, http.StatusBadRequest, "platform is required")
		return
	}

	// Optional kind: "statement" (default) or "data". data uploads accept
	// xlsx/csv/zip/... and skip OCR.
	kind := r.FormValue("kind")

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	doc, err := h.docSvc.Upload(r.Context(), year, platform, kind, header.Filename, contentType, file)
	if err != nil {
		slog.Error("failed to upload investment document", "error", err)
		mapDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, investmentDocFromDomain(doc))
}

// ListDocuments handles GET /documents?year=.
func (h *InvestmentIncomeHandler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		respondError(w, http.StatusBadRequest, "year query parameter is required")
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	docs, err := h.docSvc.ListByYear(r.Context(), year)
	if err != nil {
		slog.Error("failed to list investment documents", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list documents")
		return
	}

	items := make([]investmentDocumentResponse, 0, len(docs))
	for i := range docs {
		items = append(items, investmentDocFromDomain(&docs[i]))
	}

	respondJSON(w, http.StatusOK, items)
}

// DeleteDocument handles DELETE /documents/{id}.
func (h *InvestmentIncomeHandler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	if err := h.docSvc.Delete(r.Context(), id); err != nil {
		slog.Error("failed to delete investment document", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ExtractDocument handles POST /documents/{id}/extract.
func (h *InvestmentIncomeHandler) ExtractDocument(w http.ResponseWriter, r *http.Request) {
	if h.extractionSvc == nil {
		respondError(w, http.StatusNotImplemented, "OCR not configured")
		return
	}

	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	result, err := h.extractionSvc.ExtractFromDocument(r.Context(), id)
	if err != nil {
		slog.Error("failed to extract investment document", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}

	// Convert extraction result to response DTOs.
	type extractionResponse struct {
		Platform       string                        `json:"platform"`
		CapitalEntries []capitalIncomeResponse       `json:"capital_entries"`
		Transactions   []securityTransactionResponse `json:"transactions"`
		Confidence     float64                       `json:"confidence"`
	}

	resp := extractionResponse{
		Platform:       result.Platform,
		CapitalEntries: make([]capitalIncomeResponse, 0, len(result.CapitalEntries)),
		Transactions:   make([]securityTransactionResponse, 0, len(result.Transactions)),
		Confidence:     result.Confidence,
	}
	for i := range result.CapitalEntries {
		resp.CapitalEntries = append(resp.CapitalEntries, capitalIncomeFromDomain(&result.CapitalEntries[i]))
	}
	for i := range result.Transactions {
		resp.Transactions = append(resp.Transactions, securityTxFromDomain(&result.Transactions[i]))
	}

	respondJSON(w, http.StatusOK, resp)
}

// DownloadDocument handles GET /documents/{id}/download.
func (h *InvestmentIncomeHandler) DownloadDocument(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID")
		return
	}

	filePath, contentType, err := h.docSvc.GetFilePath(r.Context(), id)
	if err != nil {
		slog.Error("failed to get investment document file path", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}

	// Get the document metadata for the filename.
	doc, err := h.docSvc.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("failed to get investment document metadata", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": doc.Filename}))
	http.ServeFile(w, r, filePath)
}

// --- Capital income handlers ---

// ListCapitalIncome handles GET /capital-income?year=.
func (h *InvestmentIncomeHandler) ListCapitalIncome(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		respondError(w, http.StatusBadRequest, "year query parameter is required")
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	entries, err := h.investmentSvc.ListCapitalEntries(r.Context(), year)
	if err != nil {
		slog.Error("failed to list capital income entries", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list capital income entries")
		return
	}

	items := make([]capitalIncomeResponse, 0, len(entries))
	for i := range entries {
		items = append(items, capitalIncomeFromDomain(&entries[i]))
	}

	respondJSON(w, http.StatusOK, items)
}

// CreateCapitalIncome handles POST /capital-income.
func (h *InvestmentIncomeHandler) CreateCapitalIncome(w http.ResponseWriter, r *http.Request) {
	var req capitalIncomeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	incomeDate, err := time.Parse("2006-01-02", req.IncomeDate)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid income_date format, expected YYYY-MM-DD")
		return
	}

	entry := &domain.CapitalIncomeEntry{
		Year:               req.Year,
		Category:           req.Category,
		Description:        req.Description,
		IncomeDate:         incomeDate,
		GrossAmount:        domain.Amount(req.GrossAmount),
		WithheldTaxCZ:      domain.Amount(req.WithheldTaxCZ),
		WithheldTaxForeign: domain.Amount(req.WithheldTaxForeign),
		CountryCode:        req.CountryCode,
		NeedsDeclaring:     req.NeedsDeclaring,
	}

	if err := h.investmentSvc.CreateCapitalEntry(r.Context(), entry); err != nil {
		slog.Error("failed to create capital income entry", "error", err)
		mapDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, capitalIncomeFromDomain(entry))
}

// UpdateCapitalIncome handles PUT /capital-income/{id}.
func (h *InvestmentIncomeHandler) UpdateCapitalIncome(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid capital income entry ID")
		return
	}

	var req capitalIncomeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	incomeDate, err := time.Parse("2006-01-02", req.IncomeDate)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid income_date format, expected YYYY-MM-DD")
		return
	}

	entry := &domain.CapitalIncomeEntry{
		ID:                 id,
		Year:               req.Year,
		Category:           req.Category,
		Description:        req.Description,
		IncomeDate:         incomeDate,
		GrossAmount:        domain.Amount(req.GrossAmount),
		WithheldTaxCZ:      domain.Amount(req.WithheldTaxCZ),
		WithheldTaxForeign: domain.Amount(req.WithheldTaxForeign),
		CountryCode:        req.CountryCode,
		NeedsDeclaring:     req.NeedsDeclaring,
	}

	if err := h.investmentSvc.UpdateCapitalEntry(r.Context(), entry); err != nil {
		slog.Error("failed to update capital income entry", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, capitalIncomeFromDomain(entry))
}

// DeleteCapitalIncome handles DELETE /capital-income/{id}.
func (h *InvestmentIncomeHandler) DeleteCapitalIncome(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid capital income entry ID")
		return
	}

	if err := h.investmentSvc.DeleteCapitalEntry(r.Context(), id); err != nil {
		slog.Error("failed to delete capital income entry", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Security transaction handlers ---

// ListSecurityTransactions handles GET /security-transactions?year=.
func (h *InvestmentIncomeHandler) ListSecurityTransactions(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		respondError(w, http.StatusBadRequest, "year query parameter is required")
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	txs, err := h.investmentSvc.ListSecurityTransactions(r.Context(), year)
	if err != nil {
		slog.Error("failed to list security transactions", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list security transactions")
		return
	}

	items := make([]securityTransactionResponse, 0, len(txs))
	for i := range txs {
		items = append(items, securityTxFromDomain(&txs[i]))
	}

	respondJSON(w, http.StatusOK, items)
}

// CreateSecurityTransaction handles POST /security-transactions.
func (h *InvestmentIncomeHandler) CreateSecurityTransaction(w http.ResponseWriter, r *http.Request) {
	var req securityTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	txDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid transaction_date format, expected YYYY-MM-DD")
		return
	}

	tx := &domain.SecurityTransaction{
		Year:            req.Year,
		AssetType:       req.AssetType,
		AssetName:       req.AssetName,
		ISIN:            req.ISIN,
		TransactionType: req.TransactionType,
		TransactionDate: txDate,
		Quantity:        req.Quantity,
		UnitPrice:       domain.Amount(req.UnitPrice),
		TotalAmount:     domain.Amount(req.TotalAmount),
		Fees:            domain.Amount(req.Fees),
		CurrencyCode:    req.CurrencyCode,
		ExchangeRate:    req.ExchangeRate,
	}

	if err := h.investmentSvc.CreateSecurityTransaction(r.Context(), tx); err != nil {
		slog.Error("failed to create security transaction", "error", err)
		mapDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, securityTxFromDomain(tx))
}

// UpdateSecurityTransaction handles PUT /security-transactions/{id}.
func (h *InvestmentIncomeHandler) UpdateSecurityTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid security transaction ID")
		return
	}

	var req securityTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	txDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid transaction_date format, expected YYYY-MM-DD")
		return
	}

	tx := &domain.SecurityTransaction{
		ID:              id,
		Year:            req.Year,
		AssetType:       req.AssetType,
		AssetName:       req.AssetName,
		ISIN:            req.ISIN,
		TransactionType: req.TransactionType,
		TransactionDate: txDate,
		Quantity:        req.Quantity,
		UnitPrice:       domain.Amount(req.UnitPrice),
		TotalAmount:     domain.Amount(req.TotalAmount),
		Fees:            domain.Amount(req.Fees),
		CurrencyCode:    req.CurrencyCode,
		ExchangeRate:    req.ExchangeRate,
	}

	if err := h.investmentSvc.UpdateSecurityTransaction(r.Context(), tx); err != nil {
		slog.Error("failed to update security transaction", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, securityTxFromDomain(tx))
}

// DeleteSecurityTransaction handles DELETE /security-transactions/{id}.
func (h *InvestmentIncomeHandler) DeleteSecurityTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid security transaction ID")
		return
	}

	if err := h.investmentSvc.DeleteSecurityTransaction(r.Context(), id); err != nil {
		slog.Error("failed to delete security transaction", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Computation handlers ---

// GetYearSummary handles GET /summary/{year}.
func (h *InvestmentIncomeHandler) GetYearSummary(w http.ResponseWriter, r *http.Request) {
	yearStr := chi.URLParam(r, "year")
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	summary, err := h.investmentSvc.GetYearSummary(r.Context(), year)
	if err != nil {
		slog.Error("failed to get investment year summary", "error", err, "year", year)
		respondError(w, http.StatusInternalServerError, "failed to get year summary")
		return
	}

	respondJSON(w, http.StatusOK, investmentYearSummaryResponse{
		Year:                summary.Year,
		CapitalIncomeGross:  int64(summary.CapitalIncomeGross),
		CapitalIncomeTax:    int64(summary.CapitalIncomeTax),
		CapitalIncomeNet:    int64(summary.CapitalIncomeNet),
		OtherIncomeGross:    int64(summary.OtherIncomeGross),
		OtherIncomeExpenses: int64(summary.OtherIncomeExpenses),
		OtherIncomeExempt:   int64(summary.OtherIncomeExempt),
		OtherIncomeNet:      int64(summary.OtherIncomeNet),
	})
}

// RecalculateFIFO handles POST /recalculate-fifo/{year}.
func (h *InvestmentIncomeHandler) RecalculateFIFO(w http.ResponseWriter, r *http.Request) {
	yearStr := chi.URLParam(r, "year")
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	if err := h.investmentSvc.RecalculateFIFO(r.Context(), year); err != nil {
		slog.Error("failed to recalculate FIFO", "error", err, "year", year)
		mapDomainError(w, err)
		return
	}

	// Return the updated summary after recalculation.
	summary, err := h.investmentSvc.GetYearSummary(r.Context(), year)
	if err != nil {
		slog.Error("failed to get summary after FIFO recalculation", "error", err, "year", year)
		respondError(w, http.StatusInternalServerError, "FIFO recalculated but failed to get summary")
		return
	}

	respondJSON(w, http.StatusOK, investmentYearSummaryResponse{
		Year:                summary.Year,
		CapitalIncomeGross:  int64(summary.CapitalIncomeGross),
		CapitalIncomeTax:    int64(summary.CapitalIncomeTax),
		CapitalIncomeNet:    int64(summary.CapitalIncomeNet),
		OtherIncomeGross:    int64(summary.OtherIncomeGross),
		OtherIncomeExpenses: int64(summary.OtherIncomeExpenses),
		OtherIncomeExempt:   int64(summary.OtherIncomeExempt),
		OtherIncomeNet:      int64(summary.OtherIncomeNet),
	})
}
