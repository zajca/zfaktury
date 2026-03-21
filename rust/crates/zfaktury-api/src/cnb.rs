use std::collections::HashMap;
use std::time::Duration;

use chrono::NaiveDate;

use crate::error::{ApiError, Result};

const DEFAULT_BASE_URL: &str = "https://www.cnb.cz/cs/financni-trhy/devizovy-trh/kurzy-devizoveho-trhu/kurzy-devizoveho-trhu/denni_kurz.txt";
const DEFAULT_TIMEOUT: Duration = Duration::from_secs(10);
const MAX_FALLBACK_DAYS: i32 = 5;

/// Exchange rate entry from the CNB daily rate sheet.
#[derive(Debug, Clone)]
pub struct ExchangeRate {
    pub country: String,
    pub currency: String,
    /// How many units the rate applies to.
    pub amount: i32,
    /// ISO 4217 currency code.
    pub code: String,
    /// CZK per `amount` units.
    pub rate: f64,
}

/// Client for the Czech National Bank exchange rate API.
pub struct CnbClient {
    base_url: String,
    client: reqwest::blocking::Client,
}

impl Default for CnbClient {
    fn default() -> Self {
        Self::new()
    }
}

impl CnbClient {
    /// Create a new CNB client with default settings.
    pub fn new() -> Self {
        Self {
            base_url: DEFAULT_BASE_URL.to_string(),
            client: reqwest::blocking::Client::builder()
                .timeout(DEFAULT_TIMEOUT)
                .build()
                .expect("failed to build HTTP client"),
        }
    }

    /// Create a new CNB client with a custom base URL.
    pub fn with_base_url(base_url: &str) -> Self {
        Self {
            base_url: base_url.to_string(),
            client: reqwest::blocking::Client::builder()
                .timeout(DEFAULT_TIMEOUT)
                .build()
                .expect("failed to build HTTP client"),
        }
    }

    /// Get the CZK exchange rate per 1 unit of the given foreign currency
    /// for the specified date. If no rates are available for the exact date
    /// (weekends, holidays), tries up to 5 previous days.
    pub fn get_rate(&self, currency_code: &str, date: NaiveDate) -> Result<f64> {
        let currency_code = currency_code.to_uppercase();
        if currency_code.len() != 3 {
            return Err(ApiError::InvalidInput(format!(
                "invalid currency code: {}",
                currency_code
            )));
        }

        for i in 0..MAX_FALLBACK_DAYS {
            let d = date - chrono::Duration::days(i64::from(i));
            match self.fetch_rates_for_date(d) {
                Ok(rates) => {
                    if let Some(rate) = rates.get(&currency_code) {
                        return Ok(rate.rate / f64::from(rate.amount));
                    }
                    // Currency not found on this day, try previous day.
                    continue;
                }
                Err(_) => continue,
            }
        }

        Err(ApiError::ParseError(format!(
            "no exchange rates available for {} within {} days of {}",
            currency_code, MAX_FALLBACK_DAYS, date
        )))
    }

    /// Fetch and parse rates for a specific date.
    fn fetch_rates_for_date(&self, date: NaiveDate) -> Result<HashMap<String, ExchangeRate>> {
        let date_str = date.format("%d.%m.%Y").to_string();
        let url = format!("{}?date={}", self.base_url, date_str);

        let resp = self.client.get(&url).send()?;

        if resp.status().as_u16() != 200 {
            return Err(ApiError::HttpError {
                status: resp.status().as_u16(),
                body: format!("CNB returned status {}", resp.status()),
            });
        }

        let body = resp.text()?;
        parse_rates(&body)
    }
}

/// Parse the pipe-delimited CNB exchange rate format.
///
/// Format:
///   Line 1: date + sequence number (e.g., "10.03.2026 #049")
///   Line 2: column headers (zeme|mena|mnozstvi|kod|kurz)
///   Lines 3+: data rows (country|currency|amount|code|rate)
///
/// Rate uses comma as decimal separator (e.g., "25,340").
fn parse_rates(body: &str) -> Result<HashMap<String, ExchangeRate>> {
    let mut rates = HashMap::new();

    for (i, line) in body.lines().enumerate() {
        // Skip first two header lines.
        if i < 2 {
            continue;
        }

        let line = line.trim();
        if line.is_empty() {
            continue;
        }

        let parts: Vec<&str> = line.split('|').collect();
        if parts.len() != 5 {
            continue;
        }

        let amount: i32 = match parts[2].trim().parse() {
            Ok(v) => v,
            Err(_) => continue,
        };

        // Rate uses comma as decimal separator.
        let rate_str = parts[4].trim().replace(',', ".");
        let rate: f64 = match rate_str.parse() {
            Ok(v) => v,
            Err(_) => continue,
        };

        let code = parts[3].trim().to_string();
        rates.insert(
            code.clone(),
            ExchangeRate {
                country: parts[0].trim().to_string(),
                currency: parts[1].trim().to_string(),
                amount,
                code,
                rate,
            },
        );
    }

    if rates.is_empty() {
        return Err(ApiError::ParseError(
            "no rates parsed from CNB response".to_string(),
        ));
    }

    Ok(rates)
}

#[cfg(test)]
mod tests {
    use super::*;
    use wiremock::matchers::{method, query_param};
    use wiremock::{Mock, MockServer, ResponseTemplate};

    const SAMPLE_CNB_RESPONSE: &str = "\
10.03.2026 #049
zeme|mena|mnozstvi|kod|kurz
Australie|dolar|1|AUD|15,432
EMU|euro|1|EUR|25,340
USA|dolar|1|USD|23,456
Japonsko|jen|100|JPY|15,789
";

    async fn run_blocking<F, R>(f: F) -> R
    where
        F: FnOnce() -> R + Send + 'static,
        R: Send + 'static,
    {
        tokio::task::spawn_blocking(f).await.unwrap()
    }

    #[tokio::test]
    async fn test_get_rate_success() {
        let mock_server = MockServer::start().await;

        Mock::given(method("GET"))
            .and(query_param("date", "10.03.2026"))
            .respond_with(ResponseTemplate::new(200).set_body_string(SAMPLE_CNB_RESPONSE))
            .mount(&mock_server)
            .await;

        let uri = mock_server.uri();
        let (eur_rate, usd_rate) = run_blocking(move || {
            let client = CnbClient::with_base_url(&uri);
            let date = NaiveDate::from_ymd_opt(2026, 3, 10).unwrap();
            let eur = client.get_rate("EUR", date).unwrap();
            let usd = client.get_rate("USD", date).unwrap();
            (eur, usd)
        })
        .await;

        assert!((eur_rate - 25.340).abs() < 0.001);
        assert!((usd_rate - 23.456).abs() < 0.001);
    }

    #[tokio::test]
    async fn test_get_rate_yen_amount_division() {
        let mock_server = MockServer::start().await;

        Mock::given(method("GET"))
            .and(query_param("date", "10.03.2026"))
            .respond_with(ResponseTemplate::new(200).set_body_string(SAMPLE_CNB_RESPONSE))
            .mount(&mock_server)
            .await;

        let uri = mock_server.uri();
        let rate = run_blocking(move || {
            let client = CnbClient::with_base_url(&uri);
            let date = NaiveDate::from_ymd_opt(2026, 3, 10).unwrap();
            client.get_rate("JPY", date).unwrap()
        })
        .await;

        assert!((rate - 0.15789).abs() < 0.00001);
    }

    #[tokio::test]
    async fn test_get_rate_case_insensitive() {
        let mock_server = MockServer::start().await;

        Mock::given(method("GET"))
            .and(query_param("date", "10.03.2026"))
            .respond_with(ResponseTemplate::new(200).set_body_string(SAMPLE_CNB_RESPONSE))
            .mount(&mock_server)
            .await;

        let uri = mock_server.uri();
        let rate = run_blocking(move || {
            let client = CnbClient::with_base_url(&uri);
            let date = NaiveDate::from_ymd_opt(2026, 3, 10).unwrap();
            client.get_rate("eur", date).unwrap()
        })
        .await;

        assert!((rate - 25.340).abs() < 0.001);
    }

    #[tokio::test]
    async fn test_get_rate_weekend_fallback() {
        let mock_server = MockServer::start().await;

        Mock::given(method("GET"))
            .and(query_param("date", "14.03.2026")) // Saturday
            .respond_with(ResponseTemplate::new(404))
            .mount(&mock_server)
            .await;

        Mock::given(method("GET"))
            .and(query_param("date", "13.03.2026")) // Friday
            .respond_with(ResponseTemplate::new(200).set_body_string(SAMPLE_CNB_RESPONSE))
            .mount(&mock_server)
            .await;

        let uri = mock_server.uri();
        let rate = run_blocking(move || {
            let client = CnbClient::with_base_url(&uri);
            let saturday = NaiveDate::from_ymd_opt(2026, 3, 14).unwrap();
            client.get_rate("EUR", saturday).unwrap()
        })
        .await;

        assert!((rate - 25.340).abs() < 0.001);
    }

    #[test]
    fn test_get_rate_invalid_currency_code() {
        let client = CnbClient::new();
        let date = NaiveDate::from_ymd_opt(2026, 3, 10).unwrap();

        let result = client.get_rate("EU", date);
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), ApiError::InvalidInput(_)));
    }

    #[test]
    fn test_parse_rates_comma_decimal() {
        let rates = parse_rates(SAMPLE_CNB_RESPONSE).unwrap();
        assert_eq!(rates.len(), 4);

        let eur = rates.get("EUR").unwrap();
        assert_eq!(eur.country, "EMU");
        assert_eq!(eur.currency, "euro");
        assert_eq!(eur.amount, 1);
        assert!((eur.rate - 25.340).abs() < 0.001);

        let jpy = rates.get("JPY").unwrap();
        assert_eq!(jpy.amount, 100);
        assert!((jpy.rate - 15.789).abs() < 0.001);
    }

    #[test]
    fn test_parse_rates_empty() {
        let result = parse_rates("header\nheader\n");
        assert!(result.is_err());
    }
}
