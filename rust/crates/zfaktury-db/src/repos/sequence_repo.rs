use std::sync::Mutex;

use chrono::Local;
use rusqlite::{Connection, Row, params};
use zfaktury_core::repository::traits::InvoiceSequenceRepo;
use zfaktury_domain::{DomainError, InvoiceSequence};

use crate::helpers::*;

pub struct SqliteSequenceRepo {
    conn: Mutex<Connection>,
}

impl SqliteSequenceRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}

fn scan_sequence(row: &Row<'_>) -> rusqlite::Result<InvoiceSequence> {
    Ok(InvoiceSequence {
        id: row.get("id")?,
        prefix: row.get("prefix")?,
        next_number: row.get("next_number")?,
        year: row.get("year")?,
        format_pattern: row.get("format_pattern")?,
    })
}

impl InvoiceSequenceRepo for SqliteSequenceRepo {
    fn create(&self, seq: &mut InvoiceSequence) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.execute(
            "INSERT INTO invoice_sequences (prefix, next_number, year, format_pattern) VALUES (?1, ?2, ?3, ?4)",
            params![seq.prefix, seq.next_number, seq.year, seq.format_pattern],
        ).map_err(|e| { log::error!("inserting sequence: {e}"); DomainError::InvalidInput })?;
        seq.id = conn.last_insert_rowid();
        Ok(())
    }

    fn update(&self, seq: &mut InvoiceSequence) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        conn.execute(
            "UPDATE invoice_sequences SET prefix = ?1, next_number = ?2, year = ?3, format_pattern = ?4, updated_at = ?5 WHERE id = ?6 AND deleted_at IS NULL",
            params![seq.prefix, seq.next_number, seq.year, seq.format_pattern, now, seq.id],
        ).map_err(|e| { log::error!("updating sequence: {e}"); DomainError::InvalidInput })?;
        Ok(())
    }

    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        let rows = conn.execute(
            "UPDATE invoice_sequences SET deleted_at = ?1, updated_at = ?1 WHERE id = ?2 AND deleted_at IS NULL",
            params![now, id],
        ).map_err(|e| { log::error!("deleting sequence: {e}"); DomainError::InvalidInput })?;
        if rows == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }

    fn get_by_id(&self, id: i64) -> Result<InvoiceSequence, DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.query_row(
            "SELECT id, prefix, next_number, year, format_pattern FROM invoice_sequences WHERE id = ?1 AND deleted_at IS NULL",
            params![id], scan_sequence,
        ).map_err(|e| match e {
            rusqlite::Error::QueryReturnedNoRows => DomainError::NotFound,
            _ => { log::error!("querying sequence: {e}"); DomainError::InvalidInput }
        })
    }

    fn list(&self) -> Result<Vec<InvoiceSequence>, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut stmt = conn.prepare("SELECT id, prefix, next_number, year, format_pattern FROM invoice_sequences WHERE deleted_at IS NULL ORDER BY year DESC, prefix ASC")
            .map_err(|e| { log::error!("preparing sequence list: {e}"); DomainError::InvalidInput })?;
        let rows = stmt.query_map([], scan_sequence).map_err(|e| {
            log::error!("listing sequences: {e}");
            DomainError::InvalidInput
        })?;
        rows.collect::<Result<Vec<_>, _>>().map_err(|e| {
            log::error!("scanning sequences: {e}");
            DomainError::InvalidInput
        })
    }

    fn get_by_prefix_and_year(
        &self,
        prefix: &str,
        year: i32,
    ) -> Result<InvoiceSequence, DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.query_row(
            "SELECT id, prefix, next_number, year, format_pattern FROM invoice_sequences WHERE prefix = ?1 AND year = ?2 AND deleted_at IS NULL",
            params![prefix, year], scan_sequence,
        ).map_err(|e| match e {
            rusqlite::Error::QueryReturnedNoRows => DomainError::NotFound,
            _ => { log::error!("querying sequence by prefix/year: {e}"); DomainError::InvalidInput }
        })
    }

    fn count_invoices_by_sequence_id(&self, sequence_id: i64) -> Result<i64, DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.query_row(
            "SELECT COUNT(*) FROM invoices WHERE sequence_id = ?1 AND deleted_at IS NULL",
            params![sequence_id],
            |row| row.get(0),
        )
        .map_err(|e| {
            log::error!("counting invoices by sequence: {e}");
            DomainError::InvalidInput
        })
    }

    fn max_used_number(&self, sequence_id: i64) -> Result<i64, DomainError> {
        let conn = self.conn.lock().unwrap();
        let result: Option<i64> = conn.query_row(
            "SELECT MAX(CAST(SUBSTR(invoice_number, LENGTH((SELECT prefix FROM invoice_sequences WHERE id = ?1)) + 5) AS INTEGER)) FROM invoices WHERE sequence_id = ?1 AND deleted_at IS NULL",
            params![sequence_id], |row| row.get(0),
        ).map_err(|e| { log::error!("max used number: {e}"); DomainError::InvalidInput })?;
        Ok(result.unwrap_or(0))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;

    #[test]
    fn test_create_and_get() {
        let conn = new_test_db();
        let repo = SqliteSequenceRepo::new(conn);

        let mut seq = InvoiceSequence {
            id: 0,
            prefix: "FV".to_string(),
            next_number: 1,
            year: 2025,
            format_pattern: "{prefix}{year}{number:04d}".to_string(),
        };
        repo.create(&mut seq).unwrap();
        assert!(seq.id > 0);

        let fetched = repo.get_by_id(seq.id).unwrap();
        assert_eq!(fetched.prefix, "FV");
        assert_eq!(fetched.year, 2025);
    }

    #[test]
    fn test_list() {
        let conn = new_test_db();
        let repo = SqliteSequenceRepo::new(conn);

        let mut s1 = InvoiceSequence {
            id: 0,
            prefix: "FV".to_string(),
            next_number: 1,
            year: 2025,
            format_pattern: String::new(),
        };
        let mut s2 = InvoiceSequence {
            id: 0,
            prefix: "ZF".to_string(),
            next_number: 1,
            year: 2025,
            format_pattern: String::new(),
        };
        repo.create(&mut s1).unwrap();
        repo.create(&mut s2).unwrap();

        let list = repo.list().unwrap();
        assert_eq!(list.len(), 2);
    }

    #[test]
    fn test_get_by_prefix_and_year() {
        let conn = new_test_db();
        let repo = SqliteSequenceRepo::new(conn);

        let mut seq = InvoiceSequence {
            id: 0,
            prefix: "FV".to_string(),
            next_number: 1,
            year: 2025,
            format_pattern: String::new(),
        };
        repo.create(&mut seq).unwrap();

        let found = repo.get_by_prefix_and_year("FV", 2025).unwrap();
        assert_eq!(found.id, seq.id);

        let not_found = repo.get_by_prefix_and_year("FV", 2024);
        assert!(matches!(not_found, Err(DomainError::NotFound)));
    }
}
