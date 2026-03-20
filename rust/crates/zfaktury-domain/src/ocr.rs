use crate::amount::Amount;

/// Structured data extracted from a document scan.
#[derive(Debug, Clone)]
pub struct OCRResult {
    pub vendor_name: String,
    pub vendor_ico: String,
    pub vendor_dic: String,
    pub invoice_number: String,
    /// YYYY-MM-DD format.
    pub issue_date: String,
    /// YYYY-MM-DD format.
    pub due_date: String,
    pub total_amount: Amount,
    pub vat_amount: Amount,
    pub vat_rate_percent: i32,
    pub currency_code: String,
    pub description: String,
    pub items: Vec<OCRItem>,
    pub raw_text: String,
    pub confidence: f64,
}

/// A single line item extracted from a document.
#[derive(Debug, Clone)]
pub struct OCRItem {
    pub description: String,
    pub quantity: Amount,
    pub unit_price: Amount,
    pub vat_rate_percent: i32,
    pub total_amount: Amount,
}
