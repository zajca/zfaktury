use chrono::NaiveDate;
use zfaktury_domain::{Amount, InvoiceStatus, InvoiceType};

/// Minimal invoice data needed for annual tax base calculation.
pub struct InvoiceForBase {
    pub id: i64,
    pub invoice_type: InvoiceType,
    pub status: InvoiceStatus,
    pub delivery_date: Option<NaiveDate>,
    pub issue_date: NaiveDate,
    pub subtotal_amount: Amount,
}

/// Minimal expense data needed for annual tax base calculation.
pub struct ExpenseForBase {
    pub id: i64,
    pub issue_date: NaiveDate,
    pub amount: Amount,
    pub vat_amount: Amount,
    pub business_percent: i32,
    pub tax_reviewed: bool,
}

/// Result of calculating annual revenue and expenses.
#[derive(Debug, PartialEq, Eq)]
pub struct AnnualBaseResult {
    pub revenue: Amount,
    pub expenses: Amount,
    pub invoice_ids: Vec<i64>,
    pub expense_ids: Vec<i64>,
}

/// Computes aggregate revenue and expenses for the given year.
///
/// Invoices are filtered by effective date (delivery_date if present, else issue_date),
/// must be in the year range, and have status Sent/Paid/Overdue. Proforma invoices are
/// excluded. CreditNote invoices subtract from revenue.
///
/// Expenses are filtered by issue_date in the year range and must be tax_reviewed.
/// The expense amount is (amount - vat_amount) * business_percent / 100.
pub fn calculate_annual_totals(
    invoices: &[InvoiceForBase],
    expenses: &[ExpenseForBase],
    year: i32,
) -> AnnualBaseResult {
    let date_from = NaiveDate::from_ymd_opt(year, 1, 1).unwrap();
    let date_to = NaiveDate::from_ymd_opt(year, 12, 31).unwrap();

    let mut result = AnnualBaseResult {
        revenue: Amount::ZERO,
        expenses: Amount::ZERO,
        invoice_ids: Vec::new(),
        expense_ids: Vec::new(),
    };

    for inv in invoices {
        let effective_date = inv.delivery_date.unwrap_or(inv.issue_date);

        if effective_date < date_from || effective_date > date_to {
            continue;
        }

        if !matches!(
            inv.status,
            InvoiceStatus::Sent | InvoiceStatus::Paid | InvoiceStatus::Overdue
        ) {
            continue;
        }

        if inv.invoice_type == InvoiceType::Proforma {
            continue;
        }

        if inv.invoice_type == InvoiceType::CreditNote {
            result.revenue -= inv.subtotal_amount;
        } else {
            result.revenue += inv.subtotal_amount;
        }
        result.invoice_ids.push(inv.id);
    }

    for exp in expenses {
        if exp.issue_date < date_from || exp.issue_date > date_to {
            continue;
        }

        if !exp.tax_reviewed {
            continue;
        }

        let base_amount = exp.amount - exp.vat_amount;

        let business_pct = if exp.business_percent == 0 {
            100
        } else {
            exp.business_percent
        };

        result.expenses += base_amount.multiply(business_pct as f64 / 100.0);
        result.expense_ids.push(exp.id);
    }

    result
}

#[cfg(test)]
mod tests {
    use super::*;

    fn date(y: i32, m: u32, d: u32) -> NaiveDate {
        NaiveDate::from_ymd_opt(y, m, d).unwrap()
    }

    fn invoice(
        id: i64,
        inv_type: InvoiceType,
        status: InvoiceStatus,
        delivery_date: Option<NaiveDate>,
        issue_date: NaiveDate,
        subtotal: Amount,
    ) -> InvoiceForBase {
        InvoiceForBase {
            id,
            invoice_type: inv_type,
            status,
            delivery_date,
            issue_date,
            subtotal_amount: subtotal,
        }
    }

    fn expense(
        id: i64,
        issue_date: NaiveDate,
        amount: Amount,
        vat: Amount,
        biz_pct: i32,
        reviewed: bool,
    ) -> ExpenseForBase {
        ExpenseForBase {
            id,
            issue_date,
            amount,
            vat_amount: vat,
            business_percent: biz_pct,
            tax_reviewed: reviewed,
        }
    }

    // -- Invoice filtering --

    #[test]
    fn only_sent_paid_overdue_included() {
        let invoices = vec![
            invoice(
                1,
                InvoiceType::Regular,
                InvoiceStatus::Sent,
                None,
                date(2024, 6, 1),
                Amount::new(100, 0),
            ),
            invoice(
                2,
                InvoiceType::Regular,
                InvoiceStatus::Paid,
                None,
                date(2024, 6, 1),
                Amount::new(200, 0),
            ),
            invoice(
                3,
                InvoiceType::Regular,
                InvoiceStatus::Overdue,
                None,
                date(2024, 6, 1),
                Amount::new(300, 0),
            ),
            invoice(
                4,
                InvoiceType::Regular,
                InvoiceStatus::Draft,
                None,
                date(2024, 6, 1),
                Amount::new(400, 0),
            ),
            invoice(
                5,
                InvoiceType::Regular,
                InvoiceStatus::Cancelled,
                None,
                date(2024, 6, 1),
                Amount::new(500, 0),
            ),
        ];

        let result = calculate_annual_totals(&invoices, &[], 2024);
        assert_eq!(result.revenue, Amount::new(600, 0)); // 100 + 200 + 300
        assert_eq!(result.invoice_ids, vec![1, 2, 3]);
    }

    #[test]
    fn proforma_excluded() {
        let invoices = vec![
            invoice(
                1,
                InvoiceType::Proforma,
                InvoiceStatus::Sent,
                None,
                date(2024, 6, 1),
                Amount::new(100, 0),
            ),
            invoice(
                2,
                InvoiceType::Regular,
                InvoiceStatus::Sent,
                None,
                date(2024, 6, 1),
                Amount::new(200, 0),
            ),
        ];

        let result = calculate_annual_totals(&invoices, &[], 2024);
        assert_eq!(result.revenue, Amount::new(200, 0));
        assert_eq!(result.invoice_ids, vec![2]);
    }

    #[test]
    fn credit_note_subtracts() {
        let invoices = vec![
            invoice(
                1,
                InvoiceType::Regular,
                InvoiceStatus::Paid,
                None,
                date(2024, 6, 1),
                Amount::new(1_000, 0),
            ),
            invoice(
                2,
                InvoiceType::CreditNote,
                InvoiceStatus::Sent,
                None,
                date(2024, 6, 1),
                Amount::new(200, 0),
            ),
        ];

        let result = calculate_annual_totals(&invoices, &[], 2024);
        assert_eq!(result.revenue, Amount::new(800, 0));
        assert_eq!(result.invoice_ids, vec![1, 2]);
    }

    #[test]
    fn delivery_date_preferred_over_issue_date() {
        let invoices = vec![
            // Delivery date in 2024 but issue date in 2023 -> included in 2024
            invoice(
                1,
                InvoiceType::Regular,
                InvoiceStatus::Paid,
                Some(date(2024, 3, 15)),
                date(2023, 12, 20),
                Amount::new(500, 0),
            ),
            // Delivery date in 2023 but issue date in 2024 -> NOT included in 2024
            invoice(
                2,
                InvoiceType::Regular,
                InvoiceStatus::Paid,
                Some(date(2023, 12, 15)),
                date(2024, 1, 5),
                Amount::new(300, 0),
            ),
        ];

        let result = calculate_annual_totals(&invoices, &[], 2024);
        assert_eq!(result.revenue, Amount::new(500, 0));
        assert_eq!(result.invoice_ids, vec![1]);
    }

    #[test]
    fn year_boundary_filtering() {
        let invoices = vec![
            invoice(
                1,
                InvoiceType::Regular,
                InvoiceStatus::Paid,
                None,
                date(2023, 12, 31),
                Amount::new(100, 0),
            ),
            invoice(
                2,
                InvoiceType::Regular,
                InvoiceStatus::Paid,
                None,
                date(2024, 1, 1),
                Amount::new(200, 0),
            ),
            invoice(
                3,
                InvoiceType::Regular,
                InvoiceStatus::Paid,
                None,
                date(2024, 12, 31),
                Amount::new(300, 0),
            ),
            invoice(
                4,
                InvoiceType::Regular,
                InvoiceStatus::Paid,
                None,
                date(2025, 1, 1),
                Amount::new(400, 0),
            ),
        ];

        let result = calculate_annual_totals(&invoices, &[], 2024);
        assert_eq!(result.revenue, Amount::new(500, 0)); // 200 + 300
        assert_eq!(result.invoice_ids, vec![2, 3]);
    }

    // -- Expense filtering --

    #[test]
    fn expense_business_percent() {
        let expenses = vec![expense(
            1,
            date(2024, 6, 1),
            Amount::new(1_000, 0),
            Amount::new(210, 0),
            50,
            true,
        )];

        let result = calculate_annual_totals(&[], &expenses, 2024);
        // base = 1000 - 210 = 790, applied = 790 * 50% = 395
        assert_eq!(result.expenses, Amount::new(395, 0));
        assert_eq!(result.expense_ids, vec![1]);
    }

    #[test]
    fn expense_zero_percent_defaults_to_100() {
        let expenses = vec![expense(
            1,
            date(2024, 6, 1),
            Amount::new(1_000, 0),
            Amount::new(210, 0),
            0,
            true,
        )];

        let result = calculate_annual_totals(&[], &expenses, 2024);
        // base = 1000 - 210 = 790, percent 0 -> default 100%, applied = 790
        assert_eq!(result.expenses, Amount::new(790, 0));
    }

    #[test]
    fn only_tax_reviewed_expenses_included() {
        let expenses = vec![
            expense(
                1,
                date(2024, 6, 1),
                Amount::new(500, 0),
                Amount::ZERO,
                100,
                true,
            ),
            expense(
                2,
                date(2024, 6, 1),
                Amount::new(300, 0),
                Amount::ZERO,
                100,
                false,
            ),
        ];

        let result = calculate_annual_totals(&[], &expenses, 2024);
        assert_eq!(result.expenses, Amount::new(500, 0));
        assert_eq!(result.expense_ids, vec![1]);
    }

    #[test]
    fn expense_year_boundary() {
        let expenses = vec![
            expense(
                1,
                date(2023, 12, 31),
                Amount::new(100, 0),
                Amount::ZERO,
                100,
                true,
            ),
            expense(
                2,
                date(2024, 1, 1),
                Amount::new(200, 0),
                Amount::ZERO,
                100,
                true,
            ),
            expense(
                3,
                date(2024, 12, 31),
                Amount::new(300, 0),
                Amount::ZERO,
                100,
                true,
            ),
            expense(
                4,
                date(2025, 1, 1),
                Amount::new(400, 0),
                Amount::ZERO,
                100,
                true,
            ),
        ];

        let result = calculate_annual_totals(&[], &expenses, 2024);
        assert_eq!(result.expenses, Amount::new(500, 0)); // 200 + 300
        assert_eq!(result.expense_ids, vec![2, 3]);
    }

    #[test]
    fn combined_invoices_and_expenses() {
        let invoices = vec![
            invoice(
                1,
                InvoiceType::Regular,
                InvoiceStatus::Paid,
                None,
                date(2024, 3, 1),
                Amount::new(5_000, 0),
            ),
            invoice(
                2,
                InvoiceType::CreditNote,
                InvoiceStatus::Sent,
                None,
                date(2024, 4, 1),
                Amount::new(500, 0),
            ),
        ];
        let expenses = vec![expense(
            10,
            date(2024, 2, 1),
            Amount::new(2_000, 0),
            Amount::new(420, 0),
            80,
            true,
        )];

        let result = calculate_annual_totals(&invoices, &expenses, 2024);
        // Revenue: 5000 - 500 = 4500
        assert_eq!(result.revenue, Amount::new(4_500, 0));
        // Expense: (2000 - 420) * 80% = 1580 * 0.8 = 1264
        assert_eq!(result.expenses, Amount::new(1_264, 0));
    }

    #[test]
    fn empty_inputs() {
        let result = calculate_annual_totals(&[], &[], 2024);
        assert_eq!(result.revenue, Amount::ZERO);
        assert_eq!(result.expenses, Amount::ZERO);
        assert!(result.invoice_ids.is_empty());
        assert!(result.expense_ids.is_empty());
    }
}
