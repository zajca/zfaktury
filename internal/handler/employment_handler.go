package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
)

// EmploymentHandler handles HTTP requests for §6 employment income management
// (Potvrzení o zdanitelných příjmech ze závislé činnosti).
type EmploymentHandler struct {
	svc employmentService
}

// employmentService is the local interface against which this handler depends.
// The concrete implementation is *service.EmploymentCertificateService; using
// an interface keeps the handler buildable independently of the service file.
type employmentService interface {
	UploadDocument(ctx context.Context, year int, kind, filename, contentType string, content io.Reader) (*domain.EmploymentDocument, error)
	ExtractDocument(ctx context.Context, docID int64) (*domain.EmploymentCertificate, error)
	ListDocumentsByYear(ctx context.Context, year int) ([]*domain.EmploymentDocument, error)
	DeleteDocument(ctx context.Context, id int64) error
	Create(ctx context.Context, cert *domain.EmploymentCertificate) error
	Update(ctx context.Context, cert *domain.EmploymentCertificate) error
	Confirm(ctx context.Context, certID int64) error
	Get(ctx context.Context, certID int64) (*domain.EmploymentCertificate, error)
	ListByYear(ctx context.Context, year int) ([]*domain.EmploymentCertificate, error)
	Delete(ctx context.Context, certID int64) error
}

// NewEmploymentHandler returns a handler wired to the employment certificate
// service.
func NewEmploymentHandler(svc employmentService) *EmploymentHandler {
	return &EmploymentHandler{svc: svc}
}

// maxEmploymentUploadBytes caps the multipart body for /documents uploads at
// 10 MB per RFC-016.
const maxEmploymentUploadBytes = 10 << 20

// Routes returns a chi router with all employment endpoints registered.
func (h *EmploymentHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Documents.
	r.Post("/documents", h.UploadDocument)
	r.Get("/documents", h.ListDocuments)
	r.Post("/documents/{id}/extract", h.ExtractDocument)
	r.Delete("/documents/{id}", h.DeleteDocument)

	// Certificates.
	r.Get("/certificates", h.ListCertificates)
	r.Post("/certificates", h.CreateCertificate)
	r.Get("/certificates/{id}", h.GetCertificate)
	r.Put("/certificates/{id}", h.UpdateCertificate)
	r.Post("/certificates/{id}/confirm", h.ConfirmCertificate)
	r.Delete("/certificates/{id}", h.DeleteCertificate)

	return r
}

// --- DTOs ---

// employmentDocumentResponse is the JSON response for an employment document.
type employmentDocumentResponse struct {
	ID               int64  `json:"id"`
	Year             int    `json:"year"`
	Kind             string `json:"kind"`
	Filename         string `json:"filename"`
	ContentType      string `json:"content_type"`
	Size             int64  `json:"size"`
	ExtractionStatus string `json:"extraction_status"`
	ExtractionError  string `json:"extraction_error,omitempty"`
	CreatedAt        string `json:"created_at"`
}

// employmentCertificateRequest is the JSON request body for create/update of a
// certificate. Money fields are accepted in CZK (whole CZK + decimals) and
// converted to halere via amountFromCZK on entry.
type employmentCertificateRequest struct {
	Year                      int     `json:"year"`
	DocumentID                *int64  `json:"document_id,omitempty"`
	CertificateType           string  `json:"certificate_type"`
	EmployerName              string  `json:"employer_name"`
	EmployerICO               string  `json:"employer_ico"`
	EmployerAddress           string  `json:"employer_address,omitempty"`
	ContractType              string  `json:"contract_type"`
	PeriodFrom                string  `json:"period_from"`
	PeriodTo                  string  `json:"period_to"`
	GrossIncomeCZK            float64 `json:"gross_income_czk"`
	IncomeWithoutAdvanceCZK   float64 `json:"income_without_advance_czk"`
	ForeignTaxPaidCZK         float64 `json:"foreign_tax_paid_czk"`
	AdvanceTaxWithheldCZK     float64 `json:"advance_tax_withheld_czk"`
	AnnualSettlementRefundCZK float64 `json:"annual_settlement_refund_czk"`
	MonthlyBonusPaidCZK       float64 `json:"monthly_bonus_paid_czk"`
	WithheldFinalTaxCZK       float64 `json:"withheld_final_tax_czk"`
	IncludeWithholdingInDAP   bool    `json:"include_withholding_in_dap"`
	Notes                     string  `json:"notes,omitempty"`
}

// employmentCertificateResponse is the JSON response for an employment
// certificate.
type employmentCertificateResponse struct {
	ID                        int64   `json:"id"`
	Year                      int     `json:"year"`
	DocumentID                *int64  `json:"document_id,omitempty"`
	CertificateType           string  `json:"certificate_type"`
	EmployerName              string  `json:"employer_name"`
	EmployerICO               string  `json:"employer_ico"`
	EmployerAddress           string  `json:"employer_address,omitempty"`
	ContractType              string  `json:"contract_type"`
	PeriodFrom                string  `json:"period_from"`
	PeriodTo                  string  `json:"period_to"`
	GrossIncomeCZK            float64 `json:"gross_income_czk"`
	IncomeWithoutAdvanceCZK   float64 `json:"income_without_advance_czk"`
	ForeignTaxPaidCZK         float64 `json:"foreign_tax_paid_czk"`
	AdvanceTaxWithheldCZK     float64 `json:"advance_tax_withheld_czk"`
	AnnualSettlementRefundCZK float64 `json:"annual_settlement_refund_czk"`
	MonthlyBonusPaidCZK       float64 `json:"monthly_bonus_paid_czk"`
	WithheldFinalTaxCZK       float64 `json:"withheld_final_tax_czk"`
	IncludeWithholdingInDAP   bool    `json:"include_withholding_in_dap"`
	Notes                     string  `json:"notes,omitempty"`
	Status                    string  `json:"status"`
	CreatedAt                 string  `json:"created_at"`
	UpdatedAt                 string  `json:"updated_at"`
}

// --- Conversion helpers ---

// amountFromCZK converts a CZK float (e.g. 12345.67) into halere as
// domain.Amount. Result is rounded half-away-from-zero.
func amountFromCZK(czk float64) domain.Amount {
	if czk >= 0 {
		return domain.Amount(czk*100 + 0.5)
	}
	return domain.Amount(czk*100 - 0.5)
}

// amountToCZK converts halere to CZK as a float.
func amountToCZK(a domain.Amount) float64 {
	return float64(a) / 100.0
}

// employmentDocFromDomain maps a domain.EmploymentDocument to its DTO.
func employmentDocFromDomain(doc *domain.EmploymentDocument) employmentDocumentResponse {
	return employmentDocumentResponse{
		ID:               doc.ID,
		Year:             doc.Year,
		Kind:             string(doc.Kind),
		Filename:         doc.Filename,
		ContentType:      doc.ContentType,
		Size:             doc.Size,
		ExtractionStatus: doc.ExtractionStatus,
		ExtractionError:  doc.ExtractionError,
		CreatedAt:        doc.CreatedAt.Format(time.RFC3339),
	}
}

// employmentCertFromDomain maps a domain.EmploymentCertificate to its DTO.
func employmentCertFromDomain(c *domain.EmploymentCertificate) employmentCertificateResponse {
	return employmentCertificateResponse{
		ID:                        c.ID,
		Year:                      c.Year,
		DocumentID:                c.DocumentID,
		CertificateType:           string(c.CertificateType),
		EmployerName:              c.EmployerName,
		EmployerICO:               c.EmployerICO,
		EmployerAddress:           c.EmployerAddress,
		ContractType:              string(c.ContractType),
		PeriodFrom:                c.PeriodFrom.Format("2006-01-02"),
		PeriodTo:                  c.PeriodTo.Format("2006-01-02"),
		GrossIncomeCZK:            amountToCZK(c.GrossIncome),
		IncomeWithoutAdvanceCZK:   amountToCZK(c.IncomeWithoutAdvance),
		ForeignTaxPaidCZK:         amountToCZK(c.ForeignTaxPaid),
		AdvanceTaxWithheldCZK:     amountToCZK(c.AdvanceTaxWithheld),
		AnnualSettlementRefundCZK: amountToCZK(c.AnnualSettlementRefund),
		MonthlyBonusPaidCZK:       amountToCZK(c.MonthlyBonusPaid),
		WithheldFinalTaxCZK:       amountToCZK(c.WithheldFinalTax),
		IncludeWithholdingInDAP:   c.IncludeWithholdingInDAP,
		Notes:                     c.Notes,
		Status:                    c.Status,
		CreatedAt:                 c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:                 c.UpdatedAt.Format(time.RFC3339),
	}
}

// toDomain maps a request DTO to a domain.EmploymentCertificate. The id field
// is left to the caller to set on update paths.
func (r *employmentCertificateRequest) toDomain() (*domain.EmploymentCertificate, error) {
	periodFrom, err := time.Parse("2006-01-02", r.PeriodFrom)
	if err != nil {
		return nil, errors.New("invalid period_from format, expected YYYY-MM-DD")
	}
	periodTo, err := time.Parse("2006-01-02", r.PeriodTo)
	if err != nil {
		return nil, errors.New("invalid period_to format, expected YYYY-MM-DD")
	}

	cert := &domain.EmploymentCertificate{
		Year:                    r.Year,
		DocumentID:              r.DocumentID,
		CertificateType:         domain.CertificateType(r.CertificateType),
		EmployerName:            r.EmployerName,
		EmployerICO:             r.EmployerICO,
		EmployerAddress:         r.EmployerAddress,
		ContractType:            domain.ContractType(r.ContractType),
		PeriodFrom:              periodFrom,
		PeriodTo:                periodTo,
		GrossIncome:             amountFromCZK(r.GrossIncomeCZK),
		IncomeWithoutAdvance:    amountFromCZK(r.IncomeWithoutAdvanceCZK),
		ForeignTaxPaid:          amountFromCZK(r.ForeignTaxPaidCZK),
		AdvanceTaxWithheld:      amountFromCZK(r.AdvanceTaxWithheldCZK),
		AnnualSettlementRefund:  amountFromCZK(r.AnnualSettlementRefundCZK),
		MonthlyBonusPaid:        amountFromCZK(r.MonthlyBonusPaidCZK),
		WithheldFinalTax:        amountFromCZK(r.WithheldFinalTaxCZK),
		IncludeWithholdingInDAP: r.IncludeWithholdingInDAP,
		Notes:                   r.Notes,
	}
	return cert, nil
}

// --- Document handlers ---

// UploadDocument handles POST /documents (multipart form, max 10 MB).
// Required form fields: file. Required query parameters: year. Optional: kind
// (defaults to "advance").
func (h *EmploymentHandler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	// Hard cap on the request body before parsing the form to prevent memory
	// exhaustion.
	r.Body = http.MaxBytesReader(w, r.Body, maxEmploymentUploadBytes)
	if err := r.ParseMultipartForm(maxEmploymentUploadBytes); err != nil {
		respondError(w, http.StatusBadRequest, "invalid multipart form or file too large")
		return
	}

	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		yearStr = r.FormValue("year")
	}
	if yearStr == "" {
		respondError(w, http.StatusBadRequest, "year is required")
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return
	}

	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = r.FormValue("kind")
	}
	if kind == "" {
		kind = string(domain.EmploymentDocAdvance)
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer func() { _ = file.Close() }()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	doc, err := h.svc.UploadDocument(r.Context(), year, kind, header.Filename, contentType, file)
	if err != nil {
		slog.Error("uploading employment document", "error", err)
		mapDomainError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, employmentDocFromDomain(doc))
}

// ListDocuments handles GET /documents?year=.
func (h *EmploymentHandler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	year, ok := readYearQuery(w, r)
	if !ok {
		return
	}

	docs, err := h.svc.ListDocumentsByYear(r.Context(), year)
	if err != nil {
		slog.Error("listing employment documents", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list documents")
		return
	}

	items := make([]employmentDocumentResponse, 0, len(docs))
	for _, d := range docs {
		items = append(items, employmentDocFromDomain(d))
	}
	respondJSON(w, http.StatusOK, items)
}

// DeleteDocument handles DELETE /documents/{id}.
func (h *EmploymentHandler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID")
		return
	}
	if err := h.svc.DeleteDocument(r.Context(), id); err != nil {
		slog.Error("deleting employment document", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ExtractDocument handles POST /documents/{id}/extract. Returns a draft
// EmploymentCertificate parsed by the OCR provider.
func (h *EmploymentHandler) ExtractDocument(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document ID")
		return
	}
	cert, err := h.svc.ExtractDocument(r.Context(), id)
	if err != nil {
		slog.Error("extracting employment document", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, employmentCertFromDomain(cert))
}

// --- Certificate handlers ---

// ListCertificates handles GET /certificates?year=.
func (h *EmploymentHandler) ListCertificates(w http.ResponseWriter, r *http.Request) {
	year, ok := readYearQuery(w, r)
	if !ok {
		return
	}
	certs, err := h.svc.ListByYear(r.Context(), year)
	if err != nil {
		slog.Error("listing employment certificates", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list certificates")
		return
	}
	items := make([]employmentCertificateResponse, 0, len(certs))
	for _, c := range certs {
		items = append(items, employmentCertFromDomain(c))
	}
	respondJSON(w, http.StatusOK, items)
}

// GetCertificate handles GET /certificates/{id}.
func (h *EmploymentHandler) GetCertificate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid certificate ID")
		return
	}
	cert, err := h.svc.Get(r.Context(), id)
	if err != nil {
		slog.Error("fetching employment certificate", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, employmentCertFromDomain(cert))
}

// CreateCertificate handles POST /certificates.
func (h *EmploymentHandler) CreateCertificate(w http.ResponseWriter, r *http.Request) {
	var req employmentCertificateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	cert, err := req.toDomain()
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.Create(r.Context(), cert); err != nil {
		slog.Error("creating employment certificate", "error", err)
		mapDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, employmentCertFromDomain(cert))
}

// UpdateCertificate handles PUT /certificates/{id}.
func (h *EmploymentHandler) UpdateCertificate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid certificate ID")
		return
	}
	var req employmentCertificateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	cert, err := req.toDomain()
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	cert.ID = id
	if err := h.svc.Update(r.Context(), cert); err != nil {
		slog.Error("updating employment certificate", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, employmentCertFromDomain(cert))
}

// ConfirmCertificate handles POST /certificates/{id}/confirm.
func (h *EmploymentHandler) ConfirmCertificate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid certificate ID")
		return
	}
	if err := h.svc.Confirm(r.Context(), id); err != nil {
		slog.Error("confirming employment certificate", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}
	// Return refreshed certificate so callers can render the new status.
	cert, err := h.svc.Get(r.Context(), id)
	if err != nil {
		slog.Error("fetching certificate after confirm", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, employmentCertFromDomain(cert))
}

// DeleteCertificate handles DELETE /certificates/{id}.
func (h *EmploymentHandler) DeleteCertificate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid certificate ID")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		slog.Error("deleting employment certificate", "error", err, "id", id)
		mapDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

// readYearQuery parses the required ?year= query parameter and writes a 400
// response if it is missing or invalid. Returns (year, true) on success.
func readYearQuery(w http.ResponseWriter, r *http.Request) (int, bool) {
	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		respondError(w, http.StatusBadRequest, "year query parameter is required")
		return 0, false
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid year parameter")
		return 0, false
	}
	return year, true
}
