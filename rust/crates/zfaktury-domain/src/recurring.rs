use chrono::{Datelike, NaiveDate, NaiveDateTime};
use std::fmt;

use crate::amount::Amount;
use crate::contact::Contact;

/// Frequency for recurring invoices/expenses.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Frequency {
    Weekly,
    Monthly,
    Quarterly,
    Yearly,
}

impl fmt::Display for Frequency {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Frequency::Weekly => write!(f, "weekly"),
            Frequency::Monthly => write!(f, "monthly"),
            Frequency::Quarterly => write!(f, "quarterly"),
            Frequency::Yearly => write!(f, "yearly"),
        }
    }
}

/// Calculates the next date based on frequency.
fn advance_date(date: NaiveDate, frequency: &Frequency) -> NaiveDate {
    match frequency {
        Frequency::Weekly => date + chrono::Days::new(7),
        Frequency::Monthly => add_months(date, 1),
        Frequency::Quarterly => add_months(date, 3),
        Frequency::Yearly => add_months(date, 12),
    }
}

/// Adds N months to a date, clamping to end of month if needed.
fn add_months(date: NaiveDate, months: u32) -> NaiveDate {
    let month0 = date.month0() + months;
    let year = date.year() + (month0 / 12) as i32;
    let month = (month0 % 12) + 1;

    // Clamp day to the last day of the target month.
    let max_day = days_in_month(year, month);
    let day = date.day().min(max_day);

    NaiveDate::from_ymd_opt(year, month, day).unwrap()
}

fn days_in_month(year: i32, month: u32) -> u32 {
    // Next month's first day minus one day.
    if month == 12 {
        NaiveDate::from_ymd_opt(year + 1, 1, 1)
    } else {
        NaiveDate::from_ymd_opt(year, month + 1, 1)
    }
    .unwrap()
    .pred_opt()
    .unwrap()
    .day()
}

/// A template for automatically generated invoices.
#[derive(Debug, Clone)]
pub struct RecurringInvoice {
    pub id: i64,
    pub name: String,
    pub customer_id: i64,
    pub customer: Option<Contact>,
    pub frequency: Frequency,
    pub next_issue_date: NaiveDate,
    pub end_date: Option<NaiveDate>,
    pub currency_code: String,
    pub exchange_rate: Amount,
    pub payment_method: String,
    pub bank_account: String,
    pub bank_code: String,
    pub iban: String,
    pub swift: String,
    pub constant_symbol: String,
    pub notes: String,
    pub is_active: bool,
    pub items: Vec<RecurringInvoiceItem>,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
    pub deleted_at: Option<NaiveDateTime>,
}

impl RecurringInvoice {
    /// Calculates the next issue date based on frequency.
    pub fn next_date(&self) -> NaiveDate {
        advance_date(self.next_issue_date, &self.frequency)
    }
}

/// A line item on a recurring invoice template.
#[derive(Debug, Clone)]
pub struct RecurringInvoiceItem {
    pub id: i64,
    pub recurring_invoice_id: i64,
    pub description: String,
    pub quantity: Amount,
    pub unit: String,
    pub unit_price: Amount,
    pub vat_rate_percent: i32,
    pub sort_order: i32,
}

/// A template for automatically generated expenses.
#[derive(Debug, Clone)]
pub struct RecurringExpense {
    pub id: i64,
    pub name: String,
    pub vendor_id: Option<i64>,
    pub vendor: Option<Contact>,
    pub category: String,
    pub description: String,
    pub amount: Amount,
    pub currency_code: String,
    pub exchange_rate: Amount,
    pub vat_rate_percent: i32,
    pub vat_amount: Amount,
    pub is_tax_deductible: bool,
    pub business_percent: i32,
    pub payment_method: String,
    pub notes: String,
    pub frequency: Frequency,
    pub next_issue_date: NaiveDate,
    pub end_date: Option<NaiveDate>,
    pub is_active: bool,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
    pub deleted_at: Option<NaiveDateTime>,
}

impl RecurringExpense {
    /// Calculates the next issue date based on frequency.
    pub fn next_date(&self) -> NaiveDate {
        advance_date(self.next_issue_date, &self.frequency)
    }
}
