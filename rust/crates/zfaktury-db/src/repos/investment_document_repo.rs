use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::InvestmentDocumentRepo;
use zfaktury_domain::*;
pub struct SqliteInvestmentDocumentRepo {
    conn: Mutex<Connection>,
}
impl SqliteInvestmentDocumentRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}
fn pp(s: &str) -> Platform {
    match s {
        "portu" => Platform::Portu,
        "zonky" => Platform::Zonky,
        "trading212" => Platform::Trading212,
        "revolut" => Platform::Revolut,
        _ => Platform::Other,
    }
}
fn pes(s: &str) -> ExtractionStatus {
    match s {
        "extracted" => ExtractionStatus::Extracted,
        "failed" => ExtractionStatus::Failed,
        _ => ExtractionStatus::Pending,
    }
}
fn scan(row: &Row<'_>) -> rusqlite::Result<InvestmentDocument> {
    let p: String = row.get("platform")?;
    let es: String = row.get("extraction_status")?;
    let c: String = row.get("created_at")?;
    let u: String = row.get("updated_at")?;
    Ok(InvestmentDocument {
        id: row.get("id")?,
        year: row.get("year")?,
        platform: pp(&p),
        filename: row.get("filename")?,
        content_type: row.get("content_type")?,
        storage_path: row.get("storage_path")?,
        size: row.get("size")?,
        extraction_status: pes(&es),
        extraction_error: row.get("extraction_error")?,
        created_at: parse_datetime(&c).unwrap_or_default(),
        updated_at: parse_datetime(&u).unwrap_or_default(),
    })
}
impl InvestmentDocumentRepo for SqliteInvestmentDocumentRepo {
    fn create(&self, doc: &mut InvestmentDocument) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("INSERT INTO investment_documents (year,platform,filename,content_type,storage_path,size,extraction_status,extraction_error,created_at,updated_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?9)",params![doc.year,doc.platform.to_string(),doc.filename,doc.content_type,doc.storage_path,doc.size,doc.extraction_status.to_string(),doc.extraction_error,n]).map_err(|e|{log::error!("ins: {e}");DomainError::InvalidInput})?;
        doc.id = c.last_insert_rowid();
        Ok(())
    }
    fn get_by_id(&self, id: i64) -> Result<InvestmentDocument, DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row(
            "SELECT * FROM investment_documents WHERE id=?1",
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
    fn list_by_year(&self, year: i32) -> Result<Vec<InvestmentDocument>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s = c
            .prepare("SELECT * FROM investment_documents WHERE year=?1")
            .map_err(|e| {
                log::error!("p: {e}");
                DomainError::InvalidInput
            })?;
        s.query_map(params![year], scan)
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
            .execute("DELETE FROM investment_documents WHERE id=?1", params![id])
            .map_err(|e| {
                log::error!("del: {e}");
                DomainError::InvalidInput
            })?;
        if r == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }
    fn update_extraction(
        &self,
        id: i64,
        status: &str,
        extraction_error: &str,
    ) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("UPDATE investment_documents SET extraction_status=?1,extraction_error=?2,updated_at=?3 WHERE id=?4",params![status,extraction_error,n,id]).map_err(|e|{log::error!("upd: {e}");DomainError::InvalidInput})?;
        Ok(())
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;
    #[test]
    fn test_create() {
        let c = new_test_db();
        let r = SqliteInvestmentDocumentRepo::new(c);
        let mut d = InvestmentDocument {
            id: 0,
            year: 2025,
            platform: Platform::Portu,
            filename: "a.pdf".into(),
            content_type: "application/pdf".into(),
            storage_path: "/tmp".into(),
            size: 100,
            extraction_status: ExtractionStatus::Pending,
            extraction_error: String::new(),
            created_at: Default::default(),
            updated_at: Default::default(),
        };
        r.create(&mut d).unwrap();
        assert!(d.id > 0);
    }
}
