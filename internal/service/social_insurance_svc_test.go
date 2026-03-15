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

func setupSocialInsuranceSvc(t *testing.T) (*SocialInsuranceService, *sql.DB) {
	t.Helper()
	db := testutil.NewTestDB(t)
	sioRepo := repository.NewSocialInsuranceOverviewRepository(db)
	invRepo := repository.NewInvoiceRepository(db)
	expRepo := repository.NewExpenseRepository(db)
	setRepo := repository.NewSettingsRepository(db)
	tysRepo := repository.NewTaxYearSettingsRepository(db)
	tpRepo := repository.NewTaxPrepaymentRepository(db)
	svc := NewSocialInsuranceService(sioRepo, invRepo, expRepo, setRepo, tysRepo, tpRepo, nil)
	return svc, db
}

func TestSocialInsuranceService_Create(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:       2025,
		FilingType: domain.FilingTypeRegular,
	}

	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if sio.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	if sio.Status != domain.FilingStatusDraft {
		t.Errorf("Status = %q, want %q", sio.Status, domain.FilingStatusDraft)
	}
}

func TestSocialInsuranceService_Create_DefaultFilingType(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{Year: 2025}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if sio.FilingType != domain.FilingTypeRegular {
		t.Errorf("FilingType = %q, want %q", sio.FilingType, domain.FilingTypeRegular)
	}
}

func TestSocialInsuranceService_Create_DuplicateRegular(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	sio2 := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	err := svc.Create(ctx, sio2)
	if err == nil {
		t.Error("expected error for duplicate regular filing")
	}
	if !errors.Is(err, domain.ErrFilingAlreadyExists) {
		t.Errorf("expected ErrFilingAlreadyExists, got: %v", err)
	}
}

func TestSocialInsuranceService_Create_InvalidYear(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{Year: 1999, FilingType: domain.FilingTypeRegular}
	err := svc.Create(ctx, sio)
	if err == nil {
		t.Error("expected error for invalid year")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestSocialInsuranceService_Create_InvalidFilingType(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: "bad"}
	err := svc.Create(ctx, sio)
	if err == nil {
		t.Error("expected error for invalid filing_type")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestSocialInsuranceService_GetByID(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := svc.GetByID(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.Year != 2025 {
		t.Errorf("Year = %d, want 2025", got.Year)
	}
}

func TestSocialInsuranceService_GetByID_ZeroID(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestSocialInsuranceService_Delete(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := svc.Delete(ctx, sio.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := svc.GetByID(ctx, sio.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestSocialInsuranceService_Delete_Filed(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if _, err := svc.MarkFiled(ctx, sio.ID); err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}

	err := svc.Delete(ctx, sio.ID)
	if err == nil {
		t.Error("expected error when deleting filed overview")
	}
	if !errors.Is(err, domain.ErrFilingAlreadyFiled) {
		t.Errorf("expected ErrFilingAlreadyFiled, got: %v", err)
	}
}

func TestSocialInsuranceService_Delete_ZeroID(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	err := svc.Delete(ctx, 0)
	if err == nil {
		t.Error("expected error for zero ID")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestSocialInsuranceService_List(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	for _, ft := range []string{domain.FilingTypeRegular, domain.FilingTypeCorrective} {
		sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: ft}
		if err := svc.Create(ctx, sio); err != nil {
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

func TestSocialInsuranceService_MarkFiled(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	filed, err := svc.MarkFiled(ctx, sio.ID)
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
	_, err = svc.MarkFiled(ctx, sio.ID)
	if err == nil {
		t.Error("expected error for double filing")
	}
}

func TestSocialInsuranceService_Recalculate_ZeroID(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	_, err := svc.Recalculate(ctx, 0)
	if err == nil {
		t.Error("Recalculate(0) should return error")
	}
}

func TestSocialInsuranceService_Recalculate_Filed(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if _, err := svc.MarkFiled(ctx, sio.ID); err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}

	_, err := svc.Recalculate(ctx, sio.ID)
	if err == nil {
		t.Error("Recalculate() should return error for filed overview")
	}
	if !errors.Is(err, domain.ErrFilingAlreadyFiled) {
		t.Errorf("expected ErrFilingAlreadyFiled, got: %v", err)
	}
}

func TestSocialInsuranceService_Recalculate_WithInvoice(t *testing.T) {
	svc, db := setupSocialInsuranceSvc(t)
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
		"FV20250010", domain.InvoiceTypeRegular, domain.InvoiceStatusSent,
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

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	result, err := svc.Recalculate(ctx, sio.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	expectedRevenue := domain.NewAmount(500000, 0)
	if result.TotalRevenue != expectedRevenue {
		t.Errorf("TotalRevenue = %d, want %d", result.TotalRevenue, expectedRevenue)
	}

	// Used expenses = 60% flat rate = 500000 * 0.6 = 300000 CZK.
	expectedExpenses := expectedRevenue.Multiply(0.6)
	if result.TotalExpenses != expectedExpenses {
		t.Errorf("TotalExpenses = %d, want %d", result.TotalExpenses, expectedExpenses)
	}

	// TaxBase = revenue - expenses.
	expectedTaxBase := domain.Amount(int64(expectedRevenue) - int64(expectedExpenses))
	if result.TaxBase != expectedTaxBase {
		t.Errorf("TaxBase = %d, want %d", result.TaxBase, expectedTaxBase)
	}

	// AssessmentBase = TaxBase / 2.
	expectedAssessment := domain.Amount(int64(expectedTaxBase) / 2)
	if result.AssessmentBase != expectedAssessment {
		t.Errorf("AssessmentBase = %d, want %d", result.AssessmentBase, expectedAssessment)
	}

	// Insurance rate should be set.
	constants, _ := calc.GetTaxConstants(2025)
	if result.InsuranceRate != constants.SocialRate {
		t.Errorf("InsuranceRate = %d, want %d", result.InsuranceRate, constants.SocialRate)
	}

	// TotalInsurance should be > 0.
	if result.TotalInsurance <= 0 {
		t.Errorf("TotalInsurance = %d, expected > 0", result.TotalInsurance)
	}

	// NewMonthlyPrepay should be > 0.
	if result.NewMonthlyPrepay <= 0 {
		t.Errorf("NewMonthlyPrepay = %d, expected > 0", result.NewMonthlyPrepay)
	}
}

func TestSocialInsuranceService_Recalculate_MinAssessmentBase(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	// No invoices -- revenue = 0, so assessment base should be the minimum.
	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	result, err := svc.Recalculate(ctx, sio.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	constants, _ := calc.GetTaxConstants(2025)
	expectedMinBase := domain.Amount(int64(constants.SocialMinMonthly) * 12)

	if result.MinAssessmentBase != expectedMinBase {
		t.Errorf("MinAssessmentBase = %d, want %d", result.MinAssessmentBase, expectedMinBase)
	}
	if result.FinalAssessmentBase != expectedMinBase {
		t.Errorf("FinalAssessmentBase = %d, want %d (should use minimum)", result.FinalAssessmentBase, expectedMinBase)
	}

	// TotalInsurance = minBase * rate / 1000.
	expectedInsurance := domain.Amount(int64(expectedMinBase) * int64(constants.SocialRate) / 1000)
	if result.TotalInsurance != expectedInsurance {
		t.Errorf("TotalInsurance = %d, want %d", result.TotalInsurance, expectedInsurance)
	}
}
