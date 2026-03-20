use chrono::{NaiveDate, NaiveDateTime};
use std::fmt;

use crate::amount::Amount;
use crate::contact::Contact;

/// Invoice type enum.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum InvoiceType {
    Regular,
    Proforma,
    CreditNote,
}

impl fmt::Display for InvoiceType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            InvoiceType::Regular => write!(f, "regular"),
            InvoiceType::Proforma => write!(f, "proforma"),
            InvoiceType::CreditNote => write!(f, "credit_note"),
        }
    }
}

/// Invoice status enum.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum InvoiceStatus {
    Draft,
    Sent,
    Paid,
    Overdue,
    Cancelled,
}

impl fmt::Display for InvoiceStatus {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            InvoiceStatus::Draft => write!(f, "draft"),
            InvoiceStatus::Sent => write!(f, "sent"),
            InvoiceStatus::Paid => write!(f, "paid"),
            InvoiceStatus::Overdue => write!(f, "overdue"),
            InvoiceStatus::Cancelled => write!(f, "cancelled"),
        }
    }
}

/// Invoice relation type enum.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum RelationType {
    None,
    Settlement,
    CreditNote,
}

impl fmt::Display for RelationType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            RelationType::None => write!(f, ""),
            RelationType::Settlement => write!(f, "settlement"),
            RelationType::CreditNote => write!(f, "credit_note"),
        }
    }
}

/// An issued invoice.
#[derive(Debug, Clone)]
pub struct Invoice {
    pub id: i64,
    pub sequence_id: i64,
    pub invoice_number: String,
    pub invoice_type: InvoiceType,
    pub status: InvoiceStatus,

    pub issue_date: NaiveDate,
    pub due_date: NaiveDate,
    pub delivery_date: NaiveDate,
    pub variable_symbol: String,
    pub constant_symbol: String,

    // Customer
    pub customer_id: i64,
    pub customer: Option<Contact>,

    // Currency
    pub currency_code: String,
    pub exchange_rate: Amount,

    // Payment
    pub payment_method: String,
    pub bank_account: String,
    pub bank_code: String,
    pub iban: String,
    pub swift: String,

    // Amounts
    pub subtotal_amount: Amount,
    pub vat_amount: Amount,
    pub total_amount: Amount,
    pub paid_amount: Amount,

    // Notes
    pub notes: String,
    pub internal_notes: String,

    // Related invoice (for credit notes, settlements)
    pub related_invoice_id: Option<i64>,
    pub relation_type: RelationType,

    // Event timestamps
    pub sent_at: Option<NaiveDateTime>,
    pub paid_at: Option<NaiveDateTime>,

    // Line items
    pub items: Vec<InvoiceItem>,

    // Timestamps
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
    pub deleted_at: Option<NaiveDateTime>,
}

impl Invoice {
    /// Recalculates subtotal, VAT, and total from invoice items.
    pub fn calculate_totals(&mut self) {
        let mut subtotal = Amount::ZERO;
        let mut vat = Amount::ZERO;
        let mut total = Amount::ZERO;

        for item in &mut self.items {
            // item subtotal before VAT = quantity * unit_price / 100
            // (quantity is in cents, so we divide by 100)
            let item_subtotal =
                Amount::from_halere(item.quantity.halere() * item.unit_price.halere() / 100);
            let item_vat = item_subtotal.multiply(item.vat_rate_percent as f64 / 100.0);
            item.vat_amount = item_vat;
            item.total_amount = item_subtotal + item_vat;

            subtotal += item_subtotal;
            vat += item_vat;
            total += item.total_amount;
        }

        self.subtotal_amount = subtotal;
        self.vat_amount = vat;
        self.total_amount = total;
    }

    /// Returns true if the invoice is past due and not yet fully paid.
    pub fn is_overdue(&self, today: NaiveDate) -> bool {
        if self.status == InvoiceStatus::Paid || self.status == InvoiceStatus::Cancelled {
            return false;
        }
        today > self.due_date
    }

    /// Returns true if the invoice has been fully paid.
    pub fn is_paid(&self) -> bool {
        self.paid_amount >= self.total_amount && self.total_amount > Amount::ZERO
    }
}

/// A single line item on an invoice.
#[derive(Debug, Clone)]
pub struct InvoiceItem {
    pub id: i64,
    pub invoice_id: i64,
    pub description: String,
    pub quantity: Amount,
    pub unit: String,
    pub unit_price: Amount,
    pub vat_rate_percent: i32,
    pub vat_amount: Amount,
    pub total_amount: Amount,
    pub sort_order: i32,
}

/// A numbering sequence for invoices.
#[derive(Debug, Clone)]
pub struct InvoiceSequence {
    pub id: i64,
    pub prefix: String,
    pub next_number: i32,
    pub year: i32,
    pub format_pattern: String,
}

/// Filtering options for listing invoices.
#[derive(Debug, Clone, Default)]
pub struct InvoiceFilter {
    pub status: Option<InvoiceStatus>,
    pub invoice_type: Option<InvoiceType>,
    pub customer_id: Option<i64>,
    pub date_from: Option<NaiveDate>,
    pub date_to: Option<NaiveDate>,
    pub search: String,
    pub limit: i32,
    pub offset: i32,
}

/// A recorded change of an invoice's status.
#[derive(Debug, Clone)]
pub struct InvoiceStatusChange {
    pub id: i64,
    pub invoice_id: i64,
    pub old_status: InvoiceStatus,
    pub new_status: InvoiceStatus,
    pub changed_at: NaiveDateTime,
    pub note: String,
}

/// A file attachment linked to an invoice.
#[derive(Debug, Clone)]
pub struct InvoiceDocument {
    pub id: i64,
    pub invoice_id: i64,
    pub filename: String,
    pub content_type: String,
    pub storage_path: String,
    pub size: i64,
    pub created_at: NaiveDateTime,
    pub deleted_at: Option<NaiveDateTime>,
}
