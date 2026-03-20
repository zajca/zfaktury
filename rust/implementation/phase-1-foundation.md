# RFC: Phase 1 -- Foundation (Domain, Config, Calc, Test Utilities)

**Status:** Draft
**Estimated effort:** 2-3 weeks
**Depends on:** Workspace layout (Cargo.toml hierarchy)

---

## 1. Summary

Phase 1 ports the pure, side-effect-free core of ZFaktury from Go to Rust: all domain types, the Amount newtype, TOML configuration, every calculation module, and shared test utilities. Nothing in this phase touches a database, HTTP, or filesystem at runtime -- it is 100% deterministic logic with 100% test coverage on the calc layer.

Four new crates are introduced:

| Crate | Purpose | External deps |
|---|---|---|
| `zfaktury-domain` | Pure domain structs, Amount, enums, errors | `chrono`, `thiserror` |
| `zfaktury-config` | TOML config loading, data dir resolution | `serde`, `toml`, `dirs` |
| `zfaktury-core` | Calculation modules (VAT, income tax, insurance, FIFO, ...) | depends on `zfaktury-domain` only |
| `zfaktury-testutil` | Test builders, golden file helpers, future test DB setup | `zfaktury-domain`, dev-only |

---

## 2. Motivation

The Go codebase mixes pure computation with service-layer orchestration. For example, FIFO cost-basis calculation lives inside `InvestmentIncomeService.RecalculateFIFO` (lines 193-313 of `investment_income_svc.go`), interleaved with database reads and writes. The Rust rewrite is an opportunity to:

1. **Extract all calculations into pure functions** -- no `&self`, no `ctx`, no repository calls.
2. **Establish the `Amount(i64)` newtype as a first-class citizen** with Rust operator traits, preventing accidental float usage for money.
3. **Achieve 100% test coverage on calculations** from day one, using property-based testing (`proptest`) and table-driven tests (`rstest`).
4. **Define the domain vocabulary once** -- all later phases (repository, service, handler) import these types without redefinition.

---

## 3. Detailed Design

### 3.1 `zfaktury-domain` Crate

Zero runtime dependencies beyond `chrono` (for `NaiveDate`/`NaiveDateTime`) and `thiserror` (for `DomainError`). No `serde` derives -- domain types are deliberately opaque to serialization frameworks. Serde integration happens at the boundary (handler DTOs, repository mappers) in later phases.

#### 3.1.1 `amount.rs` -- `Amount(i64)` Newtype

Ported from `internal/domain/money.go`. The Go codebase represents all money as `type Amount int64` with 100 subunits per unit (halere/cents). The Rust version preserves this exactly.

```rust
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash, Default)]
pub struct Amount(i64);
```

**Constructors:**
- `Amount::new(whole: i64, fraction: i64) -> Amount` -- maps to Go's `NewAmount(whole, fraction)`. Returns `Amount(whole * 100 + fraction)`.
- `Amount::from_float(f: f64) -> Amount` -- maps to Go's `FromFloat`. Uses `f64::round(f * 100.0) as i64`. This is the ONLY place float touches Amount.
- `Amount::from_halere(halere: i64) -> Amount` -- direct constructor for raw halere values (used in tests and DB mapping).
- `Amount::ZERO` -- const zero value.

**Conversions:**
- `to_czk(&self) -> f64` -- for display only, never for arithmetic.
- `halere(&self) -> i64` -- raw value access for DB storage and integer arithmetic.

**Display trait:**
```
 0 -> "0.00"
 10050 -> "100.50"
 -99 -> "-0.99"
 -2550 -> "-25.50"
```
Special case: when the whole part is 0 and the amount is negative (`-0.XX`), the sign must be preserved. Ported from Go logic at `money.go:31-38`.

**Operator traits:**
- `Add<Amount> for Amount` and `AddAssign`
- `Sub<Amount> for Amount` and `SubAssign`
- `Neg for Amount`
- `Mul<Amount> for Amount` -- integer multiplication only (used for sign flipping in VAT: `base * sign` where sign is `Amount(1)` or `Amount(-1)`)
- `Amount::multiply(&self, factor: f64) -> Amount` -- named method for float multiplication (VAT rates, percentages). Uses `f64::round(self.0 as f64 * factor) as i64`.

**Predicate methods:**
- `is_zero(&self) -> bool`
- `is_negative(&self) -> bool`

**Currency constants:**
```rust
pub const CURRENCY_CZK: &str = "CZK";
pub const CURRENCY_EUR: &str = "EUR";
pub const CURRENCY_USD: &str = "USD";
```

**Design decision -- no `Mul<f64>` trait impl:** Deliberately using a named method `multiply(f64)` instead of `impl Mul<f64>` to make float multiplication visually distinct from integer operations. Every call site will read `.multiply(0.21)` rather than `* 0.21`, making it grep-able and audit-friendly.

#### 3.1.2 `errors.rs` -- Domain Errors

Ported from `internal/domain/errors.go`. Uses `thiserror` for `Display` + `Error` derivation.

```rust
#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
pub enum DomainError {
    #[error("not found")]
    NotFound,
    #[error("invalid input")]
    InvalidInput,
    #[error("invoice already paid")]
    PaidInvoice,
    #[error("no items")]
    NoItems,
    #[error("duplicate number")]
    DuplicateNumber,
    #[error("filing already exists for this period")]
    FilingAlreadyExists,
    #[error("filing already filed, cannot modify")]
    FilingAlreadyFiled,
    #[error("required setting not configured")]
    MissingSetting,
    #[error("invoice is not overdue")]
    InvoiceNotOverdue,
    #[error("customer has no email address")]
    NoCustomerEmail,
}
```

This enum is `Clone + PartialEq` so tests can assert error variants directly with `assert_eq!` rather than string matching. The `thiserror::Error` derive provides `Display` and `std::error::Error` impls.

#### 3.1.3 Domain Struct Modules

Each Go domain file maps to a Rust module. All structs are plain data -- no `serde` derives, no ORM annotations. Fields use `Amount` for all monetary values, `NaiveDate`/`NaiveDateTime` from `chrono` for dates, and `Option<T>` for nullable fields.

**Enums use string-backed representation for DB compatibility.** Each enum gets a `as_str(&self) -> &str` method and a `TryFrom<&str>` impl. This avoids integer-based DB columns and keeps migrations readable. Example:

```rust
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum InvoiceType {
    Regular,
    Proforma,
    CreditNote,
}

impl InvoiceType {
    pub fn as_str(&self) -> &str {
        match self {
            Self::Regular => "regular",
            Self::Proforma => "proforma",
            Self::CreditNote => "credit_note",
        }
    }
}

impl TryFrom<&str> for InvoiceType {
    type Error = DomainError;
    fn try_from(s: &str) -> Result<Self, Self::Error> {
        match s {
            "regular" => Ok(Self::Regular),
            "proforma" => Ok(Self::Proforma),
            "credit_note" => Ok(Self::CreditNote),
            _ => Err(DomainError::InvalidInput),
        }
    }
}
```

The following modules and their primary types are ported:

**`contact.rs`** (from `contact.go` + `contact_filter.go`)
- `ContactType` enum: `Company`, `Individual`
- `Contact` struct: 24 fields. Notable: `vat_unreliable_at: Option<NaiveDateTime>`, `tags: String` (comma-separated, same as Go), `payment_terms_days: Option<i32>`
- `ContactFilter` struct: `search: Option<String>`, `contact_type: Option<ContactType>`, `favorite: Option<bool>`, `limit: i64`, `offset: i64`
- Methods: `dic_country_code(&self) -> Option<&str>`, `is_eu_partner(&self) -> bool`, `has_cz_dic(&self) -> bool`

**`invoice.rs`** (from `invoice.go`)
- `InvoiceType` enum: `Regular`, `Proforma`, `CreditNote`
- `InvoiceStatus` enum: `Draft`, `Sent`, `Paid`, `Overdue`, `Cancelled`
- `RelationType` enum: `Settlement`, `CreditNote`
- `Invoice` struct: 30+ fields. All amounts are `Amount`. `customer: Option<Contact>` for eager-loaded relations. `items: Vec<InvoiceItem>`.
- `InvoiceItem` struct: 10 fields. `quantity: Amount`, `unit_price: Amount`, `vat_rate_percent: i32`, `vat_amount: Amount`, `total_amount: Amount`, `sort_order: i32`.
- `InvoiceSequence` struct: `id`, `prefix`, `next_number: i32`, `year: i32`, `format_pattern: String`.
- `InvoiceFilter` struct: `status`, `invoice_type`, `customer_id`, `date_from`, `date_to`, `search`, `limit`, `offset`.
- Methods: `calculate_totals(&mut self)`, `is_overdue(&self) -> bool`, `is_paid(&self) -> bool`.

**`expense.rs`** (from `expense.go`)
- `Expense` struct: 22 fields. `vendor: Option<Contact>`, `items: Vec<ExpenseItem>`, `business_percent: i32`.
- `ExpenseItem` struct: 10 fields, parallel to `InvoiceItem`.
- `ExpenseDocument` struct: `id`, `expense_id`, `filename`, `content_type`, `storage_path`, `size: i64`, `created_at`, `deleted_at: Option<NaiveDateTime>`.
- `ExpenseCategory` struct: `id`, `key`, `label_cs`, `label_en`, `color`, `sort_order`, `is_default`, `created_at`, `deleted_at`.
- `ExpenseFilter` struct.

**`recurring.rs`** (from `recurring.go` + `recurring_expense.go`)
- `Frequency` enum: `Weekly`, `Monthly`, `Quarterly`, `Yearly`
- `RecurringInvoice` struct with `items: Vec<RecurringInvoiceItem>`
- `RecurringExpense` struct
- `RecurringInvoice::next_date(&self) -> NaiveDate` -- ported from Go's `NextDate()`. Uses `chrono::NaiveDate::checked_add_months` for month-end edge cases. **Important difference from Go:** Go's `time.Time.AddDate(0, 1, 0)` on Jan 31 returns March 2-3 depending on year. In Rust, we use `checked_add_months` which clamps to the last day of the month (Jan 31 + 1 month = Feb 28/29). This is the desired behavior for recurring invoices -- document this as an intentional improvement.

**`tax.rs`** (from `tax.go`)
- `FilingType` enum: `Regular`, `Corrective`, `Supplementary`
- `FilingStatus` enum: `Draft`, `Ready`, `Filed`
- `TaxPeriod` struct: `year: i32`, `month: i32`, `quarter: i32`
- `VATReturn` struct: 20+ fields including `output_vat_base_21`, `input_vat_base_12`, etc. All amounts are `Amount`. `xml_data: Vec<u8>`.
- `VATControlStatement`, `VATControlStatementLine` (with `Section` enum: `A4`, `A5`, `B2`, `B3`)
- `VIESSummary`, `VIESSummaryLine`
- `IncomeTaxReturn` struct: 40+ fields covering the full Czech DPFO form (progressive brackets, credits, deductions, child benefit, prepayments, capital/other income). Every monetary field is `Amount`.
- `SocialInsuranceOverview`, `HealthInsuranceOverview`: identical structure with `tax_base`, `assessment_base`, `min_assessment_base`, `final_assessment_base`, `insurance_rate: i32`, `total_insurance`, `prepayments`, `difference`, `new_monthly_prepay`.
- `TaxYearSettings` struct: `year`, `flat_rate_percent: i32`, timestamps.
- `TaxPrepayment` struct: `year`, `month`, `tax_amount`, `social_amount`, `health_amount`.

**`tax_credits.rs`** (from `tax_credits.go`)
- `DeductionCategory` enum: `Mortgage`, `LifeInsurance`, `Pension`, `Donation`, `UnionDues`
- `TaxSpouseCredit`, `TaxChildCredit`, `TaxPersonalCredits`, `TaxDeduction`, `TaxDeductionDocument` structs.
- String constants for DB compatibility: `DeductionCategory::as_str()` returns `"mortgage"`, `"life_insurance"`, etc. matching Go's `domain.DeductionMortgage` etc.

**`investment.rs`** (from `investment_income.go`)
- `Platform` enum: `Portu`, `Zonky`, `Trading212`, `Revolut`, `Other`
- `ExtractionStatus` enum: `Pending`, `Extracted`, `Failed`
- `CapitalCategory` enum: `DividendCZ`, `DividendForeign`, `Interest`, `Coupon`, `FundDist`, `Other`
- `AssetType` enum: `Stock`, `ETF`, `Bond`, `Fund`, `Crypto`, `Other`
- `TransactionType` enum: `Buy`, `Sell`
- `InvestmentDocument`, `CapitalIncomeEntry`, `SecurityTransaction`, `InvestmentYearSummary` structs.
- **`SecurityTransaction.quantity`** is `i64` (1/10000 units, so 1 share = 10000). **Not** `Amount` -- this is a quantity, not money.
- **`SecurityTransaction.exchange_rate`** is `i64` (rate * 10000 for precision). Also not `Amount`.

**`audit.rs`** (from `audit.go` + `backup.go`)
- `AuditLogEntry`, `AuditLogFilter`
- `BackupTrigger` enum: `Manual`, `Scheduled`, `CLI`
- `BackupStatus` enum: `Running`, `Completed`, `Failed`
- `BackupRecord` struct.

**`ocr.rs`** (from `ocr.go`)
- `OCRResult`, `OCRItem` structs.

**`import.rs`** (from `import.go` + `fakturoid_import.go`)
- `ImportResult`, `FakturoidImportItem`, `FakturoidImportPreview`, `FakturoidImportResult`, `FakturoidImportLog` structs.

**`reminder.rs`** (from `reminder.go` + `invoice_status_history.go` + `invoice_document.go`)
- `PaymentReminder`, `InvoiceStatusChange`, `InvoiceDocument` structs.

**`settings.rs`** (from `bank.go` and settings patterns in handlers)
- `PdfSettings` struct: `logo_path`, `accent_color`, `footer_text`, `show_qr`, `show_bank_details`, `font_size`.
- `EmailDefaults` struct: `attach_pdf`, `attach_isdoc`, `subject_template`, `body_template`.
- Settings are stored as `HashMap<String, String>` in the DB (key-value store). These structs are convenience wrappers for typed access.

**`lib.rs`** -- Re-exports:
```rust
pub mod amount;
pub mod audit;
pub mod contact;
pub mod errors;
pub mod expense;
pub mod import;
pub mod investment;
pub mod invoice;
pub mod ocr;
pub mod recurring;
pub mod reminder;
pub mod settings;
pub mod tax;
pub mod tax_credits;

pub use amount::Amount;
pub use errors::DomainError;
```

### 3.2 `zfaktury-config` Crate

Ported from `internal/config/config.go` and `internal/config/validate.go`. Uses `serde::Deserialize` for TOML parsing (this is a boundary crate, so serde is appropriate here).

#### 3.2.1 Config Structs

```rust
#[derive(Debug, Deserialize)]
pub struct Config {
    #[serde(default)]
    pub data_dir: Option<PathBuf>,
    #[serde(default)]
    pub database: DatabaseConfig,
    #[serde(default)]
    pub log: LogConfig,
    #[serde(default)]
    pub server: ServerConfig,
    #[serde(default)]
    pub backup: BackupConfig,
    #[serde(default)]
    pub smtp: Option<SmtpConfig>,
    #[serde(default)]
    pub fio: Option<FioConfig>,
    #[serde(default)]
    pub ocr: Option<OcrConfig>,
}
```

All sub-configs follow the same pattern. `ServerConfig` defaults to port 8080. Optional integrations (`smtp`, `fio`, `ocr`) are `Option<T>` -- if the TOML section is absent, they are `None`.

#### 3.2.2 Config Loading

```rust
pub fn load(config_path: &Path) -> Result<Config, ConfigError>
pub fn resolve(explicit: Option<&Path>, init_config: bool) -> Result<PathBuf, ConfigError>
pub fn default_config_path() -> Result<PathBuf, ConfigError>
pub fn write_template(path: &Path) -> Result<(), ConfigError>
```

- `resolve()` mirrors Go's `config.Resolve()`: if explicit path given and file missing, either create template (if `init_config`) or return error. If no explicit path, use `~/.zfaktury/config.toml` and auto-create.
- `load()` reads TOML, applies `ZFAKTURY_DATA_DIR` env override, expands `~` in paths, creates data directory if needed.
- `Config::database_path(&self) -> PathBuf` -- returns `database.path` if set, otherwise `data_dir/zfaktury.db`.
- `Config::backup_destination(&self) -> PathBuf` -- returns `backup.destination` if set, otherwise `data_dir/backups`.

#### 3.2.3 Validation

Ported from `validate.go`. Called after loading:

```rust
impl Config {
    pub fn validate(&self) -> Result<(), ConfigError> { ... }
}
```

Rules:
- `server.port` must be 0-65535
- If `smtp.host` is set, `smtp.port` must be non-zero
- If `ocr.provider` is set, `ocr.api_key` must be non-empty

Validation errors are collected and returned as a single multi-line error message, same as Go.

#### 3.2.4 `ConfigError`

```rust
#[derive(Debug, thiserror::Error)]
pub enum ConfigError {
    #[error("config file not found: {path}\nUse --init-config to create a default config file")]
    NotFound { path: PathBuf },
    #[error("reading config file {path}: {source}")]
    ReadError { path: PathBuf, source: std::io::Error },
    #[error("parsing config file {path}: {source}")]
    ParseError { path: PathBuf, source: toml::de::Error },
    #[error("creating directory {path}: {source}")]
    DirCreateError { path: PathBuf, source: std::io::Error },
    #[error("config validation failed:\n  - {}", errors.join("\n  - "))]
    ValidationError { errors: Vec<String> },
    #[error("home directory not found")]
    NoHomeDir,
}
```

### 3.3 `zfaktury-core/src/calc/` Modules

All calculation functions are **pure**: they take input structs, return output structs, perform no I/O. Every function in this module is ported 1:1 from `internal/calc/` with identical semantics.

#### 3.3.1 `constants.rs`

Ported from `internal/calc/constants.go`. Per-year tax constants for 2024, 2025, 2026.

```rust
pub struct TaxYearConstants {
    pub progressive_threshold: Amount,
    pub basic_credit: Amount,
    pub spouse_credit: Amount,
    pub student_credit: Amount,
    pub disability_credit_1: Amount,
    pub disability_credit_3: Amount,
    pub disability_ztpp: Amount,
    pub child_benefit_1: Amount,
    pub child_benefit_2: Amount,
    pub child_benefit_3_plus: Amount,
    pub social_min_monthly: Amount,
    pub social_rate: i32,          // permille*10: 292 = 29.2%
    pub health_min_monthly: Amount,
    pub health_rate: i32,          // permille*10: 135 = 13.5%
    pub flat_rate_caps: HashMap<i32, Amount>,  // percent -> max amount
    pub time_test_years: i32,
    pub security_exemption_limit: Amount,
    pub spouse_income_limit: Amount,
    pub deduction_cap_mortgage: Amount,
    pub deduction_cap_life_insurance: Amount,
    pub deduction_cap_pension: Amount,
    pub deduction_cap_union_dues: Amount,
    pub max_child_bonus: Amount,
}

pub fn get_tax_constants(year: i32) -> Result<TaxYearConstants, DomainError>
```

Constants are stored in a `phf::Map` (compile-time hash map) or a simple `match` statement. Using `match` is simpler and sufficient for 3 years:

```rust
pub fn get_tax_constants(year: i32) -> Result<TaxYearConstants, DomainError> {
    match year {
        2024 => Ok(constants_2024()),
        2025 => Ok(constants_2025()),
        2026 => Ok(constants_2026()),
        _ => Err(DomainError::InvalidInput),
    }
}
```

Exact values ported from Go (all verified against the Go test suite):

| Constant | 2024 | 2025 | 2026 |
|---|---|---|---|
| `progressive_threshold` | 1,582,812 CZK | 1,582,812 CZK | 1,582,812 CZK |
| `basic_credit` | 30,840 CZK | 30,840 CZK | 30,840 CZK |
| `social_min_monthly` | 11,024 CZK | 11,584 CZK | 12,139 CZK |
| `health_min_monthly` | 10,081 CZK | 10,874 CZK | 11,396 CZK |
| `security_exemption_limit` | 0 (no limit) | 100,000,000 (1M CZK) | 100,000,000 |

#### 3.3.2 `vat.rs`

Ported from `internal/calc/vat.go`.

```rust
pub struct VATInvoiceInput {
    pub invoice_type: InvoiceType,
    pub items: Vec<VATItemInput>,
}

pub struct VATItemInput {
    pub base: Amount,
    pub vat_amount: Amount,
    pub vat_rate_percent: i32,
}

pub struct VATExpenseInput {
    pub amount: Amount,
    pub vat_amount: Amount,
    pub vat_rate_percent: i32,
    pub business_percent: i32,  // 0 treated as 100
}

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

pub fn calculate_vat_return(invoices: &[VATInvoiceInput], expenses: &[VATExpenseInput]) -> VATResult
```

Logic:
- Credit notes: sign = -1 (multiply base and vat by `Amount(-1)`)
- Regular invoices: sign = +1
- Items with VAT rate 0% are excluded (fall through the match)
- Expense business_percent: if 0, treat as 100
- Derived invariants: `total_output_vat = amount_21 + amount_12`, `net_vat = total_output - total_input`

#### 3.3.3 `income_tax.rs`

Ported from `internal/calc/income_tax.go`. The core progressive income tax calculation.

```rust
pub struct IncomeTaxInput {
    pub total_revenue: Amount,
    pub actual_expenses: Amount,
    pub flat_rate_percent: i32,
    pub constants: TaxYearConstants,
    pub spouse_credit: Amount,
    pub disability_credit: Amount,
    pub student_credit: Amount,
    pub child_benefit: Amount,
    pub total_deductions: Amount,
    pub prepayments: Amount,
    pub capital_income_net: Amount,
    pub other_income_net: Amount,
}

pub struct IncomeTaxResult {
    pub flat_rate_amount: Amount,
    pub used_expenses: Amount,
    pub tax_base: Amount,
    pub tax_base_rounded: Amount,
    pub tax_at_15: Amount,
    pub tax_at_23: Amount,
    pub total_tax: Amount,
    pub credit_basic: Amount,
    pub total_credits: Amount,
    pub tax_after_credits: Amount,
    pub tax_after_benefit: Amount,
    pub tax_due: Amount,
}

pub fn calculate_income_tax(input: &IncomeTaxInput) -> IncomeTaxResult
```

Steps (matching Go exactly):
1. **Determine used expenses:** If `flat_rate_percent > 0`, compute `revenue * percent / 100`, apply cap from `constants.flat_rate_caps`. Otherwise use `actual_expenses`.
2. **Tax base:** `revenue - used_expenses + capital_income_net + other_income_net`, clamped to 0.
3. **Apply deductions:** `tax_base - total_deductions`, clamped to 0.
4. **Round down to 100 CZK:** `(halere / 10_000) * 10_000` (integer division in halere).
5. **Progressive brackets:** If `rounded <= threshold`: 15% only. Otherwise: 15% on threshold + 23% on remainder.
6. **Tax credits:** `basic + spouse + disability + student`. Tax after credits clamped to 0 (credits cannot create a refund).
7. **Child benefit:** Subtracted from tax_after_credits. **Can go negative** (this is a bonus, not a credit).
8. **Prepayments:** Subtracted. Can go negative (= refund).

#### 3.3.4 `credits.rs`

Ported from `internal/calc/credits.go`. Four pure functions:

```rust
pub fn compute_spouse_credit(spouse_income: Amount, months_claimed: i32, spouse_ztp: bool, constants: &TaxYearConstants) -> Amount
pub fn compute_personal_credits(disability_level: i32, is_student: bool, student_months: i32, constants: &TaxYearConstants) -> (Amount, Amount)
pub fn compute_child_benefit(children: &[ChildCreditInput], constants: &TaxYearConstants) -> Amount
pub fn compute_deductions(deductions: &[DeductionInput], tax_base: Amount, constants: &TaxYearConstants) -> DeductionResult
```

**`compute_spouse_credit`:** Returns 0 if `spouse_income >= constants.spouse_income_limit` (68,000 CZK). Otherwise `credit * months / 12`, doubled if ZTP.

**`compute_personal_credits`:** Returns `(disability, student)` tuple. Disability levels: 1 = credit_1 (2,520), 2 = credit_3 (5,040), 3 = ztpp (16,140). Student credit proportional to months.

**`compute_child_benefit`:** Order-dependent: 1st child gets `child_benefit_1`, 2nd gets `child_benefit_2`, 3rd+ gets `child_benefit_3_plus`. ZTP doubles the amount. Proportional to months.

**`compute_deductions`:** Each category has a statutory cap. Multiple deductions of the same category share the cap (first gets full allocation, subsequent get remainder). Donation cap = 15% of tax_base. Unknown categories get 0. Negative claims clamped to 0.

```rust
pub struct ChildCreditInput {
    pub child_order: i32,  // 1, 2, or 3+ (>=3 treated as 3+)
    pub months_claimed: i32,
    pub ztp: bool,
}

pub struct DeductionInput {
    pub category: DeductionCategory,
    pub claimed_amount: Amount,
}

pub struct DeductionResult {
    pub allowed_amounts: Vec<Amount>,  // parallel to input slice
    pub total_allowed: Amount,
}
```

#### 3.3.5 `insurance.rs`

Ported from `internal/calc/insurance.go`. Two functions:

```rust
pub fn calculate_insurance(input: &InsuranceInput) -> InsuranceResult
pub fn resolve_used_expenses(revenue: Amount, actual_expenses: Amount, flat_rate_percent: i32, caps: &HashMap<i32, Amount>) -> Amount
```

**`calculate_insurance`:** All arithmetic in raw i64 halere.
1. `tax_base = max(0, revenue - used_expenses)`
2. `assessment_base = tax_base / 2` (integer division)
3. `min_assessment_base = min_monthly_base * 12`
4. `final_assessment_base = max(assessment_base, min_assessment_base)`
5. `total_insurance = final_assessment_base * rate_permille / 1000`
6. `difference = total_insurance - prepayments`
7. `new_monthly_prepay = ceil(total_insurance / 12)`, then round up to nearest 100 halere (1 CZK): `((monthly + 99) / 100) * 100`

**`resolve_used_expenses`:** If flat_rate_percent > 0, compute `revenue * percent / 100` with cap. Otherwise return actual_expenses. This is shared between income tax and insurance services.

#### 3.3.6 `fifo.rs` -- FIFO Cost Basis (NEW: extracted to pure function)

Currently embedded in `internal/service/investment_income_svc.go:RecalculateFIFO` (lines 193-313). The Rust version extracts the FIFO algorithm into a pure function with no DB access.

```rust
pub struct SellTransaction {
    pub id: i64,
    pub asset_name: String,
    pub asset_type: AssetType,
    pub transaction_date: NaiveDate,
    pub quantity: i64,       // 1/10000 units
    pub total_amount: Amount,
    pub fees: Amount,
}

pub struct BuyTransaction {
    pub id: i64,
    pub asset_name: String,
    pub asset_type: AssetType,
    pub transaction_date: NaiveDate,
    pub quantity: i64,
    pub total_amount: Amount,
}

pub struct FIFOResult {
    pub sell_id: i64,
    pub cost_basis: Amount,
    pub computed_gain: Amount,
    pub time_test_exempt: bool,
    pub exempt_amount: Amount,
}

pub fn calculate_fifo(
    sells: &[SellTransaction],
    buys: &[BuyTransaction],
    time_test_years: i32,
    security_exemption_limit: Amount,
) -> Vec<FIFOResult>
```

Algorithm (ported exactly from Go):
1. Group sells by `(asset_name, asset_type)`.
2. Sort group keys deterministically (name, then type) for reproducible results.
3. Track `cumulative_exempt: Amount` across all groups for the year.
4. For each group:
   a. Sort sells chronologically.
   b. Load all buys for this asset group (passed in, pre-filtered by caller).
   c. Track `consumed: HashMap<i64, i64>` (buy_id -> consumed quantity) shared across all sells in this group.
   d. For each sell: FIFO-match against buys. Compute proportional cost basis. Check time test (all matched buys must be before cutoff). Apply exemption limit if configured (> 0).

**Key difference from Go:** The Go version does DB reads inside the loop (`ListBuysForFIFO`). The Rust version takes all buys as input. The service layer (Phase 3) will pre-fetch all buys and pass them in.

**Grouping of buys:** The caller must pass buys pre-grouped or the function will accept a flat slice and internally group. Design decision: accept a flat `&[BuyTransaction]` and group internally, matching Go's behavior of loading buys per-group but doing it upfront.

#### 3.3.7 `annual_base.rs`

Ported from `internal/calc/annual_base.go`.

```rust
pub struct InvoiceForBase {
    pub id: i64,
    pub invoice_type: InvoiceType,
    pub status: InvoiceStatus,
    pub delivery_date: NaiveDate,
    pub issue_date: NaiveDate,
    pub subtotal_amount: Amount,
}

pub struct ExpenseForBase {
    pub id: i64,
    pub issue_date: NaiveDate,
    pub amount: Amount,
    pub vat_amount: Amount,
    pub business_percent: i32,
    pub tax_reviewed: bool,
}

pub struct AnnualBaseResult {
    pub revenue: Amount,
    pub expenses: Amount,
    pub invoice_ids: Vec<i64>,
    pub expense_ids: Vec<i64>,
}

pub fn calculate_annual_totals(
    invoices: &[InvoiceForBase],
    expenses: &[ExpenseForBase],
    year: i32,
) -> AnnualBaseResult
```

Invoice filtering logic:
- Use `delivery_date` if set (non-zero), otherwise `issue_date`
- Must be within Jan 1 - Dec 31 of the year
- Status must be `Sent`, `Paid`, or `Overdue`
- Skip `Proforma` type
- `CreditNote` type: subtract `subtotal_amount` from revenue
- All others: add `subtotal_amount` to revenue

Expense filtering logic:
- Must be within Jan 1 - Dec 31 of the year
- `tax_reviewed` must be true
- `base_amount = amount - vat_amount`
- `business_percent`: if 0, treat as 100
- `expenses += base_amount * business_percent / 100`

#### 3.3.8 `recurring.rs`

Ported from `recurring.go:NextDate()` logic. Simple frequency-based date advancement:

```rust
pub fn next_occurrence(current: NaiveDate, frequency: Frequency) -> NaiveDate
```

- `Weekly`: add 7 days
- `Monthly`: add 1 month (clamp to month end)
- `Quarterly`: add 3 months (clamp to month end)
- `Yearly`: add 1 year

Uses `chrono::Months` for month addition, which correctly handles month-end clamping (Jan 31 + 1 month = Feb 28/29).

#### 3.3.9 Module Re-exports (`calc/mod.rs`)

```rust
pub mod annual_base;
pub mod constants;
pub mod credits;
pub mod fifo;
pub mod income_tax;
pub mod insurance;
pub mod recurring;
pub mod vat;

pub use constants::{get_tax_constants, TaxYearConstants};
pub use credits::{compute_child_benefit, compute_deductions, compute_personal_credits, compute_spouse_credit};
pub use fifo::calculate_fifo;
pub use income_tax::calculate_income_tax;
pub use insurance::{calculate_insurance, resolve_used_expenses};
pub use vat::calculate_vat_return;
```

### 3.4 `zfaktury-testutil` Crate

Test-only crate (listed under `[dev-dependencies]` of consuming crates). Provides builder patterns, golden file helpers, and (in later phases) test DB setup.

#### 3.4.1 Builder Pattern

Each major domain entity gets a builder:

```rust
pub struct ContactBuilder { /* defaults for all fields */ }
impl ContactBuilder {
    pub fn new() -> Self { /* sensible defaults: name="Test Contact", type=Company, ... */ }
    pub fn name(mut self, name: &str) -> Self { ... }
    pub fn ico(mut self, ico: &str) -> Self { ... }
    pub fn contact_type(mut self, ct: ContactType) -> Self { ... }
    pub fn build(self) -> Contact { ... }
}
```

Builders for: `ContactBuilder`, `InvoiceBuilder`, `InvoiceItemBuilder`, `ExpenseBuilder`, `ExpenseItemBuilder`, `VATReturnBuilder`, `IncomeTaxReturnBuilder`, `SecurityTransactionBuilder`, `CapitalIncomeEntryBuilder`.

`InvoiceBuilder` has a convenience method:
```rust
pub fn with_items(mut self, items: Vec<InvoiceItem>) -> Self { ... }
```

All builders produce valid entities with realistic defaults (non-zero IDs, current timestamps, amounts in a reasonable range).

#### 3.4.2 Golden File Helper

```rust
pub fn assert_golden(name: &str, actual: &str) {
    let golden_path = Path::new("tests/golden").join(format!("{name}.golden"));
    if std::env::var("UPDATE_GOLDEN").is_ok() {
        std::fs::create_dir_all(golden_path.parent().unwrap()).unwrap();
        std::fs::write(&golden_path, actual).unwrap();
        return;
    }
    let expected = std::fs::read_to_string(&golden_path)
        .unwrap_or_else(|_| panic!("golden file not found: {golden_path:?}\nRun with UPDATE_GOLDEN=1 to create it"));
    assert_eq!(actual, expected, "golden file mismatch: {golden_path:?}");
}
```

Used for XML generation tests in later phases. Set up here to be available from day one.

#### 3.4.3 Test DB Setup (stub for Phase 2)

```rust
// Phase 1: just the signature. Actual implementation in Phase 2 when rusqlite is available.
// pub fn new_test_db() -> rusqlite::Connection { ... }
```

---

## 4. Testing Strategy

### 4.1 Test Framework

- **`rstest`** for table-driven (parametrized) tests -- replaces Go's `tests := []struct{}{...}` pattern.
- **`proptest`** for property-based testing on Amount arithmetic and tax invariants.
- Standard `#[test]` for unit tests.

### 4.2 Test Vectors Ported from Go

Every Go test case is ported as an `rstest` case. The following tables enumerate the exact ports:

**Amount tests** (from `money_test.go`):
- `TestNewAmount`: 5 cases (zero, whole only, with halere, one haler, negative)
- `TestFromFloat`: 6 cases (zero, whole, with cents, rounding up, rounding down, negative)
- `TestAmount_ToCZK`: 4 cases
- `TestAmount_String`: 6 cases (zero, whole, with halere, single digit, negative, negative fraction)
- `TestAmount_Add`: 4 cases
- `TestAmount_Sub`: 3 cases
- `TestAmount_Multiply`: 6 cases (by zero, one, two, half, 21% VAT, rounding)
- `TestAmount_IsZero`: 3 cases
- `TestAmount_IsNegative`: 3 cases

**VAT tests** (from `vat_test.go`):
- 9 cases: zero, single 21%, single 12%, mixed rates, credit note sign reversal, 0% excluded, expense 21%/100%, expense 12%/50%, expense 0%->100%, mixed invoices+expenses
- Invariant assertions: `total_output = amount_21 + amount_12`, `net = output - input`

**Income tax tests** (from `income_tax_test.go`):
- 10 cases: flat rate within cap, flat rate capped, actual expenses, negative tax base, deductions to 0, rounding, below threshold, above threshold, credits exceeding tax, child benefit negative, prepayments refund, full realistic 2025

**Insurance tests** (from `insurance_test.go` + monthly rounding):
- 7 cases: social above min, health above min, below min (uses min base), zero revenue, expenses > revenue, prepayments > total, no prepayments
- 3 rounding cases: exact /12, not divisible, zero

**Credits tests** (from `credits_test.go`):
- Spouse: 5 cases (below limit, proportional months, ZTP, at limit, above limit)
- Personal: 7 cases (disability 0-3, student 12/6 months, not student)
- Child benefit: 6 cases (1/2/3 children, ZTP, proportional, empty)
- Deductions: 7 cases (below cap, above cap, shared cap, donation 15%, empty, unknown category, negative)

**Expenses tests** (from `expenses_test.go`):
- 5 cases: flat rate within cap, capped 60%, capped 80%, no flat rate, unknown percent

### 4.3 Property-Based Tests (proptest)

```rust
// Amount arithmetic properties
proptest! {
    #[test]
    fn amount_add_commutative(a in any::<i64>(), b in any::<i64>()) {
        let x = Amount::from_halere(a);
        let y = Amount::from_halere(b);
        // Guard against overflow
        if a.checked_add(b).is_some() {
            prop_assert_eq!(x + y, y + x);
        }
    }

    #[test]
    fn amount_add_identity(a in any::<i64>()) {
        let x = Amount::from_halere(a);
        prop_assert_eq!(x + Amount::ZERO, x);
    }

    #[test]
    fn amount_sub_inverse(a in any::<i64>()) {
        let x = Amount::from_halere(a);
        // a - a = 0
        prop_assert_eq!(x - x, Amount::ZERO);
    }

    #[test]
    fn amount_negate_involution(a in any::<i64>()) {
        let x = Amount::from_halere(a);
        prop_assert_eq!(-(-x), x);
    }
}

// Tax invariants
proptest! {
    #[test]
    fn tax_base_non_negative(
        revenue in 0i64..=500_000_000i64,
        expenses in 0i64..=500_000_000i64,
    ) {
        let input = IncomeTaxInput {
            total_revenue: Amount::from_halere(revenue),
            actual_expenses: Amount::from_halere(expenses),
            flat_rate_percent: 0,
            constants: get_tax_constants(2025).unwrap(),
            ..Default::default()
        };
        let result = calculate_income_tax(&input);
        prop_assert!(result.tax_base.halere() >= 0);
        prop_assert!(result.tax_base_rounded.halere() >= 0);
    }

    #[test]
    fn tax_brackets_sum_to_total(revenue in 0i64..=500_000_000i64) {
        let input = IncomeTaxInput {
            total_revenue: Amount::from_halere(revenue),
            constants: get_tax_constants(2025).unwrap(),
            ..Default::default()
        };
        let result = calculate_income_tax(&input);
        prop_assert_eq!(result.total_tax, result.tax_at_15 + result.tax_at_23);
    }

    #[test]
    fn vat_net_invariant(
        output_21 in 0i64..=100_000_000i64,
        output_12 in 0i64..=100_000_000i64,
        input_21 in 0i64..=100_000_000i64,
    ) {
        // Construct and verify the algebraic invariant
        let invoices = vec![VATInvoiceInput {
            invoice_type: InvoiceType::Regular,
            items: vec![
                VATItemInput { base: Amount::from_halere(output_21), vat_amount: Amount::from_halere(output_21 * 21 / 100), vat_rate_percent: 21 },
                VATItemInput { base: Amount::from_halere(output_12), vat_amount: Amount::from_halere(output_12 * 12 / 100), vat_rate_percent: 12 },
            ],
        }];
        let expenses = vec![VATExpenseInput {
            amount: Amount::from_halere(input_21 + input_21 * 21 / 100),
            vat_amount: Amount::from_halere(input_21 * 21 / 100),
            vat_rate_percent: 21,
            business_percent: 100,
        }];
        let r = calculate_vat_return(&invoices, &expenses);
        prop_assert_eq!(r.net_vat, r.total_output_vat - r.total_input_vat);
    }
}
```

### 4.4 Boundary Tests

Specific edge cases tested with regular `#[test]`:
- `Amount::from_halere(i64::MAX)` and `Amount::from_halere(i64::MIN)` -- ensure no panics in Display
- `Amount::new(0, 0)` == `Amount::ZERO`
- Spouse income at exactly 68,000 CZK limit -> returns 0 (not credit)
- Progressive threshold at exact boundary -> only 15%, not 23%
- Insurance monthly rounding: exact multiples of 12 and 100
- FIFO with zero buys for a sell -> `time_test_exempt = false`
- FIFO with cumulative exemption limit exhausted across groups
- Annual base with credit note -> negative revenue contribution

### 4.5 Coverage Requirements

| Crate/Module | Line coverage | Branch coverage |
|---|---|---|
| `zfaktury-domain/src/amount.rs` | 100% | 100% |
| `zfaktury-core/src/calc/` (all modules) | 100% | 100% |
| `zfaktury-config/` | 90%+ | 90%+ |
| `zfaktury-testutil/` | N/A | N/A |
| `zfaktury-domain/` (structs, enums) | N/A (data-only, no branching logic) | N/A |

Coverage measured with `cargo-llvm-cov`.

---

## 5. Dependency Graph

```
zfaktury-testutil (dev-only)
    |
    v
zfaktury-core
    |
    v
zfaktury-domain       zfaktury-config
    |                      |
    v                      v
chrono, thiserror     serde, toml, dirs
```

`zfaktury-config` does NOT depend on `zfaktury-domain`. Config is a boundary concern that deals with TOML, paths, and strings. Domain types are not needed there.

`zfaktury-core` depends on `zfaktury-domain` (for `Amount`, enums, error types).

`zfaktury-testutil` depends on `zfaktury-domain` (for builders).

---

## 6. File Layout

```
rust/
  Cargo.toml                        # workspace root
  crates/
    zfaktury-domain/
      Cargo.toml
      src/
        lib.rs
        amount.rs
        errors.rs
        contact.rs
        invoice.rs
        expense.rs
        recurring.rs
        tax.rs
        tax_credits.rs
        investment.rs
        audit.rs
        ocr.rs
        import.rs
        reminder.rs
        settings.rs
    zfaktury-config/
      Cargo.toml
      src/
        lib.rs
      tests/
        config_test.rs
        fixtures/
          valid.toml
          minimal.toml
          invalid_port.toml
    zfaktury-core/
      Cargo.toml
      src/
        lib.rs
        calc/
          mod.rs
          constants.rs
          vat.rs
          income_tax.rs
          credits.rs
          insurance.rs
          fifo.rs
          annual_base.rs
          recurring.rs
    zfaktury-testutil/
      Cargo.toml
      src/
        lib.rs
        builders.rs
        golden.rs
```

---

## 7. Cargo.toml Dependencies

### zfaktury-domain

```toml
[dependencies]
chrono = { version = "0.4", default-features = false, features = ["std"] }
thiserror = "2"
```

### zfaktury-config

```toml
[dependencies]
serde = { version = "1", features = ["derive"] }
toml = "0.8"
dirs = "6"
thiserror = "2"

[dev-dependencies]
tempfile = "3"
```

### zfaktury-core

```toml
[dependencies]
zfaktury-domain = { path = "../zfaktury-domain" }

[dev-dependencies]
rstest = "0.24"
proptest = "1"
zfaktury-testutil = { path = "../zfaktury-testutil" }
```

### zfaktury-testutil

```toml
[dependencies]
zfaktury-domain = { path = "../zfaktury-domain" }
chrono = { version = "0.4", default-features = false, features = ["std"] }
```

---

## 8. Migration Notes (Go -> Rust)

### 8.1 Go Idiom Translations

| Go pattern | Rust equivalent |
|---|---|
| `type Amount int64` | `struct Amount(i64)` with trait impls |
| `errors.Is(err, domain.ErrNotFound)` | `matches!(err, DomainError::NotFound)` or `if let` |
| `errors.New("...")` sentinel | `DomainError` enum variant |
| `map[int]domain.Amount` | `HashMap<i32, Amount>` |
| `time.Time` | `chrono::NaiveDate` or `NaiveDateTime` |
| `*time.Time` (nullable) | `Option<NaiveDateTime>` |
| `[]byte` | `Vec<u8>` |
| `struct{}` test tables | `rstest` parametrized tests |
| `t.Helper()` + manual assert | `assert_eq!` / `assert!` macros |
| Go interface | Rust trait (defined in Phase 2) |

### 8.2 Behavioral Differences

1. **Recurring date advancement:** Go's `AddDate(0, 1, 0)` on Jan 31 overflows to March. Rust's `chrono::Months` clamps to Feb 28/29. The Rust behavior is the correct one for recurring billing. Documented as intentional.

2. **Amount multiplication:** Go uses `math.Round(float64(a) * factor)`. Rust uses `(self.0 as f64 * factor).round() as i64`. Semantically identical -- IEEE 754 round-to-nearest-even in both cases.

3. **FIFO function extraction:** Go's FIFO is a method on `InvestmentIncomeService` that interleaves DB calls. Rust's FIFO is a pure function. The service layer (Phase 3) will adapt by pre-fetching all data and calling the pure function.

### 8.3 Not Ported in Phase 1

The following Go code is NOT part of Phase 1 and will be ported in later phases:

- Repository interfaces and implementations (Phase 2)
- Service layer orchestration (Phase 3)
- HTTP handlers and DTOs (Phase 4)
- CLI commands (Phase 5)
- XML generation (vatxml, annualtaxxml) (Phase 3)
- PDF generation (Phase 4)
- QR code generation (Phase 4)
- Desktop/Wails integration (Phase 6)
- Frontend (separate track)

---

## 9. Acceptance Criteria

1. **`cargo test -p zfaktury-domain`** -- All domain types compile. Amount arithmetic matches all 35 Go test vectors. Enum string round-trips work.

2. **`cargo test -p zfaktury-config`** -- Config loads from valid TOML. Env override (`ZFAKTURY_DATA_DIR`) works. Missing required config (SMTP port when host set) fails validation. Template creation works in temp directory.

3. **`cargo test -p zfaktury-core`** -- All calc modules pass. Every Go test case ported. proptest suites pass (1000+ iterations). FIFO pure function produces identical results to Go's DB-interleaved version for known test scenarios.

4. **`cargo clippy --workspace`** -- Zero warnings.

5. **`cargo fmt --check`** -- All code formatted.

6. **`cargo-llvm-cov`** -- calc modules at 100% line + branch. amount.rs at 100%.

---

## 10. Review Checklist

Before marking Phase 1 as complete, verify:

- [ ] `Amount` uses `i64` only internally. No `f32`/`f64` stored in Amount.
- [ ] All domain structs have NO `serde` derives (pure types, no `#[derive(Serialize, Deserialize)]`).
- [ ] All enums have `as_str()` and `TryFrom<&str>` for DB string compatibility.
- [ ] Config fails fast on missing required values (SMTP port, OCR key).
- [ ] Calc functions are pure: no `&self`, no `Context`, no filesystem, no network.
- [ ] FIFO is a standalone `pub fn calculate_fifo(...)` -- not tied to any service struct.
- [ ] proptest strategies cover edge cases: 0, negative amounts, `i64::MAX`, `i64::MIN`.
- [ ] All Go calc test vectors ported to rstest tables with exact same expected values.
- [ ] `recursive.rs` date advancement uses month-end clamping (not Go's overflow behavior).
- [ ] No `unwrap()` in library code (only in tests and builders).
- [ ] All public items have doc comments.
- [ ] `cargo doc --no-deps` builds without warnings.

---

## 11. Open Questions

1. **Should `Amount` implement `Serialize`/`Deserialize`?** Current decision: NO in the domain crate. But `zfaktury-core` calc tests need to construct `TaxYearConstants` with `HashMap<i32, Amount>`. This works fine without serde -- we just construct them manually.

2. **Should we add `num-traits` for more numeric operations?** Current decision: NO. The Amount type needs very few operations (add, sub, multiply-by-float, negate). Adding a numeric traits dependency is overkill.

3. **`phf` vs `match` for tax constants lookup?** Current decision: `match`. Only 3 years of data. A match statement is clearer and has zero compile-time cost.

4. **Should `DomainError` carry context strings?** Current decision: NO. Domain errors are sentinel-like. Context is added by the service layer via `thiserror` or `anyhow`. E.g., `DomainError::NotFound` is returned bare; the service wraps it: `Err(ServiceError::InvoiceNotFound { id })`.

---

## 12. Timeline

| Week | Deliverable |
|---|---|
| Week 1 | `zfaktury-domain` complete: Amount, all 34+ structs, enums, errors. All Amount tests passing. |
| Week 1-2 | `zfaktury-config` complete: TOML loading, validation, env override. Tests with temp dirs. |
| Week 2 | `zfaktury-core` calc modules: constants, vat, income_tax, credits, insurance, annual_base, recurring. All Go test vectors ported. |
| Week 2-3 | `zfaktury-core` FIFO module: pure function extraction, comprehensive tests. `zfaktury-testutil` builders. proptest suites. 100% coverage confirmed. |
| Week 3 | Review, clippy cleanup, documentation, final acceptance. |
