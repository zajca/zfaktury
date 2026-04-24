// Package service contains business logic orchestration for ZFaktury.
package service

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
)

// maxBundleSize is a safety cap on the cumulative size of files added to a tax
// return bundle ZIP. Prevents unbounded memory use when assembling the archive
// in memory. 100 MB comfortably fits any plausible set of proof documents.
const maxBundleSize int64 = 100 << 20

// deductionCategoryLabels maps domain deduction category constants to the
// human-readable, ASCII-safe labels used as filename prefixes inside the ZIP.
var deductionCategoryLabels = map[string]string{
	domain.DeductionMortgage:      "uroky-hypoteka",
	domain.DeductionLifeInsurance: "zivotni-pojisteni",
	domain.DeductionPension:       "penzijni-sporeni",
	domain.DeductionDonation:      "dar",
	domain.DeductionUnionDues:     "odborove",
}

// IncomeTaxBundleService assembles a ZIP archive containing the DPFO XML and
// all mandatory proof documents (prilohy) for an income tax return.
//
// The investment document service is optional: if nil, investment statements
// are simply omitted from the archive. Only tax deduction documents are
// considered mandatory attachments to DPFO per §15.
type IncomeTaxBundleService struct {
	itrRepo          repository.IncomeTaxReturnRepo
	deductionRepo    repository.TaxDeductionRepo
	deductionDocRepo repository.TaxDeductionDocumentRepo
	itrSvc           *IncomeTaxReturnService
	deductionDocSvc  *TaxDeductionDocumentService
	investmentDocSvc *InvestmentDocumentService // nullable
	audit            *AuditService
	dataDir          string
}

// NewIncomeTaxBundleService constructs the bundle service. `investmentDocSvc`
// may be nil if investment integration is unavailable or disabled.
func NewIncomeTaxBundleService(
	itrRepo repository.IncomeTaxReturnRepo,
	deductionRepo repository.TaxDeductionRepo,
	deductionDocRepo repository.TaxDeductionDocumentRepo,
	itrSvc *IncomeTaxReturnService,
	deductionDocSvc *TaxDeductionDocumentService,
	investmentDocSvc *InvestmentDocumentService,
	audit *AuditService,
	dataDir string,
) *IncomeTaxBundleService {
	return &IncomeTaxBundleService{
		itrRepo:          itrRepo,
		deductionRepo:    deductionRepo,
		deductionDocRepo: deductionDocRepo,
		itrSvc:           itrSvc,
		deductionDocSvc:  deductionDocSvc,
		investmentDocSvc: investmentDocSvc,
		audit:            audit,
		dataDir:          dataDir,
	}
}

// GenerateBundle produces a ZIP archive for the income tax return with the
// given ID. The archive contains:
//
//	priznani-{year}.xml
//	prilohy/odpocty/{category}-{deductionID}-{filename}   (per proof document)
//	prilohy/investice/{year}-{platform}-{docID}-{filename} (optional)
//	CHYBEJICI-SOUBORY.txt                                  (only if any files
//	                                                        referenced in the DB
//	                                                        were missing on disk)
//
// Returns the bytes of the ZIP and a suggested filename like "priznani-2025.zip".
func (s *IncomeTaxBundleService) GenerateBundle(ctx context.Context, incomeTaxReturnID int64) ([]byte, string, error) {
	if incomeTaxReturnID == 0 {
		return nil, "", fmt.Errorf("income_tax_return ID is required: %w", domain.ErrInvalidInput)
	}

	itr, err := s.itrRepo.GetByID(ctx, incomeTaxReturnID)
	if err != nil {
		return nil, "", fmt.Errorf("fetching income_tax_return for bundle: %w", err)
	}

	// Ensure XML is generated before packaging. If the stored XMLData is empty
	// we regenerate it so the caller always gets a complete archive.
	if len(itr.XMLData) == 0 {
		if s.itrSvc == nil {
			return nil, "", fmt.Errorf("income tax return XML not generated and no generator available: %w", domain.ErrInvalidInput)
		}
		updated, genErr := s.itrSvc.GenerateXML(ctx, incomeTaxReturnID)
		if genErr != nil {
			return nil, "", fmt.Errorf("generating income_tax_return XML for bundle: %w", genErr)
		}
		itr = updated
	}

	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)

	var totalSize int64
	var missing []string

	// 1. Write the DPFO XML at the archive root.
	xmlName := fmt.Sprintf("priznani-%d.xml", itr.Year)
	if err := writeZipFile(zw, xmlName, itr.XMLData); err != nil {
		return nil, "", fmt.Errorf("writing XML to bundle: %w", err)
	}
	totalSize += int64(len(itr.XMLData))

	// 2. Add deduction proof documents under prilohy/odpocty/.
	deductions, err := s.deductionRepo.ListByYear(ctx, itr.Year)
	if err != nil {
		return nil, "", fmt.Errorf("listing deductions for bundle: %w", err)
	}

	// Stable ordering keeps archives reproducible for the same input.
	sort.Slice(deductions, func(i, j int) bool {
		if deductions[i].Category != deductions[j].Category {
			return deductions[i].Category < deductions[j].Category
		}
		return deductions[i].ID < deductions[j].ID
	})

	for i := range deductions {
		ded := &deductions[i]
		docs, err := s.deductionDocRepo.ListByDeductionID(ctx, ded.ID)
		if err != nil {
			return nil, "", fmt.Errorf("listing documents for deduction %d: %w", ded.ID, err)
		}
		label := deductionCategoryLabels[ded.Category]
		if label == "" {
			label = sanitizeBundleSegment(ded.Category)
		}
		for j := range docs {
			doc := &docs[j]
			entryName, size, err := s.addDeductionDoc(ctx, zw, label, ded.ID, doc, totalSize)
			if err != nil {
				return nil, "", err
			}
			if entryName == "" {
				// File missing on disk; record and continue.
				missing = append(missing, fmt.Sprintf("odpocty/%s (id=%d, filename=%q)", label, doc.ID, doc.Filename))
				continue
			}
			totalSize += size
		}
	}

	// 3. Optionally add investment documents under prilohy/investice/.
	if s.investmentDocSvc != nil {
		invDocs, err := s.investmentDocSvc.ListByYear(ctx, itr.Year)
		if err != nil {
			return nil, "", fmt.Errorf("listing investment documents for bundle: %w", err)
		}
		for i := range invDocs {
			doc := &invDocs[i]
			entryName, size, err := s.addInvestmentDoc(ctx, zw, doc, totalSize)
			if err != nil {
				return nil, "", err
			}
			if entryName == "" {
				missing = append(missing, fmt.Sprintf("investice/%d-%s (id=%d, filename=%q)", doc.Year, doc.Platform, doc.ID, doc.Filename))
				continue
			}
			totalSize += size
		}
	}

	// 4. If any files were missing, include a manifest listing them so the
	//    user knows the archive is incomplete without silently dropping data.
	if len(missing) > 0 {
		note := "Nasledujici soubory byly v databazi evidovany, ale chybely na disku a nemohly byt pribaleny:\n\n" +
			strings.Join(missing, "\n") + "\n"
		if err := writeZipFile(zw, "CHYBEJICI-SOUBORY.txt", []byte(note)); err != nil {
			return nil, "", fmt.Errorf("writing missing-files note: %w", err)
		}
	}

	if err := zw.Close(); err != nil {
		return nil, "", fmt.Errorf("closing zip writer: %w", err)
	}

	if s.audit != nil {
		s.audit.Log(ctx, "income_tax_return", incomeTaxReturnID, "generate_bundle", nil, nil)
	}

	filename := fmt.Sprintf("priznani-%d.zip", itr.Year)
	return buf.Bytes(), filename, nil
}

// addDeductionDoc reads the on-disk file for a deduction document and adds it
// to the archive under prilohy/odpocty/. Returns the entry name and its size.
// If the file is missing, returns ("", 0, nil) so the caller can record it in
// the missing-files note instead of failing the whole bundle.
func (s *IncomeTaxBundleService) addDeductionDoc(ctx context.Context, zw *zip.Writer, label string, deductionID int64, doc *domain.TaxDeductionDocument, totalSize int64) (string, int64, error) {
	if s.deductionDocSvc == nil {
		return "", 0, fmt.Errorf("deduction document service is not configured: %w", domain.ErrInvalidInput)
	}
	absPath, _, err := s.deductionDocSvc.GetFilePath(ctx, doc.ID)
	if err != nil {
		// Missing file or missing DB record = graceful: skip + note in manifest.
		// Any other error (bad config, path-traversal rejection) must bubble so
		// an operator sees it instead of silently dropping data.
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, domain.ErrNotFound) {
			return "", 0, nil
		}
		return "", 0, fmt.Errorf("resolving deduction document path %d: %w", doc.ID, err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", 0, nil
		}
		return "", 0, fmt.Errorf("stat deduction document %d: %w", doc.ID, err)
	}
	if totalSize+info.Size() > maxBundleSize {
		return "", 0, fmt.Errorf("bundle size would exceed limit of %d MB", maxBundleSize>>20)
	}

	safeName := sanitizeBundleSegment(doc.Filename)
	if safeName == "" {
		safeName = fmt.Sprintf("dokument-%d", doc.ID)
	}
	entryName := fmt.Sprintf("prilohy/odpocty/%s-%d-%s", label, deductionID, safeName)

	if err := copyFileToZip(zw, entryName, absPath, maxBundleSize-totalSize); err != nil {
		return "", 0, fmt.Errorf("copying deduction document %d into bundle: %w", doc.ID, err)
	}
	return entryName, info.Size(), nil
}

// addInvestmentDoc adds a broker statement to the archive under
// prilohy/investice/. Same missing-file semantics as addDeductionDoc.
func (s *IncomeTaxBundleService) addInvestmentDoc(ctx context.Context, zw *zip.Writer, doc *domain.InvestmentDocument, totalSize int64) (string, int64, error) {
	absPath, _, err := s.investmentDocSvc.GetFilePath(ctx, doc.ID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, domain.ErrNotFound) {
			return "", 0, nil
		}
		return "", 0, fmt.Errorf("resolving investment document path %d: %w", doc.ID, err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", 0, nil
		}
		return "", 0, fmt.Errorf("stat investment document %d: %w", doc.ID, err)
	}
	if totalSize+info.Size() > maxBundleSize {
		return "", 0, fmt.Errorf("bundle size would exceed limit of %d MB", maxBundleSize>>20)
	}

	safeName := sanitizeBundleSegment(doc.Filename)
	if safeName == "" {
		safeName = fmt.Sprintf("dokument-%d", doc.ID)
	}
	platform := sanitizeBundleSegment(doc.Platform)
	entryName := fmt.Sprintf("prilohy/investice/%d-%s-%d-%s", doc.Year, platform, doc.ID, safeName)

	if err := copyFileToZip(zw, entryName, absPath, maxBundleSize-totalSize); err != nil {
		return "", 0, fmt.Errorf("copying investment document %d into bundle: %w", doc.ID, err)
	}
	return entryName, info.Size(), nil
}

// writeZipFile writes a single in-memory byte slice as a ZIP entry.
func writeZipFile(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return fmt.Errorf("creating zip entry %q: %w", name, err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing zip entry %q: %w", name, err)
	}
	return nil
}

// copyFileToZip streams a file from disk into the ZIP, enforcing a byte limit
// at read time. The limit closes the TOCTOU window between os.Stat and the
// actual copy: if the source grew (or was swapped for something larger) between
// the pre-flight stat and now, we refuse the entry instead of blowing past the
// bundle cap.
func copyFileToZip(zw *zip.Writer, name string, sourcePath string, sizeLimit int64) error {
	if sizeLimit <= 0 {
		return fmt.Errorf("invalid size limit for zip entry %q", name)
	}
	f, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("opening %q: %w", sourcePath, err)
	}
	defer func() { _ = f.Close() }()

	w, err := zw.Create(name)
	if err != nil {
		return fmt.Errorf("creating zip entry %q: %w", name, err)
	}
	// Read one byte past the limit so we can detect overflow.
	n, err := io.Copy(w, io.LimitReader(f, sizeLimit+1))
	if err != nil {
		return fmt.Errorf("copying file into zip entry %q: %w", name, err)
	}
	if n > sizeLimit {
		return fmt.Errorf("zip entry %q exceeds size limit of %d bytes", name, sizeLimit)
	}
	return nil
}

// sanitizeBundleSegment strips path separators, null bytes, and leading dots so
// the result can be safely embedded in a ZIP entry name without producing any
// traversal-like paths. Returns an empty string if nothing usable remains.
func sanitizeBundleSegment(name string) string {
	name = strings.ReplaceAll(name, "\x00", "")
	name = filepath.Base(name)
	// filepath.Base returns "." or string(Separator) for some inputs we want to drop.
	if name == "." || name == string(filepath.Separator) {
		return ""
	}
	name = strings.ReplaceAll(name, "/", "")
	name = strings.ReplaceAll(name, "\\", "")
	name = strings.TrimSpace(name)
	name = strings.TrimLeft(name, ".")
	return name
}
