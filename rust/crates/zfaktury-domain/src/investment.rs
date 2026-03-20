use chrono::{NaiveDate, NaiveDateTime};
use std::fmt;

use crate::amount::Amount;

/// Investment document platform.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Platform {
    Portu,
    Zonky,
    Trading212,
    Revolut,
    Other,
}

impl fmt::Display for Platform {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Platform::Portu => write!(f, "portu"),
            Platform::Zonky => write!(f, "zonky"),
            Platform::Trading212 => write!(f, "trading212"),
            Platform::Revolut => write!(f, "revolut"),
            Platform::Other => write!(f, "other"),
        }
    }
}

/// Extraction status for investment documents.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ExtractionStatus {
    Pending,
    Extracted,
    Failed,
}

impl fmt::Display for ExtractionStatus {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ExtractionStatus::Pending => write!(f, "pending"),
            ExtractionStatus::Extracted => write!(f, "extracted"),
            ExtractionStatus::Failed => write!(f, "failed"),
        }
    }
}

/// Capital income category.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum CapitalCategory {
    DividendCZ,
    DividendForeign,
    Interest,
    Coupon,
    FundDistribution,
    Other,
}

impl fmt::Display for CapitalCategory {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            CapitalCategory::DividendCZ => write!(f, "dividend_cz"),
            CapitalCategory::DividendForeign => write!(f, "dividend_foreign"),
            CapitalCategory::Interest => write!(f, "interest"),
            CapitalCategory::Coupon => write!(f, "coupon"),
            CapitalCategory::FundDistribution => write!(f, "fund_distribution"),
            CapitalCategory::Other => write!(f, "other"),
        }
    }
}

/// Asset type for security transactions.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum AssetType {
    Stock,
    ETF,
    Bond,
    Fund,
    Crypto,
    Other,
}

impl fmt::Display for AssetType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            AssetType::Stock => write!(f, "stock"),
            AssetType::ETF => write!(f, "etf"),
            AssetType::Bond => write!(f, "bond"),
            AssetType::Fund => write!(f, "fund"),
            AssetType::Crypto => write!(f, "crypto"),
            AssetType::Other => write!(f, "other"),
        }
    }
}

/// Transaction type for security transactions.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum TransactionType {
    Buy,
    Sell,
}

impl fmt::Display for TransactionType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            TransactionType::Buy => write!(f, "buy"),
            TransactionType::Sell => write!(f, "sell"),
        }
    }
}

/// An uploaded broker statement.
#[derive(Debug, Clone)]
pub struct InvestmentDocument {
    pub id: i64,
    pub year: i32,
    pub platform: Platform,
    pub filename: String,
    pub content_type: String,
    pub storage_path: String,
    pub size: i64,
    pub extraction_status: ExtractionStatus,
    pub extraction_error: String,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
}

/// A single section 8 income entry.
#[derive(Debug, Clone)]
pub struct CapitalIncomeEntry {
    pub id: i64,
    pub year: i32,
    pub document_id: Option<i64>,
    pub category: CapitalCategory,
    pub description: String,
    pub income_date: NaiveDate,
    pub gross_amount: Amount,
    pub withheld_tax_cz: Amount,
    pub withheld_tax_foreign: Amount,
    pub country_code: String,
    pub needs_declaring: bool,
    pub net_amount: Amount,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
}

/// A single buy or sell transaction (section 10).
#[derive(Debug, Clone)]
pub struct SecurityTransaction {
    pub id: i64,
    pub year: i32,
    pub document_id: Option<i64>,
    pub asset_type: AssetType,
    pub asset_name: String,
    pub isin: String,
    pub transaction_type: TransactionType,
    pub transaction_date: NaiveDate,
    /// Quantity in 1/10000 units (1 share = 10000).
    pub quantity: i64,
    pub unit_price: Amount,
    pub total_amount: Amount,
    pub fees: Amount,
    pub currency_code: String,
    /// Exchange rate * 10000 (for precision).
    pub exchange_rate: i64,
    pub cost_basis: Amount,
    pub computed_gain: Amount,
    pub time_test_exempt: bool,
    pub exempt_amount: Amount,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
}

/// Aggregated investment income for a year.
#[derive(Debug, Clone)]
pub struct InvestmentYearSummary {
    pub year: i32,
    // Section 8 capital income
    pub capital_income_gross: Amount,
    pub capital_income_tax: Amount,
    pub capital_income_net: Amount,
    // Section 10 other income
    pub other_income_gross: Amount,
    pub other_income_expenses: Amount,
    pub other_income_exempt: Amount,
    pub other_income_net: Amount,
}
