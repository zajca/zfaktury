package repository

import (
	"context"
	"database/sql"
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
	// Revenue counts only sent/paid/overdue invoices.
	testutil.MarkInvoiceSent(t, db, inv1.ID)
	testutil.MarkInvoiceSent(t, db, inv2.ID)
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
	inv1 := testutil.SeedInvoice(t, db, customer1.ID, []domain.InvoiceItem{
		{Description: "Big project", Quantity: 100, UnitPrice: 50000, VATRatePercent: 21},
	})
	inv2 := testutil.SeedInvoice(t, db, customer1.ID, []domain.InvoiceItem{
		{Description: "Another project", Quantity: 100, UnitPrice: 30000, VATRatePercent: 21},
	})

	// Seed 1 invoice for customer2 (lower total).
	inv3 := testutil.SeedInvoice(t, db, customer2.ID, []domain.InvoiceItem{
		{Description: "Small project", Quantity: 100, UnitPrice: 10000, VATRatePercent: 21},
	})
	// Revenue counts only sent/paid/overdue invoices.
	testutil.MarkInvoiceSent(t, db, inv1.ID)
	testutil.MarkInvoiceSent(t, db, inv2.ID)
	testutil.MarkInvoiceSent(t, db, inv3.ID)

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

	inv := testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Q service", Quantity: 100, UnitPrice: 10000, VATRatePercent: 21},
	})
	testutil.MarkInvoiceSent(t, db, inv.ID)

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

	inv := testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Revenue item", Quantity: 100, UnitPrice: 20000, VATRatePercent: 21},
	})
	testutil.MarkInvoiceSent(t, db, inv.ID)
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

// seedRevenueWithCreditNotes seeds one regular invoice (total 121000), one
// native credit note stored negative (total -24200) and one imported credit
// note stored positive (total 12100, Fakturoid "correction"), all delivered on
// deliveryDate. It returns the customer ID and the net revenue the reports must
// show: 121000 - 24200 - 12100 = 84700.
func seedRevenueWithCreditNotes(t *testing.T, db *sql.DB, ctx context.Context, deliveryDate time.Time) (customerID int64, net domain.Amount) {
	t.Helper()

	contact := testutil.SeedContact(t, db, nil)
	dateStr := deliveryDate.Format("2006-01-02")

	inv := testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Item", Quantity: 100, UnitPrice: 100000, VATRatePercent: 21},
	})
	if _, err := db.ExecContext(ctx, "UPDATE invoices SET status = ?, issue_date = ?, delivery_date = ? WHERE id = ?",
		domain.InvoiceStatusSent, dateStr, dateStr, inv.ID); err != nil {
		t.Fatalf("updating invoice dates: %v", err)
	}

	// Native credit note: stored negative (negated unit prices).
	testutil.SeedCreditNote(t, db, contact.ID, deliveryDate, domain.InvoiceStatusPaid, []domain.InvoiceItem{
		{Description: "Credit", Quantity: 100, UnitPrice: 20000, VATRatePercent: 21},
	}, true)
	// Imported credit note (Fakturoid "correction"): stored positive.
	testutil.SeedCreditNote(t, db, contact.ID, deliveryDate, domain.InvoiceStatusPaid, []domain.InvoiceItem{
		{Description: "Correction", Quantity: 100, UnitPrice: 10000, VATRatePercent: 21},
	}, false)

	return contact.ID, inv.TotalAmount - 24200 - 12100
}

func TestReportRepo_MonthlyRevenue_CreditNotesNet(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	_, net := seedRevenueWithCreditNotes(t, db, ctx, time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC))

	result, err := repo.MonthlyRevenue(ctx, 2025)
	if err != nil {
		t.Fatalf("MonthlyRevenue() error: %v", err)
	}
	found := false
	for _, ma := range result {
		if ma.Month == 3 {
			found = true
			if ma.Amount != net {
				t.Errorf("March revenue = %d, want %d (net of credit notes)", ma.Amount, net)
			}
		}
	}
	if !found {
		t.Error("March not found in monthly revenue")
	}
}

func TestReportRepo_QuarterlyRevenue_CreditNotesNet(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	_, net := seedRevenueWithCreditNotes(t, db, ctx, time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC))

	result, err := repo.QuarterlyRevenue(ctx, 2025)
	if err != nil {
		t.Fatalf("QuarterlyRevenue() error: %v", err)
	}
	found := false
	for _, qa := range result {
		if qa.Quarter == 1 {
			found = true
			if qa.Amount != net {
				t.Errorf("Q1 revenue = %d, want %d (net of credit notes)", qa.Amount, net)
			}
		}
	}
	if !found {
		t.Error("Q1 not found in quarterly revenue")
	}
}

func TestReportRepo_YearlyRevenue_CreditNotesNet(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	_, net := seedRevenueWithCreditNotes(t, db, ctx, time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC))

	total, err := repo.YearlyRevenue(ctx, 2025)
	if err != nil {
		t.Fatalf("YearlyRevenue() error: %v", err)
	}
	if total != net {
		t.Errorf("yearly revenue = %d, want %d (net of credit notes)", total, net)
	}
}

func TestReportRepo_TopCustomers_CreditNotesNet(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	customerID, net := seedRevenueWithCreditNotes(t, db, ctx, time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC))

	result, err := repo.TopCustomers(ctx, 2025, 10)
	if err != nil {
		t.Fatalf("TopCustomers() error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 customer, got %d", len(result))
	}
	if result[0].CustomerID != customerID {
		t.Errorf("customer ID = %d, want %d", result[0].CustomerID, customerID)
	}
	if result[0].Total != net {
		t.Errorf("customer revenue = %d, want %d (net of credit notes)", result[0].Total, net)
	}
	if result[0].InvoiceCount != 3 {
		t.Errorf("invoice count = %d, want 3 (invoice + 2 credit notes)", result[0].InvoiceCount)
	}
}

// seedRevenueExcludingDraftsAndCancelled seeds a sent regular invoice (total
// 121000) plus documents that must NOT count as revenue: a draft regular
// invoice (total 60500), a draft credit note stored positive (total 12100,
// imported style) and a cancelled credit note stored negative (total -24200,
// native style), all delivered on deliveryDate. It returns the revenue the
// reports must show: only the sent invoice, 121000.
func seedRevenueExcludingDraftsAndCancelled(t *testing.T, db *sql.DB, ctx context.Context, deliveryDate time.Time) domain.Amount {
	t.Helper()

	contact := testutil.SeedContact(t, db, nil)
	dateStr := deliveryDate.Format("2006-01-02")

	// Sent regular invoice: counts.
	inv := testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Item", Quantity: 100, UnitPrice: 100000, VATRatePercent: 21},
	})
	if _, err := db.ExecContext(ctx, "UPDATE invoices SET status = ?, issue_date = ?, delivery_date = ? WHERE id = ?",
		domain.InvoiceStatusSent, dateStr, dateStr, inv.ID); err != nil {
		t.Fatalf("updating invoice: %v", err)
	}

	// Draft regular invoice: must not count.
	draft := testutil.SeedInvoice(t, db, contact.ID, []domain.InvoiceItem{
		{Description: "Draft item", Quantity: 100, UnitPrice: 50000, VATRatePercent: 21},
	})
	if _, err := db.ExecContext(ctx, "UPDATE invoices SET issue_date = ?, delivery_date = ? WHERE id = ?",
		dateStr, dateStr, draft.ID); err != nil {
		t.Fatalf("updating draft invoice dates: %v", err)
	}

	// Draft credit note stored positive (Fakturoid "correction" style): must not count.
	testutil.SeedCreditNote(t, db, contact.ID, deliveryDate, domain.InvoiceStatusDraft, []domain.InvoiceItem{
		{Description: "Draft correction", Quantity: 100, UnitPrice: 10000, VATRatePercent: 21},
	}, false)
	// Cancelled credit note stored negative (native style): must not count.
	testutil.SeedCreditNote(t, db, contact.ID, deliveryDate, domain.InvoiceStatusCancelled, []domain.InvoiceItem{
		{Description: "Cancelled credit", Quantity: 100, UnitPrice: 20000, VATRatePercent: 21},
	}, true)

	return inv.TotalAmount
}

func TestReportRepo_Revenue_ExcludesDraftAndCancelled(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	want := seedRevenueExcludingDraftsAndCancelled(t, db, ctx, time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC))

	monthly, err := repo.MonthlyRevenue(ctx, 2025)
	if err != nil {
		t.Fatalf("MonthlyRevenue() error: %v", err)
	}
	for _, ma := range monthly {
		if ma.Month == 3 && ma.Amount != want {
			t.Errorf("March revenue = %d, want %d (only sent/paid/overdue count)", ma.Amount, want)
		}
	}

	yearly, err := repo.YearlyRevenue(ctx, 2025)
	if err != nil {
		t.Fatalf("YearlyRevenue() error: %v", err)
	}
	if yearly != want {
		t.Errorf("yearly revenue = %d, want %d (only sent/paid/overdue count)", yearly, want)
	}

	customers, err := repo.TopCustomers(ctx, 2025, 10)
	if err != nil {
		t.Fatalf("TopCustomers() error: %v", err)
	}
	if len(customers) != 1 {
		t.Fatalf("expected 1 customer, got %d", len(customers))
	}
	if customers[0].Total != want {
		t.Errorf("customer revenue = %d, want %d (only sent/paid/overdue count)", customers[0].Total, want)
	}
	if customers[0].InvoiceCount != 1 {
		t.Errorf("invoice count = %d, want 1 (only the sent invoice)", customers[0].InvoiceCount)
	}
}

func TestReportRepo_ProfitLossMonthly_CreditNotesNet(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewReportRepository(db)
	ctx := context.Background()

	_, net := seedRevenueWithCreditNotes(t, db, ctx, time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC))

	revenue, _, err := repo.ProfitLossMonthly(ctx, 2025)
	if err != nil {
		t.Fatalf("ProfitLossMonthly() error: %v", err)
	}
	found := false
	for _, ma := range revenue {
		if ma.Month == 3 {
			found = true
			if ma.Amount != net {
				t.Errorf("March profit/loss revenue = %d, want %d (net of credit notes)", ma.Amount, net)
			}
		}
	}
	if !found {
		t.Error("March not found in profit/loss revenue")
	}
}
