use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::TaxDeductionDocumentRepo;
use zfaktury_domain::*;
pub struct SqliteTaxDeductionDocumentRepo {
    conn: Mutex<Connection>,
}
impl SqliteTaxDeductionDocumentRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}
fn scan(row: &Row<'_>) -> rusqlite::Result<TaxDeductionDocument> {
    let c: String = row.get("created_at")?;
    let d: Option<String> = row.get("deleted_at")?;
    Ok(TaxDeductionDocument {
        id: row.get("id")?,
        tax_deduction_id: row.get("tax_deduction_id")?,
        filename: row.get("filename")?,
        content_type: row.get("content_type")?,
        storage_path: row.get("storage_path")?,
        size: row.get("size")?,
        extracted_amount: Amount::from_halere(row.get::<_, i64>("extracted_amount")?),
        confidence: row.get("confidence")?,
        created_at: parse_datetime(&c).unwrap_or_default(),
        deleted_at: parse_datetime_optional(d.as_deref()).unwrap_or(None),
    })
}
impl TaxDeductionDocumentRepo for SqliteTaxDeductionDocumentRepo {
    fn create(&self, doc: &mut TaxDeductionDocument) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("INSERT INTO tax_deduction_documents (tax_deduction_id,filename,content_type,storage_path,size,extracted_amount,confidence,created_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?8)",params![doc.tax_deduction_id,doc.filename,doc.content_type,doc.storage_path,doc.size,doc.extracted_amount.halere(),doc.confidence,n]).map_err(|e|{log::error!("ins: {e}");DomainError::InvalidInput})?;
        doc.id = c.last_insert_rowid();
        Ok(())
    }
    fn get_by_id(&self, id: i64) -> Result<TaxDeductionDocument, DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row(
            "SELECT * FROM tax_deduction_documents WHERE id=?1 AND deleted_at IS NULL",
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
    fn list_by_deduction_id(
        &self,
        deduction_id: i64,
    ) -> Result<Vec<TaxDeductionDocument>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s=c.prepare("SELECT * FROM tax_deduction_documents WHERE tax_deduction_id=?1 AND deleted_at IS NULL").map_err(|e|{log::error!("p: {e}");DomainError::InvalidInput})?;
        s.query_map(params![deduction_id], scan)
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
        let n = format_datetime(&Local::now().naive_local());
        let r=c.execute("UPDATE tax_deduction_documents SET deleted_at=?1 WHERE id=?2 AND deleted_at IS NULL",params![n,id]).map_err(|e|{log::error!("del: {e}");DomainError::InvalidInput})?;
        if r == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }
    fn update_extraction(
        &self,
        id: i64,
        amount: Amount,
        confidence: f64,
    ) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        c.execute(
            "UPDATE tax_deduction_documents SET extracted_amount=?1,confidence=?2 WHERE id=?3",
            params![amount.halere(), confidence, id],
        )
        .map_err(|e| {
            log::error!("upd: {e}");
            DomainError::InvalidInput
        })?;
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
        let n = format_datetime(&Local::now().naive_local());
        c.execute("INSERT INTO tax_deductions (year,category,description,created_at,updated_at) VALUES (2025,'mortgage','Test',?1,?1)",params![n]).unwrap();
        let did = c.last_insert_rowid();
        let r = SqliteTaxDeductionDocumentRepo::new(c);
        let mut doc = TaxDeductionDocument {
            id: 0,
            tax_deduction_id: did,
            filename: "a.pdf".into(),
            content_type: "application/pdf".into(),
            storage_path: "/tmp/a".into(),
            size: 100,
            extracted_amount: Amount::ZERO,
            confidence: 0.0,
            created_at: Default::default(),
            deleted_at: None,
        };
        r.create(&mut doc).unwrap();
        assert!(doc.id > 0);
    }
}
