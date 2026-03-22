use std::sync::Arc;

use zfaktury_domain::{DomainError, InvestmentDocument};

use super::audit_svc::AuditService;
use crate::repository::traits::{
    CapitalIncomeRepo, InvestmentDocumentRepo, SecurityTransactionRepo,
};

/// Service for investment document management (broker statements).
pub struct InvestmentDocumentService {
    repo: Arc<dyn InvestmentDocumentRepo + Send + Sync>,
    capital_repo: Arc<dyn CapitalIncomeRepo + Send + Sync>,
    security_repo: Arc<dyn SecurityTransactionRepo + Send + Sync>,
    data_dir: String,
    audit: Option<Arc<AuditService>>,
}

impl InvestmentDocumentService {
    pub fn new(
        repo: Arc<dyn InvestmentDocumentRepo + Send + Sync>,
        capital_repo: Arc<dyn CapitalIncomeRepo + Send + Sync>,
        security_repo: Arc<dyn SecurityTransactionRepo + Send + Sync>,
        data_dir: String,
        audit: Option<Arc<AuditService>>,
    ) -> Self {
        Self {
            repo,
            capital_repo,
            security_repo,
            data_dir,
            audit,
        }
    }

    pub fn data_dir(&self) -> &str {
        &self.data_dir
    }

    pub fn create_record(&self, doc: &mut InvestmentDocument) -> Result<(), DomainError> {
        if doc.year < 2000 || doc.year > 2100 || doc.filename.is_empty() {
            return Err(DomainError::InvalidInput);
        }
        self.repo.create(doc)?;
        if let Some(ref audit) = self.audit {
            audit.log("investment_document", doc.id, "create", None, None);
        }
        Ok(())
    }

    pub fn get_by_id(&self, id: i64) -> Result<InvestmentDocument, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)
    }

    pub fn list_by_year(&self, year: i32) -> Result<Vec<InvestmentDocument>, DomainError> {
        self.repo.list_by_year(year)
    }

    /// Updates the extraction status and error message for a document.
    pub fn update_extraction_status(
        &self,
        id: i64,
        status: &str,
        extraction_error: &str,
    ) -> Result<(), DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.update_extraction(id, status, extraction_error)?;
        if let Some(ref audit) = self.audit {
            audit.log("investment_document", id, "update_extraction", None, None);
        }
        Ok(())
    }

    /// Deletes the document and all linked capital income entries and security transactions.
    pub fn delete(&self, id: i64) -> Result<(), DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)?;
        self.capital_repo.delete_by_document_id(id)?;
        self.security_repo.delete_by_document_id(id)?;
        self.repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("investment_document", id, "delete", None, None);
        }
        Ok(())
    }
}
