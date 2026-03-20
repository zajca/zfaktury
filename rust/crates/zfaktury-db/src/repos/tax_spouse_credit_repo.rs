use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::TaxSpouseCreditRepo;
use zfaktury_domain::*;
pub struct SqliteTaxSpouseCreditRepo {
    conn: Mutex<Connection>,
}
impl SqliteTaxSpouseCreditRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}
fn scan(row: &Row<'_>) -> rusqlite::Result<TaxSpouseCredit> {
    let c: String = row.get("created_at")?;
    let u: String = row.get("updated_at")?;
    Ok(TaxSpouseCredit {
        id: row.get("id")?,
        year: row.get("year")?,
        spouse_name: row.get("spouse_name")?,
        spouse_birth_number: row.get("spouse_birth_number")?,
        spouse_income: Amount::from_halere(row.get::<_, i64>("spouse_income")?),
        spouse_ztp: row.get::<_, i32>("spouse_ztp")? != 0,
        months_claimed: row.get("months_claimed")?,
        credit_amount: Amount::from_halere(row.get::<_, i64>("credit_amount")?),
        created_at: parse_datetime(&c).unwrap_or_default(),
        updated_at: parse_datetime(&u).unwrap_or_default(),
    })
}
impl TaxSpouseCreditRepo for SqliteTaxSpouseCreditRepo {
    fn upsert(&self, cr: &mut TaxSpouseCredit) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("INSERT INTO tax_spouse_credits (year,spouse_name,spouse_birth_number,spouse_income,spouse_ztp,months_claimed,credit_amount,created_at,updated_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?8) ON CONFLICT(year) DO UPDATE SET spouse_name=excluded.spouse_name,spouse_birth_number=excluded.spouse_birth_number,spouse_income=excluded.spouse_income,spouse_ztp=excluded.spouse_ztp,months_claimed=excluded.months_claimed,credit_amount=excluded.credit_amount,updated_at=excluded.updated_at",params![cr.year,cr.spouse_name,cr.spouse_birth_number,cr.spouse_income.halere(),cr.spouse_ztp as i32,cr.months_claimed,cr.credit_amount.halere(),n]).map_err(|e|{log::error!("upsert: {e}");DomainError::InvalidInput})?;
        if cr.id == 0 {
            cr.id = c.last_insert_rowid();
        }
        Ok(())
    }
    fn get_by_year(&self, year: i32) -> Result<TaxSpouseCredit, DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row(
            "SELECT * FROM tax_spouse_credits WHERE year=?1",
            params![year],
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
    fn delete_by_year(&self, year: i32) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        c.execute(
            "DELETE FROM tax_spouse_credits WHERE year=?1",
            params![year],
        )
        .map_err(|e| {
            log::error!("del: {e}");
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
    fn test_upsert() {
        let c = new_test_db();
        let r = SqliteTaxSpouseCreditRepo::new(c);
        let mut cr = TaxSpouseCredit {
            id: 0,
            year: 2025,
            spouse_name: "Test".into(),
            spouse_birth_number: "".into(),
            spouse_income: Amount::ZERO,
            spouse_ztp: false,
            months_claimed: 12,
            credit_amount: Amount::from_halere(24840_00),
            created_at: Default::default(),
            updated_at: Default::default(),
        };
        r.upsert(&mut cr).unwrap();
        let f = r.get_by_year(2025).unwrap();
        assert_eq!(f.spouse_name, "Test");
    }
}
