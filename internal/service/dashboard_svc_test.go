package service

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestDashboardService_GetDashboard_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewDashboardRepository(db)
	svc := NewDashboardService(repo)
	ctx := context.Background()

	data, err := svc.GetDashboard(ctx)
	if err != nil {
		t.Fatalf("GetDashboard() error: %v", err)
	}

	if data.RevenueCurrentMonth != 0 {
		t.Errorf("RevenueCurrentMonth = %d, want 0", data.RevenueCurrentMonth)
	}
	if data.ExpensesCurrentMonth != 0 {
		t.Errorf("ExpensesCurrentMonth = %d, want 0", data.ExpensesCurrentMonth)
	}
	if data.UnpaidCount != 0 {
		t.Errorf("UnpaidCount = %d, want 0", data.UnpaidCount)
	}
	if data.UnpaidTotal != 0 {
		t.Errorf("UnpaidTotal = %d, want 0", data.UnpaidTotal)
	}
	if data.OverdueCount != 0 {
		t.Errorf("OverdueCount = %d, want 0", data.OverdueCount)
	}
	if data.OverdueTotal != 0 {
		t.Errorf("OverdueTotal = %d, want 0", data.OverdueTotal)
	}
	if len(data.MonthlyRevenue) != 0 {
		t.Errorf("MonthlyRevenue len = %d, want 0", len(data.MonthlyRevenue))
	}
	if len(data.MonthlyExpenses) != 0 {
		t.Errorf("MonthlyExpenses len = %d, want 0", len(data.MonthlyExpenses))
	}
	if len(data.RecentInvoices) != 0 {
		t.Errorf("RecentInvoices len = %d, want 0", len(data.RecentInvoices))
	}
	if len(data.RecentExpenses) != 0 {
		t.Errorf("RecentExpenses len = %d, want 0", len(data.RecentExpenses))
	}
}

func TestDashboardService_GetDashboard_WithData(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewDashboardRepository(db)
	svc := NewDashboardService(repo)
	ctx := context.Background()

	// Seed a contact for invoices.
	contact := testutil.SeedContact(t, db, nil)

	// Seed two invoices with known item amounts.
	// SeedInvoice uses time.Now() for dates, so they appear in the current month.
	items1 := []domain.InvoiceItem{
		{
			Description:    "Service A",
			Quantity:       100,
			Unit:           "ks",
			UnitPrice:      domain.NewAmount(1000, 0),
			VATRatePercent: 0,
		},
	}
	inv1 := testutil.SeedInvoice(t, db, contact.ID, items1)

	items2 := []domain.InvoiceItem{
		{
			Description:    "Service B",
			Quantity:       100,
			Unit:           "ks",
			UnitPrice:      domain.NewAmount(2000, 0),
			VATRatePercent: 0,
		},
	}
	inv2 := testutil.SeedInvoice(t, db, contact.ID, items2)

	// Seed an expense in the current month.
	testutil.SeedExpense(t, db, &domain.Expense{
		Description:  "Office supplies",
		Amount:       domain.NewAmount(500, 0),
		IssueDate:    time.Now(),
		CurrencyCode: domain.CurrencyCZK,
	})

	data, err := svc.GetDashboard(ctx)
	if err != nil {
		t.Fatalf("GetDashboard() error: %v", err)
	}

	// Revenue should be the sum of both invoices' total amounts.
	expectedRevenue := inv1.TotalAmount + inv2.TotalAmount
	if data.RevenueCurrentMonth != expectedRevenue {
		t.Errorf("RevenueCurrentMonth = %d, want %d", data.RevenueCurrentMonth, expectedRevenue)
	}

	// Expenses should include our seeded expense.
	expectedExpenses := domain.NewAmount(500, 0)
	if data.ExpensesCurrentMonth != expectedExpenses {
		t.Errorf("ExpensesCurrentMonth = %d, want %d", data.ExpensesCurrentMonth, expectedExpenses)
	}

	// Both invoices are draft status (unpaid, not credit notes).
	if data.UnpaidCount != 2 {
		t.Errorf("UnpaidCount = %d, want 2", data.UnpaidCount)
	}

	// Recent invoices should include both.
	if len(data.RecentInvoices) != 2 {
		t.Errorf("RecentInvoices len = %d, want 2", len(data.RecentInvoices))
	}

	// Recent expenses should include the one we seeded.
	if len(data.RecentExpenses) != 1 {
		t.Errorf("RecentExpenses len = %d, want 1", len(data.RecentExpenses))
	}

	// Monthly revenue should have an entry for current month.
	now := time.Now()
	currentMonth := int(now.Month())
	found := false
	for _, mr := range data.MonthlyRevenue {
		if mr.Month == currentMonth {
			found = true
			if mr.Amount != expectedRevenue {
				t.Errorf("MonthlyRevenue[%d] = %d, want %d", currentMonth, mr.Amount, expectedRevenue)
			}
		}
	}
	if !found {
		t.Errorf("MonthlyRevenue missing entry for current month %d", currentMonth)
	}
}
