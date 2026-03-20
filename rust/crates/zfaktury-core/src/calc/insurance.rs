use std::collections::HashMap;
use zfaktury_domain::Amount;

/// Parameters for an insurance calculation.
#[derive(Debug, Clone)]
pub struct InsuranceInput {
    /// Total revenue.
    pub revenue: Amount,
    /// Already resolved expenses (flat-rate or actual).
    pub used_expenses: Amount,
    /// Minimum monthly assessment base (e.g. constants.social_min_monthly or health_min_monthly).
    pub min_monthly_base: Amount,
    /// Rate in permille*10 (292 for social, 135 for health).
    pub rate_permille: i32,
    /// Prepayments already made.
    pub prepayments: Amount,
}

/// Computed insurance values.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct InsuranceResult {
    /// Tax base: max(0, revenue - expenses).
    pub tax_base: Amount,
    /// Assessment base: tax_base / 2.
    pub assessment_base: Amount,
    /// Minimum assessment base: min_monthly_base * 12.
    pub min_assessment_base: Amount,
    /// Final assessment base: max(assessment_base, min_assessment_base).
    pub final_assessment_base: Amount,
    /// Total insurance: final_base * rate / 1000.
    pub total_insurance: Amount,
    /// Difference: total_insurance - prepayments.
    pub difference: Amount,
    /// New monthly prepayment: ceil(total/12) rounded up to nearest CZK (100 halere).
    pub new_monthly_prepay: Amount,
}

/// Computes the annual insurance overview from the given input.
pub fn calculate_insurance(input: &InsuranceInput) -> InsuranceResult {
    // tax_base = revenue - used_expenses, clamped to 0.
    let mut tax_base = input.revenue.halere() - input.used_expenses.halere();
    if tax_base < 0 {
        tax_base = 0;
    }

    // assessment_base = tax_base / 2 (integer division in halere).
    let assessment_base = tax_base / 2;

    // min_assessment_base = min_monthly_base * 12.
    let min_assessment_base = input.min_monthly_base.halere() * 12;

    // final_base = max(assessment_base, min_assessment_base).
    let final_base = if min_assessment_base > assessment_base {
        min_assessment_base
    } else {
        assessment_base
    };

    // total_insurance = final_base * rate_permille / 1000.
    let total_insurance = final_base * input.rate_permille as i64 / 1000;

    // difference = total_insurance - prepayments.
    let difference = total_insurance - input.prepayments.halere();

    // new_monthly_prepay = ceil(total_insurance / 12), rounded up to nearest 100 halere (1 CZK).
    let mut monthly_halere = total_insurance / 12;
    if total_insurance % 12 != 0 {
        monthly_halere += 1;
    }
    let rounded_up_czk = ((monthly_halere + 99) / 100) * 100;

    InsuranceResult {
        tax_base: Amount::from_halere(tax_base),
        assessment_base: Amount::from_halere(assessment_base),
        min_assessment_base: Amount::from_halere(min_assessment_base),
        final_assessment_base: Amount::from_halere(final_base),
        total_insurance: Amount::from_halere(total_insurance),
        difference: Amount::from_halere(difference),
        new_monthly_prepay: Amount::from_halere(rounded_up_czk),
    }
}

/// Determines the expenses to use in tax calculations.
/// If flat_rate_percent > 0, computes revenue * flat_rate_percent/100, applies the
/// cap from caps map if one exists for the given percent, and returns the result.
/// Otherwise returns actual_expenses unchanged.
pub fn resolve_used_expenses(
    revenue: Amount,
    actual_expenses: Amount,
    flat_rate_percent: i32,
    caps: &HashMap<i32, Amount>,
) -> Amount {
    if flat_rate_percent > 0 {
        let amount = revenue.multiply(flat_rate_percent as f64 / 100.0);
        if let Some(&cap) = caps.get(&flat_rate_percent)
            && amount > cap
        {
            return cap;
        }
        amount
    } else {
        actual_expenses
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::calc::constants::get_tax_constants;
    use rstest::rstest;

    fn amt(whole: i64, fraction: i64) -> Amount {
        Amount::new(whole, fraction)
    }

    #[test]
    fn test_below_minimum_uses_minimum() {
        // Small revenue so assessment base is below minimum.
        let c = get_tax_constants(2024).unwrap();
        let input = InsuranceInput {
            revenue: amt(200_000, 0),
            used_expenses: amt(100_000, 0),
            min_monthly_base: c.social_min_monthly,
            rate_permille: c.social_rate,
            prepayments: Amount::ZERO,
        };

        let result = calculate_insurance(&input);
        // tax_base = 200,000 - 100,000 = 100,000 CZK = 10,000,000 halere
        assert_eq!(result.tax_base, amt(100_000, 0));
        // assessment = 50,000 CZK = 5,000,000 halere
        assert_eq!(result.assessment_base, amt(50_000, 0));
        // min = 11,024 * 12 = 132,288 CZK
        let min = Amount::from_halere(c.social_min_monthly.halere() * 12);
        assert_eq!(result.min_assessment_base, min);
        // 50,000 < 132,288 so final = 132,288
        assert_eq!(result.final_assessment_base, min);
    }

    #[test]
    fn test_above_minimum_uses_actual() {
        let c = get_tax_constants(2024).unwrap();
        let input = InsuranceInput {
            revenue: amt(2_000_000, 0),
            used_expenses: amt(500_000, 0),
            min_monthly_base: c.social_min_monthly,
            rate_permille: c.social_rate,
            prepayments: Amount::ZERO,
        };

        let result = calculate_insurance(&input);
        // tax_base = 1,500,000 CZK
        assert_eq!(result.tax_base, amt(1_500_000, 0));
        // assessment = 750,000 CZK
        assert_eq!(result.assessment_base, amt(750_000, 0));
        // min = 132,288 CZK, 750,000 > 132,288 -> use actual
        assert_eq!(result.final_assessment_base, amt(750_000, 0));
    }

    #[test]
    fn test_social_rate_application() {
        let c = get_tax_constants(2024).unwrap();
        let input = InsuranceInput {
            revenue: amt(2_000_000, 0),
            used_expenses: amt(500_000, 0),
            min_monthly_base: c.social_min_monthly,
            rate_permille: c.social_rate, // 292
            prepayments: Amount::ZERO,
        };

        let result = calculate_insurance(&input);
        // final_base = 750,000 CZK = 75,000,000 halere
        // total = 75,000,000 * 292 / 1000 = 21,900,000 halere = 219,000 CZK
        assert_eq!(result.total_insurance, amt(219_000, 0));
    }

    #[test]
    fn test_health_rate_application() {
        let c = get_tax_constants(2024).unwrap();
        let input = InsuranceInput {
            revenue: amt(2_000_000, 0),
            used_expenses: amt(500_000, 0),
            min_monthly_base: c.health_min_monthly,
            rate_permille: c.health_rate, // 135
            prepayments: Amount::ZERO,
        };

        let result = calculate_insurance(&input);
        // final_base = 750,000 CZK = 75,000,000 halere
        // total = 75,000,000 * 135 / 1000 = 10,125,000 halere = 101,250 CZK
        assert_eq!(result.total_insurance, amt(101_250, 0));
    }

    #[test]
    fn test_monthly_prepayment_rounding() {
        let c = get_tax_constants(2024).unwrap();
        let input = InsuranceInput {
            revenue: amt(2_000_000, 0),
            used_expenses: amt(500_000, 0),
            min_monthly_base: c.social_min_monthly,
            rate_permille: c.social_rate,
            prepayments: Amount::ZERO,
        };

        let result = calculate_insurance(&input);
        // total = 219,000 CZK = 21,900,000 halere
        // monthly = 21,900,000 / 12 = 1,825,000 halere (exact, no remainder)
        // rounded up to 100: 1,825,000 already divisible by 100
        assert_eq!(result.new_monthly_prepay, amt(18_250, 0));
    }

    #[test]
    fn test_monthly_prepayment_rounds_up_to_czk() {
        // Use values that produce a non-round monthly amount.
        let input = InsuranceInput {
            revenue: amt(1_000_000, 0),
            used_expenses: amt(400_000, 0),
            min_monthly_base: amt(11_024, 0),
            rate_permille: 292,
            prepayments: Amount::ZERO,
        };

        let result = calculate_insurance(&input);
        // tax_base = 600,000 CZK = 60,000,000 halere
        // assessment = 300,000 CZK = 30,000,000 halere
        // min = 11,024 * 100 * 12 = 13,228,800 halere
        // final = 30,000,000 (above min)
        // total = 30,000,000 * 292 / 1000 = 8,760,000 halere = 87,600 CZK
        // monthly = 8,760,000 / 12 = 730,000 halere = 7,300 CZK (exact)
        assert_eq!(result.total_insurance, amt(87_600, 0));
        assert_eq!(result.new_monthly_prepay, amt(7_300, 0));
    }

    #[test]
    fn test_monthly_prepayment_non_exact_division() {
        // Create a scenario where total_insurance / 12 has a remainder.
        let input = InsuranceInput {
            revenue: amt(1_000_001, 0), // 100,000,100 halere
            used_expenses: Amount::ZERO,
            min_monthly_base: Amount::ZERO, // no minimum for this test
            rate_permille: 292,
            prepayments: Amount::ZERO,
        };

        let result = calculate_insurance(&input);
        // tax_base = 100,000,100 halere
        // assessment = 50,000,050 halere
        // total = 50,000,050 * 292 / 1000 = 14,600,014 halere (truncated)
        let total = result.total_insurance.halere();
        assert_eq!(total, 14_600_014);
        // monthly = 14,600,014 / 12 = 1,216,667 remainder 10
        // so monthly_halere = 1,216,668 (ceil)
        // rounded up to 100: (1,216,668 + 99) / 100 * 100 = 1,216,767 / 100 * 100 = 1,216,700
        assert_eq!(result.new_monthly_prepay, Amount::from_halere(1_216_700));
    }

    #[test]
    fn test_difference_calculation() {
        let c = get_tax_constants(2024).unwrap();
        let input = InsuranceInput {
            revenue: amt(2_000_000, 0),
            used_expenses: amt(500_000, 0),
            min_monthly_base: c.social_min_monthly,
            rate_permille: c.social_rate,
            prepayments: amt(200_000, 0),
        };

        let result = calculate_insurance(&input);
        // total = 219,000 CZK, prepayments = 200,000 CZK
        assert_eq!(result.difference, amt(19_000, 0));
    }

    #[test]
    fn test_negative_difference_overpaid() {
        let c = get_tax_constants(2024).unwrap();
        let input = InsuranceInput {
            revenue: amt(2_000_000, 0),
            used_expenses: amt(500_000, 0),
            min_monthly_base: c.social_min_monthly,
            rate_permille: c.social_rate,
            prepayments: amt(250_000, 0),
        };

        let result = calculate_insurance(&input);
        // total = 219,000, prepayments = 250,000 -> diff = -31,000
        assert_eq!(result.difference, amt(-31_000, 0));
        assert!(result.difference.is_negative());
    }

    #[test]
    fn test_zero_revenue_uses_minimum() {
        let c = get_tax_constants(2024).unwrap();
        let input = InsuranceInput {
            revenue: Amount::ZERO,
            used_expenses: Amount::ZERO,
            min_monthly_base: c.social_min_monthly,
            rate_permille: c.social_rate,
            prepayments: Amount::ZERO,
        };

        let result = calculate_insurance(&input);
        assert_eq!(result.tax_base, Amount::ZERO);
        assert_eq!(result.assessment_base, Amount::ZERO);
        // Uses minimum
        let min = Amount::from_halere(c.social_min_monthly.halere() * 12);
        assert_eq!(result.final_assessment_base, min);
    }

    #[test]
    fn test_negative_tax_base_clamped() {
        let c = get_tax_constants(2024).unwrap();
        let input = InsuranceInput {
            revenue: amt(100_000, 0),
            used_expenses: amt(200_000, 0),
            min_monthly_base: c.social_min_monthly,
            rate_permille: c.social_rate,
            prepayments: Amount::ZERO,
        };

        let result = calculate_insurance(&input);
        assert_eq!(result.tax_base, Amount::ZERO);
        assert_eq!(result.assessment_base, Amount::ZERO);
    }

    // -- resolve_used_expenses tests --

    #[test]
    fn test_resolve_actual_expenses_when_no_flat_rate() {
        let caps = HashMap::new();
        let result = resolve_used_expenses(amt(1_000_000, 0), amt(400_000, 0), 0, &caps);
        assert_eq!(result, amt(400_000, 0));
    }

    #[test]
    fn test_resolve_flat_rate_below_cap() {
        let mut caps = HashMap::new();
        caps.insert(60, amt(1_200_000, 0));

        let result = resolve_used_expenses(amt(1_000_000, 0), amt(400_000, 0), 60, &caps);
        // 60% of 1M = 600,000 (below cap)
        assert_eq!(result, amt(600_000, 0));
    }

    #[test]
    fn test_resolve_flat_rate_above_cap() {
        let mut caps = HashMap::new();
        caps.insert(60, amt(1_200_000, 0));

        let result = resolve_used_expenses(amt(3_000_000, 0), amt(400_000, 0), 60, &caps);
        // 60% of 3M = 1,800,000, cap is 1,200,000
        assert_eq!(result, amt(1_200_000, 0));
    }

    #[rstest]
    #[case(80, amt(1_600_000, 0))]
    #[case(60, amt(1_200_000, 0))]
    #[case(40, amt(800_000, 0))]
    #[case(30, amt(600_000, 0))]
    fn test_resolve_all_flat_rate_caps(#[case] pct: i32, #[case] expected_cap: Amount) {
        let caps = get_tax_constants(2024).unwrap().flat_rate_caps;
        // Revenue large enough to exceed cap
        let result = resolve_used_expenses(amt(10_000_000, 0), Amount::ZERO, pct, &caps);
        assert_eq!(result, expected_cap);
    }

    #[test]
    fn test_resolve_flat_rate_no_matching_cap() {
        let caps = HashMap::new(); // empty caps
        let result = resolve_used_expenses(amt(1_000_000, 0), amt(400_000, 0), 60, &caps);
        // No cap -> use full flat rate
        assert_eq!(result, amt(600_000, 0));
    }

    mod proptests {
        use super::*;
        use proptest::prelude::*;

        proptest! {
            #[test]
            fn final_assessment_base_at_least_minimum(
                revenue in 0i64..1_000_000_000i64,
                expenses in 0i64..1_000_000_000i64,
            ) {
                let c = get_tax_constants(2024).unwrap();
                let input = InsuranceInput {
                    revenue: Amount::from_halere(revenue),
                    used_expenses: Amount::from_halere(expenses),
                    min_monthly_base: c.social_min_monthly,
                    rate_permille: c.social_rate,
                    prepayments: Amount::ZERO,
                };

                let result = calculate_insurance(&input);
                prop_assert!(result.final_assessment_base >= result.min_assessment_base);
            }

            #[test]
            fn total_insurance_equals_base_times_rate(
                revenue in 0i64..1_000_000_000i64,
                expenses in 0i64..500_000_000i64,
            ) {
                let c = get_tax_constants(2024).unwrap();
                let input = InsuranceInput {
                    revenue: Amount::from_halere(revenue),
                    used_expenses: Amount::from_halere(expenses),
                    min_monthly_base: c.social_min_monthly,
                    rate_permille: c.social_rate,
                    prepayments: Amount::ZERO,
                };

                let result = calculate_insurance(&input);
                let expected = result.final_assessment_base.halere() * c.social_rate as i64 / 1000;
                prop_assert_eq!(result.total_insurance.halere(), expected);
            }

            #[test]
            fn tax_base_is_non_negative(
                revenue in 0i64..1_000_000_000i64,
                expenses in 0i64..1_000_000_000i64,
            ) {
                let c = get_tax_constants(2024).unwrap();
                let input = InsuranceInput {
                    revenue: Amount::from_halere(revenue),
                    used_expenses: Amount::from_halere(expenses),
                    min_monthly_base: c.social_min_monthly,
                    rate_permille: c.social_rate,
                    prepayments: Amount::ZERO,
                };

                let result = calculate_insurance(&input);
                prop_assert!(result.tax_base.halere() >= 0);
            }

            #[test]
            fn monthly_prepay_rounds_up_to_czk(
                revenue in 100_000_000i64..1_000_000_000i64,
            ) {
                let c = get_tax_constants(2024).unwrap();
                let input = InsuranceInput {
                    revenue: Amount::from_halere(revenue),
                    used_expenses: Amount::ZERO,
                    min_monthly_base: c.social_min_monthly,
                    rate_permille: c.social_rate,
                    prepayments: Amount::ZERO,
                };

                let result = calculate_insurance(&input);
                // Must be divisible by 100 halere (= 1 CZK)
                prop_assert_eq!(result.new_monthly_prepay.halere() % 100, 0);
            }
        }
    }
}
