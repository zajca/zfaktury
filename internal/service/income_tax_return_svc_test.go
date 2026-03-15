package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/calc"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func setupIncomeTaxSvc(t *testing.T) (*IncomeTaxReturnService, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	itrRepo := repository.NewIncomeTaxReturnRepository(db)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	setRepo := repository.NewSettingsRepository(db)
	tysRepo := repository.NewTaxYearSettingsRepository(db)
	tpRepo := repository.NewTaxPrepaymentRepository(db)
	svc := NewIncomeTaxReturnService(itrRepo, invRepo, expRepo, setRepo, tysRepo, tpRepo, nil, nil)
	return svc, db
}

func TestIncomeTaxReturnService_Create(t *testing.T) {
	svc, _ := setupIncomeTaxSvc(t)
	ctx := context.Background()

	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}

	if err := svc.Create(ctx, itr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if itr.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	if itr.Status != domain.FilingStatusDraft {
		t.Errorf("Status = %q, want %q", itr.Status, domain.FilingStatusDraft)
	}
}

func TestIncomeTaxReturnService_Create_DefaultFilingType(t *testing.T) {
	svc, _ := setupIncomeTaxSvc(t)
	ctx := context.Background()

	itr := &domain.IncomeTaxReturn{
		Year: 2025,
	}

	if err := svc.Create(ctx, itr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if itr.FilingType != domain.FilingTypeRegular {
		t.Errorf("FilingType = %q, want %q", itr.FilingType, domain.FilingTypeRegular)
	}
}

func TestIncomeTaxReturnService_Create_DuplicateRegular(t *testing.T) {
	svc, _ := setupIncomeTaxSvc(t)
	ctx := context.Background()

	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, itr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	itr2 := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}
	err := svc.Create(ctx, itr2)
	if err == nil {
		t.Error("expected error for duplicate regular filing")
	}
	if !errors.Is(err, domain.ErrFilingAlreadyExists) {
		t.Errorf("expected ErrFilingAlreadyExists, got: %v", err)
	}
}

func TestIncomeTaxReturnService_Create_InvalidYear(t *testing.T) {
	svc, _ := setupIncomeTaxSvc(t)
	ctx := context.Background()

	for _, year := range []int{1999, 2101} {
		itr := &domain.IncomeTaxReturn{
			Year:       year,
			FilingType: domain.FilingTypeRegular,
		}
		err := svc.Create(ctx, itr)
		if err == nil {
			t.Errorf("expected error for year %d", year)
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput for year %d, got: %v", year, err)
		}
	}
}

func TestIncomeTaxReturnService_Create_InvalidFilingType(t *testing.T) {
	svc, _ := setupIncomeTaxSvc(t)
	ctx := context.Background()

	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: "invalid",
	}
	err := svc.Create(ctx, itr)
	if err == nil {
		t.Error("expected error for invalid filing_type")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestIncomeTaxReturnService_GetByID(t *testing.T) {
	svc, _ := setupIncomeTaxSvc(t)
	ctx := context.Background()

	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, itr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := svc.GetByID(ctx, itr.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Year != 2025 {
		t.Errorf("Year = %d, want 2025", got.Year)
	}
}

func TestIncomeTaxReturnService_GetByID_ZeroID(t *testing.T) {
	svc, _ := setupIncomeTaxSvc(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestIncomeTaxReturnService_Delete(t *testing.T) {
	svc, _ := setupIncomeTaxSvc(t)
	ctx := context.Background()

	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, itr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.Delete(ctx, itr.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := svc.GetByID(ctx, itr.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestIncomeTaxReturnService_Delete_Filed(t *testing.T) {
	svc, _ := setupIncomeTaxSvc(t)
	ctx := context.Background()

	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, itr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// Mark as filed.
	if _, err := svc.MarkFiled(ctx, itr.ID); err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}

	err := svc.Delete(ctx, itr.ID)
	if err == nil {
		t.Error("expected error when deleting filed return")
	}
	if !errors.Is(err, domain.ErrFilingAlreadyFiled) {
		t.Errorf("expected ErrFilingAlreadyFiled, got: %v", err)
	}
}

func TestIncomeTaxReturnService_Delete_ZeroID(t *testing.T) {
	svc, _ := setupIncomeTaxSvc(t)
	ctx := context.Background()

	err := svc.Delete(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestIncomeTaxReturnService_List(t *testing.T) {
	svc, _ := setupIncomeTaxSvc(t)
	ctx := context.Background()

	for _, ft := range []string{domain.FilingTypeRegular, domain.FilingTypeCorrective} {
		itr := &domain.IncomeTaxReturn{
			Year:       2025,
			FilingType: ft,
		}
		if err := svc.Create(ctx, itr); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	returns, err := svc.List(ctx, 2025)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(returns) != 2 {
		t.Errorf("List() returned %d items, want 2", len(returns))
	}
}

func TestIncomeTaxReturnService_List_DefaultYear(t *testing.T) {
	svc, _ := setupIncomeTaxSvc(t)
	ctx := context.Background()

	// year=0 should default to current year and not error.
	_, err := svc.List(ctx, 0)
	if err != nil {
		t.Fatalf("List(0) error: %v", err)
	}
}

func TestIncomeTaxReturnService_MarkFiled(t *testing.T) {
	svc, _ := setupIncomeTaxSvc(t)
	ctx := context.Background()

	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, itr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	filed, err := svc.MarkFiled(ctx, itr.ID)
	if err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}
	if filed.Status != domain.FilingStatusFiled {
		t.Errorf("Status = %q, want %q", filed.Status, domain.FilingStatusFiled)
	}
	if filed.FiledAt == nil {
		t.Error("expected FiledAt to be set")
	}

	// Cannot mark as filed again.
	_, err = svc.MarkFiled(ctx, itr.ID)
	if err == nil {
		t.Error("expected error for double filing")
	}
	if !errors.Is(err, domain.ErrFilingAlreadyFiled) {
		t.Errorf("expected ErrFilingAlreadyFiled, got: %v", err)
	}
}

func TestIncomeTaxReturnService_Recalculate_ZeroID(t *testing.T) {
	svc, _ := setupIncomeTaxSvc(t)
	ctx := context.Background()

	_, err := svc.Recalculate(ctx, 0)
	if err == nil {
		t.Error("Recalculate(0) should return error")
	}
}

func TestIncomeTaxReturnService_Recalculate_Filed(t *testing.T) {
	svc, _ := setupIncomeTaxSvc(t)
	ctx := context.Background()

	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, itr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if _, err := svc.MarkFiled(ctx, itr.ID); err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}

	_, err := svc.Recalculate(ctx, itr.ID)
	if err == nil {
		t.Error("Recalculate() should return error for filed return")
	}
	if !errors.Is(err, domain.ErrFilingAlreadyFiled) {
		t.Errorf("expected ErrFilingAlreadyFiled, got: %v", err)
	}
}

func TestIncomeTaxReturnService_Recalculate_WithInvoice(t *testing.T) {
	svc, db := setupIncomeTaxSvc(t)
	ctx := context.Background()

	// Seed a customer contact.
	contact := testutil.SeedContact(t, db, nil)

	// Seed an invoice with specific date in 2025, status=sent.
	jan15 := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	now := time.Now()
	_, err := db.ExecContext(ctx, `
		INSERT INTO invoices (
			invoice_number, type, status,
			issue_date, due_date, delivery_date, variable_symbol, constant_symbol,
			customer_id, currency_code, exchange_rate,
			payment_method, bank_account, bank_code, iban, swift,
			subtotal_amount, vat_amount, total_amount, paid_amount,
			notes, internal_notes,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"FV20250001", domain.InvoiceTypeRegular, domain.InvoiceStatusSent,
		jan15.Format("2006-01-02"), jan15.AddDate(0, 0, 14).Format("2006-01-02"), jan15.Format("2006-01-02"),
		"", "", contact.ID, "CZK", 100,
		"bank_transfer", "", "", "", "",
		domain.NewAmount(100000, 0), domain.NewAmount(21000, 0), domain.NewAmount(121000, 0), 0,
		"", "",
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("inserting invoice: %v", err)
	}

	// Set up tax year settings with 60% flat rate.
	tysRepo := repository.NewTaxYearSettingsRepository(db)
	if err := tysRepo.Upsert(ctx, &domain.TaxYearSettings{Year: 2025, FlatRatePercent: 60}); err != nil {
		t.Fatalf("Upsert tax year settings: %v", err)
	}

	// Create and recalculate.
	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, itr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	result, err := svc.Recalculate(ctx, itr.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	// Revenue = 100000 CZK = 10000000000 halere ... wait, NewAmount(100000, 0) = 10000000 halere.
	// Actually domain.NewAmount(100000, 0) = 100000 * 100 = 10000000 halere.
	// The invoice subtotal_amount is stored as domain.NewAmount(100000, 0).
	expectedRevenue := domain.NewAmount(100000, 0)
	if result.TotalRevenue != expectedRevenue {
		t.Errorf("TotalRevenue = %d, want %d", result.TotalRevenue, expectedRevenue)
	}

	// Flat rate at 60%: 10000000 * 0.6 = 6000000 halere.
	expectedFlatRate := expectedRevenue.Multiply(0.6)
	if result.FlatRateAmount != expectedFlatRate {
		t.Errorf("FlatRateAmount = %d, want %d", result.FlatRateAmount, expectedFlatRate)
	}
	if result.FlatRatePercent != 60 {
		t.Errorf("FlatRatePercent = %d, want 60", result.FlatRatePercent)
	}

	// TaxBase = Revenue - FlatRateAmount.
	expectedTaxBase := expectedRevenue - expectedFlatRate
	if result.TaxBase != expectedTaxBase {
		t.Errorf("TaxBase = %d, want %d", result.TaxBase, expectedTaxBase)
	}

	// CreditBasic should be set from constants.
	constants, _ := calc.GetTaxConstants(2025)
	if result.CreditBasic != constants.BasicCredit {
		t.Errorf("CreditBasic = %d, want %d", result.CreditBasic, constants.BasicCredit)
	}
}

func TestIncomeTaxReturnService_Recalculate_ActualExpenses(t *testing.T) {
	svc, db := setupIncomeTaxSvc(t)
	ctx := context.Background()

	// Seed contact and invoice.
	contact := testutil.SeedContact(t, db, nil)
	now := time.Now()
	jan15 := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	_, err := db.ExecContext(ctx, `
		INSERT INTO invoices (
			invoice_number, type, status,
			issue_date, due_date, delivery_date, variable_symbol, constant_symbol,
			customer_id, currency_code, exchange_rate,
			payment_method, bank_account, bank_code, iban, swift,
			subtotal_amount, vat_amount, total_amount, paid_amount,
			notes, internal_notes,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"FV20250002", domain.InvoiceTypeRegular, domain.InvoiceStatusPaid,
		jan15.Format("2006-01-02"), jan15.AddDate(0, 0, 14).Format("2006-01-02"), jan15.Format("2006-01-02"),
		"", "", contact.ID, "CZK", 100,
		"bank_transfer", "", "", "", "",
		domain.NewAmount(200000, 0), 0, domain.NewAmount(200000, 0), 0,
		"", "",
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("inserting invoice: %v", err)
	}

	// Seed a tax-reviewed expense.
	exp := testutil.SeedExpense(t, db, &domain.Expense{
		Description:     "Office rent",
		IssueDate:       jan15,
		Amount:          domain.NewAmount(50000, 0),
		VATAmount:       0,
		BusinessPercent: 100,
	})
	expRepo := repository.NewExpenseRepository(db)
	if err := expRepo.MarkTaxReviewed(ctx, []int64{exp.ID}); err != nil {
		t.Fatalf("MarkTaxReviewed: %v", err)
	}

	// No flat rate configured, so actual expenses should be used.
	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, itr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	result, err := svc.Recalculate(ctx, itr.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	if result.TotalRevenue != domain.NewAmount(200000, 0) {
		t.Errorf("TotalRevenue = %d, want %d", result.TotalRevenue, domain.NewAmount(200000, 0))
	}
	if result.ActualExpenses != domain.NewAmount(50000, 0) {
		t.Errorf("ActualExpenses = %d, want %d", result.ActualExpenses, domain.NewAmount(50000, 0))
	}
	if result.FlatRatePercent != 0 {
		t.Errorf("FlatRatePercent = %d, want 0 (actual)", result.FlatRatePercent)
	}
	if result.UsedExpenses != domain.NewAmount(50000, 0) {
		t.Errorf("UsedExpenses = %d, want %d (actual)", result.UsedExpenses, domain.NewAmount(50000, 0))
	}

	expectedTaxBase := domain.NewAmount(200000, 0) - domain.NewAmount(50000, 0)
	if result.TaxBase != expectedTaxBase {
		t.Errorf("TaxBase = %d, want %d", result.TaxBase, expectedTaxBase)
	}
}

func TestIncomeTaxReturnService_Recalculate_WithPrepayments(t *testing.T) {
	svc, db := setupIncomeTaxSvc(t)
	ctx := context.Background()

	// Add tax prepayments for 2025.
	tpRepo := repository.NewTaxPrepaymentRepository(db)
	prepayments := []domain.TaxPrepayment{
		{Month: 1, TaxAmount: domain.NewAmount(5000, 0), SocialAmount: 0, HealthAmount: 0},
		{Month: 2, TaxAmount: domain.NewAmount(5000, 0), SocialAmount: 0, HealthAmount: 0},
	}
	if err := tpRepo.UpsertAll(ctx, 2025, prepayments); err != nil {
		t.Fatalf("UpsertAll prepayments: %v", err)
	}

	itr := &domain.IncomeTaxReturn{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}
	if err := svc.Create(ctx, itr); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	result, err := svc.Recalculate(ctx, itr.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	expectedPrepayments := domain.NewAmount(10000, 0)
	if result.Prepayments != expectedPrepayments {
		t.Errorf("Prepayments = %d, want %d", result.Prepayments, expectedPrepayments)
	}
}
