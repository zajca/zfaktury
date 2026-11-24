package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/testutil"
)

func TestSocialInsuranceOverviewRepository_Create(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		ActivityMonths: domain.FullYearActivityMonths,
	}

	if err := repo.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if sio.ID == 0 {
		t.Error("expected non-zero ID after Create")
	}
	if sio.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if sio.Status != domain.FilingStatusDraft {
		t.Errorf("Status = %q, want %q", sio.Status, domain.FilingStatusDraft)
	}
}

func TestSocialInsuranceOverviewRepository_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		TotalRevenue:   domain.NewAmount(800000, 0),
		TotalExpenses:  domain.NewAmount(300000, 0),
		TotalInsurance: domain.NewAmount(145000, 0),
		ActivityMonths: domain.FullYearActivityMonths,
	}
	if err := repo.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	if got.Year != 2025 {
		t.Errorf("Year = %d, want 2025", got.Year)
	}
	if got.TotalRevenue != domain.NewAmount(800000, 0) {
		t.Errorf("TotalRevenue = %d, want %d", got.TotalRevenue, domain.NewAmount(800000, 0))
	}
	if got.TotalInsurance != domain.NewAmount(145000, 0) {
		t.Errorf("TotalInsurance = %d, want %d", got.TotalInsurance, domain.NewAmount(145000, 0))
	}
}

func TestSocialInsuranceOverviewRepository_AdvanceFieldsRoundTrip(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:              2025,
		FilingType:        domain.FilingTypeRegular,
		ActivityMonths:    domain.FullYearActivityMonths,
		FilingMethod:      domain.FilingMethodElectronic,
		AdvanceStartMonth: 6,
		AdvancePeriods: []domain.AdvancePeriod{
			{FromMonth: 6, MonthlyBase: domain.NewAmount(19_587, 0), MonthlyPrepay: domain.NewAmount(5_720, 0)},
			{FromMonth: 7, MonthlyBase: domain.NewAmount(17_139, 0), MonthlyPrepay: domain.NewAmount(5_005, 0)},
		},
		Warnings: []string{domain.WarningFilingMethodMissing, domain.WarningAdvanceMinimumMissing},
	}
	if err := repo.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.FilingMethod != domain.FilingMethodElectronic || got.AdvanceStartMonth != 6 {
		t.Errorf("method/start = %q/%d, want electronic/6", got.FilingMethod, got.AdvanceStartMonth)
	}
	if len(got.AdvancePeriods) != 2 || got.AdvancePeriods[0] != sio.AdvancePeriods[0] || got.AdvancePeriods[1] != sio.AdvancePeriods[1] {
		t.Errorf("AdvancePeriods = %+v, want %+v", got.AdvancePeriods, sio.AdvancePeriods)
	}
	if len(got.Warnings) != 2 || got.Warnings[0] != domain.WarningFilingMethodMissing || got.Warnings[1] != domain.WarningAdvanceMinimumMissing {
		t.Errorf("Warnings = %v", got.Warnings)
	}

	got.AdvancePeriods = nil
	got.Warnings = nil
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("Update() error: %v", err)
	}
	var storedPeriods, storedWarnings string
	if err := db.QueryRowContext(ctx, `SELECT advance_periods, warnings FROM social_insurance_overviews WHERE id = ?`, sio.ID).Scan(&storedPeriods, &storedWarnings); err != nil {
		t.Fatalf("reading stored columns: %v", err)
	}
	if storedPeriods != "" || storedWarnings != "" {
		t.Errorf("empty periods/warnings stored as %q/%q, want empty strings", storedPeriods, storedWarnings)
	}
	again, err := repo.GetByID(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetByID() after clearing: %v", err)
	}
	if again.AdvancePeriods != nil || again.Warnings != nil {
		t.Errorf("periods/warnings = %v/%v, want nil", again.AdvancePeriods, again.Warnings)
	}
}

func TestSocialInsuranceOverviewRepository_RejectsUnorderedAdvancePeriods(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
	if err := repo.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	stored := `[{"from_month":7,"monthly_base":1,"monthly_prepay":1},{"from_month":6,"monthly_base":1,"monthly_prepay":1}]`
	if _, err := db.ExecContext(ctx, `UPDATE social_insurance_overviews SET advance_periods = ? WHERE id = ?`, stored, sio.ID); err != nil {
		t.Fatalf("storing periods: %v", err)
	}

	if _, err := repo.GetByID(ctx, sio.ID); err == nil {
		t.Fatal("GetByID() error = nil, want an error for unordered periods")
	}
}

func TestSocialInsuranceOverviewRepository_SecondaryEmploymentRoundTrip(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	employment := domain.NewAmount(500_000, 0)
	reported := domain.NewAmount(450_000, 0)
	sio := &domain.SocialInsuranceOverview{
		Year:                     2025,
		FilingType:               domain.FilingTypeRegular,
		ActivityMonths:           domain.FullYearActivityMonths,
		SecondaryMonths:          domain.ActivityMonths(0b111111000000),
		SecondaryReason:          domain.SecondaryReasonEmployment,
		EmploymentAssessmentBase: &employment,
		ReportedEmploymentBase:   &reported,
		MainTaxBaseShare:         domain.NewAmount(120_000, 0),
		SecondaryTaxBaseShare:    domain.NewAmount(120_000, 0),
		SecondaryAssessmentBase:  domain.NewAmount(66_000, 0),
		MainPartialBase:          domain.NewAmount(97_770, 0),
		SecondaryPartialBase:     domain.NewAmount(30_732, 0),
		SecondaryDecisiveAmount:  domain.NewAmount(55_864, 0),
		SumBase:                  domain.NewAmount(2_700_000, 0),
		InsuredBase:              domain.NewAmount(1_734_736, 0),
	}
	if err := repo.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.SecondaryMonths != sio.SecondaryMonths || got.SecondaryReason != domain.SecondaryReasonEmployment {
		t.Errorf("secondary = %d/%q, want %d/employment", got.SecondaryMonths, got.SecondaryReason, sio.SecondaryMonths)
	}
	if got.EmploymentAssessmentBase == nil || *got.EmploymentAssessmentBase != employment {
		t.Errorf("EmploymentAssessmentBase = %v, want %d", got.EmploymentAssessmentBase, employment)
	}
	if got.ReportedEmploymentBase == nil || *got.ReportedEmploymentBase != reported {
		t.Errorf("ReportedEmploymentBase = %v, want %d", got.ReportedEmploymentBase, reported)
	}
	if got.MainTaxBaseShare != sio.MainTaxBaseShare || got.SecondaryTaxBaseShare != sio.SecondaryTaxBaseShare ||
		got.SecondaryAssessmentBase != sio.SecondaryAssessmentBase || got.MainPartialBase != sio.MainPartialBase ||
		got.SecondaryPartialBase != sio.SecondaryPartialBase || got.SecondaryDecisiveAmount != sio.SecondaryDecisiveAmount ||
		got.SumBase != sio.SumBase || got.InsuredBase != sio.InsuredBase {
		t.Errorf("stored rows = %+v, want the values written", got)
	}

	got.SecondaryMonths = 0
	got.SecondaryReason = domain.SecondaryReasonNone
	got.EmploymentAssessmentBase = nil
	got.ReportedEmploymentBase = nil
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("Update() error: %v", err)
	}
	again, err := repo.GetByID(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetByID() after clearing: %v", err)
	}
	if again.SecondaryMonths != 0 || again.SecondaryReason != domain.SecondaryReasonNone {
		t.Errorf("secondary after clearing = %d/%q, want 0/empty", again.SecondaryMonths, again.SecondaryReason)
	}
	if again.EmploymentAssessmentBase != nil || again.ReportedEmploymentBase != nil {
		t.Errorf("employment fields after clearing = %v/%v, want nil", again.EmploymentAssessmentBase, again.ReportedEmploymentBase)
	}
}

func TestSocialInsuranceOverviewRepository_RejectsSecondaryMonthsOutOfRange(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
	if err := repo.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE social_insurance_overviews SET secondary_months = 4095 WHERE id = ?`, sio.ID); err != nil {
		t.Fatalf("storing the highest valid mask: %v", err)
	}
	if _, err := repo.GetByID(ctx, sio.ID); err != nil {
		t.Fatalf("GetByID() with mask 4095 error: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE social_insurance_overviews SET secondary_months = 4096 WHERE id = ?`, sio.ID); err == nil {
		t.Fatal("storing mask 4096 error = nil, want the CHECK constraint to reject it")
	}
}

func TestSocialInsuranceOverviewRepository_NewOverviewHasNoSecondaryOrEmployment(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
	if err := repo.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	got, err := repo.GetByID(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.SecondaryMonths != 0 || got.EmploymentAssessmentBase != nil || got.ReportedEmploymentBase != nil {
		t.Errorf("defaults = %d/%v/%v, want no secondary months and no employment", got.SecondaryMonths, got.EmploymentAssessmentBase, got.ReportedEmploymentBase)
	}
}

func TestSocialInsuranceOverviewRepository_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent social_insurance_overview")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestSocialInsuranceOverviewRepository_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		ActivityMonths: domain.FullYearActivityMonths,
	}
	if err := repo.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	sio.TotalRevenue = domain.NewAmount(1200000, 0)
	sio.TotalInsurance = domain.NewAmount(175000, 0)
	sio.Difference = domain.NewAmount(50000, 0)

	if err := repo.Update(ctx, sio); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	got, err := repo.GetByID(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.TotalRevenue != domain.NewAmount(1200000, 0) {
		t.Errorf("TotalRevenue = %d, want %d", got.TotalRevenue, domain.NewAmount(1200000, 0))
	}
	if got.TotalInsurance != domain.NewAmount(175000, 0) {
		t.Errorf("TotalInsurance = %d, want %d", got.TotalInsurance, domain.NewAmount(175000, 0))
	}
}

func TestSocialInsuranceOverviewRepository_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		ActivityMonths: domain.FullYearActivityMonths,
	}
	if err := repo.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := repo.Delete(ctx, sio.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	_, err := repo.GetByID(ctx, sio.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestSocialInsuranceOverviewRepository_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent delete")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestSocialInsuranceOverviewRepository_List(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	for i, ft := range []string{domain.FilingTypeRegular, domain.FilingTypeCorrective} {
		sio := &domain.SocialInsuranceOverview{
			Year:           2025,
			FilingType:     ft,
			ActivityMonths: domain.FullYearActivityMonths,
		}
		if err := repo.Create(ctx, sio); err != nil {
			t.Fatalf("Create() %d error: %v", i, err)
		}
	}

	// Different year.
	sio2024 := &domain.SocialInsuranceOverview{
		Year:           2024,
		FilingType:     domain.FilingTypeRegular,
		ActivityMonths: domain.FullYearActivityMonths,
	}
	if err := repo.Create(ctx, sio2024); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	returns, err := repo.List(ctx, 2025)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(returns) != 2 {
		t.Errorf("List(2025) returned %d items, want 2", len(returns))
	}

	returns2024, err := repo.List(ctx, 2024)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(returns2024) != 1 {
		t.Errorf("List(2024) returned %d items, want 1", len(returns2024))
	}
}

func TestSocialInsuranceOverviewRepository_GetByYear(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		ActivityMonths: domain.FullYearActivityMonths,
	}
	if err := repo.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByYear(ctx, 2025, domain.FilingTypeRegular)
	if err != nil {
		t.Fatalf("GetByYear() error: %v", err)
	}
	if got.ID != sio.ID {
		t.Errorf("GetByYear() ID = %d, want %d", got.ID, sio.ID)
	}

	// Non-existent.
	_, err = repo.GetByYear(ctx, 2099, domain.FilingTypeRegular)
	if err == nil {
		t.Error("expected error for non-existent year")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestActivityMonthsFromDB(t *testing.T) {
	tests := []struct {
		name    string
		value   int64
		want    domain.ActivityMonths
		wantErr bool
	}{
		{name: "single month", value: 1, want: domain.ActivityMonths(1)},
		{name: "july to december", value: 4032, want: domain.ActivityMonths(4032)},
		{name: "full year", value: 4095, want: domain.FullYearActivityMonths},
		{name: "zero mask", value: 0, wantErr: true},
		{name: "negative", value: -1, wantErr: true},
		{name: "bit outside the year", value: 4096, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := activityMonthsFromDB(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatal("activityMonthsFromDB() error = nil, want an out-of-range error")
				}
				if errors.Is(err, domain.ErrInvalidInput) {
					t.Errorf("activityMonthsFromDB() error = %v, want a plain error instead of ErrInvalidInput", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("activityMonthsFromDB() error: %v", err)
			}
			if got != tt.want {
				t.Errorf("activityMonthsFromDB() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSocialInsuranceOverviewRepository_ActivityMonthsRoundTrip(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	sio := &domain.SocialInsuranceOverview{
		Year:           2025,
		FilingType:     domain.FilingTypeRegular,
		ActivityMonths: domain.ActivityMonths(4032),
	}
	if err := repo.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.ActivityMonths != domain.ActivityMonths(4032) {
		t.Errorf("ActivityMonths after Create = %d, want 4032", got.ActivityMonths)
	}

	got.ActivityMonths = domain.ActivityMonths(4092)
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	reloaded, err := repo.GetByID(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetByID() after update error: %v", err)
	}
	if reloaded.ActivityMonths != domain.ActivityMonths(4092) {
		t.Errorf("ActivityMonths after Update = %d, want 4092", reloaded.ActivityMonths)
	}
}

func TestSocialInsuranceOverviewRepository_ActivityMonthsMigrationBackfill(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	// A row that omits activity_months takes the default, which stands for a
	// full year of activity.
	now := time.Now().Format(time.RFC3339)
	_, err := db.ExecContext(ctx, `
		INSERT INTO social_insurance_overviews (year, filing_type, created_at, updated_at)
		VALUES (2024, 'regular', ?, ?)`, now, now)
	if err != nil {
		t.Fatalf("inserting row without activity_months: %v", err)
	}

	got, err := repo.GetByYear(ctx, 2024, domain.FilingTypeRegular)
	if err != nil {
		t.Fatalf("GetByYear() error: %v", err)
	}
	if got.ActivityMonths != domain.FullYearActivityMonths {
		t.Errorf("ActivityMonths after migration = %d, want %d", got.ActivityMonths, domain.FullYearActivityMonths)
	}
}

func TestSocialInsuranceOverviewRepository_NewMonthlyBase(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := NewSocialInsuranceOverviewRepository(db)
	ctx := context.Background()

	base := domain.NewAmount(16_295, 0)
	sio := &domain.SocialInsuranceOverview{
		Year:           2024,
		FilingType:     domain.FilingTypeRegular,
		ActivityMonths: domain.FullYearActivityMonths,
		NewMonthlyBase: &base,
	}
	if err := repo.Create(ctx, sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	got, err := repo.GetByID(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if got.NewMonthlyBase == nil {
		t.Fatal("NewMonthlyBase = nil, want the stored base")
	}
	if *got.NewMonthlyBase != base {
		t.Errorf("NewMonthlyBase = %d, want %d", *got.NewMonthlyBase, base)
	}

	got.NewMonthlyBase = nil
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	cleared, err := repo.GetByID(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetByID() after clearing error: %v", err)
	}
	if cleared.NewMonthlyBase != nil {
		t.Errorf("NewMonthlyBase = %d, want nil after clearing", *cleared.NewMonthlyBase)
	}
}
