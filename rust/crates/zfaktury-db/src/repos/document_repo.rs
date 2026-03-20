use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::DocumentRepo;
use zfaktury_domain::{DomainError, ExpenseDocument};

pub struct SqliteDocumentRepo {
    conn: Mutex<Connection>,
}
impl SqliteDocumentRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}

fn scan_doc(row: &Row<'_>) -> rusqlite::Result<ExpenseDocument> {
    let created_str: String = row.get("created_at")?;
    let deleted_str: Option<String> = row.get("deleted_at")?;
    Ok(ExpenseDocument {
        id: row.get("id")?,
        expense_id: row.get("expense_id")?,
        filename: row.get("filename")?,
        content_type: row.get("content_type")?,
        storage_path: row.get("storage_path")?,
        size: row.get("size")?,
        created_at: parse_datetime_or_default(&created_str),
        deleted_at: parse_datetime_optional(deleted_str.as_deref()).unwrap_or(None),
    })
}

impl DocumentRepo for SqliteDocumentRepo {
    fn create(&self, doc: &mut ExpenseDocument) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        conn.execute("INSERT INTO expense_documents (expense_id, filename, content_type, storage_path, size, created_at) VALUES (?1,?2,?3,?4,?5,?6)",
            params![doc.expense_id, doc.filename, doc.content_type, doc.storage_path, doc.size, now])
            .map_err(|e| { log::error!("inserting expense doc: {e}"); DomainError::InvalidInput })?;
        doc.id = conn.last_insert_rowid();
        Ok(())
    }
    fn get_by_id(&self, id: i64) -> Result<ExpenseDocument, DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.query_row("SELECT id, expense_id, filename, content_type, storage_path, size, created_at, deleted_at FROM expense_documents WHERE id = ?1 AND deleted_at IS NULL", params![id], scan_doc)
            .map_err(|e| match e { rusqlite::Error::QueryReturnedNoRows => DomainError::NotFound, _ => { log::error!("querying doc: {e}"); DomainError::InvalidInput }})
    }
    fn list_by_expense_id(&self, expense_id: i64) -> Result<Vec<ExpenseDocument>, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut stmt = conn.prepare("SELECT id, expense_id, filename, content_type, storage_path, size, created_at, deleted_at FROM expense_documents WHERE expense_id = ?1 AND deleted_at IS NULL")
            .map_err(|e| { log::error!("preparing doc list: {e}"); DomainError::InvalidInput })?;
        stmt.query_map(params![expense_id], scan_doc)
            .map_err(|e| {
                log::error!("listing docs: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("scanning docs: {e}");
                DomainError::InvalidInput
            })
    }
    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        let rows = conn
            .execute(
                "UPDATE expense_documents SET deleted_at = ?1 WHERE id = ?2 AND deleted_at IS NULL",
                params![now, id],
            )
            .map_err(|e| {
                log::error!("deleting doc: {e}");
                DomainError::InvalidInput
            })?;
        if rows == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }
    fn count_by_expense_id(&self, expense_id: i64) -> Result<i64, DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.query_row(
            "SELECT COUNT(*) FROM expense_documents WHERE expense_id = ?1 AND deleted_at IS NULL",
            params![expense_id],
            |row| row.get(0),
        )
        .map_err(|e| {
            log::error!("counting docs: {e}");
            DomainError::InvalidInput
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;

    fn create_expense(conn: &Connection) -> i64 {
        let now = format_datetime(&Local::now().naive_local());
        conn.execute("INSERT INTO expenses (description, issue_date, created_at, updated_at) VALUES ('test', '2025-01-01', ?1, ?1)", params![now]).unwrap();
        conn.last_insert_rowid()
    }

    #[test]
    fn test_crud() {
        let conn = new_test_db();
        let eid = create_expense(&conn);
        let repo = SqliteDocumentRepo::new(conn);
        let mut doc = ExpenseDocument {
            id: 0,
            expense_id: eid,
            filename: "test.pdf".into(),
            content_type: "application/pdf".into(),
            storage_path: "/tmp/test.pdf".into(),
            size: 1024,
            created_at: Default::default(),
            deleted_at: None,
        };
        repo.create(&mut doc).unwrap();
        assert!(doc.id > 0);
        let fetched = repo.get_by_id(doc.id).unwrap();
        assert_eq!(fetched.filename, "test.pdf");
        assert_eq!(repo.count_by_expense_id(eid).unwrap(), 1);
        repo.delete(doc.id).unwrap();
        assert_eq!(repo.count_by_expense_id(eid).unwrap(), 0);
    }
}
