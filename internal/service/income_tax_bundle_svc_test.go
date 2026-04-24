package service

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

// bundleTestEnv wires up the full set of dependencies for exercising the
// IncomeTaxBundleService against a real in-memory SQLite database plus a
// temporary directory rooted at dataDir/tax-documents.
type bundleTestEnv struct {
	db          *sql.DB
	dataDir     string
	itrRepo     repository.IncomeTaxReturnRepo
	deductRepo  repository.TaxDeductionRepo
	deductDoc   repository.TaxDeductionDocumentRepo
	itrSvc      *IncomeTaxReturnService
	docSvc      *TaxDeductionDocumentService
	bundleSvc   *IncomeTaxBundleService
	settingsRep repository.SettingsRepo
}

func setupBundleEnv(t *testing.T) *bundleTestEnv {
	t.Helper()
	db := testutil.NewTestDB(t)
	dataDir := t.TempDir()

	itrRepo := repository.NewIncomeTaxReturnRepository(db)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	setRepo := repository.NewSettingsRepository(db)
	tysRepo := repository.NewTaxYearSettingsRepository(db)
	tpRepo := repository.NewTaxPrepaymentRepository(db)
	deductRepo := repository.NewTaxDeductionRepository(db)
	deductDocRepo := repository.NewTaxDeductionDocumentRepository(db)

	itrSvc := NewIncomeTaxReturnService(
		itrRepo, invRepo, expRepo, setRepo, tysRepo, tpRepo, nil, nil,
	)
	docSvc := NewTaxDeductionDocumentService(deductDocRepo, deductRepo, dataDir, nil)
	bundleSvc := NewIncomeTaxBundleService(
		itrRepo, deductRepo, deductDocRepo,
		itrSvc, docSvc, nil, nil, dataDir,
	)

	return &bundleTestEnv{
		db:          db,
		dataDir:     dataDir,
		itrRepo:     itrRepo,
		deductRepo:  deductRepo,
		deductDoc:   deductDocRepo,
		itrSvc:      itrSvc,
		docSvc:      docSvc,
		bundleSvc:   bundleSvc,
		settingsRep: setRepo,
	}
}

// seedITR creates an income_tax_return row with XML pre-populated so the
// bundle service does not attempt to regenerate (which would require more
// settings wiring than these unit tests care about).
func (e *bundleTestEnv) seedITR(t *testing.T, year int, xml []byte) *domain.IncomeTaxReturn {
	t.Helper()
	itr := &domain.IncomeTaxReturn{
		Year:       year,
		FilingType: domain.FilingTypeRegular,
		Status:     domain.FilingStatusDraft,
		XMLData:    xml,
	}
	if err := e.itrRepo.Create(context.Background(), itr); err != nil {
		t.Fatalf("creating income_tax_return: %v", err)
	}
	// Create() does not persist XMLData; follow-up Update pushes it.
	if err := e.itrRepo.Update(context.Background(), itr); err != nil {
		t.Fatalf("updating income_tax_return xml: %v", err)
	}
	return itr
}

func (e *bundleTestEnv) seedDeduction(t *testing.T, year int, category string) *domain.TaxDeduction {
	t.Helper()
	ded := &domain.TaxDeduction{
		Year:          year,
		Category:      category,
		Description:   "test",
		ClaimedAmount: domain.NewAmount(1000, 0),
		MaxAmount:     domain.NewAmount(150000, 0),
		AllowedAmount: domain.NewAmount(1000, 0),
	}
	if err := e.deductRepo.Create(context.Background(), ded); err != nil {
		t.Fatalf("creating deduction: %v", err)
	}
	return ded
}

// seedDeductionDoc creates a document record AND writes the corresponding
// file to disk inside dataDir/tax-documents/{deductionID}/. Returns the
// absolute storage path so tests can optionally remove it.
func (e *bundleTestEnv) seedDeductionDoc(t *testing.T, deductionID int64, filename string, payload []byte) (*domain.TaxDeductionDocument, string) {
	t.Helper()
	storageDir := filepath.Join(e.dataDir, "tax-documents", fmt.Sprintf("%d", deductionID))
	if err := os.MkdirAll(storageDir, 0750); err != nil {
		t.Fatalf("mkdir storage: %v", err)
	}
	storagePath := filepath.Join(storageDir, "uuid_"+filename)
	if err := os.WriteFile(storagePath, payload, 0640); err != nil {
		t.Fatalf("writing doc file: %v", err)
	}
	doc := &domain.TaxDeductionDocument{
		TaxDeductionID: deductionID,
		Filename:       filename,
		ContentType:    "application/pdf",
		StoragePath:    storagePath,
		Size:           int64(len(payload)),
	}
	if err := e.deductDoc.Create(context.Background(), doc); err != nil {
		t.Fatalf("creating deduction doc: %v", err)
	}
	return doc, storagePath
}

// readZip parses the ZIP bytes into a name -> content map for assertions.
func readZip(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("opening zip: %v", err)
	}
	out := make(map[string][]byte, len(zr.File))
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("opening zip entry %q: %v", f.Name, err)
		}
		b, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("reading zip entry %q: %v", f.Name, err)
		}
		out[f.Name] = b
	}
	return out
}

func TestBundleService_HappyPath(t *testing.T) {
	env := setupBundleEnv(t)

	xmlBytes := []byte(`<?xml version="1.0"?><Pisemnost/>`)
	itr := env.seedITR(t, 2025, xmlBytes)
	ded := env.seedDeduction(t, 2025, domain.DeductionMortgage)
	docPayload := []byte("PDF-MOCK-CONTENT")
	doc, _ := env.seedDeductionDoc(t, ded.ID, "potvrzeni.pdf", docPayload)

	data, filename, err := env.bundleSvc.GenerateBundle(context.Background(), itr.ID)
	if err != nil {
		t.Fatalf("GenerateBundle: %v", err)
	}
	if filename != "priznani-2025.zip" {
		t.Errorf("filename = %q, want %q", filename, "priznani-2025.zip")
	}

	entries := readZip(t, data)

	// XML at archive root.
	gotXML, ok := entries["priznani-2025.xml"]
	if !ok {
		t.Fatalf("expected priznani-2025.xml in archive, got keys: %v", keys(entries))
	}
	if !bytes.Equal(gotXML, xmlBytes) {
		t.Errorf("xml bytes mismatch: got %q, want %q", gotXML, xmlBytes)
	}

	// Deduction document with expected prefix + deduction ID + sanitized filename.
	wantEntry := fmt.Sprintf("prilohy/odpocty/uroky-hypoteka-%d-potvrzeni.pdf", ded.ID)
	gotDoc, ok := entries[wantEntry]
	if !ok {
		t.Fatalf("expected %q in archive, got keys: %v", wantEntry, keys(entries))
	}
	if !bytes.Equal(gotDoc, docPayload) {
		t.Errorf("doc bytes mismatch")
	}
	// No missing-files note on a healthy run.
	if _, found := entries["CHYBEJICI-SOUBORY.txt"]; found {
		t.Error("unexpected CHYBEJICI-SOUBORY.txt in healthy bundle")
	}

	// suppress unused warning for doc
	_ = doc
}

func TestBundleService_NoDeductions(t *testing.T) {
	env := setupBundleEnv(t)

	xmlBytes := []byte(`<?xml version="1.0"?><Pisemnost/>`)
	itr := env.seedITR(t, 2024, xmlBytes)

	data, _, err := env.bundleSvc.GenerateBundle(context.Background(), itr.ID)
	if err != nil {
		t.Fatalf("GenerateBundle: %v", err)
	}
	entries := readZip(t, data)

	if len(entries) != 1 {
		t.Errorf("expected exactly 1 entry in ZIP, got %d: %v", len(entries), keys(entries))
	}
	if _, ok := entries["priznani-2024.xml"]; !ok {
		t.Errorf("missing XML in archive, keys: %v", keys(entries))
	}
}

func TestBundleService_MissingFileOnDisk(t *testing.T) {
	env := setupBundleEnv(t)

	xmlBytes := []byte(`<?xml version="1.0"?><Pisemnost/>`)
	itr := env.seedITR(t, 2025, xmlBytes)
	ded := env.seedDeduction(t, 2025, domain.DeductionLifeInsurance)

	_, storagePath := env.seedDeductionDoc(t, ded.ID, "pojistka.pdf", []byte("PDF"))
	// Simulate the file being removed after registration (user deleted, rsync
	// lost it, etc). The bundle must still succeed and note the gap.
	if err := os.Remove(storagePath); err != nil {
		t.Fatalf("removing seeded doc: %v", err)
	}

	data, _, err := env.bundleSvc.GenerateBundle(context.Background(), itr.ID)
	if err != nil {
		t.Fatalf("GenerateBundle: %v", err)
	}
	entries := readZip(t, data)

	if _, ok := entries["priznani-2025.xml"]; !ok {
		t.Errorf("missing XML in archive")
	}
	note, ok := entries["CHYBEJICI-SOUBORY.txt"]
	if !ok {
		t.Fatalf("expected CHYBEJICI-SOUBORY.txt when a file is missing, keys: %v", keys(entries))
	}
	if !strings.Contains(string(note), "pojistka.pdf") {
		t.Errorf("missing-files note does not mention the missing filename: %s", note)
	}
	// The document itself should NOT be in the archive.
	for name := range entries {
		if strings.HasPrefix(name, "prilohy/odpocty/") {
			t.Errorf("unexpected deduction doc in archive: %s", name)
		}
	}
}

func TestBundleService_InvalidID(t *testing.T) {
	env := setupBundleEnv(t)

	_, _, err := env.bundleSvc.GenerateBundle(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error for zero ID")
	}
}

func TestBundleService_FilenameSanitization(t *testing.T) {
	env := setupBundleEnv(t)

	xmlBytes := []byte(`<?xml version="1.0"?><Pisemnost/>`)
	itr := env.seedITR(t, 2025, xmlBytes)
	ded := env.seedDeduction(t, 2025, domain.DeductionDonation)

	// A filename containing path separators must not escape the prilohy/odpocty/ segment.
	_, _ = env.seedDeductionDoc(t, ded.ID, "nasty_name.pdf", []byte("X"))

	// Stuff a record with an embedded path separator in the Filename column after the fact
	// to exercise sanitization of already-stored names.
	_, err := env.db.Exec(`UPDATE tax_deduction_documents SET filename = ? WHERE tax_deduction_id = ?`,
		"../../etc/passwd", ded.ID)
	if err != nil {
		t.Fatalf("tainting filename: %v", err)
	}

	data, _, err := env.bundleSvc.GenerateBundle(context.Background(), itr.ID)
	if err != nil {
		t.Fatalf("GenerateBundle: %v", err)
	}
	entries := readZip(t, data)
	for name := range entries {
		if strings.Contains(name, "..") {
			t.Errorf("zip entry contains traversal: %q", name)
		}
		if strings.HasPrefix(name, "/") {
			t.Errorf("zip entry is absolute: %q", name)
		}
	}
}

func TestBundleService_GenerateBundle_InvalidPathRejected(t *testing.T) {
	// Ensures that documents whose storage path escapes dataDir cause the
	// bundle to fail loudly — not silently drop the entry — so the operator
	// sees that something is wrong with the stored doc. Crucially, under no
	// circumstances may the external file's content end up inside the ZIP.
	env := setupBundleEnv(t)

	xmlBytes := []byte(`<?xml version="1.0"?><Pisemnost/>`)
	itr := env.seedITR(t, 2025, xmlBytes)
	ded := env.seedDeduction(t, 2025, domain.DeductionPension)

	evil := filepath.Join(t.TempDir(), "not-in-datadir.pdf")
	if err := os.WriteFile(evil, []byte("leaked"), 0640); err != nil {
		t.Fatalf("creating outside-datadir file: %v", err)
	}

	doc := &domain.TaxDeductionDocument{
		TaxDeductionID: ded.ID,
		Filename:       "x.pdf",
		ContentType:    "application/pdf",
		StoragePath:    evil,
		Size:           6,
		CreatedAt:      time.Now(),
	}
	if err := env.deductDoc.Create(context.Background(), doc); err != nil {
		t.Fatalf("seeding evil doc: %v", err)
	}

	data, _, err := env.bundleSvc.GenerateBundle(context.Background(), itr.ID)
	if err == nil {
		t.Fatalf("expected GenerateBundle to fail for out-of-dataDir path, got success; archive keys: %v", keys(readZip(t, data)))
	}
	// Even on the failure path, no bytes of the evil file should leak.
	if data != nil && bytes.Contains(data, []byte("leaked")) {
		t.Errorf("bundle leaked file outside dataDir in error-path bytes")
	}
}

// keys is a small helper to produce sorted-ish key slices for error messages.
func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
