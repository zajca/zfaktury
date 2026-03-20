use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::BackupHistoryRepo;
use zfaktury_domain::*;
pub struct SqliteBackupRepo {
    conn: Mutex<Connection>,
}
impl SqliteBackupRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}
fn pbs(s: &str) -> BackupStatus {
    match s {
        "completed" => BackupStatus::Completed,
        "failed" => BackupStatus::Failed,
        _ => BackupStatus::Running,
    }
}
fn pbt(s: &str) -> BackupTrigger {
    match s {
        "scheduled" => BackupTrigger::Scheduled,
        "cli" => BackupTrigger::CLI,
        _ => BackupTrigger::Manual,
    }
}
fn scan(row: &Row<'_>) -> rusqlite::Result<BackupRecord> {
    let st: String = row.get("status")?;
    let tr: String = row.get("trigger")?;
    let c: String = row.get("created_at")?;
    let ca: Option<String> = row.get("completed_at")?;
    Ok(BackupRecord {
        id: row.get("id")?,
        filename: row.get("filename")?,
        status: pbs(&st),
        trigger: pbt(&tr),
        destination: row.get("destination")?,
        size_bytes: row.get("size_bytes")?,
        file_count: row.get("file_count")?,
        db_migration_version: row.get("db_migration_version")?,
        duration_ms: row.get("duration_ms")?,
        error_message: row.get("error_message")?,
        created_at: parse_datetime_or_default(&c),
        completed_at: parse_datetime_optional(ca.as_deref()).unwrap_or(None),
    })
}
impl BackupHistoryRepo for SqliteBackupRepo {
    fn create(&self, r: &mut BackupRecord) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("INSERT INTO backup_history (filename,status,trigger,destination,size_bytes,file_count,db_migration_version,duration_ms,error_message,created_at,completed_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11)",params![r.filename,r.status.to_string(),r.trigger.to_string(),r.destination,r.size_bytes,r.file_count,r.db_migration_version,r.duration_ms,r.error_message,n,format_datetime_opt(&r.completed_at)]).map_err(|e|{log::error!("ins: {e}");DomainError::InvalidInput})?;
        r.id = c.last_insert_rowid();
        Ok(())
    }
    fn update(&self, r: &mut BackupRecord) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        c.execute("UPDATE backup_history SET status=?1,size_bytes=?2,file_count=?3,duration_ms=?4,error_message=?5,completed_at=?6 WHERE id=?7",params![r.status.to_string(),r.size_bytes,r.file_count,r.duration_ms,r.error_message,format_datetime_opt(&r.completed_at),r.id]).map_err(|e|{log::error!("upd: {e}");DomainError::InvalidInput})?;
        Ok(())
    }
    fn get_by_id(&self, id: i64) -> Result<BackupRecord, DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row(
            "SELECT * FROM backup_history WHERE id=?1",
            params![id],
            scan,
        )
        .map_err(|e| match e {
            rusqlite::Error::QueryReturnedNoRows => DomainError::NotFound,
            _ => {
                log::error!("q: {e}");
                DomainError::InvalidInput
            }
        })
    }
    fn list(&self) -> Result<Vec<BackupRecord>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s = c
            .prepare("SELECT * FROM backup_history ORDER BY created_at DESC")
            .map_err(|e| {
                log::error!("p: {e}");
                DomainError::InvalidInput
            })?;
        s.query_map([], scan)
            .map_err(|e| {
                log::error!("l: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("s: {e}");
                DomainError::InvalidInput
            })
    }
    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let r = c
            .execute("DELETE FROM backup_history WHERE id=?1", params![id])
            .map_err(|e| {
                log::error!("del: {e}");
                DomainError::InvalidInput
            })?;
        if r == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;
    #[test]
    fn test_create_list() {
        let c = new_test_db();
        let r = SqliteBackupRepo::new(c);
        let mut b = BackupRecord {
            id: 0,
            filename: "backup.zip".into(),
            status: BackupStatus::Running,
            trigger: BackupTrigger::Manual,
            destination: "/tmp".into(),
            size_bytes: 0,
            file_count: 0,
            db_migration_version: 24,
            duration_ms: 0,
            error_message: String::new(),
            created_at: Default::default(),
            completed_at: None,
        };
        r.create(&mut b).unwrap();
        assert!(b.id > 0);
        let l = r.list().unwrap();
        assert_eq!(l.len(), 1);
    }
}
