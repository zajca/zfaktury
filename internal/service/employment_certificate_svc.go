package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/service/ocr"
)

// employmentDocumentRepo is the local repository contract consumed by
// EmploymentCertificateService for Potvrzení uploads. The lead promotes this
// interface into internal/repository/interfaces.go during merge.
type employmentDocumentRepo interface {
	Create(ctx context.Context, doc *domain.EmploymentDocument) error
	GetByID(ctx context.Context, id int64) (*domain.EmploymentDocument, error)
	ListByYear(ctx context.Context, year int) ([]*domain.EmploymentDocument, error)
	Delete(ctx context.Context, id int64) error
	UpdateExtraction(ctx context.Context, id int64, status, errMsg string) error
}

// employmentCertificateRepo is the local repository contract for parsed/manual
// Potvrzení certificates. Lead promotes into interfaces.go during merge.
type employmentCertificateRepo interface {
	Create(ctx context.Context, cert *domain.EmploymentCertificate) error
	GetByID(ctx context.Context, id int64) (*domain.EmploymentCertificate, error)
	Update(ctx context.Context, cert *domain.EmploymentCertificate) error
	Delete(ctx context.Context, id int64) error
	ListByYear(ctx context.Context, year int) ([]*domain.EmploymentCertificate, error)
	ListConfirmedByYear(ctx context.Context, year int) ([]*domain.EmploymentCertificate, error)
}

// employmentAllowedContentTypes is the MIME allowlist for Potvrzení uploads
// (RFC-016 §Service: PDF/JPEG/PNG/WEBP). HEIC is intentionally excluded — Czech
// employers do not issue HEIC Potvrzení.
var employmentAllowedContentTypes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
}

// employmentMaxDocumentSize caps employment Potvrzení uploads at 10 MB per RFC-016 §Service.
const employmentMaxDocumentSize = 10 << 20 // 10 MB

// employmentOCRSupportedContentTypes mirrors ocrSupportedInvestmentContentTypes —
// the OCR vision provider does not support WEBP yet, so confine extraction to
// PDF / JPEG / PNG.
var employmentOCRSupportedContentTypes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
}

// employmentCertificateStatusDraft / Confirmed mirror the status column.
const (
	employmentCertificateStatusDraft     = "draft"
	employmentCertificateStatusConfirmed = "confirmed"
)

// EmploymentCertificateService provides business logic for Potvrzení o
// zdanitelných příjmech ze závislé činnosti (§6 ZDP) document upload, AI
// extraction, and certificate CRUD.
type EmploymentCertificateService struct {
	docs    employmentDocumentRepo
	certs   employmentCertificateRepo
	ocr     ocr.Provider
	audit   *AuditService
	dataDir string
}

// NewEmploymentCertificateService wires a new EmploymentCertificateService.
// `ocrProvider` may be nil if the deployment runs without AI (manual entry only);
// ExtractDocument will return ErrInvalidInput in that case.
func NewEmploymentCertificateService(
	docs employmentDocumentRepo,
	certs employmentCertificateRepo,
	ocrProvider ocr.Provider,
	audit *AuditService,
	dataDir string,
) *EmploymentCertificateService {
	return &EmploymentCertificateService{
		docs:    docs,
		certs:   certs,
		ocr:     ocrProvider,
		audit:   audit,
		dataDir: dataDir,
	}
}

// UploadDocument validates and stores a new Potvrzení document. Storage layout:
// `DataDir/employment_docs/{year}/{uuid}_{filename}`. The row is persisted with
// `extraction_status='pending'`; ExtractDocument is the next step.
func (s *EmploymentCertificateService) UploadDocument(
	ctx context.Context,
	year int,
	kind, filename, contentType string,
	content io.Reader,
) (*domain.EmploymentDocument, error) {
	if year < 2000 || year > 2100 {
		return nil, fmt.Errorf("year must be between 2000 and 2100, got %d: %w", year, domain.ErrInvalidInput)
	}

	docKind := domain.EmploymentDocumentKind(kind)
	if docKind == "" {
		docKind = domain.EmploymentDocAdvance
	}
	switch docKind {
	case domain.EmploymentDocAdvance, domain.EmploymentDocWithholding, domain.EmploymentDocBonus:
		// ok
	default:
		return nil, fmt.Errorf("kind %q is not valid; allowed: advance, withholding, bonus: %w", kind, domain.ErrInvalidInput)
	}

	if !employmentAllowedContentTypes[contentType] {
		return nil, fmt.Errorf("content type %q is not allowed; allowed types: application/pdf, image/jpeg, image/png, image/webp: %w", contentType, domain.ErrInvalidInput)
	}

	filename = sanitizeFilename(filename)
	if filename == "" {
		return nil, fmt.Errorf("filename is required: %w", domain.ErrInvalidInput)
	}

	// Read file with size guard: read one extra byte to detect overflow.
	limited := io.LimitReader(content, employmentMaxDocumentSize+1)
	fileBytes, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("reading uploaded employment document: %w", err)
	}
	if int64(len(fileBytes)) > employmentMaxDocumentSize {
		return nil, fmt.Errorf("file size exceeds maximum of %d MB: %w", employmentMaxDocumentSize>>20, domain.ErrInvalidInput)
	}

	// Detect actual content type from file bytes to defeat MIME spoofing.
	detectedType := http.DetectContentType(fileBytes)
	if idx := strings.IndexByte(detectedType, ';'); idx != -1 {
		detectedType = strings.TrimSpace(detectedType[:idx])
	}
	if detectedType == "application/octet-stream" {
		detectedType = detectByMagicBytes(fileBytes, contentType)
	}
	if !employmentAllowedContentTypes[detectedType] {
		return nil, fmt.Errorf("detected content type %q is not allowed: %w", detectedType, domain.ErrInvalidInput)
	}
	contentType = detectedType

	// {dataDir}/employment_docs/{year}/{uuid}_{filename}
	storageDir := filepath.Join(s.dataDir, "employment_docs", fmt.Sprintf("%d", year))
	if err := os.MkdirAll(storageDir, 0750); err != nil {
		return nil, fmt.Errorf("creating employment document storage directory: %w", err)
	}

	storageName := uuid.New().String() + "_" + filename
	storagePath := filepath.Join(storageDir, storageName)

	if err := os.WriteFile(storagePath, fileBytes, 0640); err != nil {
		return nil, fmt.Errorf("writing employment document to disk: %w", err)
	}

	doc := &domain.EmploymentDocument{
		Year:             year,
		Kind:             docKind,
		Filename:         filename,
		ContentType:      contentType,
		StoragePath:      storagePath,
		Size:             int64(len(fileBytes)),
		ExtractionStatus: domain.ExtractionPending,
	}

	if err := s.docs.Create(ctx, doc); err != nil {
		// Clean up file if DB write fails.
		_ = os.Remove(storagePath)
		return nil, fmt.Errorf("saving employment document record: %w", err)
	}

	if s.audit != nil {
		s.audit.Log(ctx, "employment_document", doc.ID, "create", nil, map[string]any{
			"id":           doc.ID,
			"year":         doc.Year,
			"kind":         string(doc.Kind),
			"filename":     doc.Filename,
			"content_type": doc.ContentType,
		})
	}

	return doc, nil
}

// ExtractDocument runs the AI provider against an uploaded Potvrzení and
// persists a draft EmploymentCertificate. On success the document's
// extraction_status is set to "extracted"; on any failure it is set to
// "failed" with the error message recorded.
func (s *EmploymentCertificateService) ExtractDocument(ctx context.Context, docID int64) (*domain.EmploymentCertificate, error) {
	if docID == 0 {
		return nil, fmt.Errorf("document ID is required: %w", domain.ErrInvalidInput)
	}
	if s.ocr == nil {
		return nil, fmt.Errorf("OCR provider is not configured: %w", domain.ErrInvalidInput)
	}

	doc, err := s.docs.GetByID(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("fetching employment document for extraction: %w", err)
	}

	if !employmentOCRSupportedContentTypes[doc.ContentType] {
		return nil, fmt.Errorf("document content type %q is not supported for extraction; supported: application/pdf, image/jpeg, image/png: %w", doc.ContentType, domain.ErrInvalidInput)
	}

	// Validate path is inside our data dir before reading (mirrors investment flow).
	expectedPrefix := filepath.Join(s.dataDir, "employment_docs") + string(filepath.Separator)
	absPath, err := filepath.EvalSymlinks(doc.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("invalid storage path: %w", err)
	}
	if !strings.HasPrefix(absPath, expectedPrefix) {
		return nil, fmt.Errorf("document storage path is outside allowed directory: %w", domain.ErrInvalidInput)
	}

	fileData, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading employment document file: %w", err)
	}

	rawResponse, err := s.ocr.ProcessWithPrompt(
		ctx,
		fileData,
		doc.ContentType,
		ocr.EmploymentSystemPrompt(),
		ocr.EmploymentUserPrompt(),
	)
	if err != nil {
		_ = s.docs.UpdateExtraction(ctx, docID, domain.ExtractionFailed, err.Error())
		return nil, fmt.Errorf("AI processing failed for employment document: %w", err)
	}

	parsed, err := ocr.ParseEmploymentResponse(rawResponse)
	if err != nil {
		_ = s.docs.UpdateExtraction(ctx, docID, domain.ExtractionFailed, err.Error())
		return nil, fmt.Errorf("parsing employment AI response: %w", err)
	}

	cert, err := s.buildCertificateFromExtraction(parsed, doc)
	if err != nil {
		_ = s.docs.UpdateExtraction(ctx, docID, domain.ExtractionFailed, err.Error())
		return nil, fmt.Errorf("building employment certificate from extraction: %w", err)
	}

	if err := s.validateCertificate(cert); err != nil {
		_ = s.docs.UpdateExtraction(ctx, docID, domain.ExtractionFailed, err.Error())
		return nil, fmt.Errorf("validating extracted employment certificate: %w", err)
	}

	if err := s.certs.Create(ctx, cert); err != nil {
		_ = s.docs.UpdateExtraction(ctx, docID, domain.ExtractionFailed, err.Error())
		return nil, fmt.Errorf("creating employment certificate from extraction: %w", err)
	}

	if err := s.docs.UpdateExtraction(ctx, docID, domain.ExtractionExtracted, ""); err != nil {
		return nil, fmt.Errorf("updating employment document extraction status: %w", err)
	}

	if s.audit != nil {
		s.audit.Log(ctx, "employment_certificate", cert.ID, "create", nil, map[string]any{
			"id":               cert.ID,
			"year":             cert.Year,
			"document_id":      cert.DocumentID,
			"certificate_type": string(cert.CertificateType),
			"employer_ico":     cert.EmployerICO,
			"source":           "ocr",
		})
	}

	return cert, nil
}

// buildCertificateFromExtraction maps the OCR response into a domain certificate.
// CZK floats are converted to halere via ocr.CzkToHalere (already used by
// investment_extraction_svc.go). The year is derived from period_from / the
// document; period dates are parsed with the helper here so we can fail fast
// with ErrInvalidInput instead of a generic time.Parse error.
func (s *EmploymentCertificateService) buildCertificateFromExtraction(
	parsed *ocr.EmploymentExtractionResponse,
	doc *domain.EmploymentDocument,
) (*domain.EmploymentCertificate, error) {
	periodFrom, err := time.Parse(time.DateOnly, parsed.PeriodFrom)
	if err != nil {
		return nil, fmt.Errorf("parsing period_from %q: %w", parsed.PeriodFrom, domain.ErrInvalidInput)
	}
	periodTo, err := time.Parse(time.DateOnly, parsed.PeriodTo)
	if err != nil {
		return nil, fmt.Errorf("parsing period_to %q: %w", parsed.PeriodTo, domain.ErrInvalidInput)
	}

	certType := domain.CertificateType(parsed.CertificateType)
	switch certType {
	case domain.CertificateAdvance, domain.CertificateWithholding:
		// ok
	default:
		// Default to advance for unknown values rather than rejecting outright;
		// the user can edit before confirming.
		certType = domain.CertificateAdvance
	}

	contractType := domain.ContractType(parsed.ContractType)
	switch contractType {
	case domain.ContractDPC, domain.ContractDPP, domain.ContractHPP, domain.ContractOther:
		// ok
	default:
		contractType = domain.ContractOther
	}

	docID := doc.ID
	cert := &domain.EmploymentCertificate{
		Year:                    doc.Year,
		DocumentID:              &docID,
		CertificateType:         certType,
		EmployerName:            parsed.EmployerName,
		EmployerICO:             parsed.EmployerICO,
		EmployerAddress:         parsed.EmployerAddress,
		ContractType:            contractType,
		PeriodFrom:              periodFrom,
		PeriodTo:                periodTo,
		GrossIncome:             domain.Amount(ocr.CzkToHalere(parsed.GrossIncomeCZK)),
		IncomeWithoutAdvance:    domain.Amount(ocr.CzkToHalere(parsed.IncomeWithoutAdvanceCZK)),
		ForeignTaxPaid:          domain.Amount(ocr.CzkToHalere(parsed.ForeignTaxPaidCZK)),
		AdvanceTaxWithheld:      domain.Amount(ocr.CzkToHalere(parsed.AdvanceTaxWithheldCZK)),
		AnnualSettlementRefund:  domain.Amount(ocr.CzkToHalere(parsed.AnnualSettlementRefundCZK)),
		MonthlyBonusPaid:        domain.Amount(ocr.CzkToHalere(parsed.MonthlyBonusPaidCZK)),
		WithheldFinalTax:        domain.Amount(ocr.CzkToHalere(parsed.WithheldFinalTaxCZK)),
		IncludeWithholdingInDAP: false,
		Status:                  employmentCertificateStatusDraft,
	}
	return cert, nil
}

// Create validates and persists a manually-entered certificate.
func (s *EmploymentCertificateService) Create(ctx context.Context, cert *domain.EmploymentCertificate) error {
	if cert == nil {
		return fmt.Errorf("certificate is required: %w", domain.ErrInvalidInput)
	}
	if cert.Status == "" {
		cert.Status = employmentCertificateStatusDraft
	}
	if err := s.validateCertificate(cert); err != nil {
		return err
	}
	if err := s.certs.Create(ctx, cert); err != nil {
		return fmt.Errorf("creating employment certificate: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "employment_certificate", cert.ID, "create", nil, map[string]any{
			"id":               cert.ID,
			"year":             cert.Year,
			"certificate_type": string(cert.CertificateType),
			"employer_ico":     cert.EmployerICO,
			"source":           "manual",
		})
	}
	return nil
}

// Update modifies an existing draft certificate. Confirmed certificates are
// locked — caller must Delete and re-Create for any change.
func (s *EmploymentCertificateService) Update(ctx context.Context, cert *domain.EmploymentCertificate) error {
	if cert == nil || cert.ID == 0 {
		return fmt.Errorf("certificate ID is required: %w", domain.ErrInvalidInput)
	}
	existing, err := s.certs.GetByID(ctx, cert.ID)
	if err != nil {
		return fmt.Errorf("fetching employment certificate for update: %w", err)
	}
	if existing.Status != employmentCertificateStatusDraft {
		return fmt.Errorf("only draft certificates can be updated, status=%q: %w", existing.Status, domain.ErrInvalidInput)
	}

	// Preserve immutable fields.
	cert.CreatedAt = existing.CreatedAt
	if cert.Status == "" {
		cert.Status = existing.Status
	}
	if err := s.validateCertificate(cert); err != nil {
		return err
	}
	if err := s.certs.Update(ctx, cert); err != nil {
		return fmt.Errorf("updating employment certificate: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "employment_certificate", cert.ID, "update", existing, cert)
	}
	return nil
}

// Confirm flips status from draft to confirmed. Confirmed certificates feed the
// §6 aggregation in IncomeTaxReturnService.Recalculate.
func (s *EmploymentCertificateService) Confirm(ctx context.Context, certID int64) error {
	if certID == 0 {
		return fmt.Errorf("certificate ID is required: %w", domain.ErrInvalidInput)
	}
	cert, err := s.certs.GetByID(ctx, certID)
	if err != nil {
		return fmt.Errorf("fetching employment certificate for confirm: %w", err)
	}
	if cert.Status == employmentCertificateStatusConfirmed {
		return nil
	}
	cert.Status = employmentCertificateStatusConfirmed
	if err := s.certs.Update(ctx, cert); err != nil {
		return fmt.Errorf("confirming employment certificate: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "employment_certificate", certID, "confirm", nil, map[string]any{
			"status": cert.Status,
		})
	}
	return nil
}

// ListByYear returns all non-deleted certificates for a year.
func (s *EmploymentCertificateService) ListByYear(ctx context.Context, year int) ([]*domain.EmploymentCertificate, error) {
	if year < 2000 || year > 2100 {
		return nil, fmt.Errorf("year must be between 2000 and 2100, got %d: %w", year, domain.ErrInvalidInput)
	}
	certs, err := s.certs.ListByYear(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("listing employment certificates: %w", err)
	}
	return certs, nil
}

// Get returns a single certificate by ID.
func (s *EmploymentCertificateService) Get(ctx context.Context, certID int64) (*domain.EmploymentCertificate, error) {
	if certID == 0 {
		return nil, fmt.Errorf("certificate ID is required: %w", domain.ErrInvalidInput)
	}
	cert, err := s.certs.GetByID(ctx, certID)
	if err != nil {
		return nil, fmt.Errorf("fetching employment certificate: %w", err)
	}
	return cert, nil
}

// Delete soft-deletes a certificate.
func (s *EmploymentCertificateService) Delete(ctx context.Context, certID int64) error {
	if certID == 0 {
		return fmt.Errorf("certificate ID is required: %w", domain.ErrInvalidInput)
	}
	if err := s.certs.Delete(ctx, certID); err != nil {
		return fmt.Errorf("deleting employment certificate: %w", err)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "employment_certificate", certID, "delete", nil, nil)
	}
	return nil
}

// ListDocumentsByYear returns all uploaded employment documents for a year.
func (s *EmploymentCertificateService) ListDocumentsByYear(ctx context.Context, year int) ([]*domain.EmploymentDocument, error) {
	if year < 2000 || year > 2100 {
		return nil, fmt.Errorf("year must be between 2000 and 2100, got %d: %w", year, domain.ErrInvalidInput)
	}
	docs, err := s.docs.ListByYear(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("listing employment documents: %w", err)
	}
	return docs, nil
}

// GetDocument returns a single employment document by ID.
func (s *EmploymentCertificateService) GetDocument(ctx context.Context, docID int64) (*domain.EmploymentDocument, error) {
	if docID == 0 {
		return nil, fmt.Errorf("document ID is required: %w", domain.ErrInvalidInput)
	}
	doc, err := s.docs.GetByID(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("fetching employment document: %w", err)
	}
	return doc, nil
}

// DeleteDocument removes the document row and best-effort removes the file from disk.
// Linked certificates have their document_id set to NULL via FK ON DELETE SET NULL.
func (s *EmploymentCertificateService) DeleteDocument(ctx context.Context, docID int64) error {
	if docID == 0 {
		return fmt.Errorf("document ID is required: %w", domain.ErrInvalidInput)
	}
	doc, err := s.docs.GetByID(ctx, docID)
	if err != nil {
		return fmt.Errorf("fetching employment document for delete: %w", err)
	}
	if err := s.docs.Delete(ctx, docID); err != nil {
		return fmt.Errorf("deleting employment document record: %w", err)
	}
	if doc.StoragePath != "" {
		_ = os.Remove(doc.StoragePath)
	}
	if s.audit != nil {
		s.audit.Log(ctx, "employment_document", docID, "delete", nil, nil)
	}
	return nil
}

// GetDocumentFilePath returns the on-disk path and content type for serving a
// document, validating the storage path stays inside the configured data dir
// to defeat path traversal / symlink escapes (mirror of investment flow).
func (s *EmploymentCertificateService) GetDocumentFilePath(ctx context.Context, docID int64) (string, string, error) {
	if docID == 0 {
		return "", "", fmt.Errorf("document ID is required: %w", domain.ErrInvalidInput)
	}
	doc, err := s.docs.GetByID(ctx, docID)
	if err != nil {
		return "", "", fmt.Errorf("fetching employment document for file path: %w", err)
	}
	expectedPrefix := filepath.Join(s.dataDir, "employment_docs") + string(filepath.Separator)
	absPath, err := filepath.EvalSymlinks(doc.StoragePath)
	if err != nil {
		return "", "", fmt.Errorf("invalid storage path: %w", err)
	}
	if !strings.HasPrefix(absPath, expectedPrefix) {
		return "", "", fmt.Errorf("document storage path is outside allowed directory: %w", domain.ErrInvalidInput)
	}
	return absPath, doc.ContentType, nil
}

// validateCertificate enforces the rules from RFC-016 §Service.
func (s *EmploymentCertificateService) validateCertificate(cert *domain.EmploymentCertificate) error {
	if cert.Year < 2000 || cert.Year > 2100 {
		return fmt.Errorf("year must be between 2000 and 2100, got %d: %w", cert.Year, domain.ErrInvalidInput)
	}
	if strings.TrimSpace(cert.EmployerName) == "" {
		return fmt.Errorf("employer name is required: %w", domain.ErrInvalidInput)
	}
	if cert.EmployerICO != "" {
		if err := domain.ValidateICO(cert.EmployerICO); err != nil {
			return fmt.Errorf("employer IČO is invalid: %w", err)
		}
	}
	switch cert.CertificateType {
	case domain.CertificateAdvance, domain.CertificateWithholding:
		// ok
	default:
		return fmt.Errorf("certificate_type must be advance or withholding, got %q: %w", cert.CertificateType, domain.ErrInvalidInput)
	}
	switch cert.ContractType {
	case domain.ContractDPC, domain.ContractDPP, domain.ContractHPP, domain.ContractOther, "":
		// ok ("" gets defaulted by repo)
	default:
		return fmt.Errorf("contract_type must be dpc/dpp/hpp/other, got %q: %w", cert.ContractType, domain.ErrInvalidInput)
	}
	if cert.PeriodFrom.IsZero() || cert.PeriodTo.IsZero() {
		return fmt.Errorf("period_from and period_to are required: %w", domain.ErrInvalidInput)
	}
	if cert.PeriodFrom.After(cert.PeriodTo) {
		return fmt.Errorf("period_from must be on or before period_to: %w", domain.ErrInvalidInput)
	}
	if cert.PeriodFrom.Year() != cert.Year || cert.PeriodTo.Year() != cert.Year {
		return fmt.Errorf("period_from and period_to must fall within year %d: %w", cert.Year, domain.ErrInvalidInput)
	}
	for name, amt := range map[string]domain.Amount{
		"gross_income":             cert.GrossIncome,
		"income_without_advance":   cert.IncomeWithoutAdvance,
		"foreign_tax_paid":         cert.ForeignTaxPaid,
		"advance_tax_withheld":     cert.AdvanceTaxWithheld,
		"annual_settlement_refund": cert.AnnualSettlementRefund,
		"monthly_bonus_paid":       cert.MonthlyBonusPaid,
		"withheld_final_tax":       cert.WithheldFinalTax,
	} {
		if amt < 0 {
			return fmt.Errorf("%s must be non-negative, got %d: %w", name, amt, domain.ErrInvalidInput)
		}
	}
	if cert.WithheldFinalTax > 0 && cert.CertificateType != domain.CertificateWithholding {
		return fmt.Errorf("withheld_final_tax > 0 only allowed for withholding certificates: %w", domain.ErrInvalidInput)
	}
	if cert.IncludeWithholdingInDAP && cert.CertificateType != domain.CertificateWithholding {
		return fmt.Errorf("include_withholding_in_dap=true only allowed for withholding certificates: %w", domain.ErrInvalidInput)
	}
	if cert.AnnualSettlementRefund > cert.AdvanceTaxWithheld {
		return fmt.Errorf("annual_settlement_refund cannot exceed advance_tax_withheld: %w", domain.ErrInvalidInput)
	}
	return nil
}
