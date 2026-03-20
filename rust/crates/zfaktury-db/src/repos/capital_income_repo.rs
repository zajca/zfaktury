use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::CapitalIncomeRepo;
use zfaktury_domain::*;
pub struct SqliteCapitalIncomeRepo {
    conn: Mutex<Connection>,
}
impl SqliteCapitalIncomeRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}
fn pcc(s: &str) -> CapitalCategory {
    match s {
        "dividend_cz" => CapitalCategory::DividendCZ,
        "dividend_foreign" => CapitalCategory::DividendForeign,
        "interest" => CapitalCategory::Interest,
        "coupon" => CapitalCategory::Coupon,
        "fund_distribution" => CapitalCategory::FundDistribution,
        _ => CapitalCategory::Other,
    }
}
fn scan(row: &Row<'_>) -> rusqlite::Result<CapitalIncomeEntry> {
    let cat: String = row.get("category")?;
    let id_str: String = row.get("income_date")?;
    let c: String = row.get("created_at")?;
    let u: String = row.get("updated_at")?;
    Ok(CapitalIncomeEntry {
        id: row.get("id")?,
        year: row.get("year")?,
        document_id: row.get("document_id")?,
        category: pcc(&cat),
        description: row.get("description")?,
        income_date: parse_date(&id_str).unwrap_or_default(),
        gross_amount: Amount::from_halere(row.get::<_, i64>("gross_amount")?),
        withheld_tax_cz: Amount::from_halere(row.get::<_, i64>("withheld_tax_cz")?),
        withheld_tax_foreign: Amount::from_halere(row.get::<_, i64>("withheld_tax_foreign")?),
        country_code: row.get("country_code")?,
        needs_declaring: row.get::<_, i32>("needs_declaring")? != 0,
        net_amount: Amount::from_halere(row.get::<_, i64>("net_amount")?),
        created_at: parse_datetime(&c).unwrap_or_default(),
        updated_at: parse_datetime(&u).unwrap_or_default(),
    })
}
impl CapitalIncomeRepo for SqliteCapitalIncomeRepo {
    fn create(&self, e: &mut CapitalIncomeEntry) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("INSERT INTO capital_income_entries (year,document_id,category,description,income_date,gross_amount,withheld_tax_cz,withheld_tax_foreign,country_code,needs_declaring,net_amount,created_at,updated_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11,?12,?12)",params![e.year,e.document_id,e.category.to_string(),e.description,format_date(&e.income_date),e.gross_amount.halere(),e.withheld_tax_cz.halere(),e.withheld_tax_foreign.halere(),e.country_code,e.needs_declaring as i32,e.net_amount.halere(),n]).map_err(|e2|{log::error!("ins: {e2}");DomainError::InvalidInput})?;
        e.id = c.last_insert_rowid();
        Ok(())
    }
    fn update(&self, e: &mut CapitalIncomeEntry) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("UPDATE capital_income_entries SET category=?1,description=?2,income_date=?3,gross_amount=?4,withheld_tax_cz=?5,withheld_tax_foreign=?6,country_code=?7,needs_declaring=?8,net_amount=?9,updated_at=?10 WHERE id=?11",params![e.category.to_string(),e.description,format_date(&e.income_date),e.gross_amount.halere(),e.withheld_tax_cz.halere(),e.withheld_tax_foreign.halere(),e.country_code,e.needs_declaring as i32,e.net_amount.halere(),n,e.id]).map_err(|e2|{log::error!("upd: {e2}");DomainError::InvalidInput})?;
        Ok(())
    }
    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let r = c
            .execute(
                "DELETE FROM capital_income_entries WHERE id=?1",
                params![id],
            )
            .map_err(|e| {
                log::error!("del: {e}");
                DomainError::InvalidInput
            })?;
        if r == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }
    fn get_by_id(&self, id: i64) -> Result<CapitalIncomeEntry, DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row(
            "SELECT * FROM capital_income_entries WHERE id=?1",
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
    fn list_by_year(&self, year: i32) -> Result<Vec<CapitalIncomeEntry>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s = c
            .prepare("SELECT * FROM capital_income_entries WHERE year=?1 ORDER BY income_date")
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
    fn list_by_document_id(
        &self,
        document_id: i64,
    ) -> Result<Vec<CapitalIncomeEntry>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s = c
            .prepare("SELECT * FROM capital_income_entries WHERE document_id=?1")
            .map_err(|e| {
                log::error!("p: {e}");
                DomainError::InvalidInput
            })?;
        s.query_map(params![document_id], scan)
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
    fn sum_by_year(&self, year: i32) -> Result<(Amount, Amount, Amount), DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row("SELECT COALESCE(SUM(gross_amount),0),COALESCE(SUM(withheld_tax_cz+withheld_tax_foreign),0),COALESCE(SUM(net_amount),0) FROM capital_income_entries WHERE year=?1",params![year],|r|Ok((Amount::from_halere(r.get::<_,i64>(0)?),Amount::from_halere(r.get::<_,i64>(1)?),Amount::from_halere(r.get::<_,i64>(2)?)))).map_err(|e|{log::error!("sum: {e}");DomainError::InvalidInput})
    }
    fn delete_by_document_id(&self, document_id: i64) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        c.execute(
            "DELETE FROM capital_income_entries WHERE document_id=?1",
            params![document_id],
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
    use chrono::NaiveDate;
    #[test]
    fn test_create() {
        let c = new_test_db();
        let r = SqliteCapitalIncomeRepo::new(c);
        let mut e = CapitalIncomeEntry {
            id: 0,
            year: 2025,
            document_id: None,
            category: CapitalCategory::DividendCZ,
            description: "Div".into(),
            income_date: NaiveDate::from_ymd_opt(2025, 6, 1).unwrap(),
            gross_amount: Amount::from_halere(10000),
            withheld_tax_cz: Amount::from_halere(1500),
            withheld_tax_foreign: Amount::ZERO,
            country_code: "CZ".into(),
            needs_declaring: false,
            net_amount: Amount::from_halere(8500),
            created_at: Default::default(),
            updated_at: Default::default(),
        };
        r.create(&mut e).unwrap();
        assert!(e.id > 0);
    }
}
