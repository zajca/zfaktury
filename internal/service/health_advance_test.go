package service

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/testutil"
)

// seedHealthInvoice inserts one issued 2025 invoice of the given total in CZK.
func seedHealthInvoice(t *testing.T, db *sql.DB, totalCZK int64) {
	t.Helper()
	contact := testutil.SeedContact(t, db, nil)
	jan15 := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	now := time.Now()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO invoices (
			invoice_number, type, status,
			issue_date, due_date, delivery_date, variable_symbol, constant_symbol,
			customer_id, currency_code, exchange_rate,
			payment_method, bank_account, bank_code, iban, swift,
			subtotal_amount, vat_amount, total_amount, paid_amount,
			notes, internal_notes,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"FV20250099", domain.InvoiceTypeRegular, domain.InvoiceStatusSent,
		jan15.Format("2006-01-02"), jan15.AddDate(0, 0, 14).Format("2006-01-02"), jan15.Format("2006-01-02"),
		"", "", contact.ID, "CZK", 100,
		"bank_transfer", "", "", "", "",
		domain.NewAmount(totalCZK, 0), 0, domain.NewAmount(totalCZK, 0), 0,
		"", "",
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("inserting invoice: %v", err)
	}
}

func saveHealthFilingMethod(t *testing.T, db *sql.DB, year int, method domain.FilingMethod) {
	t.Helper()
	repo := repository.NewTaxYearSettingsRepository(db)
	if err := repo.Upsert(context.Background(), &domain.TaxYearSettings{Year: year, FilingMethod: method}); err != nil {
		t.Fatalf("storing filing method: %v", err)
	}
}

func createHealthOverview(t *testing.T, svc *HealthInsuranceService, year int, months domain.ActivityMonths) *domain.HealthInsuranceOverview {
	t.Helper()
	hi := &domain.HealthInsuranceOverview{Year: year, FilingType: domain.FilingTypeRegular, ActivityMonths: months}
	if err := svc.Create(context.Background(), hi); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	return hi
}

func TestHealthRecalculate_StartMonthFromFilingMethod(t *testing.T) {
	tests := []struct {
		method       domain.FilingMethod
		wantStart    int
		wantWarnings []string
	}{
		{domain.FilingMethodStandard, 5, nil},
		{domain.FilingMethodElectronic, 6, nil},
		{domain.FilingMethodAdvisor, 8, nil},
		{domain.FilingMethodUnset, 5, []string{domain.WarningFilingMethodMissing}},
	}
	for _, tt := range tests {
		t.Run(string(tt.method), func(t *testing.T) {
			svc, db := setupHealthInsuranceSvc(t)
			saveHealthFilingMethod(t, db, 2025, tt.method)
			hi := createHealthOverview(t, svc, 2025, domain.FullYearActivityMonths)

			result, err := svc.Recalculate(context.Background(), hi.ID)
			if err != nil {
				t.Fatalf("Recalculate() error: %v", err)
			}

			if result.AdvanceStartMonth != tt.wantStart {
				t.Errorf("AdvanceStartMonth = %d, want %d", result.AdvanceStartMonth, tt.wantStart)
			}
			if want := domain.NewAmount(3_306, 0); result.NewMonthlyPrepay != want {
				t.Errorf("NewMonthlyPrepay = %d, want %d", result.NewMonthlyPrepay, want)
			}
			if !slices.Equal(result.Warnings, tt.wantWarnings) {
				t.Errorf("Warnings = %v, want %v", result.Warnings, tt.wantWarnings)
			}
		})
	}
}

func TestHealthRecalculate_2026NoNextYearRules(t *testing.T) {
	svc, db := setupHealthInsuranceSvc(t)
	saveHealthFilingMethod(t, db, 2026, domain.FilingMethodStandard)
	hi := createHealthOverview(t, svc, 2026, domain.FullYearActivityMonths)

	result, err := svc.Recalculate(context.Background(), hi.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}

	if want := domain.NewAmount(3_306, 0); result.NewMonthlyPrepay != want {
		t.Errorf("NewMonthlyPrepay = %d, want %d", result.NewMonthlyPrepay, want)
	}
	if !slices.Contains(result.Warnings, domain.WarningAdvanceMinimumMissing) {
		t.Errorf("Warnings = %v, want advance_minimum_missing", result.Warnings)
	}
	if result.AdvanceStartMonth != 5 {
		t.Errorf("AdvanceStartMonth = %d, want 5 (deadline 3 May 2027)", result.AdvanceStartMonth)
	}
}

func TestHealthSetNoMinimum_PartialMonths(t *testing.T) {
	svc, db := setupHealthInsuranceSvc(t)
	saveHealthFilingMethod(t, db, 2025, domain.FilingMethodStandard)
	seedHealthInvoice(t, db, 100_000)
	hi := createHealthOverview(t, svc, 2025, domain.FullYearActivityMonths)

	jan2jun, err := domain.NewActivityMonths([]int{1, 2, 3, 4, 5, 6})
	if err != nil {
		t.Fatalf("NewActivityMonths: %v", err)
	}
	result, err := svc.SetNoMinimum(context.Background(), hi.ID, jan2jun, domain.HealthNoMinimumEmployee)
	if err != nil {
		t.Fatalf("SetNoMinimum() error: %v", err)
	}

	if want := domain.NewAmount(139_671, 0); result.MinAssessmentBase != want || result.FinalAssessmentBase != want {
		t.Errorf("Min/Final = %d/%d, want %d", result.MinAssessmentBase, result.FinalAssessmentBase, want)
	}
	if want := domain.NewAmount(18_856, 0); result.TotalInsurance != want {
		t.Errorf("TotalInsurance = %d, want %d", result.TotalInsurance, want)
	}
	if want := domain.NewAmount(3_306, 0); result.NewMonthlyPrepay != want {
		t.Errorf("NewMonthlyPrepay = %d, want %d", result.NewMonthlyPrepay, want)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("Warnings = %v, want none", result.Warnings)
	}
}

func TestHealthSetNoMinimum_AllMonthsNoAdvance(t *testing.T) {
	svc, db := setupHealthInsuranceSvc(t)
	saveHealthFilingMethod(t, db, 2025, domain.FilingMethodStandard)
	seedHealthInvoice(t, db, 100_000)
	hi := createHealthOverview(t, svc, 2025, domain.FullYearActivityMonths)

	result, err := svc.SetNoMinimum(context.Background(), hi.ID, domain.FullYearActivityMonths, domain.HealthNoMinimumStateInsured)
	if err != nil {
		t.Fatalf("SetNoMinimum() error: %v", err)
	}

	if want := domain.NewAmount(50_000, 0); result.FinalAssessmentBase != want {
		t.Errorf("FinalAssessmentBase = %d, want %d", result.FinalAssessmentBase, want)
	}
	if want := domain.NewAmount(6_750, 0); result.TotalInsurance != want {
		t.Errorf("TotalInsurance = %d, want %d", result.TotalInsurance, want)
	}
	if result.NewMonthlyPrepay != 0 {
		t.Errorf("NewMonthlyPrepay = %d, want 0", result.NewMonthlyPrepay)
	}
	if want := []string{domain.WarningHealthAdvanceNotRequired}; !slices.Equal(result.Warnings, want) {
		t.Errorf("Warnings = %v, want %v", result.Warnings, want)
	}
}

func TestHealthSetNoMinimum_Validation(t *testing.T) {
	jan, _ := domain.NewActivityMonths([]int{1})
	secondHalf, _ := domain.NewActivityMonths([]int{7, 8, 9, 10, 11, 12})
	tests := []struct {
		name     string
		activity domain.ActivityMonths
		months   domain.ActivityMonths
		reason   domain.HealthNoMinimumReason
	}{
		{"month outside activity", secondHalf, jan, domain.HealthNoMinimumEmployee},
		{"months without a reason", domain.FullYearActivityMonths, jan, domain.HealthNoMinimumNone},
		{"reason without months", domain.FullYearActivityMonths, 0, domain.HealthNoMinimumEmployee},
		{"unknown reason", domain.FullYearActivityMonths, jan, domain.HealthNoMinimumReason("x")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := setupHealthInsuranceSvc(t)
			hi := createHealthOverview(t, svc, 2025, tt.activity)

			_, err := svc.SetNoMinimum(context.Background(), hi.ID, tt.months, tt.reason)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("SetNoMinimum() error = %v, want ErrInvalidInput", err)
			}
			stored, err := svc.GetByID(context.Background(), hi.ID)
			if err != nil {
				t.Fatalf("GetByID() error: %v", err)
			}
			if stored.NoMinimumMonths != 0 || stored.NoMinimumReason != "" {
				t.Errorf("stored no-minimum = %d/%q, want nothing written", stored.NoMinimumMonths, stored.NoMinimumReason)
			}
		})
	}
}

func TestHealthSetActivityMonths_RejectsDroppingNoMinimumMonth(t *testing.T) {
	svc, _ := setupHealthInsuranceSvc(t)
	ctx := context.Background()
	hi := createHealthOverview(t, svc, 2025, domain.FullYearActivityMonths)
	jan, _ := domain.NewActivityMonths([]int{1})
	if _, err := svc.SetNoMinimum(ctx, hi.ID, jan, domain.HealthNoMinimumZTP); err != nil {
		t.Fatalf("SetNoMinimum() error: %v", err)
	}
	secondHalf, _ := domain.NewActivityMonths([]int{7, 8, 9, 10, 11, 12})

	_, err := svc.SetActivityMonths(ctx, hi.ID, secondHalf)
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("SetActivityMonths() error = %v, want ErrInvalidInput", err)
	}
}

func TestHealthSetNoMinimum_Filed(t *testing.T) {
	svc, _ := setupHealthInsuranceSvc(t)
	ctx := context.Background()
	hi := createHealthOverview(t, svc, 2025, domain.FullYearActivityMonths)
	if _, err := svc.MarkFiled(ctx, hi.ID); err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}
	jan, _ := domain.NewActivityMonths([]int{1})

	_, err := svc.SetNoMinimum(ctx, hi.ID, jan, domain.HealthNoMinimumZTP)
	if !errors.Is(err, domain.ErrFilingAlreadyFiled) {
		t.Fatalf("SetNoMinimum() error = %v, want ErrFilingAlreadyFiled", err)
	}
}

func TestHealthRecalculate_TaxYearSettingsErrorIsReturned(t *testing.T) {
	svc, db := setupHealthInsuranceSvc(t)
	ctx := context.Background()
	hi := createHealthOverview(t, svc, 2025, domain.FullYearActivityMonths)
	if _, err := db.ExecContext(ctx, `DROP TABLE tax_year_settings`); err != nil {
		t.Fatalf("dropping tax_year_settings: %v", err)
	}

	_, err := svc.Recalculate(ctx, hi.ID)
	if err == nil || !strings.Contains(err.Error(), "tax year settings") {
		t.Fatalf("Recalculate() error = %v, want it to mention tax year settings", err)
	}
	stored, err := svc.GetByID(ctx, hi.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if stored.AdvanceStartMonth != 0 {
		t.Errorf("stored AdvanceStartMonth = %d, want unchanged 0", stored.AdvanceStartMonth)
	}
}

func TestHealthMarkFiled_EarlyFilingMovesStart(t *testing.T) {
	svc, db := setupHealthInsuranceSvc(t)
	ctx := context.Background()
	saveHealthFilingMethod(t, db, 2025, domain.FilingMethodElectronic)
	hi := createHealthOverview(t, svc, 2025, domain.FullYearActivityMonths)
	recalculated, err := svc.Recalculate(ctx, hi.ID)
	if err != nil {
		t.Fatalf("Recalculate() error: %v", err)
	}
	if recalculated.AdvanceStartMonth != 6 {
		t.Fatalf("AdvanceStartMonth = %d, want 6", recalculated.AdvanceStartMonth)
	}
	svc.now = func() time.Time { return time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC) }

	filed, err := svc.MarkFiled(ctx, hi.ID)
	if err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}

	if filed.AdvanceStartMonth != 3 {
		t.Errorf("AdvanceStartMonth = %d, want 3", filed.AdvanceStartMonth)
	}
	if filed.NewMonthlyPrepay != recalculated.NewMonthlyPrepay {
		t.Errorf("NewMonthlyPrepay = %d, want unchanged %d", filed.NewMonthlyPrepay, recalculated.NewMonthlyPrepay)
	}
}
