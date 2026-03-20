use std::sync::Arc;

use chrono::Datelike;
use zfaktury_domain::{Amount, DomainError};

use crate::repository::traits::DashboardRepo;
use crate::repository::types::{MonthlyAmount, RecentExpense, RecentInvoice};

/// Aggregated dashboard data.
pub struct DashboardData {
    pub revenue_current_month: Amount,
    pub expenses_current_month: Amount,
    pub unpaid_count: i64,
    pub unpaid_total: Amount,
    pub overdue_count: i64,
    pub overdue_total: Amount,
    pub monthly_revenue: Vec<MonthlyAmount>,
    pub monthly_expenses: Vec<MonthlyAmount>,
    pub recent_invoices: Vec<RecentInvoice>,
    pub recent_expenses: Vec<RecentExpense>,
}

/// Service for the dashboard view.
pub struct DashboardService {
    repo: Arc<dyn DashboardRepo + Send + Sync>,
}

impl DashboardService {
    pub fn new(repo: Arc<dyn DashboardRepo + Send + Sync>) -> Self {
        Self { repo }
    }

    pub fn get_dashboard(&self) -> Result<DashboardData, DomainError> {
        let now = chrono::Local::now();
        let year = now.date_naive().year();
        let month = now.date_naive().month() as i32;

        let revenue = self.repo.revenue_current_month(year, month)?;
        let expenses = self.repo.expenses_current_month(year, month)?;
        let (unpaid_count, unpaid_total) = self.repo.unpaid_invoices()?;
        let (overdue_count, overdue_total) = self.repo.overdue_invoices()?;
        let monthly_revenue = self.repo.monthly_revenue(year)?;
        let monthly_expenses = self.repo.monthly_expenses(year)?;
        let recent_invoices = self.repo.recent_invoices(5)?;
        let recent_expenses = self.repo.recent_expenses(5)?;

        Ok(DashboardData {
            revenue_current_month: revenue,
            expenses_current_month: expenses,
            unpaid_count,
            unpaid_total,
            overdue_count,
            overdue_total,
            monthly_revenue,
            monthly_expenses,
            recent_invoices,
            recent_expenses,
        })
    }
}
