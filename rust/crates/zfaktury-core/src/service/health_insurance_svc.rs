use std::sync::Arc;

use zfaktury_domain::{DomainError, FilingStatus, FilingType, HealthInsuranceOverview};

use super::audit_svc::AuditService;
use crate::repository::traits::{
    ExpenseRepo, HealthInsuranceOverviewRepo, InvoiceRepo, SettingsRepo, TaxPrepaymentRepo,
    TaxYearSettingsRepo,
};

/// Service for health insurance overview management.
#[allow(dead_code)]
pub struct HealthInsuranceService {
    repo: Arc<dyn HealthInsuranceOverviewRepo + Send + Sync>,
    invoice_repo: Arc<dyn InvoiceRepo + Send + Sync>,
    expense_repo: Arc<dyn ExpenseRepo + Send + Sync>,
    settings_repo: Arc<dyn SettingsRepo + Send + Sync>,
    tax_year_settings_repo: Arc<dyn TaxYearSettingsRepo + Send + Sync>,
    tax_prepayment_repo: Arc<dyn TaxPrepaymentRepo + Send + Sync>,
    audit: Option<Arc<AuditService>>,
}

impl HealthInsuranceService {
    pub fn new(
        repo: Arc<dyn HealthInsuranceOverviewRepo + Send + Sync>,
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

    pub fn create(&self, hi: &mut HealthInsuranceOverview) -> Result<(), DomainError> {
        if hi.year < 2000 || hi.year > 2100 {
            return Err(DomainError::InvalidInput);
        }
        if hi.filing_type == FilingType::Regular
            && self
                .repo
                .get_by_year(hi.year, &hi.filing_type.to_string())
                .is_ok()
        {
            return Err(DomainError::FilingAlreadyExists);
        }
        self.repo.create(hi)?;
        if let Some(ref audit) = self.audit {
            audit.log("health_insurance", hi.id, "create", None, None);
        }
        Ok(())
    }

    pub fn get_by_id(&self, id: i64) -> Result<HealthInsuranceOverview, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)
    }

    pub fn list(&self, year: i32) -> Result<Vec<HealthInsuranceOverview>, DomainError> {
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
        let hi = self.repo.get_by_id(id)?;
        if hi.status == FilingStatus::Filed {
            return Err(DomainError::FilingAlreadyFiled);
        }
        self.repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("health_insurance", id, "delete", None, None);
        }
        Ok(())
    }

    pub fn mark_filed(&self, id: i64) -> Result<HealthInsuranceOverview, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        let mut hi = self.repo.get_by_id(id)?;
        if hi.status == FilingStatus::Filed {
            return Err(DomainError::FilingAlreadyFiled);
        }
        hi.status = FilingStatus::Filed;
        hi.filed_at = Some(chrono::Local::now().naive_local());
        self.repo.update(&mut hi)?;
        if let Some(ref audit) = self.audit {
            audit.log("health_insurance", id, "mark_filed", None, None);
        }
        Ok(hi)
    }
}
