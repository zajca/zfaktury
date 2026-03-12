package service

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestReportService_RevenueReport_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewReportRepository(db)
	svc := NewReportService(repo)
	ctx := context.Background()

	report, err := svc.RevenueReport(ctx, time.Now().Year())
	if err != nil {
		t.Fatalf("RevenueReport() error: %v", err)
	}

	if report.Year != time.Now().Year() {
		t.Errorf("Year = %d, want %d", report.Year, time.Now().Year())
	}
	if report.Total != 0 {
		t.Errorf("Total = %d, want 0", report.Total)
	}
	if len(report.Monthly) != 0 {
		t.Errorf("Monthly len = %d, want 0", len(report.Monthly))
	}
	if len(report.Quarterly) != 0 {
		t.Errorf("Quarterly len = %d, want 0", len(report.Quarterly))
	}
}

func TestReportService_RevenueReport_WithData(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewReportRepository(db)
	svc := NewReportService(repo)
	ctx := context.Background()

	contact := testutil.SeedContact(t, db, nil)
	year := time.Now().Year()

	// Seed invoices with known amounts (SeedInvoice uses time.Now() for delivery_date).
	items1 := []domain.InvoiceItem{
		{
			Description:    "Consulting",
			Quantity:       100,
			Unit:           "hod",
			UnitPrice:      domain.NewAmount(1000, 0),
			VATRatePercent: 0,
		},
	}
	inv1 := testutil.SeedInvoice(t, db, contact.ID, items1)

	items2 := []domain.InvoiceItem{
		{
			Description:    "Development",
			Quantity:       100,
			Unit:           "hod",
			UnitPrice:      domain.NewAmount(3000, 0),
			VATRatePercent: 0,
		},
	}
	inv2 := testutil.SeedInvoice(t, db, contact.ID, items2)

	report, err := svc.RevenueReport(ctx, year)
	if err != nil {
		t.Fatalf("RevenueReport() error: %v", err)
	}

	expectedTotal := inv1.TotalAmount + inv2.TotalAmount
	if report.Total != expectedTotal {
		t.Errorf("Total = %d, want %d", report.Total, expectedTotal)
	}

	// Monthly should have at least one entry for the current month.
	if len(report.Monthly) == 0 {
		t.Error("Monthly should have at least one entry")
	}

	// Quarterly should have at least one entry.
	if len(report.Quarterly) == 0 {
		t.Error("Quarterly should have at least one entry")
	}

	// Verify quarterly amount matches total (all invoices are in the same quarter).
	currentQuarter := (int(time.Now().Month())-1)/3 + 1
	for _, q := range report.Quarterly {
		if q.Quarter == currentQuarter {
			if q.Amount != expectedTotal {
				t.Errorf("Quarterly[Q%d] = %d, want %d", currentQuarter, q.Amount, expectedTotal)
			}
		}
	}
}

func TestReportService_ExpenseReport_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewReportRepository(db)
	svc := NewReportService(repo)
	ctx := context.Background()

	report, err := svc.ExpenseReport(ctx, time.Now().Year())
	if err != nil {
		t.Fatalf("ExpenseReport() error: %v", err)
	}

	if report.Year != time.Now().Year() {
		t.Errorf("Year = %d, want %d", report.Year, time.Now().Year())
	}
	if len(report.Monthly) != 0 {
		t.Errorf("Monthly len = %d, want 0", len(report.Monthly))
	}
	if len(report.Quarterly) != 0 {
		t.Errorf("Quarterly len = %d, want 0", len(report.Quarterly))
	}
	if len(report.Categories) != 0 {
		t.Errorf("Categories len = %d, want 0", len(report.Categories))
	}
}

func TestReportService_TopCustomers_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewReportRepository(db)
	svc := NewReportService(repo)
	ctx := context.Background()

	customers, err := svc.TopCustomers(ctx, time.Now().Year())
	if err != nil {
		t.Fatalf("TopCustomers() error: %v", err)
	}

	if len(customers) != 0 {
		t.Errorf("TopCustomers len = %d, want 0", len(customers))
	}
}

func TestReportService_ProfitLoss_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewReportRepository(db)
	svc := NewReportService(repo)
	ctx := context.Background()

	report, err := svc.ProfitLoss(ctx, time.Now().Year())
	if err != nil {
		t.Fatalf("ProfitLoss() error: %v", err)
	}

	if report.Year != time.Now().Year() {
		t.Errorf("Year = %d, want %d", report.Year, time.Now().Year())
	}
	if len(report.MonthlyRevenue) != 0 {
		t.Errorf("MonthlyRevenue len = %d, want 0", len(report.MonthlyRevenue))
	}
	if len(report.MonthlyExpenses) != 0 {
		t.Errorf("MonthlyExpenses len = %d, want 0", len(report.MonthlyExpenses))
	}
}
