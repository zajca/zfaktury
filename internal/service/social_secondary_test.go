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
)

// setupSecondaryOverview creates a full-year 2025 overview of a taxpayer who
// files the return by 1 April (the overview is due on 4 May 2026, so advances
// start in June 2026) with the given income.
func setupSecondaryOverview(t *testing.T, incomeCZK int64) (*SocialInsuranceService, *sql.DB, *domain.SocialInsuranceOverview) {
	t.Helper()
	svc, db := setupSocialInsuranceSvc(t)
	saveHealthFilingMethod(t, db, 2025, domain.FilingMethodStandard)
	if incomeCZK > 0 {
		seedHealthInvoice(t, db, incomeCZK)
	}
	sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
	if err := svc.Create(context.Background(), sio); err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	return svc, db, sio
}

func monthsOf(t *testing.T, m ...int) domain.ActivityMonths {
	t.Helper()
	mask, err := domain.NewActivityMonths(m)
	if err != nil {
		t.Fatalf("NewActivityMonths(%v) error: %v", m, err)
	}
	return mask
}

func czk(v int64) domain.Amount { return domain.NewAmount(v, 0) }

func assertPeriods(t *testing.T, got, want []domain.AdvancePeriod) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("AdvancePeriods = %+v, want %+v", got, want)
	}
}

func TestSetSecondaryActivity_MixedYear(t *testing.T) {
	svc, _, sio := setupSecondaryOverview(t, 240_000)

	got, err := svc.SetSecondaryActivity(context.Background(), sio.ID, monthsOf(t, 7, 8, 9, 10, 11, 12), domain.SecondaryReasonEmployment)
	if err != nil {
		t.Fatalf("SetSecondaryActivity() error: %v", err)
	}

	rows := []struct {
		name      string
		got, want domain.Amount
	}{
		{"MainTaxBaseShare", got.MainTaxBaseShare, czk(120_000)},
		{"SecondaryTaxBaseShare", got.SecondaryTaxBaseShare, czk(120_000)},
		{"AssessmentBase", got.AssessmentBase, czk(66_000)},
		{"SecondaryAssessmentBase", got.SecondaryAssessmentBase, czk(66_000)},
		{"MainPartialBase", got.MainPartialBase, czk(97_770)},
		{"SecondaryPartialBase", got.SecondaryPartialBase, czk(30_732)},
		{"MinAssessmentBase", got.MinAssessmentBase, czk(128_502)},
		{"FinalAssessmentBase", got.FinalAssessmentBase, czk(163_770)},
		{"SumBase", got.SumBase, czk(163_770)},
		{"InsuredBase", got.InsuredBase, czk(163_770)},
		{"TotalInsurance", got.TotalInsurance, czk(47_821)},
		{"SecondaryDecisiveAmount", got.SecondaryDecisiveAmount, czk(55_864)},
		{"NewMonthlyPrepay", got.NewMonthlyPrepay, czk(3_212)},
	}
	for _, r := range rows {
		if r.got != r.want {
			t.Errorf("%s = %d, want %d", r.name, r.got, r.want)
		}
	}
	if !got.AdvanceAsSecondary() {
		t.Error("AdvanceAsSecondary() = false, want true")
	}
	assertPeriods(t, got.AdvancePeriods, []domain.AdvancePeriod{{FromMonth: 6, MonthlyBase: czk(11_000), MonthlyPrepay: czk(3_212)}})
	if got.NewMonthlyBase == nil || *got.NewMonthlyBase != czk(11_000) {
		t.Errorf("NewMonthlyBase = %v, want 11000 CZK", got.NewMonthlyBase)
	}
	if len(got.Warnings) != 0 {
		t.Errorf("Warnings = %v, want none", got.Warnings)
	}
}

func TestSetSecondaryActivity_ReversedMonthsKeepMainAdvance(t *testing.T) {
	svc, _, sio := setupSecondaryOverview(t, 240_000)

	got, err := svc.SetSecondaryActivity(context.Background(), sio.ID, monthsOf(t, 1, 2, 3, 4, 5, 6), domain.SecondaryReasonEmployment)
	if err != nil {
		t.Fatalf("SetSecondaryActivity() error: %v", err)
	}

	if got.AdvanceAsSecondary() {
		t.Error("AdvanceAsSecondary() = true, want false when the year ends with main activity")
	}
	assertPeriods(t, got.AdvancePeriods, []domain.AdvancePeriod{
		{FromMonth: 6, MonthlyBase: czk(19_587), MonthlyPrepay: czk(5_720)},
		{FromMonth: 7, MonthlyBase: czk(17_139), MonthlyPrepay: czk(5_005)},
	})
}

func TestSetSecondaryActivity_SecondaryPartBelowDecisiveAmount(t *testing.T) {
	svc, _, sio := setupSecondaryOverview(t, 100_000)

	got, err := svc.SetSecondaryActivity(context.Background(), sio.ID, monthsOf(t, 7, 8, 9, 10, 11, 12), domain.SecondaryReasonEmployment)
	if err != nil {
		t.Fatalf("SetSecondaryActivity() error: %v", err)
	}

	if got.FinalAssessmentBase != czk(97_770) || got.TotalInsurance != czk(28_549) {
		t.Errorf("final/total = %d/%d, want 97770/28549 CZK", got.FinalAssessmentBase, got.TotalInsurance)
	}
	assertPeriods(t, got.AdvancePeriods, []domain.AdvancePeriod{{FromMonth: 6, MonthlyBase: czk(5_387), MonthlyPrepay: czk(1_574)}})
	if slices.Contains(got.Warnings, domain.WarningSocialAdvanceNotRequired) {
		t.Errorf("Warnings = %v, want no social_advance_not_required: the main months make the person participate", got.Warnings)
	}
}

func TestSetSecondaryActivity_SecondaryOnlyBelowDecisiveAmount(t *testing.T) {
	svc, _, sio := setupSecondaryOverview(t, 100_000)

	got, err := svc.SetSecondaryActivity(context.Background(), sio.ID, domain.FullYearActivityMonths, domain.SecondaryReasonStudent)
	if err != nil {
		t.Fatalf("SetSecondaryActivity() error: %v", err)
	}

	if got.FinalAssessmentBase != 0 || got.TotalInsurance != 0 || got.SecondaryAssessmentBase != 0 {
		t.Errorf("final/total/secondary = %d/%d/%d, want all 0", got.FinalAssessmentBase, got.TotalInsurance, got.SecondaryAssessmentBase)
	}
	assertPeriods(t, got.AdvancePeriods, []domain.AdvancePeriod{{FromMonth: 6}})
	if got.NewMonthlyPrepay != 0 || got.NewMonthlyBase == nil || *got.NewMonthlyBase != 0 {
		t.Errorf("advance = %d/%v, want 0/0", got.NewMonthlyPrepay, got.NewMonthlyBase)
	}
	if !slices.Contains(got.Warnings, domain.WarningSocialAdvanceNotRequired) {
		t.Errorf("Warnings = %v, want social_advance_not_required", got.Warnings)
	}

	cleared, err := svc.SetSecondaryActivity(context.Background(), sio.ID, 0, domain.SecondaryReasonNone)
	if err != nil {
		t.Fatalf("clearing the secondary activity: %v", err)
	}
	if cleared.SecondaryMonths != 0 || cleared.SecondaryReason != domain.SecondaryReasonNone {
		t.Errorf("secondary = %d/%q, want cleared", cleared.SecondaryMonths, cleared.SecondaryReason)
	}
	if slices.Contains(cleared.Warnings, domain.WarningSocialAdvanceNotRequired) {
		t.Errorf("Warnings = %v, want no social_advance_not_required after clearing", cleared.Warnings)
	}
	if want := czk(195_540); cleared.FinalAssessmentBase != want {
		t.Errorf("FinalAssessmentBase = %d, want %d (main minimum)", cleared.FinalAssessmentBase, want)
	}
}

func TestSetEmploymentBase_LowersInsuredBaseButNotTheAdvance(t *testing.T) {
	svc, _, sio := setupSecondaryOverview(t, 4_000_000)
	ctx := context.Background()

	base := czk(500_000)
	got, err := svc.SetEmploymentBase(ctx, sio.ID, &base)
	if err != nil {
		t.Fatalf("SetEmploymentBase() error: %v", err)
	}

	if got.InsuredBase != czk(1_734_736) || got.TotalInsurance != czk(506_543) || got.SumBase != czk(2_700_000) {
		t.Errorf("insured/total/sum = %d/%d/%d, want 1734736/506543/2700000 CZK", got.InsuredBase, got.TotalInsurance, got.SumBase)
	}
	if got.ReportedEmploymentBase == nil || *got.ReportedEmploymentBase != base {
		t.Errorf("ReportedEmploymentBase = %v, want %d", got.ReportedEmploymentBase, base)
	}
	if got.EmploymentAssessmentBase == nil || *got.EmploymentAssessmentBase != base {
		t.Errorf("EmploymentAssessmentBase = %v, want %d", got.EmploymentAssessmentBase, base)
	}
	assertPeriods(t, got.AdvancePeriods, []domain.AdvancePeriod{{FromMonth: 6, MonthlyBase: czk(183_334), MonthlyPrepay: czk(53_534)}})

	removed, err := svc.SetEmploymentBase(ctx, sio.ID, nil)
	if err != nil {
		t.Fatalf("SetEmploymentBase(nil) error: %v", err)
	}
	if removed.InsuredBase != czk(2_200_000) || removed.TotalInsurance != czk(642_400) {
		t.Errorf("insured/total = %d/%d, want 2200000/642400 CZK", removed.InsuredBase, removed.TotalInsurance)
	}
	if removed.EmploymentAssessmentBase != nil || removed.ReportedEmploymentBase != nil {
		t.Errorf("employment fields = %v/%v, want nil", removed.EmploymentAssessmentBase, removed.ReportedEmploymentBase)
	}
}

func TestSetEmploymentBase_SmallBaseIsNotReported(t *testing.T) {
	svc, _, sio := setupSecondaryOverview(t, 0)

	base := czk(30_000)
	got, err := svc.SetEmploymentBase(context.Background(), sio.ID, &base)
	if err != nil {
		t.Fatalf("SetEmploymentBase() error: %v", err)
	}
	if got.ReportedEmploymentBase != nil {
		t.Errorf("ReportedEmploymentBase = %d, want nil", *got.ReportedEmploymentBase)
	}
	if got.InsuredBase != czk(195_540) || got.TotalInsurance != czk(57_098) {
		t.Errorf("insured/total = %d/%d, want 195540/57098 CZK", got.InsuredBase, got.TotalInsurance)
	}
}

func TestSecondaryActivity_InvalidInputWritesNothing(t *testing.T) {
	svc, _, sio := setupSecondaryOverview(t, 240_000)
	ctx := context.Background()
	if _, err := svc.SetActivityMonths(ctx, sio.ID, monthsOf(t, 1, 2, 3, 4, 5, 6)); err != nil {
		t.Fatalf("SetActivityMonths() error: %v", err)
	}
	before, err := svc.GetByID(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}

	negative, halere := domain.Amount(-100), domain.NewAmount(12_345, 67)
	tests := []struct {
		name string
		call func() error
	}{
		{"month outside activity", func() error {
			_, err := svc.SetSecondaryActivity(ctx, sio.ID, monthsOf(t, 7), domain.SecondaryReasonEmployment)
			return err
		}},
		{"months without a reason", func() error {
			_, err := svc.SetSecondaryActivity(ctx, sio.ID, monthsOf(t, 1), domain.SecondaryReasonNone)
			return err
		}},
		{"reason without months", func() error {
			_, err := svc.SetSecondaryActivity(ctx, sio.ID, 0, domain.SecondaryReasonCare)
			return err
		}},
		{"negative employment base", func() error {
			_, err := svc.SetEmploymentBase(ctx, sio.ID, &negative)
			return err
		}},
		{"employment base with halere", func() error {
			_, err := svc.SetEmploymentBase(ctx, sio.ID, &halere)
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("error = %v, want ErrInvalidInput", err)
			}
			after, err := svc.GetByID(ctx, sio.ID)
			if err != nil {
				t.Fatalf("GetByID() error: %v", err)
			}
			if after.SecondaryMonths != before.SecondaryMonths || after.EmploymentAssessmentBase != nil || !after.UpdatedAt.Equal(before.UpdatedAt) {
				t.Errorf("the overview changed after a rejected call: %+v", after)
			}
		})
	}
}

func TestSetActivityMonths_RejectsDroppingASecondaryMonth(t *testing.T) {
	svc, _, sio := setupSecondaryOverview(t, 240_000)
	ctx := context.Background()
	if _, err := svc.SetSecondaryActivity(ctx, sio.ID, monthsOf(t, 7, 8, 9, 10, 11, 12), domain.SecondaryReasonEmployment); err != nil {
		t.Fatalf("SetSecondaryActivity() error: %v", err)
	}

	_, err := svc.SetActivityMonths(ctx, sio.ID, monthsOf(t, 1, 2, 3, 4, 5, 6))
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("SetActivityMonths() error = %v, want ErrInvalidInput", err)
	}
	stored, err := svc.GetByID(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GetByID() error: %v", err)
	}
	if stored.ActivityMonths != domain.FullYearActivityMonths {
		t.Errorf("ActivityMonths = %d, want unchanged full year", stored.ActivityMonths)
	}
}

func TestSecondaryActivity_FiledOverviewCannotChange(t *testing.T) {
	svc, _, sio := setupSecondaryOverview(t, 240_000)
	ctx := context.Background()
	if _, err := svc.MarkFiled(ctx, sio.ID); err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}

	if _, err := svc.SetSecondaryActivity(ctx, sio.ID, monthsOf(t, 7), domain.SecondaryReasonEmployment); !errors.Is(err, domain.ErrFilingAlreadyFiled) {
		t.Errorf("SetSecondaryActivity() error = %v, want ErrFilingAlreadyFiled", err)
	}
	base := czk(1_000)
	if _, err := svc.SetEmploymentBase(ctx, sio.ID, &base); !errors.Is(err, domain.ErrFilingAlreadyFiled) {
		t.Errorf("SetEmploymentBase() error = %v, want ErrFilingAlreadyFiled", err)
	}
}

func TestMarkFiled_EarlyFilingMovesTheSecondaryAdvanceStart(t *testing.T) {
	svc, _, sio := setupSecondaryOverview(t, 240_000)
	ctx := context.Background()
	if _, err := svc.SetSecondaryActivity(ctx, sio.ID, monthsOf(t, 7, 8, 9, 10, 11, 12), domain.SecondaryReasonEmployment); err != nil {
		t.Fatalf("SetSecondaryActivity() error: %v", err)
	}
	svc.now = func() time.Time { return time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC) }

	got, err := svc.MarkFiled(ctx, sio.ID)
	if err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}

	if got.AdvanceStartMonth != 4 {
		t.Errorf("AdvanceStartMonth = %d, want 4", got.AdvanceStartMonth)
	}
	assertPeriods(t, got.AdvancePeriods, []domain.AdvancePeriod{{FromMonth: 4, MonthlyBase: czk(11_000), MonthlyPrepay: czk(3_212)}})
	if slices.Contains(got.Warnings, domain.WarningAdvanceDiffersFromXML) {
		t.Errorf("Warnings = %v, want no advance_differs_from_xml: the base did not change", got.Warnings)
	}
}

func TestMarkFiled_SecondaryOnlyNotRequiredKeepsZeroAdvance(t *testing.T) {
	svc, _, sio := setupSecondaryOverview(t, 100_000)
	ctx := context.Background()
	if _, err := svc.SetSecondaryActivity(ctx, sio.ID, domain.FullYearActivityMonths, domain.SecondaryReasonStudent); err != nil {
		t.Fatalf("SetSecondaryActivity() error: %v", err)
	}
	svc.now = func() time.Time { return time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC) }

	got, err := svc.MarkFiled(ctx, sio.ID)
	if err != nil {
		t.Fatalf("MarkFiled() error: %v", err)
	}
	assertPeriods(t, got.AdvancePeriods, []domain.AdvancePeriod{{FromMonth: 4}})
}

func TestGenerateXML_MixedYear(t *testing.T) {
	svc, db, sio := setupSecondaryOverview(t, 240_000)
	ctx := context.Background()
	seedSocialInsuranceXMLSettings(t, db)
	if _, err := svc.SetSecondaryActivity(ctx, sio.ID, monthsOf(t, 7, 8, 9, 10, 11, 12), domain.SecondaryReasonEmployment); err != nil {
		t.Fatalf("SetSecondaryActivity() error: %v", err)
	}

	got, err := svc.GenerateXML(ctx, sio.ID)
	if err != nil {
		t.Fatalf("GenerateXML() error: %v", err)
	}
	xml := string(got.XMLData)
	for _, want := range []string{`<druc>S</druc>`, `<zam>A</zam>`, `<vvz h="66000" v="66000">`, `<zal ved="V" vz="11000" dp="3212">`, `<poj>47821</poj>`} {
		if !strings.Contains(xml, want) {
			t.Errorf("XML does not contain %q", want)
		}
	}
}
