package database

import (
	"context"
	"database/sql"
	"testing"
)

// A database at the state of migration 037 keeps its overviews through
// migrations 038-043 and the new columns take the values of a full year of main
// activity without an employment base.
func TestMigrations038To043_KeepExistingOverviews(t *testing.T) {
	ctx := context.Background()
	db, provider := newSequenceMigrationProvider(t)

	if _, err := provider.UpTo(ctx, 37); err != nil {
		t.Fatalf("migrating to 37: %v", err)
	}
	execAll(t, db,
		`INSERT INTO social_insurance_overviews (year, filing_type, final_assessment_base, total_insurance, new_monthly_prepay, status, created_at, updated_at)
		 VALUES (2024, 'regular', 15829300, 4622100, 385200, 'filed', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		`INSERT INTO health_insurance_overviews (year, filing_type, created_at, updated_at)
		 VALUES (2024, 'regular', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		`INSERT INTO tax_year_settings (year, created_at, updated_at)
		 VALUES (2024, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
	)

	if _, err := provider.UpTo(ctx, 43); err != nil {
		t.Fatalf("migrating to 43: %v", err)
	}

	var (
		finalBase, sumBase, insuredBase, prepay        int64
		activityMonths, secondaryMonths, startMonth    int
		secondaryReason, filingMethod, status          string
		newMonthlyBase, employmentBase, reportedEmploy sql.NullInt64
	)
	err := db.QueryRow(`SELECT final_assessment_base, sum_base, insured_base, new_monthly_prepay, activity_months,
			secondary_months, advance_start_month, secondary_reason, filing_method, status,
			new_monthly_base, employment_assessment_base, reported_employment_base
		FROM social_insurance_overviews WHERE year = 2024`).Scan(
		&finalBase, &sumBase, &insuredBase, &prepay, &activityMonths,
		&secondaryMonths, &startMonth, &secondaryReason, &filingMethod, &status,
		&newMonthlyBase, &employmentBase, &reportedEmploy)
	if err != nil {
		t.Fatalf("reading the social overview after migrating: %v", err)
	}
	if finalBase != 15829300 || prepay != 385200 || status != "filed" {
		t.Errorf("stored values changed: final base %d, advance %d, status %q", finalBase, prepay, status)
	}
	if activityMonths != 4095 {
		t.Errorf("activity_months = %d, want 4095", activityMonths)
	}
	if sumBase != finalBase || insuredBase != finalBase {
		t.Errorf("sum_base %d, insured_base %d, want both %d", sumBase, insuredBase, finalBase)
	}
	if secondaryMonths != 0 || secondaryReason != "" || startMonth != 0 || filingMethod != "" {
		t.Errorf("new columns = (secondary %d %q, start %d, method %q), want empty defaults",
			secondaryMonths, secondaryReason, startMonth, filingMethod)
	}
	if newMonthlyBase.Valid || employmentBase.Valid || reportedEmploy.Valid {
		t.Errorf("nullable columns = (%v, %v, %v), want NULL", newMonthlyBase, employmentBase, reportedEmploy)
	}

	if n := countRows(t, db, `SELECT COUNT(*) FROM health_insurance_overviews WHERE year = 2024 AND activity_months = 4095 AND no_minimum_months = 0 AND advance_start_month = 0`); n != 1 {
		t.Errorf("health overview rows with default new columns = %d, want 1", n)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM tax_year_settings WHERE year = 2024 AND filing_method = ''`); n != 1 {
		t.Errorf("tax year settings rows with default filing method = %d, want 1", n)
	}
}
