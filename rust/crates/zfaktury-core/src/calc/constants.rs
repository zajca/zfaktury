use std::collections::HashMap;
use zfaktury_domain::Amount;

/// Tax computation constants for a specific year.
#[derive(Debug, Clone)]
pub struct TaxYearConstants {
    /// Progressive tax threshold for 23% rate (in halere).
    pub progressive_threshold: Amount,
    /// Basic taxpayer credit.
    pub basic_credit: Amount,
    /// Spouse credit.
    pub spouse_credit: Amount,
    /// Student credit.
    pub student_credit: Amount,
    /// Disability credit level 1 and 2.
    pub disability_credit_1: Amount,
    /// Disability credit level 3.
    pub disability_credit_3: Amount,
    /// ZTP/P holder credit.
    pub disability_ztpp: Amount,
    /// Child benefit for 1st child.
    pub child_benefit_1: Amount,
    /// Child benefit for 2nd child.
    pub child_benefit_2: Amount,
    /// Child benefit for 3rd and subsequent children.
    pub child_benefit_3_plus: Amount,
    /// ZTP surcharge for children (doubled automatically).
    pub child_benefit_ztp: Amount,
    /// Minimum monthly assessment base for social insurance (CSSZ).
    pub social_min_monthly: Amount,
    /// Social insurance rate in permille*10 (e.g. 292 = 29.2%).
    pub social_rate: i32,
    /// Minimum monthly assessment base for health insurance (ZP).
    pub health_min_monthly: Amount,
    /// Health insurance rate in permille*10 (e.g. 135 = 13.5%).
    pub health_rate: i32,
    /// Flat rate expense caps: percent -> max amount in halere.
    pub flat_rate_caps: HashMap<i32, Amount>,
    /// Years to hold asset for time test exemption.
    pub time_test_years: i32,
    /// Max exempt amount for securities per year (0 = no limit).
    pub security_exemption_limit: Amount,
    /// Max spouse income for credit eligibility.
    pub spouse_income_limit: Amount,
    /// Max deduction for mortgage interest.
    pub deduction_cap_mortgage: Amount,
    /// Max deduction for life insurance.
    pub deduction_cap_life_insurance: Amount,
    /// Max deduction for pension savings.
    pub deduction_cap_pension: Amount,
    /// Max deduction for union dues.
    pub deduction_cap_union_dues: Amount,
    /// Max annual child tax bonus.
    pub max_child_bonus: Amount,
}

/// Error returned when tax constants are unavailable for a given year.
#[derive(Debug, thiserror::Error)]
#[error("no tax constants for year {year}")]
pub struct UnsupportedYearError {
    pub year: i32,
}

/// Returns the tax constants for a given year.
pub fn get_tax_constants(year: i32) -> Result<TaxYearConstants, UnsupportedYearError> {
    match year {
        2024 => Ok(constants_2024()),
        2025 => Ok(constants_2025()),
        2026 => Ok(constants_2026()),
        _ => Err(UnsupportedYearError { year }),
    }
}

fn flat_rate_caps() -> HashMap<i32, Amount> {
    let mut caps = HashMap::new();
    caps.insert(80, Amount::new(1_600_000, 0));
    caps.insert(60, Amount::new(1_200_000, 0));
    caps.insert(40, Amount::new(800_000, 0));
    caps.insert(30, Amount::new(600_000, 0));
    caps
}

fn constants_2024() -> TaxYearConstants {
    TaxYearConstants {
        progressive_threshold: Amount::new(1_582_812, 0),
        basic_credit: Amount::new(30_840, 0),
        spouse_credit: Amount::new(24_840, 0),
        student_credit: Amount::new(4_020, 0),
        disability_credit_1: Amount::new(2_520, 0),
        disability_credit_3: Amount::new(5_040, 0),
        disability_ztpp: Amount::new(16_140, 0),
        child_benefit_1: Amount::new(15_204, 0),
        child_benefit_2: Amount::new(22_320, 0),
        child_benefit_3_plus: Amount::new(27_840, 0),
        child_benefit_ztp: Amount::new(0, 0), // doubled automatically
        social_min_monthly: Amount::new(11_024, 0),
        social_rate: 292,
        health_min_monthly: Amount::new(10_081, 0),
        health_rate: 135,
        flat_rate_caps: flat_rate_caps(),
        time_test_years: 3,
        security_exemption_limit: Amount::new(0, 0), // no limit before 2025

        spouse_income_limit: Amount::new(68_000, 0),
        deduction_cap_mortgage: Amount::new(150_000, 0),
        deduction_cap_life_insurance: Amount::new(24_000, 0),
        deduction_cap_pension: Amount::new(24_000, 0),
        deduction_cap_union_dues: Amount::new(3_000, 0),
        max_child_bonus: Amount::new(60_300, 0),
    }
}

fn constants_2025() -> TaxYearConstants {
    TaxYearConstants {
        progressive_threshold: Amount::new(1_582_812, 0),
        basic_credit: Amount::new(30_840, 0),
        spouse_credit: Amount::new(24_840, 0),
        student_credit: Amount::new(4_020, 0),
        disability_credit_1: Amount::new(2_520, 0),
        disability_credit_3: Amount::new(5_040, 0),
        disability_ztpp: Amount::new(16_140, 0),
        child_benefit_1: Amount::new(15_204, 0),
        child_benefit_2: Amount::new(22_320, 0),
        child_benefit_3_plus: Amount::new(27_840, 0),
        child_benefit_ztp: Amount::new(0, 0),
        social_min_monthly: Amount::new(11_584, 0),
        social_rate: 292,
        health_min_monthly: Amount::new(10_874, 0),
        health_rate: 135,
        flat_rate_caps: flat_rate_caps(),
        time_test_years: 3,
        security_exemption_limit: Amount::new(100_000_000, 0), // 1M CZK

        spouse_income_limit: Amount::new(68_000, 0),
        deduction_cap_mortgage: Amount::new(150_000, 0),
        deduction_cap_life_insurance: Amount::new(24_000, 0),
        deduction_cap_pension: Amount::new(24_000, 0),
        deduction_cap_union_dues: Amount::new(3_000, 0),
        max_child_bonus: Amount::new(60_300, 0),
    }
}

fn constants_2026() -> TaxYearConstants {
    TaxYearConstants {
        progressive_threshold: Amount::new(1_582_812, 0),
        basic_credit: Amount::new(30_840, 0),
        spouse_credit: Amount::new(24_840, 0),
        student_credit: Amount::new(4_020, 0),
        disability_credit_1: Amount::new(2_520, 0),
        disability_credit_3: Amount::new(5_040, 0),
        disability_ztpp: Amount::new(16_140, 0),
        child_benefit_1: Amount::new(15_204, 0),
        child_benefit_2: Amount::new(22_320, 0),
        child_benefit_3_plus: Amount::new(27_840, 0),
        child_benefit_ztp: Amount::new(0, 0),
        social_min_monthly: Amount::new(12_139, 0),
        social_rate: 292,
        health_min_monthly: Amount::new(11_396, 0),
        health_rate: 135,
        flat_rate_caps: flat_rate_caps(),
        time_test_years: 3,
        security_exemption_limit: Amount::new(100_000_000, 0), // 1M CZK

        spouse_income_limit: Amount::new(68_000, 0),
        deduction_cap_mortgage: Amount::new(150_000, 0),
        deduction_cap_life_insurance: Amount::new(24_000, 0),
        deduction_cap_pension: Amount::new(24_000, 0),
        deduction_cap_union_dues: Amount::new(3_000, 0),
        max_child_bonus: Amount::new(60_300, 0),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use rstest::rstest;

    #[rstest]
    #[case(2024)]
    #[case(2025)]
    #[case(2026)]
    fn test_supported_years(#[case] year: i32) {
        assert!(get_tax_constants(year).is_ok());
    }

    #[rstest]
    #[case(2023)]
    #[case(2027)]
    #[case(0)]
    #[case(-1)]
    fn test_unsupported_years(#[case] year: i32) {
        let err = get_tax_constants(year).unwrap_err();
        assert_eq!(err.year, year);
    }

    #[rstest]
    #[case(2024, Amount::new(1_582_812, 0))]
    #[case(2025, Amount::new(1_582_812, 0))]
    #[case(2026, Amount::new(1_582_812, 0))]
    fn test_progressive_threshold(#[case] year: i32, #[case] expected: Amount) {
        let c = get_tax_constants(year).unwrap();
        assert_eq!(c.progressive_threshold, expected);
    }

    #[rstest]
    #[case(2024, Amount::new(30_840, 0))]
    #[case(2025, Amount::new(30_840, 0))]
    #[case(2026, Amount::new(30_840, 0))]
    fn test_basic_credit(#[case] year: i32, #[case] expected: Amount) {
        let c = get_tax_constants(year).unwrap();
        assert_eq!(c.basic_credit, expected);
    }

    #[rstest]
    #[case(2024, Amount::new(24_840, 0))]
    #[case(2025, Amount::new(24_840, 0))]
    #[case(2026, Amount::new(24_840, 0))]
    fn test_spouse_credit(#[case] year: i32, #[case] expected: Amount) {
        let c = get_tax_constants(year).unwrap();
        assert_eq!(c.spouse_credit, expected);
    }

    #[rstest]
    #[case(2024, Amount::new(4_020, 0))]
    #[case(2025, Amount::new(4_020, 0))]
    #[case(2026, Amount::new(4_020, 0))]
    fn test_student_credit(#[case] year: i32, #[case] expected: Amount) {
        let c = get_tax_constants(year).unwrap();
        assert_eq!(c.student_credit, expected);
    }

    #[rstest]
    #[case(2024, Amount::new(2_520, 0))]
    #[case(2025, Amount::new(2_520, 0))]
    #[case(2026, Amount::new(2_520, 0))]
    fn test_disability_credit_1(#[case] year: i32, #[case] expected: Amount) {
        let c = get_tax_constants(year).unwrap();
        assert_eq!(c.disability_credit_1, expected);
    }

    #[rstest]
    #[case(2024, Amount::new(5_040, 0))]
    #[case(2025, Amount::new(5_040, 0))]
    #[case(2026, Amount::new(5_040, 0))]
    fn test_disability_credit_3(#[case] year: i32, #[case] expected: Amount) {
        let c = get_tax_constants(year).unwrap();
        assert_eq!(c.disability_credit_3, expected);
    }

    #[rstest]
    #[case(2024, Amount::new(16_140, 0))]
    #[case(2025, Amount::new(16_140, 0))]
    #[case(2026, Amount::new(16_140, 0))]
    fn test_disability_ztpp(#[case] year: i32, #[case] expected: Amount) {
        let c = get_tax_constants(year).unwrap();
        assert_eq!(c.disability_ztpp, expected);
    }

    #[rstest]
    #[case(
        2024,
        Amount::new(15_204, 0),
        Amount::new(22_320, 0),
        Amount::new(27_840, 0)
    )]
    #[case(
        2025,
        Amount::new(15_204, 0),
        Amount::new(22_320, 0),
        Amount::new(27_840, 0)
    )]
    #[case(
        2026,
        Amount::new(15_204, 0),
        Amount::new(22_320, 0),
        Amount::new(27_840, 0)
    )]
    fn test_child_benefits(
        #[case] year: i32,
        #[case] b1: Amount,
        #[case] b2: Amount,
        #[case] b3: Amount,
    ) {
        let c = get_tax_constants(year).unwrap();
        assert_eq!(c.child_benefit_1, b1);
        assert_eq!(c.child_benefit_2, b2);
        assert_eq!(c.child_benefit_3_plus, b3);
    }

    #[rstest]
    #[case(2024, Amount::new(11_024, 0), Amount::new(10_081, 0))]
    #[case(2025, Amount::new(11_584, 0), Amount::new(10_874, 0))]
    #[case(2026, Amount::new(12_139, 0), Amount::new(11_396, 0))]
    fn test_insurance_minimums(
        #[case] year: i32,
        #[case] social_min: Amount,
        #[case] health_min: Amount,
    ) {
        let c = get_tax_constants(year).unwrap();
        assert_eq!(c.social_min_monthly, social_min);
        assert_eq!(c.health_min_monthly, health_min);
    }

    #[rstest]
    #[case(2024)]
    #[case(2025)]
    #[case(2026)]
    fn test_insurance_rates(#[case] year: i32) {
        let c = get_tax_constants(year).unwrap();
        assert_eq!(c.social_rate, 292);
        assert_eq!(c.health_rate, 135);
    }

    #[rstest]
    #[case(2024)]
    #[case(2025)]
    #[case(2026)]
    fn test_flat_rate_caps(#[case] year: i32) {
        let c = get_tax_constants(year).unwrap();
        assert_eq!(c.flat_rate_caps[&80], Amount::new(1_600_000, 0));
        assert_eq!(c.flat_rate_caps[&60], Amount::new(1_200_000, 0));
        assert_eq!(c.flat_rate_caps[&40], Amount::new(800_000, 0));
        assert_eq!(c.flat_rate_caps[&30], Amount::new(600_000, 0));
    }

    #[test]
    fn test_2024_security_exemption_no_limit() {
        let c = get_tax_constants(2024).unwrap();
        assert_eq!(c.security_exemption_limit, Amount::ZERO);
    }

    #[rstest]
    #[case(2025)]
    #[case(2026)]
    fn test_2025_2026_security_exemption_1m_czk(#[case] year: i32) {
        let c = get_tax_constants(year).unwrap();
        assert_eq!(c.security_exemption_limit, Amount::new(100_000_000, 0));
    }

    #[rstest]
    #[case(2024)]
    #[case(2025)]
    #[case(2026)]
    fn test_time_test_years(#[case] year: i32) {
        let c = get_tax_constants(year).unwrap();
        assert_eq!(c.time_test_years, 3);
    }

    #[rstest]
    #[case(2024)]
    #[case(2025)]
    #[case(2026)]
    fn test_deduction_caps(#[case] year: i32) {
        let c = get_tax_constants(year).unwrap();
        assert_eq!(c.spouse_income_limit, Amount::new(68_000, 0));
        assert_eq!(c.deduction_cap_mortgage, Amount::new(150_000, 0));
        assert_eq!(c.deduction_cap_life_insurance, Amount::new(24_000, 0));
        assert_eq!(c.deduction_cap_pension, Amount::new(24_000, 0));
        assert_eq!(c.deduction_cap_union_dues, Amount::new(3_000, 0));
        assert_eq!(c.max_child_bonus, Amount::new(60_300, 0));
    }
}
