use std::sync::Arc;

use zfaktury_domain::{DomainError, ExpenseDocument};

use super::audit_svc::AuditService;
use crate::repository::traits::DocumentRepo;

const MAX_DOCS_PER_EXPENSE: i64 = 10;

/// Service for expense document management (file upload/download/delete).
pub struct DocumentService {
    repo: Arc<dyn DocumentRepo + Send + Sync>,
    data_dir: String,
    audit: Option<Arc<AuditService>>,
}

impl DocumentService {
    pub fn new(
        repo: Arc<dyn DocumentRepo + Send + Sync>,
        data_dir: String,
        audit: Option<Arc<AuditService>>,
    ) -> Self {
        Self {
            repo,
            data_dir,
            audit,
        }
    }

    /// Returns the data directory for document storage.
    pub fn data_dir(&self) -> &str {
        &self.data_dir
    }

    /// Registers a new document record for an expense.
    /// File I/O (writing bytes to disk) is handled by the caller/handler layer.
    pub fn create_record(&self, doc: &mut ExpenseDocument) -> Result<(), DomainError> {
        if doc.expense_id == 0 {
            return Err(DomainError::InvalidInput);
        }
        if doc.filename.is_empty() {
            return Err(DomainError::InvalidInput);
        }

        let count = self.repo.count_by_expense_id(doc.expense_id)?;
        if count >= MAX_DOCS_PER_EXPENSE {
            return Err(DomainError::InvalidInput);
        }

        self.repo.create(doc)?;
        if let Some(ref audit) = self.audit {
            audit.log("document", doc.id, "create", None, None);
        }
        Ok(())
    }

    pub fn get_by_id(&self, id: i64) -> Result<ExpenseDocument, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)
    }

    pub fn list_by_expense_id(&self, expense_id: i64) -> Result<Vec<ExpenseDocument>, DomainError> {
        if expense_id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.list_by_expense_id(expense_id)
    }

    pub fn delete(&self, id: i64) -> Result<(), DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)?; // verify exists
        self.repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("document", id, "delete", None, None);
        }
        Ok(())
    }
}
