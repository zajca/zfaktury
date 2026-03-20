use std::sync::Arc;

use zfaktury_domain::{DomainError, FilingStatus, FilingType, IncomeTaxReturn};

use super::audit_svc::AuditService;
use crate::repository::traits::{
    ExpenseRepo, IncomeTaxReturnRepo, InvoiceRepo, SettingsRepo, TaxPrepaymentRepo,
    TaxYearSettingsRepo,
};

/// Service for income tax return management.
pub struct IncomeTaxReturnService {
    repo: Arc<dyn IncomeTaxReturnRepo + Send + Sync>,
    invoice_repo: Arc<dyn InvoiceRepo + Send + Sync>,
    expense_repo: Arc<dyn ExpenseRepo + Send + Sync>,
    settings_repo: Arc<dyn SettingsRepo + Send + Sync>,
    tax_year_settings_repo: Arc<dyn TaxYearSettingsRepo + Send + Sync>,
    tax_prepayment_repo: Arc<dyn TaxPrepaymentRepo + Send + Sync>,
    audit: Option<Arc<AuditService>>,
}

impl IncomeTaxReturnService {
    pub fn new(
        repo: Arc<dyn IncomeTaxReturnRepo + Send + Sync>,
        invoice_repo: Arc<dyn InvoiceRepo + Send + Sync>,
        expense_repo: Arc<dyn ExpenseRepo + Send + Sync>,
        settings_repo: Arc<dyn SettingsRepo + Send + Sync>,
        tax_year_settings_repo: Arc<dyn TaxYearSettingsRepo + Send + Sync>,
        tax_prepayment_repo: Arc<dyn TaxPrepaymentRepo + Send + Sync>,
        audit: Option<Arc<AuditService>>,
    ) -> Self {
        Self {
            repo,
            invoice_repo,
            expense_repo,
            settings_repo,
            tax_year_settings_repo,
            tax_prepayment_repo,
            audit,
        }
    }

    pub fn create(&self, itr: &mut IncomeTaxReturn) -> Result<(), DomainError> {
        if itr.year < 2000 || itr.year > 2100 {
            return Err(DomainError::InvalidInput);
        }
        if itr.filing_type == FilingType::Regular {
            if self
                .repo
                .get_by_year(itr.year, &itr.filing_type.to_string())
                .is_ok()
            {
                return Err(DomainError::FilingAlreadyExists);
            }
        }
        if itr.status == FilingStatus::Draft { /* default */ }
        self.repo.create(itr)?;
        if let Some(ref audit) = self.audit {
            audit.log("income_tax_return", itr.id, "create", None, None);
        }
        Ok(())
    }

    pub fn get_by_id(&self, id: i64) -> Result<IncomeTaxReturn, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)
    }

    pub fn list(&self, year: i32) -> Result<Vec<IncomeTaxReturn>, DomainError> {
        let y = if year == 0 {
            {
                use chrono::Datelike;
                chrono::Local::now().date_naive().year()
            }
        } else {
            year
        };
        self.repo.list(y)
    }

    pub fn delete(&self, id: i64) -> Result<(), DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        let itr = self.repo.get_by_id(id)?;
        if itr.status == FilingStatus::Filed {
            return Err(DomainError::FilingAlreadyFiled);
        }
        self.repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("income_tax_return", id, "delete", None, None);
        }
        Ok(())
    }

    pub fn mark_filed(&self, id: i64) -> Result<IncomeTaxReturn, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        let mut itr = self.repo.get_by_id(id)?;
        if itr.status == FilingStatus::Filed {
            return Err(DomainError::FilingAlreadyFiled);
        }
        itr.status = FilingStatus::Filed;
        itr.filed_at = Some(chrono::Local::now().naive_local());
        self.repo.update(&mut itr)?;
        if let Some(ref audit) = self.audit {
            audit.log("income_tax_return", id, "mark_filed", None, None);
        }
        Ok(itr)
    }
}
