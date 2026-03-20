use rusqlite::{Connection, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::ReportRepo;
use zfaktury_core::repository::types::*;
use zfaktury_domain::*;
pub struct SqliteReportRepo {
    conn: Mutex<Connection>,
}
impl SqliteReportRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}
impl ReportRepo for SqliteReportRepo {
    fn monthly_revenue(&self, year: i32) -> Result<Vec<MonthlyAmount>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s=c.prepare("SELECT CAST(SUBSTR(issue_date,6,2) AS INTEGER),COALESCE(SUM(total_amount),0) FROM invoices WHERE status='paid' AND SUBSTR(issue_date,1,4)=?1 AND deleted_at IS NULL GROUP BY 1 ORDER BY 1").map_err(|e|{log::error!("p: {e}");DomainError::InvalidInput})?;
        s.query_map(params![year.to_string()], |r| {
            Ok(MonthlyAmount {
                month: r.get(0)?,
                amount: Amount::from_halere(r.get::<_, i64>(1)?),
            })
        })
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
    fn quarterly_revenue(&self, year: i32) -> Result<Vec<QuarterlyAmount>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s=c.prepare("SELECT ((CAST(SUBSTR(issue_date,6,2) AS INTEGER)-1)/3)+1,COALESCE(SUM(total_amount),0) FROM invoices WHERE status='paid' AND SUBSTR(issue_date,1,4)=?1 AND deleted_at IS NULL GROUP BY 1 ORDER BY 1").map_err(|e|{log::error!("p: {e}");DomainError::InvalidInput})?;
        s.query_map(params![year.to_string()], |r| {
            Ok(QuarterlyAmount {
                quarter: r.get(0)?,
                amount: Amount::from_halere(r.get::<_, i64>(1)?),
            })
        })
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
    fn yearly_revenue(&self, year: i32) -> Result<Amount, DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row("SELECT COALESCE(SUM(total_amount),0) FROM invoices WHERE status='paid' AND SUBSTR(issue_date,1,4)=?1 AND deleted_at IS NULL",params![year.to_string()],|r|Ok(Amount::from_halere(r.get::<_,i64>(0)?))).map_err(|e|{log::error!("q: {e}");DomainError::InvalidInput})
    }
    fn monthly_expenses(&self, year: i32) -> Result<Vec<MonthlyAmount>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s=c.prepare("SELECT CAST(SUBSTR(issue_date,6,2) AS INTEGER),COALESCE(SUM(amount),0) FROM expenses WHERE SUBSTR(issue_date,1,4)=?1 AND deleted_at IS NULL GROUP BY 1 ORDER BY 1").map_err(|e|{log::error!("p: {e}");DomainError::InvalidInput})?;
        s.query_map(params![year.to_string()], |r| {
            Ok(MonthlyAmount {
                month: r.get(0)?,
                amount: Amount::from_halere(r.get::<_, i64>(1)?),
            })
        })
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
    fn quarterly_expenses(&self, year: i32) -> Result<Vec<QuarterlyAmount>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s=c.prepare("SELECT ((CAST(SUBSTR(issue_date,6,2) AS INTEGER)-1)/3)+1,COALESCE(SUM(amount),0) FROM expenses WHERE SUBSTR(issue_date,1,4)=?1 AND deleted_at IS NULL GROUP BY 1 ORDER BY 1").map_err(|e|{log::error!("p: {e}");DomainError::InvalidInput})?;
        s.query_map(params![year.to_string()], |r| {
            Ok(QuarterlyAmount {
                quarter: r.get(0)?,
                amount: Amount::from_halere(r.get::<_, i64>(1)?),
            })
        })
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
    fn category_expenses(&self, year: i32) -> Result<Vec<CategoryAmount>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s=c.prepare("SELECT COALESCE(category,'other'),COALESCE(SUM(amount),0) FROM expenses WHERE SUBSTR(issue_date,1,4)=?1 AND deleted_at IS NULL GROUP BY 1 ORDER BY 2 DESC").map_err(|e|{log::error!("p: {e}");DomainError::InvalidInput})?;
        s.query_map(params![year.to_string()], |r| {
            Ok(CategoryAmount {
                category: r.get(0)?,
                amount: Amount::from_halere(r.get::<_, i64>(1)?),
            })
        })
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
    fn top_customers(&self, year: i32, limit: i32) -> Result<Vec<CustomerRevenue>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s=c.prepare("SELECT i.customer_id,COALESCE(c.name,''),SUM(i.total_amount),COUNT(*) FROM invoices i LEFT JOIN contacts c ON c.id=i.customer_id WHERE i.status='paid' AND SUBSTR(i.issue_date,1,4)=?1 AND i.deleted_at IS NULL GROUP BY i.customer_id ORDER BY 3 DESC LIMIT ?2").map_err(|e|{log::error!("p: {e}");DomainError::InvalidInput})?;
        s.query_map(params![year.to_string(), limit], |r| {
            Ok(CustomerRevenue {
                customer_id: r.get(0)?,
                customer_name: r.get(1)?,
                total: Amount::from_halere(r.get::<_, i64>(2)?),
                invoice_count: r.get(3)?,
            })
        })
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
    fn profit_loss_monthly(
        &self,
        year: i32,
    ) -> Result<(Vec<MonthlyAmount>, Vec<MonthlyAmount>), DomainError> {
        Ok((self.monthly_revenue(year)?, self.monthly_expenses(year)?))
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;
    #[test]
    fn test_empty() {
        let c = new_test_db();
        let r = SqliteReportRepo::new(c);
        assert_eq!(r.yearly_revenue(2025).unwrap(), Amount::ZERO);
    }
}
