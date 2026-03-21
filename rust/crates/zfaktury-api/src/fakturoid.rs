use std::thread;
use std::time::Duration;

use serde::de::{self, Visitor};
use serde::{Deserialize, Deserializer, Serialize};

use crate::error::{ApiError, Result};

const DEFAULT_TIMEOUT: Duration = Duration::from_secs(30);
const PAGE_DELAY: Duration = Duration::from_millis(700);
const TOKEN_URL: &str = "https://app.fakturoid.cz/api/v3/oauth/token";

/// Custom deserializer for values that can be either JSON numbers or strings.
/// Fakturoid API returns some numeric fields (e.g., exchange_rate) as strings.
#[derive(Debug, Clone, Copy, Default)]
pub struct FlexFloat64(pub f64);

impl<'de> Deserialize<'de> for FlexFloat64 {
    fn deserialize<D>(deserializer: D) -> std::result::Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct FlexFloat64Visitor;

        impl<'de> Visitor<'de> for FlexFloat64Visitor {
            type Value = FlexFloat64;

            fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
                formatter.write_str("a number or a string containing a number")
            }

            fn visit_f64<E: de::Error>(self, v: f64) -> std::result::Result<FlexFloat64, E> {
                Ok(FlexFloat64(v))
            }

            fn visit_i64<E: de::Error>(self, v: i64) -> std::result::Result<FlexFloat64, E> {
                Ok(FlexFloat64(v as f64))
            }

            fn visit_u64<E: de::Error>(self, v: u64) -> std::result::Result<FlexFloat64, E> {
                Ok(FlexFloat64(v as f64))
            }

            fn visit_str<E: de::Error>(self, v: &str) -> std::result::Result<FlexFloat64, E> {
                v.parse::<f64>()
                    .map(FlexFloat64)
                    .map_err(|_| de::Error::custom(format!("cannot parse '{}' as float64", v)))
            }

            fn visit_none<E: de::Error>(self) -> std::result::Result<FlexFloat64, E> {
                Ok(FlexFloat64(0.0))
            }

            fn visit_unit<E: de::Error>(self) -> std::result::Result<FlexFloat64, E> {
                Ok(FlexFloat64(0.0))
            }
        }

        deserializer.deserialize_any(FlexFloat64Visitor)
    }
}

impl Serialize for FlexFloat64 {
    fn serialize<S>(&self, serializer: S) -> std::result::Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_f64(self.0)
    }
}

/// A Fakturoid subject (contact).
#[derive(Debug, Clone, Deserialize)]
pub struct Subject {
    pub id: i64,
    #[serde(default)]
    pub name: String,
    /// ICO.
    #[serde(default)]
    pub registration_no: String,
    /// DIC.
    #[serde(default)]
    pub vat_no: String,
    #[serde(default)]
    pub street: String,
    #[serde(default)]
    pub city: String,
    #[serde(default)]
    pub zip: String,
    #[serde(default)]
    pub country: String,
    /// "cislo/kod" format.
    #[serde(default)]
    pub bank_account: String,
    #[serde(default)]
    pub iban: String,
    #[serde(default)]
    pub email: String,
    #[serde(default)]
    pub phone: String,
    #[serde(default)]
    pub web: String,
    /// "customer", "supplier", "both".
    #[serde(default, rename = "type")]
    pub subject_type: String,
    /// Payment terms in days.
    #[serde(default)]
    pub due: i32,
}

/// A file attachment on a Fakturoid entity.
#[derive(Debug, Clone, Deserialize)]
pub struct Attachment {
    pub id: i64,
    #[serde(default)]
    pub filename: String,
    #[serde(default)]
    pub content_type: String,
    #[serde(default)]
    pub download_url: String,
}

/// A line item on a Fakturoid invoice.
#[derive(Debug, Clone, Deserialize)]
pub struct InvoiceLine {
    #[serde(default)]
    pub name: String,
    #[serde(default)]
    pub quantity: FlexFloat64,
    #[serde(default)]
    pub unit_name: String,
    #[serde(default)]
    pub unit_price: FlexFloat64,
    #[serde(default)]
    pub vat_rate: FlexFloat64,
}

/// A payment on a Fakturoid invoice.
#[derive(Debug, Clone, Deserialize)]
pub struct Payment {
    /// "YYYY-MM-DD".
    #[serde(default)]
    pub paid_on: String,
}

/// A Fakturoid invoice.
#[derive(Debug, Clone, Deserialize)]
pub struct Invoice {
    pub id: i64,
    #[serde(default)]
    pub number: String,
    /// "invoice", "proforma", "partial_proforma", "correction", "tax_document", "final_invoice".
    #[serde(default)]
    pub document_type: String,
    /// "open", "sent", "overdue", "paid", "cancelled", "uncollectible".
    #[serde(default)]
    pub status: String,
    /// "YYYY-MM-DD".
    #[serde(default)]
    pub issued_on: String,
    #[serde(default)]
    pub due_on: String,
    #[serde(default)]
    pub taxable_fulfillment_due: String,
    #[serde(default)]
    pub variable_symbol: String,
    #[serde(default)]
    pub subject_id: i64,
    #[serde(default)]
    pub currency: String,
    #[serde(default)]
    pub exchange_rate: FlexFloat64,
    #[serde(default)]
    pub subtotal: FlexFloat64,
    #[serde(default)]
    pub total: FlexFloat64,
    /// "bank", "cash", "card", "cod", "paypal", "custom".
    #[serde(default)]
    pub payment_method: String,
    #[serde(default)]
    pub note: String,
    #[serde(default)]
    pub lines: Vec<InvoiceLine>,
    #[serde(default)]
    pub payments: Vec<Payment>,
    #[serde(default)]
    pub attachments: Vec<Attachment>,
}

/// A line item on a Fakturoid expense.
#[derive(Debug, Clone, Deserialize)]
pub struct ExpenseLine {
    #[serde(default)]
    pub name: String,
    #[serde(default)]
    pub quantity: FlexFloat64,
    #[serde(default)]
    pub unit_price: FlexFloat64,
    #[serde(default)]
    pub vat_rate: FlexFloat64,
}

/// A Fakturoid expense.
#[derive(Debug, Clone, Deserialize)]
pub struct Expense {
    pub id: i64,
    #[serde(default)]
    pub original_number: String,
    #[serde(default)]
    pub issued_on: String,
    #[serde(default)]
    pub subject_id: i64,
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub total: FlexFloat64,
    #[serde(default)]
    pub currency: String,
    #[serde(default)]
    pub exchange_rate: FlexFloat64,
    /// "bank", "cash", "card", etc.
    #[serde(default)]
    pub payment_method: String,
    #[serde(default)]
    pub private_note: String,
    #[serde(default)]
    pub lines: Vec<ExpenseLine>,
    #[serde(default)]
    pub attachments: Vec<Attachment>,
}

/// Client for the Fakturoid API v3 with OAuth2 Client Credentials.
const FAKTUROID_HOST: &str = "app.fakturoid.cz";

pub struct FakturoidClient {
    base_url: String,
    token_url: String,
    email: String,
    client_id: String,
    client_secret: String,
    access_token: Option<String>,
    /// Allowed host for attachment downloads (SSRF protection).
    allowed_download_host: String,
    client: reqwest::blocking::Client,
}

impl FakturoidClient {
    /// Create a new Fakturoid client.
    ///
    /// - `slug`: Fakturoid account slug
    /// - `email`: user's email (for User-Agent header)
    /// - `client_id`/`client_secret`: OAuth2 credentials
    pub fn new(slug: &str, email: &str, client_id: &str, client_secret: &str) -> Self {
        Self {
            base_url: format!("https://app.fakturoid.cz/api/v3/accounts/{}", slug),
            token_url: TOKEN_URL.to_string(),
            email: email.to_string(),
            client_id: client_id.to_string(),
            client_secret: client_secret.to_string(),
            access_token: None,
            allowed_download_host: FAKTUROID_HOST.to_string(),
            client: reqwest::blocking::Client::builder()
                .timeout(DEFAULT_TIMEOUT)
                .build()
                .expect("failed to build HTTP client"),
        }
    }

    /// Override the base URL and token URL (for testing).
    pub fn with_urls(mut self, base_url: &str, token_url: &str) -> Self {
        // Derive allowed download host from the base URL for testing.
        if let Ok(parsed) = url::Url::parse(base_url)
            && let Some(host) = parsed.host_str()
        {
            let port_suffix = parsed.port().map(|p| format!(":{}", p)).unwrap_or_default();
            self.allowed_download_host = format!("{}{}", host, port_suffix);
        }
        self.base_url = base_url.to_string();
        self.token_url = token_url.to_string();
        self
    }

    /// Obtain an OAuth2 access token using Client Credentials flow.
    /// Must be called before making API requests.
    pub fn authenticate(&mut self) -> Result<()> {
        let resp = self
            .client
            .post(&self.token_url)
            .basic_auth(&self.client_id, Some(&self.client_secret))
            .header("Content-Type", "application/x-www-form-urlencoded")
            .header("Accept", "application/json")
            .header("User-Agent", format!("ZFaktury ({})", self.email))
            .body("grant_type=client_credentials")
            .send()?;

        if resp.status().as_u16() != 200 {
            let status = resp.status().as_u16();
            let body = resp.text().unwrap_or_default();
            return Err(ApiError::OAuthError(format!("HTTP {}: {}", status, body)));
        }

        #[derive(Deserialize)]
        struct TokenResponse {
            access_token: String,
        }

        let token_resp: TokenResponse = resp.json()?;
        if token_resp.access_token.is_empty() {
            return Err(ApiError::OAuthError(
                "token response missing access_token".to_string(),
            ));
        }

        self.access_token = Some(token_resp.access_token);
        Ok(())
    }

    /// List all subjects (contacts) from the Fakturoid account.
    pub fn list_subjects(&self) -> Result<Vec<Subject>> {
        let raw = self.list_paginated("subjects.json")?;
        raw.into_iter()
            .map(|r| {
                serde_json::from_value(r)
                    .map_err(|e| ApiError::ParseError(format!("parsing subject: {}", e)))
            })
            .collect()
    }

    /// List all invoices from the Fakturoid account.
    pub fn list_invoices(&self) -> Result<Vec<Invoice>> {
        let raw = self.list_paginated("invoices.json")?;
        raw.into_iter()
            .map(|r| {
                serde_json::from_value(r)
                    .map_err(|e| ApiError::ParseError(format!("parsing invoice: {}", e)))
            })
            .collect()
    }

    /// List all expenses from the Fakturoid account.
    pub fn list_expenses(&self) -> Result<Vec<Expense>> {
        let raw = self.list_paginated("expenses.json")?;
        raw.into_iter()
            .map(|r| {
                serde_json::from_value(r)
                    .map_err(|e| ApiError::ParseError(format!("parsing expense: {}", e)))
            })
            .collect()
    }

    /// Download a file attachment from the given absolute URL.
    /// The URL must belong to app.fakturoid.cz to prevent SSRF attacks.
    pub fn download_attachment(&self, download_url: &str) -> Result<(Vec<u8>, String)> {
        let parsed = url::Url::parse(download_url)
            .map_err(|_| ApiError::InvalidInput("invalid attachment URL".to_string()))?;
        let url_host = {
            let host = parsed.host_str().unwrap_or("");
            match parsed.port() {
                Some(p) => format!("{}:{}", host, p),
                None => host.to_string(),
            }
        };
        if url_host != self.allowed_download_host {
            return Err(ApiError::InvalidInput(
                "attachment URL must be on app.fakturoid.cz".to_string(),
            ));
        }

        let token = self.access_token.as_deref().ok_or_else(|| {
            ApiError::OAuthError("not authenticated, call authenticate() first".to_string())
        })?;

        let resp = self
            .client
            .get(download_url)
            .header("Authorization", format!("Bearer {}", token))
            .header("User-Agent", format!("ZFaktury ({})", self.email))
            .send()?;

        match resp.status().as_u16() {
            200 => {}
            204 => {
                return Err(ApiError::HttpError {
                    status: 204,
                    body: "attachment has no content".to_string(),
                });
            }
            status => {
                return Err(ApiError::HttpError {
                    status,
                    body: format!("attachment download error: HTTP {}", status),
                });
            }
        }

        let content_type = resp
            .headers()
            .get("Content-Type")
            .and_then(|v| v.to_str().ok())
            .unwrap_or("application/octet-stream")
            .to_string();

        let bytes = resp.bytes()?.to_vec();
        Ok((bytes, content_type))
    }

    /// Fetch all pages of a paginated Fakturoid API endpoint.
    fn list_paginated(&self, path: &str) -> Result<Vec<serde_json::Value>> {
        let token = self.access_token.as_deref().ok_or_else(|| {
            ApiError::OAuthError("not authenticated, call authenticate() first".to_string())
        })?;

        let mut all = Vec::new();
        let mut page = 1;

        loop {
            let url = format!("{}/{}?page={}", self.base_url, path, page);

            let resp = self
                .client
                .get(&url)
                .header("Authorization", format!("Bearer {}", token))
                .header("User-Agent", format!("ZFaktury ({})", self.email))
                .header("Accept", "application/json")
                .send()?;

            if resp.status().as_u16() != 200 {
                return Err(ApiError::HttpError {
                    status: resp.status().as_u16(),
                    body: format!("fakturoid API error: HTTP {} for {}", resp.status(), path),
                });
            }

            let items: Vec<serde_json::Value> = resp.json()?;

            if items.is_empty() {
                break;
            }

            all.extend(items);
            page += 1;

            // Rate limiting delay between pages.
            thread::sleep(PAGE_DELAY);
        }

        Ok(all)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use wiremock::matchers::{header, method, path, query_param};
    use wiremock::{Mock, MockServer, ResponseTemplate};

    async fn run_blocking<F, R>(f: F) -> R
    where
        F: FnOnce() -> R + Send + 'static,
        R: Send + 'static,
    {
        tokio::task::spawn_blocking(f).await.unwrap()
    }

    #[tokio::test]
    async fn test_authenticate_success() {
        let mock_server = MockServer::start().await;

        Mock::given(method("POST"))
            .and(path("/oauth/token"))
            .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!({
                "access_token": "test-token-123",
                "token_type": "Bearer",
                "expires_in": 7200
            })))
            .mount(&mock_server)
            .await;

        let uri = mock_server.uri();
        let token = run_blocking(move || {
            let token_url = format!("{}/oauth/token", uri);
            let mut client = FakturoidClient::new(
                "test-slug",
                "test@example.com",
                "client-id",
                "client-secret",
            )
            .with_urls(&format!("{}/api/v3/accounts/test-slug", uri), &token_url);
            client.authenticate().unwrap();
            client.access_token.clone()
        })
        .await;

        assert_eq!(token.as_deref(), Some("test-token-123"));
    }

    #[tokio::test]
    async fn test_authenticate_failure() {
        let mock_server = MockServer::start().await;

        Mock::given(method("POST"))
            .and(path("/oauth/token"))
            .respond_with(ResponseTemplate::new(401).set_body_string("Unauthorized"))
            .mount(&mock_server)
            .await;

        let uri = mock_server.uri();
        let result = run_blocking(move || {
            let token_url = format!("{}/oauth/token", uri);
            let mut client =
                FakturoidClient::new("test-slug", "test@example.com", "client-id", "wrong-secret")
                    .with_urls(&format!("{}/api/v3/accounts/test-slug", uri), &token_url);
            client.authenticate()
        })
        .await;

        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), ApiError::OAuthError(_)));
    }

    #[tokio::test]
    async fn test_list_subjects_paginated() {
        let mock_server = MockServer::start().await;

        Mock::given(method("GET"))
            .and(path("/api/v3/accounts/test-slug/subjects.json"))
            .and(query_param("page", "1"))
            .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!([
                {"id": 1, "name": "Subject 1", "registration_no": "12345678"},
                {"id": 2, "name": "Subject 2", "registration_no": "87654321"}
            ])))
            .mount(&mock_server)
            .await;

        Mock::given(method("GET"))
            .and(path("/api/v3/accounts/test-slug/subjects.json"))
            .and(query_param("page", "2"))
            .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!([])))
            .mount(&mock_server)
            .await;

        let uri = mock_server.uri();
        let subjects = run_blocking(move || {
            let base_url = format!("{}/api/v3/accounts/test-slug", uri);
            let mut client =
                FakturoidClient::new("test-slug", "test@example.com", "cid", "csecret")
                    .with_urls(&base_url, "unused");
            client.access_token = Some("test-token".to_string());
            client.list_subjects().unwrap()
        })
        .await;

        assert_eq!(subjects.len(), 2);
        assert_eq!(subjects[0].name, "Subject 1");
        assert_eq!(subjects[1].registration_no, "87654321");
    }

    #[tokio::test]
    async fn test_list_invoices() {
        let mock_server = MockServer::start().await;

        Mock::given(method("GET"))
            .and(path("/api/v3/accounts/test-slug/invoices.json"))
            .and(query_param("page", "1"))
            .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!([
                {
                    "id": 100,
                    "number": "2026-001",
                    "status": "paid",
                    "total": 12100.0,
                    "exchange_rate": "1.0",
                    "lines": [{"name": "Service", "quantity": 1, "unit_price": "10000.0", "vat_rate": 21}]
                }
            ])))
            .mount(&mock_server)
            .await;

        Mock::given(method("GET"))
            .and(path("/api/v3/accounts/test-slug/invoices.json"))
            .and(query_param("page", "2"))
            .respond_with(ResponseTemplate::new(200).set_body_json(serde_json::json!([])))
            .mount(&mock_server)
            .await;

        let uri = mock_server.uri();
        let invoices = run_blocking(move || {
            let base_url = format!("{}/api/v3/accounts/test-slug", uri);
            let mut client =
                FakturoidClient::new("test-slug", "test@example.com", "cid", "csecret")
                    .with_urls(&base_url, "unused");
            client.access_token = Some("test-token".to_string());
            client.list_invoices().unwrap()
        })
        .await;

        assert_eq!(invoices.len(), 1);
        assert_eq!(invoices[0].number, "2026-001");
        assert_eq!(invoices[0].status, "paid");
        assert!((invoices[0].total.0 - 12100.0).abs() < 0.01);
        assert!((invoices[0].exchange_rate.0 - 1.0).abs() < 0.01);
        assert_eq!(invoices[0].lines.len(), 1);
        assert!((invoices[0].lines[0].unit_price.0 - 10000.0).abs() < 0.01);
    }

    #[tokio::test]
    async fn test_download_attachment() {
        let mock_server = MockServer::start().await;

        Mock::given(method("GET"))
            .and(path("/attachment/123"))
            .and(header("Authorization", "Bearer test-token"))
            .respond_with(
                ResponseTemplate::new(200)
                    .set_body_bytes(vec![0x25, 0x50, 0x44, 0x46])
                    .insert_header("Content-Type", "application/pdf"),
            )
            .mount(&mock_server)
            .await;

        let uri = mock_server.uri();
        let (data, content_type) = run_blocking(move || {
            let base_url = format!("{}/api/v3/accounts/test-slug", uri);
            let mut client =
                FakturoidClient::new("test-slug", "test@example.com", "cid", "csecret")
                    .with_urls(&base_url, "unused");
            client.access_token = Some("test-token".to_string());
            let url = format!(
                "{}/attachment/123",
                base_url.replace("/api/v3/accounts/test-slug", "")
            );
            client.download_attachment(&url).unwrap()
        })
        .await;

        assert_eq!(data, vec![0x25, 0x50, 0x44, 0x46]);
        assert_eq!(content_type, "application/pdf");
    }

    #[test]
    fn test_download_attachment_ssrf_rejected() {
        let mut client = FakturoidClient::new("slug", "test@example.com", "cid", "csecret");
        client.access_token = Some("test-token".to_string());
        // Attempt to download from a non-Fakturoid host.
        let result = client.download_attachment("https://evil.example.com/steal-data");
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), ApiError::InvalidInput(_)));
    }

    #[test]
    fn test_download_attachment_invalid_url() {
        let mut client = FakturoidClient::new("slug", "test@example.com", "cid", "csecret");
        client.access_token = Some("test-token".to_string());
        let result = client.download_attachment("not-a-valid-url");
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), ApiError::InvalidInput(_)));
    }

    #[test]
    fn test_not_authenticated() {
        let client = FakturoidClient::new("slug", "email", "cid", "csecret");
        let result = client.list_subjects();
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), ApiError::OAuthError(_)));
    }

    #[test]
    fn test_flex_float64_from_number() {
        let v: FlexFloat64 = serde_json::from_str("42.5").unwrap();
        assert!((v.0 - 42.5).abs() < 0.001);
    }

    #[test]
    fn test_flex_float64_from_integer() {
        let v: FlexFloat64 = serde_json::from_str("42").unwrap();
        assert!((v.0 - 42.0).abs() < 0.001);
    }

    #[test]
    fn test_flex_float64_from_string() {
        let v: FlexFloat64 = serde_json::from_str(r#""25.340""#).unwrap();
        assert!((v.0 - 25.340).abs() < 0.001);
    }

    #[test]
    fn test_flex_float64_from_null() {
        let v: FlexFloat64 = serde_json::from_str("null").unwrap();
        assert!((v.0).abs() < 0.001);
    }

    #[test]
    fn test_flex_float64_invalid_string() {
        let result: std::result::Result<FlexFloat64, _> = serde_json::from_str(r#""not-a-number""#);
        assert!(result.is_err());
    }
}
