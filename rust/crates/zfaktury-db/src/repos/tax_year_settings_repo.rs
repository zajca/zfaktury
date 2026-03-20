use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::TaxYearSettingsRepo;
use zfaktury_domain::*;
pub struct SqliteTaxYearSettingsRepo {
    conn: Mutex<Connection>,
}
impl SqliteTaxYearSettingsRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}
fn scan(row: &Row<'_>) -> rusqlite::Result<TaxYearSettings> {
    let c: String = row.get("created_at")?;
    let u: String = row.get("updated_at")?;
    Ok(TaxYearSettings {
        year: row.get("year")?,
        flat_rate_percent: row.get("flat_rate_percent")?,
        created_at: parse_datetime_or_default(&c),
        updated_at: parse_datetime_or_default(&u),
    })
}
impl TaxYearSettingsRepo for SqliteTaxYearSettingsRepo {
    fn get_by_year(&self, year: i32) -> Result<TaxYearSettings, DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row(
            "SELECT * FROM tax_year_settings WHERE year=?1",
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
    fn upsert(&self, tys: &mut TaxYearSettings) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("INSERT INTO tax_year_settings (year,flat_rate_percent,created_at,updated_at) VALUES (?1,?2,?3,?3) ON CONFLICT(year) DO UPDATE SET flat_rate_percent=excluded.flat_rate_percent,updated_at=excluded.updated_at",params![tys.year,tys.flat_rate_percent,n]).map_err(|e|{log::error!("upsert: {e}");DomainError::InvalidInput})?;
        Ok(())
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;
    #[test]
    fn test_upsert_and_get() {
        let c = new_test_db();
        let r = SqliteTaxYearSettingsRepo::new(c);
        let mut t = TaxYearSettings {
            year: 2026,
            flat_rate_percent: 60,
            created_at: Default::default(),
            updated_at: Default::default(),
        };
        r.upsert(&mut t).unwrap();
        let f = r.get_by_year(2026).unwrap();
        assert_eq!(f.flat_rate_percent, 60);
    }
}
