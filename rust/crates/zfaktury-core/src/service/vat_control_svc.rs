use std::sync::Arc;

use zfaktury_domain::{
    DomainError, FilingStatus, FilingType, VATControlStatement, VATControlStatementLine,
};

use super::audit_svc::AuditService;
use crate::repository::traits::{ContactRepo, ExpenseRepo, InvoiceRepo, VATControlStatementRepo};

/// Service for VAT control statement management.
pub struct VATControlStatementService {
    repo: Arc<dyn VATControlStatementRepo + Send + Sync>,
    invoices: Arc<dyn InvoiceRepo + Send + Sync>,
    expenses: Arc<dyn ExpenseRepo + Send + Sync>,
    contacts: Arc<dyn ContactRepo + Send + Sync>,
    audit: Option<Arc<AuditService>>,
}

impl VATControlStatementService {
    pub fn new(
        repo: Arc<dyn VATControlStatementRepo + Send + Sync>,
        invoices: Arc<dyn InvoiceRepo + Send + Sync>,
        expenses: Arc<dyn ExpenseRepo + Send + Sync>,
        contacts: Arc<dyn ContactRepo + Send + Sync>,
        audit: Option<Arc<AuditService>>,
    ) -> Self {
        Self {
            repo,
            invoices,
            expenses,
            contacts,
            audit,
        }
    }

    pub fn create(&self, cs: &mut VATControlStatement) -> Result<(), DomainError> {
        if cs.period.year < 2000
            || cs.period.year > 2100
            || cs.period.month < 1
            || cs.period.month > 12
        {
            return Err(DomainError::InvalidInput);
        }
        if cs.filing_type == FilingType::Regular
            && self
                .repo
                .get_by_period(cs.period.year, cs.period.month, &cs.filing_type.to_string())
                .is_ok()
        {
            return Err(DomainError::DuplicateNumber);
        }
        cs.status = FilingStatus::Draft;
        self.repo.create(cs)?;
        if let Some(ref audit) = self.audit {
            audit.log("vat_control_statement", cs.id, "create", None, None);
        }
        Ok(())
    }

    pub fn get_by_id(&self, id: i64) -> Result<VATControlStatement, DomainError> {
        self.repo.get_by_id(id)
    }

    pub fn list(&self, year: i32) -> Result<Vec<VATControlStatement>, DomainError> {
        self.repo.list(year)
    }

    pub fn delete(&self, id: i64) -> Result<(), DomainError> {
        let cs = self.repo.get_by_id(id)?;
        if cs.status == FilingStatus::Filed {
            return Err(DomainError::InvalidInput);
        }
        self.repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("vat_control_statement", id, "delete", None, None);
        }
        Ok(())
    }

    pub fn get_lines(
        &self,
        control_statement_id: i64,
    ) -> Result<Vec<VATControlStatementLine>, DomainError> {
        self.repo.get_lines(control_statement_id)
    }

    pub fn mark_filed(&self, id: i64) -> Result<(), DomainError> {
        let mut cs = self.repo.get_by_id(id)?;
        cs.status = FilingStatus::Filed;
        cs.filed_at = Some(chrono::Local::now().naive_local());
        self.repo.update(&mut cs)?;
        if let Some(ref audit) = self.audit {
            audit.log("vat_control_statement", id, "mark_filed", None, None);
        }
        Ok(())
    }
}
