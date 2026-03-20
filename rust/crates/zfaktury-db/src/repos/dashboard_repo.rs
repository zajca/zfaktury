use crate::helpers::*;
use rusqlite::{Connection, params};
use std::sync::Mutex;
use zfaktury_core::repository::traits::DashboardRepo;
use zfaktury_core::repository::types::*;
use zfaktury_domain::*;
pub struct SqliteDashboardRepo {
    conn: Mutex<Connection>,
}
impl SqliteDashboardRepo {
    pub fn new(conn: Connection) -> Self {
        Self {
            conn: Mutex::new(conn),
        }
    }
}
fn pis(s: &str) -> InvoiceStatus {
    match s {
        "sent" => InvoiceStatus::Sent,
        "paid" => InvoiceStatus::Paid,
        "overdue" => InvoiceStatus::Overdue,
        "cancelled" => InvoiceStatus::Cancelled,
        _ => InvoiceStatus::Draft,
    }
}
impl DashboardRepo for SqliteDashboardRepo {
    fn revenue_current_month(&self, year: i32, month: i32) -> Result<Amount, DomainError> {
        let c = self.conn.lock().unwrap();
        let prefix = format!("{year}-{month:02}");
        c.query_row("SELECT COALESCE(SUM(total_amount),0) FROM invoices WHERE status='paid' AND issue_date LIKE ?1||'%' AND deleted_at IS NULL",params![prefix],|r|Ok(Amount::from_halere(r.get::<_,i64>(0)?))).map_err(|e|{log::error!("rev: {e}");DomainError::InvalidInput})
    }
    fn expenses_current_month(&self, year: i32, month: i32) -> Result<Amount, DomainError> {
        let c = self.conn.lock().unwrap();
        let prefix = format!("{year}-{month:02}");
        c.query_row("SELECT COALESCE(SUM(amount),0) FROM expenses WHERE issue_date LIKE ?1||'%' AND deleted_at IS NULL",params![prefix],|r|Ok(Amount::from_halere(r.get::<_,i64>(0)?))).map_err(|e|{log::error!("exp: {e}");DomainError::InvalidInput})
    }
    fn unpaid_invoices(&self) -> Result<(i64, Amount), DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row("SELECT COUNT(*),COALESCE(SUM(total_amount-paid_amount),0) FROM invoices WHERE status IN ('sent','draft') AND deleted_at IS NULL",[],|r|Ok((r.get::<_,i64>(0)?,Amount::from_halere(r.get::<_,i64>(1)?)))).map_err(|e|{log::error!("unpaid: {e}");DomainError::InvalidInput})
    }
    fn overdue_invoices(&self) -> Result<(i64, Amount), DomainError> {
        let c = self.conn.lock().unwrap();
        c.query_row("SELECT COUNT(*),COALESCE(SUM(total_amount-paid_amount),0) FROM invoices WHERE status='overdue' AND deleted_at IS NULL",[],|r|Ok((r.get::<_,i64>(0)?,Amount::from_halere(r.get::<_,i64>(1)?)))).map_err(|e|{log::error!("overdue: {e}");DomainError::InvalidInput})
    }
    fn monthly_revenue(&self, year: i32) -> Result<Vec<MonthlyAmount>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s=c.prepare("SELECT CAST(SUBSTR(issue_date,6,2) AS INTEGER) AS m,COALESCE(SUM(total_amount),0) FROM invoices WHERE status='paid' AND SUBSTR(issue_date,1,4)=?1 AND deleted_at IS NULL GROUP BY m ORDER BY m").map_err(|e|{log::error!("p: {e}");DomainError::InvalidInput})?;
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
    fn monthly_expenses(&self, year: i32) -> Result<Vec<MonthlyAmount>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s=c.prepare("SELECT CAST(SUBSTR(issue_date,6,2) AS INTEGER) AS m,COALESCE(SUM(amount),0) FROM expenses WHERE SUBSTR(issue_date,1,4)=?1 AND deleted_at IS NULL GROUP BY m ORDER BY m").map_err(|e|{log::error!("p: {e}");DomainError::InvalidInput})?;
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
    fn recent_invoices(&self, limit: i32) -> Result<Vec<RecentInvoice>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s=c.prepare("SELECT i.id,i.invoice_number,COALESCE(c.name,'') AS customer_name,i.total_amount,i.status,i.issue_date FROM invoices i LEFT JOIN contacts c ON c.id=i.customer_id WHERE i.deleted_at IS NULL ORDER BY i.created_at DESC LIMIT ?1").map_err(|e|{log::error!("p: {e}");DomainError::InvalidInput})?;
        s.query_map(params![limit], |r| {
            let st: String = r.get("status")?;
            let id_str: String = r.get("issue_date")?;
            Ok(RecentInvoice {
                id: r.get("id")?,
                invoice_number: r.get("invoice_number")?,
                customer_name: r.get("customer_name")?,
                total_amount: Amount::from_halere(r.get::<_, i64>("total_amount")?),
                status: pis(&st),
                issue_date: parse_date_or_default(&id_str),
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
    fn recent_expenses(&self, limit: i32) -> Result<Vec<RecentExpense>, DomainError> {
        let c = self.conn.lock().unwrap();
        let mut s=c.prepare("SELECT id,description,COALESCE(category,'') AS category,amount,issue_date FROM expenses WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT ?1").map_err(|e|{log::error!("p: {e}");DomainError::InvalidInput})?;
        s.query_map(params![limit], |r| {
            let id_str: String = r.get("issue_date")?;
            Ok(RecentExpense {
                id: r.get("id")?,
                description: r.get("description")?,
                category: r.get("category")?,
                amount: Amount::from_halere(r.get::<_, i64>("amount")?),
                issue_date: parse_date_or_default(&id_str),
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
}
#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_db::new_test_db;
    #[test]
    fn test_empty_dashboard() {
        let c = new_test_db();
        let r = SqliteDashboardRepo::new(c);
        assert_eq!(r.revenue_current_month(2025, 1).unwrap(), Amount::ZERO);
        let (cnt, _) = r.unpaid_invoices().unwrap();
        assert_eq!(cnt, 0);
    }
}
