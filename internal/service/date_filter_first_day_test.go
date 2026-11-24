package service

import (
	"context"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/fakturoid"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestVATReturnService_Recalculate_IncludesExpenseOnFirstDayOfPeriod(t *testing.T) {
	db := testutil.NewTestDB(t)
	vatRepo := repository.NewVATReturnRepository(db)
	svc := NewVATReturnService(vatRepo, repository.NewInvoiceRepository(db),
		repository.NewExpenseRepository(db), repository.NewSettingsRepository(db), nil)
	ctx := context.Background()

	vendor := testutil.SeedContact(t, db, &domain.Contact{Name: "Vendor s.r.o.", DIC: "CZ55667788"})
	expense := testutil.SeedExpense(t, db, &domain.Expense{
		VendorID:        &vendor.ID,
		ExpenseNumber:   "VF2025101",
		Description:     "First day purchase",
		IssueDate:       time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		Amount:          domain.NewAmount(12100, 0),
		VATRatePercent:  21,
		VATAmount:       domain.NewAmount(2100, 0),
		IsTaxDeductible: true,
		BusinessPercent: 100,
	})

	vr := &domain.VATReturn{
		Period:     domain.TaxPeriod{Year: 2025, Month: 3},
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, vr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	result, err := svc.Recalculate(ctx, vr.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}
	if result.TotalInputVAT != domain.NewAmount(2100, 0) {
		t.Errorf("TotalInputVAT = %d, want %d", result.TotalInputVAT, domain.NewAmount(2100, 0))
	}

	linked, err := vatRepo.GetLinkedExpenseIDs(ctx, vr.ID)
	if err != nil {
		t.Fatalf("GetLinkedExpenseIDs() error: %v", err)
	}
	if len(linked) != 1 || linked[0] != expense.ID {
		t.Errorf("linked expense IDs = %v, want [%d]", linked, expense.ID)
	}
}

func TestFakturoidImportService_PreviewExpense_FindsExistingExpenseByIssueDate(t *testing.T) {
	db := testutil.NewTestDB(t)
	expenseRepo := repository.NewExpenseRepository(db)
	svc := NewFakturoidImportService(
		repository.NewFakturoidImportLogRepository(db), nil, nil, expenseRepo,
		nil, nil, nil, nil, nil,
	)

	existing := testutil.SeedExpense(t, db, &domain.Expense{
		ExpenseNumber: "EXP-DUP-1",
		Description:   "Already booked",
		IssueDate:     time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
	})

	item := svc.previewExpense(context.Background(), fakturoid.Expense{
		ID:             9001,
		OriginalNumber: "EXP-DUP-1",
		Description:    "Already booked",
		Total:          fakturoid.FlexFloat64(1000),
		IssuedOn:       "2025-03-15",
	}, map[int64]int64{})

	if item.Status != "duplicate" {
		t.Fatalf("Status = %q, want duplicate", item.Status)
	}
	if item.ExistingID == nil || *item.ExistingID != existing.ID {
		t.Errorf("ExistingID = %v, want %d", item.ExistingID, existing.ID)
	}
}
