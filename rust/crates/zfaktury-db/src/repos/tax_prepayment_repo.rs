use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::TaxPrepaymentRepo;
use zfaktury_domain::*;
pub struct SqliteTaxPrepaymentRepo {
    conn: Mutex<Connection>,
}
impl SqliteTaxPrepaymentRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}
fn scan(row: &Row<'_>) -> rusqlite::Result<TaxPrepayment> {
    Ok(TaxPrepayment {
        year: row.get("year")?,
        month: row.get("month")?,
        tax_amount: Amount::from_halere(row.get::<_, i64>("tax_amount")?),
        social_amount: Amount::from_halere(row.get::<_, i64>("social_amount")?),
        health_amount: Amount::from_halere(row.get::<_, i64>("health_amount")?),
    })
}
impl TaxPrepaymentRepo for SqliteTaxPrepaymentRepo {
    fn list_by_year(&self, year: i32) -> Result<Vec<TaxPrepayment>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s = c
            .prepare("SELECT * FROM tax_prepayments WHERE year=?1 ORDER BY month")
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
    fn upsert_all(&self, year: i32, prepayments: &[TaxPrepayment]) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let tx = c.unchecked_transaction().map_err(|e| {
            log::error!("tx: {e}");
            DomainError::InvalidInput
        })?;
        tx.execute("DELETE FROM tax_prepayments WHERE year=?1", params![year])
            .map_err(|e| {
                log::error!("del: {e}");
                DomainError::InvalidInput
            })?;
        for p in prepayments {
            tx.execute("INSERT INTO tax_prepayments (year,month,tax_amount,social_amount,health_amount) VALUES (?1,?2,?3,?4,?5)",params![year,p.month,p.tax_amount.halere(),p.social_amount.halere(),p.health_amount.halere()]).map_err(|e|{log::error!("ins: {e}");DomainError::InvalidInput})?;
        }
        tx.commit().map_err(|e| {
            log::error!("commit: {e}");
            DomainError::InvalidInput
        })?;
        Ok(())
    }
    fn sum_by_year(&self, year: i32) -> Result<(Amount, Amount, Amount), DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row("SELECT COALESCE(SUM(tax_amount),0),COALESCE(SUM(social_amount),0),COALESCE(SUM(health_amount),0) FROM tax_prepayments WHERE year=?1",params![year],|r|Ok((Amount::from_halere(r.get::<_,i64>(0)?),Amount::from_halere(r.get::<_,i64>(1)?),Amount::from_halere(r.get::<_,i64>(2)?)))).map_err(|e|{log::error!("sum: {e}");DomainError::InvalidInput})
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;
    #[test]
    fn test_upsert_and_sum() {
        let c = new_test_db();
        let r = SqliteTaxPrepaymentRepo::new(c);
        let pp = vec![TaxPrepayment {
            year: 2025,
            month: 1,
            tax_amount: Amount::from_halere(1000),
            social_amount: Amount::from_halere(2000),
            health_amount: Amount::from_halere(3000),
        }];
        r.upsert_all(2025, &pp).unwrap();
        let (t, s, h) = r.sum_by_year(2025).unwrap();
        assert_eq!(t, Amount::from_halere(1000));
        assert_eq!(s, Amount::from_halere(2000));
        assert_eq!(h, Amount::from_halere(3000));
    }
}
