use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::TaxPersonalCreditsRepo;
use zfaktury_domain::*;
pub struct SqliteTaxPersonalCreditsRepo {
    conn: Mutex<Connection>,
}
impl SqliteTaxPersonalCreditsRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}
fn scan(row: &Row<'_>) -> rusqlite::Result<TaxPersonalCredits> {
    let c: String = row.get("created_at")?;
    let u: String = row.get("updated_at")?;
    Ok(TaxPersonalCredits {
        year: row.get("year")?,
        is_student: row.get::<_, i32>("is_student")? != 0,
        student_months: row.get("student_months")?,
        disability_level: row.get("disability_level")?,
        credit_student: Amount::from_halere(row.get::<_, i64>("credit_student")?),
        credit_disability: Amount::from_halere(row.get::<_, i64>("credit_disability")?),
        created_at: parse_datetime(&c).unwrap_or_default(),
        updated_at: parse_datetime(&u).unwrap_or_default(),
    })
}
impl TaxPersonalCreditsRepo for SqliteTaxPersonalCreditsRepo {
    fn upsert(&self, cr: &mut TaxPersonalCredits) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("INSERT INTO tax_personal_credits (year,is_student,student_months,disability_level,credit_student,credit_disability,created_at,updated_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?7) ON CONFLICT(year) DO UPDATE SET is_student=excluded.is_student,student_months=excluded.student_months,disability_level=excluded.disability_level,credit_student=excluded.credit_student,credit_disability=excluded.credit_disability,updated_at=excluded.updated_at",params![cr.year,cr.is_student as i32,cr.student_months,cr.disability_level,cr.credit_student.halere(),cr.credit_disability.halere(),n]).map_err(|e|{log::error!("upsert: {e}");DomainError::InvalidInput})?;
        Ok(())
    }
    fn get_by_year(&self, year: i32) -> Result<TaxPersonalCredits, DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row(
            "SELECT * FROM tax_personal_credits WHERE year=?1",
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
}
#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;
    #[test]
    fn test_upsert() {
        let c = new_test_db();
        let r = SqliteTaxPersonalCreditsRepo::new(c);
        let mut cr = TaxPersonalCredits {
            year: 2025,
            is_student: false,
            student_months: 0,
            disability_level: 0,
            credit_student: Amount::ZERO,
            credit_disability: Amount::ZERO,
            created_at: Default::default(),
            updated_at: Default::default(),
        };
        r.upsert(&mut cr).unwrap();
        let f = r.get_by_year(2025).unwrap();
        assert_eq!(f.year, 2025);
    }
}
