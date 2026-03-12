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

func TestDashboardHandler_GetDashboard_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	dashRepo := repository.NewDashboardRepository(db)
	dashSvc := service.NewDashboardService(dashRepo)
	h := NewDashboardHandler(dashSvc)

	r := chi.NewRouter()
	r.Get("/", h.GetDashboard)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp dashboardResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.RevenueCurrentMonth != 0 {
		t.Errorf("RevenueCurrentMonth = %d, want 0", resp.RevenueCurrentMonth)
	}
	if resp.ExpensesCurrentMonth != 0 {
		t.Errorf("ExpensesCurrentMonth = %d, want 0", resp.ExpensesCurrentMonth)
	}
	if resp.UnpaidCount != 0 {
		t.Errorf("UnpaidCount = %d, want 0", resp.UnpaidCount)
	}
	if resp.UnpaidTotal != 0 {
		t.Errorf("UnpaidTotal = %d, want 0", resp.UnpaidTotal)
	}
	if resp.OverdueCount != 0 {
		t.Errorf("OverdueCount = %d, want 0", resp.OverdueCount)
	}
	if resp.OverdueTotal != 0 {
		t.Errorf("OverdueTotal = %d, want 0", resp.OverdueTotal)
	}
	if resp.MonthlyRevenue == nil {
		t.Error("MonthlyRevenue should not be nil")
	}
	if resp.MonthlyExpenses == nil {
		t.Error("MonthlyExpenses should not be nil")
	}
	if resp.RecentInvoices == nil {
		t.Error("RecentInvoices should not be nil")
	}
	if resp.RecentExpenses == nil {
		t.Error("RecentExpenses should not be nil")
	}
}

func TestDashboardHandler_GetDashboard_WithData(t *testing.T) {
	db := testutil.NewTestDB(t)
	dashRepo := repository.NewDashboardRepository(db)
	dashSvc := service.NewDashboardService(dashRepo)
	h := NewDashboardHandler(dashSvc)

	// Seed a contact.
	contact := testutil.SeedContact(t, db, nil)

	// Seed an invoice with delivery_date in current month so it shows up in revenue.
	now := time.Now()
	items := []domain.InvoiceItem{
		{
			Description:    "Consulting",
			Quantity:       100,
			Unit:           "hod",
			UnitPrice:      domain.NewAmount(1000, 0),
			VATRatePercent: 21,
		},
	}
	inv := testutil.SeedInvoice(t, db, contact.ID, items)
	// The invoice is seeded with delivery_date = now, so it should appear in current month revenue.
	_ = inv

	// Seed an expense in the current month.
	testutil.SeedExpense(t, db, &domain.Expense{
		Description:  "Office rent",
		Amount:       domain.NewAmount(500, 0),
		IssueDate:    now,
		CurrencyCode: domain.CurrencyCZK,
		Category:     "rent",
	})

	r := chi.NewRouter()
	r.Get("/", h.GetDashboard)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp dashboardResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	// Invoice total should be non-zero (1000*100 quantity*unit_price with VAT).
	if resp.RevenueCurrentMonth == 0 {
		t.Error("RevenueCurrentMonth should be non-zero after seeding invoice")
	}

	// Expense should be non-zero.
	if resp.ExpensesCurrentMonth == 0 {
		t.Error("ExpensesCurrentMonth should be non-zero after seeding expense")
	}

	// Unpaid count should be at least 1 (draft invoice).
	if resp.UnpaidCount < 1 {
		t.Errorf("UnpaidCount = %d, want >= 1", resp.UnpaidCount)
	}

	// Recent invoices should contain our seeded invoice.
	if len(resp.RecentInvoices) < 1 {
		t.Error("expected at least 1 recent invoice")
	}

	// Recent expenses should contain our seeded expense.
	if len(resp.RecentExpenses) < 1 {
		t.Error("expected at least 1 recent expense")
	}
}
