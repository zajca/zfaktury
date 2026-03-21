use std::time::Duration;

use chrono::NaiveDateTime;
use serde::Deserialize;
use zfaktury_domain::{Contact, ContactType};

use crate::error::{ApiError, Result};

/// Default timestamp for ARES-returned contacts (epoch).
/// Actual timestamps are set when the contact is persisted.
fn default_timestamp() -> NaiveDateTime {
    NaiveDateTime::default()
}

const DEFAULT_BASE_URL: &str = "https://ares.gov.cz/ekonomicke-subjekty-v-be/rest";
const DEFAULT_TIMEOUT: Duration = Duration::from_secs(10);

/// Client for the Czech ARES (Administrative Register of Economic Subjects) API.
pub struct AresClient {
    base_url: String,
    client: reqwest::blocking::Client,
}

impl Default for AresClient {
    fn default() -> Self {
        Self::new()
    }
}

impl AresClient {
    /// Create a new ARES client with default settings.
    pub fn new() -> Self {
        Self {
            base_url: DEFAULT_BASE_URL.to_string(),
            client: reqwest::blocking::Client::builder()
                .timeout(DEFAULT_TIMEOUT)
                .build()
                .expect("failed to build HTTP client"),
        }
    }

    /// Create a new ARES client with a custom base URL and timeout.
    pub fn with_config(base_url: &str, timeout: Duration) -> Self {
        Self {
            base_url: base_url.to_string(),
            client: reqwest::blocking::Client::builder()
                .timeout(timeout)
                .build()
                .expect("failed to build HTTP client"),
        }
    }

    /// Look up a company by its ICO (8-digit identification number) in the ARES registry.
    pub fn lookup_by_ico(&self, ico: &str) -> Result<Contact> {
        // Validate: ICO must be exactly 8 digits.
        if ico.len() != 8 || !ico.chars().all(|c| c.is_ascii_digit()) {
            return Err(ApiError::InvalidInput(
                "ICO must be exactly 8 digits".to_string(),
            ));
        }

        let url = format!("{}/ekonomicke-subjekty/{}", self.base_url, ico);

        let resp = self
            .client
            .get(&url)
            .header("Accept", "application/json")
            .send()?;

        match resp.status().as_u16() {
            200 => {}
            404 => return Err(ApiError::NotFound),
            429 => return Err(ApiError::RateLimited),
            status => {
                let body = resp.text().unwrap_or_default();
                return Err(ApiError::HttpError {
                    status,
                    body: truncate(&body, 500),
                });
            }
        }

        let ares_resp: AresResponse = resp.json()?;
        Ok(ares_resp.to_contact())
    }
}

/// ARES API response for /ekonomicke-subjekty/{ICO}.
#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
struct AresResponse {
    ico: String,
    obchodni_jmeno: String,
    #[serde(default)]
    dic: String,
    sidlo: AresSidlo,
}

/// Registered address from the ARES response.
#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
struct AresSidlo {
    #[serde(default)]
    textova_adresa: String,
    #[serde(default)]
    nazev_obce: String,
    #[serde(default)]
    psc: i32,
    #[serde(default)]
    nazev_ulice: String,
    #[serde(default)]
    cislo_domovni: i32,
    #[serde(default)]
    cislo_orientacni: i32,
}

impl AresResponse {
    fn to_contact(&self) -> Contact {
        let street = self.build_street();
        let zip = if self.sidlo.psc == 0 {
            String::new()
        } else {
            format!("{:05}", self.sidlo.psc)
        };

        Contact {
            id: 0,
            contact_type: ContactType::Company,
            name: self.obchodni_jmeno.clone(),
            ico: self.ico.clone(),
            dic: self.dic.clone(),
            street,
            city: self.sidlo.nazev_obce.clone(),
            zip,
            country: "CZ".to_string(),
            email: String::new(),
            phone: String::new(),
            web: String::new(),
            bank_account: String::new(),
            bank_code: String::new(),
            iban: String::new(),
            swift: String::new(),
            payment_terms_days: 0,
            tags: String::new(),
            notes: String::new(),
            is_favorite: false,
            vat_unreliable_at: None,
            created_at: default_timestamp(),
            updated_at: default_timestamp(),
            deleted_at: None,
        }
    }

    fn build_street(&self) -> String {
        if self.sidlo.nazev_ulice.is_empty() && self.sidlo.cislo_domovni == 0 {
            return self.sidlo.textova_adresa.clone();
        }

        let number = if self.sidlo.cislo_domovni > 0 {
            if self.sidlo.cislo_orientacni > 0 {
                format!(
                    "{}/{}",
                    self.sidlo.cislo_domovni, self.sidlo.cislo_orientacni
                )
            } else {
                self.sidlo.cislo_domovni.to_string()
            }
        } else {
            String::new()
        };

        if !self.sidlo.nazev_ulice.is_empty() && !number.is_empty() {
            format!("{} {}", self.sidlo.nazev_ulice, number)
        } else if !self.sidlo.nazev_ulice.is_empty() {
            self.sidlo.nazev_ulice.clone()
        } else {
            number
        }
    }
}

fn truncate(s: &str, max_len: usize) -> String {
    if s.len() <= max_len {
        s.to_string()
    } else {
        format!("{}...", &s[..max_len])
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use wiremock::matchers::{method, path};
    use wiremock::{Mock, MockServer, ResponseTemplate};

    /// Run a blocking closure on a separate thread so reqwest's internal
    /// runtime does not conflict with tokio's test runtime.
    async fn run_blocking<F, R>(f: F) -> R
    where
        F: FnOnce() -> R + Send + 'static,
        R: Send + 'static,
    {
        tokio::task::spawn_blocking(f).await.unwrap()
    }

    #[tokio::test]
    async fn test_lookup_by_ico_success() {
        let mock_server = MockServer::start().await;

        let body = serde_json::json!({
            "ico": "27074358",
            "obchodniJmeno": "Alza.cz a.s.",
            "dic": "CZ27074358",
            "sidlo": {
                "textovaAdresa": "Janovskeho 3769/2",
                "nazevObce": "Praha",
                "psc": 17000,
                "nazevUlice": "Janovskeho",
                "cisloDomovni": 3769,
                "cisloOrientacni": 2
            }
        });

        Mock::given(method("GET"))
            .and(path("/ekonomicke-subjekty/27074358"))
            .respond_with(ResponseTemplate::new(200).set_body_json(&body))
            .mount(&mock_server)
            .await;

        let uri = mock_server.uri();
        let contact = run_blocking(move || {
            let client = AresClient::with_config(&uri, Duration::from_secs(5));
            client.lookup_by_ico("27074358").unwrap()
        })
        .await;

        assert_eq!(contact.name, "Alza.cz a.s.");
        assert_eq!(contact.ico, "27074358");
        assert_eq!(contact.dic, "CZ27074358");
        assert_eq!(contact.street, "Janovskeho 3769/2");
        assert_eq!(contact.city, "Praha");
        assert_eq!(contact.zip, "17000");
        assert_eq!(contact.country, "CZ");
        assert_eq!(contact.contact_type, ContactType::Company);
    }

    #[tokio::test]
    async fn test_lookup_by_ico_not_found() {
        let mock_server = MockServer::start().await;

        Mock::given(method("GET"))
            .and(path("/ekonomicke-subjekty/99999999"))
            .respond_with(ResponseTemplate::new(404))
            .mount(&mock_server)
            .await;

        let uri = mock_server.uri();
        let result = run_blocking(move || {
            let client = AresClient::with_config(&uri, Duration::from_secs(5));
            client.lookup_by_ico("99999999")
        })
        .await;

        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), ApiError::NotFound));
    }

    #[test]
    fn test_lookup_by_ico_invalid_format_short() {
        let client = AresClient::new();
        let result = client.lookup_by_ico("1234567");
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), ApiError::InvalidInput(_)));
    }

    #[test]
    fn test_lookup_by_ico_invalid_format_letters() {
        let client = AresClient::new();
        let result = client.lookup_by_ico("1234567a");
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), ApiError::InvalidInput(_)));
    }

    #[test]
    fn test_lookup_by_ico_invalid_format_long() {
        let client = AresClient::new();
        let result = client.lookup_by_ico("123456789");
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), ApiError::InvalidInput(_)));
    }

    #[tokio::test]
    async fn test_lookup_by_ico_street_without_orientacni() {
        let mock_server = MockServer::start().await;

        let body = serde_json::json!({
            "ico": "12345678",
            "obchodniJmeno": "Test s.r.o.",
            "dic": "",
            "sidlo": {
                "textovaAdresa": "Hlavni 42",
                "nazevObce": "Brno",
                "psc": 60200,
                "nazevUlice": "Hlavni",
                "cisloDomovni": 42,
                "cisloOrientacni": 0
            }
        });

        Mock::given(method("GET"))
            .and(path("/ekonomicke-subjekty/12345678"))
            .respond_with(ResponseTemplate::new(200).set_body_json(&body))
            .mount(&mock_server)
            .await;

        let uri = mock_server.uri();
        let contact = run_blocking(move || {
            let client = AresClient::with_config(&uri, Duration::from_secs(5));
            client.lookup_by_ico("12345678").unwrap()
        })
        .await;

        assert_eq!(contact.street, "Hlavni 42");
        assert_eq!(contact.zip, "60200");
    }

    #[tokio::test]
    async fn test_lookup_by_ico_textova_adresa_fallback() {
        let mock_server = MockServer::start().await;

        let body = serde_json::json!({
            "ico": "12345678",
            "obchodniJmeno": "Test s.r.o.",
            "sidlo": {
                "textovaAdresa": "Namesti Miru 1, Praha 2",
                "nazevObce": "Praha",
                "psc": 12000,
                "nazevUlice": "",
                "cisloDomovni": 0,
                "cisloOrientacni": 0
            }
        });

        Mock::given(method("GET"))
            .and(path("/ekonomicke-subjekty/12345678"))
            .respond_with(ResponseTemplate::new(200).set_body_json(&body))
            .mount(&mock_server)
            .await;

        let uri = mock_server.uri();
        let contact = run_blocking(move || {
            let client = AresClient::with_config(&uri, Duration::from_secs(5));
            client.lookup_by_ico("12345678").unwrap()
        })
        .await;

        assert_eq!(contact.street, "Namesti Miru 1, Praha 2");
    }
}
