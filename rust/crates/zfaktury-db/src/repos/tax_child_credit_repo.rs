use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::TaxChildCreditRepo;
use zfaktury_domain::*;
pub struct SqliteTaxChildCreditRepo {
    conn: Mutex<Connection>,
}
impl SqliteTaxChildCreditRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}
fn scan(row: &Row<'_>) -> rusqlite::Result<TaxChildCredit> {
    let c: String = row.get("created_at")?;
    let u: String = row.get("updated_at")?;
    Ok(TaxChildCredit {
        id: row.get("id")?,
        year: row.get("year")?,
        child_name: row.get("child_name")?,
        birth_number: row.get("birth_number")?,
        child_order: row.get("child_order")?,
        months_claimed: row.get("months_claimed")?,
        ztp: row.get::<_, i32>("ztp")? != 0,
        credit_amount: Amount::from_halere(row.get::<_, i64>("credit_amount")?),
        created_at: parse_datetime(&c).unwrap_or_default(),
        updated_at: parse_datetime(&u).unwrap_or_default(),
    })
}
impl TaxChildCreditRepo for SqliteTaxChildCreditRepo {
    fn create(&self, cr: &mut TaxChildCredit) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("INSERT INTO tax_child_credits (year,child_name,birth_number,child_order,months_claimed,ztp,credit_amount,created_at,updated_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?8)",params![cr.year,cr.child_name,cr.birth_number,cr.child_order,cr.months_claimed,cr.ztp as i32,cr.credit_amount.halere(),n]).map_err(|e|{log::error!("ins: {e}");DomainError::InvalidInput})?;
        cr.id = c.last_insert_rowid();
        Ok(())
    }
    fn update(&self, cr: &mut TaxChildCredit) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("UPDATE tax_child_credits SET child_name=?1,birth_number=?2,child_order=?3,months_claimed=?4,ztp=?5,credit_amount=?6,updated_at=?7 WHERE id=?8",params![cr.child_name,cr.birth_number,cr.child_order,cr.months_claimed,cr.ztp as i32,cr.credit_amount.halere(),n,cr.id]).map_err(|e|{log::error!("upd: {e}");DomainError::InvalidInput})?;
        Ok(())
    }
    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let r = c
            .execute("DELETE FROM tax_child_credits WHERE id=?1", params![id])
            .map_err(|e| {
                log::error!("del: {e}");
                DomainError::InvalidInput
            })?;
        if r == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }
    fn list_by_year(&self, year: i32) -> Result<Vec<TaxChildCredit>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s = c
            .prepare("SELECT * FROM tax_child_credits WHERE year=?1 ORDER BY child_order")
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
    fn test_create_list() {
        let c = new_test_db();
        let r = SqliteTaxChildCreditRepo::new(c);
        let mut cr = TaxChildCredit {
            id: 0,
            year: 2025,
            child_name: "Child".into(),
            birth_number: "".into(),
            child_order: 1,
            months_claimed: 12,
            ztp: false,
            credit_amount: Amount::from_halere(15204_00),
            created_at: Default::default(),
            updated_at: Default::default(),
        };
        r.create(&mut cr).unwrap();
        let l = r.list_by_year(2025).unwrap();
        assert_eq!(l.len(), 1);
    }
}
