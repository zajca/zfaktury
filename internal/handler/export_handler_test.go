package handler

import (
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/testutil"
)

func setupExportHandler(t *testing.T) (*ExportHandler, *chi.Mux) {
	t.Helper()
	db := testutil.NewTestDB(t)

	invoiceRepo := repository.NewInvoiceRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	contactRepo := repository.NewContactRepository(db)
	seqRepo := repository.NewSequenceRepository(db)

	contactSvc := service.NewContactService(contactRepo, nil, nil)
	seqSvc := service.NewSequenceService(seqRepo, nil)
	invoiceSvc := service.NewInvoiceService(invoiceRepo, contactSvc, seqSvc, nil)
	expenseSvc := service.NewExpenseService(expenseRepo, nil)

	h := NewExportHandler(invoiceSvc, expenseSvc)

	r := chi.NewRouter()
	r.Get("/invoices", h.ExportInvoices)
	r.Get("/expenses", h.ExportExpenses)

	return h, r
}

func TestExportHandler_ExportInvoices_Empty(t *testing.T) {
	_, r := setupExportHandler(t)

	year := time.Now().Year()
	req := httptest.NewRequest(http.MethodGet, "/invoices", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Check Content-Type.
	ct := w.Header().Get("Content-Type")
	if ct != "text/csv; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/csv; charset=utf-8")
	}

	// Check Content-Disposition.
	cd := w.Header().Get("Content-Disposition")
	if !strings.Contains(cd, ".csv") {
		t.Errorf("Content-Disposition = %q, expected CSV filename", cd)
	}

	body := w.Body.Bytes()

	// Check UTF-8 BOM.
	if len(body) < 3 || body[0] != 0xEF || body[1] != 0xBB || body[2] != 0xBF {
		t.Error("expected UTF-8 BOM at start of CSV")
	}

	// Parse CSV (skip BOM).
	reader := csv.NewReader(strings.NewReader(string(body[3:])))
	reader.Comma = ';'
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("parsing CSV: %v", err)
	}

	// Should have only the header row.
	if len(records) != 1 {
		t.Errorf("expected 1 row (header only), got %d", len(records))
	}

	// Verify header columns.
	expectedHeaders := []string{
		"Cislo", "Typ", "Stav", "Odberatel", "Datum vystaveni",
		"Datum splatnosti", "DUZP", "Castka bez DPH", "DPH", "Celkem", "Mena",
	}
	if len(records[0]) != len(expectedHeaders) {
		t.Fatalf("header column count = %d, want %d", len(records[0]), len(expectedHeaders))
	}
	for i, h := range expectedHeaders {
		if records[0][i] != h {
			t.Errorf("header[%d] = %q, want %q", i, records[0][i], h)
		}
	}
	_ = year
}

func TestExportHandler_ExportInvoices_WithData(t *testing.T) {
	db := testutil.NewTestDB(t)

	invoiceRepo := repository.NewInvoiceRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	contactRepo := repository.NewContactRepository(db)
	seqRepo := repository.NewSequenceRepository(db)

	contactSvc := service.NewContactService(contactRepo, nil, nil)
	seqSvc := service.NewSequenceService(seqRepo, nil)
	invoiceSvc := service.NewInvoiceService(invoiceRepo, contactSvc, seqSvc, nil)
	expenseSvc := service.NewExpenseService(expenseRepo, nil)

	h := NewExportHandler(invoiceSvc, expenseSvc)

	// Seed data.
	contact := testutil.SeedContact(t, db, &domain.Contact{Name: "Export Test Corp"})
	items := []domain.InvoiceItem{
		{
			Description:    "Development",
			Quantity:       100,
			Unit:           "hod",
			UnitPrice:      domain.NewAmount(1500, 0),
			VATRatePercent: 21,
		},
	}
	testutil.SeedInvoice(t, db, contact.ID, items)

	r := chi.NewRouter()
	r.Get("/invoices", h.ExportInvoices)

	req := httptest.NewRequest(http.MethodGet, "/invoices", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	body := w.Body.Bytes()

	// Skip BOM.
	reader := csv.NewReader(strings.NewReader(string(body[3:])))
	reader.Comma = ';'
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("parsing CSV: %v", err)
	}

	// Header + at least 1 data row.
	if len(records) < 2 {
		t.Fatalf("expected at least 2 rows (header + data), got %d", len(records))
	}

	// Check data row has non-empty invoice number.
	dataRow := records[1]
	if dataRow[0] == "" {
		t.Error("invoice number (column 0) should not be empty")
	}
	// Type should be "regular".
	if dataRow[1] != "regular" {
		t.Errorf("invoice type = %q, want %q", dataRow[1], "regular")
	}
	// Currency should be CZK.
	if dataRow[10] != "CZK" {
		t.Errorf("currency = %q, want %q", dataRow[10], "CZK")
	}
}

func TestExportHandler_ExportInvoices_InvalidYear(t *testing.T) {
	db := testutil.NewTestDB(t)

	invoiceRepo := repository.NewInvoiceRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	contactRepo := repository.NewContactRepository(db)
	seqRepo := repository.NewSequenceRepository(db)

	contactSvc := service.NewContactService(contactRepo, nil, nil)
	seqSvc := service.NewSequenceService(seqRepo, nil)
	invoiceSvc := service.NewInvoiceService(invoiceRepo, contactSvc, seqSvc, nil)
	expenseSvc := service.NewExpenseService(expenseRepo, nil)

	h := NewExportHandler(invoiceSvc, expenseSvc)

	r := chi.NewRouter()
	r.Get("/invoices", h.ExportInvoices)

	req := httptest.NewRequest(http.MethodGet, "/invoices?year=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestExportHandler_ExportExpenses_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)

	invoiceRepo := repository.NewInvoiceRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	contactRepo := repository.NewContactRepository(db)
	seqRepo := repository.NewSequenceRepository(db)

	contactSvc := service.NewContactService(contactRepo, nil, nil)
	seqSvc := service.NewSequenceService(seqRepo, nil)
	invoiceSvc := service.NewInvoiceService(invoiceRepo, contactSvc, seqSvc, nil)
	expenseSvc := service.NewExpenseService(expenseRepo, nil)

	h := NewExportHandler(invoiceSvc, expenseSvc)

	r := chi.NewRouter()
	r.Get("/expenses", h.ExportExpenses)

	req := httptest.NewRequest(http.MethodGet, "/expenses", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Check Content-Type.
	ct := w.Header().Get("Content-Type")
	if ct != "text/csv; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/csv; charset=utf-8")
	}

	body := w.Body.Bytes()

	// Check UTF-8 BOM.
	if len(body) < 3 || body[0] != 0xEF || body[1] != 0xBB || body[2] != 0xBF {
		t.Error("expected UTF-8 BOM at start of CSV")
	}

	// Parse CSV (skip BOM).
	reader := csv.NewReader(strings.NewReader(string(body[3:])))
	reader.Comma = ';'
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("parsing CSV: %v", err)
	}

	// Should have only the header row.
	if len(records) != 1 {
		t.Errorf("expected 1 row (header only), got %d", len(records))
	}

	// Verify expense header columns.
	expectedHeaders := []string{
		"Cislo", "Popis", "Kategorie", "Dodavatel", "Datum", "Castka", "DPH", "Mena",
	}
	if len(records[0]) != len(expectedHeaders) {
		t.Fatalf("header column count = %d, want %d", len(records[0]), len(expectedHeaders))
	}
	for i, h := range expectedHeaders {
		if records[0][i] != h {
			t.Errorf("header[%d] = %q, want %q", i, records[0][i], h)
		}
	}
}

func TestExportHandler_ExportExpenses_WithData(t *testing.T) {
	db := testutil.NewTestDB(t)

	invoiceRepo := repository.NewInvoiceRepository(db)
	expenseRepo := repository.NewExpenseRepository(db)
	contactRepo := repository.NewContactRepository(db)
	seqRepo := repository.NewSequenceRepository(db)

	contactSvc := service.NewContactService(contactRepo, nil, nil)
	seqSvc := service.NewSequenceService(seqRepo, nil)
	invoiceSvc := service.NewInvoiceService(invoiceRepo, contactSvc, seqSvc, nil)
	expenseSvc := service.NewExpenseService(expenseRepo, nil)

	h := NewExportHandler(invoiceSvc, expenseSvc)

	// Seed an expense in the current year.
	testutil.SeedExpense(t, db, &domain.Expense{
		ExpenseNumber: "VY-2026-001",
		Description:   "Office supplies",
		Amount:        domain.NewAmount(250, 50),
		IssueDate:     time.Now(),
		CurrencyCode:  domain.CurrencyCZK,
		Category:      "supplies",
	})

	r := chi.NewRouter()
	r.Get("/expenses", h.ExportExpenses)

	req := httptest.NewRequest(http.MethodGet, "/expenses", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	body := w.Body.Bytes()

	// Skip BOM.
	reader := csv.NewReader(strings.NewReader(string(body[3:])))
	reader.Comma = ';'
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("parsing CSV: %v", err)
	}

	// Header + at least 1 data row.
	if len(records) < 2 {
		t.Fatalf("expected at least 2 rows (header + data), got %d", len(records))
	}

	dataRow := records[1]
	// Description should match.
	if dataRow[1] != "Office supplies" {
		t.Errorf("description = %q, want %q", dataRow[1], "Office supplies")
	}
	// Category should match.
	if dataRow[2] != "supplies" {
		t.Errorf("category = %q, want %q", dataRow[2], "supplies")
	}
	// Currency should be CZK.
	if dataRow[7] != "CZK" {
		t.Errorf("currency = %q, want %q", dataRow[7], "CZK")
	}
}
