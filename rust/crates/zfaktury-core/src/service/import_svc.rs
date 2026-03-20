use std::sync::Arc;

use zfaktury_domain::{Amount, CURRENCY_CZK, DomainError, Expense};

use super::document_svc::DocumentService;
use super::expense_svc::ExpenseService;
use super::ocr_svc::OCRService;

/// Service for importing expenses from uploaded documents.
pub struct ImportService {
    expenses: Arc<ExpenseService>,
    documents: Arc<DocumentService>,
    ocr: Option<Arc<OCRService>>,
}

/// Result of an import operation.
pub struct ImportResult {
    pub expense: Expense,
}

impl ImportService {
    pub fn new(
        expenses: Arc<ExpenseService>,
        documents: Arc<DocumentService>,
        ocr: Option<Arc<OCRService>>,
    ) -> Self {
        Self {
            expenses,
            documents,
            ocr,
        }
    }

    /// Creates a skeleton expense for an imported document.
    /// The actual file upload is handled by the caller/handler layer.
    pub fn create_skeleton_expense(&self, filename: &str) -> Result<Expense, DomainError> {
        let now = chrono::Local::now().naive_local();
        let today = chrono::Local::now().date_naive();
        let mut expense = Expense {
            id: 0,
            vendor_id: None,
            vendor: None,
            expense_number: String::new(),
            category: String::new(),
            description: filename.to_string(),
            issue_date: today,
            amount: Amount::new(1, 0), // 1 CZK minimum placeholder
            currency_code: CURRENCY_CZK.to_string(),
            exchange_rate: Amount::ZERO,
            vat_rate_percent: 0,
            vat_amount: Amount::ZERO,
            is_tax_deductible: false,
            business_percent: 100,
            payment_method: "bank_transfer".to_string(),
            document_path: String::new(),
            notes: String::new(),
            tax_reviewed_at: None,
            items: Vec::new(),
            created_at: now,
            updated_at: now,
            deleted_at: None,
        };

        self.expenses.create(&mut expense)?;
        Ok(expense)
    }
}
