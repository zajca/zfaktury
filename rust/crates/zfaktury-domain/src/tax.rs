use chrono::NaiveDateTime;
use std::fmt;

use crate::amount::Amount;

/// Amount threshold (in halere) for individual vs aggregated transactions
/// in control statements. 10,000 CZK = 1,000,000 haleru.
pub const CONTROL_STATEMENT_THRESHOLD: Amount = Amount::new(10_000, 0);

/// VIES service code for services (Article 196 directive).
pub const VIES_SERVICE_CODE_3: &str = "3";

/// Filing type for tax submissions.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum FilingType {
    Regular,
    Corrective,
    Supplementary,
}

impl fmt::Display for FilingType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            FilingType::Regular => write!(f, "regular"),
            FilingType::Corrective => write!(f, "corrective"),
            FilingType::Supplementary => write!(f, "supplementary"),
        }
    }
}

/// Filing status.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum FilingStatus {
    Draft,
    Ready,
    Filed,
}

impl fmt::Display for FilingStatus {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            FilingStatus::Draft => write!(f, "draft"),
            FilingStatus::Ready => write!(f, "ready"),
            FilingStatus::Filed => write!(f, "filed"),
        }
    }
}

/// Control statement section.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ControlSection {
    A4,
    A5,
    B2,
    B3,
}

impl fmt::Display for ControlSection {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ControlSection::A4 => write!(f, "A4"),
            ControlSection::A5 => write!(f, "A5"),
            ControlSection::B2 => write!(f, "B2"),
            ControlSection::B3 => write!(f, "B3"),
        }
    }
}

/// Identifies a tax reporting period.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TaxPeriod {
    pub year: i32,
    /// 1-12, 0 if quarterly.
    pub month: i32,
    /// 1-4, 0 if monthly.
    pub quarter: i32,
}

/// VAT return (Priznani k DPH).
#[derive(Debug, Clone)]
pub struct VATReturn {
    pub id: i64,
    pub period: TaxPeriod,
    pub filing_type: FilingType,

    // Output VAT (dan na vystupu) - standard rates
    pub output_vat_base_21: Amount,
    pub output_vat_amount_21: Amount,
    pub output_vat_base_12: Amount,
    pub output_vat_amount_12: Amount,
    pub output_vat_base_0: Amount,

    // Reverse charge output (preneseni danove povinnosti)
    pub reverse_charge_base_21: Amount,
    pub reverse_charge_amount_21: Amount,
    pub reverse_charge_base_12: Amount,
    pub reverse_charge_amount_12: Amount,

    // Input VAT (dan na vstupu)
    pub input_vat_base_21: Amount,
    pub input_vat_amount_21: Amount,
    pub input_vat_base_12: Amount,
    pub input_vat_amount_12: Amount,

    // Result
    pub total_output_vat: Amount,
    pub total_input_vat: Amount,
    pub net_vat: Amount,

    // XML blob for re-download
    pub xml_data: Vec<u8>,

    pub status: FilingStatus,
    pub filed_at: Option<NaiveDateTime>,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
}

/// Junction record linking a VAT return to an invoice.
#[derive(Debug, Clone)]
pub struct VATReturnInvoice {
    pub vat_return_id: i64,
    pub invoice_id: i64,
}

/// Junction record linking a VAT return to an expense.
#[derive(Debug, Clone)]
pub struct VATReturnExpense {
    pub vat_return_id: i64,
    pub expense_id: i64,
}

/// VAT control statement (Kontrolni hlaseni).
#[derive(Debug, Clone)]
pub struct VATControlStatement {
    pub id: i64,
    pub period: TaxPeriod,
    pub filing_type: FilingType,

    // XML blob for re-download
    pub xml_data: Vec<u8>,

    pub status: FilingStatus,
    pub filed_at: Option<NaiveDateTime>,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
}

/// A single line in a control statement.
#[derive(Debug, Clone)]
pub struct VATControlStatementLine {
    pub id: i64,
    pub control_statement_id: i64,
    pub section: ControlSection,
    pub partner_dic: String,
    pub document_number: String,
    /// Datum povinnosti priznat dan (YYYY-MM-DD).
    pub dppd: String,
    pub base: Amount,
    pub vat: Amount,
    pub vat_rate_percent: i32,
    pub invoice_id: Option<i64>,
    pub expense_id: Option<i64>,
}

/// VIES recapitulative statement (Souhrnne hlaseni).
#[derive(Debug, Clone)]
pub struct VIESSummary {
    pub id: i64,
    pub period: TaxPeriod,
    pub filing_type: FilingType,

    // XML blob for re-download
    pub xml_data: Vec<u8>,

    pub status: FilingStatus,
    pub filed_at: Option<NaiveDateTime>,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
}

/// A single line in a VIES summary (one per EU partner).
#[derive(Debug, Clone)]
pub struct VIESSummaryLine {
    pub id: i64,
    pub vies_summary_id: i64,
    pub partner_dic: String,
    /// 2-letter ISO country code (e.g. "DE", "SK").
    pub country_code: String,
    /// Base amount in CZK (no VAT for intra-EU).
    pub total_amount: Amount,
    /// Service code, typically "3" for services.
    pub service_code: String,
}

/// Per-year tax configuration such as flat rate percent.
#[derive(Debug, Clone)]
pub struct TaxYearSettings {
    pub year: i32,
    pub flat_rate_percent: i32,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
}

/// Monthly prepayment amounts for a given year.
#[derive(Debug, Clone)]
pub struct TaxPrepayment {
    pub year: i32,
    pub month: i32,
    pub tax_amount: Amount,
    pub social_amount: Amount,
    pub health_amount: Amount,
}
