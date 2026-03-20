use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::VIESSummaryRepo;
use zfaktury_domain::*;
pub struct SqliteVIESRepo {
    conn: Mutex<Connection>,
}
impl SqliteVIESRepo {
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
fn scan(row: &Row<'_>) -> rusqlite::Result<VIESSummary> {
    let ft: String = row.get("filing_type")?;
    let st: String = row.get("status")?;
    let fa: Option<String> = row.get("filed_at")?;
    let c: String = row.get("created_at")?;
    let u: String = row.get("updated_at")?;
    Ok(VIESSummary {
        id: row.get("id")?,
        period: TaxPeriod {
            year: row.get("year")?,
            month: 0,
            quarter: row.get("quarter")?,
        },
        filing_type: pft(&ft),
        xml_data: row
            .get::<_, Option<Vec<u8>>>("xml_data")?
            .unwrap_or_default(),
        status: pfs(&st),
        filed_at: parse_datetime_optional(fa.as_deref()).unwrap_or(None),
        created_at: parse_datetime_or_default(&c),
        updated_at: parse_datetime_or_default(&u),
    })
}
fn scan_line(row: &Row<'_>) -> rusqlite::Result<VIESSummaryLine> {
    Ok(VIESSummaryLine {
        id: row.get("id")?,
        vies_summary_id: row.get("vies_summary_id")?,
        partner_dic: row.get("partner_dic")?,
        country_code: row.get("country_code")?,
        total_amount: Amount::from_halere(row.get::<_, i64>("total_amount")?),
        service_code: row.get("service_code")?,
    })
}
impl VIESSummaryRepo for SqliteVIESRepo {
    fn create(&self, vs: &mut VIESSummary) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("INSERT INTO vies_summaries (year,quarter,filing_type,xml_data,status,filed_at,created_at,updated_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?7)",params![vs.period.year,vs.period.quarter,vs.filing_type.to_string(),&vs.xml_data as &[u8],vs.status.to_string(),format_datetime_opt(&vs.filed_at),n]).map_err(|e|{log::error!("insert: {e}");DomainError::InvalidInput})?;
        vs.id = c.last_insert_rowid();
        Ok(())
    }
    fn update(&self, vs: &mut VIESSummary) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute(
            "UPDATE vies_summaries SET xml_data=?1,status=?2,filed_at=?3,updated_at=?4 WHERE id=?5",
            params![
                &vs.xml_data as &[u8],
                vs.status.to_string(),
                format_datetime_opt(&vs.filed_at),
                n,
                vs.id
            ],
        )
        .map_err(|e| {
            log::error!("update: {e}");
            DomainError::InvalidInput
        })?;
        Ok(())
    }
    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let r = c
            .execute("DELETE FROM vies_summaries WHERE id=?1", params![id])
            .map_err(|e| {
                log::error!("del: {e}");
                DomainError::InvalidInput
            })?;
        if r == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }
    fn get_by_id(&self, id: i64) -> Result<VIESSummary, DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row(
            "SELECT * FROM vies_summaries WHERE id=?1",
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
    fn list(&self, year: i32) -> Result<Vec<VIESSummary>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s = c
            .prepare("SELECT * FROM vies_summaries WHERE year=?1 ORDER BY quarter")
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
    fn get_by_period(
        &self,
        year: i32,
        quarter: i32,
        filing_type: &str,
    ) -> Result<VIESSummary, DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row(
            "SELECT * FROM vies_summaries WHERE year=?1 AND quarter=?2 AND filing_type=?3",
            params![year, quarter, filing_type],
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
    fn create_lines(&self, lines: &[VIESSummaryLine]) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        for l in lines {
            c.execute("INSERT INTO vies_summary_lines (vies_summary_id,partner_dic,country_code,total_amount,service_code) VALUES (?1,?2,?3,?4,?5)",params![l.vies_summary_id,l.partner_dic,l.country_code,l.total_amount.halere(),l.service_code]).map_err(|e|{log::error!("ins: {e}");DomainError::InvalidInput})?;
        }
        Ok(())
    }
    fn delete_lines(&self, vies_summary_id: i64) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        c.execute(
            "DELETE FROM vies_summary_lines WHERE vies_summary_id=?1",
            params![vies_summary_id],
        )
        .map_err(|e| {
            log::error!("del: {e}");
            DomainError::InvalidInput
        })?;
        Ok(())
    }
    fn get_lines(&self, vies_summary_id: i64) -> Result<Vec<VIESSummaryLine>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s = c
            .prepare("SELECT * FROM vies_summary_lines WHERE vies_summary_id=?1")
            .map_err(|e| {
                log::error!("p: {e}");
                DomainError::InvalidInput
            })?;
        s.query_map(params![vies_summary_id], scan_line)
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
        let conn = new_test_db();
        let repo = SqliteVIESRepo::new(conn);
        let mut vs = VIESSummary {
            id: 0,
            period: TaxPeriod {
                year: 2025,
                month: 0,
                quarter: 1,
            },
            filing_type: FilingType::Regular,
            xml_data: Vec::new(),
            status: FilingStatus::Draft,
            filed_at: None,
            created_at: Default::default(),
            updated_at: Default::default(),
        };
        repo.create(&mut vs).unwrap();
        assert!(vs.id > 0);
    }
}
