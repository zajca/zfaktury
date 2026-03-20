use std::sync::Arc;

use zfaktury_domain::{Amount, DomainError};

use crate::repository::traits::ReportRepo;
use crate::repository::types::{CategoryAmount, CustomerRevenue, MonthlyAmount, QuarterlyAmount};

/// Revenue report data.
pub struct RevenueReport {
    pub year: i32,
    pub monthly: Vec<MonthlyAmount>,
    pub quarterly: Vec<QuarterlyAmount>,
    pub total: Amount,
}

/// Expense report data.
pub struct ExpenseReport {
    pub year: i32,
    pub monthly: Vec<MonthlyAmount>,
    pub quarterly: Vec<QuarterlyAmount>,
    pub categories: Vec<CategoryAmount>,
}

/// Profit/loss report data.
pub struct ProfitLossReport {
    pub year: i32,
    pub monthly_revenue: Vec<MonthlyAmount>,
    pub monthly_expenses: Vec<MonthlyAmount>,
}

/// Service for report generation.
pub struct ReportService {
    repo: Arc<dyn ReportRepo + Send + Sync>,
}

impl ReportService {
    pub fn new(repo: Arc<dyn ReportRepo + Send + Sync>) -> Self {
        Self { repo }
    }

    pub fn revenue_report(&self, year: i32) -> Result<RevenueReport, DomainError> {
        let monthly = self.repo.monthly_revenue(year)?;
        let quarterly = self.repo.quarterly_revenue(year)?;
        let total = self.repo.yearly_revenue(year)?;
        Ok(RevenueReport {
            year,
            monthly,
            quarterly,
            total,
        })
    }

    pub fn expense_report(&self, year: i32) -> Result<ExpenseReport, DomainError> {
        let monthly = self.repo.monthly_expenses(year)?;
        let quarterly = self.repo.quarterly_expenses(year)?;
        let categories = self.repo.category_expenses(year)?;
        Ok(ExpenseReport {
            year,
            monthly,
            quarterly,
            categories,
        })
    }

    pub fn top_customers(&self, year: i32) -> Result<Vec<CustomerRevenue>, DomainError> {
        self.repo.top_customers(year, 10)
    }

    pub fn profit_loss(&self, year: i32) -> Result<ProfitLossReport, DomainError> {
        let (revenue, expenses) = self.repo.profit_loss_monthly(year)?;
        Ok(ProfitLossReport {
            year,
            monthly_revenue: revenue,
            monthly_expenses: expenses,
        })
    }
}
