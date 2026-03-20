use std::sync::Arc;

use zfaktury_domain::{DomainError, TaxDeductionDocument};

use super::audit_svc::AuditService;
use crate::repository::traits::{TaxDeductionDocumentRepo, TaxDeductionRepo};

/// Service for tax deduction document management.
pub struct TaxDeductionDocumentService {
    repo: Arc<dyn TaxDeductionDocumentRepo + Send + Sync>,
    deduction_repo: Arc<dyn TaxDeductionRepo + Send + Sync>,
    data_dir: String,
    audit: Option<Arc<AuditService>>,
}

impl TaxDeductionDocumentService {
    pub fn new(
        repo: Arc<dyn TaxDeductionDocumentRepo + Send + Sync>,
        deduction_repo: Arc<dyn TaxDeductionRepo + Send + Sync>,
        data_dir: String,
        audit: Option<Arc<AuditService>>,
    ) -> Self {
        Self {
            repo,
            deduction_repo,
            data_dir,
            audit,
        }
    }

    pub fn data_dir(&self) -> &str {
        &self.data_dir
    }

    pub fn create_record(&self, doc: &mut TaxDeductionDocument) -> Result<(), DomainError> {
        if doc.tax_deduction_id == 0 || doc.filename.is_empty() {
            return Err(DomainError::InvalidInput);
        }
        self.deduction_repo.get_by_id(doc.tax_deduction_id)?; // verify exists
        self.repo.create(doc)?;
        if let Some(ref audit) = self.audit {
            audit.log("tax_deduction_document", doc.id, "create", None, None);
        }
        Ok(())
    }

    pub fn get_by_id(&self, id: i64) -> Result<TaxDeductionDocument, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)
    }

    pub fn list_by_deduction_id(
        &self,
        deduction_id: i64,
    ) -> Result<Vec<TaxDeductionDocument>, DomainError> {
        if deduction_id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.list_by_deduction_id(deduction_id)
    }

    pub fn delete(&self, id: i64) -> Result<(), DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)?;
        self.repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("tax_deduction_document", id, "delete", None, None);
        }
        Ok(())
    }
}
