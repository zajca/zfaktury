package repository

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestReportRepo_MonthlyRevenue_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	result, err := repo.MonthlyRevenue(ctx, time.Now().Year())
	if err != nil {
		t.Fatalf("MonthlyRevenue() error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestReportRepo_MonthlyRevenue_WithData(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	now := time.Now()
	contact := testutil.SeedContact(t, db, nil)

	// Seed invoice in current month.
	inv1 := testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Item A", Quantity: 100, UnitPrice: 10000, VATRatePercent: 21},
	})

	// Seed another invoice in a different month (3 months ago).
	inv2 := testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Item B", Quantity: 200, UnitPrice: 5000, VATRatePercent: 21},
	})
	threeMonthsAgo := now.AddDate(0, -3, 0)
	_, err := db.ExecContext(ctx, "UPDATE invoices SET delivery_date = ? WHERE id = ?",
		threeMonthsAgo.Format("2006-01-02"), inv2.ID)
	if err != nil {
		t.Fatalf("updating delivery_date: %v", err)
	}

	result, err := repo.MonthlyRevenue(ctx, now.Year())
	if err != nil {
		t.Fatalf("MonthlyRevenue() error: %v", err)
	}

	// Verify current month entry exists.
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

	// If threeMonthsAgo is in the same year, verify that entry too.
	if threeMonthsAgo.Year() == now.Year() {
		found = false
		for _, ma := range result {
			if ma.Month == int(threeMonthsAgo.Month()) {
				if ma.Amount != inv2.TotalAmount {
					t.Errorf("month %d amount = %d, want %d", ma.Month, ma.Amount, inv2.TotalAmount)
				}
				found = true
			}
		}
		if !found {
			t.Errorf("month %d not found in results", int(threeMonthsAgo.Month()))
		}
	}
}

func TestReportRepo_QuarterlyRevenue_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	result, err := repo.QuarterlyRevenue(ctx, time.Now().Year())
	if err != nil {
		t.Fatalf("QuarterlyRevenue() error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestReportRepo_YearlyRevenue_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	total, err := repo.YearlyRevenue(ctx, time.Now().Year())
	if err != nil {
		t.Fatalf("YearlyRevenue() error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0, got %d", total)
	}
}

func TestReportRepo_MonthlyExpenses_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	result, err := repo.MonthlyExpenses(ctx, time.Now().Year())
	if err != nil {
		t.Fatalf("MonthlyExpenses() error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestReportRepo_QuarterlyExpenses_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	result, err := repo.QuarterlyExpenses(ctx, time.Now().Year())
	if err != nil {
		t.Fatalf("QuarterlyExpenses() error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestReportRepo_CategoryExpenses_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	result, err := repo.CategoryExpenses(ctx, time.Now().Year())
	if err != nil {
		t.Fatalf("CategoryExpenses() error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestReportRepo_CategoryExpenses_WithData(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	now := time.Now()

	// Seed expenses in different categories with current year dates.
	testutil.SeedExpense(t, db, &domain.Expense{
		Description: "Office rent",
		Category:    "rent",
		Amount:      domain.NewAmount(15000, 0),
		IssueDate:   now,
	})
	testutil.SeedExpense(t, db, &domain.Expense{
		Description: "More rent",
		Category:    "rent",
		Amount:      domain.NewAmount(15000, 0),
		IssueDate:   now,
	})
	testutil.SeedExpense(t, db, &domain.Expense{
		Description: "Train ticket",
		Category:    "travel",
		Amount:      domain.NewAmount(2000, 0),
		IssueDate:   now,
	})

	result, err := repo.CategoryExpenses(ctx, now.Year())
	if err != nil {
		t.Fatalf("CategoryExpenses() error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(result))
	}

	// Results are ordered by total DESC, so rent should be first.
	if result[0].Category != "rent" {
		t.Errorf("first category = %q, want %q", result[0].Category, "rent")
	}
	expectedRent := domain.NewAmount(30000, 0)
	if result[0].Amount != expectedRent {
		t.Errorf("rent amount = %d, want %d", result[0].Amount, expectedRent)
	}
	if result[1].Category != "travel" {
		t.Errorf("second category = %q, want %q", result[1].Category, "travel")
	}
	expectedTravel := domain.NewAmount(2000, 0)
	if result[1].Amount != expectedTravel {
		t.Errorf("travel amount = %d, want %d", result[1].Amount, expectedTravel)
	}
}

func TestReportRepo_TopCustomers_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	result, err := repo.TopCustomers(ctx, time.Now().Year(), 10)
	if err != nil {
		t.Fatalf("TopCustomers() error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestReportRepo_TopCustomers_WithData(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	now := time.Now()

	// Create two customers.
	customer1 := testutil.SeedContact(t, db, &domain.Contact{Name: "Big Customer"})
	customer2 := testutil.SeedContact(t, db, &domain.Contact{Name: "Small Customer"})

	// Seed 2 invoices for customer1 (higher total).
	testutil.SeedInvoice(t, db, customer1.ID, []domain.InvoiceItem{
		{Description: "Big project", Quantity: 100, UnitPrice: 50000, VATRatePercent: 21},
	})
	testutil.SeedInvoice(t, db, customer1.ID, []domain.InvoiceItem{
		{Description: "Another project", Quantity: 100, UnitPrice: 30000, VATRatePercent: 21},
	})

	// Seed 1 invoice for customer2 (lower total).
	testutil.SeedInvoice(t, db, customer2.ID, []domain.InvoiceItem{
		{Description: "Small project", Quantity: 100, UnitPrice: 10000, VATRatePercent: 21},
	})

	result, err := repo.TopCustomers(ctx, now.Year(), 10)
	if err != nil {
		t.Fatalf("TopCustomers() error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 customers, got %d", len(result))
	}

	// Ordered by total DESC, so Big Customer should be first.
	if result[0].CustomerName != "Big Customer" {
		t.Errorf("first customer = %q, want %q", result[0].CustomerName, "Big Customer")
	}
	if result[0].InvoiceCount != 2 {
		t.Errorf("first customer invoice count = %d, want 2", result[0].InvoiceCount)
	}
	if result[1].CustomerName != "Small Customer" {
		t.Errorf("second customer = %q, want %q", result[1].CustomerName, "Small Customer")
	}
	if result[1].InvoiceCount != 1 {
		t.Errorf("second customer invoice count = %d, want 1", result[1].InvoiceCount)
	}
}

func TestReportRepo_QuarterlyRevenue_WithData(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	now := time.Now()
	contact := testutil.SeedContact(t, db, nil)

	testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Q service", Quantity: 100, UnitPrice: 10000, VATRatePercent: 21},
	})

	result, err := repo.QuarterlyRevenue(ctx, now.Year())
	if err != nil {
		t.Fatalf("QuarterlyRevenue() error: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected at least 1 quarterly entry")
	}
}

func TestReportRepo_MonthlyExpenses_WithData(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	now := time.Now()
	testutil.SeedExpense(t, db, &domain.Expense{
		Description: "Expense",
		Amount:      domain.NewAmount(1000, 0),
		IssueDate:   now,
		Category:    "software",
	})

	result, err := repo.MonthlyExpenses(ctx, now.Year())
	if err != nil {
		t.Fatalf("MonthlyExpenses() error: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected at least 1 monthly expense entry")
	}
}

func TestReportRepo_QuarterlyExpenses_WithData(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	now := time.Now()
	testutil.SeedExpense(t, db, &domain.Expense{
		Description: "Q expense",
		Amount:      domain.NewAmount(500, 0),
		IssueDate:   now,
		Category:    "rent",
	})

	result, err := repo.QuarterlyExpenses(ctx, now.Year())
	if err != nil {
		t.Fatalf("QuarterlyExpenses() error: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected at least 1 quarterly expense entry")
	}
}

func TestReportRepo_ProfitLossMonthly_WithData(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	now := time.Now()
	contact := testutil.SeedContact(t, db, nil)

	testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Revenue item", Quantity: 100, UnitPrice: 20000, VATRatePercent: 21},
	})
	testutil.SeedExpense(t, db, &domain.Expense{
		Description: "Cost item",
		Amount:      domain.NewAmount(5000, 0),
		IssueDate:   now,
		Category:    "hosting",
	})

	revenue, expenses, err := repo.ProfitLossMonthly(ctx, now.Year())
	if err != nil {
		t.Fatalf("ProfitLossMonthly() error: %v", err)
	}
	if len(revenue) == 0 {
		t.Error("expected non-empty revenue")
	}
	if len(expenses) == 0 {
		t.Error("expected non-empty expenses")
	}
}

func TestReportRepo_ProfitLossMonthly_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	revenue, expenses, err := repo.ProfitLossMonthly(ctx, time.Now().Year())
	if err != nil {
		t.Fatalf("ProfitLossMonthly() error: %v", err)
	}
	if len(revenue) != 0 {
		t.Errorf("expected empty revenue slice, got %d items", len(revenue))
	}
	if len(expenses) != 0 {
		t.Errorf("expected empty expenses slice, got %d items", len(expenses))
	}
}
