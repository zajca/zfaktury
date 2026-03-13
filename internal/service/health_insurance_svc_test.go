package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

func setupHealthInsuranceSvc(t *testing.T) (*HealthInsuranceService, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	hioRepo := repository.NewHealthInsuranceOverviewRepository(db)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	setRepo := repository.NewSettingsRepository(db)
	tysRepo := repository.NewTaxYearSettingsRepository(db)
	tpRepo := repository.NewTaxPrepaymentRepository(db)
	svc := NewHealthInsuranceService(hioRepo, invRepo, expRepo, setRepo, tysRepo, tpRepo, nil)
	return svc, db
}

func TestHealthInsuranceService_Create(t *testing.T) {
	svc, _ := setupHealthInsuranceSvc(t)
	ctx := context.Background()

	hi := &domain.HealthInsuranceOverview{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}

	if err := svc.Create(ctx, hi); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if hi.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	if hi.Status != domain.FilingStatusDraft {
		t.Errorf("Status = %q, want %q", hi.Status, domain.FilingStatusDraft)
	}
}

func TestHealthInsuranceService_Create_DefaultFilingType(t *testing.T) {
	svc, _ := setupHealthInsuranceSvc(t)
	ctx := context.Background()

	hi := &domain.HealthInsuranceOverview{Year: 2025}
	if err := svc.Create(ctx, hi); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if hi.FilingType != domain.FilingTypeRegular {
		t.Errorf("FilingType = %q, want %q", hi.FilingType, domain.FilingTypeRegular)
	}
}

func TestHealthInsuranceService_Create_DuplicateRegular(t *testing.T) {
	svc, _ := setupHealthInsuranceSvc(t)
	ctx := context.Background()

	hi := &domain.HealthInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, hi); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	hi2 := &domain.HealthInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	err := svc.Create(ctx, hi2)
	if err == nil {
		t.Error("expected error for duplicate regular filing")
	}
	if !errors.Is(err, domain.ErrFilingAlreadyExists) {
		t.Errorf("expected ErrFilingAlreadyExists, got: %v", err)
	}
}

func TestHealthInsuranceService_Create_InvalidYear(t *testing.T) {
	svc, _ := setupHealthInsuranceSvc(t)
	ctx := context.Background()

	hi := &domain.HealthInsuranceOverview{Year: 1999, FilingType: domain.FilingTypeRegular}
	err := svc.Create(ctx, hi)
	if err == nil {
		t.Error("expected error for invalid year")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestHealthInsuranceService_Create_InvalidFilingType(t *testing.T) {
	svc, _ := setupHealthInsuranceSvc(t)
	ctx := context.Background()

	hi := &domain.HealthInsuranceOverview{Year: 2025, FilingType: "bad"}
	err := svc.Create(ctx, hi)
	if err == nil {
		t.Error("expected error for invalid filing_type")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestHealthInsuranceService_GetByID(t *testing.T) {
	svc, _ := setupHealthInsuranceSvc(t)
	ctx := context.Background()

	hi := &domain.HealthInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, hi); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := svc.GetByID(ctx, hi.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Year != 2025 {
		t.Errorf("Year = %d, want 2025", got.Year)
	}
}

func TestHealthInsuranceService_GetByID_ZeroID(t *testing.T) {
	svc, _ := setupHealthInsuranceSvc(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestHealthInsuranceService_Delete(t *testing.T) {
	svc, _ := setupHealthInsuranceSvc(t)
	ctx := context.Background()

	hi := &domain.HealthInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, hi); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.Delete(ctx, hi.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := svc.GetByID(ctx, hi.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestHealthInsuranceService_Delete_Filed(t *testing.T) {
	svc, _ := setupHealthInsuranceSvc(t)
	ctx := context.Background()

	hi := &domain.HealthInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, hi); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if _, err := svc.MarkFiled(ctx, hi.ID); err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}

	err := svc.Delete(ctx, hi.ID)
	if err == nil {
		t.Error("expected error when deleting filed overview")
	}
	if !errors.Is(err, domain.ErrFilingAlreadyFiled) {
		t.Errorf("expected ErrFilingAlreadyFiled, got: %v", err)
	}
}

func TestHealthInsuranceService_Delete_ZeroID(t *testing.T) {
	svc, _ := setupHealthInsuranceSvc(t)
	ctx := context.Background()

	err := svc.Delete(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestHealthInsuranceService_List(t *testing.T) {
	svc, _ := setupHealthInsuranceSvc(t)
	ctx := context.Background()

	for _, ft := range []string{domain.FilingTypeRegular, domain.FilingTypeCorrective} {
		hi := &domain.HealthInsuranceOverview{Year: 2025, FilingType: ft}
		if err := svc.Create(ctx, hi); err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	result, err := svc.List(ctx, 2025)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("List() returned %d items, want 2", len(result))
	}
}

func TestHealthInsuranceService_MarkFiled(t *testing.T) {
	svc, _ := setupHealthInsuranceSvc(t)
	ctx := context.Background()

	hi := &domain.HealthInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, hi); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	filed, err := svc.MarkFiled(ctx, hi.ID)
	if err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}
	if filed.Status != domain.FilingStatusFiled {
		t.Errorf("Status = %q, want %q", filed.Status, domain.FilingStatusFiled)
	}
	if filed.FiledAt == nil {
		t.Error("expected FiledAt to be set")
	}

	// Double filing should fail.
	_, err = svc.MarkFiled(ctx, hi.ID)
	if err == nil {
		t.Error("expected error for double filing")
	}
}

func TestHealthInsuranceService_Recalculate_ZeroID(t *testing.T) {
	svc, _ := setupHealthInsuranceSvc(t)
	ctx := context.Background()

	_, err := svc.Recalculate(ctx, 0)
	if err == nil {
		t.Error("Recalculate(0) should return error")
	}
}

func TestHealthInsuranceService_Recalculate_Filed(t *testing.T) {
	svc, _ := setupHealthInsuranceSvc(t)
	ctx := context.Background()

	hi := &domain.HealthInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, hi); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if _, err := svc.MarkFiled(ctx, hi.ID); err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}

	_, err := svc.Recalculate(ctx, hi.ID)
	if err == nil {
		t.Error("Recalculate() should return error for filed overview")
	}
	if !errors.Is(err, domain.ErrFilingAlreadyFiled) {
		t.Errorf("expected ErrFilingAlreadyFiled, got: %v", err)
	}
}

func TestHealthInsuranceService_Recalculate_WithInvoice(t *testing.T) {
	svc, db := setupHealthInsuranceSvc(t)
	ctx := context.Background()

	contact := testutil.SeedContact(t, db, nil)
	jan15 := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	now := time.Now()

	// Seed invoice: 500,000 CZK revenue.
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
		"FV20250020", domain.InvoiceTypeRegular, domain.InvoiceStatusSent,
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

	// Set up flat rate 60%.
	tysRepo := repository.NewTaxYearSettingsRepository(db)
	if err := tysRepo.Upsert(ctx, &domain.TaxYearSettings{Year: 2025, FlatRatePercent: 60}); err != nil {
		t.Fatalf("Upsert tax year settings: %v", err)
	}

	hi := &domain.HealthInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, hi); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	result, err := svc.Recalculate(ctx, hi.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	expectedRevenue := domain.NewAmount(500000, 0)
	if result.TotalRevenue != expectedRevenue {
		t.Errorf("TotalRevenue = %d, want %d", result.TotalRevenue, expectedRevenue)
	}

	expectedExpenses := expectedRevenue.Multiply(0.6)
	if result.TotalExpenses != expectedExpenses {
		t.Errorf("TotalExpenses = %d, want %d", result.TotalExpenses, expectedExpenses)
	}

	expectedTaxBase := domain.Amount(int64(expectedRevenue) - int64(expectedExpenses))
	if result.TaxBase != expectedTaxBase {
		t.Errorf("TaxBase = %d, want %d", result.TaxBase, expectedTaxBase)
	}

	expectedAssessment := domain.Amount(int64(expectedTaxBase) / 2)
	if result.AssessmentBase != expectedAssessment {
		t.Errorf("AssessmentBase = %d, want %d", result.AssessmentBase, expectedAssessment)
	}

	constants, _ := GetTaxConstants(2025)
	if result.InsuranceRate != constants.HealthRate {
		t.Errorf("InsuranceRate = %d, want %d", result.InsuranceRate, constants.HealthRate)
	}

	if result.TotalInsurance <= 0 {
		t.Errorf("TotalInsurance = %d, expected > 0", result.TotalInsurance)
	}

	if result.NewMonthlyPrepay <= 0 {
		t.Errorf("NewMonthlyPrepay = %d, expected > 0", result.NewMonthlyPrepay)
	}
}

func TestHealthInsuranceService_Recalculate_MinAssessmentBase(t *testing.T) {
	svc, _ := setupHealthInsuranceSvc(t)
	ctx := context.Background()

	// No invoices -- revenue = 0, so should use minimum assessment base.
	hi := &domain.HealthInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, hi); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	result, err := svc.Recalculate(ctx, hi.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	constants, _ := GetTaxConstants(2025)
	expectedMinBase := domain.Amount(int64(constants.HealthMinMonthly) * 12)

	if result.MinAssessmentBase != expectedMinBase {
		t.Errorf("MinAssessmentBase = %d, want %d", result.MinAssessmentBase, expectedMinBase)
	}
	if result.FinalAssessmentBase != expectedMinBase {
		t.Errorf("FinalAssessmentBase = %d, want %d (should use minimum)", result.FinalAssessmentBase, expectedMinBase)
	}

	expectedInsurance := domain.Amount(int64(expectedMinBase) * int64(constants.HealthRate) / 1000)
	if result.TotalInsurance != expectedInsurance {
		t.Errorf("TotalInsurance = %d, want %d", result.TotalInsurance, expectedInsurance)
	}
}

func TestHealthInsuranceService_Recalculate_WithPrepayments(t *testing.T) {
	svc, db := setupHealthInsuranceSvc(t)
	ctx := context.Background()

	// Add health prepayments for 2025.
	tpRepo := repository.NewTaxPrepaymentRepository(db)
	prepayments := []domain.TaxPrepayment{
		{Month: 1, TaxAmount: 0, SocialAmount: 0, HealthAmount: domain.NewAmount(3000, 0)},
		{Month: 2, TaxAmount: 0, SocialAmount: 0, HealthAmount: domain.NewAmount(3000, 0)},
		{Month: 3, TaxAmount: 0, SocialAmount: 0, HealthAmount: domain.NewAmount(3000, 0)},
	}
	if err := tpRepo.UpsertAll(ctx, 2025, prepayments); err != nil {
		t.Fatalf("UpsertAll prepayments: %v", err)
	}

	hi := &domain.HealthInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, hi); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	result, err := svc.Recalculate(ctx, hi.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	expectedPrepayments := domain.NewAmount(9000, 0)
	if result.Prepayments != expectedPrepayments {
		t.Errorf("Prepayments = %d, want %d", result.Prepayments, expectedPrepayments)
	}
}
