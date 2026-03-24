use std::time::Duration;

use base64::Engine;
use base64::engine::general_purpose::STANDARD as BASE64;
use serde::{Deserialize, Serialize};
use zfaktury_domain::{Amount, OCRItem, OCRResult};

use crate::error::{ApiError, Result};

/// System prompt for OCR invoice extraction (Czech).
const SYSTEM_PROMPT: &str = r#"Jsi OCR asistent pro zpracování faktur a účtenek. Analyzuj obrázek a extrahuj strukturovaná data.

Vrať POUZE platný JSON objekt (bez markdown, bez komentářů) s následující strukturou:
{
  "vendor_name": "název dodavatele",
  "vendor_ico": "IČO dodavatele",
  "vendor_dic": "DIČ dodavatele",
  "invoice_number": "číslo faktury/dokladu",
  "issue_date": "datum vystavení ve formátu YYYY-MM-DD",
  "due_date": "datum splatnosti ve formátu YYYY-MM-DD",
  "total_amount": celková částka v CZK (číslo, např. 1234.56),
  "vat_amount": částka DPH v CZK (číslo),
  "vat_rate_percent": sazba DPH v procentech (celé číslo, např. 21),
  "currency_code": "kód měny (CZK, EUR, USD)",
  "description": "krátký popis dokladu",
  "items": [
    {
      "description": "popis položky",
      "quantity": množství (číslo, např. 1.5),
      "unit_price": jednotková cena v CZK (číslo),
      "vat_rate_percent": sazba DPH položky (celé číslo),
      "total_amount": celková cena položky v CZK (číslo)
    }
  ],
  "raw_text": "neupravený text z dokladu",
  "confidence": míra jistoty 0.0-1.0
}

Důležité:
- Částky jsou v korunách (CZK) jako desetinná čísla (např. 1234.56 = 1234 Kč a 56 haléřů)
- Pokud údaj není na dokladu, použij prázdný řetězec pro textová pole, 0 pro čísla
- Datum vždy ve formátu YYYY-MM-DD
- Pro confidence použij hodnotu podle toho, jak jsi si jistý správností extrakce"#;

/// User prompt for OCR invoice extraction.
const USER_PROMPT: &str = "Analyzuj tento doklad (faktura/účtenka) a extrahuj všechna dostupná data do JSON formátu podle zadané struktury.";

const SUPPORTED_CONTENT_TYPES: &[&str] = &["image/jpeg", "image/png", "application/pdf"];

/// Trait for OCR document processing providers.
pub trait OcrProvider: Send + Sync {
    /// Process an image and extract structured invoice data.
    fn process_image(&self, image_data: &[u8], content_type: &str) -> Result<OCRResult>;
    /// Return the provider name.
    fn name(&self) -> &str;
}

/// OCR provider using Anthropic's Claude Messages API.
pub struct AnthropicProvider {
    api_key: String,
    base_url: String,
    model: String,
    client: reqwest::blocking::Client,
}

impl AnthropicProvider {
    /// Create a new Anthropic OCR provider.
    pub fn new(api_key: &str, model: &str) -> Self {
        let model = if model.is_empty() {
            "claude-sonnet-4-20250514".to_string()
        } else {
            model.to_string()
        };
        Self {
            api_key: api_key.to_string(),
            base_url: "https://api.anthropic.com/v1/messages".to_string(),
            model,
            client: reqwest::blocking::Client::builder()
                .timeout(Duration::from_secs(30))
                .build()
                .expect("failed to build HTTP client"),
        }
    }

    /// Override the API endpoint URL (for testing).
    pub fn with_base_url(mut self, url: &str) -> Self {
        self.base_url = url.to_string();
        self
    }
}

impl OcrProvider for AnthropicProvider {
    fn process_image(&self, image_data: &[u8], content_type: &str) -> Result<OCRResult> {
        validate_content_type(content_type)?;

        let b64_data = BASE64.encode(image_data);

        let req_body = AnthropicRequest {
            model: &self.model,
            max_tokens: 4096,
            system: SYSTEM_PROMPT,
            messages: vec![AnthropicMessage {
                role: "user",
                content: vec![
                    AnthropicContentPart::Image {
                        source: AnthropicImageSource {
                            type_: "base64",
                            media_type: content_type,
                            data: &b64_data,
                        },
                    },
                    AnthropicContentPart::Text { text: USER_PROMPT },
                ],
            }],
        };

        let resp = self
            .client
            .post(&self.base_url)
            .header("Content-Type", "application/json")
            .header("x-api-key", &self.api_key)
            .header("anthropic-version", "2023-06-01")
            .json(&req_body)
            .send()?;

        if resp.status().as_u16() != 200 {
            let status = resp.status().as_u16();
            let body = resp.text().unwrap_or_default();
            return Err(ApiError::HttpError {
                status,
                body: truncate(&body, 500),
            });
        }

        let resp_body: AnthropicResponse = resp.json()?;

        if let Some(err) = resp_body.error {
            return Err(ApiError::HttpError {
                status: 400,
                body: err.message,
            });
        }

        let text = resp_body
            .content
            .iter()
            .find(|c| c.type_ == "text")
            .map(|c| c.text.as_str())
            .ok_or_else(|| {
                ApiError::ParseError("anthropic returned no text content".to_string())
            })?;

        parse_ocr_json(text)
    }

    fn name(&self) -> &str {
        "claude"
    }
}

/// OCR provider using OpenAI-compatible Chat Completions API.
/// Works with OpenAI, OpenRouter, Gemini, Mistral.
pub struct OpenAIProvider {
    provider_name: String,
    api_key: String,
    base_url: String,
    model: String,
    client: reqwest::blocking::Client,
}

impl OpenAIProvider {
    /// Create a new OpenAI provider.
    pub fn new_openai(api_key: &str, model: &str) -> Self {
        let model = if model.is_empty() {
            "gpt-4o".to_string()
        } else {
            model.to_string()
        };
        Self {
            provider_name: "openai".to_string(),
            api_key: api_key.to_string(),
            base_url: "https://api.openai.com/v1/chat/completions".to_string(),
            model,
            client: reqwest::blocking::Client::builder()
                .timeout(Duration::from_secs(25))
                .build()
                .expect("failed to build HTTP client"),
        }
    }

    /// Create a new OpenRouter provider.
    pub fn new_openrouter(api_key: &str, model: &str) -> Self {
        let model = if model.is_empty() {
            "google/gemini-2.0-flash-001".to_string()
        } else {
            model.to_string()
        };
        Self {
            provider_name: "openrouter".to_string(),
            api_key: api_key.to_string(),
            base_url: "https://openrouter.ai/api/v1/chat/completions".to_string(),
            model,
            client: reqwest::blocking::Client::builder()
                .timeout(Duration::from_secs(25))
                .build()
                .expect("failed to build HTTP client"),
        }
    }

    /// Override the API endpoint URL (for testing).
    pub fn with_base_url(mut self, url: &str) -> Self {
        self.base_url = url.to_string();
        self
    }
}

impl OcrProvider for OpenAIProvider {
    fn process_image(&self, image_data: &[u8], content_type: &str) -> Result<OCRResult> {
        validate_content_type(content_type)?;

        let b64_data = BASE64.encode(image_data);
        let data_url = format!("data:{};base64,{}", content_type, b64_data);

        let req_body = ChatRequest {
            model: &self.model,
            messages: vec![
                ChatMessage {
                    role: "system",
                    content: vec![ChatContentPart::Text {
                        text: SYSTEM_PROMPT,
                    }],
                },
                ChatMessage {
                    role: "user",
                    content: vec![
                        ChatContentPart::Text { text: USER_PROMPT },
                        ChatContentPart::ImageUrl {
                            image_url: ImageUrl { url: &data_url },
                        },
                    ],
                },
            ],
            max_tokens: 4096,
            temperature: 0.1,
        };

        let resp = self
            .client
            .post(&self.base_url)
            .header("Content-Type", "application/json")
            .header("Authorization", format!("Bearer {}", self.api_key))
            .json(&req_body)
            .send()?;

        if resp.status().as_u16() != 200 {
            let status = resp.status().as_u16();
            let body = resp.text().unwrap_or_default();
            return Err(ApiError::HttpError {
                status,
                body: truncate(&body, 500),
            });
        }

        let resp_body: ChatResponse = resp.json()?;

        if let Some(err) = resp_body.error {
            return Err(ApiError::HttpError {
                status: 400,
                body: err.message,
            });
        }

        let content = resp_body
            .choices
            .first()
            .map(|c| c.message.content.as_str())
            .ok_or_else(|| {
                ApiError::ParseError(format!("{} returned no choices", self.provider_name))
            })?;

        parse_ocr_json(content)
    }

    fn name(&self) -> &str {
        &self.provider_name
    }
}

// -- Anthropic API types --

#[derive(Serialize)]
struct AnthropicRequest<'a> {
    model: &'a str,
    max_tokens: i32,
    system: &'a str,
    messages: Vec<AnthropicMessage<'a>>,
}

#[derive(Serialize)]
struct AnthropicMessage<'a> {
    role: &'a str,
    content: Vec<AnthropicContentPart<'a>>,
}

#[derive(Serialize)]
#[serde(tag = "type")]
enum AnthropicContentPart<'a> {
    #[serde(rename = "image")]
    Image { source: AnthropicImageSource<'a> },
    #[serde(rename = "text")]
    Text { text: &'a str },
}

#[derive(Serialize)]
struct AnthropicImageSource<'a> {
    #[serde(rename = "type")]
    type_: &'a str,
    media_type: &'a str,
    data: &'a str,
}

#[derive(Deserialize)]
struct AnthropicResponse {
    #[serde(default)]
    content: Vec<AnthropicResponseContent>,
    error: Option<AnthropicError>,
}

#[derive(Deserialize)]
struct AnthropicResponseContent {
    #[serde(rename = "type")]
    type_: String,
    #[serde(default)]
    text: String,
}

#[derive(Deserialize)]
struct AnthropicError {
    message: String,
}

// -- OpenAI-compatible API types --

#[derive(Serialize)]
struct ChatRequest<'a> {
    model: &'a str,
    messages: Vec<ChatMessage<'a>>,
    max_tokens: i32,
    temperature: f64,
}

#[derive(Serialize)]
struct ChatMessage<'a> {
    role: &'a str,
    content: Vec<ChatContentPart<'a>>,
}

#[derive(Serialize)]
#[serde(tag = "type")]
enum ChatContentPart<'a> {
    #[serde(rename = "text")]
    Text { text: &'a str },
    #[serde(rename = "image_url")]
    ImageUrl { image_url: ImageUrl<'a> },
}

#[derive(Serialize)]
struct ImageUrl<'a> {
    url: &'a str,
}

#[derive(Deserialize)]
struct ChatResponse {
    #[serde(default)]
    choices: Vec<ChatChoice>,
    error: Option<ChatApiError>,
}

#[derive(Deserialize)]
struct ChatChoice {
    message: ChatResponseMessage,
}

#[derive(Deserialize)]
struct ChatResponseMessage {
    content: String,
}

#[derive(Deserialize)]
struct ChatApiError {
    message: String,
}

// -- OCR JSON parsing --

#[derive(Deserialize)]
struct OcrJsonResponse {
    #[serde(default)]
    vendor_name: String,
    #[serde(default)]
    vendor_ico: String,
    #[serde(default)]
    vendor_dic: String,
    #[serde(default)]
    invoice_number: String,
    #[serde(default)]
    issue_date: String,
    #[serde(default)]
    due_date: String,
    #[serde(default)]
    total_amount: f64,
    #[serde(default)]
    vat_amount: f64,
    #[serde(default)]
    vat_rate_percent: i32,
    #[serde(default)]
    currency_code: String,
    #[serde(default)]
    description: String,
    #[serde(default)]
    items: Vec<OcrItemResponse>,
    #[serde(default)]
    raw_text: String,
    #[serde(default)]
    confidence: f64,
}

#[derive(Deserialize)]
struct OcrItemResponse {
    #[serde(default)]
    description: String,
    #[serde(default)]
    quantity: f64,
    #[serde(default)]
    unit_price: f64,
    #[serde(default)]
    vat_rate_percent: i32,
    #[serde(default)]
    total_amount: f64,
}

/// Parse OCR JSON output (possibly wrapped in markdown code fences) into an OCRResult.
fn parse_ocr_json(content: &str) -> Result<OCRResult> {
    let content = strip_code_fences(content);

    let resp: OcrJsonResponse = serde_json::from_str(content)
        .map_err(|e| ApiError::ParseError(format!("parsing OCR JSON: {}", e)))?;

    let items = resp
        .items
        .iter()
        .map(|item| OCRItem {
            description: item.description.clone(),
            quantity: Amount::from_float(item.quantity),
            unit_price: Amount::from_float(item.unit_price),
            vat_rate_percent: item.vat_rate_percent,
            total_amount: Amount::from_float(item.total_amount),
        })
        .collect();

    Ok(OCRResult {
        vendor_name: resp.vendor_name,
        vendor_ico: resp.vendor_ico,
        vendor_dic: resp.vendor_dic,
        invoice_number: resp.invoice_number,
        issue_date: resp.issue_date,
        due_date: resp.due_date,
        total_amount: Amount::from_float(resp.total_amount),
        vat_amount: Amount::from_float(resp.vat_amount),
        vat_rate_percent: resp.vat_rate_percent,
        currency_code: resp.currency_code,
        description: resp.description,
        items,
        raw_text: resp.raw_text,
        confidence: resp.confidence,
    })
}

/// Validate that the content type is supported for OCR processing.
fn validate_content_type(content_type: &str) -> Result<()> {
    if SUPPORTED_CONTENT_TYPES.contains(&content_type) {
        Ok(())
    } else {
        Err(ApiError::UnsupportedContentType(format!(
            "{}; supported: {}",
            content_type,
            SUPPORTED_CONTENT_TYPES.join(", ")
        )))
    }
}

/// Strip markdown code fences from content if present.
fn strip_code_fences(s: &str) -> &str {
    let s = s.trim();
    let s = if let Some(rest) = s.strip_prefix("```json") {
        rest
    } else if let Some(rest) = s.strip_prefix("```") {
        rest
    } else {
        s
    };
    let s = if let Some(rest) = s.strip_suffix("```") {
        rest
    } else {
        s
    };
    s.trim()
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
    use wiremock::matchers::{header, method};
    use wiremock::{Mock, MockServer, ResponseTemplate};

    async fn run_blocking<F, R>(f: F) -> R
    where
        F: FnOnce() -> R + Send + 'static,
        R: Send + 'static,
    {
        tokio::task::spawn_blocking(f).await.unwrap()
    }

    #[test]
    fn test_validate_content_type_supported() {
        assert!(validate_content_type("image/jpeg").is_ok());
        assert!(validate_content_type("image/png").is_ok());
        assert!(validate_content_type("application/pdf").is_ok());
    }

    #[test]
    fn test_validate_content_type_unsupported() {
        let result = validate_content_type("image/gif");
        assert!(result.is_err());
        assert!(matches!(
            result.unwrap_err(),
            ApiError::UnsupportedContentType(_)
        ));
    }

    #[test]
    fn test_strip_code_fences_json() {
        let input = "```json\n{\"key\": \"value\"}\n```";
        assert_eq!(strip_code_fences(input), r#"{"key": "value"}"#);
    }

    #[test]
    fn test_strip_code_fences_plain() {
        let input = "```\n{\"key\": \"value\"}\n```";
        assert_eq!(strip_code_fences(input), r#"{"key": "value"}"#);
    }

    #[test]
    fn test_strip_code_fences_none() {
        let input = r#"{"key": "value"}"#;
        assert_eq!(strip_code_fences(input), r#"{"key": "value"}"#);
    }

    #[test]
    fn test_parse_ocr_json_success() {
        let json = r#"{
            "vendor_name": "Test s.r.o.",
            "vendor_ico": "12345678",
            "vendor_dic": "CZ12345678",
            "invoice_number": "2026-001",
            "issue_date": "2026-03-15",
            "due_date": "2026-04-15",
            "total_amount": 1210.00,
            "vat_amount": 210.00,
            "vat_rate_percent": 21,
            "currency_code": "CZK",
            "description": "Consulting services",
            "items": [
                {
                    "description": "Consulting",
                    "quantity": 10.0,
                    "unit_price": 100.00,
                    "vat_rate_percent": 21,
                    "total_amount": 1210.00
                }
            ],
            "raw_text": "Faktura 2026-001",
            "confidence": 0.95
        }"#;

        let result = parse_ocr_json(json).unwrap();
        assert_eq!(result.vendor_name, "Test s.r.o.");
        assert_eq!(result.vendor_ico, "12345678");
        assert_eq!(result.total_amount, Amount::from_float(1210.00));
        assert_eq!(result.vat_amount, Amount::from_float(210.00));
        assert_eq!(result.vat_rate_percent, 21);
        assert_eq!(result.items.len(), 1);
        assert_eq!(result.items[0].description, "Consulting");
        assert_eq!(result.items[0].quantity, Amount::from_float(10.0));
        assert!((result.confidence - 0.95).abs() < 0.001);
    }

    #[test]
    fn test_parse_ocr_json_with_code_fences() {
        let json = r#"```json
        {"vendor_name": "Test", "total_amount": 100.0, "confidence": 0.8}
        ```"#;

        let result = parse_ocr_json(json).unwrap();
        assert_eq!(result.vendor_name, "Test");
    }

    #[test]
    fn test_parse_ocr_json_invalid() {
        let result = parse_ocr_json("not json at all");
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), ApiError::ParseError(_)));
    }

    #[tokio::test]
    async fn test_anthropic_provider_success() {
        let mock_server = MockServer::start().await;

        let response_body = serde_json::json!({
            "content": [{
                "type": "text",
                "text": r#"{"vendor_name": "Mock Vendor", "total_amount": 500.0, "confidence": 0.9}"#
            }]
        });

        Mock::given(method("POST"))
            .and(header("x-api-key", "test-key"))
            .and(header("anthropic-version", "2023-06-01"))
            .respond_with(ResponseTemplate::new(200).set_body_json(&response_body))
            .mount(&mock_server)
            .await;

        let uri = mock_server.uri();
        let result = run_blocking(move || {
            let provider =
                AnthropicProvider::new("test-key", "claude-sonnet-4-20250514").with_base_url(&uri);
            provider
                .process_image(b"fake-image-data", "image/jpeg")
                .unwrap()
        })
        .await;

        assert_eq!(result.vendor_name, "Mock Vendor");
        assert_eq!(result.total_amount, Amount::from_float(500.0));
    }

    #[tokio::test]
    async fn test_anthropic_provider_api_error() {
        let mock_server = MockServer::start().await;

        Mock::given(method("POST"))
            .respond_with(ResponseTemplate::new(429).set_body_string("rate limited"))
            .mount(&mock_server)
            .await;

        let uri = mock_server.uri();
        let result = run_blocking(move || {
            let provider = AnthropicProvider::new("test-key", "").with_base_url(&uri);
            provider.process_image(b"fake-image-data", "image/jpeg")
        })
        .await;

        assert!(result.is_err());
        assert!(matches!(
            result.unwrap_err(),
            ApiError::HttpError { status: 429, .. }
        ));
    }

    #[tokio::test]
    async fn test_openai_provider_success() {
        let mock_server = MockServer::start().await;

        let response_body = serde_json::json!({
            "choices": [{
                "message": {
                    "content": r#"{"vendor_name": "OpenAI Vendor", "total_amount": 750.50, "confidence": 0.85}"#
                }
            }]
        });

        Mock::given(method("POST"))
            .and(header("Authorization", "Bearer test-key"))
            .respond_with(ResponseTemplate::new(200).set_body_json(&response_body))
            .mount(&mock_server)
            .await;

        let uri = mock_server.uri();
        let result = run_blocking(move || {
            let provider = OpenAIProvider::new_openai("test-key", "gpt-4o").with_base_url(&uri);
            provider
                .process_image(b"fake-image-data", "image/png")
                .unwrap()
        })
        .await;

        assert_eq!(result.vendor_name, "OpenAI Vendor");
        assert_eq!(result.total_amount, Amount::from_float(750.50));
    }

    #[test]
    fn test_system_prompt_contains_czech() {
        assert!(SYSTEM_PROMPT.contains("Jsi OCR asistent"));
        assert!(SYSTEM_PROMPT.contains("faktur"));
        assert!(SYSTEM_PROMPT.contains("JSON"));
        assert!(SYSTEM_PROMPT.contains("YYYY-MM-DD"));
    }

    #[test]
    fn test_anthropic_provider_name() {
        let provider = AnthropicProvider::new("key", "");
        assert_eq!(provider.name(), "claude");
    }

    #[test]
    fn test_openai_provider_name() {
        let provider = OpenAIProvider::new_openai("key", "");
        assert_eq!(provider.name(), "openai");
    }

    #[test]
    fn test_openrouter_provider_name() {
        let provider = OpenAIProvider::new_openrouter("key", "");
        assert_eq!(provider.name(), "openrouter");
    }

    #[test]
    fn test_unsupported_content_type() {
        let provider = AnthropicProvider::new("key", "");
        let result = provider.process_image(b"data", "image/gif");
        assert!(result.is_err());
        assert!(matches!(
            result.unwrap_err(),
            ApiError::UnsupportedContentType(_)
        ));
    }
}
