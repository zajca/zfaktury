use chrono::NaiveDateTime;

use crate::expense::{Expense, ExpenseDocument};
use crate::ocr::OCRResult;

/// Result of importing an expense from a document upload.
#[derive(Debug, Clone)]
pub struct ImportResult {
    pub expense: Expense,
    pub document: ExpenseDocument,
    pub ocr: Option<OCRResult>,
}

/// A single entity to be imported from Fakturoid.
#[derive(Debug, Clone)]
pub struct FakturoidImportItem {
    pub fakturoid_id: i64,
    /// "new", "duplicate", "conflict"
    pub status: String,
    /// Set when status is "duplicate".
    pub existing_id: Option<i64>,
    /// Human-readable explanation for status.
    pub reason: String,
}

/// Preview of what will be imported from Fakturoid.
#[derive(Debug, Clone)]
pub struct FakturoidImportPreview {
    pub contacts: Vec<FakturoidImportItem>,
    pub invoices: Vec<FakturoidImportItem>,
    pub expenses: Vec<FakturoidImportItem>,
}

/// Results of an import operation from Fakturoid.
#[derive(Debug, Clone)]
pub struct FakturoidImportResult {
    pub contacts_created: i32,
    pub contacts_skipped: i32,
    pub invoices_created: i32,
    pub invoices_skipped: i32,
    pub expenses_created: i32,
    pub expenses_skipped: i32,
    pub attachments_downloaded: i32,
    pub attachments_skipped: i32,
    pub errors: Vec<String>,
}

/// Record of an imported entity from Fakturoid.
#[derive(Debug, Clone)]
pub struct FakturoidImportLog {
    pub id: i64,
    pub fakturoid_entity_type: String,
    pub fakturoid_id: i64,
    pub local_entity_type: String,
    pub local_id: i64,
    pub imported_at: NaiveDateTime,
}
