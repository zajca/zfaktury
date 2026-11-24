package database

import (
	"context"
	"testing"
)

func TestMigration037_IncomeTaxCreditRows(t *testing.T) {
	ctx := context.Background()
	db, provider := newSequenceMigrationProvider(t)

	if _, err := provider.UpTo(ctx, 30); err != nil {
		t.Fatalf("migrating to 30: %v", err)
	}

	execAll(t, db,
		`INSERT INTO tax_personal_credits (year, disability_level, created_at, updated_at) VALUES (2021, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		`INSERT INTO tax_personal_credits (year, disability_level, created_at, updated_at) VALUES (2022, 2, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		`INSERT INTO tax_personal_credits (year, disability_level, created_at, updated_at) VALUES (2023, 3, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
	)

	if _, err := provider.UpTo(ctx, 37); err != nil {
		t.Fatalf("migrating to 37: %v", err)
	}

	assertLevelZTPP := func(year, wantLevel, wantZTPP int) {
		t.Helper()
		var level, ztpp int
		if err := db.QueryRow(`SELECT disability_level, ztpp FROM tax_personal_credits WHERE year = ?`, year).Scan(&level, &ztpp); err != nil {
			t.Fatalf("reading year %d: %v", year, err)
		}
		if level != wantLevel || ztpp != wantZTPP {
			t.Errorf("year %d = (level %d, ztpp %d), want (%d, %d)", year, level, ztpp, wantLevel, wantZTPP)
		}
	}
	assertLevelZTPP(2021, 1, 0)
	assertLevelZTPP(2022, 2, 0)
	assertLevelZTPP(2023, 0, 1)

	for _, col := range []string{"credit_disability_12", "credit_disability_3", "credit_disability_ztpp"} {
		if n := countRows(t, db, `SELECT COUNT(*) FROM pragma_table_info('income_tax_returns') WHERE name = '`+col+`'`); n != 1 {
			t.Errorf("income_tax_returns column %s count = %d, want 1", col, n)
		}
	}

	if _, err := provider.Down(ctx); err != nil {
		t.Fatalf("migrating down from 37: %v", err)
	}

	var level int
	if err := db.QueryRow(`SELECT disability_level FROM tax_personal_credits WHERE year = 2023`).Scan(&level); err != nil {
		t.Fatalf("reading year 2023 after down: %v", err)
	}
	if level != 3 {
		t.Errorf("year 2023 level after down = %d, want 3", level)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM pragma_table_info('tax_personal_credits') WHERE name = 'ztpp'`); n != 0 {
		t.Errorf("ztpp column still present after down")
	}
}
