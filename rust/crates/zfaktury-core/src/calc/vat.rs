use zfaktury_domain::Amount;

/// Invoice input for VAT return calculation.
#[derive(Debug, Clone)]
pub struct VATInvoiceInput {
    /// Invoice type: true if credit note (sign = -1).
    pub is_credit_note: bool,
    /// Invoice items.
    pub items: Vec<VATItemInput>,
}

/// Single invoice item for VAT calculation.
#[derive(Debug, Clone)]
pub struct VATItemInput {
    /// Pre-computed base: quantity * unit_price / 100.
    pub base: Amount,
    /// VAT amount for this item.
    pub vat_amount: Amount,
    /// VAT rate in percent (21 or 12).
    pub vat_rate_percent: i32,
}

/// Expense input for VAT return calculation.
#[derive(Debug, Clone)]
pub struct VATExpenseInput {
    /// Total amount including VAT.
    pub amount: Amount,
    /// VAT amount.
    pub vat_amount: Amount,
    /// VAT rate in percent (21 or 12).
    pub vat_rate_percent: i32,
    /// Business use percentage (0 means 100%).
    pub business_percent: i32,
}

/// Calculated VAT return values.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct VATResult {
    pub output_vat_base_21: Amount,
    pub output_vat_amount_21: Amount,
    pub output_vat_base_12: Amount,
    pub output_vat_amount_12: Amount,
    pub input_vat_base_21: Amount,
    pub input_vat_amount_21: Amount,
    pub input_vat_base_12: Amount,
    pub input_vat_amount_12: Amount,
    pub total_output_vat: Amount,
    pub total_input_vat: Amount,
    pub net_vat: Amount,
}

/// Computes VAT return totals from invoices and expenses.
pub fn calculate_vat_return(
    invoices: &[VATInvoiceInput],
    expenses: &[VATExpenseInput],
) -> VATResult {
    let mut r = VATResult::default();

    for inv in invoices {
        let sign: i64 = if inv.is_credit_note { -1 } else { 1 };

        for item in &inv.items {
            let base = item.base.multiply(sign as f64);
            let vat = item.vat_amount.multiply(sign as f64);

            match item.vat_rate_percent {
                21 => {
                    r.output_vat_base_21 += base;
                    r.output_vat_amount_21 += vat;
                }
                12 => {
                    r.output_vat_base_12 += base;
                    r.output_vat_amount_12 += vat;
                }
                _ => {}
            }
        }
    }

    for exp in expenses {
        let business_pct = if exp.business_percent == 0 {
            100
        } else {
            exp.business_percent
        };

        let factor = business_pct as f64 / 100.0;
        let input_vat = exp.vat_amount.multiply(factor);
        let input_base = (exp.amount - exp.vat_amount).multiply(factor);

        match exp.vat_rate_percent {
            21 => {
                r.input_vat_base_21 += input_base;
                r.input_vat_amount_21 += input_vat;
            }
            12 => {
                r.input_vat_base_12 += input_base;
                r.input_vat_amount_12 += input_vat;
            }
            _ => {}
        }
    }

    r.total_output_vat = r.output_vat_amount_21 + r.output_vat_amount_12;
    r.total_input_vat = r.input_vat_amount_21 + r.input_vat_amount_12;
    r.net_vat = r.total_output_vat - r.total_input_vat;

    r
}

#[cfg(test)]
mod tests {
    use super::*;
    use rstest::rstest;

    fn amt(whole: i64, fraction: i64) -> Amount {
        Amount::new(whole, fraction)
    }

    #[test]
    fn test_empty_inputs_return_zero() {
        let result = calculate_vat_return(&[], &[]);
        assert_eq!(result, VATResult::default());
    }

    #[test]
    fn test_single_invoice_21_percent() {
        let invoices = vec![VATInvoiceInput {
            is_credit_note: false,
            items: vec![VATItemInput {
                base: amt(10_000, 0),
                vat_amount: amt(2_100, 0),
                vat_rate_percent: 21,
            }],
        }];

        let result = calculate_vat_return(&invoices, &[]);
        assert_eq!(result.output_vat_base_21, amt(10_000, 0));
        assert_eq!(result.output_vat_amount_21, amt(2_100, 0));
        assert_eq!(result.total_output_vat, amt(2_100, 0));
        assert_eq!(result.net_vat, amt(2_100, 0));
    }

    #[test]
    fn test_single_invoice_12_percent() {
        let invoices = vec![VATInvoiceInput {
            is_credit_note: false,
            items: vec![VATItemInput {
                base: amt(10_000, 0),
                vat_amount: amt(1_200, 0),
                vat_rate_percent: 12,
            }],
        }];

        let result = calculate_vat_return(&invoices, &[]);
        assert_eq!(result.output_vat_base_12, amt(10_000, 0));
        assert_eq!(result.output_vat_amount_12, amt(1_200, 0));
        assert_eq!(result.total_output_vat, amt(1_200, 0));
        assert_eq!(result.net_vat, amt(1_200, 0));
    }

    #[test]
    fn test_credit_note_reverses_sign() {
        let invoices = vec![VATInvoiceInput {
            is_credit_note: true,
            items: vec![VATItemInput {
                base: amt(5_000, 0),
                vat_amount: amt(1_050, 0),
                vat_rate_percent: 21,
            }],
        }];

        let result = calculate_vat_return(&invoices, &[]);
        assert_eq!(result.output_vat_base_21, amt(-5_000, 0));
        assert_eq!(result.output_vat_amount_21, amt(-1_050, 0));
        assert_eq!(result.total_output_vat, amt(-1_050, 0));
        assert_eq!(result.net_vat, amt(-1_050, 0));
    }

    #[test]
    fn test_multiple_invoices_aggregated() {
        let invoices = vec![
            VATInvoiceInput {
                is_credit_note: false,
                items: vec![VATItemInput {
                    base: amt(10_000, 0),
                    vat_amount: amt(2_100, 0),
                    vat_rate_percent: 21,
                }],
            },
            VATInvoiceInput {
                is_credit_note: false,
                items: vec![
                    VATItemInput {
                        base: amt(20_000, 0),
                        vat_amount: amt(4_200, 0),
                        vat_rate_percent: 21,
                    },
                    VATItemInput {
                        base: amt(5_000, 0),
                        vat_amount: amt(600, 0),
                        vat_rate_percent: 12,
                    },
                ],
            },
        ];

        let result = calculate_vat_return(&invoices, &[]);
        assert_eq!(result.output_vat_base_21, amt(30_000, 0));
        assert_eq!(result.output_vat_amount_21, amt(6_300, 0));
        assert_eq!(result.output_vat_base_12, amt(5_000, 0));
        assert_eq!(result.output_vat_amount_12, amt(600, 0));
        assert_eq!(result.total_output_vat, amt(6_900, 0));
    }

    #[test]
    fn test_expense_100_percent_business() {
        let expenses = vec![VATExpenseInput {
            amount: amt(12_100, 0),    // total including VAT
            vat_amount: amt(2_100, 0), // VAT portion
            vat_rate_percent: 21,
            business_percent: 100,
        }];

        let result = calculate_vat_return(&[], &expenses);
        assert_eq!(result.input_vat_base_21, amt(10_000, 0));
        assert_eq!(result.input_vat_amount_21, amt(2_100, 0));
        assert_eq!(result.total_input_vat, amt(2_100, 0));
        assert_eq!(result.net_vat, amt(-2_100, 0));
    }

    #[test]
    fn test_expense_50_percent_business() {
        let expenses = vec![VATExpenseInput {
            amount: amt(12_100, 0),
            vat_amount: amt(2_100, 0),
            vat_rate_percent: 21,
            business_percent: 50,
        }];

        let result = calculate_vat_return(&[], &expenses);
        assert_eq!(result.input_vat_base_21, amt(5_000, 0));
        assert_eq!(result.input_vat_amount_21, amt(1_050, 0));
        assert_eq!(result.total_input_vat, amt(1_050, 0));
    }

    #[test]
    fn test_expense_zero_percent_means_100() {
        let expenses_zero = vec![VATExpenseInput {
            amount: amt(12_100, 0),
            vat_amount: amt(2_100, 0),
            vat_rate_percent: 21,
            business_percent: 0,
        }];

        let expenses_100 = vec![VATExpenseInput {
            amount: amt(12_100, 0),
            vat_amount: amt(2_100, 0),
            vat_rate_percent: 21,
            business_percent: 100,
        }];

        let r0 = calculate_vat_return(&[], &expenses_zero);
        let r100 = calculate_vat_return(&[], &expenses_100);
        assert_eq!(r0, r100);
    }

    #[test]
    fn test_combined_invoices_and_expenses_net_vat() {
        let invoices = vec![VATInvoiceInput {
            is_credit_note: false,
            items: vec![VATItemInput {
                base: amt(100_000, 0),
                vat_amount: amt(21_000, 0),
                vat_rate_percent: 21,
            }],
        }];

        let expenses = vec![VATExpenseInput {
            amount: amt(60_500, 0),
            vat_amount: amt(10_500, 0),
            vat_rate_percent: 21,
            business_percent: 100,
        }];

        let result = calculate_vat_return(&invoices, &expenses);
        assert_eq!(result.total_output_vat, amt(21_000, 0));
        assert_eq!(result.total_input_vat, amt(10_500, 0));
        assert_eq!(result.net_vat, amt(10_500, 0));
    }

    #[rstest]
    #[case(21, 100)]
    #[case(21, 50)]
    #[case(12, 100)]
    #[case(12, 75)]
    fn test_expense_rates_and_percentages(#[case] vat_rate: i32, #[case] biz_pct: i32) {
        let total = amt(10_000, 0);
        let vat = amt(1_000, 0);
        let expenses = vec![VATExpenseInput {
            amount: total,
            vat_amount: vat,
            vat_rate_percent: vat_rate,
            business_percent: biz_pct,
        }];

        let result = calculate_vat_return(&[], &expenses);
        let factor = biz_pct as f64 / 100.0;
        let expected_vat = vat.multiply(factor);
        let expected_base = (total - vat).multiply(factor);

        match vat_rate {
            21 => {
                assert_eq!(result.input_vat_amount_21, expected_vat);
                assert_eq!(result.input_vat_base_21, expected_base);
            }
            12 => {
                assert_eq!(result.input_vat_amount_12, expected_vat);
                assert_eq!(result.input_vat_base_12, expected_base);
            }
            _ => unreachable!(),
        }
    }
}
