use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::HealthInsuranceOverviewRepo;
use zfaktury_domain::*;
pub struct SqliteHealthInsuranceRepo {
    conn: Mutex<Connection>,
}
impl SqliteHealthInsuranceRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}
fn pft(s: &str) -> FilingType {
    match s {
        "corrective" => FilingType::Corrective,
        "supplementary" => FilingType::Supplementary,
        _ => FilingType::Regular,
    }
}
fn pfs(s: &str) -> FilingStatus {
    match s {
        "ready" => FilingStatus::Ready,
        "filed" => FilingStatus::Filed,
        _ => FilingStatus::Draft,
    }
}
fn scan(row: &Row<'_>) -> rusqlite::Result<HealthInsuranceOverview> {
    let ft: String = row.get("filing_type")?;
    let st: String = row.get("status")?;
    let fa: Option<String> = row.get("filed_at")?;
    let c: String = row.get("created_at")?;
    let u: String = row.get("updated_at")?;
    Ok(HealthInsuranceOverview {
        id: row.get("id")?,
        year: row.get("year")?,
        filing_type: pft(&ft),
        total_revenue: Amount::from_halere(row.get::<_, i64>("total_revenue")?),
        total_expenses: Amount::from_halere(row.get::<_, i64>("total_expenses")?),
        tax_base: Amount::from_halere(row.get::<_, i64>("tax_base")?),
        assessment_base: Amount::from_halere(row.get::<_, i64>("assessment_base")?),
        min_assessment_base: Amount::from_halere(row.get::<_, i64>("min_assessment_base")?),
        final_assessment_base: Amount::from_halere(row.get::<_, i64>("final_assessment_base")?),
        insurance_rate: row.get("insurance_rate")?,
        total_insurance: Amount::from_halere(row.get::<_, i64>("total_insurance")?),
        prepayments: Amount::from_halere(row.get::<_, i64>("prepayments")?),
        difference: Amount::from_halere(row.get::<_, i64>("difference")?),
        new_monthly_prepay: Amount::from_halere(row.get::<_, i64>("new_monthly_prepay")?),
        xml_data: row
            .get::<_, Option<Vec<u8>>>("xml_data")?
            .unwrap_or_default(),
        status: pfs(&st),
        filed_at: parse_datetime_optional(fa.as_deref()).unwrap_or(None),
        created_at: parse_datetime(&c).unwrap_or_default(),
        updated_at: parse_datetime(&u).unwrap_or_default(),
    })
}
impl HealthInsuranceOverviewRepo for SqliteHealthInsuranceRepo {
    fn create(&self, sio: &mut HealthInsuranceOverview) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("INSERT INTO health_insurance_overviews (year,filing_type,total_revenue,total_expenses,tax_base,assessment_base,min_assessment_base,final_assessment_base,insurance_rate,total_insurance,prepayments,difference,new_monthly_prepay,xml_data,status,filed_at,created_at,updated_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11,?12,?13,?14,?15,?16,?17,?17)",params![sio.year,sio.filing_type.to_string(),sio.total_revenue.halere(),sio.total_expenses.halere(),sio.tax_base.halere(),sio.assessment_base.halere(),sio.min_assessment_base.halere(),sio.final_assessment_base.halere(),sio.insurance_rate,sio.total_insurance.halere(),sio.prepayments.halere(),sio.difference.halere(),sio.new_monthly_prepay.halere(),&sio.xml_data as &[u8],sio.status.to_string(),format_datetime_opt(&sio.filed_at),n]).map_err(|e|{log::error!("insert: {e}");DomainError::InvalidInput})?;
        sio.id = c.last_insert_rowid();
        Ok(())
    }
    fn update(&self, sio: &mut HealthInsuranceOverview) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("UPDATE health_insurance_overviews SET total_revenue=?1,total_expenses=?2,tax_base=?3,assessment_base=?4,min_assessment_base=?5,final_assessment_base=?6,insurance_rate=?7,total_insurance=?8,prepayments=?9,difference=?10,new_monthly_prepay=?11,xml_data=?12,status=?13,filed_at=?14,updated_at=?15 WHERE id=?16",params![sio.total_revenue.halere(),sio.total_expenses.halere(),sio.tax_base.halere(),sio.assessment_base.halere(),sio.min_assessment_base.halere(),sio.final_assessment_base.halere(),sio.insurance_rate,sio.total_insurance.halere(),sio.prepayments.halere(),sio.difference.halere(),sio.new_monthly_prepay.halere(),&sio.xml_data as &[u8],sio.status.to_string(),format_datetime_opt(&sio.filed_at),n,sio.id]).map_err(|e|{log::error!("update: {e}");DomainError::InvalidInput})?;
        Ok(())
    }
    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let r = c
            .execute(
                "DELETE FROM health_insurance_overviews WHERE id=?1",
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
    fn get_by_id(&self, id: i64) -> Result<HealthInsuranceOverview, DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row(
            "SELECT * FROM health_insurance_overviews WHERE id=?1",
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
    fn list(&self, year: i32) -> Result<Vec<HealthInsuranceOverview>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s = c
            .prepare("SELECT * FROM health_insurance_overviews WHERE year=?1")
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
    fn get_by_year(
        &self,
        year: i32,
        filing_type: &str,
    ) -> Result<HealthInsuranceOverview, DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row(
            "SELECT * FROM health_insurance_overviews WHERE year=?1 AND filing_type=?2",
            params![year, filing_type],
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
    fn test_create() {
        let c = new_test_db();
        let r = SqliteHealthInsuranceRepo::new(c);
        let mut sio = HealthInsuranceOverview {
            id: 0,
            year: 2025,
            filing_type: FilingType::Regular,
            total_revenue: Amount::ZERO,
            total_expenses: Amount::ZERO,
            tax_base: Amount::ZERO,
            assessment_base: Amount::ZERO,
            min_assessment_base: Amount::ZERO,
            final_assessment_base: Amount::ZERO,
            insurance_rate: 135,
            total_insurance: Amount::ZERO,
            prepayments: Amount::ZERO,
            difference: Amount::ZERO,
            new_monthly_prepay: Amount::ZERO,
            xml_data: Vec::new(),
            status: FilingStatus::Draft,
            filed_at: None,
            created_at: Default::default(),
            updated_at: Default::default(),
        };
        r.create(&mut sio).unwrap();
        assert!(sio.id > 0);
    }
}
