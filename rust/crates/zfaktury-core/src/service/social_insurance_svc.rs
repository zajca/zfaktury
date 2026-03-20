use std::sync::Arc;

use zfaktury_domain::{DomainError, FilingStatus, FilingType, SocialInsuranceOverview};

use super::audit_svc::AuditService;
use crate::repository::traits::{
    ExpenseRepo, InvoiceRepo, SettingsRepo, SocialInsuranceOverviewRepo, TaxPrepaymentRepo,
    TaxYearSettingsRepo,
};

/// Service for social insurance overview management.
pub struct SocialInsuranceService {
    repo: Arc<dyn SocialInsuranceOverviewRepo + Send + Sync>,
    invoice_repo: Arc<dyn InvoiceRepo + Send + Sync>,
    expense_repo: Arc<dyn ExpenseRepo + Send + Sync>,
    settings_repo: Arc<dyn SettingsRepo + Send + Sync>,
    tax_year_settings_repo: Arc<dyn TaxYearSettingsRepo + Send + Sync>,
    tax_prepayment_repo: Arc<dyn TaxPrepaymentRepo + Send + Sync>,
    audit: Option<Arc<AuditService>>,
}

impl SocialInsuranceService {
    pub fn new(
        repo: Arc<dyn SocialInsuranceOverviewRepo + Send + Sync>,
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

    pub fn create(&self, sio: &mut SocialInsuranceOverview) -> Result<(), DomainError> {
        if sio.year < 2000 || sio.year > 2100 {
            return Err(DomainError::InvalidInput);
        }
        if sio.filing_type == FilingType::Regular
            && self
                .repo
                .get_by_year(sio.year, &sio.filing_type.to_string())
                .is_ok()
        {
            return Err(DomainError::FilingAlreadyExists);
        }
        self.repo.create(sio)?;
        if let Some(ref audit) = self.audit {
            audit.log("social_insurance", sio.id, "create", None, None);
        }
        Ok(())
    }

    pub fn get_by_id(&self, id: i64) -> Result<SocialInsuranceOverview, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        self.repo.get_by_id(id)
    }

    pub fn list(&self, year: i32) -> Result<Vec<SocialInsuranceOverview>, DomainError> {
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
        let sio = self.repo.get_by_id(id)?;
        if sio.status == FilingStatus::Filed {
            return Err(DomainError::FilingAlreadyFiled);
        }
        self.repo.delete(id)?;
        if let Some(ref audit) = self.audit {
            audit.log("social_insurance", id, "delete", None, None);
        }
        Ok(())
    }

    pub fn mark_filed(&self, id: i64) -> Result<SocialInsuranceOverview, DomainError> {
        if id == 0 {
            return Err(DomainError::InvalidInput);
        }
        let mut sio = self.repo.get_by_id(id)?;
        if sio.status == FilingStatus::Filed {
            return Err(DomainError::FilingAlreadyFiled);
        }
        sio.status = FilingStatus::Filed;
        sio.filed_at = Some(chrono::Local::now().naive_local());
        self.repo.update(&mut sio)?;
        if let Some(ref audit) = self.audit {
            audit.log("social_insurance", id, "mark_filed", None, None);
        }
        Ok(sio)
    }
}
