use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::AuditLogRepo;
use zfaktury_domain::*;
pub struct SqliteAuditLogRepo {
    conn: Mutex<Connection>,
}
impl SqliteAuditLogRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}
fn scan(row: &Row<'_>) -> rusqlite::Result<AuditLogEntry> {
    let c: String = row.get("created_at")?;
    Ok(AuditLogEntry {
        id: row.get("id")?,
        entity_type: row.get("entity_type")?,
        entity_id: row.get("entity_id")?,
        action: row.get("action")?,
        old_values: row
            .get::<_, Option<String>>("old_values")?
            .unwrap_or_default(),
        new_values: row
            .get::<_, Option<String>>("new_values")?
            .unwrap_or_default(),
        created_at: parse_datetime_or_default(&c),
    })
}
impl AuditLogRepo for SqliteAuditLogRepo {
    fn create(&self, e: &mut AuditLogEntry) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("INSERT INTO audit_log (entity_type,entity_id,action,old_values,new_values,created_at) VALUES (?1,?2,?3,?4,?5,?6)",params![e.entity_type,e.entity_id,e.action,e.old_values,e.new_values,n]).map_err(|e2|{log::error!("ins: {e2}");DomainError::InvalidInput})?;
        e.id = c.last_insert_rowid();
        Ok(())
    }
    fn list_by_entity(
        &self,
        entity_type: &str,
        entity_id: i64,
    ) -> Result<Vec<AuditLogEntry>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s=c.prepare("SELECT * FROM audit_log WHERE entity_type=?1 AND entity_id=?2 ORDER BY created_at DESC").map_err(|e|{log::error!("p: {e}");DomainError::InvalidInput})?;
        s.query_map(params![entity_type, entity_id], scan)
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
    fn list(&self, filter: &AuditLogFilter) -> Result<(Vec<AuditLogEntry>, i64), DomainError> {
        let c = self.conn.lock().unwrap();
        let mut w = String::from("1=1");
        let mut pv: Vec<Box<dyn rusqlite::types::ToSql>> = Vec::new();
        if !filter.entity_type.is_empty() {
            let i = pv.len() + 1;
            w.push_str(&format!(" AND entity_type=?{i}"));
            pv.push(Box::new(filter.entity_type.clone()));
        }
        if let Some(eid) = filter.entity_id {
            let i = pv.len() + 1;
            w.push_str(&format!(" AND entity_id=?{i}"));
            pv.push(Box::new(eid));
        }
        if !filter.action.is_empty() {
            let i = pv.len() + 1;
            w.push_str(&format!(" AND action=?{i}"));
            pv.push(Box::new(filter.action.clone()));
        }
        let pr: Vec<&dyn rusqlite::types::ToSql> = pv.iter().map(|p| p.as_ref()).collect();
        let total: i64 = c
            .query_row(
                &format!("SELECT COUNT(*) FROM audit_log WHERE {w}"),
                pr.as_slice(),
                |r| r.get(0),
            )
            .map_err(|e| {
                log::error!("cnt: {e}");
                DomainError::InvalidInput
            })?;
        let mut q = format!("SELECT * FROM audit_log WHERE {w} ORDER BY created_at DESC");
        if filter.limit > 0 {
            let next = pv.len() + 1;
            q.push_str(&format!(" LIMIT ?{} OFFSET ?{}", next, next + 1));
            pv.push(Box::new(filter.limit as i64));
            pv.push(Box::new(filter.offset as i64));
        }
        let pr2: Vec<&dyn rusqlite::types::ToSql> = pv.iter().map(|p| p.as_ref()).collect();
        let mut st = c.prepare(&q).map_err(|e| {
            log::error!("p: {e}");
            DomainError::InvalidInput
        })?;
        let list = st
            .query_map(pr2.as_slice(), scan)
            .map_err(|e| {
                log::error!("l: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("s: {e}");
                DomainError::InvalidInput
            })?;
        Ok((list, total))
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;
    #[test]
    fn test_create_list() {
        let c = new_test_db();
        let r = SqliteAuditLogRepo::new(c);
        let mut e = AuditLogEntry {
            id: 0,
            entity_type: "contact".into(),
            entity_id: 1,
            action: "create".into(),
            old_values: String::new(),
            new_values: "{}".into(),
            created_at: Default::default(),
        };
        r.create(&mut e).unwrap();
        let l = r.list_by_entity("contact", 1).unwrap();
        assert_eq!(l.len(), 1);
    }
}
