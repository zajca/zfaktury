use crate::helpers::*;
use chrono::Local;
use rusqlite::{Connection, Row, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::IncomeTaxReturnRepo;
use zfaktury_domain::*;
pub struct SqliteIncomeTaxReturnRepo {
    conn: Mutex<Connection>,
}
impl SqliteIncomeTaxReturnRepo {
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
fn scan(row: &Row<'_>) -> rusqlite::Result<IncomeTaxReturn> {
    let ft: String = row.get("filing_type")?;
    let st: String = row.get("status")?;
    let fa: Option<String> = row.get("filed_at")?;
    let c: String = row.get("created_at")?;
    let u: String = row.get("updated_at")?;
    Ok(IncomeTaxReturn {
        id: row.get("id")?,
        year: row.get("year")?,
        filing_type: pft(&ft),
        total_revenue: Amount::from_halere(row.get::<_, i64>("total_revenue")?),
        actual_expenses: Amount::from_halere(row.get::<_, i64>("actual_expenses")?),
        flat_rate_percent: row.get("flat_rate_percent")?,
        flat_rate_amount: Amount::from_halere(row.get::<_, i64>("flat_rate_amount")?),
        used_expenses: Amount::from_halere(row.get::<_, i64>("used_expenses")?),
        tax_base: Amount::from_halere(row.get::<_, i64>("tax_base")?),
        total_deductions: Amount::from_halere(row.get::<_, i64>("total_deductions")?),
        tax_base_rounded: Amount::from_halere(row.get::<_, i64>("tax_base_rounded")?),
        tax_at_15: Amount::from_halere(row.get::<_, i64>("tax_at_15")?),
        tax_at_23: Amount::from_halere(row.get::<_, i64>("tax_at_23")?),
        total_tax: Amount::from_halere(row.get::<_, i64>("total_tax")?),
        credit_basic: Amount::from_halere(row.get::<_, i64>("credit_basic")?),
        credit_spouse: Amount::from_halere(row.get::<_, i64>("credit_spouse")?),
        credit_disability: Amount::from_halere(row.get::<_, i64>("credit_disability")?),
        credit_student: Amount::from_halere(row.get::<_, i64>("credit_student")?),
        total_credits: Amount::from_halere(row.get::<_, i64>("total_credits")?),
        tax_after_credits: Amount::from_halere(row.get::<_, i64>("tax_after_credits")?),
        child_benefit: Amount::from_halere(row.get::<_, i64>("child_benefit")?),
        tax_after_benefit: Amount::from_halere(row.get::<_, i64>("tax_after_benefit")?),
        prepayments: Amount::from_halere(row.get::<_, i64>("prepayments")?),
        tax_due: Amount::from_halere(row.get::<_, i64>("tax_due")?),
        capital_income_gross: Amount::from_halere(row.get::<_, i64>("capital_income_gross")?),
        capital_income_tax: Amount::from_halere(row.get::<_, i64>("capital_income_tax")?),
        capital_income_net: Amount::from_halere(row.get::<_, i64>("capital_income_net")?),
        other_income_gross: Amount::from_halere(row.get::<_, i64>("other_income_gross")?),
        other_income_expenses: Amount::from_halere(row.get::<_, i64>("other_income_expenses")?),
        other_income_exempt: Amount::from_halere(row.get::<_, i64>("other_income_exempt")?),
        other_income_net: Amount::from_halere(row.get::<_, i64>("other_income_net")?),
        xml_data: row
            .get::<_, Option<Vec<u8>>>("xml_data")?
            .unwrap_or_default(),
        status: pfs(&st),
        filed_at: parse_datetime_optional(fa.as_deref()).unwrap_or(None),
        created_at: parse_datetime(&c).unwrap_or_default(),
        updated_at: parse_datetime(&u).unwrap_or_default(),
    })
}
impl IncomeTaxReturnRepo for SqliteIncomeTaxReturnRepo {
    fn create(&self, itr: &mut IncomeTaxReturn) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("INSERT INTO income_tax_returns (year,filing_type,total_revenue,actual_expenses,flat_rate_percent,flat_rate_amount,used_expenses,tax_base,total_deductions,tax_base_rounded,tax_at_15,tax_at_23,total_tax,credit_basic,credit_spouse,credit_disability,credit_student,total_credits,tax_after_credits,child_benefit,tax_after_benefit,prepayments,tax_due,capital_income_gross,capital_income_tax,capital_income_net,other_income_gross,other_income_expenses,other_income_exempt,other_income_net,xml_data,status,filed_at,created_at,updated_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11,?12,?13,?14,?15,?16,?17,?18,?19,?20,?21,?22,?23,?24,?25,?26,?27,?28,?29,?30,?31,?32,?33,?34,?34)",params![itr.year,itr.filing_type.to_string(),itr.total_revenue.halere(),itr.actual_expenses.halere(),itr.flat_rate_percent,itr.flat_rate_amount.halere(),itr.used_expenses.halere(),itr.tax_base.halere(),itr.total_deductions.halere(),itr.tax_base_rounded.halere(),itr.tax_at_15.halere(),itr.tax_at_23.halere(),itr.total_tax.halere(),itr.credit_basic.halere(),itr.credit_spouse.halere(),itr.credit_disability.halere(),itr.credit_student.halere(),itr.total_credits.halere(),itr.tax_after_credits.halere(),itr.child_benefit.halere(),itr.tax_after_benefit.halere(),itr.prepayments.halere(),itr.tax_due.halere(),itr.capital_income_gross.halere(),itr.capital_income_tax.halere(),itr.capital_income_net.halere(),itr.other_income_gross.halere(),itr.other_income_expenses.halere(),itr.other_income_exempt.halere(),itr.other_income_net.halere(),&itr.xml_data as &[u8],itr.status.to_string(),format_datetime_opt(&itr.filed_at),n]).map_err(|e|{log::error!("insert: {e}");DomainError::InvalidInput})?;
        itr.id = c.last_insert_rowid();
        Ok(())
    }
    fn update(&self, itr: &mut IncomeTaxReturn) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let n = format_datetime(&Local::now().naive_local());
        c.execute("UPDATE income_tax_returns SET total_revenue=?1,actual_expenses=?2,flat_rate_percent=?3,flat_rate_amount=?4,used_expenses=?5,tax_base=?6,total_deductions=?7,tax_base_rounded=?8,tax_at_15=?9,tax_at_23=?10,total_tax=?11,credit_basic=?12,credit_spouse=?13,credit_disability=?14,credit_student=?15,total_credits=?16,tax_after_credits=?17,child_benefit=?18,tax_after_benefit=?19,prepayments=?20,tax_due=?21,capital_income_gross=?22,capital_income_tax=?23,capital_income_net=?24,other_income_gross=?25,other_income_expenses=?26,other_income_exempt=?27,other_income_net=?28,xml_data=?29,status=?30,filed_at=?31,updated_at=?32 WHERE id=?33",params![itr.total_revenue.halere(),itr.actual_expenses.halere(),itr.flat_rate_percent,itr.flat_rate_amount.halere(),itr.used_expenses.halere(),itr.tax_base.halere(),itr.total_deductions.halere(),itr.tax_base_rounded.halere(),itr.tax_at_15.halere(),itr.tax_at_23.halere(),itr.total_tax.halere(),itr.credit_basic.halere(),itr.credit_spouse.halere(),itr.credit_disability.halere(),itr.credit_student.halere(),itr.total_credits.halere(),itr.tax_after_credits.halere(),itr.child_benefit.halere(),itr.tax_after_benefit.halere(),itr.prepayments.halere(),itr.tax_due.halere(),itr.capital_income_gross.halere(),itr.capital_income_tax.halere(),itr.capital_income_net.halere(),itr.other_income_gross.halere(),itr.other_income_expenses.halere(),itr.other_income_exempt.halere(),itr.other_income_net.halere(),&itr.xml_data as &[u8],itr.status.to_string(),format_datetime_opt(&itr.filed_at),n,itr.id]).map_err(|e|{log::error!("update: {e}");DomainError::InvalidInput})?;
        Ok(())
    }
    fn delete(&self, id: i64) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        let r = c
            .execute("DELETE FROM income_tax_returns WHERE id=?1", params![id])
            .map_err(|e| {
                log::error!("del: {e}");
                DomainError::InvalidInput
            })?;
        if r == 0 {
            return Err(DomainError::NotFound);
        }
        Ok(())
    }
    fn get_by_id(&self, id: i64) -> Result<IncomeTaxReturn, DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row(
            "SELECT * FROM income_tax_returns WHERE id=?1",
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
    fn list(&self, year: i32) -> Result<Vec<IncomeTaxReturn>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s = c
            .prepare("SELECT * FROM income_tax_returns WHERE year=?1")
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
    fn get_by_year(&self, year: i32, filing_type: &str) -> Result<IncomeTaxReturn, DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row(
            "SELECT * FROM income_tax_returns WHERE year=?1 AND filing_type=?2",
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
    fn link_invoices(&self, id: i64, invoice_ids: &[i64]) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        for iid in invoice_ids {
            c.execute("INSERT OR IGNORE INTO income_tax_return_invoices (income_tax_return_id,invoice_id) VALUES (?1,?2)",params![id,iid]).map_err(|e|{log::error!("link: {e}");DomainError::InvalidInput})?;
        }
        Ok(())
    }
    fn link_expenses(&self, id: i64, expense_ids: &[i64]) -> Result<(), DomainError> {
        let c = self.conn.lock().unwrap();
        for eid in expense_ids {
            c.execute("INSERT OR IGNORE INTO income_tax_return_expenses (income_tax_return_id,expense_id) VALUES (?1,?2)",params![id,eid]).map_err(|e|{log::error!("link: {e}");DomainError::InvalidInput})?;
        }
        Ok(())
    }
    fn get_linked_invoice_ids(&self, id: i64) -> Result<Vec<i64>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s = c
            .prepare(
                "SELECT invoice_id FROM income_tax_return_invoices WHERE income_tax_return_id=?1",
            )
            .map_err(|e| {
                log::error!("p: {e}");
                DomainError::InvalidInput
            })?;
        s.query_map(params![id], |r| r.get(0))
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
    fn get_linked_expense_ids(&self, id: i64) -> Result<Vec<i64>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s = c
            .prepare(
                "SELECT expense_id FROM income_tax_return_expenses WHERE income_tax_return_id=?1",
            )
            .map_err(|e| {
                log::error!("p: {e}");
                DomainError::InvalidInput
            })?;
        s.query_map(params![id], |r| r.get(0))
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
        let repo = SqliteIncomeTaxReturnRepo::new(conn);
        let mut itr = IncomeTaxReturn {
            id: 0,
            year: 2025,
            filing_type: FilingType::Regular,
            total_revenue: Amount::ZERO,
            actual_expenses: Amount::ZERO,
            flat_rate_percent: 60,
            flat_rate_amount: Amount::ZERO,
            used_expenses: Amount::ZERO,
            tax_base: Amount::ZERO,
            total_deductions: Amount::ZERO,
            tax_base_rounded: Amount::ZERO,
            tax_at_15: Amount::ZERO,
            tax_at_23: Amount::ZERO,
            total_tax: Amount::ZERO,
            credit_basic: Amount::ZERO,
            credit_spouse: Amount::ZERO,
            credit_disability: Amount::ZERO,
            credit_student: Amount::ZERO,
            total_credits: Amount::ZERO,
            tax_after_credits: Amount::ZERO,
            child_benefit: Amount::ZERO,
            tax_after_benefit: Amount::ZERO,
            prepayments: Amount::ZERO,
            tax_due: Amount::ZERO,
            capital_income_gross: Amount::ZERO,
            capital_income_tax: Amount::ZERO,
            capital_income_net: Amount::ZERO,
            other_income_gross: Amount::ZERO,
            other_income_expenses: Amount::ZERO,
            other_income_exempt: Amount::ZERO,
            other_income_net: Amount::ZERO,
            xml_data: Vec::new(),
            status: FilingStatus::Draft,
            filed_at: None,
            created_at: Default::default(),
            updated_at: Default::default(),
        };
        repo.create(&mut itr).unwrap();
        assert!(itr.id > 0);
    }
}
