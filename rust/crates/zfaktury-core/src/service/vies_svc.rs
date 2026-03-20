use std::sync::Arc;

use zfaktury_domain::{DomainError, FilingStatus, FilingType, VIESSummary, VIESSummaryLine};

use super::audit_svc::AuditService;
use crate::repository::traits::{ContactRepo, InvoiceRepo, VIESSummaryRepo};

/// Service for VIES recapitulative statement management.
pub struct VIESSummaryService {
    repo: Arc<dyn VIESSummaryRepo + Send + Sync>,
    invoices: Arc<dyn InvoiceRepo + Send + Sync>,
    contacts: Arc<dyn ContactRepo + Send + Sync>,
    audit: Option<Arc<AuditService>>,
}

impl VIESSummaryService {
    pub fn new(
        repo: Arc<dyn VIESSummaryRepo + Send + Sync>,
        invoices: Arc<dyn InvoiceRepo + Send + Sync>,
        contacts: Arc<dyn ContactRepo + Send + Sync>,
        audit: Option<Arc<AuditService>>,
    ) -> Self {
        Self {
            repo,
            invoices,
            contacts,
            audit,
        }
    }

    pub fn create(&self, vs: &mut VIESSummary) -> Result<(), DomainError> {
        if vs.period.year < 2000
            || vs.period.year > 2100
            || vs.period.quarter < 1
            || vs.period.quarter > 4
        {
            return Err(DomainError::InvalidInput);
        }
        if vs.filing_type == FilingType::Regular {
            if self
                .repo
                .get_by_period(
                    vs.period.year,
                    vs.period.quarter,
                    &vs.filing_type.to_string(),
                )
                .is_ok()
            {
                return Err(DomainError::DuplicateNumber);
            }
        }
        vs.status = FilingStatus::Draft;
        self.repo.create(vs)?;
        if let Some(ref audit) = self.audit {
            audit.log("vies_summary", vs.id, "create", None, None);
        }
        Ok(())
    }

    pub fn get_by_id(&self, id: i64) -> Result<VIESSummary, DomainError> {
        self.repo.get_by_id(id)
    }

    pub fn get_lines(&self, vies_summary_id: i64) -> Result<Vec<VIESSummaryLine>, DomainError> {
        self.repo.get_lines(vies_summary_id)
    }

    pub fn list(&self, year: i32) -> Result<Vec<VIESSummary>, DomainError> {
        self.repo.list(year)
    }

    pub fn delete(&self, id: i64) -> Result<(), DomainError> {
        let vs = self.repo.get_by_id(id)?;
        if vs.status == FilingStatus::Filed {
            return Err(DomainError::InvalidInput);
        }
        self.repo.delete_lines(id)?;
        self.repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("vies_summary", id, "delete", None, None);
        }
        Ok(())
    }

    pub fn mark_filed(&self, id: i64) -> Result<(), DomainError> {
        let mut vs = self.repo.get_by_id(id)?;
        if vs.status == FilingStatus::Filed {
            return Err(DomainError::InvalidInput);
        }
        vs.status = FilingStatus::Filed;
        vs.filed_at = Some(chrono::Local::now().naive_local());
        self.repo.update(&mut vs)?;
        if let Some(ref audit) = self.audit {
            audit.log("vies_summary", id, "mark_filed", None, None);
        }
        Ok(())
    }
}
