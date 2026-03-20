use chrono::{NaiveDate, NaiveDateTime};
use std::collections::HashMap;

use crate::amount::Amount;
use crate::contact::Contact;

/// A business expense / received invoice.
#[derive(Debug, Clone)]
pub struct Expense {
    pub id: i64,
    pub vendor_id: Option<i64>,
    pub vendor: Option<Contact>,
    pub expense_number: String,
    pub category: String,
    pub description: String,

    pub issue_date: NaiveDate,
    pub amount: Amount,
    pub currency_code: String,
    pub exchange_rate: Amount,

    pub vat_rate_percent: i32,
    pub vat_amount: Amount,

    pub is_tax_deductible: bool,
    pub business_percent: i32,
    pub payment_method: String,

    pub document_path: String,
    pub notes: String,

    pub tax_reviewed_at: Option<NaiveDateTime>,

    // Line items
    pub items: Vec<ExpenseItem>,

    // Timestamps
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
    pub deleted_at: Option<NaiveDateTime>,
}

impl Expense {
    /// Recalculates Amount and VATAmount from expense items.
    /// When items is empty, leaves fields untouched (backward compat).
    pub fn calculate_totals(&mut self) {
        if self.items.is_empty() {
            return;
        }

        let mut subtotal = Amount::ZERO;
        let mut vat = Amount::ZERO;
        let mut total = Amount::ZERO;

        // Track which VAT rate has the highest subtotal share for dominant rate.
        let mut rate_subtotals: HashMap<i32, Amount> = HashMap::new();

        for item in &mut self.items {
            // item subtotal before VAT = quantity * unit_price / 100
            let item_subtotal =
                Amount::from_halere(item.quantity.halere() * item.unit_price.halere() / 100);
            let item_vat = item_subtotal.multiply(item.vat_rate_percent as f64 / 100.0);
            item.vat_amount = item_vat;
            item.total_amount = item_subtotal + item_vat;

            subtotal += item_subtotal;
            vat += item_vat;
            total += item.total_amount;

            let entry = rate_subtotals.entry(item.vat_rate_percent).or_default();
            *entry += item_subtotal;
        }

        self.amount = total;
        self.vat_amount = vat;

        // Set dominant VAT rate (rate with highest subtotal share).
        let mut max_subtotal = Amount::ZERO;
        for (&rate, &sub) in &rate_subtotals {
            if sub > max_subtotal {
                max_subtotal = sub;
                self.vat_rate_percent = rate;
            }
        }
    }
}

/// A single line item on an expense.
#[derive(Debug, Clone)]
pub struct ExpenseItem {
    pub id: i64,
    pub expense_id: i64,
    pub description: String,
    pub quantity: Amount,
    pub unit: String,
    pub unit_price: Amount,
    pub vat_rate_percent: i32,
    pub vat_amount: Amount,
    pub total_amount: Amount,
    pub sort_order: i32,
}

/// A file attachment linked to an expense.
#[derive(Debug, Clone)]
pub struct ExpenseDocument {
    pub id: i64,
    pub expense_id: i64,
    pub filename: String,
    pub content_type: String,
    pub storage_path: String,
    pub size: i64,
    pub created_at: NaiveDateTime,
    pub deleted_at: Option<NaiveDateTime>,
}

/// Filtering options for listing expenses.
#[derive(Debug, Clone, Default)]
pub struct ExpenseFilter {
    pub category: String,
    pub vendor_id: Option<i64>,
    pub date_from: Option<NaiveDate>,
    pub date_to: Option<NaiveDate>,
    pub search: String,
    pub tax_reviewed: Option<bool>,
    pub limit: i32,
    pub offset: i32,
}
