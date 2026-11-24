package handler

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// downloadHeader performs a GET and returns the Content-Disposition header of a 200 response.
func downloadHeader(t *testing.T, r *chi.Mux, path string) string {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s: status = %d, body = %s", path, w.Code, w.Body.String())
	}
	return w.Header().Get("Content-Disposition")
}

func execSQL(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func TestDownloadFilename_VATReturn(t *testing.T) {
	r, db := setupVATReturnRouter(t)
	id := createVATReturn(t, r)
	execSQL(t, db, `UPDATE vat_returns SET xml_data = '<xml/>' WHERE id = ?`, id)

	if got, want := downloadHeader(t, r, fmt.Sprintf("/api/v1/vat-returns/%d/xml", id)), "attachment; filename=dph-2025-01.xml"; got != want {
		t.Errorf("Content-Disposition = %q, want %q", got, want)
	}

	execSQL(t, db, `UPDATE vat_returns SET filing_type = 'corrective' WHERE id = ?`, id)
	if got, want := downloadHeader(t, r, fmt.Sprintf("/api/v1/vat-returns/%d/xml", id)), "attachment; filename=dph-2025-01-opravne.xml"; got != want {
		t.Errorf("corrective Content-Disposition = %q, want %q", got, want)
	}
}

func TestDownloadFilename_ControlStatement(t *testing.T) {
	r, db := setupVATControlRouter(t)
	id := createVATControlStatement(t, r, 2025, 3)
	execSQL(t, db, `UPDATE vat_control_statements SET xml_data = '<xml/>' WHERE id = ?`, id)

	if got, want := downloadHeader(t, r, fmt.Sprintf("/api/v1/vat-control-statements/%d/xml", id)), "attachment; filename=kh-2025-03.xml"; got != want {
		t.Errorf("Content-Disposition = %q, want %q", got, want)
	}

	execSQL(t, db, `UPDATE vat_control_statements SET filing_type = 'supplementary' WHERE id = ?`, id)
	if got, want := downloadHeader(t, r, fmt.Sprintf("/api/v1/vat-control-statements/%d/xml", id)), "attachment; filename=kh-2025-03-nasledne.xml"; got != want {
		t.Errorf("supplementary Content-Disposition = %q, want %q", got, want)
	}
}

func TestDownloadFilename_VIES(t *testing.T) {
	r, db := setupVIESRouter(t)
	vs := createTestVIESSummary(t, r, 2025, 1)
	execSQL(t, db, `UPDATE vies_summaries SET xml_data = '<xml/>' WHERE id = ?`, vs.ID)

	if got, want := downloadHeader(t, r, fmt.Sprintf("/api/v1/vies-summaries/%d/xml", vs.ID)), "attachment; filename=sh-2025-q1.xml"; got != want {
		t.Errorf("Content-Disposition = %q, want %q", got, want)
	}

	execSQL(t, db, `UPDATE vies_summaries SET filing_type = 'supplementary' WHERE id = ?`, vs.ID)
	if got, want := downloadHeader(t, r, fmt.Sprintf("/api/v1/vies-summaries/%d/xml", vs.ID)), "attachment; filename=sh-2025-q1-nasledne.xml"; got != want {
		t.Errorf("supplementary Content-Disposition = %q, want %q", got, want)
	}
}

func TestDownloadFilename_SocialInsurance(t *testing.T) {
	r, db := setupSocialInsuranceRouter(t)
	id := createSocialInsurance(t, r, 2025)
	execSQL(t, db, `UPDATE social_insurance_overviews SET xml_data = '<xml/>' WHERE id = ?`, id)

	if got, want := downloadHeader(t, r, fmt.Sprintf("/api/v1/social-insurance/%d/xml", id)), "attachment; filename=cssz-2025.xml"; got != want {
		t.Errorf("Content-Disposition = %q, want %q", got, want)
	}

	execSQL(t, db, `UPDATE social_insurance_overviews SET filing_type = 'supplementary' WHERE id = ?`, id)
	if got, want := downloadHeader(t, r, fmt.Sprintf("/api/v1/social-insurance/%d/xml", id)), "attachment; filename=cssz-2025-zmenovy.xml"; got != want {
		t.Errorf("supplementary Content-Disposition = %q, want %q", got, want)
	}
}

func TestDownloadFilename_HealthInsurance(t *testing.T) {
	r, db := setupHealthInsuranceRouter(t)
	id := createHealthInsurance(t, r, 2025)
	execSQL(t, db, `UPDATE health_insurance_overviews SET xml_data = '<xml/>' WHERE id = ?`, id)

	if got, want := downloadHeader(t, r, fmt.Sprintf("/api/v1/health-insurance/%d/xml", id)), "attachment; filename=zp-2025.xml"; got != want {
		t.Errorf("Content-Disposition = %q, want %q", got, want)
	}

	execSQL(t, db, `UPDATE health_insurance_overviews SET filing_type = 'supplementary' WHERE id = ?`, id)
	if got, want := downloadHeader(t, r, fmt.Sprintf("/api/v1/health-insurance/%d/xml", id)), "attachment; filename=zp-2025-zmenovy.xml"; got != want {
		t.Errorf("supplementary Content-Disposition = %q, want %q", got, want)
	}
}

func TestDownloadFilename_InvoicePDFAndISDOC(t *testing.T) {
	r, db, seqID := setupInvoiceRouterWithSettings(t)
	seedSettings(t, db)
	customerID := createInvoiceCustomer(t, r)
	created := createTestInvoice(t, r, customerID, seqID)
	execSQL(t, db, `UPDATE invoices SET invoice_number = 'FV/2026/001' WHERE id = ?`, created.ID)

	if got, want := downloadHeader(t, r, fmt.Sprintf("/api/v1/invoices/%d/pdf", created.ID)), "attachment; filename=faktura-fv-2026-001.pdf"; got != want {
		t.Errorf("PDF Content-Disposition = %q, want %q", got, want)
	}
	if got, want := downloadHeader(t, r, fmt.Sprintf("/api/v1/invoices/%d/isdoc", created.ID)), "attachment; filename=faktura-fv-2026-001.isdoc"; got != want {
		t.Errorf("ISDOC Content-Disposition = %q, want %q", got, want)
	}
}

func TestDownloadFilename_ISDOCBatchDuplicateEntries(t *testing.T) {
	r, db, seqID := setupInvoiceRouterWithSettings(t)
	seedSettings(t, db)
	customerID := createInvoiceCustomer(t, r)

	var ids []int64
	for _, number := range []string{"FV/2026/009", "FV-2026-009", "FV 2026 009"} {
		inv := createTestInvoice(t, r, customerID, seqID)
		execSQL(t, db, `UPDATE invoices SET invoice_number = ? WHERE id = ?`, number, inv.ID)
		ids = append(ids, inv.ID)
	}

	body := fmt.Sprintf(`{"invoice_ids":[%d,%d,%d]}`, ids[0], ids[1], ids[2])
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invoices/export/isdoc", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if got, want := w.Header().Get("Content-Disposition"), "attachment; filename=faktury-isdoc.zip"; got != want {
		t.Errorf("Content-Disposition = %q, want %q", got, want)
	}

	zr, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
	if err != nil {
		t.Fatalf("reading zip: %v", err)
	}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	want := []string{"faktura-fv-2026-009-2.isdoc", "faktura-fv-2026-009-3.isdoc", "faktura-fv-2026-009.isdoc"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("zip entries = %v, want %v", names, want)
	}
}
