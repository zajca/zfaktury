package database

import (
	"database/sql"
	"embed"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed testdata/v024_seed.sql
var v024SeedFS embed.FS

func TestMain(m *testing.M) {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		panic("set goose dialect: " + err.Error())
	}
	os.Exit(m.Run())
}

// migrateUpTo runs every migration in internal/database/migrations
// up to and including the targetVersion.
func migrateUpTo(t *testing.T, db *sql.DB, targetVersion int64) {
	t.Helper()
	if _, err := db.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		t.Fatalf("disable fk: %v", err)
	}
	if err := goose.UpTo(db, "migrations", targetVersion); err != nil {
		t.Fatalf("goose up to %d: %v", targetVersion, err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("re-enable fk: %v", err)
	}
}

func openMigratedDB(t *testing.T, applyV025 bool) *sql.DB {
	t.Helper()
	tmp := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", tmp+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrateUpTo(t, db, 24)

	seed, err := v024SeedFS.ReadFile("testdata/v024_seed.sql")
	if err != nil {
		t.Fatalf("read seed: %v", err)
	}
	if _, err := db.Exec(string(seed)); err != nil {
		t.Fatalf("apply seed: %v", err)
	}

	if applyV025 {
		migrateUpTo(t, db, 25)
	}
	return db
}

func TestMultiCompanyMigration_CreatesDefaultCompanyFromSettings(t *testing.T) {
	db := openMigratedDB(t, true)
	var name, ico, dic string
	var vat int
	var accentColor sql.NullString
	var logoPath sql.NullString
	var createdAt, updatedAt string
	err := db.QueryRow(`SELECT name, ico, dic, vat_registered, accent_color, logo_path, created_at, updated_at FROM companies WHERE id = 1`).
		Scan(&name, &ico, &dic, &vat, &accentColor, &logoPath, &createdAt, &updatedAt)
	if err != nil {
		t.Fatalf("query company: %v", err)
	}
	if name != "Manas OSVČ" || ico != "12345678" || dic != "CZ12345678" || vat != 1 {
		t.Errorf("got name=%q ico=%q dic=%q vat=%d", name, ico, dic, vat)
	}
	if !accentColor.Valid || accentColor.String != "#1a56db" {
		t.Errorf("accent_color = %v, want '#1a56db'", accentColor)
	}
	if logoPath.Valid {
		t.Errorf("logo_path = %v, want NULL (empty string seeded; NULLIF should drop it)", logoPath)
	}
	if _, err := time.Parse(time.RFC3339, createdAt); err != nil {
		t.Errorf("created_at = %q does not parse as RFC3339: %v", createdAt, err)
	}
	if _, err := time.Parse(time.RFC3339, updatedAt); err != nil {
		t.Errorf("updated_at = %q does not parse as RFC3339: %v", updatedAt, err)
	}
}

func TestMultiCompanyMigration_StripsIdentityKeysFromSettings(t *testing.T) {
	db := openMigratedDB(t, true)
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM settings WHERE key IN
		('company_name','ico','dic','vat_registered','street','house_number','city','zip',
		 'email','phone','first_name','last_name','bank_account','bank_code','iban','swift',
		 'logo_path','accent_color')`).Scan(&n)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("found %d identity keys remaining in settings, want 0", n)
	}
}

func TestMultiCompanyMigration_PreservesNonIdentitySettings(t *testing.T) {
	db := openMigratedDB(t, true)
	var v string
	if err := db.QueryRow(`SELECT value FROM settings WHERE key = 'default_payment_method'`).Scan(&v); err != nil {
		t.Fatalf("query: %v", err)
	}
	if v != "bank_transfer" {
		t.Errorf("value = %q, want bank_transfer", v)
	}
}

func TestMultiCompanyMigration_FreshInstallProducesEmptyCompanies(t *testing.T) {
	// Fresh install: no v024 seed applied.
	tmp := filepath.Join(t.TempDir(), "fresh.db")
	db, err := sql.Open("sqlite", tmp+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migrateUpTo(t, db, 25)

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM companies`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("fresh install company count = %d, want 0", n)
	}
}

func TestMultiCompanyMigration_BackfillsContactsToDefaultCompany(t *testing.T) {
	db := openMigratedDB(t, true)
	rows, err := db.Query(`SELECT id, company_id FROM contacts ORDER BY id`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	type row struct{ ID, CompanyID int64 }
	var got []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.ID, &r.CompanyID); err != nil {
			t.Fatal(err)
		}
		got = append(got, r)
	}
	if len(got) != 2 {
		t.Fatalf("got %d contacts, want 2 from seed", len(got))
	}
	for _, r := range got {
		if r.CompanyID != 1 {
			t.Errorf("contact %d company_id = %d, want 1", r.ID, r.CompanyID)
		}
	}
}

func TestMultiCompanyMigration_BackfillsLeafEntitiesToDefaultCompany(t *testing.T) {
	db := openMigratedDB(t, true)
	// Tables the v024 seed populates with at least one row.
	tables := []string{"invoices", "invoice_items", "expenses"}
	for _, tbl := range tables {
		t.Run(tbl, func(t *testing.T) {
			var n, bad int
			if err := db.QueryRow(`SELECT COUNT(*) FROM ` + tbl).Scan(&n); err != nil {
				t.Fatalf("count: %v", err)
			}
			if n == 0 {
				t.Skipf("seed populates 0 rows in %s", tbl)
			}
			if err := db.QueryRow(`SELECT COUNT(*) FROM ` + tbl + ` WHERE company_id != 1`).Scan(&bad); err != nil {
				t.Fatalf("count bad: %v", err)
			}
			if bad != 0 {
				t.Errorf("%s: %d rows with company_id != 1", tbl, bad)
			}
		})
	}
}

func TestMultiCompanyMigration_AllPerCompanyTablesHaveColumn(t *testing.T) {
	db := openMigratedDB(t, true)
	tables := []string{
		"contacts", "expense_categories",
		"invoices", "invoice_items", "invoice_status_history", "invoice_documents",
		"recurring_invoices", "recurring_invoice_items",
		"expenses", "expense_items", "expense_documents",
		"recurring_expenses",
		"invoice_sequences",
		"payment_reminders",
		"tax_year_settings", "tax_prepayments",
		"tax_spouse_credits", "tax_child_credits", "tax_personal_credits",
		"tax_deductions", "tax_deduction_documents",
		"vat_returns", "vat_return_invoices", "vat_return_expenses",
		"vat_control_statements", "vat_control_statement_lines",
		"vies_summaries", "vies_summary_lines",
		"income_tax_returns", "social_insurance_overviews", "health_insurance_overviews",
		"investment_documents", "capital_income_entries", "security_transactions",
		"fakturoid_import_log",
		"settings",
	}
	for _, tbl := range tables {
		t.Run(tbl, func(t *testing.T) {
			rows, err := db.Query(`PRAGMA table_info(` + tbl + `)`)
			if err != nil {
				t.Fatalf("pragma: %v", err)
			}
			defer rows.Close()
			has := false
			for rows.Next() {
				var cid int
				var name, typ string
				var notnull, pk int
				var dflt sql.NullString
				_ = rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk)
				if name == "company_id" {
					has = true
				}
			}
			if !has {
				t.Errorf("%s: missing company_id column", tbl)
			}
		})
	}
}

// TestMultiCompanyMigrationProductionSized runs migration 025 against
// a synthetic ~5k-invoice fixture and asserts completion under 30s.
// Gated by the ZFAKTURY_RUN_BIG_MIGRATION_TEST environment variable
// (see Phase 2 task 15) so it stays out of the default CI run.
func TestMultiCompanyMigrationProductionSized(t *testing.T) {
	if os.Getenv("ZFAKTURY_RUN_BIG_MIGRATION_TEST") == "" {
		t.Skip("set ZFAKTURY_RUN_BIG_MIGRATION_TEST=1 to enable")
	}
	t.Skip("populated in Phase 2 task 15")
}
