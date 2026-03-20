use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::VATControlStatementRepo;
use zfaktury_domain::*;

pub struct SqliteVATControlRepo {
    conn: Mutex<Connection>,
}
impl SqliteVATControlRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}
fn parse_ft(s: &str) -> FilingType {
    match s {
        "corrective" => FilingType::Corrective,
        "supplementary" => FilingType::Supplementary,
        _ => FilingType::Regular,
    }
}
fn parse_fs(s: &str) -> FilingStatus {
    match s {
        "ready" => FilingStatus::Ready,
        "filed" => FilingStatus::Filed,
        _ => FilingStatus::Draft,
    }
}
fn parse_cs(s: &str) -> ControlSection {
    match s {
        "A4" => ControlSection::A4,
        "A5" => ControlSection::A5,
        "B2" => ControlSection::B2,
        _ => ControlSection::B3,
    }
}

fn scan(row: &Row<'_>) -> rusqlite::Result<VATControlStatement> {
    let ft: String = row.get("filing_type")?;
    let st: String = row.get("status")?;
    let fa: Option<String> = row.get("filed_at")?;
    let c: String = row.get("created_at")?;
    let u: String = row.get("updated_at")?;
    Ok(VATControlStatement {
        id: row.get("id")?,
        period: TaxPeriod {
            year: row.get("year")?,
            month: row.get("month")?,
            quarter: 0,
        },
        filing_type: parse_ft(&ft),
        xml_data: row
            .get::<_, Option<Vec<u8>>>("xml_data")?
            .unwrap_or_default(),
        status: parse_fs(&st),
        filed_at: parse_datetime_optional(fa.as_deref()).unwrap_or(None),
        created_at: parse_datetime(&c).unwrap_or_default(),
        updated_at: parse_datetime(&u).unwrap_or_default(),
    })
}
fn scan_line(row: &Row<'_>) -> rusqlite::Result<VATControlStatementLine> {
    let sec: String = row.get("section")?;
    Ok(VATControlStatementLine {
        id: row.get("id")?,
        control_statement_id: row.get("control_statement_id")?,
        section: parse_cs(&sec),
        partner_dic: row.get("partner_dic")?,
        document_number: row.get("document_number")?,
        dppd: row.get("dppd")?,
        base: Amount::from_halere(row.get::<_, i64>("base")?),
        vat: Amount::from_halere(row.get::<_, i64>("vat")?),
        vat_rate_percent: row.get("vat_rate_percent")?,
        invoice_id: row.get("invoice_id")?,
        expense_id: row.get("expense_id")?,
    })
}

impl VATControlStatementRepo for SqliteVATControlRepo {
    fn create(&self, cs: &mut VATControlStatement) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        conn.execute("INSERT INTO vat_control_statements (year,month,filing_type,xml_data,status,filed_at,created_at,updated_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?7)",
            params![cs.period.year,cs.period.month,cs.filing_type.to_string(),&cs.xml_data as &[u8],cs.status.to_string(),format_datetime_opt(&cs.filed_at),now])
            .map_err(|e|{log::error!("insert cs: {e}");DomainError::InvalidInput})?;
        cs.id = conn.last_insert_rowid();
        Ok(())
    }
    fn update(&self, cs: &mut VATControlStatement) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        conn.execute("UPDATE vat_control_statements SET xml_data=?1,status=?2,filed_at=?3,updated_at=?4 WHERE id=?5",
            params![&cs.xml_data as &[u8],cs.status.to_string(),format_datetime_opt(&cs.filed_at),now,cs.id])
            .map_err(|e|{log::error!("update cs: {e}");DomainError::InvalidInput})?;
        Ok(())
    }
    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let r = conn
            .execute(
                "DELETE FROM vat_control_statements WHERE id=?1",
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
    fn get_by_id(&self, id: i64) -> Result<VATControlStatement, DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.query_row(
            "SELECT * FROM vat_control_statements WHERE id=?1",
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
    fn list(&self, year: i32) -> Result<Vec<VATControlStatement>, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut s = conn
            .prepare("SELECT * FROM vat_control_statements WHERE year=?1 ORDER BY month")
            .map_err(|e| {
                log::error!("prep: {e}");
                DomainError::InvalidInput
            })?;
        s.query_map(params![year], scan)
            .map_err(|e| {
                log::error!("list: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("scan: {e}");
                DomainError::InvalidInput
            })
    }
    fn get_by_period(
        &self,
        year: i32,
        month: i32,
        filing_type: &str,
    ) -> Result<VATControlStatement, DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.query_row(
            "SELECT * FROM vat_control_statements WHERE year=?1 AND month=?2 AND filing_type=?3",
            params![year, month, filing_type],
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
    fn create_lines(&self, lines: &[VATControlStatementLine]) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        for l in lines {
            conn.execute("INSERT INTO vat_control_statement_lines (control_statement_id,section,partner_dic,document_number,dppd,base,vat,vat_rate_percent,invoice_id,expense_id) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10)",
            params![l.control_statement_id,l.section.to_string(),l.partner_dic,l.document_number,l.dppd,l.base.halere(),l.vat.halere(),l.vat_rate_percent,l.invoice_id,l.expense_id])
            .map_err(|e|{log::error!("insert line: {e}");DomainError::InvalidInput})?;
        }
        Ok(())
    }
    fn delete_lines(&self, control_statement_id: i64) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.execute(
            "DELETE FROM vat_control_statement_lines WHERE control_statement_id=?1",
            params![control_statement_id],
        )
        .map_err(|e| {
            log::error!("del lines: {e}");
            DomainError::InvalidInput
        })?;
        Ok(())
    }
    fn get_lines(
        &self,
        control_statement_id: i64,
    ) -> Result<Vec<VATControlStatementLine>, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut s = conn
            .prepare("SELECT * FROM vat_control_statement_lines WHERE control_statement_id=?1")
            .map_err(|e| {
                log::error!("prep: {e}");
                DomainError::InvalidInput
            })?;
        s.query_map(params![control_statement_id], scan_line)
            .map_err(|e| {
                log::error!("list: {e}");
                DomainError::InvalidInput
            })?
            .collect::<Result<Vec<_>, _>>()
            .map_err(|e| {
                log::error!("scan: {e}");
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
        let repo = SqliteVATControlRepo::new(conn);
        let mut cs = VATControlStatement {
            id: 0,
            period: TaxPeriod {
                year: 2025,
                month: 1,
                quarter: 0,
            },
            filing_type: FilingType::Regular,
            xml_data: Vec::new(),
            status: FilingStatus::Draft,
            filed_at: None,
            created_at: Default::default(),
            updated_at: Default::default(),
        };
        repo.create(&mut cs).unwrap();
        assert!(cs.id > 0);
        let f = repo.get_by_id(cs.id).unwrap();
        assert_eq!(f.period.month, 1);
    }
}
