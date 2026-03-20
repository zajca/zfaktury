use chrono::NaiveDate;
use zfaktury_domain::{Amount, InvoiceStatus};

/// Monthly aggregated amount for dashboard/report charts.
pub struct MonthlyAmount {
    pub month: i32,
    pub amount: Amount,
}

/// Quarterly aggregated amount for report charts.
pub struct QuarterlyAmount {
    pub quarter: i32,
    pub amount: Amount,
}

/// Expense amount per category for report pie charts.
pub struct CategoryAmount {
    pub category: String,
    pub amount: Amount,
}

/// Top customer revenue for reports.
pub struct CustomerRevenue {
    pub customer_id: i64,
    pub customer_name: String,
    pub total: Amount,
    pub invoice_count: i32,
}

/// Recent invoice summary for the dashboard.
pub struct RecentInvoice {
    pub id: i64,
    pub invoice_number: String,
    pub customer_name: String,
    pub total_amount: Amount,
    pub status: InvoiceStatus,
    pub issue_date: NaiveDate,
}

/// Recent expense summary for the dashboard.
pub struct RecentExpense {
    pub id: i64,
    pub description: String,
    pub category: String,
    pub amount: Amount,
    pub issue_date: NaiveDate,
}
