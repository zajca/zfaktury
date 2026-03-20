# Phase 4: External APIs

**Crate:** `zfaktury-api`
**Estimated effort:** 1 week
**Dependencies:** `zfaktury-domain` (Phase 2)

## Objectives

Port all external API clients from the Go codebase to Rust:

- ARES (Czech business registry lookup)
- CNB (Czech National Bank exchange rates)
- FIO Bank (bank transaction imports)
- SMTP (email sending with attachments)
- OCR (AI-based invoice data extraction)
- Fakturoid (invoicing service import)

## Crate Dependencies

```toml
[dependencies]
zfaktury-domain = { path = "../zfaktury-domain" }
reqwest = { version = "0.12", features = ["blocking", "json"] }
lettre = { version = "0.11", features = ["smtp-transport", "builder", "rustls-tls"] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"
chrono = { version = "0.4", features = ["serde"] }
base64 = "0.22"
thiserror = "2"
tracing = "0.1"

[dev-dependencies]
wiremock = "0.6"
tokio = { version = "1", features = ["rt", "macros"] }  # wiremock requires async runtime
```

> **Note on blocking vs async:** The application uses synchronous execution throughout.
> `reqwest::blocking` is used for all HTTP clients. `wiremock` requires a tokio runtime
> for tests only -- production code remains fully synchronous.

## Error Types

`src/error.rs` -- shared error enum for all API clients:

```rust
use thiserror::Error;

#[derive(Debug, Error)]
pub enum ApiError {
    #[error("HTTP request failed: {0}")]
    Http(#[from] reqwest::Error),

    #[error("not found: {0}")]
    NotFound(String),

    #[error("rate limited")]
    RateLimited,

    #[error("invalid input: {0}")]
    InvalidInput(String),

    #[error("authentication failed: {0}")]
    AuthFailed(String),

    #[error("response too large (limit: {limit} bytes)")]
    ResponseTooLarge { limit: u64 },

    #[error("parse error: {0}")]
    Parse(String),

    #[error("timeout")]
    Timeout,

    #[error("SMTP error: {0}")]
    Smtp(String),

    #[error("API error: {status} {body}")]
    ApiResponse { status: u16, body: String },

    #[error("{0}")]
    Other(String),
}
```

---

## ARES Client

**Go source:** `internal/ares/client.go`, `internal/ares/types.go`
**Purpose:** Czech business registry (ARES) lookup by ICO

### Module: `src/ares/`

**`src/ares/client.rs`**

```rust
pub struct AresClient {
    base_url: String,
    client: reqwest::blocking::Client,
}

impl AresClient {
    /// Create a new ARES client with default settings.
    pub fn new() -> Self;

    /// Create with a custom timeout.
    pub fn with_timeout(timeout: Duration) -> Self;

    /// Override the base URL (for testing with wiremock).
    pub fn with_base_url(base_url: &str) -> Self;

    /// Look up a company by its ICO (8-digit identification number).
    /// Returns a `Contact` with type_=Company, country="CZ".
    pub fn lookup_by_ico(&self, ico: &str) -> Result<Contact, ApiError>;
}
```

**Configuration:**

| Parameter | Value |
|-----------|-------|
| Base URL | `https://ares.gov.cz/ekonomicke-subjekty-v-be/rest` |
| Endpoint | `GET /ekonomicke-subjekty/{ICO}` |
| Timeout | 10 seconds |
| Response limit | 1 MB |
| Accept header | `application/json` |

**ICO validation:** Must match `^\d{8}$`. Reject with `ApiError::InvalidInput` otherwise.

**Error mapping:**

| HTTP Status | Error |
|-------------|-------|
| 200 | Parse response |
| 404 | `ApiError::NotFound("subject not found")` |
| 429 | `ApiError::RateLimited` |
| Other | `ApiError::ApiResponse { status, body }` |

**`src/ares/types.rs`** -- serde response types:

```rust
#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
struct AresResponse {
    ico: String,
    obchodni_jmeno: String,
    #[serde(default)]
    dic: Option<String>,
    sidlo: AresAddress,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
struct AresAddress {
    textova_adresa: Option<String>,
    nazev_obce: Option<String>,
    #[serde(default)]
    psc: Option<i32>,
    nazev_ulice: Option<String>,
    #[serde(default)]
    cislo_domovni: Option<i32>,
    #[serde(default)]
    cislo_orientacni: Option<i32>,
    nazev_casti_obce: Option<String>,
}
```

**Mapping to `Contact`:**

- `type_` = `ContactType::Company`
- `name` = `obchodni_jmeno`
- `ico` = `ico`
- `dic` = `dic` (if present)
- `street` = built from address components (same logic as Go `buildStreet`)
- `city` = `nazev_obce`
- `zip` = PSC formatted as 5-digit string with leading zeros (`format!("{:05}", psc)`)
- `country` = `"CZ"`

**Street building logic:**

1. If `nazev_ulice` is empty AND `cislo_domovni` is 0/None -> use `textova_adresa`
2. Otherwise: street = `nazev_ulice`
3. If `cislo_domovni` > 0: append house number
4. If `cislo_orientacni` > 0: format as `{domovni}/{orientacni}`
5. Join street name and number with space

---

## CNB Exchange Rate Client

**Go source:** `internal/service/cnb/client.go`, `internal/service/cnb/types.go`
**Purpose:** Czech National Bank daily exchange rates

### Module: `src/cnb/`

**`src/cnb/client.rs`**

```rust
pub struct CnbClient {
    base_url: String,
    client: reqwest::blocking::Client,
    cache: Mutex<HashMap<String, CacheEntry>>,  // keyed by "DD.MM.YYYY"
}

struct CacheEntry {
    rates: HashMap<String, ExchangeRate>,  // keyed by currency code
    fetched_at: Instant,
}

impl CnbClient {
    pub fn new() -> Self;
    pub fn with_base_url(base_url: &str) -> Self;

    /// Get CZK exchange rate per 1 unit of the given foreign currency.
    /// Tries the exact date first, then falls back up to 5 previous days
    /// (to handle weekends/holidays).
    pub fn get_rate(&self, currency_code: &str, date: NaiveDate) -> Result<f64, ApiError>;
}
```

**Configuration:**

| Parameter | Value |
|-----------|-------|
| Base URL | `https://www.cnb.cz/cs/financni-trhy/devizovy-trh/kurzy-devizoveho-trhu/kurzy-devizoveho-trhu/denni_kurz.txt` |
| Query param | `?date=DD.MM.YYYY` |
| Timeout | 10 seconds |
| Response limit | 256 KB |
| Cache TTL | 1 hour per date |
| Max fallback days | 5 |

**Currency code validation:** Must be exactly 3 uppercase ASCII characters. The input is uppercased automatically.

**Response format:** Pipe-delimited text (not JSON).

```
10.03.2026 #049
zeme|mena|mnozstvi|kod|kurz
Australie|dolar|1|AUD|14,820
...
```

- Line 1: date + sequence number (skip)
- Line 2: column headers (skip)
- Lines 3+: data rows
- Decimal separator is comma (replace `,` with `.` before parsing)

**`src/cnb/types.rs`**

```rust
#[derive(Debug, Clone)]
pub struct ExchangeRate {
    pub country: String,
    pub currency: String,
    pub amount: i32,       // how many units the rate applies to
    pub code: String,      // ISO 4217 (EUR, USD, etc.)
    pub rate: f64,          // CZK per `amount` units
}
```

**Rate calculation:** The CNB rate is CZK per `amount` units. Return `rate / amount` to get CZK per 1 unit.

**Fallback logic:** For a given date, if fetching or parsing fails, try the previous day. Repeat up to 5 times (covers weekends and holidays). If all 5 fail, return error.

**Cache:** `HashMap<String, CacheEntry>` protected by `std::sync::Mutex`. Check `fetched_at` against 1-hour TTL before returning cached data. The cache key is the date formatted as `DD.MM.YYYY`.

---

## FIO Bank Client

**Go source:** No existing Go implementation (only `FIOConfig` in config and `BankTransaction` domain type)
**Purpose:** Import bank transactions from FIO Bank API

### Module: `src/fio/`

**`src/fio/client.rs`**

```rust
pub struct FioClient {
    api_token: String,
    base_url: String,
    client: reqwest::blocking::Client,
}

impl FioClient {
    pub fn new(api_token: &str) -> Self;
    pub fn with_base_url(base_url: &str) -> Self;

    /// Fetch transactions for the given date range.
    /// Maps FIO API response to domain `BankTransaction` values.
    pub fn get_transactions(
        &self,
        from: NaiveDate,
        to: NaiveDate,
    ) -> Result<Vec<BankTransaction>, ApiError>;
}
```

**Configuration:**

| Parameter | Value |
|-----------|-------|
| Base URL | `https://www.fio.cz/ib_api/rest` |
| Endpoint | `GET /periods/{token}/{from}/{to}/transactions.json` |
| Date format | `YYYY-MM-DD` |
| Timeout | 30 seconds |
| Response limit | 10 MB |

**FIO API response structure** (JSON):

```json
{
  "accountStatement": {
    "info": { "accountId": "...", "bankId": "...", ... },
    "transactionList": {
      "transaction": [
        {
          "column0": { "value": 12345, "name": "ID pohybu" },
          "column1": { "value": "2026-03-15+0100", "name": "Datum" },
          "column2": { "value": 1234.56, "name": "Objem" },
          "column3": { "value": "CZK", "name": "Mena" },
          "column5": { "value": "1234567890", "name": "Protiucet" },
          "column10": { "value": "Jmeno protiuctu", "name": "Nazev protiuctu" },
          "column4": { "value": "123456", "name": "VS" },
          "column14": { "value": "0558", "name": "KS" },
          "column15": { "value": "123", "name": "SS" },
          "column16": { "value": "Zprava pro prijemce", "name": "Zprava" }
        }
      ]
    }
  }
}
```

> FIO uses numbered `column{N}` keys. Each column is either `{ "value": ..., "name": "..." }` or `null`.

**Mapping FIO columns to `BankTransaction`:**

| FIO Column | Field |
|------------|-------|
| column0 | (internal ID, not mapped) |
| column1 | `transaction_date` (parse date, strip timezone) |
| column2 | `amount` (convert to halere via `domain::Amount`) |
| column3 | `currency` |
| column5 | `counterparty_account` |
| column10 | `counterparty_name` |
| column4 | `variable_symbol` |
| column14 | `constant_symbol` |
| column15 | `specific_symbol` |
| column16 | `message` |

**Rate limiting:** FIO Bank allows 1 request per 30 seconds per token. The client does NOT enforce this -- the caller is responsible. If the API returns 409 (Conflict), map to `ApiError::RateLimited`.

---

## SMTP Email Sender

**Go source:** `internal/service/email/sender.go`
**Purpose:** Send emails with attachments (invoice PDFs, ISDOC XML)

### Module: `src/email/`

The Go implementation uses raw `net/smtp` with manual MIME construction. The Rust version uses `lettre` which handles MIME building, TLS negotiation, and encoding natively.

**`src/email/sender.rs`**

```rust
pub struct SmtpConfig {
    pub host: String,
    pub port: u16,
    pub username: String,
    pub password: String,
    pub from: String,
}

pub struct EmailSender {
    config: SmtpConfig,
}

impl EmailSender {
    pub fn new(config: SmtpConfig) -> Self;

    /// Returns true when SMTP host is configured (non-empty).
    pub fn is_configured(&self) -> bool;

    /// Send an email message. Fails if SMTP is not configured.
    pub fn send(&self, message: EmailMessage) -> Result<(), ApiError>;
}
```

**`src/email/types.rs`**

```rust
pub struct EmailMessage {
    pub to: Vec<String>,
    pub cc: Vec<String>,
    pub bcc: Vec<String>,
    pub subject: String,
    pub body_html: String,
    pub body_text: String,
    pub attachments: Vec<EmailAttachment>,
}

pub struct EmailAttachment {
    pub filename: String,
    pub content_type: String,
    pub data: Vec<u8>,
}
```

**TLS mode selection** (matching Go behavior):

| Port | Mode | lettre Transport |
|------|------|-----------------|
| 465 | Implicit TLS | `SmtpTransport::relay()` with implicit TLS |
| 587 | STARTTLS | `SmtpTransport::starttls_relay()` |
| Other | Plain SMTP | `SmtpTransport::builder_dangerous()` (no TLS) |

**Implementation details:**

- TLS minimum version: 1.2 (via `rustls` -- lettre's `rustls-tls` feature)
- Authentication: Optional. Use `Credentials::new(username, password)` only when username is non-empty
- MIME structure with attachments: `multipart/mixed` containing `multipart/alternative` (text + HTML) + attachment parts
- MIME structure without attachments: `multipart/alternative` (text + HTML)
- Subject encoding: lettre handles RFC 2047 UTF-8 encoding automatically
- Header injection prevention: strip `\r\n`, `\r`, `\n` from all header values before passing to lettre (defense in depth, though lettre also validates)
- Recipient deduplication: combine To + Cc + Bcc, deduplicate by email address (using `HashSet`)

**lettre message building:**

```rust
use lettre::message::{header::ContentType, Attachment, MultiPart, SinglePart, Mailbox};
use lettre::{Message, SmtpTransport, Transport};

// Build message
let mut email = Message::builder()
    .from(config.from.parse::<Mailbox>()?)
    .subject(&sanitize_header(&msg.subject));

for to in &msg.to {
    email = email.to(to.parse::<Mailbox>()?);
}
// ... cc, bcc

// Build body
let alternative = MultiPart::alternative()
    .singlepart(SinglePart::plain(msg.body_text.clone()))
    .singlepart(SinglePart::html(msg.body_html.clone()));

let body = if msg.attachments.is_empty() {
    alternative.into()
} else {
    let mut mixed = MultiPart::mixed().multipart(alternative);
    for att in &msg.attachments {
        let content_type = att.content_type.parse::<ContentType>()?;
        let attachment = Attachment::new(att.filename.clone())
            .body(att.data.clone(), content_type);
        mixed = mixed.singlepart(attachment);
    }
    mixed.into()
};
```

---

## OCR Provider

**Go source:** `internal/service/ocr/` (provider.go, anthropic.go, openai.go, shared.go, investment_prompt.go, factory.go)
**Purpose:** Extract invoice data from images/PDFs using AI APIs

### Module: `src/ocr/`

**`src/ocr/provider.rs` -- trait:**

```rust
pub trait OcrProvider: Send + Sync {
    /// Process an image and extract structured invoice data.
    fn process_image(
        &self,
        image_data: &[u8],
        content_type: &str,
    ) -> Result<OcrResult, ApiError>;

    /// Process an image with custom prompts and return raw text response.
    fn process_with_prompt(
        &self,
        image_data: &[u8],
        content_type: &str,
        system_prompt: &str,
        user_prompt: &str,
    ) -> Result<String, ApiError>;

    /// Provider name (e.g., "claude", "openai", "openrouter").
    fn name(&self) -> &str;
}
```

**Supported content types:** `image/jpeg`, `image/png`, `application/pdf`. Reject others with `ApiError::InvalidInput`.

### Anthropic Provider

**`src/ocr/anthropic.rs`**

```rust
pub struct AnthropicProvider {
    api_key: String,
    base_url: String,
    model: String,
    client: reqwest::blocking::Client,
}

impl AnthropicProvider {
    pub fn new(api_key: &str, model: Option<&str>) -> Self;
    pub fn with_base_url(mut self, url: &str) -> Self;
}
```

| Parameter | Value |
|-----------|-------|
| Base URL | `https://api.anthropic.com/v1/messages` |
| Default model | `claude-sonnet-4-20250514` |
| API version header | `anthropic-version: 2023-06-01` |
| Auth header | `x-api-key: {api_key}` |
| Max tokens | 4096 |
| Timeout | 30 seconds |
| Response limit | 2 MB |

**Request body** (Anthropic Messages API):

```json
{
  "model": "claude-sonnet-4-20250514",
  "max_tokens": 4096,
  "system": "<system_prompt>",
  "messages": [{
    "role": "user",
    "content": [
      { "type": "image", "source": { "type": "base64", "media_type": "image/jpeg", "data": "<base64>" } },
      { "type": "text", "text": "<user_prompt>" }
    ]
  }]
}
```

**Response parsing:** Extract `content[0].text` where `type == "text"`, then parse as OCR JSON.

### OpenAI-Compatible Provider

**`src/ocr/openai.rs`**

Supports OpenAI, OpenRouter, Mistral, and Gemini via the OpenAI Chat Completions API format.

```rust
pub struct OpenAICompatibleProvider {
    name: String,
    api_key: String,
    base_url: String,
    model: String,
    client: reqwest::blocking::Client,
}
```

**Factory constructors:**

| Constructor | Default Model | Base URL |
|------------|---------------|----------|
| `new_openai(api_key, model)` | `gpt-4o` | `https://api.openai.com/v1/chat/completions` |
| `new_openrouter(api_key, model)` | `google/gemini-2.0-flash-001` | `https://openrouter.ai/api/v1/chat/completions` |
| `new_mistral(api_key, model)` | `pixtral-large-latest` | `https://api.mistral.ai/v1/chat/completions` |
| `new_gemini(api_key, model)` | `gemini-2.0-flash` | `https://generativelanguage.googleapis.com/v1beta/openai/chat/completions` |

All share `with_base_url(url)` for testing.

| Parameter | Value |
|-----------|-------|
| Auth header | `Authorization: Bearer {api_key}` |
| Max tokens | 4096 |
| Temperature | 0.1 |
| Timeout | 25 seconds |
| Response limit | 2 MB |

**Image encoding:** Data URL format: `data:{content_type};base64,{base64_data}`

**Request body** (Chat Completions API):

```json
{
  "model": "gpt-4o",
  "max_tokens": 4096,
  "temperature": 0.1,
  "messages": [
    { "role": "system", "content": [{ "type": "text", "text": "<system_prompt>" }] },
    { "role": "user", "content": [
      { "type": "text", "text": "<user_prompt>" },
      { "type": "image_url", "image_url": { "url": "data:image/jpeg;base64,..." } }
    ]}
  ]
}
```

**Response parsing:** Extract `choices[0].message.content`, then parse as OCR JSON.

### Prompts

**`src/ocr/prompts.rs`**

Contains the Czech-language prompts as `const &str` values. Port directly from Go `shared.go` and `investment_prompt.go`.

**System prompt** (`SYSTEM_PROMPT`): Instructs the AI to extract structured invoice data and return JSON only.

**User prompt** (`USER_PROMPT`): "Analyzuj tento doklad (faktura/uctenka) a extrahuj vsechna dostupna data do JSON formatu podle zadane struktury."

**Investment system prompt** (`INVESTMENT_SYSTEM_PROMPT`): For brokerage/platform document extraction.

**Investment user prompt** (`investment_user_prompt(platform: &str) -> String`): Platform-specific hints for Portu, Zonky, Trading212, Revolut.

### OCR JSON Parsing

**`src/ocr/parse.rs`**

```rust
/// Parse OCR JSON response from AI model output into domain OcrResult.
/// Handles markdown code fences (```json ... ```) by stripping them first.
pub fn parse_ocr_json(content: &str) -> Result<OcrResult, ApiError>;

/// Parse investment extraction JSON from AI model output.
pub fn parse_investment_json(content: &str) -> Result<InvestmentExtractionResult, ApiError>;
```

**OCR JSON response schema** (what the AI returns):

```rust
#[derive(Debug, Deserialize)]
struct OcrJsonResponse {
    vendor_name: String,
    vendor_ico: String,
    vendor_dic: String,
    invoice_number: String,
    issue_date: String,         // "YYYY-MM-DD"
    due_date: String,           // "YYYY-MM-DD"
    total_amount: f64,          // CZK as decimal (1234.56)
    vat_amount: f64,
    vat_rate_percent: i32,
    currency_code: String,
    description: String,
    items: Vec<OcrItemResponse>,
    raw_text: String,
    confidence: f64,            // 0.0-1.0
}

#[derive(Debug, Deserialize)]
struct OcrItemResponse {
    description: String,
    quantity: f64,
    unit_price: f64,
    vat_rate_percent: i32,
    total_amount: f64,
}
```

**Amount conversion:** CZK floats from AI -> `domain::Amount` (halere):
- `czk_to_halere(czk: f64) -> i64` = `(czk * 100.0 + 0.5) as i64`
- `float_to_cents(f: f64) -> i64` = `(f * 100.0 + 0.5) as i64` (for quantities)

**Code fence stripping:** Remove leading ` ```json ` or ` ``` ` and trailing ` ``` `, plus surrounding whitespace.

### Provider Factory

**`src/ocr/factory.rs`**

```rust
/// Create an OCR provider by name.
/// Supported: "openai" (default), "openrouter", "mistral", "gemini", "claude".
pub fn new_provider(
    provider_name: &str,
    api_key: &str,
    model: Option<&str>,
    base_url: Option<&str>,
) -> Result<Box<dyn OcrProvider>, ApiError>;
```

---

## Fakturoid Import Client

**Go source:** `internal/fakturoid/client.go`, `internal/fakturoid/types.go`
**Purpose:** Import contacts, invoices, and expenses from Fakturoid

### Module: `src/fakturoid/`

**`src/fakturoid/client.rs`**

```rust
pub struct FakturoidClient {
    base_url: String,
    token_url: String,
    email: String,
    client_id: String,
    client_secret: String,
    access_token: Option<String>,
    client: reqwest::blocking::Client,
}

impl FakturoidClient {
    /// Create a new Fakturoid client.
    /// `slug` is the Fakturoid account slug (used in base URL).
    pub fn new(slug: &str, email: &str, client_id: &str, client_secret: &str) -> Self;

    /// Override the base URL (for testing).
    pub fn with_base_url(mut self, base_url: &str) -> Self;

    /// Obtain an OAuth2 access token (Client Credentials flow).
    /// Must be called before making API requests.
    pub fn authenticate(&mut self) -> Result<(), ApiError>;

    /// List all subjects (contacts) with auto-pagination.
    pub fn list_subjects(&self) -> Result<Vec<FakturoidSubject>, ApiError>;

    /// List all invoices with auto-pagination.
    pub fn list_invoices(&self) -> Result<Vec<FakturoidInvoice>, ApiError>;

    /// List all expenses with auto-pagination.
    pub fn list_expenses(&self) -> Result<Vec<FakturoidExpense>, ApiError>;

    /// Download a file attachment. Returns (bytes, content_type).
    pub fn download_attachment(&self, url: &str) -> Result<(Vec<u8>, String), ApiError>;
}
```

**Configuration:**

| Parameter | Value |
|-----------|-------|
| Base URL | `https://app.fakturoid.cz/api/v3/accounts/{slug}` |
| Token URL | `https://app.fakturoid.cz/api/v3/oauth/token` |
| Timeout | 30 seconds |
| User-Agent | `ZFaktury ({email})` |
| Page delay | 700 ms between pages (~85 req/min, under 100 limit) |
| Response limit | 10 MB per page |
| Attachment limit | 50 MB |

**OAuth2 flow:**

1. POST to token URL with `grant_type=client_credentials`
2. Basic auth: `client_id:client_secret`
3. Headers: `Content-Type: application/x-www-form-urlencoded`, `Accept: application/json`
4. Parse `access_token` from JSON response
5. Store token in client for subsequent requests

**Auto-pagination:**

1. Start at `page=1`
2. GET `{base_url}/{resource}.json?page={page}`
3. Headers: `Authorization: Bearer {token}`, `Accept: application/json`, `User-Agent: ZFaktury ({email})`
4. Parse JSON array
5. If empty array -> done, break
6. Append items, increment page
7. Sleep 700ms between pages (rate limiting)
8. Repeat

**`src/fakturoid/types.rs`**

```rust
/// Custom deserializer for float values that may come as JSON strings.
/// Fakturoid API returns some numeric fields (e.g., exchange_rate) as strings.
#[derive(Debug, Clone, Copy)]
pub struct FlexFloat64(pub f64);

// Implement Deserialize to handle both number and string JSON values.

#[derive(Debug, Deserialize)]
pub struct FakturoidSubject {
    pub id: i64,
    pub name: String,
    pub registration_no: Option<String>,   // ICO
    pub vat_no: Option<String>,            // DIC
    pub street: Option<String>,
    pub city: Option<String>,
    pub zip: Option<String>,
    pub country: Option<String>,
    pub bank_account: Option<String>,      // "cislo/kod" format
    pub iban: Option<String>,
    pub email: Option<String>,
    pub phone: Option<String>,
    pub web: Option<String>,
    #[serde(rename = "type")]
    pub type_: Option<String>,             // "customer", "supplier", "both"
    pub due: Option<i32>,                  // payment terms days
}

#[derive(Debug, Deserialize)]
pub struct FakturoidAttachment {
    pub id: i64,
    pub filename: String,
    pub content_type: String,
    pub download_url: String,
}

#[derive(Debug, Deserialize)]
pub struct FakturoidInvoiceLine {
    pub name: String,
    pub quantity: FlexFloat64,
    pub unit_name: Option<String>,
    pub unit_price: FlexFloat64,
    pub vat_rate: FlexFloat64,
}

#[derive(Debug, Deserialize)]
pub struct FakturoidPayment {
    pub paid_on: Option<String>,           // "YYYY-MM-DD"
}

#[derive(Debug, Deserialize)]
pub struct FakturoidInvoice {
    pub id: i64,
    pub number: String,
    pub document_type: String,             // "invoice", "proforma", "correction", etc.
    pub status: String,                    // "open", "sent", "overdue", "paid", etc.
    pub issued_on: String,                 // "YYYY-MM-DD"
    pub due_on: Option<String>,
    pub taxable_fulfillment_due: Option<String>,
    pub variable_symbol: Option<String>,
    pub subject_id: i64,
    pub currency: String,
    pub exchange_rate: FlexFloat64,
    pub subtotal: FlexFloat64,
    pub total: FlexFloat64,
    pub payment_method: Option<String>,    // "bank", "cash", "card", etc.
    pub note: Option<String>,
    pub lines: Vec<FakturoidInvoiceLine>,
    #[serde(default)]
    pub payments: Vec<FakturoidPayment>,
    #[serde(default)]
    pub attachments: Vec<FakturoidAttachment>,
}

#[derive(Debug, Deserialize)]
pub struct FakturoidExpenseLine {
    pub name: String,
    pub quantity: FlexFloat64,
    pub unit_price: FlexFloat64,
    pub vat_rate: FlexFloat64,
}

#[derive(Debug, Deserialize)]
pub struct FakturoidExpense {
    pub id: i64,
    pub original_number: Option<String>,
    pub issued_on: String,
    pub subject_id: i64,
    pub description: Option<String>,
    pub total: FlexFloat64,
    pub currency: String,
    pub exchange_rate: FlexFloat64,
    pub payment_method: Option<String>,
    pub private_note: Option<String>,
    pub lines: Vec<FakturoidExpenseLine>,
    #[serde(default)]
    pub attachments: Vec<FakturoidAttachment>,
}
```

---

## Module Structure

```
zfaktury-api/
  Cargo.toml
  src/
    lib.rs              # pub mod declarations, re-exports
    error.rs            # ApiError enum
    ares/
      mod.rs
      client.rs
      types.rs
    cnb/
      mod.rs
      client.rs
      types.rs
    fio/
      mod.rs
      client.rs
      types.rs
    email/
      mod.rs
      sender.rs
      types.rs
    ocr/
      mod.rs
      provider.rs       # OcrProvider trait
      anthropic.rs
      openai.rs
      prompts.rs        # Czech prompt constants
      parse.rs          # JSON parsing (OCR + investment)
      factory.rs
    fakturoid/
      mod.rs
      client.rs
      types.rs
```

---

## Testing Strategy

All HTTP clients are tested with `wiremock`. Since `wiremock` is async-only, tests use `#[tokio::test]` with `tokio::task::spawn_blocking` for the synchronous client calls.

**Test helper pattern:**

```rust
#[tokio::test]
async fn test_ares_lookup_success() {
    let mock_server = MockServer::start().await;

    Mock::given(method("GET"))
        .and(path("/ekonomicke-subjekty/12345678"))
        .respond_with(ResponseTemplate::new(200).set_body_json(ares_fixture()))
        .mount(&mock_server)
        .await;

    let client = AresClient::with_base_url(&mock_server.uri());
    let result = tokio::task::spawn_blocking(move || {
        client.lookup_by_ico("12345678")
    }).await.unwrap();

    let contact = result.unwrap();
    assert_eq!(contact.name, "Test Company s.r.o.");
    assert_eq!(contact.country, "CZ");
}
```

### Test Scenarios per Client

**ARES:**
- Successful lookup with full address
- Successful lookup with fallback to `textova_adresa` (no street/house number)
- ICO not found (404)
- Invalid ICO format (not 8 digits)
- Rate limited (429)
- Server error (500)
- Malformed JSON response
- Timeout

**CNB:**
- Rate for known currency on a weekday
- Weekend fallback (Saturday -> uses Friday data)
- Multiple fallback days (holiday period)
- Unknown currency code
- Invalid currency code (not 3 chars)
- Malformed pipe-delimited response
- Empty response (no rates parsed)
- Cache hit (verify no second HTTP request)

**FIO:**
- Successful transaction list with all fields populated
- Transactions with null/missing columns
- Empty transaction list
- Rate limited (409)
- Authentication error (invalid token)

**Fakturoid:**
- Successful OAuth2 authentication
- Auth failure (401)
- List subjects with single page
- List invoices with multi-page pagination (verify 700ms delay)
- List expenses (empty result)
- Download attachment success
- Download attachment 204 No Content error
- FlexFloat64 deserializing from both number and string

**Email (lettre test transport):**
- Successful send with text + HTML body
- Successful send with attachments
- Not configured error (empty host)
- Recipient deduplication (same email in To and Cc)
- Non-ASCII subject encoding
- Header injection prevention (subject with CRLF)

**OCR:**
- Anthropic: successful image extraction with valid JSON
- Anthropic: API error in response body
- OpenAI: successful extraction
- OpenAI: no choices in response
- Parse: valid JSON with all fields
- Parse: JSON wrapped in markdown code fences
- Parse: low confidence result (still valid)
- Parse: malformed JSON from model
- Factory: known provider names resolve correctly
- Factory: unknown provider name returns error
- Investment prompt: platform-specific hints

---

## Acceptance Criteria

1. All clients compile without warnings (`cargo build`)
2. All unit tests pass (`cargo test`)
3. ARES lookup returns correct `Contact` domain struct with proper address building
4. CNB rates handle weekend/holiday fallback correctly; cache prevents redundant HTTP requests
5. FIO client parses the column-based JSON format into `BankTransaction` values
6. Email sends with attachments verified via lettre's `MockTransport` (no real SMTP needed)
7. OCR parses AI responses to `OcrResult` correctly, including code fence stripping
8. Fakturoid handles OAuth2 token acquisition, auto-pagination with rate limiting, and `FlexFloat64` deserialization
9. All error paths tested (not just happy path)

## Test Coverage Requirements

| Scope | Target |
|-------|--------|
| All modules in `zfaktury-api/` | 90%+ |
| HTTP client error handling | 100% |
| Response parsing | 100% |
| OCR JSON parsing + code fence stripping | 100% |
| FlexFloat64 deserialization | 100% |

## Review Checklist

- [ ] All HTTP clients use configurable base URLs for testing
- [ ] Timeouts set on all HTTP requests (no unbounded waits)
- [ ] Response size limits enforced (1 MB ARES, 256 KB CNB, 2 MB OCR, 10 MB Fakturoid, 50 MB attachments)
- [ ] No secrets hardcoded -- all from config/constructor parameters
- [ ] Rate limiting respected (Fakturoid 700ms inter-page delay)
- [ ] Error types are descriptive, not generic "request failed" strings
- [ ] SMTP TLS minimum version 1.2 enforced
- [ ] Header injection prevention in email sender
- [ ] `FlexFloat64` handles both JSON number and string inputs
- [ ] OCR code fence stripping handles `json`, bare ` ``` `, and no fences
- [ ] CNB comma-to-dot decimal conversion tested
- [ ] FIO null column handling tested (missing optional fields)
- [ ] All serde types use `#[serde(default)]` where API fields may be absent
- [ ] No `unwrap()` in production code -- all errors propagated via `Result`
