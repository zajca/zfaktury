use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::FakturoidImportLogRepo;
use zfaktury_domain::*;
pub struct SqliteFakturoidImportRepo {
    conn: Mutex<Connection>,
}
impl SqliteFakturoidImportRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}
fn scan(row: &Row<'_>) -> rusqlite::Result<FakturoidImportLog> {
    let i: String = row.get("imported_at")?;
    Ok(FakturoidImportLog {
        id: row.get("id")?,
        fakturoid_entity_type: row.get("fakturoid_entity_type")?,
        fakturoid_id: row.get("fakturoid_id")?,
        local_entity_type: row.get("local_entity_type")?,
        local_id: row.get("local_id")?,
        imported_at: parse_datetime_or_default(&i),
    })
}
impl FakturoidImportLogRepo for SqliteFakturoidImportRepo {
    fn create(&self, e: &mut FakturoidImportLog) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("INSERT INTO fakturoid_import_log (fakturoid_entity_type,fakturoid_id,local_entity_type,local_id,imported_at) VALUES (?1,?2,?3,?4,?5)",params![e.fakturoid_entity_type,e.fakturoid_id,e.local_entity_type,e.local_id,n]).map_err(|e2|{log::error!("ins: {e2}");DomainError::InvalidInput})?;
        e.id = c.last_insert_rowid();
        Ok(())
    }
    fn find_by_fakturoid_id(
        &self,
        entity_type: &str,
        fakturoid_id: i64,
    ) -> Result<Option<FakturoidImportLog>, DomainError> {
        let c = self.conn.lock().unwrap();
        match c.query_row(
            "SELECT * FROM fakturoid_import_log WHERE fakturoid_entity_type=?1 AND fakturoid_id=?2",
            params![entity_type, fakturoid_id],
            scan,
        ) {
            Ok(v) => Ok(Some(v)),
            Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
            Err(e) => {
                log::error!("q: {e}");
                Err(DomainError::InvalidInput)
            }
        }
    }
    fn list_by_entity_type(
        &self,
        entity_type: &str,
    ) -> Result<Vec<FakturoidImportLog>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s = c
            .prepare("SELECT * FROM fakturoid_import_log WHERE fakturoid_entity_type=?1")
            .map_err(|e| {
                log::error!("p: {e}");
                DomainError::InvalidInput
            })?;
        s.query_map(params![entity_type], scan)
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
}
#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;
    #[test]
    fn test_create_find() {
        let c = new_test_db();
        let r = SqliteFakturoidImportRepo::new(c);
        let mut e = FakturoidImportLog {
            id: 0,
            fakturoid_entity_type: "contact".into(),
            fakturoid_id: 42,
            local_entity_type: "contact".into(),
            local_id: 1,
            imported_at: Default::default(),
        };
        r.create(&mut e).unwrap();
        let f = r.find_by_fakturoid_id("contact", 42).unwrap();
        assert!(f.is_some());
        let n = r.find_by_fakturoid_id("contact", 99).unwrap();
        assert!(n.is_none());
    }
}
