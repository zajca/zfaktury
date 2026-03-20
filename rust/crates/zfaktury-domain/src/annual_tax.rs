use chrono::NaiveDateTime;

use crate::amount::Amount;
use crate::tax::{FilingStatus, FilingType};

/// DPFO - danove priznani fyzickych osob (income tax return for individuals).
#[derive(Debug, Clone)]
pub struct IncomeTaxReturn {
    pub id: i64,
    pub year: i32,
    pub filing_type: FilingType,

    // Section 7 - business income
    pub total_revenue: Amount,
    pub actual_expenses: Amount,
    pub flat_rate_percent: i32,
    pub flat_rate_amount: Amount,
    pub used_expenses: Amount,

    // Tax base
    pub tax_base: Amount,
    pub total_deductions: Amount,
    pub tax_base_rounded: Amount,

    // Tax calculation (15% / 23% progressive)
    pub tax_at_15: Amount,
    pub tax_at_23: Amount,
    pub total_tax: Amount,

    // Credits (slevy na dani)
    pub credit_basic: Amount,
    pub credit_spouse: Amount,
    pub credit_disability: Amount,
    pub credit_student: Amount,
    pub total_credits: Amount,
    pub tax_after_credits: Amount,

    // Child benefit (danove zvyhodneni)
    pub child_benefit: Amount,
    pub tax_after_benefit: Amount,

    // Prepayments & result
    pub prepayments: Amount,
    pub tax_due: Amount,

    // Section 8 capital income
    pub capital_income_gross: Amount,
    pub capital_income_tax: Amount,
    pub capital_income_net: Amount,

    // Section 10 other income (securities, crypto)
    pub other_income_gross: Amount,
    pub other_income_expenses: Amount,
    pub other_income_exempt: Amount,
    pub other_income_net: Amount,

    pub xml_data: Vec<u8>,
    pub status: FilingStatus,
    pub filed_at: Option<NaiveDateTime>,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
}

/// Prehled OSVC pro CSSZ (social insurance overview).
#[derive(Debug, Clone)]
pub struct SocialInsuranceOverview {
    pub id: i64,
    pub year: i32,
    pub filing_type: FilingType,

    pub total_revenue: Amount,
    pub total_expenses: Amount,
    pub tax_base: Amount,
    pub assessment_base: Amount,
    pub min_assessment_base: Amount,
    pub final_assessment_base: Amount,

    /// Insurance rate stored as permille * 10 (e.g. 292 = 29.2%).
    pub insurance_rate: i32,
    pub total_insurance: Amount,
    pub prepayments: Amount,
    pub difference: Amount,
    pub new_monthly_prepay: Amount,

    pub xml_data: Vec<u8>,
    pub status: FilingStatus,
    pub filed_at: Option<NaiveDateTime>,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
}

/// Prehled OSVC pro ZP (health insurance overview).
#[derive(Debug, Clone)]
pub struct HealthInsuranceOverview {
    pub id: i64,
    pub year: i32,
    pub filing_type: FilingType,

    pub total_revenue: Amount,
    pub total_expenses: Amount,
    pub tax_base: Amount,
    pub assessment_base: Amount,
    pub min_assessment_base: Amount,
    pub final_assessment_base: Amount,

    /// Insurance rate stored as permille * 10 (e.g. 135 = 13.5%).
    pub insurance_rate: i32,
    pub total_insurance: Amount,
    pub prepayments: Amount,
    pub difference: Amount,
    pub new_monthly_prepay: Amount,

    pub xml_data: Vec<u8>,
    pub status: FilingStatus,
    pub filed_at: Option<NaiveDateTime>,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
}
