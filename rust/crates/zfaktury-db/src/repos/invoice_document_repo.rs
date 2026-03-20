use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::InvoiceDocumentRepo;
use zfaktury_domain::{DomainError, InvoiceDocument};

pub struct SqliteInvoiceDocumentRepo {
    conn: Mutex<Connection>,
}
impl SqliteInvoiceDocumentRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}

fn scan(row: &Row<'_>) -> rusqlite::Result<InvoiceDocument> {
    let c: String = row.get("created_at")?;
    let d: Option<String> = row.get("deleted_at")?;
    Ok(InvoiceDocument {
        id: row.get("id")?,
        invoice_id: row.get("invoice_id")?,
        filename: row.get("filename")?,
        content_type: row.get("content_type")?,
        storage_path: row.get("storage_path")?,
        size: row.get("size")?,
        created_at: parse_datetime_or_default(&c),
        deleted_at: parse_datetime_optional(d.as_deref()).unwrap_or(None),
    })
}

impl InvoiceDocumentRepo for SqliteInvoiceDocumentRepo {
    fn create(&self, doc: &mut InvoiceDocument) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        conn.execute("INSERT INTO invoice_documents (invoice_id,filename,content_type,storage_path,size,created_at) VALUES (?1,?2,?3,?4,?5,?6)",
            params![doc.invoice_id,doc.filename,doc.content_type,doc.storage_path,doc.size,now]).map_err(|e|{log::error!("insert inv doc: {e}");DomainError::InvalidInput})?;
        doc.id = conn.last_insert_rowid();
        Ok(())
    }
    fn get_by_id(&self, id: i64) -> Result<InvoiceDocument, DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.query_row("SELECT id,invoice_id,filename,content_type,storage_path,size,created_at,deleted_at FROM invoice_documents WHERE id=?1 AND deleted_at IS NULL",params![id],scan)
            .map_err(|e| match e { rusqlite::Error::QueryReturnedNoRows=>DomainError::NotFound, _=>{log::error!("q inv doc: {e}");DomainError::InvalidInput}})
    }
    fn list_by_invoice_id(&self, invoice_id: i64) -> Result<Vec<InvoiceDocument>, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut s = conn.prepare("SELECT id,invoice_id,filename,content_type,storage_path,size,created_at,deleted_at FROM invoice_documents WHERE invoice_id=?1 AND deleted_at IS NULL").map_err(|e|{log::error!("p inv doc: {e}");DomainError::InvalidInput})?;
        s.query_map(params![invoice_id], scan)
            .map_err(|e| {
                log::error!("l inv doc: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("s inv doc: {e}");
                DomainError::InvalidInput
            })
    }
    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        let r = conn
            .execute(
                "UPDATE invoice_documents SET deleted_at=?1 WHERE id=?2 AND deleted_at IS NULL",
                params![now, id],
            )
            .map_err(|e| {
                log::error!("d inv doc: {e}");
                DomainError::InvalidInput
            })?;
        if r == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }
    fn count_by_invoice_id(&self, invoice_id: i64) -> Result<i64, DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.query_row(
            "SELECT COUNT(*) FROM invoice_documents WHERE invoice_id=?1 AND deleted_at IS NULL",
            params![invoice_id],
            |r| r.get(0),
        )
        .map_err(|e| {
            log::error!("cnt inv doc: {e}");
            DomainError::InvalidInput
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;

    #[test]
    fn test_crud() {
        let conn = new_test_db();
        let now = format_datetime(&Local::now().naive_local());
        conn.execute("INSERT INTO contacts (type,name,country,created_at,updated_at) VALUES ('company','C','CZ',?1,?1)",params![now]).unwrap();
        let cid = conn.last_insert_rowid();
        conn.execute("INSERT INTO invoices (invoice_number,customer_id,issue_date,due_date,created_at,updated_at) VALUES ('INV1',?1,'2025-01-01','2025-01-15',?2,?2)",params![cid,now]).unwrap();
        let iid = conn.last_insert_rowid();
        let repo = SqliteInvoiceDocumentRepo::new(conn);
        let mut doc = InvoiceDocument {
            id: 0,
            invoice_id: iid,
            filename: "a.pdf".into(),
            content_type: "application/pdf".into(),
            storage_path: "/tmp/a".into(),
            size: 100,
            created_at: Default::default(),
            deleted_at: None,
        };
        repo.create(&mut doc).unwrap();
        assert!(doc.id > 0);
        assert_eq!(repo.count_by_invoice_id(iid).unwrap(), 1);
        repo.delete(doc.id).unwrap();
        assert_eq!(repo.count_by_invoice_id(iid).unwrap(), 0);
    }
}
