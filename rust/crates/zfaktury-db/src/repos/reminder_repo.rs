use crate::helpers::*;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::ReminderRepo;
use zfaktury_domain::{DomainError, PaymentReminder};

pub struct SqliteReminderRepo {
    conn: Mutex<Connection>,
}
impl SqliteReminderRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}

fn scan(row: &Row<'_>) -> rusqlite::Result<PaymentReminder> {
    let sent: String = row.get("sent_at")?;
    let created: String = row.get("created_at")?;
    Ok(PaymentReminder {
        id: row.get("id")?,
        invoice_id: row.get("invoice_id")?,
        reminder_number: row.get("reminder_number")?,
        sent_at: parse_datetime(&sent).unwrap_or_default(),
        sent_to: row.get("sent_to")?,
        subject: row.get("subject")?,
        body_preview: row.get("body_preview")?,
        created_at: parse_datetime(&created).unwrap_or_default(),
    })
}

impl ReminderRepo for SqliteReminderRepo {
    fn create(&self, r: &mut PaymentReminder) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.execute("INSERT INTO payment_reminders (invoice_id,reminder_number,sent_at,sent_to,subject,body_preview,created_at) VALUES (?1,?2,?3,?4,?5,?6,?7)",
            params![r.invoice_id, r.reminder_number, format_datetime(&r.sent_at), r.sent_to, r.subject, r.body_preview, format_datetime(&r.created_at)])
            .map_err(|e|{log::error!("insert reminder: {e}");DomainError::InvalidInput})?;
        r.id = conn.last_insert_rowid();
        Ok(())
    }
    fn list_by_invoice_id(&self, invoice_id: i64) -> Result<Vec<PaymentReminder>, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut s = conn.prepare("SELECT id,invoice_id,reminder_number,sent_at,sent_to,subject,body_preview,created_at FROM payment_reminders WHERE invoice_id=?1 ORDER BY reminder_number ASC")
            .map_err(|e|{log::error!("prep: {e}");DomainError::InvalidInput})?;
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
    fn count_by_invoice_id(&self, invoice_id: i64) -> Result<i64, DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.query_row(
            "SELECT COUNT(*) FROM payment_reminders WHERE invoice_id=?1",
            params![invoice_id],
            |r| r.get(0),
        )
        .map_err(|e| {
            log::error!("count: {e}");
            DomainError::InvalidInput
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;
    use chrono::Local;
    #[test]
    fn test_create_and_count() {
        let conn = new_test_db();
        let now = format_datetime(&Local::now().naive_local());
        conn.execute("INSERT INTO contacts (type,name,country,created_at,updated_at) VALUES ('company','C','CZ',?1,?1)",params![now]).unwrap();
        let cid = conn.last_insert_rowid();
        conn.execute("INSERT INTO invoices (invoice_number,customer_id,issue_date,due_date,created_at,updated_at) VALUES ('INV1',?1,'2025-01-01','2025-01-15',?2,?2)",params![cid,now]).unwrap();
        let iid = conn.last_insert_rowid();
        let repo = SqliteReminderRepo::new(conn);
        let mut r = PaymentReminder {
            id: 0,
            invoice_id: iid,
            reminder_number: 1,
            sent_at: Local::now().naive_local(),
            sent_to: "test@example.com".into(),
            subject: "Reminder".into(),
            body_preview: "Please pay".into(),
            created_at: Local::now().naive_local(),
        };
        repo.create(&mut r).unwrap();
        assert_eq!(repo.count_by_invoice_id(iid).unwrap(), 1);
    }
}
