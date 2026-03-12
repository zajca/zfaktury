package repository

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestDashboardRepo_RevenueCurrentMonth_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDashboardRepository(db)
	ctx := context.Background()

	now := time.Now()
	rev, err := repo.RevenueCurrentMonth(ctx, now.Year(), int(now.Month()))
	if err != nil {
		t.Fatalf("RevenueCurrentMonth() error: %v", err)
	}
	if rev != 0 {
		t.Errorf("expected 0 revenue, got %d", rev)
	}
}

func TestDashboardRepo_RevenueCurrentMonth_WithData(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDashboardRepository(db)
	ctx := context.Background()

	now := time.Now()
	contact := testutil.SeedContact(t, db, nil)

	// Seed two invoices with current month delivery date (SeedInvoice uses time.Now()).
	inv1 := testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Item A", Quantity: 100, UnitPrice: 10000, VATRatePercent: 21},
	})
	inv2 := testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Item B", Quantity: 200, UnitPrice: 5000, VATRatePercent: 21},
	})

	expectedTotal := inv1.TotalAmount + inv2.TotalAmount

	rev, err := repo.RevenueCurrentMonth(ctx, now.Year(), int(now.Month()))
	if err != nil {
		t.Fatalf("RevenueCurrentMonth() error: %v", err)
	}
	if rev != expectedTotal {
		t.Errorf("revenue = %d, want %d", rev, expectedTotal)
	}
}

func TestDashboardRepo_ExpensesCurrentMonth_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDashboardRepository(db)
	ctx := context.Background()

	now := time.Now()
	total, err := repo.ExpensesCurrentMonth(ctx, now.Year(), int(now.Month()))
	if err != nil {
		t.Fatalf("ExpensesCurrentMonth() error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 expenses, got %d", total)
	}
}

func TestDashboardRepo_UnpaidInvoices_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDashboardRepository(db)
	ctx := context.Background()

	count, total, err := repo.UnpaidInvoices(ctx)
	if err != nil {
		t.Fatalf("UnpaidInvoices() error: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
}

func TestDashboardRepo_UnpaidInvoices_WithData(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDashboardRepository(db)
	ctx := context.Background()

	contact := testutil.SeedContact(t, db, nil)

	// SeedInvoice creates draft invoices by default (unpaid).
	inv1 := testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Item", Quantity: 100, UnitPrice: 10000, VATRatePercent: 21},
	})
	inv2 := testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Item", Quantity: 100, UnitPrice: 20000, VATRatePercent: 21},
	})

	// Mark one as "sent" (still unpaid).
	_, err := db.ExecContext(ctx, "UPDATE invoices SET status = 'sent' WHERE id = ?", inv2.ID)
	if err != nil {
		t.Fatalf("updating status: %v", err)
	}

	// Create a paid invoice that should NOT be counted.
	inv3 := testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Item", Quantity: 100, UnitPrice: 50000, VATRatePercent: 21},
	})
	_, err = db.ExecContext(ctx, "UPDATE invoices SET status = 'paid' WHERE id = ?", inv3.ID)
	if err != nil {
		t.Fatalf("updating status: %v", err)
	}

	count, total, err := repo.UnpaidInvoices(ctx)
	if err != nil {
		t.Fatalf("UnpaidInvoices() error: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	expectedTotal := inv1.TotalAmount + inv2.TotalAmount
	if total != expectedTotal {
		t.Errorf("total = %d, want %d", total, expectedTotal)
	}
}

func TestDashboardRepo_OverdueInvoices_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDashboardRepository(db)
	ctx := context.Background()

	count, total, err := repo.OverdueInvoices(ctx)
	if err != nil {
		t.Fatalf("OverdueInvoices() error: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
}

func TestDashboardRepo_MonthlyRevenue_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDashboardRepository(db)
	ctx := context.Background()

	result, err := repo.MonthlyRevenue(ctx, time.Now().Year())
	if err != nil {
		t.Fatalf("MonthlyRevenue() error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestDashboardRepo_MonthlyRevenue_WithData(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDashboardRepository(db)
	ctx := context.Background()

	now := time.Now()
	contact := testutil.SeedContact(t, db, nil)

	// Seed invoice in current month.
	inv1 := testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Item", Quantity: 100, UnitPrice: 10000, VATRatePercent: 21},
	})

	// Seed invoice in a different month (2 months ago).
	inv2 := testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Item", Quantity: 100, UnitPrice: 20000, VATRatePercent: 21},
	})
	twoMonthsAgo := now.AddDate(0, -2, 0)
	_, err := db.ExecContext(ctx, "UPDATE invoices SET delivery_date = ? WHERE id = ?",
		twoMonthsAgo.Format("2006-01-02"), inv2.ID)
	if err != nil {
		t.Fatalf("updating delivery_date: %v", err)
	}

	result, err := repo.MonthlyRevenue(ctx, now.Year())
	if err != nil {
		t.Fatalf("MonthlyRevenue() error: %v", err)
	}

	// Depending on whether twoMonthsAgo is in the same year, we might get 1 or 2 entries.
	if twoMonthsAgo.Year() == now.Year() {
		if len(result) != 2 {
			t.Errorf("expected 2 monthly entries, got %d", len(result))
		}
		// Find current month entry and verify amount.
		found := false
		for _, ma := range result {
			if ma.Month == int(now.Month()) {
				if ma.Amount != inv1.TotalAmount {
					t.Errorf("current month amount = %d, want %d", ma.Amount, inv1.TotalAmount)
				}
				found = true
			}
		}
		if !found {
			t.Error("current month not found in results")
		}
	} else {
		// twoMonthsAgo is in previous year, only current month should appear.
		if len(result) != 1 {
			t.Errorf("expected 1 monthly entry, got %d", len(result))
		}
		if result[0].Amount != inv1.TotalAmount {
			t.Errorf("amount = %d, want %d", result[0].Amount, inv1.TotalAmount)
		}
	}
}

func TestDashboardRepo_RecentInvoices_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDashboardRepository(db)
	ctx := context.Background()

	result, err := repo.RecentInvoices(ctx, 5)
	if err != nil {
		t.Fatalf("RecentInvoices() error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestDashboardRepo_RecentInvoices_Limit(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDashboardRepository(db)
	ctx := context.Background()

	contact := testutil.SeedContact(t, db, nil)

	// Seed 5 invoices.
	for i := 0; i < 5; i++ {
		testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
			{Description: "Item", Quantity: 100, UnitPrice: 10000, VATRatePercent: 21},
		})
	}

	// Request only 3.
	result, err := repo.RecentInvoices(ctx, 3)
	if err != nil {
		t.Fatalf("RecentInvoices() error: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}
}

func TestDashboardRepo_MonthlyExpenses_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDashboardRepository(db)
	ctx := context.Background()

	result, err := repo.MonthlyExpenses(ctx, time.Now().Year())
	if err != nil {
		t.Fatalf("MonthlyExpenses() error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestDashboardRepo_MonthlyExpenses_WithData(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDashboardRepository(db)
	ctx := context.Background()

	now := time.Now()
	testutil.SeedExpense(t, db, &domain.Expense{
		Description: "Expense A",
		Amount:      domain.NewAmount(500, 0),
		IssueDate:   now,
		Category:    "software",
	})
	testutil.SeedExpense(t, db, &domain.Expense{
		Description: "Expense B",
		Amount:      domain.NewAmount(300, 0),
		IssueDate:   now,
		Category:    "hosting",
	})

	result, err := repo.MonthlyExpenses(ctx, now.Year())
	if err != nil {
		t.Fatalf("MonthlyExpenses() error: %v", err)
	}

	// Should have at least one entry for current month.
	found := false
	for _, ma := range result {
		if ma.Month == int(now.Month()) {
			expected := domain.NewAmount(800, 0)
			if ma.Amount != expected {
				t.Errorf("current month expense = %d, want %d", ma.Amount, expected)
			}
			found = true
		}
	}
	if !found {
		t.Errorf("current month not found in results")
	}
}

func TestDashboardRepo_RecentExpenses_WithData(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDashboardRepository(db)
	ctx := context.Background()

	// Seed 3 expenses.
	for i := 0; i < 3; i++ {
		testutil.SeedExpense(t, db, &domain.Expense{
			Description: "Expense",
			Amount:      domain.NewAmount(100, 0),
			IssueDate:   time.Now(),
			Category:    "software",
		})
	}

	result, err := repo.RecentExpenses(ctx, 2)
	if err != nil {
		t.Fatalf("RecentExpenses() error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
}

func TestDashboardRepo_RecentExpenses_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewDashboardRepository(db)
	ctx := context.Background()

	result, err := repo.RecentExpenses(ctx, 5)
	if err != nil {
		t.Fatalf("RecentExpenses() error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}
