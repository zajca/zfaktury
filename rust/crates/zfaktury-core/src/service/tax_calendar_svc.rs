use chrono::{Datelike, NaiveDate, Weekday};

/// A single tax deadline in the Czech tax calendar.
#[derive(Debug, Clone)]
pub struct TaxDeadline {
    pub name: String,
    pub date: NaiveDate,
    pub deadline_type: String,
    pub description: String,
}

/// Stateless service providing Czech tax calendar with holiday-aware deadline shifting.
pub struct TaxCalendarService;

impl Default for TaxCalendarService {
    fn default() -> Self {
        Self::new()
    }
}

impl TaxCalendarService {
    pub fn new() -> Self {
        Self
    }

    /// Returns all tax deadlines for the given year, shifted to business days.
    pub fn get_deadlines(&self, year: i32) -> Vec<TaxDeadline> {
        let holidays = czech_public_holidays(year);
        let month_names = [
            "January",
            "February",
            "March",
            "April",
            "May",
            "June",
            "July",
            "August",
            "September",
            "October",
            "November",
            "December",
        ];

        let mut deadlines = Vec::new();

        // Monthly VAT: 25th of the following month.
        for m in 1..=12i32 {
            let (due_year, due_month) = if m == 12 {
                (year + 1, 1)
            } else {
                (year, m + 1)
            };
            let due_holidays = if due_year != year {
                czech_public_holidays(due_year)
            } else {
                holidays.clone()
            };
            let date = next_business_day(
                NaiveDate::from_ymd_opt(due_year, due_month as u32, 25).unwrap(),
                &due_holidays,
            );
            deadlines.push(TaxDeadline {
                name: format!("VAT return - {}", month_names[(m - 1) as usize]),
                date,
                deadline_type: "vat".to_string(),
                description: format!(
                    "Monthly VAT return for {} {}",
                    month_names[(m - 1) as usize],
                    year
                ),
            });
        }

        // Income tax: April 1.
        deadlines.push(TaxDeadline {
            name: "Income tax return".to_string(),
            date: next_business_day(NaiveDate::from_ymd_opt(year, 4, 1).unwrap(), &holidays),
            deadline_type: "income_tax".to_string(),
            description: "Annual income tax return filing deadline".to_string(),
        });

        // Social/Health insurance: May 2.
        for (name, dtype, desc) in [
            (
                "Social insurance overview",
                "social",
                "Annual social insurance overview filing deadline",
            ),
            (
                "Health insurance overview",
                "health",
                "Annual health insurance overview filing deadline",
            ),
        ] {
            deadlines.push(TaxDeadline {
                name: name.to_string(),
                date: next_business_day(NaiveDate::from_ymd_opt(year, 5, 2).unwrap(), &holidays),
                deadline_type: dtype.to_string(),
                description: desc.to_string(),
            });
        }

        // Quarterly income tax advances.
        for (month, day, label) in [(3, 15, "Q1"), (6, 15, "Q2"), (9, 15, "Q3"), (12, 15, "Q4")] {
            deadlines.push(TaxDeadline {
                name: format!("Income tax advance {}", label),
                date: next_business_day(
                    NaiveDate::from_ymd_opt(year, month, day).unwrap(),
                    &holidays,
                ),
                deadline_type: "advance".to_string(),
                description: format!("Quarterly income tax advance payment {}", label),
            });
        }

        deadlines.sort_by_key(|d| d.date);
        deadlines
    }
}

fn next_business_day(mut date: NaiveDate, holidays: &[NaiveDate]) -> NaiveDate {
    while matches!(date.weekday(), Weekday::Sat | Weekday::Sun) || holidays.contains(&date) {
        date += chrono::Duration::days(1);
    }
    date
}

fn czech_public_holidays(year: i32) -> Vec<NaiveDate> {
    let easter = easter_sunday(year);
    let good_friday = easter - chrono::Duration::days(2);
    let easter_monday = easter + chrono::Duration::days(1);

    vec![
        NaiveDate::from_ymd_opt(year, 1, 1).unwrap(),
        good_friday,
        easter_monday,
        NaiveDate::from_ymd_opt(year, 5, 1).unwrap(),
        NaiveDate::from_ymd_opt(year, 5, 8).unwrap(),
        NaiveDate::from_ymd_opt(year, 7, 5).unwrap(),
        NaiveDate::from_ymd_opt(year, 7, 6).unwrap(),
        NaiveDate::from_ymd_opt(year, 9, 28).unwrap(),
        NaiveDate::from_ymd_opt(year, 10, 28).unwrap(),
        NaiveDate::from_ymd_opt(year, 11, 17).unwrap(),
        NaiveDate::from_ymd_opt(year, 12, 24).unwrap(),
        NaiveDate::from_ymd_opt(year, 12, 25).unwrap(),
        NaiveDate::from_ymd_opt(year, 12, 26).unwrap(),
    ]
}

/// Computes Easter Sunday using the Anonymous Gregorian algorithm.
fn easter_sunday(year: i32) -> NaiveDate {
    let a = year % 19;
    let b = year / 100;
    let c = year % 100;
    let d = b / 4;
    let e = b % 4;
    let f = (b + 8) / 25;
    let g = (b - f + 1) / 3;
    let h = (19 * a + b - d - g + 15) % 30;
    let i = c / 4;
    let k = c % 4;
    let l = (32 + 2 * e + 2 * i - h - k) % 7;
    let m = (a + 11 * h + 22 * l) / 451;
    let month = (h + l - 7 * m + 114) / 31;
    let day = ((h + l - 7 * m + 114) % 31) + 1;
    NaiveDate::from_ymd_opt(year, month as u32, day as u32).unwrap()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_easter_2024() {
        assert_eq!(
            easter_sunday(2024),
            NaiveDate::from_ymd_opt(2024, 3, 31).unwrap()
        );
    }

    #[test]
    fn test_deadlines_sorted_by_date() {
        let svc = TaxCalendarService::new();
        let deadlines = svc.get_deadlines(2024);
        assert!(!deadlines.is_empty());
        for w in deadlines.windows(2) {
            assert!(w[0].date <= w[1].date);
        }
    }

    #[test]
    fn test_no_weekend_deadlines() {
        let svc = TaxCalendarService::new();
        let deadlines = svc.get_deadlines(2024);
        for d in &deadlines {
            assert!(
                !matches!(d.date.weekday(), Weekday::Sat | Weekday::Sun),
                "Deadline {} falls on weekend: {}",
                d.name,
                d.date
            );
        }
    }
}
