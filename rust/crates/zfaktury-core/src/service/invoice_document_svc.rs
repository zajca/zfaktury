use std::sync::Arc;

use zfaktury_domain::{DomainError, InvoiceDocument};

use super::audit_svc::AuditService;
use crate::repository::traits::InvoiceDocumentRepo;

const MAX_DOCS_PER_INVOICE: i64 = 10;

/// Service for invoice document management.
pub struct InvoiceDocumentService {
    repo: Arc<dyn InvoiceDocumentRepo + Send + Sync>,
    data_dir: String,
    audit: Option<Arc<AuditService>>,
}

impl InvoiceDocumentService {
    pub fn new(
        repo: Arc<dyn InvoiceDocumentRepo + Send + Sync>,
        data_dir: String,
        audit: Option<Arc<AuditService>>,
    ) -> Self {
        Self {
            repo,
            data_dir,
            audit,
        }
    }

    pub fn data_dir(&self) -> &str {
        &self.data_dir
    }

    pub fn create_record(&self, doc: &mut InvoiceDocument) -> Result<(), DomainError> {
        if doc.invoice_id == 0 || doc.filename.is_empty() {
            return Err(DomainError::InvalidInput);
        }
        let count = self.repo.count_by_invoice_id(doc.invoice_id)?;
        if count >= MAX_DOCS_PER_INVOICE {
            return Err(DomainError::InvalidInput);
        }
        self.repo.create(doc)?;
        if let Some(ref audit) = self.audit {
            audit.log("invoice_document", doc.id, "create", None, None);
        }
        Ok(())
    }

    pub fn get_by_id(&self, id: i64) -> Result<InvoiceDocument, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)
    }

    pub fn list_by_invoice_id(&self, invoice_id: i64) -> Result<Vec<InvoiceDocument>, DomainError> {
        if invoice_id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.list_by_invoice_id(invoice_id)
    }

    pub fn delete(&self, id: i64) -> Result<(), DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)?;
        self.repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("invoice_document", id, "delete", None, None);
        }
        Ok(())
    }
}
