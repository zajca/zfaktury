use std::sync::Arc;

use zfaktury_domain::{Amount, DomainError, TaxPrepayment, TaxYearSettings};

use super::audit_svc::AuditService;
use crate::repository::traits::{TaxPrepaymentRepo, TaxYearSettingsRepo};

/// Service for per-year tax settings and prepayments.
pub struct TaxYearSettingsService {
    settings_repo: Arc<dyn TaxYearSettingsRepo + Send + Sync>,
    prepayment_repo: Arc<dyn TaxPrepaymentRepo + Send + Sync>,
    audit: Option<Arc<AuditService>>,
}

impl TaxYearSettingsService {
    pub fn new(
        settings_repo: Arc<dyn TaxYearSettingsRepo + Send + Sync>,
        prepayment_repo: Arc<dyn TaxPrepaymentRepo + Send + Sync>,
        audit: Option<Arc<AuditService>>,
    ) -> Self {
        Self {
            settings_repo,
            prepayment_repo,
            audit,
        }
    }

    /// Returns tax year settings for a given year, defaulting to zero values if not found.
    pub fn get_by_year(&self, year: i32) -> Result<TaxYearSettings, DomainError> {
        match self.settings_repo.get_by_year(year) {
            Ok(tys) => Ok(tys),
            Err(DomainError::NotFound) => Ok(TaxYearSettings {
                year,
                flat_rate_percent: 0,
                created_at: chrono::NaiveDateTime::default(),
                updated_at: chrono::NaiveDateTime::default(),
            }),
            Err(e) => Err(e),
        }
    }

    /// Returns 12 months of prepayments for a given year, filling missing months with zeros.
    pub fn get_prepayments(&self, year: i32) -> Result<Vec<TaxPrepayment>, DomainError> {
        let existing = self.prepayment_repo.list_by_year(year)?;
        let mut by_month = std::collections::HashMap::new();
        for tp in existing {
            by_month.insert(tp.month, tp);
        }
        let mut result = Vec::with_capacity(12);
        for m in 1..=12 {
            result.push(by_month.remove(&m).unwrap_or(TaxPrepayment {
                year,
                month: m,
                tax_amount: Amount::ZERO,
                social_amount: Amount::ZERO,
                health_amount: Amount::ZERO,
            }));
        }
        Ok(result)
    }

    /// Upserts both tax year settings and all 12 months of prepayments.
    pub fn save(
        &self,
        year: i32,
        flat_rate_percent: i32,
        prepayments: &[TaxPrepayment],
    ) -> Result<(), DomainError> {
        let mut tys = TaxYearSettings {
            year,
            flat_rate_percent,
            created_at: chrono::NaiveDateTime::default(),
            updated_at: chrono::NaiveDateTime::default(),
        };
        self.settings_repo.upsert(&mut tys)?;
        self.prepayment_repo.upsert_all(year, prepayments)?;
        if let Some(ref audit) = self.audit {
            audit.log("tax_year_settings", year as i64, "save", None, None);
        }
        Ok(())
    }

    /// Returns the annual sum of prepayments.
    pub fn get_prepayment_sums(&self, year: i32) -> Result<(Amount, Amount, Amount), DomainError> {
        self.prepayment_repo.sum_by_year(year)
    }
}
