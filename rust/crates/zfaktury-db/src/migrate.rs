use rusqlite::Connection;

use crate::error::DbError;

/// Embedded migration SQL files, sorted by version number.
/// Each entry is (version, name, sql).
const MIGRATIONS: &[(i64, &str, &str)] = &[
    (
        1,
        "initial_schema",
        include_str!("../../../migrations/V1__initial_schema.sql"),
    ),
    (
        4,
        "expense_categories",
        include_str!("../../../migrations/V4__expense_categories.sql"),
    ),
    (
        5,
        "vat_unreliable_timestamp",
        include_str!("../../../migrations/V5__vat_unreliable_timestamp.sql"),
    ),
    (
        6,
        "expense_documents",
        include_str!("../../../migrations/V6__expense_documents.sql"),
    ),
    (
        7,
        "invoice_relations",
        include_str!("../../../migrations/V7__invoice_relations.sql"),
    ),
    (
        8,
        "tax_review",
        include_str!("../../../migrations/V8__tax_review.sql"),
    ),
    (
        9,
        "recurring_invoices",
        include_str!("../../../migrations/V9__recurring_invoices.sql"),
    ),
    (
        10,
        "recurring_expenses",
        include_str!("../../../migrations/V10__recurring_expenses.sql"),
    ),
    (
        12,
        "invoice_status_history",
        include_str!("../../../migrations/V12__invoice_status_history.sql"),
    ),
    (
        13,
        "payment_reminders",
        include_str!("../../../migrations/V13__payment_reminders.sql"),
    ),
    (
        14,
        "vat_filings",
        include_str!("../../../migrations/V14__vat_filings.sql"),
    ),
    (
        15,
        "annual_tax",
        include_str!("../../../migrations/V15__annual_tax.sql"),
    ),
    (
        16,
        "tax_prepayments",
        include_str!("../../../migrations/V16__tax_prepayments.sql"),
    ),
    (
        17,
        "tax_credits_deductions",
        include_str!("../../../migrations/V17__tax_credits_deductions.sql"),
    ),
    (
        18,
        "spouse_months",
        include_str!("../../../migrations/V18__spouse_months.sql"),
    ),
    (
        19,
        "investment_income",
        include_str!("../../../migrations/V19__investment_income.sql"),
    ),
    (
        20,
        "fakturoid_import_log",
        include_str!("../../../migrations/V20__fakturoid_import_log.sql"),
    ),
    (
        21,
        "audit_log_expand",
        include_str!("../../../migrations/V21__audit_log_expand.sql"),
    ),
    (
        22,
        "invoice_documents",
        include_str!("../../../migrations/V22__invoice_documents.sql"),
    ),
    (
        23,
        "expense_items",
        include_str!("../../../migrations/V23__expense_items.sql"),
    ),
    (
        24,
        "backup_history",
        include_str!("../../../migrations/V24__backup_history.sql"),
    ),
];

/// Run all pending migrations on the given connection.
/// Creates the migration tracking table if it doesn't exist.
/// Detects goose's `goose_db_version` table and bridges from it.
pub fn run_migrations(conn: &Connection) -> Result<(), DbError> {
    // Create our own migration tracking table.
    conn.execute_batch(
        "CREATE TABLE IF NOT EXISTS _zfaktury_migrations (
            version INTEGER PRIMARY KEY,
            name TEXT NOT NULL,
            applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
        );",
    )?;

    // Bridge from goose: detect existing goose_db_version table and read current version.
    let goose_version = detect_goose_version(conn);

    // Determine already-applied versions from our table.
    let applied: Vec<i64> = {
        let mut stmt = conn.prepare("SELECT version FROM _zfaktury_migrations ORDER BY version")?;
        let rows = stmt.query_map([], |row| row.get(0))?;
        rows.collect::<Result<Vec<i64>, _>>()?
    };

    for &(version, name, sql) in MIGRATIONS {
        // Skip if already applied via our tracker.
        if applied.contains(&version) {
            continue;
        }
        // Skip if goose already applied this version.
        if version <= goose_version {
            // Record it in our tracker so we don't re-check next time.
            conn.execute(
                "INSERT OR IGNORE INTO _zfaktury_migrations (version, name) VALUES (?1, ?2)",
                rusqlite::params![version, name],
            )?;
            continue;
        }

        log::info!("applying migration V{version}: {name}");
        conn.execute_batch(sql).map_err(|e| {
            DbError::Migration(format!("migration V{version} ({name}) failed: {e}"))
        })?;

        conn.execute(
            "INSERT INTO _zfaktury_migrations (version, name) VALUES (?1, ?2)",
            rusqlite::params![version, name],
        )?;
    }

    Ok(())
}

/// Detect the highest version from goose's tracking table, if it exists.
/// Returns 0 if the table doesn't exist or is empty.
fn detect_goose_version(conn: &Connection) -> i64 {
    // Check if the goose_db_version table exists.
    let exists: bool = conn
        .query_row(
            "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='goose_db_version'",
            [],
            |row| row.get::<_, i64>(0),
        )
        .map(|c| c > 0)
        .unwrap_or(false);

    if !exists {
        return 0;
    }

    conn.query_row(
        "SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied = 1",
        [],
        |row| row.get(0),
    )
    .unwrap_or(0)
}

/// Returns the current (highest applied) migration version.
pub fn current_version(conn: &Connection) -> Result<i64, DbError> {
    // Ensure the table exists first.
    let exists: bool = conn
        .query_row(
            "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='_zfaktury_migrations'",
            [],
            |row| row.get::<_, i64>(0),
        )
        .map(|c| c > 0)?;

    if !exists {
        return Ok(0);
    }

    let version: i64 = conn.query_row(
        "SELECT COALESCE(MAX(version), 0) FROM _zfaktury_migrations",
        [],
        |row| row.get(0),
    )?;
    Ok(version)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::connection::open_memory;

    #[test]
    fn test_run_migrations_on_empty_db() {
        let conn = open_memory().unwrap();
        run_migrations(&conn).unwrap();

        let version = current_version(&conn).unwrap();
        assert_eq!(version, 24);

        // Verify a few tables exist.
        let table_count: i64 = conn
            .query_row(
                "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN (
                    'contacts', 'invoices', 'expenses', 'settings', 'backup_history',
                    'vat_returns', 'income_tax_returns', 'security_transactions'
                )",
                [],
                |row| row.get(0),
            )
            .unwrap();
        assert_eq!(table_count, 8);
    }

    #[test]
    fn test_migrations_idempotent() {
        let conn = open_memory().unwrap();
        run_migrations(&conn).unwrap();
        // Running again should be a no-op.
        run_migrations(&conn).unwrap();
        assert_eq!(current_version(&conn).unwrap(), 24);
    }

    #[test]
    fn test_default_categories_inserted() {
        let conn = open_memory().unwrap();
        run_migrations(&conn).unwrap();

        let count: i64 = conn
            .query_row("SELECT COUNT(*) FROM expense_categories", [], |row| {
                row.get(0)
            })
            .unwrap();
        assert_eq!(count, 16);
    }
}
