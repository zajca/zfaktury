package database

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
)

// newSequenceMigrationProvider opens a file-backed SQLite database on a single
// connection with foreign keys off (as Migrate does) and returns a goose
// provider over the embedded migrations.
func newSequenceMigrationProvider(t *testing.T) (*sql.DB, *goose.Provider) {
	t.Helper()

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "migrate.db"))
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		t.Fatalf("disabling foreign keys: %v", err)
	}

	sub, err := fs.Sub(MigrationsFS(), "migrations")
	if err != nil {
		t.Fatalf("getting migrations sub-fs: %v", err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, sub)
	if err != nil {
		t.Fatalf("creating goose provider: %v", err)
	}
	return db, provider
}

func execAll(t *testing.T, db *sql.DB, statements ...string) {
	t.Helper()
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("executing %q: %v", stmt, err)
		}
	}
}

func countRows(t *testing.T, db *sql.DB, query string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query).Scan(&n); err != nil {
		t.Fatalf("querying %q: %v", query, err)
	}
	return n
}

func assertForeignKeyCheckEmpty(t *testing.T, db *sql.DB) {
	t.Helper()
	rows, err := db.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatalf("running foreign_key_check: %v", err)
	}
	defer func() { _ = rows.Close() }()
	if rows.Next() {
		t.Error("PRAGMA foreign_key_check reported violations")
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("reading foreign_key_check: %v", err)
	}
}

func TestMigration029_InvoiceSequencesUniqueActive(t *testing.T) {
	ctx := context.Background()
	db, provider := newSequenceMigrationProvider(t)

	if _, err := provider.UpTo(ctx, 28); err != nil {
		t.Fatalf("migrating to 28: %v", err)
	}

	// Pre-migration data: a live sequence (id 1) and a soft-deleted one (id 2),
	// each referenced by an invoice.
	execAll(t, db,
		`INSERT INTO contacts (type, name) VALUES ('company', 'Example Customer s.r.o.')`,
		`INSERT INTO invoice_sequences (id, prefix, next_number, year, format_pattern) VALUES (1, 'FV', 5, 2026, '{prefix}{year}{number:04d}')`,
		`INSERT INTO invoice_sequences (id, prefix, next_number, year, format_pattern, deleted_at) VALUES (2, 'ZF', 3, 2026, '{prefix}{number}', '2026-02-01T00:00:00Z')`,
		`INSERT INTO invoices (sequence_id, invoice_number, issue_date, due_date, customer_id) VALUES (1, 'FV20260001', '2026-01-10', '2026-01-24', 1)`,
		`INSERT INTO invoices (sequence_id, invoice_number, issue_date, due_date, customer_id) VALUES (2, 'ZF-1', '2026-01-11', '2026-01-25', 1)`,
	)

	// The constraint blocks recreating a soft-deleted prefix/year before 029.
	if _, err := db.Exec(`INSERT INTO invoice_sequences (prefix, next_number, year) VALUES ('ZF', 1, 2026)`); err == nil {
		t.Fatal("expected UNIQUE violation before migration 029")
	}

	if _, err := provider.UpTo(ctx, 29); err != nil {
		t.Fatalf("migrating to 29: %v", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enabling foreign keys: %v", err)
	}
	assertForeignKeyCheckEmpty(t, db)

	// Ids, values and invoice references survive the rebuild.
	var prefix string
	var nextNumber int
	var deletedAt sql.NullString
	if err := db.QueryRow(`SELECT prefix, next_number, deleted_at FROM invoice_sequences WHERE id = 1`).Scan(&prefix, &nextNumber, &deletedAt); err != nil {
		t.Fatalf("reading sequence 1: %v", err)
	}
	if prefix != "FV" || nextNumber != 5 || deletedAt.Valid {
		t.Errorf("sequence 1 = (%q, %d, %v), want (FV, 5, NULL)", prefix, nextNumber, deletedAt)
	}
	if err := db.QueryRow(`SELECT prefix, next_number, deleted_at FROM invoice_sequences WHERE id = 2`).Scan(&prefix, &nextNumber, &deletedAt); err != nil {
		t.Fatalf("reading sequence 2: %v", err)
	}
	if prefix != "ZF" || nextNumber != 3 || !deletedAt.Valid {
		t.Errorf("sequence 2 = (%q, %d, %v), want (ZF, 3, set)", prefix, nextNumber, deletedAt)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM invoices i JOIN invoice_sequences s ON s.id = i.sequence_id`); n != 2 {
		t.Errorf("invoices joined to sequences = %d, want 2", n)
	}

	// A soft-deleted prefix/year can be recreated; a live duplicate cannot.
	execAll(t, db, `INSERT INTO invoice_sequences (prefix, next_number, year) VALUES ('ZF', 1, 2026)`)
	if _, err := db.Exec(`INSERT INTO invoice_sequences (prefix, next_number, year) VALUES ('ZF', 1, 2026)`); err == nil {
		t.Error("expected unique violation for two live ZF/2026 rows")
	}
	if _, err := db.Exec(`INSERT INTO invoice_sequences (prefix, next_number, year) VALUES ('FV', 1, 2026)`); err == nil {
		t.Error("expected unique violation for two live FV/2026 rows")
	}

	// A foreign key to a missing sequence is still rejected.
	if _, err := db.Exec(`INSERT INTO invoices (sequence_id, invoice_number, issue_date, due_date, customer_id) VALUES (999, 'X-1', '2026-01-10', '2026-01-24', 1)`); err == nil {
		t.Error("expected foreign key violation for missing sequence")
	}
}

func TestMigration029_Down(t *testing.T) {
	ctx := context.Background()
	db, provider := newSequenceMigrationProvider(t)

	if _, err := provider.UpTo(ctx, 29); err != nil {
		t.Fatalf("migrating to 29: %v", err)
	}
	execAll(t, db,
		`INSERT INTO invoice_sequences (id, prefix, next_number, year) VALUES (1, 'FV', 2, 2026)`,
	)

	if _, err := provider.DownTo(ctx, 28); err != nil {
		t.Fatalf("migrating down to 28: %v", err)
	}

	if n := countRows(t, db, `SELECT COUNT(*) FROM invoice_sequences WHERE id = 1 AND prefix = 'FV'`); n != 1 {
		t.Errorf("sequence rows after down = %d, want 1", n)
	}
	_, err := db.Exec(`INSERT INTO invoice_sequences (prefix, next_number, year) VALUES ('FV', 1, 2026)`)
	if err == nil || !strings.Contains(err.Error(), "UNIQUE") {
		t.Errorf("expected UNIQUE violation after down migration, got %v", err)
	}
}
