use std::collections::HashMap;

use chrono::{Datelike, NaiveDate};
use zfaktury_domain::Amount;

/// A sell transaction for FIFO cost basis calculation.
pub struct SellTransaction {
    pub id: i64,
    pub asset_name: String,
    pub asset_type: String,
    pub transaction_date: NaiveDate,
    pub quantity: i64,
    pub total_amount: Amount,
    pub fees: Amount,
}

/// A buy transaction for FIFO cost basis calculation.
pub struct BuyTransaction {
    pub id: i64,
    pub asset_name: String,
    pub asset_type: String,
    pub transaction_date: NaiveDate,
    pub quantity: i64,
    pub total_amount: Amount,
}

/// Result of FIFO cost basis calculation for a single sell.
#[derive(Debug, PartialEq, Eq)]
pub struct FIFOResult {
    pub sell_id: i64,
    pub cost_basis: Amount,
    pub computed_gain: Amount,
    pub time_test_exempt: bool,
    pub exempt_amount: Amount,
}

/// Key for grouping transactions by asset.
#[derive(Debug, Clone, PartialEq, Eq, PartialOrd, Ord, Hash)]
struct AssetGroupKey {
    asset_name: String,
    asset_type: String,
}

/// Calculates FIFO cost basis for sell transactions.
///
/// Sells are grouped by (asset_name, asset_type) and processed alphabetically.
/// Within each group, sells are processed chronologically.
/// Buys are matched in FIFO order with consumed quantity tracked across sells.
///
/// Time test: all matched buys must be older than `time_test_years` from sell date.
/// Exemption: if time_test_exempt and gain > 0, exempt up to remaining cumulative limit.
pub fn calculate_fifo(
    sells: &[SellTransaction],
    buys: &[BuyTransaction],
    time_test_years: i32,
    exemption_limit: Amount,
) -> Vec<FIFOResult> {
    if sells.is_empty() {
        return Vec::new();
    }

    // Group sells by (asset_name, asset_type).
    let mut groups: HashMap<AssetGroupKey, Vec<usize>> = HashMap::new();
    for (i, sell) in sells.iter().enumerate() {
        let key = AssetGroupKey {
            asset_name: sell.asset_name.clone(),
            asset_type: sell.asset_type.clone(),
        };
        groups.entry(key).or_default().push(i);
    }

    // Sort group keys for deterministic processing.
    let mut keys: Vec<AssetGroupKey> = groups.keys().cloned().collect();
    keys.sort();

    // Track cumulative exempt amount across all groups.
    let mut cumulative_exempt = Amount::ZERO;
    let mut results: Vec<(usize, FIFOResult)> = Vec::with_capacity(sells.len());

    for key in &keys {
        let sell_indices = &groups[key];

        // Sort sells by transaction date within group.
        let mut sorted_indices = sell_indices.clone();
        sorted_indices.sort_by_key(|&i| sells[i].transaction_date);

        // Get buys for this asset group, sorted by date (FIFO order).
        let mut group_buys: Vec<&BuyTransaction> = buys
            .iter()
            .filter(|b| b.asset_name == key.asset_name && b.asset_type == key.asset_type)
            .collect();
        group_buys.sort_by_key(|b| b.transaction_date);

        // Track consumed quantity per buy ID (shared across all sells of this group).
        let mut consumed: HashMap<i64, i64> = HashMap::new();

        for &sell_idx in &sorted_indices {
            let sell = &sells[sell_idx];
            let mut remaining_qty = sell.quantity;
            let mut cost_basis = Amount::ZERO;
            let mut all_buys_exempt = true;

            let time_test_cutoff = subtract_years(sell.transaction_date, time_test_years);

            for buy in &group_buys {
                if remaining_qty <= 0 {
                    break;
                }

                let available_qty = buy.quantity - consumed.get(&buy.id).copied().unwrap_or(0);
                if available_qty <= 0 {
                    continue;
                }

                let take_qty = available_qty.min(remaining_qty);

                // Proportional cost basis: buy.total_amount * take_qty / buy.quantity
                let buy_cost =
                    Amount::from_halere(buy.total_amount.halere() * take_qty / buy.quantity);
                cost_basis += buy_cost;

                // Time test: buy must be before the cutoff date.
                if buy.transaction_date >= time_test_cutoff {
                    all_buys_exempt = false;
                }

                *consumed.entry(buy.id).or_insert(0) += take_qty;
                remaining_qty -= take_qty;
            }

            // If we couldn't match all sell quantity to buys, not exempt.
            if remaining_qty > 0 {
                all_buys_exempt = false;
            }

            let computed_gain = sell.total_amount - sell.fees - cost_basis;
            let mut time_test_exempt = all_buys_exempt && computed_gain > Amount::ZERO;

            let mut exempt_amount = Amount::ZERO;
            if time_test_exempt {
                exempt_amount = computed_gain;
                // Apply exemption limit if configured (> 0).
                if exemption_limit > Amount::ZERO {
                    let remaining = exemption_limit - cumulative_exempt;
                    if remaining <= Amount::ZERO {
                        exempt_amount = Amount::ZERO;
                        time_test_exempt = false;
                    } else if exempt_amount > remaining {
                        exempt_amount = remaining;
                    }
                }
                cumulative_exempt += exempt_amount;
            }

            results.push((
                sell_idx,
                FIFOResult {
                    sell_id: sell.id,
                    cost_basis,
                    computed_gain,
                    time_test_exempt,
                    exempt_amount,
                },
            ));
        }
    }

    // Return results in original sell order.
    results.sort_by_key(|(idx, _)| *idx);
    results.into_iter().map(|(_, r)| r).collect()
}

/// Subtracts N years from a date, handling leap year edge cases.
fn subtract_years(date: NaiveDate, years: i32) -> NaiveDate {
    let target_year = date.year() - years;
    NaiveDate::from_ymd_opt(target_year, date.month(), date.day()).unwrap_or_else(|| {
        // Feb 29 in a leap year -> Mar 1 in non-leap year (matches Go AddDate behavior)
        NaiveDate::from_ymd_opt(target_year, 3, 1).expect("March 1 always exists")
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    fn date(y: i32, m: u32, d: u32) -> NaiveDate {
        NaiveDate::from_ymd_opt(y, m, d).unwrap()
    }

    // -- Simple FIFO tests --

    #[test]
    fn simple_one_buy_one_sell() {
        let buys = [BuyTransaction {
            id: 1,
            asset_name: "AAPL".into(),
            asset_type: "stock".into(),
            transaction_date: date(2020, 1, 1),
            quantity: 10,
            total_amount: Amount::new(1_000, 0),
        }];
        let sells = [SellTransaction {
            id: 100,
            asset_name: "AAPL".into(),
            asset_type: "stock".into(),
            transaction_date: date(2024, 6, 1),
            quantity: 10,
            total_amount: Amount::new(2_000, 0),
            fees: Amount::new(10, 0),
        }];

        let results = calculate_fifo(&sells, &buys, 3, Amount::ZERO);
        assert_eq!(results.len(), 1);
        assert_eq!(results[0].sell_id, 100);
        assert_eq!(results[0].cost_basis, Amount::new(1_000, 0));
        // gain = 2000 - 10 - 1000 = 990
        assert_eq!(results[0].computed_gain, Amount::new(990, 0));
        assert!(results[0].time_test_exempt); // 2020 -> 2024, 4 years > 3
        assert_eq!(results[0].exempt_amount, Amount::new(990, 0));
    }

    #[test]
    fn multiple_buys_fifo_order() {
        let buys = [
            BuyTransaction {
                id: 1,
                asset_name: "BTC".into(),
                asset_type: "crypto".into(),
                transaction_date: date(2020, 1, 1),
                quantity: 5,
                total_amount: Amount::new(500, 0),
            },
            BuyTransaction {
                id: 2,
                asset_name: "BTC".into(),
                asset_type: "crypto".into(),
                transaction_date: date(2021, 1, 1),
                quantity: 5,
                total_amount: Amount::new(1_000, 0),
            },
        ];
        let sells = [SellTransaction {
            id: 100,
            asset_name: "BTC".into(),
            asset_type: "crypto".into(),
            transaction_date: date(2024, 6, 1),
            quantity: 8,
            total_amount: Amount::new(4_000, 0),
            fees: Amount::ZERO,
        }];

        let results = calculate_fifo(&sells, &buys, 3, Amount::ZERO);
        assert_eq!(results.len(), 1);
        // FIFO: take 5 from buy 1 (cost 500) + 3 from buy 2 (cost 1000*3/5 = 600)
        assert_eq!(
            results[0].cost_basis,
            Amount::new(500, 0) + Amount::new(600, 0)
        );
        // gain = 4000 - 0 - 1100 = 2900
        assert_eq!(results[0].computed_gain, Amount::new(2_900, 0));
    }

    #[test]
    fn partial_sell() {
        let buys = [BuyTransaction {
            id: 1,
            asset_name: "ETH".into(),
            asset_type: "crypto".into(),
            transaction_date: date(2020, 1, 1),
            quantity: 10,
            total_amount: Amount::new(1_000, 0),
        }];
        let sells = [SellTransaction {
            id: 100,
            asset_name: "ETH".into(),
            asset_type: "crypto".into(),
            transaction_date: date(2024, 6, 1),
            quantity: 3,
            total_amount: Amount::new(600, 0),
            fees: Amount::ZERO,
        }];

        let results = calculate_fifo(&sells, &buys, 3, Amount::ZERO);
        assert_eq!(results.len(), 1);
        // Cost basis = 1000 * 3 / 10 = 300
        assert_eq!(results[0].cost_basis, Amount::new(300, 0));
        assert_eq!(results[0].computed_gain, Amount::new(300, 0));
    }

    // -- Time test tests --

    #[test]
    fn time_test_buy_old_enough_exempt() {
        let buys = [BuyTransaction {
            id: 1,
            asset_name: "X".into(),
            asset_type: "stock".into(),
            transaction_date: date(2020, 1, 1),
            quantity: 10,
            total_amount: Amount::new(100, 0),
        }];
        let sells = [SellTransaction {
            id: 100,
            asset_name: "X".into(),
            asset_type: "stock".into(),
            transaction_date: date(2024, 1, 1),
            quantity: 10,
            total_amount: Amount::new(500, 0),
            fees: Amount::ZERO,
        }];

        let results = calculate_fifo(&sells, &buys, 3, Amount::ZERO);
        // Buy 2020-01-01, sell 2024-01-01, cutoff = 2021-01-01
        // Buy is before cutoff -> exempt
        assert!(results[0].time_test_exempt);
    }

    #[test]
    fn time_test_buy_too_recent_not_exempt() {
        let buys = [BuyTransaction {
            id: 1,
            asset_name: "X".into(),
            asset_type: "stock".into(),
            transaction_date: date(2022, 6, 1),
            quantity: 10,
            total_amount: Amount::new(100, 0),
        }];
        let sells = [SellTransaction {
            id: 100,
            asset_name: "X".into(),
            asset_type: "stock".into(),
            transaction_date: date(2024, 1, 1),
            quantity: 10,
            total_amount: Amount::new(500, 0),
            fees: Amount::ZERO,
        }];

        let results = calculate_fifo(&sells, &buys, 3, Amount::ZERO);
        // Buy 2022-06-01, sell 2024-01-01, cutoff = 2021-01-01
        // Buy is NOT before cutoff -> not exempt
        assert!(!results[0].time_test_exempt);
    }

    #[test]
    fn time_test_mixed_buys_not_exempt() {
        let buys = [
            BuyTransaction {
                id: 1,
                asset_name: "X".into(),
                asset_type: "stock".into(),
                transaction_date: date(2019, 1, 1), // old enough
                quantity: 5,
                total_amount: Amount::new(50, 0),
            },
            BuyTransaction {
                id: 2,
                asset_name: "X".into(),
                asset_type: "stock".into(),
                transaction_date: date(2023, 1, 1), // too recent
                quantity: 5,
                total_amount: Amount::new(50, 0),
            },
        ];
        let sells = [SellTransaction {
            id: 100,
            asset_name: "X".into(),
            asset_type: "stock".into(),
            transaction_date: date(2024, 6, 1),
            quantity: 10,
            total_amount: Amount::new(500, 0),
            fees: Amount::ZERO,
        }];

        let results = calculate_fifo(&sells, &buys, 3, Amount::ZERO);
        // Not all buys pass time test -> not exempt
        assert!(!results[0].time_test_exempt);
        assert_eq!(results[0].exempt_amount, Amount::ZERO);
    }

    // -- Exemption limit tests --

    #[test]
    fn cumulative_exemption_limit() {
        let buys = [
            BuyTransaction {
                id: 1,
                asset_name: "AAPL".into(),
                asset_type: "stock".into(),
                transaction_date: date(2020, 1, 1),
                quantity: 10,
                total_amount: Amount::new(100, 0),
            },
            BuyTransaction {
                id: 2,
                asset_name: "GOOG".into(),
                asset_type: "stock".into(),
                transaction_date: date(2020, 1, 1),
                quantity: 10,
                total_amount: Amount::new(200, 0),
            },
        ];
        let sells = [
            SellTransaction {
                id: 100,
                asset_name: "AAPL".into(),
                asset_type: "stock".into(),
                transaction_date: date(2024, 6, 1),
                quantity: 10,
                total_amount: Amount::new(900, 0),
                fees: Amount::ZERO,
            },
            SellTransaction {
                id: 101,
                asset_name: "GOOG".into(),
                asset_type: "stock".into(),
                transaction_date: date(2024, 6, 1),
                quantity: 10,
                total_amount: Amount::new(1_000, 0),
                fees: Amount::ZERO,
            },
        ];

        // Exemption limit = 1000 CZK
        let results = calculate_fifo(&sells, &buys, 3, Amount::new(1_000, 0));
        assert_eq!(results.len(), 2);

        // AAPL: gain = 900 - 100 = 800, exempt 800 (under limit)
        let aapl = results.iter().find(|r| r.sell_id == 100).unwrap();
        assert!(aapl.time_test_exempt);
        assert_eq!(aapl.computed_gain, Amount::new(800, 0));
        assert_eq!(aapl.exempt_amount, Amount::new(800, 0));

        // GOOG: gain = 1000 - 200 = 800, but only 200 remaining (1000 - 800)
        let goog = results.iter().find(|r| r.sell_id == 101).unwrap();
        assert!(goog.time_test_exempt);
        assert_eq!(goog.computed_gain, Amount::new(800, 0));
        assert_eq!(goog.exempt_amount, Amount::new(200, 0));
    }

    #[test]
    fn exemption_limit_exhausted() {
        let buys = [
            BuyTransaction {
                id: 1,
                asset_name: "A".into(),
                asset_type: "stock".into(),
                transaction_date: date(2020, 1, 1),
                quantity: 10,
                total_amount: Amount::new(100, 0),
            },
            BuyTransaction {
                id: 2,
                asset_name: "B".into(),
                asset_type: "stock".into(),
                transaction_date: date(2020, 1, 1),
                quantity: 10,
                total_amount: Amount::new(100, 0),
            },
        ];
        let sells = [
            SellTransaction {
                id: 100,
                asset_name: "A".into(),
                asset_type: "stock".into(),
                transaction_date: date(2024, 6, 1),
                quantity: 10,
                total_amount: Amount::new(600, 0),
                fees: Amount::ZERO,
            },
            SellTransaction {
                id: 101,
                asset_name: "B".into(),
                asset_type: "stock".into(),
                transaction_date: date(2024, 6, 1),
                quantity: 10,
                total_amount: Amount::new(700, 0),
                fees: Amount::ZERO,
            },
        ];

        // Exemption limit = 500 CZK (A's gain is 500, exhausts limit)
        let results = calculate_fifo(&sells, &buys, 3, Amount::new(500, 0));

        let a = results.iter().find(|r| r.sell_id == 100).unwrap();
        assert!(a.time_test_exempt);
        assert_eq!(a.computed_gain, Amount::new(500, 0));
        assert_eq!(a.exempt_amount, Amount::new(500, 0));

        // B: limit exhausted -> not exempt
        let b = results.iter().find(|r| r.sell_id == 101).unwrap();
        assert!(!b.time_test_exempt);
        assert_eq!(b.exempt_amount, Amount::ZERO);
    }

    // -- Multiple asset groups --

    #[test]
    fn multiple_asset_groups_independent() {
        let buys = [
            BuyTransaction {
                id: 1,
                asset_name: "AAPL".into(),
                asset_type: "stock".into(),
                transaction_date: date(2020, 1, 1),
                quantity: 10,
                total_amount: Amount::new(1_000, 0),
            },
            BuyTransaction {
                id: 2,
                asset_name: "BTC".into(),
                asset_type: "crypto".into(),
                transaction_date: date(2020, 1, 1),
                quantity: 5,
                total_amount: Amount::new(500, 0),
            },
        ];
        let sells = [
            SellTransaction {
                id: 100,
                asset_name: "AAPL".into(),
                asset_type: "stock".into(),
                transaction_date: date(2024, 6, 1),
                quantity: 5,
                total_amount: Amount::new(1_000, 0),
                fees: Amount::ZERO,
            },
            SellTransaction {
                id: 101,
                asset_name: "BTC".into(),
                asset_type: "crypto".into(),
                transaction_date: date(2024, 6, 1),
                quantity: 3,
                total_amount: Amount::new(600, 0),
                fees: Amount::ZERO,
            },
        ];

        let results = calculate_fifo(&sells, &buys, 3, Amount::ZERO);
        assert_eq!(results.len(), 2);

        let aapl = results.iter().find(|r| r.sell_id == 100).unwrap();
        // Cost = 1000 * 5 / 10 = 500
        assert_eq!(aapl.cost_basis, Amount::new(500, 0));
        assert_eq!(aapl.computed_gain, Amount::new(500, 0));

        let btc = results.iter().find(|r| r.sell_id == 101).unwrap();
        // Cost = 500 * 3 / 5 = 300
        assert_eq!(btc.cost_basis, Amount::new(300, 0));
        assert_eq!(btc.computed_gain, Amount::new(300, 0));
    }

    // -- Unmatched quantity --

    #[test]
    fn unmatched_quantity_not_exempt() {
        let buys = [BuyTransaction {
            id: 1,
            asset_name: "X".into(),
            asset_type: "stock".into(),
            transaction_date: date(2020, 1, 1),
            quantity: 5,
            total_amount: Amount::new(500, 0),
        }];
        let sells = [SellTransaction {
            id: 100,
            asset_name: "X".into(),
            asset_type: "stock".into(),
            transaction_date: date(2024, 6, 1),
            quantity: 10, // more than available buys
            total_amount: Amount::new(2_000, 0),
            fees: Amount::ZERO,
        }];

        let results = calculate_fifo(&sells, &buys, 3, Amount::ZERO);
        assert_eq!(results.len(), 1);
        // Only 5 of 10 matched
        assert_eq!(results[0].cost_basis, Amount::new(500, 0));
        assert!(!results[0].time_test_exempt);
    }

    // -- Edge cases --

    #[test]
    fn empty_sells() {
        let results = calculate_fifo(&[], &[], 3, Amount::ZERO);
        assert!(results.is_empty());
    }

    #[test]
    fn no_gain_not_exempt() {
        let buys = [BuyTransaction {
            id: 1,
            asset_name: "X".into(),
            asset_type: "stock".into(),
            transaction_date: date(2020, 1, 1),
            quantity: 10,
            total_amount: Amount::new(1_000, 0),
        }];
        let sells = [SellTransaction {
            id: 100,
            asset_name: "X".into(),
            asset_type: "stock".into(),
            transaction_date: date(2024, 6, 1),
            quantity: 10,
            total_amount: Amount::new(800, 0), // loss
            fees: Amount::ZERO,
        }];

        let results = calculate_fifo(&sells, &buys, 3, Amount::ZERO);
        assert_eq!(results[0].computed_gain, Amount::new(-200, 0));
        assert!(!results[0].time_test_exempt);
        assert_eq!(results[0].exempt_amount, Amount::ZERO);
    }

    #[test]
    fn no_limit_means_unlimited_exemption() {
        let buys = [BuyTransaction {
            id: 1,
            asset_name: "X".into(),
            asset_type: "stock".into(),
            transaction_date: date(2020, 1, 1),
            quantity: 10,
            total_amount: Amount::new(100, 0),
        }];
        let sells = [SellTransaction {
            id: 100,
            asset_name: "X".into(),
            asset_type: "stock".into(),
            transaction_date: date(2024, 6, 1),
            quantity: 10,
            total_amount: Amount::new(10_000_000, 0), // huge gain
            fees: Amount::ZERO,
        }];

        // Limit 0 = no limit
        let results = calculate_fifo(&sells, &buys, 3, Amount::ZERO);
        assert!(results[0].time_test_exempt);
        let expected_gain = Amount::new(10_000_000, 0) - Amount::new(100, 0);
        assert_eq!(results[0].exempt_amount, expected_gain);
    }

    #[test]
    fn consumed_quantity_shared_across_sells_in_group() {
        let buys = [BuyTransaction {
            id: 1,
            asset_name: "X".into(),
            asset_type: "stock".into(),
            transaction_date: date(2020, 1, 1),
            quantity: 10,
            total_amount: Amount::new(1_000, 0),
        }];
        let sells = [
            SellTransaction {
                id: 100,
                asset_name: "X".into(),
                asset_type: "stock".into(),
                transaction_date: date(2024, 1, 1),
                quantity: 6,
                total_amount: Amount::new(1_200, 0),
                fees: Amount::ZERO,
            },
            SellTransaction {
                id: 101,
                asset_name: "X".into(),
                asset_type: "stock".into(),
                transaction_date: date(2024, 6, 1),
                quantity: 6, // only 4 remaining from buy
                total_amount: Amount::new(1_200, 0),
                fees: Amount::ZERO,
            },
        ];

        let results = calculate_fifo(&sells, &buys, 3, Amount::ZERO);
        assert_eq!(results.len(), 2);

        let first = results.iter().find(|r| r.sell_id == 100).unwrap();
        // Cost = 1000 * 6 / 10 = 600
        assert_eq!(first.cost_basis, Amount::new(600, 0));
        assert!(first.time_test_exempt);

        let second = results.iter().find(|r| r.sell_id == 101).unwrap();
        // Only 4 matched (10 - 6 = 4), cost = 1000 * 4 / 10 = 400
        assert_eq!(second.cost_basis, Amount::new(400, 0));
        // 2 units unmatched -> not exempt
        assert!(!second.time_test_exempt);
    }
}
