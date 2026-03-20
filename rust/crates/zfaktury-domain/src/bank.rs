use chrono::{NaiveDate, NaiveDateTime};

use crate::amount::Amount;

/// An imported bank transaction.
#[derive(Debug, Clone)]
pub struct BankTransaction {
    pub id: i64,
    pub bank_account: String,
    pub transaction_date: NaiveDate,
    pub amount: Amount,
    pub currency: String,
    pub counterparty_account: String,
    pub counterparty_name: String,
    pub variable_symbol: String,
    pub constant_symbol: String,
    pub specific_symbol: String,
    pub message: String,
    pub invoice_id: Option<i64>,
    pub matched: bool,
    pub created_at: NaiveDateTime,
}
