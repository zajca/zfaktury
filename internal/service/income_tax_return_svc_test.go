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

// setupIncomeTaxSvcWithCredits creates the service with a wired TaxCreditsService for
// tests that exercise credits/deductions computation.
func setupIncomeTaxSvcWithCredits(t *testing.T) (*IncomeTaxReturnService, *TaxCreditsService, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	itrRepo := repository.NewIncomeTaxReturnRepository(db)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	setRepo := repository.NewSettingsRepository(db)
	tysRepo := repository.NewTaxYearSettingsRepository(db)
	tpRepo := repository.NewTaxPrepaymentRepository(db)

	spouseRepo := repository.NewTaxSpouseCreditRepository(db)
	childRepo := repository.NewTaxChildCreditRepository(db)
	personalRepo := repository.NewTaxPersonalCreditsRepository(db)
	deductionRepo := repository.NewTaxDeductionRepository(db)
	creditsSvc := NewTaxCreditsService(spouseRepo, childRepo, personalRepo, deductionRepo, nil)

	svc := NewIncomeTaxReturnService(itrRepo, invRepo, expRepo, setRepo, tysRepo, tpRepo, creditsSvc, nil)
	return svc, creditsSvc, db
}

func TestIncomeTaxReturnService_Recalculate_DeductionBreakdown(t *testing.T) {
	svc, creditsSvc, db := setupIncomeTaxSvcWithCredits(t)
	ctx := context.Background()

	// Seed a 2025 invoice so that tax base > 0 (required for donation cap = 15% of tax base).
	contact := testutil.SeedContact(t, db, nil)
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
		"FV20250100", domain.InvoiceTypeRegular, domain.InvoiceStatusSent,
		jan15.Format("2006-01-02"), jan15.AddDate(0, 0, 14).Format("2006-01-02"), jan15.Format("2006-01-02"),
		"", "", contact.ID, "CZK", 100,
		"bank_transfer", "", "", "", "",
		domain.NewAmount(500000, 0), 0, domain.NewAmount(500000, 0), 0,
		"", "",
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("inserting invoice: %v", err)
	}

	// Seed deductions across all 5 categories.
	deductions := []domain.TaxDeduction{
		{
			Year:          2025,
			Category:      domain.DeductionMortgage,
			Description:   "Hypoteka",
			ClaimedAmount: domain.NewAmount(50000, 0),
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			Year:          2025,
			Category:      domain.DeductionLifeInsurance,
			Description:   "Zivotni pojisteni",
			ClaimedAmount: domain.NewAmount(12000, 0),
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			Year:          2025,
			Category:      domain.DeductionPension,
			Description:   "Penzijni sporeni",
			ClaimedAmount: domain.NewAmount(8000, 0),
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			Year:          2025,
			Category:      domain.DeductionDonation,
			Description:   "Dar",
			ClaimedAmount: domain.NewAmount(3000, 0),
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			Year:          2025,
			Category:      domain.DeductionUnionDues,
			Description:   "Odborove prispevky",
			ClaimedAmount: domain.NewAmount(1000, 0),
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
	for i := range deductions {
		if err := creditsSvc.CreateDeduction(ctx, &deductions[i]); err != nil {
			t.Fatalf("CreateDeduction(%s) error: %v", deductions[i].Category, err)
		}
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

	// Per-category sums should match the claimed amounts (all under statutory caps).
	if result.DeductionMortgage != domain.NewAmount(50000, 0) {
		t.Errorf("DeductionMortgage = %d, want %d", result.DeductionMortgage, domain.NewAmount(50000, 0))
	}
	if result.DeductionLifeInsurance != domain.NewAmount(12000, 0) {
		t.Errorf("DeductionLifeInsurance = %d, want %d", result.DeductionLifeInsurance, domain.NewAmount(12000, 0))
	}
	if result.DeductionPension != domain.NewAmount(8000, 0) {
		t.Errorf("DeductionPension = %d, want %d", result.DeductionPension, domain.NewAmount(8000, 0))
	}
	if result.DeductionDonation != domain.NewAmount(3000, 0) {
		t.Errorf("DeductionDonation = %d, want %d", result.DeductionDonation, domain.NewAmount(3000, 0))
	}
	if result.DeductionUnionDues != domain.NewAmount(1000, 0) {
		t.Errorf("DeductionUnionDues = %d, want %d", result.DeductionUnionDues, domain.NewAmount(1000, 0))
	}

	// Total should match sum of individual categories.
	expectedTotal := result.DeductionMortgage + result.DeductionLifeInsurance +
		result.DeductionPension + result.DeductionDonation + result.DeductionUnionDues
	if result.TotalDeductions != expectedTotal {
		t.Errorf("TotalDeductions = %d, want sum %d", result.TotalDeductions, expectedTotal)
	}
}

// setupIncomeTaxSvcWithEmployment wires the service with the employment
// certificate repo for §6 aggregation tests.
func setupIncomeTaxSvcWithEmployment(t *testing.T) (*IncomeTaxReturnService, *TaxCreditsService, *repository.EmploymentCertificateRepository, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	itrRepo := repository.NewIncomeTaxReturnRepository(db)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	setRepo := repository.NewSettingsRepository(db)
	tysRepo := repository.NewTaxYearSettingsRepository(db)
	tpRepo := repository.NewTaxPrepaymentRepository(db)

	spouseRepo := repository.NewTaxSpouseCreditRepository(db)
	childRepo := repository.NewTaxChildCreditRepository(db)
	personalRepo := repository.NewTaxPersonalCreditsRepository(db)
	deductionRepo := repository.NewTaxDeductionRepository(db)
	creditsSvc := NewTaxCreditsService(spouseRepo, childRepo, personalRepo, deductionRepo, nil)
	empRepo := repository.NewEmploymentCertificateRepository(db)

	svc := NewIncomeTaxReturnService(itrRepo, invRepo, expRepo, setRepo, tysRepo, tpRepo, creditsSvc, nil)
	svc.SetEmploymentCertificateRepo(empRepo)
	return svc, creditsSvc, empRepo, db
}

// seedConfirmedCert helper inserts a confirmed certificate via the repo.
func seedConfirmedCert(t *testing.T, repo *repository.EmploymentCertificateRepository, cert *domain.EmploymentCertificate) {
	t.Helper()
	cert.Status = "confirmed"
	if err := repo.Create(context.Background(), cert); err != nil {
		t.Fatalf("seedConfirmedCert: %v", err)
	}
}

func TestIncomeTaxReturnService_Recalculate_AggregatesSection6(t *testing.T) {
	svc, _, empRepo, _ := setupIncomeTaxSvcWithEmployment(t)
	ctx := context.Background()

	// Two advance certs and one withholding cert (with include_in_dap=true).
	seedConfirmedCert(t, empRepo, &domain.EmploymentCertificate{
		Year:               2025,
		CertificateType:    domain.CertificateAdvance,
		ContractType:       domain.ContractDPC,
		EmployerName:       "Acme",
		EmployerICO:        "12345678",
		PeriodFrom:         time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodTo:           time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
		GrossIncome:        domain.NewAmount(100_000, 0),
		ForeignTaxPaid:     domain.NewAmount(2_000, 0),
		AdvanceTaxWithheld: domain.NewAmount(15_000, 0),
		MonthlyBonusPaid:   domain.NewAmount(1_500, 0),
	})
	seedConfirmedCert(t, empRepo, &domain.EmploymentCertificate{
		Year:                   2025,
		CertificateType:        domain.CertificateAdvance,
		ContractType:           domain.ContractDPP,
		EmployerName:           "Beta",
		EmployerICO:            "87654321",
		PeriodFrom:             time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		PeriodTo:               time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		GrossIncome:            domain.NewAmount(80_000, 0),
		AdvanceTaxWithheld:     domain.NewAmount(12_000, 0),
		AnnualSettlementRefund: domain.NewAmount(1_000, 0),
	})
	seedConfirmedCert(t, empRepo, &domain.EmploymentCertificate{
		Year:                    2025,
		CertificateType:         domain.CertificateWithholding,
		ContractType:            domain.ContractDPP,
		EmployerName:            "Gamma",
		EmployerICO:             "11111111",
		PeriodFrom:              time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		PeriodTo:                time.Date(2025, 4, 30, 0, 0, 0, 0, time.UTC),
		GrossIncome:             domain.NewAmount(20_000, 0),
		WithheldFinalTax:        domain.NewAmount(3_000, 0),
		IncludeWithholdingInDAP: true,
	})

	itr := &domain.IncomeTaxReturn{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, itr); err != nil {
		t.Fatalf("Create: %v", err)
	}
	result, err := svc.Recalculate(ctx, itr.ID)
	if err != nil {
		t.Fatalf("Recalculate: %v", err)
	}

	// Section6GrossIncome = 100_000 (Acme) + 80_000 (Beta) + 20_000 (Gamma withholding included).
	wantGross := domain.NewAmount(200_000, 0)
	if result.Section6GrossIncome != wantGross {
		t.Errorf("Section6GrossIncome = %d, want %d", result.Section6GrossIncome, wantGross)
	}
	// Section6ForeignTax = 2_000 (Acme).
	wantForeign := domain.NewAmount(2_000, 0)
	if result.Section6ForeignTax != wantForeign {
		t.Errorf("Section6ForeignTax = %d, want %d", result.Section6ForeignTax, wantForeign)
	}
	// Section6TaxBase = Section6GrossIncome - Section6ForeignTax = 200_000 - 2_000.
	wantBase := wantGross - wantForeign
	if result.Section6TaxBase != wantBase {
		t.Errorf("Section6TaxBase = %d, want %d", result.Section6TaxBase, wantBase)
	}
	// Section6AdvanceWithheld = (15_000 - 0) + (12_000 - 1_000) = 26_000.
	wantAdv := domain.NewAmount(26_000, 0)
	if result.Section6AdvanceWithheld != wantAdv {
		t.Errorf("Section6AdvanceWithheld = %d, want %d", result.Section6AdvanceWithheld, wantAdv)
	}
	// Section6WithholdingCredited = 3_000 (Gamma).
	wantWh := domain.NewAmount(3_000, 0)
	if result.Section6WithholdingCredited != wantWh {
		t.Errorf("Section6WithholdingCredited = %d, want %d", result.Section6WithholdingCredited, wantWh)
	}
	// Section6MonthlyBonusPaid = 1_500.
	wantBonus := domain.NewAmount(1_500, 0)
	if result.Section6MonthlyBonusPaid != wantBonus {
		t.Errorf("Section6MonthlyBonusPaid = %d, want %d", result.Section6MonthlyBonusPaid, wantBonus)
	}
	if result.Section6CertsAdvance != 2 {
		t.Errorf("Section6CertsAdvance = %d, want 2", result.Section6CertsAdvance)
	}
	if result.Section6CertsWithholding != 1 {
		t.Errorf("Section6CertsWithholding = %d, want 1", result.Section6CertsWithholding)
	}
	// Section6TaxBase should also flow into the §16 progressive tax base.
	if result.TaxBase < wantBase {
		t.Errorf("TaxBase = %d should include Section6TaxBase >= %d", result.TaxBase, wantBase)
	}
}

// TestIncomeTaxReturnService_Recalculate_WithholdingNotIncludedSkipped verifies
// the §38g(6) opt-in: certificates with include_withholding_in_dap=false are
// dropped from §6 totals.
func TestIncomeTaxReturnService_Recalculate_WithholdingNotIncludedSkipped(t *testing.T) {
	svc, _, empRepo, _ := setupIncomeTaxSvcWithEmployment(t)
	ctx := context.Background()

	seedConfirmedCert(t, empRepo, &domain.EmploymentCertificate{
		Year:                    2025,
		CertificateType:         domain.CertificateWithholding,
		ContractType:            domain.ContractDPP,
		EmployerName:            "Gamma",
		EmployerICO:             "11111111",
		PeriodFrom:              time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		PeriodTo:                time.Date(2025, 4, 30, 0, 0, 0, 0, time.UTC),
		GrossIncome:             domain.NewAmount(20_000, 0),
		WithheldFinalTax:        domain.NewAmount(3_000, 0),
		IncludeWithholdingInDAP: false,
	})

	itr := &domain.IncomeTaxReturn{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, itr); err != nil {
		t.Fatalf("Create: %v", err)
	}
	result, err := svc.Recalculate(ctx, itr.ID)
	if err != nil {
		t.Fatalf("Recalculate: %v", err)
	}
	if result.Section6GrossIncome != 0 {
		t.Errorf("Section6GrossIncome = %d, want 0 (cert opted out)", result.Section6GrossIncome)
	}
	if result.Section6CertsWithholding != 0 {
		t.Errorf("Section6CertsWithholding = %d, want 0", result.Section6CertsWithholding)
	}
}

// TestIncomeTaxReturnService_Recalculate_MonthlyBonusDoesNotReduceChildBenefit
// is a regression for K3: MonthlyBonusPaid (ř.89) is informational about
// employer-paid bonuses and must NOT be subtracted from the calculated child
// benefit (ř.72). The reconciliation happens via ř.84/87/89 vs total tax/bonus.
func TestIncomeTaxReturnService_Recalculate_MonthlyBonusDoesNotReduceChildBenefit(t *testing.T) {
	svc, creditsSvc, empRepo, _ := setupIncomeTaxSvcWithEmployment(t)
	ctx := context.Background()

	// Configure a child credit so ChildBenefit > 0.
	now := time.Now()
	childRepoCred := &domain.TaxChildCredit{
		Year:          2025,
		ChildName:     "Anna",
		BirthNumber:   "0501010001",
		ChildOrder:    1,
		MonthsClaimed: 12,
		ZTP:           false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := creditsSvc.CreateChild(ctx, childRepoCred); err != nil {
		t.Fatalf("CreateChild: %v", err)
	}

	// Confirmed advance cert with monthly bonus paid.
	seedConfirmedCert(t, empRepo, &domain.EmploymentCertificate{
		Year:               2025,
		CertificateType:    domain.CertificateAdvance,
		ContractType:       domain.ContractHPP,
		EmployerName:       "Acme",
		EmployerICO:        "12345678",
		PeriodFrom:         time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodTo:           time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		GrossIncome:        domain.NewAmount(300_000, 0),
		AdvanceTaxWithheld: domain.NewAmount(45_000, 0),
		MonthlyBonusPaid:   domain.NewAmount(10_000, 0),
	})

	itr := &domain.IncomeTaxReturn{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, itr); err != nil {
		t.Fatalf("Create: %v", err)
	}
	result, err := svc.Recalculate(ctx, itr.ID)
	if err != nil {
		t.Fatalf("Recalculate: %v", err)
	}

	// MonthlyBonusPaid is captured.
	wantBonus := domain.NewAmount(10_000, 0)
	if result.Section6MonthlyBonusPaid != wantBonus {
		t.Errorf("Section6MonthlyBonusPaid = %d, want %d", result.Section6MonthlyBonusPaid, wantBonus)
	}
	// ChildBenefit must equal the full computed annual amount, untouched by
	// the §6 employer-paid bonus.
	expectedChildBenefit, err := creditsSvc.ComputeChildBenefit(ctx, 2025)
	if err != nil {
		t.Fatalf("ComputeChildBenefit: %v", err)
	}
	if result.ChildBenefit != expectedChildBenefit {
		t.Errorf("ChildBenefit = %d, want %d (must NOT be reduced by Section6MonthlyBonusPaid)", result.ChildBenefit, expectedChildBenefit)
	}
}

// TestIncomeTaxReturnService_Recalculate_Section6CertsBonusStaysZero is a
// regression for N5: Section6CertsBonus tracks the count of standalone
// "Potvrzení o vyplaceném daňovém bonusu" forms, NOT advance certs that
// happen to include MonthlyBonusPaid > 0. MVP keeps it at 0.
func TestIncomeTaxReturnService_Recalculate_Section6CertsBonusStaysZero(t *testing.T) {
	svc, _, empRepo, _ := setupIncomeTaxSvcWithEmployment(t)
	ctx := context.Background()

	seedConfirmedCert(t, empRepo, &domain.EmploymentCertificate{
		Year:               2025,
		CertificateType:    domain.CertificateAdvance,
		ContractType:       domain.ContractHPP,
		EmployerName:       "Acme",
		EmployerICO:        "12345678",
		PeriodFrom:         time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodTo:           time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		GrossIncome:        domain.NewAmount(300_000, 0),
		AdvanceTaxWithheld: domain.NewAmount(45_000, 0),
		MonthlyBonusPaid:   domain.NewAmount(10_000, 0),
	})

	itr := &domain.IncomeTaxReturn{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, itr); err != nil {
		t.Fatalf("Create: %v", err)
	}
	result, err := svc.Recalculate(ctx, itr.ID)
	if err != nil {
		t.Fatalf("Recalculate: %v", err)
	}
	if result.Section6CertsBonus != 0 {
		t.Errorf("Section6CertsBonus = %d, want 0 (advance certs with bonus must NOT bump this counter)", result.Section6CertsBonus)
	}
	if result.Section6CertsAdvance != 1 {
		t.Errorf("Section6CertsAdvance = %d, want 1", result.Section6CertsAdvance)
	}
}

// TestRecalculate_EmitsProgressiveRateWarning verifies the
// WarningProgressiveRateReview token is appended when consolidated tax base
// crosses 36× průměrná mzda for 2025 (1 676 052 Kč).
func TestRecalculate_EmitsProgressiveRateWarning(t *testing.T) {
	svc, _, empRepo, _ := setupIncomeTaxSvcWithEmployment(t)
	ctx := context.Background()

	// Single advance cert with gross income > threshold.
	seedConfirmedCert(t, empRepo, &domain.EmploymentCertificate{
		Year:               2025,
		CertificateType:    domain.CertificateAdvance,
		ContractType:       domain.ContractHPP,
		EmployerName:       "BigPay",
		EmployerICO:        "27082440",
		PeriodFrom:         time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodTo:           time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		GrossIncome:        domain.NewAmount(2_000_000, 0), // > 1 676 052 Kč
		AdvanceTaxWithheld: domain.NewAmount(300_000, 0),
	})

	itr := &domain.IncomeTaxReturn{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, itr); err != nil {
		t.Fatalf("Create: %v", err)
	}
	result, err := svc.Recalculate(ctx, itr.ID)
	if err != nil {
		t.Fatalf("Recalculate: %v", err)
	}

	found := false
	for _, w := range result.Warnings {
		if w == domain.WarningProgressiveRateReview {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Warnings = %v, want to contain %q", result.Warnings, domain.WarningProgressiveRateReview)
	}
}

// TestRecalculate_NoWarningBelowThreshold verifies no warning is emitted
// when the consolidated base sits comfortably below 36× průměrná mzda.
func TestRecalculate_NoWarningBelowThreshold(t *testing.T) {
	svc, _, empRepo, _ := setupIncomeTaxSvcWithEmployment(t)
	ctx := context.Background()

	seedConfirmedCert(t, empRepo, &domain.EmploymentCertificate{
		Year:               2025,
		CertificateType:    domain.CertificateAdvance,
		ContractType:       domain.ContractDPC,
		EmployerName:       "SmallPay",
		EmployerICO:        "27082440",
		PeriodFrom:         time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodTo:           time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		GrossIncome:        domain.NewAmount(200_000, 0),
		AdvanceTaxWithheld: domain.NewAmount(30_000, 0),
	})

	itr := &domain.IncomeTaxReturn{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, itr); err != nil {
		t.Fatalf("Create: %v", err)
	}
	result, err := svc.Recalculate(ctx, itr.ID)
	if err != nil {
		t.Fatalf("Recalculate: %v", err)
	}

	for _, w := range result.Warnings {
		if w == domain.WarningProgressiveRateReview {
			t.Errorf("unexpected progressive rate warning at low base: Warnings = %v", result.Warnings)
		}
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
