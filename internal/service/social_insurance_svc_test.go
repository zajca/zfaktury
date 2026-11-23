package service

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/annualtaxxml"
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
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		ActivityMonths: domain.FullYearActivityMonths,
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

	sio := &domain.SocialInsuranceOverview{Year: 2025, ActivityMonths: domain.FullYearActivityMonths}
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

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	sio2 := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
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

	sio := &domain.SocialInsuranceOverview{Year: 1999, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
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

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: "bad", ActivityMonths: domain.FullYearActivityMonths}
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

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
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

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
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

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
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
		sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: ft, ActivityMonths: domain.FullYearActivityMonths}
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

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
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

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
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

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
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

	// CSSZ 2025 assessment base = 55% of tax base.
	expectedAssessment := domain.Amount(int64(expectedTaxBase) * 550 / 1000)
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

// CSSZ recalculates January-June 2026 as if the lower minimum applied from
// January, so the annual overview minimum is 12 x 17 139 CZK.
func TestSocialInsuranceService_Recalculate_MinAssessmentBase2026(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{Year: 2026, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	result, err := svc.Recalculate(ctx, sio.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	want := domain.NewAmount(205_668, 0)
	if result.MinAssessmentBase != want {
		t.Errorf("MinAssessmentBase = %d, want %d", result.MinAssessmentBase, want)
	}
	if result.FinalAssessmentBase != want {
		t.Errorf("FinalAssessmentBase = %d, want %d", result.FinalAssessmentBase, want)
	}
}

func TestSocialInsuranceService_Recalculate_MinAssessmentBase(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	// No invoices -- revenue = 0, so assessment base should be the minimum.
	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
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

	// TotalInsurance = minBase * rate / 1000, rounded up to whole CZK for CSSZ.
	expectedInsurance := domain.Amount(((int64(expectedMinBase)*int64(constants.SocialRate)/1000 + 99) / 100) * 100)
	if result.TotalInsurance != expectedInsurance {
		t.Errorf("TotalInsurance = %d, want %d", result.TotalInsurance, expectedInsurance)
	}
}

// CSSZ caps the annual assessment base at 48x the average wage (2 234 736 Kč
// for 2025), so a high-revenue OSVČ pays premium only up to the cap.
func TestSocialInsuranceService_Recalculate_MaxAssessmentBase(t *testing.T) {
	svc, db := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	contact := testutil.SeedContact(t, db, nil)
	jan15 := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	now := time.Now()

	// Seed invoice: 5,000,000 CZK revenue, no flat rate -> assessment base 55 %
	// of 5,000,000 CZK is above the 2025 maximum.
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
		"FV20250011", domain.InvoiceTypeRegular, domain.InvoiceStatusSent,
		jan15.Format("2006-01-02"), jan15.AddDate(0, 0, 14).Format("2006-01-02"), jan15.Format("2006-01-02"),
		"", "", contact.ID, "CZK", 100,
		"bank_transfer", "", "", "", "",
		domain.NewAmount(5_000_000, 0), 0, domain.NewAmount(5_000_000, 0), 0,
		"", "",
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("inserting invoice: %v", err)
	}

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	result, err := svc.Recalculate(ctx, sio.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	constants, err := calc.GetTaxConstants(2025)
	if err != nil {
		t.Fatalf("GetTaxConstants(2025) error: %v", err)
	}
	wantAssessmentBase := domain.NewAmount(2_750_000, 0)
	if result.AssessmentBase != wantAssessmentBase {
		t.Errorf("AssessmentBase = %d, want %d", result.AssessmentBase, wantAssessmentBase)
	}
	if result.FinalAssessmentBase != constants.SocialMaxAnnualBase {
		t.Errorf("FinalAssessmentBase = %d, want %d (social maximum)", result.FinalAssessmentBase, constants.SocialMaxAnnualBase)
	}

	wantInsurance := domain.NewAmount(652_543, 0)
	if result.TotalInsurance != wantInsurance {
		t.Errorf("TotalInsurance = %d, want %d", result.TotalInsurance, wantInsurance)
	}
}

func TestSocialInsuranceService_Create_MissingActivityMonths(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular}
	err := svc.Create(ctx, sio)
	if err == nil {
		t.Fatal("Create() error = nil, want ErrInvalidInput")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("Create() error = %v, want ErrInvalidInput", err)
	}
}

// July-December activity prorates both the minimum base and the monthly advance.
func TestSocialInsuranceService_Recalculate_PartialYear(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		ActivityMonths: domain.ActivityMonths(4032),
	}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	result, err := svc.Recalculate(ctx, sio.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	if got, want := result.MinAssessmentBase, domain.NewAmount(97_770, 0); got != want {
		t.Errorf("MinAssessmentBase = %d, want %d", got, want)
	}
	if got, want := result.FinalAssessmentBase, domain.NewAmount(97_770, 0); got != want {
		t.Errorf("FinalAssessmentBase = %d, want %d", got, want)
	}
	if got, want := result.TotalInsurance, domain.NewAmount(28_549, 0); got != want {
		t.Errorf("TotalInsurance = %d, want %d", got, want)
	}
	if got, want := result.Difference, domain.NewAmount(28_549, 0); got != want {
		t.Errorf("Difference = %d, want %d", got, want)
	}
	if got, want := result.NewMonthlyPrepay, domain.NewAmount(4_759, 0); got != want {
		t.Errorf("NewMonthlyPrepay = %d, want %d", got, want)
	}
}

func TestSocialInsuranceService_SetActivityMonths(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		ActivityMonths: domain.FullYearActivityMonths,
	}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	updated, err := svc.SetActivityMonths(ctx, sio.ID, domain.ActivityMonths(4032))
	if err != nil {
		t.Fatalf("SetActivityMonths() error: %v", err)
	}
	if updated.ActivityMonths != domain.ActivityMonths(4032) {
		t.Errorf("ActivityMonths = %d, want 4032", updated.ActivityMonths)
	}
	if got, want := updated.MinAssessmentBase, domain.NewAmount(97_770, 0); got != want {
		t.Errorf("MinAssessmentBase = %d, want %d", got, want)
	}
	if got, want := updated.TotalInsurance, domain.NewAmount(28_549, 0); got != want {
		t.Errorf("TotalInsurance = %d, want %d", got, want)
	}
	if got, want := updated.NewMonthlyPrepay, domain.NewAmount(4_759, 0); got != want {
		t.Errorf("NewMonthlyPrepay = %d, want %d", got, want)
	}

	stored, err := svc.GetByID(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if stored.ActivityMonths != domain.ActivityMonths(4032) {
		t.Errorf("stored ActivityMonths = %d, want 4032", stored.ActivityMonths)
	}
}

func TestSocialInsuranceService_SetActivityMonths_RecalculateFails(t *testing.T) {
	svc, db := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		ActivityMonths: domain.FullYearActivityMonths,
	}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// A year without tax constants makes the recalculation fail.
	if _, err := db.ExecContext(ctx, `UPDATE social_insurance_overviews SET year = 2019 WHERE id = ?`, sio.ID); err != nil {
		t.Fatalf("setting unsupported year: %v", err)
	}

	if _, err := svc.SetActivityMonths(ctx, sio.ID, domain.ActivityMonths(4032)); err == nil {
		t.Fatal("SetActivityMonths() error = nil, want a recalculation error")
	}

	stored, err := svc.GetByID(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if stored.ActivityMonths != domain.FullYearActivityMonths {
		t.Errorf("stored ActivityMonths = %d, want %d after a failed recalculation", stored.ActivityMonths, domain.FullYearActivityMonths)
	}
}

func TestSocialInsuranceService_SetActivityMonths_ClearsStoredXML(t *testing.T) {
	svc, db := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		ActivityMonths: domain.FullYearActivityMonths,
	}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if _, err := db.ExecContext(ctx, `UPDATE social_insurance_overviews SET xml_data = '<xml/>' WHERE id = ?`, sio.ID); err != nil {
		t.Fatalf("storing XML: %v", err)
	}

	if _, err := svc.SetActivityMonths(ctx, sio.ID, domain.ActivityMonths(4032)); err != nil {
		t.Fatalf("SetActivityMonths() error: %v", err)
	}

	stored, err := svc.GetXMLData(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetXMLData() error: %v", err)
	}
	if len(stored) != 0 {
		t.Errorf("stored XML = %q, want it cleared after changing the months", stored)
	}
}

func TestSocialInsuranceService_SetActivityMonths_Filed(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		ActivityMonths: domain.FullYearActivityMonths,
	}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if _, err := svc.MarkFiled(ctx, sio.ID); err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}

	_, err := svc.SetActivityMonths(ctx, sio.ID, domain.ActivityMonths(4032))
	if !errors.Is(err, domain.ErrFilingAlreadyFiled) {
		t.Errorf("SetActivityMonths() error = %v, want ErrFilingAlreadyFiled", err)
	}
}

func TestSocialInsuranceService_SetActivityMonths_InvalidInput(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		ActivityMonths: domain.FullYearActivityMonths,
	}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if _, err := svc.SetActivityMonths(ctx, sio.ID, domain.ActivityMonths(0)); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("SetActivityMonths(mask 0) error = %v, want ErrInvalidInput", err)
	}
	if _, err := svc.SetActivityMonths(ctx, 0, domain.FullYearActivityMonths); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("SetActivityMonths(id 0) error = %v, want ErrInvalidInput", err)
	}
}

// The advance is bounded by the minimum and maximum of the year it is paid in,
// so a 2024 overview uses the 2025 bounds (minimum 16 295 CZK monthly).
func TestRecalculate_2024UsesPaymentYearBounds(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{Year: 2024, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	result, err := svc.Recalculate(ctx, sio.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	// The 2024 overview year gives a 13 191 CZK monthly base, which would be
	// 3 852 CZK unbounded; the 2025 payment-year minimum (16 295 CZK) lifts the
	// advance to 4 759 CZK.
	if want := domain.NewAmount(4_759, 0); result.NewMonthlyPrepay != want {
		t.Errorf("NewMonthlyPrepay = %d, want %d", result.NewMonthlyPrepay, want)
	}
}

// The 2026 minimum is split into two periods, so the start month of the new
// advance is still open and the 2025 overview keeps the unbounded advance.
func TestRecalculate_2025KeepsUnboundedAdvance_TwoMinimumPeriods(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	result, err := svc.Recalculate(ctx, sio.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	if want := domain.NewAmount(4_759, 0); result.NewMonthlyPrepay != want {
		t.Errorf("NewMonthlyPrepay = %d, want %d (2025 base 16 295 CZK, unbounded)", result.NewMonthlyPrepay, want)
	}
}

// There is no rule set for 2027 yet, so the 2026 overview keeps the unbounded
// advance instead of failing.
func TestRecalculate_2026KeepsUnboundedAdvance_NoNextYearRules(t *testing.T) {
	svc, _ := setupSocialInsuranceSvc(t)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{Year: 2026, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	result, err := svc.Recalculate(ctx, sio.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	if want := domain.NewAmount(5_005, 0); result.NewMonthlyPrepay != want {
		t.Errorf("NewMonthlyPrepay = %d, want %d (2026 base 17 139 CZK, unbounded)", result.NewMonthlyPrepay, want)
	}
}

// seedSocialInsuranceXMLSettings stores the taxpayer data the CSSZ XML needs.
func seedSocialInsuranceXMLSettings(t *testing.T, db *sql.DB) {
	t.Helper()
	settingsRepo := repository.NewSettingsRepository(db)
	settings := map[string]string{
		"cssz_code":    "775",
		"first_name":   "Jan",
		"last_name":    "Testovaci",
		"birth_number": "8001010006",
		"birth_date":   "1980-01-01",
		"street":       "Testovaci",
		"house_number": "1",
		"zip":          "11000",
		"city":         "Praha",
	}
	if err := settingsRepo.SetBulk(context.Background(), settings); err != nil {
		t.Fatalf("storing settings: %v", err)
	}
}

// The XML must report the base and the advance from one stored recalculation,
// so the 2024 overview keeps the 2025 payment-year bounds.
func TestGenerateXML_2024UsesStoredBoundedBase(t *testing.T) {
	svc, db := setupSocialInsuranceSvc(t)
	ctx := context.Background()
	seedSocialInsuranceXMLSettings(t, db)

	sio := &domain.SocialInsuranceOverview{Year: 2024, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if _, err := svc.Recalculate(ctx, sio.ID); err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	generated, err := svc.GenerateXML(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GenerateXML() error: %v", err)
	}

	var doc annualtaxxml.OSVC
	if err := xml.Unmarshal(generated.XMLData, &doc); err != nil {
		t.Fatalf("unmarshal generated XML: %v", err)
	}
	if doc.Prehled.Zal.VZ != "16295" {
		t.Errorf("zal@vz = %q, want 16295", doc.Prehled.Zal.VZ)
	}
	if doc.Prehled.Zal.DP != "4759" {
		t.Errorf("zal@dp = %q, want 4759", doc.Prehled.Zal.DP)
	}
}

// Overviews stored before the base was kept have none; the base is then taken
// as the final annual base over the months of activity, which matches their
// stored advance only when both formulas agree.
func TestGenerateXML_LegacyRowWithoutStoredBase(t *testing.T) {
	svc, db := setupSocialInsuranceSvc(t)
	ctx := context.Background()
	seedSocialInsuranceXMLSettings(t, db)

	sio := &domain.SocialInsuranceOverview{
		Year:                2024,
		FilingType:          domain.FilingTypeRegular,
		ActivityMonths:      domain.FullYearActivityMonths,
		AssessmentBase:      domain.NewAmount(55_000, 0),
		MinAssessmentBase:   domain.NewAmount(158_292, 0),
		FinalAssessmentBase: domain.NewAmount(158_292, 0),
		InsuranceRate:       292,
		TotalInsurance:      domain.NewAmount(46_221, 0),
		NewMonthlyPrepay:    domain.NewAmount(3_852, 0),
	}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	generated, err := svc.GenerateXML(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GenerateXML() error: %v", err)
	}

	var doc annualtaxxml.OSVC
	if err := xml.Unmarshal(generated.XMLData, &doc); err != nil {
		t.Fatalf("unmarshal generated XML: %v", err)
	}
	if doc.Prehled.Zal.VZ != "13191" {
		t.Errorf("zal@vz = %q, want 13191 (the legacy base)", doc.Prehled.Zal.VZ)
	}
	if doc.Prehled.Zal.DP != "3852" {
		t.Errorf("zal@dp = %q, want 3852", doc.Prehled.Zal.DP)
	}
}

func TestGenerateXML_RejectsInconsistentStoredAdvance(t *testing.T) {
	svc, db := setupSocialInsuranceSvc(t)
	ctx := context.Background()
	seedSocialInsuranceXMLSettings(t, db)

	sio := &domain.SocialInsuranceOverview{Year: 2024, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
	if err := svc.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if _, err := svc.Recalculate(ctx, sio.ID); err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE social_insurance_overviews SET new_monthly_prepay = ? WHERE id = ?`, domain.NewAmount(9_999, 0), sio.ID); err != nil {
		t.Fatalf("updating stored advance: %v", err)
	}

	_, err := svc.GenerateXML(ctx, sio.ID)
	if err == nil {
		t.Fatal("GenerateXML() error = nil, want ErrInvalidInput")
	}
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("GenerateXML() error = %v, want ErrInvalidInput", err)
	}
}

// Overviews stored by the release before the base was kept computed the advance
// as the annual premium divided by 12, rounded up to whole CZK, so for some
// final bases their stored advance does not follow from the monthly base.
// final base 158 293 CZK: base 13 192, expected advance 3 853, stored 3 852.
func TestGenerateXML_LegacyProductionRowWithoutStoredBase(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		filed      bool
		wantPhrase string
	}{
		{name: "not filed", status: domain.FilingStatusDraft, wantPhrase: "recalculate the overview"},
		{name: "already filed", status: domain.FilingStatusFiled, filed: true, wantPhrase: "already filed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, db := setupSocialInsuranceSvc(t)
			ctx := context.Background()
			seedSocialInsuranceXMLSettings(t, db)

			sio := &domain.SocialInsuranceOverview{
				Year:                2024,
				FilingType:          domain.FilingTypeRegular,
				ActivityMonths:      domain.FullYearActivityMonths,
				AssessmentBase:      domain.NewAmount(87_062, 0),
				MinAssessmentBase:   domain.NewAmount(158_292, 0),
				FinalAssessmentBase: domain.NewAmount(158_293, 0),
				InsuranceRate:       292,
				TotalInsurance:      domain.NewAmount(46_221, 0),
				NewMonthlyPrepay:    domain.NewAmount(3_852, 0),
				Status:              tt.status,
			}
			if tt.filed {
				now := time.Now()
				sio.FiledAt = &now
			}
			if err := svc.Create(ctx, sio); err != nil {
				t.Fatalf("Create() error: %v", err)
			}

			_, err := svc.GenerateXML(ctx, sio.ID)
			if err == nil {
				t.Fatal("GenerateXML() error = nil, want ErrInvalidInput")
			}
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("GenerateXML() error = %v, want ErrInvalidInput", err)
			}
			if !strings.Contains(err.Error(), tt.wantPhrase) {
				t.Errorf("GenerateXML() error = %v, want it to contain %q", err, tt.wantPhrase)
			}
		})
	}
}
