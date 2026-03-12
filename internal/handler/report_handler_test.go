package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestReportHandler_Revenue_DefaultYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	reportRepo := repository.NewReportRepository(db)
	reportSvc := service.NewReportService(reportRepo)
	h := NewReportHandler(reportSvc)

	r := chi.NewRouter()
	r.Get("/revenue", h.Revenue)

	req := httptest.NewRequest(http.MethodGet, "/revenue", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp revenueReportResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	currentYear := time.Now().Year()
	if resp.Year != currentYear {
		t.Errorf("Year = %d, want %d", resp.Year, currentYear)
	}
	if resp.Monthly == nil {
		t.Error("Monthly should not be nil")
	}
	if resp.Quarterly == nil {
		t.Error("Quarterly should not be nil")
	}
}

func TestReportHandler_Revenue_WithYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	reportRepo := repository.NewReportRepository(db)
	reportSvc := service.NewReportService(reportRepo)
	h := NewReportHandler(reportSvc)

	// Seed data for 2025.
	contact := testutil.SeedContact(t, db, nil)
	// Manually insert an invoice with delivery_date in 2025 via direct SQL.
	_, err := db.Exec(`
		INSERT INTO invoices (
			invoice_number, type, status, issue_date, due_date, delivery_date,
			customer_id, currency_code, exchange_rate, payment_method,
			subtotal_amount, vat_amount, total_amount, paid_amount,
			created_at, updated_at
		) VALUES (
			'FV2025-001', 'regular', 'sent', '2025-03-15', '2025-03-29', '2025-03-15',
			?, 'CZK', 100, 'bank_transfer',
			100000, 21000, 121000, 0,
			'2025-03-15T10:00:00Z', '2025-03-15T10:00:00Z'
		)`, contact.ID)
	if err != nil {
		t.Fatalf("seeding 2025 invoice: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/revenue", h.Revenue)

	req := httptest.NewRequest(http.MethodGet, "/revenue?year=2025", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp revenueReportResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Year != 2025 {
		t.Errorf("Year = %d, want 2025", resp.Year)
	}
	if resp.Total == 0 {
		t.Error("Total should be non-zero for year 2025")
	}
}

func TestReportHandler_Revenue_InvalidYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	reportRepo := repository.NewReportRepository(db)
	reportSvc := service.NewReportService(reportRepo)
	h := NewReportHandler(reportSvc)

	r := chi.NewRouter()
	r.Get("/revenue", h.Revenue)

	req := httptest.NewRequest(http.MethodGet, "/revenue?year=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestReportHandler_Revenue_OutOfRangeYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	reportRepo := repository.NewReportRepository(db)
	reportSvc := service.NewReportService(reportRepo)
	h := NewReportHandler(reportSvc)

	r := chi.NewRouter()
	r.Get("/revenue", h.Revenue)

	req := httptest.NewRequest(http.MethodGet, "/revenue?year=1999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestReportHandler_Expenses_DefaultYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	reportRepo := repository.NewReportRepository(db)
	reportSvc := service.NewReportService(reportRepo)
	h := NewReportHandler(reportSvc)

	// Seed an expense in the current year.
	testutil.SeedExpense(t, db, &domain.Expense{
		Description:  "Software license",
		Amount:       domain.NewAmount(200, 0),
		IssueDate:    time.Now(),
		CurrencyCode: domain.CurrencyCZK,
		Category:     "software",
	})

	r := chi.NewRouter()
	r.Get("/expenses", h.Expenses)

	req := httptest.NewRequest(http.MethodGet, "/expenses", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp expenseReportResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	currentYear := time.Now().Year()
	if resp.Year != currentYear {
		t.Errorf("Year = %d, want %d", resp.Year, currentYear)
	}
	if resp.Monthly == nil {
		t.Error("Monthly should not be nil")
	}
	if resp.Quarterly == nil {
		t.Error("Quarterly should not be nil")
	}
	if resp.Categories == nil {
		t.Error("Categories should not be nil")
	}
}

func TestReportHandler_TopCustomers(t *testing.T) {
	db := testutil.NewTestDB(t)
	reportRepo := repository.NewReportRepository(db)
	reportSvc := service.NewReportService(reportRepo)
	h := NewReportHandler(reportSvc)

	// Seed contacts and invoices for current year.
	contact1 := testutil.SeedContact(t, db, &domain.Contact{Name: "Alpha Corp"})
	contact2 := testutil.SeedContact(t, db, &domain.Contact{Name: "Beta Inc"})

	items := []domain.InvoiceItem{
		{
			Description:    "Service A",
			Quantity:       100,
			Unit:           "hod",
			UnitPrice:      domain.NewAmount(500, 0),
			VATRatePercent: 21,
		},
	}
	testutil.SeedInvoice(t, db, contact1.ID, items)
	testutil.SeedInvoice(t, db, contact1.ID, items)
	testutil.SeedInvoice(t, db, contact2.ID, items)

	r := chi.NewRouter()
	r.Get("/top-customers", h.TopCustomers)

	req := httptest.NewRequest(http.MethodGet, "/top-customers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []topCustomerResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if len(resp) < 2 {
		t.Fatalf("expected at least 2 customers, got %d", len(resp))
	}

	// First customer should be contact1 (2 invoices).
	if resp[0].CustomerID != contact1.ID {
		t.Errorf("top customer ID = %d, want %d", resp[0].CustomerID, contact1.ID)
	}
	if resp[0].InvoiceCount != 2 {
		t.Errorf("top customer invoice count = %d, want 2", resp[0].InvoiceCount)
	}
	if resp[0].CustomerName == "" {
		t.Error("top customer name should not be empty")
	}
}

func TestReportHandler_ProfitLoss(t *testing.T) {
	db := testutil.NewTestDB(t)
	reportRepo := repository.NewReportRepository(db)
	reportSvc := service.NewReportService(reportRepo)
	h := NewReportHandler(reportSvc)

	// Seed data for current year.
	contact := testutil.SeedContact(t, db, nil)
	items := []domain.InvoiceItem{
		{
			Description:    "Dev work",
			Quantity:       100,
			Unit:           "hod",
			UnitPrice:      domain.NewAmount(800, 0),
			VATRatePercent: 21,
		},
	}
	testutil.SeedInvoice(t, db, contact.ID, items)
	testutil.SeedExpense(t, db, &domain.Expense{
		Description:  "Hosting",
		Amount:       domain.NewAmount(300, 0),
		IssueDate:    time.Now(),
		CurrencyCode: domain.CurrencyCZK,
		Category:     "hosting",
	})

	r := chi.NewRouter()
	r.Get("/profit-loss", h.ProfitLoss)

	req := httptest.NewRequest(http.MethodGet, "/profit-loss", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp profitLossResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	currentYear := time.Now().Year()
	if resp.Year != currentYear {
		t.Errorf("Year = %d, want %d", resp.Year, currentYear)
	}
	if resp.MonthlyRevenue == nil {
		t.Error("MonthlyRevenue should not be nil")
	}
	if resp.MonthlyExpenses == nil {
		t.Error("MonthlyExpenses should not be nil")
	}
}
