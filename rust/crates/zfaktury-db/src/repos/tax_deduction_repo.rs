use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::TaxDeductionRepo;
use zfaktury_domain::*;
pub struct SqliteTaxDeductionRepo {
    conn: Mutex<Connection>,
}
impl SqliteTaxDeductionRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}
fn pdc(s: &str) -> DeductionCategory {
    match s {
        "mortgage" => DeductionCategory::Mortgage,
        "life_insurance" => DeductionCategory::LifeInsurance,
        "pension" => DeductionCategory::Pension,
        "donation" => DeductionCategory::Donation,
        _ => DeductionCategory::UnionDues,
    }
}
fn scan(row: &Row<'_>) -> rusqlite::Result<TaxDeduction> {
    let cat: String = row.get("category")?;
    let c: String = row.get("created_at")?;
    let u: String = row.get("updated_at")?;
    Ok(TaxDeduction {
        id: row.get("id")?,
        year: row.get("year")?,
        category: pdc(&cat),
        description: row.get("description")?,
        claimed_amount: Amount::from_halere(row.get::<_, i64>("claimed_amount")?),
        max_amount: Amount::from_halere(row.get::<_, i64>("max_amount")?),
        allowed_amount: Amount::from_halere(row.get::<_, i64>("allowed_amount")?),
        created_at: parse_datetime(&c).unwrap_or_default(),
        updated_at: parse_datetime(&u).unwrap_or_default(),
    })
}
impl TaxDeductionRepo for SqliteTaxDeductionRepo {
    fn create(&self, d: &mut TaxDeduction) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("INSERT INTO tax_deductions (year,category,description,claimed_amount,max_amount,allowed_amount,created_at,updated_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?7)",params![d.year,d.category.to_string(),d.description,d.claimed_amount.halere(),d.max_amount.halere(),d.allowed_amount.halere(),n]).map_err(|e|{log::error!("ins: {e}");DomainError::InvalidInput})?;
        d.id = c.last_insert_rowid();
        Ok(())
    }
    fn update(&self, d: &mut TaxDeduction) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("UPDATE tax_deductions SET category=?1,description=?2,claimed_amount=?3,max_amount=?4,allowed_amount=?5,updated_at=?6 WHERE id=?7",params![d.category.to_string(),d.description,d.claimed_amount.halere(),d.max_amount.halere(),d.allowed_amount.halere(),n,d.id]).map_err(|e|{log::error!("upd: {e}");DomainError::InvalidInput})?;
        Ok(())
    }
    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let r = c
            .execute("DELETE FROM tax_deductions WHERE id=?1", params![id])
            .map_err(|e| {
                log::error!("del: {e}");
                DomainError::InvalidInput
            })?;
        if r == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }
    fn get_by_id(&self, id: i64) -> Result<TaxDeduction, DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row(
            "SELECT * FROM tax_deductions WHERE id=?1",
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
    fn list_by_year(&self, year: i32) -> Result<Vec<TaxDeduction>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s = c
            .prepare("SELECT * FROM tax_deductions WHERE year=?1")
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
}
#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;
    #[test]
    fn test_create() {
        let c = new_test_db();
        let r = SqliteTaxDeductionRepo::new(c);
        let mut d = TaxDeduction {
            id: 0,
            year: 2025,
            category: DeductionCategory::Mortgage,
            description: "Test".into(),
            claimed_amount: Amount::from_halere(100000),
            max_amount: Amount::from_halere(300000_00),
            allowed_amount: Amount::from_halere(100000),
            created_at: Default::default(),
            updated_at: Default::default(),
        };
        r.create(&mut d).unwrap();
        assert!(d.id > 0);
    }
}
