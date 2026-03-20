use crate::helpers::*;
#[cfg(test)]
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::StatusHistoryRepo;
use zfaktury_domain::*;

pub struct SqliteStatusHistoryRepo {
    conn: Mutex<Connection>,
}
impl SqliteStatusHistoryRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}

fn parse_status(s: &str) -> InvoiceStatus {
    match s {
        "sent" => InvoiceStatus::Sent,
        "paid" => InvoiceStatus::Paid,
        "overdue" => InvoiceStatus::Overdue,
        "cancelled" => InvoiceStatus::Cancelled,
        _ => InvoiceStatus::Draft,
    }
}

fn scan(row: &Row<'_>) -> rusqlite::Result<InvoiceStatusChange> {
    let old: String = row.get("old_status")?;
    let new: String = row.get("new_status")?;
    let at: String = row.get("changed_at")?;
    Ok(InvoiceStatusChange {
        id: row.get("id")?,
        invoice_id: row.get("invoice_id")?,
        old_status: parse_status(&old),
        new_status: parse_status(&new),
        changed_at: parse_datetime_or_default(&at),
        note: row.get("note")?,
    })
}

impl StatusHistoryRepo for SqliteStatusHistoryRepo {
    fn create(&self, c: &mut InvoiceStatusChange) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.execute("INSERT INTO invoice_status_history (invoice_id,old_status,new_status,changed_at,note) VALUES (?1,?2,?3,?4,?5)",
            params![c.invoice_id, c.old_status.to_string(), c.new_status.to_string(), format_datetime(&c.changed_at), c.note])
            .map_err(|e|{log::error!("insert status history: {e}");DomainError::InvalidInput})?;
        c.id = conn.last_insert_rowid();
        Ok(())
    }
    fn list_by_invoice_id(&self, invoice_id: i64) -> Result<Vec<InvoiceStatusChange>, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut s = conn.prepare("SELECT id,invoice_id,old_status,new_status,changed_at,note FROM invoice_status_history WHERE invoice_id=?1 ORDER BY changed_at ASC")
            .map_err(|e|{log::error!("prep status history: {e}");DomainError::InvalidInput})?;
        s.query_map(params![invoice_id], scan)
            .map_err(|e| {
                log::error!("list: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("scan: {e}");
                DomainError::InvalidInput
            })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;
    #[test]
    fn test_create_and_list() {
        let conn = new_test_db();
        let now = format_datetime(&Local::now().naive_local());
        conn.execute("INSERT INTO contacts (type,name,country,created_at,updated_at) VALUES ('company','C','CZ',?1,?1)",params![now]).unwrap();
        let cid = conn.last_insert_rowid();
        conn.execute("INSERT INTO invoices (invoice_number,customer_id,issue_date,due_date,created_at,updated_at) VALUES ('INV1',?1,'2025-01-01','2025-01-15',?2,?2)",params![cid,now]).unwrap();
        let iid = conn.last_insert_rowid();
        let repo = SqliteStatusHistoryRepo::new(conn);
        let mut change = InvoiceStatusChange {
            id: 0,
            invoice_id: iid,
            old_status: InvoiceStatus::Draft,
            new_status: InvoiceStatus::Sent,
            changed_at: Local::now().naive_local(),
            note: String::new(),
        };
        repo.create(&mut change).unwrap();
        let list = repo.list_by_invoice_id(iid).unwrap();
        assert_eq!(list.len(), 1);
        assert_eq!(list[0].new_status, InvoiceStatus::Sent);
    }
}
