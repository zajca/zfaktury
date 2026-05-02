package service

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service/ocr"
	"github.com/zajca/zfaktury/internal/testutil"
)

// employmentMockOCRProvider implements ocr.Provider for §6 extraction tests.
type employmentMockOCRProvider struct {
	promptResponse string
	promptErr      error
}

func (m *employmentMockOCRProvider) ProcessImage(_ context.Context, _ []byte, _ string) (*domain.OCRResult, error) {
	return nil, nil
}

func (m *employmentMockOCRProvider) ProcessWithPrompt(_ context.Context, _ []byte, _ string, _, _ string) (string, error) {
	return m.promptResponse, m.promptErr
}

func (m *employmentMockOCRProvider) Name() string {
	return "employment-mock"
}

var _ ocr.Provider = (*employmentMockOCRProvider)(nil)

// newEmploymentCertSvc wires the service against real SQLite for tests.
func newEmploymentCertSvc(t *testing.T, provider ocr.Provider) (
	*EmploymentCertificateService,
	*repository.EmploymentDocumentRepository,
	*repository.EmploymentCertificateRepository,
	string,
) {
	t.Helper()
	db := testutil.NewTestDB(t)
	docRepo := repository.NewEmploymentDocumentRepository(db)
	certRepo := repository.NewEmploymentCertificateRepository(db)
	dataDir := t.TempDir()
	svc := NewEmploymentCertificateService(docRepo, certRepo, provider, nil, dataDir)
	return svc, docRepo, certRepo, dataDir
}

const mockEmploymentAdvanceJSON = `{
	"certificate_type": "advance",
	"employer_name": "Acme s.r.o.",
	"employer_ico": "27082440",
	"employer_address": "Praha 1",
	"contract_type": "dpc",
	"period_from": "2025-01-01",
	"period_to": "2025-12-31",
	"gross_income_czk": 120000.0,
	"income_without_advance_czk": 0.0,
	"foreign_tax_paid_czk": 0.0,
	"advance_tax_withheld_czk": 18000.0,
	"annual_settlement_refund_czk": 0.0,
	"monthly_bonus_paid_czk": 0.0,
	"withheld_final_tax_czk": 0.0,
	"confidence": 0.95,
	"raw_text": "Potvrzeni..."
}`

// validCert returns a baseline confirmed-ready advance certificate.
func validCert() *domain.EmploymentCertificate {
	return &domain.EmploymentCertificate{
		Year:                    2025,
		CertificateType:         domain.CertificateAdvance,
		ContractType:            domain.ContractDPC,
		EmployerName:            "Acme s.r.o.",
		EmployerICO:             "27082440",
		EmployerAddress:         "Praha 1",
		PeriodFrom:              time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodTo:                time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		GrossIncome:             domain.NewAmount(120_000, 0),
		AdvanceTaxWithheld:      domain.NewAmount(18_000, 0),
		IncludeWithholdingInDAP: false,
		Status:                  employmentCertificateStatusDraft,
	}
}

func TestEmploymentCertSvc_UploadDocument_HappyPath(t *testing.T) {
	svc, docRepo, _, dataDir := newEmploymentCertSvc(t, nil)
	ctx := context.Background()

	doc, err := svc.UploadDocument(ctx, 2025, "advance", "potv.pdf", "application/pdf", bytes.NewReader(pdfMagic))
	if err != nil {
		t.Fatalf("UploadDocument: %v", err)
	}
	if doc.ID == 0 {
		t.Error("expected non-zero document ID")
	}
	if doc.Year != 2025 {
		t.Errorf("Year = %d, want 2025", doc.Year)
	}
	if doc.Kind != domain.EmploymentDocAdvance {
		t.Errorf("Kind = %q, want %q", doc.Kind, domain.EmploymentDocAdvance)
	}
	if doc.ExtractionStatus != domain.ExtractionPending {
		t.Errorf("ExtractionStatus = %q, want pending", doc.ExtractionStatus)
	}
	// File written under {dataDir}/employment_docs/{year}/...
	expectedPrefix := filepath.Join(dataDir, "employment_docs", "2025") + string(filepath.Separator)
	if !strings.HasPrefix(doc.StoragePath, expectedPrefix) {
		t.Errorf("StoragePath = %q does not start with %q", doc.StoragePath, expectedPrefix)
	}
	// Reload to verify persistence.
	got, err := docRepo.GetByID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Filename != "potv.pdf" {
		t.Errorf("Filename = %q, want potv.pdf", got.Filename)
	}
}

func TestEmploymentCertSvc_UploadDocument_DefaultsKindToAdvance(t *testing.T) {
	svc, _, _, _ := newEmploymentCertSvc(t, nil)
	doc, err := svc.UploadDocument(context.Background(), 2025, "", "p.pdf", "application/pdf", bytes.NewReader(pdfMagic))
	if err != nil {
		t.Fatalf("UploadDocument: %v", err)
	}
	if doc.Kind != domain.EmploymentDocAdvance {
		t.Errorf("Kind = %q, want %q", doc.Kind, domain.EmploymentDocAdvance)
	}
}

func TestEmploymentCertSvc_UploadDocument_RejectsTextPlain(t *testing.T) {
	svc, _, _, _ := newEmploymentCertSvc(t, nil)
	_, err := svc.UploadDocument(context.Background(), 2025, "advance", "fake.pdf", "text/plain", bytes.NewReader([]byte("not a pdf")))
	if err == nil {
		t.Fatal("expected error for text/plain content type")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestEmploymentCertSvc_UploadDocument_RejectsOversizedFile(t *testing.T) {
	svc, _, _, _ := newEmploymentCertSvc(t, nil)
	oversized := bytes.NewReader(bytes.Repeat([]byte("a"), employmentMaxDocumentSize+1))
	_, err := svc.UploadDocument(context.Background(), 2025, "advance", "big.pdf", "application/pdf", oversized)
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestEmploymentCertSvc_UploadDocument_RejectsInvalidYear(t *testing.T) {
	svc, _, _, _ := newEmploymentCertSvc(t, nil)
	_, err := svc.UploadDocument(context.Background(), 1999, "advance", "p.pdf", "application/pdf", bytes.NewReader(pdfMagic))
	if err == nil {
		t.Fatal("expected error for invalid year")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestEmploymentCertSvc_UploadDocument_RejectsInvalidKind(t *testing.T) {
	svc, _, _, _ := newEmploymentCertSvc(t, nil)
	_, err := svc.UploadDocument(context.Background(), 2025, "garbage", "p.pdf", "application/pdf", bytes.NewReader(pdfMagic))
	if err == nil {
		t.Fatal("expected error for invalid kind")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestEmploymentCertSvc_ExtractDocument_HappyPath(t *testing.T) {
	provider := &employmentMockOCRProvider{promptResponse: mockEmploymentAdvanceJSON}
	svc, docRepo, certRepo, _ := newEmploymentCertSvc(t, provider)
	ctx := context.Background()

	doc, err := svc.UploadDocument(ctx, 2025, "advance", "p.pdf", "application/pdf", bytes.NewReader(pdfMagic))
	if err != nil {
		t.Fatalf("UploadDocument: %v", err)
	}

	cert, err := svc.ExtractDocument(ctx, doc.ID)
	if err != nil {
		t.Fatalf("ExtractDocument: %v", err)
	}
	if cert.ID == 0 {
		t.Error("expected non-zero cert ID")
	}
	if cert.Status != employmentCertificateStatusDraft {
		t.Errorf("Status = %q, want draft", cert.Status)
	}
	if cert.CertificateType != domain.CertificateAdvance {
		t.Errorf("CertificateType = %q, want advance", cert.CertificateType)
	}
	if cert.GrossIncome != domain.NewAmount(120_000, 0) {
		t.Errorf("GrossIncome = %d, want %d", cert.GrossIncome, domain.NewAmount(120_000, 0))
	}
	if cert.AdvanceTaxWithheld != domain.NewAmount(18_000, 0) {
		t.Errorf("AdvanceTaxWithheld = %d, want %d", cert.AdvanceTaxWithheld, domain.NewAmount(18_000, 0))
	}
	if cert.DocumentID == nil || *cert.DocumentID != doc.ID {
		t.Errorf("DocumentID = %v, want %d", cert.DocumentID, doc.ID)
	}

	// Document extraction status should now be "extracted".
	updatedDoc, err := docRepo.GetByID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updatedDoc.ExtractionStatus != domain.ExtractionExtracted {
		t.Errorf("ExtractionStatus = %q, want extracted", updatedDoc.ExtractionStatus)
	}

	// Cert is persisted.
	saved, err := certRepo.GetByID(ctx, cert.ID)
	if err != nil {
		t.Fatalf("certRepo.GetByID: %v", err)
	}
	if saved.EmployerName != "Acme s.r.o." {
		t.Errorf("EmployerName = %q, want Acme s.r.o.", saved.EmployerName)
	}
}

func TestEmploymentCertSvc_ExtractDocument_ProviderError_MarksFailed(t *testing.T) {
	provider := &employmentMockOCRProvider{promptErr: errors.New("AI offline")}
	svc, docRepo, _, _ := newEmploymentCertSvc(t, provider)
	ctx := context.Background()

	doc, err := svc.UploadDocument(ctx, 2025, "advance", "p.pdf", "application/pdf", bytes.NewReader(pdfMagic))
	if err != nil {
		t.Fatalf("UploadDocument: %v", err)
	}
	if _, err := svc.ExtractDocument(ctx, doc.ID); err == nil {
		t.Fatal("expected error from provider")
	}
	updatedDoc, err := docRepo.GetByID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updatedDoc.ExtractionStatus != domain.ExtractionFailed {
		t.Errorf("ExtractionStatus = %q, want failed", updatedDoc.ExtractionStatus)
	}
	if updatedDoc.ExtractionError == "" {
		t.Error("expected ExtractionError to be set")
	}
}

func TestEmploymentCertSvc_ExtractDocument_InvalidJSON_MarksFailed(t *testing.T) {
	provider := &employmentMockOCRProvider{promptResponse: "not json"}
	svc, docRepo, certRepo, _ := newEmploymentCertSvc(t, provider)
	ctx := context.Background()

	doc, err := svc.UploadDocument(ctx, 2025, "advance", "p.pdf", "application/pdf", bytes.NewReader(pdfMagic))
	if err != nil {
		t.Fatalf("UploadDocument: %v", err)
	}
	if _, err := svc.ExtractDocument(ctx, doc.ID); err == nil {
		t.Fatal("expected error parsing invalid JSON")
	}
	updatedDoc, err := docRepo.GetByID(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updatedDoc.ExtractionStatus != domain.ExtractionFailed {
		t.Errorf("ExtractionStatus = %q, want failed", updatedDoc.ExtractionStatus)
	}
	// No certificate should have been created.
	certs, err := certRepo.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear: %v", err)
	}
	if len(certs) != 0 {
		t.Errorf("ListByYear returned %d certs, want 0 (none should be persisted on failure)", len(certs))
	}
}

func TestEmploymentCertSvc_ExtractDocument_NoOCRProvider(t *testing.T) {
	svc, _, _, _ := newEmploymentCertSvc(t, nil)
	ctx := context.Background()
	doc, err := svc.UploadDocument(ctx, 2025, "advance", "p.pdf", "application/pdf", bytes.NewReader(pdfMagic))
	if err != nil {
		t.Fatalf("UploadDocument: %v", err)
	}
	_, err = svc.ExtractDocument(ctx, doc.ID)
	if err == nil {
		t.Fatal("expected error when provider is nil")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestEmploymentCertSvc_Create_HappyPath(t *testing.T) {
	svc, _, certRepo, _ := newEmploymentCertSvc(t, nil)
	ctx := context.Background()

	cert := validCert()
	if err := svc.Create(ctx, cert); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if cert.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	got, err := certRepo.GetByID(ctx, cert.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.EmployerName != "Acme s.r.o." {
		t.Errorf("EmployerName = %q, want Acme s.r.o.", got.EmployerName)
	}
}

func TestEmploymentCertSvc_Create_RejectsBadPeriod(t *testing.T) {
	svc, _, _, _ := newEmploymentCertSvc(t, nil)

	cert := validCert()
	cert.PeriodFrom = time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)
	cert.PeriodTo = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	err := svc.Create(context.Background(), cert)
	if err == nil {
		t.Fatal("expected error: period_from > period_to")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestEmploymentCertSvc_Create_RejectsPeriodOutsideYear(t *testing.T) {
	svc, _, _, _ := newEmploymentCertSvc(t, nil)

	cert := validCert()
	cert.PeriodFrom = time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	err := svc.Create(context.Background(), cert)
	if err == nil {
		t.Fatal("expected error: period outside year")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestEmploymentCertSvc_Create_RejectsBadICO(t *testing.T) {
	svc, _, _, _ := newEmploymentCertSvc(t, nil)

	cert := validCert()
	cert.EmployerICO = "ABC123"
	err := svc.Create(context.Background(), cert)
	if err == nil {
		t.Fatal("expected error: bad ICO")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}

	cert2 := validCert()
	cert2.EmployerICO = "1234567" // 7 digits, not 8
	if err := svc.Create(context.Background(), cert2); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput for 7-digit ICO, got %v", err)
	}
}

func TestEmploymentCertSvc_Create_RejectsRefundOverWithheld(t *testing.T) {
	svc, _, _, _ := newEmploymentCertSvc(t, nil)

	cert := validCert()
	cert.AdvanceTaxWithheld = domain.NewAmount(1000, 0)
	cert.AnnualSettlementRefund = domain.NewAmount(2000, 0)
	err := svc.Create(context.Background(), cert)
	if err == nil {
		t.Fatal("expected error: refund > withheld")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestEmploymentCertSvc_Create_RejectsWithholdingFinalTaxOnAdvance(t *testing.T) {
	svc, _, _, _ := newEmploymentCertSvc(t, nil)

	cert := validCert()
	cert.CertificateType = domain.CertificateAdvance
	cert.WithheldFinalTax = domain.NewAmount(500, 0)
	err := svc.Create(context.Background(), cert)
	if err == nil {
		t.Fatal("expected error: WithheldFinalTax > 0 on advance cert")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestEmploymentCertSvc_Create_RejectsIncludeWithholdingOnAdvance(t *testing.T) {
	svc, _, _, _ := newEmploymentCertSvc(t, nil)

	cert := validCert()
	cert.CertificateType = domain.CertificateAdvance
	cert.IncludeWithholdingInDAP = true
	err := svc.Create(context.Background(), cert)
	if err == nil {
		t.Fatal("expected error: include_withholding_in_dap on advance cert")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestEmploymentCertSvc_Create_RejectsNegativeAmount(t *testing.T) {
	svc, _, _, _ := newEmploymentCertSvc(t, nil)

	cert := validCert()
	cert.GrossIncome = domain.Amount(-1)
	err := svc.Create(context.Background(), cert)
	if err == nil {
		t.Fatal("expected error: negative amount")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestEmploymentCertSvc_Create_AcceptsEmptyICO(t *testing.T) {
	// Empty ICO is allowed (e.g. zahraniční zaměstnavatel without IČO).
	svc, _, _, _ := newEmploymentCertSvc(t, nil)
	cert := validCert()
	cert.EmployerICO = ""
	if err := svc.Create(context.Background(), cert); err != nil {
		t.Fatalf("Create with empty ICO should succeed, got %v", err)
	}
}

func TestEmploymentCertSvc_Update_OnlyDraft(t *testing.T) {
	svc, _, _, _ := newEmploymentCertSvc(t, nil)
	ctx := context.Background()

	cert := validCert()
	if err := svc.Create(ctx, cert); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Update while draft: ok.
	cert.GrossIncome = domain.NewAmount(150_000, 0)
	if err := svc.Update(ctx, cert); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Confirm.
	if err := svc.Confirm(ctx, cert.ID); err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	// Update after confirm: rejected.
	confirmed, err := svc.Get(ctx, cert.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	confirmed.GrossIncome = domain.NewAmount(200_000, 0)
	err = svc.Update(ctx, confirmed)
	if err == nil {
		t.Fatal("expected error: cannot update confirmed cert")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestEmploymentCertSvc_Confirm_FlipsStatus(t *testing.T) {
	svc, _, _, _ := newEmploymentCertSvc(t, nil)
	ctx := context.Background()

	cert := validCert()
	if err := svc.Create(ctx, cert); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Confirm(ctx, cert.ID); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	got, err := svc.Get(ctx, cert.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != employmentCertificateStatusConfirmed {
		t.Errorf("Status = %q, want confirmed", got.Status)
	}
}

func TestEmploymentCertSvc_Delete_SoftDeletes(t *testing.T) {
	svc, _, _, _ := newEmploymentCertSvc(t, nil)
	ctx := context.Background()

	cert := validCert()
	if err := svc.Create(ctx, cert); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Delete(ctx, cert.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := svc.Get(ctx, cert.ID)
	if err == nil {
		t.Error("expected error fetching soft-deleted cert")
	}
}

func TestEmploymentCertSvc_ListByYear(t *testing.T) {
	svc, _, _, _ := newEmploymentCertSvc(t, nil)
	ctx := context.Background()

	// Use mod-11 valid IČOs so ValidateICO passes.
	icos := []string{"27082440", "26168685", "45274649"}
	for i := 0; i < 3; i++ {
		cert := validCert()
		cert.EmployerICO = icos[i]
		if err := svc.Create(ctx, cert); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}
	list, err := svc.ListByYear(ctx, 2025)
	if err != nil {
		t.Fatalf("ListByYear: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("ListByYear returned %d, want 3", len(list))
	}
}

// TestEmploymentCertSvc_AuditEmitsCorrectCategory verifies the service emits
// audit events with the exact category strings the lead must register.
func TestEmploymentCertSvc_AuditEmitsCorrectCategory(t *testing.T) {
	db := testutil.NewTestDB(t)
	docRepo := repository.NewEmploymentDocumentRepository(db)
	certRepo := repository.NewEmploymentCertificateRepository(db)
	auditRepo := repository.NewAuditLogRepository(db)
	auditSvc := NewAuditService(auditRepo)
	dataDir := t.TempDir()
	svc := NewEmploymentCertificateService(docRepo, certRepo, nil, auditSvc, dataDir)
	ctx := context.Background()

	doc, err := svc.UploadDocument(ctx, 2025, "advance", "p.pdf", "application/pdf", bytes.NewReader(pdfMagic))
	if err != nil {
		t.Fatalf("UploadDocument: %v", err)
	}
	cert := validCert()
	if err := svc.Create(ctx, cert); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Confirm(ctx, cert.ID); err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	docEntries, err := auditSvc.ListByEntity(ctx, "employment_document", doc.ID)
	if err != nil {
		t.Fatalf("ListByEntity employment_document: %v", err)
	}
	if len(docEntries) == 0 {
		t.Error("expected at least one employment_document audit entry")
	}

	certEntries, err := auditSvc.ListByEntity(ctx, "employment_certificate", cert.ID)
	if err != nil {
		t.Fatalf("ListByEntity employment_certificate: %v", err)
	}
	if len(certEntries) < 2 {
		t.Errorf("expected at least 2 employment_certificate audit entries (create+confirm), got %d", len(certEntries))
	}
}
