use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::VATReturnRepo;
use zfaktury_domain::*;

pub struct SqliteVATReturnRepo {
    conn: Mutex<Connection>,
}
impl SqliteVATReturnRepo {
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

fn scan(row: &Row<'_>) -> rusqlite::Result<VATReturn> {
    let ft: String = row.get("filing_type")?;
    let st: String = row.get("status")?;
    let fa: Option<String> = row.get("filed_at")?;
    let c: String = row.get("created_at")?;
    let u: String = row.get("updated_at")?;
    Ok(VATReturn {
        id: row.get("id")?,
        period: TaxPeriod {
            year: row.get("year")?,
            month: row.get("month")?,
            quarter: row.get("quarter")?,
        },
        filing_type: parse_ft(&ft),
        output_vat_base_21: Amount::from_halere(row.get::<_, i64>("output_vat_base_21")?),
        output_vat_amount_21: Amount::from_halere(row.get::<_, i64>("output_vat_amount_21")?),
        output_vat_base_12: Amount::from_halere(row.get::<_, i64>("output_vat_base_12")?),
        output_vat_amount_12: Amount::from_halere(row.get::<_, i64>("output_vat_amount_12")?),
        output_vat_base_0: Amount::from_halere(row.get::<_, i64>("output_vat_base_0")?),
        reverse_charge_base_21: Amount::from_halere(row.get::<_, i64>("reverse_charge_base_21")?),
        reverse_charge_amount_21: Amount::from_halere(
            row.get::<_, i64>("reverse_charge_amount_21")?,
        ),
        reverse_charge_base_12: Amount::from_halere(row.get::<_, i64>("reverse_charge_base_12")?),
        reverse_charge_amount_12: Amount::from_halere(
            row.get::<_, i64>("reverse_charge_amount_12")?,
        ),
        input_vat_base_21: Amount::from_halere(row.get::<_, i64>("input_vat_base_21")?),
        input_vat_amount_21: Amount::from_halere(row.get::<_, i64>("input_vat_amount_21")?),
        input_vat_base_12: Amount::from_halere(row.get::<_, i64>("input_vat_base_12")?),
        input_vat_amount_12: Amount::from_halere(row.get::<_, i64>("input_vat_amount_12")?),
        total_output_vat: Amount::from_halere(row.get::<_, i64>("total_output_vat")?),
        total_input_vat: Amount::from_halere(row.get::<_, i64>("total_input_vat")?),
        net_vat: Amount::from_halere(row.get::<_, i64>("net_vat")?),
        xml_data: row
            .get::<_, Option<Vec<u8>>>("xml_data")?
            .unwrap_or_default(),
        status: parse_fs(&st),
        filed_at: parse_datetime_optional(fa.as_deref()).unwrap_or(None),
        created_at: parse_datetime(&c).unwrap_or_default(),
        updated_at: parse_datetime(&u).unwrap_or_default(),
    })
}

impl VATReturnRepo for SqliteVATReturnRepo {
    fn create(&self, vr: &mut VATReturn) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        conn.execute("INSERT INTO vat_returns (year,month,quarter,filing_type,output_vat_base_21,output_vat_amount_21,output_vat_base_12,output_vat_amount_12,output_vat_base_0,reverse_charge_base_21,reverse_charge_amount_21,reverse_charge_base_12,reverse_charge_amount_12,input_vat_base_21,input_vat_amount_21,input_vat_base_12,input_vat_amount_12,total_output_vat,total_input_vat,net_vat,xml_data,status,filed_at,created_at,updated_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11,?12,?13,?14,?15,?16,?17,?18,?19,?20,?21,?22,?23,?24,?24)",
            params![vr.period.year,vr.period.month,vr.period.quarter,vr.filing_type.to_string(),vr.output_vat_base_21.halere(),vr.output_vat_amount_21.halere(),vr.output_vat_base_12.halere(),vr.output_vat_amount_12.halere(),vr.output_vat_base_0.halere(),vr.reverse_charge_base_21.halere(),vr.reverse_charge_amount_21.halere(),vr.reverse_charge_base_12.halere(),vr.reverse_charge_amount_12.halere(),vr.input_vat_base_21.halere(),vr.input_vat_amount_21.halere(),vr.input_vat_base_12.halere(),vr.input_vat_amount_12.halere(),vr.total_output_vat.halere(),vr.total_input_vat.halere(),vr.net_vat.halere(),&vr.xml_data as &[u8],vr.status.to_string(),format_datetime_opt(&vr.filed_at),now])
            .map_err(|e|{log::error!("insert vat return: {e}");DomainError::InvalidInput})?;
        vr.id = conn.last_insert_rowid();
        Ok(())
    }
    fn update(&self, vr: &mut VATReturn) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let now = format_datetime(&Local::now().naive_local());
        conn.execute("UPDATE vat_returns SET output_vat_base_21=?1,output_vat_amount_21=?2,output_vat_base_12=?3,output_vat_amount_12=?4,output_vat_base_0=?5,reverse_charge_base_21=?6,reverse_charge_amount_21=?7,reverse_charge_base_12=?8,reverse_charge_amount_12=?9,input_vat_base_21=?10,input_vat_amount_21=?11,input_vat_base_12=?12,input_vat_amount_12=?13,total_output_vat=?14,total_input_vat=?15,net_vat=?16,xml_data=?17,status=?18,filed_at=?19,updated_at=?20 WHERE id=?21",
            params![vr.output_vat_base_21.halere(),vr.output_vat_amount_21.halere(),vr.output_vat_base_12.halere(),vr.output_vat_amount_12.halere(),vr.output_vat_base_0.halere(),vr.reverse_charge_base_21.halere(),vr.reverse_charge_amount_21.halere(),vr.reverse_charge_base_12.halere(),vr.reverse_charge_amount_12.halere(),vr.input_vat_base_21.halere(),vr.input_vat_amount_21.halere(),vr.input_vat_base_12.halere(),vr.input_vat_amount_12.halere(),vr.total_output_vat.halere(),vr.total_input_vat.halere(),vr.net_vat.halere(),&vr.xml_data as &[u8],vr.status.to_string(),format_datetime_opt(&vr.filed_at),now,vr.id])
            .map_err(|e|{log::error!("update vat return: {e}");DomainError::InvalidInput})?;
        Ok(())
    }
    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        let r = conn
            .execute("DELETE FROM vat_returns WHERE id=?1", params![id])
            .map_err(|e| {
                log::error!("del: {e}");
                DomainError::InvalidInput
            })?;
        if r == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }
    fn get_by_id(&self, id: i64) -> Result<VATReturn, DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.query_row("SELECT * FROM vat_returns WHERE id=?1", params![id], scan)
            .map_err(|e| match e {
                rusqlite::Error::QueryReturnedNoRows => DomainError::NotFound,
                _ => {
                    log::error!("q: {e}");
                    DomainError::InvalidInput
                }
            })
    }
    fn list(&self, year: i32) -> Result<Vec<VATReturn>, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut s = conn
            .prepare("SELECT * FROM vat_returns WHERE year=?1 ORDER BY month,quarter")
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
        quarter: i32,
        filing_type: &str,
    ) -> Result<VATReturn, DomainError> {
        let conn = self.conn.lock().unwrap();
        conn.query_row("SELECT * FROM vat_returns WHERE year=?1 AND month=?2 AND quarter=?3 AND filing_type=?4",params![year,month,quarter,filing_type],scan)
            .map_err(|e| match e { rusqlite::Error::QueryReturnedNoRows=>DomainError::NotFound, _=>{log::error!("q: {e}");DomainError::InvalidInput}})
    }
    fn link_invoices(&self, vat_return_id: i64, invoice_ids: &[i64]) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        for id in invoice_ids {
            conn.execute("INSERT OR IGNORE INTO vat_return_invoices (vat_return_id,invoice_id) VALUES (?1,?2)",params![vat_return_id,id]).map_err(|e|{log::error!("link inv: {e}");DomainError::InvalidInput})?;
        }
        Ok(())
    }
    fn link_expenses(&self, vat_return_id: i64, expense_ids: &[i64]) -> Result<(), DomainError> {
        let conn = self.conn.lock().unwrap();
        for id in expense_ids {
            conn.execute("INSERT OR IGNORE INTO vat_return_expenses (vat_return_id,expense_id) VALUES (?1,?2)",params![vat_return_id,id]).map_err(|e|{log::error!("link exp: {e}");DomainError::InvalidInput})?;
        }
        Ok(())
    }
    fn get_linked_invoice_ids(&self, vat_return_id: i64) -> Result<Vec<i64>, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut s = conn
            .prepare("SELECT invoice_id FROM vat_return_invoices WHERE vat_return_id=?1")
            .map_err(|e| {
                log::error!("prep: {e}");
                DomainError::InvalidInput
            })?;
        s.query_map(params![vat_return_id], |r| r.get(0))
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
    fn get_linked_expense_ids(&self, vat_return_id: i64) -> Result<Vec<i64>, DomainError> {
        let conn = self.conn.lock().unwrap();
        let mut s = conn
            .prepare("SELECT expense_id FROM vat_return_expenses WHERE vat_return_id=?1")
            .map_err(|e| {
                log::error!("prep: {e}");
                DomainError::InvalidInput
            })?;
        s.query_map(params![vat_return_id], |r| r.get(0))
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
    fn test_create_and_get() {
        let conn = new_test_db();
        let repo = SqliteVATReturnRepo::new(conn);
        let mut vr = VATReturn {
            id: 0,
            period: TaxPeriod {
                year: 2025,
                month: 1,
                quarter: 0,
            },
            filing_type: FilingType::Regular,
            output_vat_base_21: Amount::ZERO,
            output_vat_amount_21: Amount::ZERO,
            output_vat_base_12: Amount::ZERO,
            output_vat_amount_12: Amount::ZERO,
            output_vat_base_0: Amount::ZERO,
            reverse_charge_base_21: Amount::ZERO,
            reverse_charge_amount_21: Amount::ZERO,
            reverse_charge_base_12: Amount::ZERO,
            reverse_charge_amount_12: Amount::ZERO,
            input_vat_base_21: Amount::ZERO,
            input_vat_amount_21: Amount::ZERO,
            input_vat_base_12: Amount::ZERO,
            input_vat_amount_12: Amount::ZERO,
            total_output_vat: Amount::ZERO,
            total_input_vat: Amount::ZERO,
            net_vat: Amount::ZERO,
            xml_data: Vec::new(),
            status: FilingStatus::Draft,
            filed_at: None,
            created_at: Default::default(),
            updated_at: Default::default(),
        };
        repo.create(&mut vr).unwrap();
        assert!(vr.id > 0);
        let f = repo.get_by_id(vr.id).unwrap();
        assert_eq!(f.period.year, 2025);
    }
}
