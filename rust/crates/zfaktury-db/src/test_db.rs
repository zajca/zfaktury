use rusqlite::Connection;

use crate::connection::open_memory;
use crate::migrate::run_migrations;

/// Create an in-memory SQLite database with all migrations applied.
/// Intended for use in integration tests.
pub fn new_test_db() -> Connection {
    let conn = open_memory().expect("failed to open in-memory database");
    run_migrations(&conn).expect("failed to run migrations");
    conn
}
