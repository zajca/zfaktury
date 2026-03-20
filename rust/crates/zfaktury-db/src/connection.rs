use std::path::Path;

use rusqlite::Connection;

use crate::error::DbError;

/// Open a SQLite connection with recommended pragmas.
/// Sets WAL mode, enables foreign keys, and configures busy timeout.
pub fn open_connection(path: &Path) -> Result<Connection, DbError> {
    let conn = Connection::open(path)?;
    configure_connection(&conn)?;
    Ok(conn)
}

/// Open an in-memory SQLite connection with recommended pragmas.
pub fn open_memory() -> Result<Connection, DbError> {
    let conn = Connection::open_in_memory()?;
    configure_connection(&conn)?;
    Ok(conn)
}

fn configure_connection(conn: &Connection) -> Result<(), DbError> {
    conn.execute_batch(
        "PRAGMA journal_mode = WAL;
         PRAGMA foreign_keys = ON;
         PRAGMA busy_timeout = 5000;",
    )?;
    Ok(())
}
