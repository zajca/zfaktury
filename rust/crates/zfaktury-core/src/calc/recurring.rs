use chrono::{Datelike, NaiveDate};
use zfaktury_domain::Frequency;

/// Calculates the next occurrence date based on the given frequency.
///
/// - Weekly: +7 days
/// - Monthly: same day next month, clamped to month end (e.g. Jan 31 -> Feb 28/29)
/// - Quarterly: +3 months with same clamping
/// - Yearly: +1 year, handling Feb 29 -> Feb 28 in non-leap years
pub fn next_occurrence(current: NaiveDate, frequency: &Frequency) -> NaiveDate {
    match frequency {
        Frequency::Weekly => current + chrono::Days::new(7),
        Frequency::Monthly => add_months(current, 1),
        Frequency::Quarterly => add_months(current, 3),
        Frequency::Yearly => add_months(current, 12),
    }
}

/// Adds N months to a date, clamping the day to the last day of the target month.
fn add_months(date: NaiveDate, months: u32) -> NaiveDate {
    let month0 = date.month0() + months;
    let year = date.year() + (month0 / 12) as i32;
    let month = (month0 % 12) + 1;

    let max_day = days_in_month(year, month);
    let day = date.day().min(max_day);

    NaiveDate::from_ymd_opt(year, month, day).unwrap()
}

/// Returns the number of days in a given month.
fn days_in_month(year: i32, month: u32) -> u32 {
    if month == 12 {
        NaiveDate::from_ymd_opt(year + 1, 1, 1)
    } else {
        NaiveDate::from_ymd_opt(year, month + 1, 1)
    }
    .unwrap()
    .pred_opt()
    .unwrap()
    .day()
}

#[cfg(test)]
mod tests {
    use super::*;
    use rstest::rstest;

    fn date(y: i32, m: u32, d: u32) -> NaiveDate {
        NaiveDate::from_ymd_opt(y, m, d).unwrap()
    }

    // -- Weekly --

    #[test]
    fn weekly_simple() {
        assert_eq!(
            next_occurrence(date(2024, 3, 1), &Frequency::Weekly),
            date(2024, 3, 8)
        );
    }

    #[test]
    fn weekly_across_month() {
        assert_eq!(
            next_occurrence(date(2024, 3, 28), &Frequency::Weekly),
            date(2024, 4, 4)
        );
    }

    // -- Monthly --

    #[rstest]
    #[case(date(2024, 1, 15), date(2024, 2, 15))]
    #[case(date(2024, 1, 31), date(2024, 2, 29))] // leap year 2024
    #[case(date(2023, 1, 31), date(2023, 2, 28))] // non-leap year
    #[case(date(2024, 3, 31), date(2024, 4, 30))] // March 31 -> April 30
    #[case(date(2024, 12, 15), date(2025, 1, 15))] // year rollover
    fn monthly(#[case] input: NaiveDate, #[case] expected: NaiveDate) {
        assert_eq!(next_occurrence(input, &Frequency::Monthly), expected);
    }

    // -- Quarterly --

    #[rstest]
    #[case(date(2024, 1, 31), date(2024, 4, 30))] // Jan 31 -> Apr 30
    #[case(date(2024, 1, 15), date(2024, 4, 15))] // normal
    #[case(date(2024, 11, 15), date(2025, 2, 15))] // year rollover
    #[case(date(2024, 11, 30), date(2025, 2, 28))] // Nov 30 -> Feb 28
    fn quarterly(#[case] input: NaiveDate, #[case] expected: NaiveDate) {
        assert_eq!(next_occurrence(input, &Frequency::Quarterly), expected);
    }

    // -- Yearly --

    #[rstest]
    #[case(date(2024, 2, 29), date(2025, 2, 28))] // leap -> non-leap
    #[case(date(2023, 2, 28), date(2024, 2, 28))] // non-leap -> leap (stays 28)
    #[case(date(2024, 6, 15), date(2025, 6, 15))] // normal
    #[case(date(2024, 1, 31), date(2025, 1, 31))] // Jan 31 stays Jan 31
    fn yearly(#[case] input: NaiveDate, #[case] expected: NaiveDate) {
        assert_eq!(next_occurrence(input, &Frequency::Yearly), expected);
    }
}
