# Phase 5: Core Services -- RFC

**Crate:** `zfaktury-core/src/service/`
**Estimated duration:** 2-3 weeks
**Dependencies:** Phase 2 (Domain types), Phase 3 (Repository traits), Phase 4 (calc module)

## 1. Objectives

- Port all 35 services from Go `internal/service/` to Rust
- Wire the complete dependency graph using `Arc<T>` for GPUI shared ownership
- Replicate all business validation, state machine enforcement, and audit logging
- Achieve 90%+ test coverage using mocked repository traits
- Ensure all service types are `Send + Sync` for safe concurrent use

## 2. Architecture

### 2.1 Dependency Inversion

Services depend on repository traits, not concrete implementations. This enables testing with mock repos and swapping storage backends.

```rust
use std::sync::Arc;

pub struct InvoiceService {
    repo: Arc<dyn InvoiceRepo + Send + Sync>,
    contact_svc: Arc<ContactService>,
    sequence_svc: Arc<SequenceService>,
    audit_svc: Arc<AuditService>,
}
```

All repository dependencies are held as `Arc<dyn Trait + Send + Sync>`. Service-to-service dependencies are held as `Arc<ConcreteService>`.

### 2.2 Error Handling Strategy

Every service method returns `Result<T, ServiceError>`. The `ServiceError` type wraps domain errors with context, mirroring Go's `fmt.Errorf("creating invoice: %w", err)` pattern.

```rust
use thiserror::Error;

#[derive(Error, Debug)]
pub enum ServiceError {
    #[error("{context}: {source}")]
    Wrapped {
        context: String,
        source: Box<dyn std::error::Error + Send + Sync>,
    },

    #[error("{0}")]
    Domain(#[from] DomainError),

    #[error("{0}")]
    Repo(#[from] RepoError),
}

impl ServiceError {
    /// Wrap an error with descriptive context.
    /// Pattern: "verb-ing entity" (e.g., "creating invoice", "fetching contact").
    pub fn wrap(context: impl Into<String>, source: impl Into<Box<dyn std::error::Error + Send + Sync>>) -> Self {
        Self::Wrapped {
            context: context.into(),
            source: source.into(),
        }
    }
}
```

Domain-level sentinel errors (from `domain/errors.rs`):

```rust
#[derive(Error, Debug, Clone, PartialEq)]
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

### 2.3 Audit Logging Pattern

Every mutation (create/update/delete) calls `audit_svc.log()`. Audit failures are logged via `tracing::error!` but never propagate to callers -- this matches Go behavior exactly.

```rust
impl AuditService {
    /// Log an audit event. Errors are traced but never returned.
    pub fn log(
        &self,
        entity_type: &str,
        entity_id: i64,
        action: &str,
        old_values: Option<&dyn Serialize>,
        new_values: Option<&dyn Serialize>,
    ) {
        let entry = AuditLogEntry {
            entity_type: entity_type.to_string(),
            entity_id,
            action: action.to_string(),
            old_values: old_values.and_then(|v| serde_json::to_string(v).ok()).unwrap_or_default(),
            new_values: new_values.and_then(|v| serde_json::to_string(v).ok()).unwrap_or_default(),
            ..Default::default()
        };

        if let Err(e) = self.repo.create(&entry) {
            tracing::error!(
                entity_type = entity_type,
                entity_id = entity_id,
                action = action,
                error = %e,
                "failed to create audit log entry"
            );
        }
    }
}
```

### 2.4 Async vs Sync Decision

Services use **synchronous** methods. SQLite access through `rusqlite` is inherently synchronous, and wrapping everything in async would add complexity without benefit. GPUI runs service calls on background threads via `cx.background_executor().spawn()`, so blocking is acceptable.

If the app later needs async (e.g., for HTTP-based repos or S3 backup), individual services can be made async at that point. The trait-based architecture makes this migration straightforward.

## 3. Complete Service Inventory

### 3.1 Foundation Services (Tier 0 -- no service dependencies)

#### 3.1.1 AuditService -- `audit_svc.rs`

**Go source:** `internal/service/audit_svc.go` (64 lines)

**Dependencies:** `Arc<dyn AuditLogRepo>`

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `log` | `(&self, entity_type: &str, entity_id: i64, action: &str, old: Option<&dyn Serialize>, new: Option<&dyn Serialize>)` | Fire-and-forget; errors traced, never returned |
| `list_by_entity` | `(&self, entity_type: &str, entity_id: i64) -> Result<Vec<AuditLogEntry>>` | |
| `list` | `(&self, filter: AuditLogFilter) -> Result<(Vec<AuditLogEntry>, usize)>` | Paginated with total count |

**Key behavior:**
- JSON serialization of old/new values via `serde_json`
- Serialization errors logged but swallowed
- DB write errors logged but swallowed
- This is the only service where errors are intentionally suppressed

#### 3.1.2 TaxCalendarService -- `tax_calendar_svc.rs`

**Go source:** `internal/service/tax_calendar_svc.go` (164 lines)

**Dependencies:** None (stateless)

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `get_deadlines` | `(&self, year: i32) -> Vec<TaxDeadline>` | Pure computation, no Result needed |

**Key behavior:**
- Czech public holidays (Law 245/2000 Sb.) including Easter via Computus algorithm
- Business day shifting: if deadline falls on weekend/holiday, shift to next business day
- Monthly VAT deadlines (25th of following month)
- Annual deadlines: income tax (April 1), social/health insurance (May 2)
- Quarterly tax advance deadlines (March 15, June 15, Sept 15, Dec 15)
- Returns sorted by date

---

### 3.2 Settings & Simple Entity Services (Tier 1 -- depend on AuditService only)

#### 3.2.1 SettingsService -- `settings_svc.rs`

**Go source:** `internal/service/settings_svc.go` (252 lines)

**Dependencies:** `Arc<dyn SettingsRepo>`, `Arc<AuditService>`

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `get_all` | `(&self) -> Result<HashMap<String, String>>` | |
| `get` | `(&self, key: &str) -> Result<String>` | Validates key against known set |
| `set` | `(&self, key: &str, value: &str) -> Result<()>` | Audit logs changed value |
| `set_bulk` | `(&self, settings: HashMap<String, String>) -> Result<()>` | Only audits changed keys |
| `get_pdf_settings` | `(&self) -> Result<PdfSettings>` | Returns struct with defaults applied |
| `save_pdf_settings` | `(&self, settings: PdfSettings) -> Result<()>` | Converts struct to individual keys |

**Validation:**
- Key must be non-empty and exist in the `KNOWN_KEYS` set
- Unknown keys rejected with error (not silently ignored)
- `set_bulk` validates all keys before writing any

**Known keys** (const set):
- Company: `company_name`, `ico`, `dic`, `vat_registered`, `street`, `city`, `zip`, `email`, `phone`
- Bank: `bank_account`, `bank_code`, `iban`, `swift`
- Email: `email_attach_pdf`, `email_attach_isdoc`, `email_subject_template`, `email_body_template`
- Personal: `first_name`, `last_name`, `house_number`
- Office codes: `health_insurance_code`, `financni_urad_code`, `cssz_code`, `c_ufo`, `c_pracufo`, `c_okec`
- PDF: `pdf.logo_path`, `pdf.accent_color`, `pdf.footer_text`, `pdf.show_qr`, `pdf.show_bank_details`, `pdf.font_size`

**PdfSettings defaults:**
- `accent_color`: `"#2563eb"` if empty
- `font_size`: `"normal"` if empty
- `show_qr`: `true` if key absent
- `show_bank_details`: `true` if key absent

#### 3.2.2 ContactService -- `contact_svc.rs`

**Go source:** `internal/service/contact_svc.go` (135 lines)

**Dependencies:** `Arc<dyn ContactRepo>`, `Option<Arc<dyn AresClient>>`, `Arc<AuditService>`

**External trait:**
```rust
#[async_trait]
pub trait AresClient: Send + Sync {
    fn lookup_by_ico(&self, ico: &str) -> Result<Contact>;
}
```

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `create` | `(&self, contact: &mut Contact) -> Result<()>` | Sets defaults for type, validates name |
| `update` | `(&self, contact: &Contact) -> Result<()>` | Fetches existing for audit |
| `delete` | `(&self, id: i64) -> Result<()>` | Soft delete |
| `get_by_id` | `(&self, id: i64) -> Result<Contact>` | |
| `list` | `(&self, filter: ContactFilter) -> Result<(Vec<Contact>, usize)>` | Clamps limit to 1..=100, default 20 |
| `lookup_ares` | `(&self, ico: &str) -> Result<Contact>` | Returns error if ARES client not configured |

**Validation:**
- `name` required
- `type` defaults to `"company"`, must be `"company"` or `"individual"`
- `ico` format validation (8 digits) if provided -- port from Go (not currently validated in Go, but specified in CLAUDE.md)

#### 3.2.3 SequenceService -- `sequence_svc.rs`

**Go source:** `internal/service/sequence_svc.go` (193 lines)

**Dependencies:** `Arc<dyn InvoiceSequenceRepo>`, `Arc<AuditService>`

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `create` | `(&self, seq: &mut InvoiceSequence) -> Result<()>` | Checks prefix+year uniqueness |
| `update` | `(&self, seq: &InvoiceSequence) -> Result<()>` | Prevents lowering next_number below used |
| `delete` | `(&self, id: i64) -> Result<()>` | Blocked if invoices reference sequence |
| `get_by_id` | `(&self, id: i64) -> Result<InvoiceSequence>` | |
| `list` | `(&self) -> Result<Vec<InvoiceSequence>>` | |
| `get_or_create_for_year` | `(&self, prefix: &str, year: i32) -> Result<InvoiceSequence>` | Race-condition safe with retry |
| `format_preview` | `(seq: &InvoiceSequence) -> String` | Static helper, e.g., "FV20260001" |

**Key behavior:**
- `get_or_create_for_year`: tries lookup, creates if `NotFound`, retries lookup on create failure (race condition handling)
- Default `format_pattern`: `"{prefix}{year}{number:04d}"`
- Default `next_number`: 1

#### 3.2.4 CategoryService -- `category_svc.rs`

**Go source:** `internal/service/category_svc.go` (155 lines)

**Dependencies:** `Arc<dyn CategoryRepo>`, `Arc<AuditService>`

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `create` | `(&self, cat: &mut ExpenseCategory) -> Result<()>` | Checks key uniqueness |
| `update` | `(&self, cat: &ExpenseCategory) -> Result<()>` | Checks key uniqueness excluding self |
| `delete` | `(&self, id: i64) -> Result<()>` | Blocks deletion of default categories |
| `get_by_id` | `(&self, id: i64) -> Result<ExpenseCategory>` | |
| `list` | `(&self) -> Result<Vec<ExpenseCategory>>` | |

**Validation:**
- `key`: required, lowercase alphanumeric + underscores only (`^[a-z0-9_]+$`)
- `label_cs`: required (Czech label)
- `label_en`: required (English label)
- `color`: if provided, must be valid hex (`#RGB` or `#RRGGBB`), defaults to `"#6B7280"`
- Default categories (`is_default = true`) cannot be deleted

---

### 3.3 Document Services (Tier 1 -- depend on AuditService, filesystem)

#### 3.3.1 DocumentService -- `document_svc.rs`

**Go source:** `internal/service/document_svc.go` (249 lines)

**Dependencies:** `Arc<dyn DocumentRepo>`, `PathBuf` (data_dir), `Arc<AuditService>`

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `upload` | `(&self, expense_id: i64, filename: &str, content_type: &str, data: &[u8]) -> Result<ExpenseDocument>` | Full validation pipeline |
| `get_by_id` | `(&self, id: i64) -> Result<ExpenseDocument>` | |
| `list_by_expense_id` | `(&self, expense_id: i64) -> Result<Vec<ExpenseDocument>>` | |
| `delete` | `(&self, id: i64) -> Result<()>` | Soft-delete record + best-effort file removal |
| `get_file_path` | `(&self, id: i64) -> Result<(PathBuf, String)>` | Path traversal validation |

**Upload validation pipeline:**
1. Content type allowlist: `image/jpeg`, `image/png`, `application/pdf`, `image/webp`, `image/heic`
2. Filename sanitization: strip path separators, null bytes, limit to 255 chars
3. Per-expense document count limit: max 10
4. Size limit: 20 MB (read one extra byte to detect overflow)
5. Magic byte detection: verify actual content type matches declared type
6. MIME spoofing prevention: `detectByMagicBytes` for PDF, WEBP, HEIC

**Storage path:** `{data_dir}/documents/{expense_id}/{uuid}_{filename}`

**Path traversal protection:** `get_file_path` validates stored path starts with `{data_dir}/documents/`

#### 3.3.2 InvoiceDocumentService -- `invoice_document_svc.rs`

**Go source:** `internal/service/invoice_document_svc.go`

Same pattern as DocumentService but for invoices.

**Storage path:** `{data_dir}/documents/invoices/{invoice_id}/{uuid}_{filename}`

**Methods:** `upload`, `get_by_id`, `list_by_invoice_id`, `delete`, `get_file_path`

#### 3.3.3 TaxDeductionDocumentService -- `tax_deduction_document_svc.rs`

**Go source:** `internal/service/tax_deduction_document_svc.go`

Same upload/storage pattern for tax deduction supporting documents.

**Storage path:** `{data_dir}/tax-documents/deductions/{deduction_id}/{uuid}_{filename}`

**Methods:** `upload`, `get_by_id`, `list_by_deduction_id`, `delete`

#### 3.3.4 InvestmentDocumentService -- `investment_document_svc.rs`

**Go source:** `internal/service/investment_document_svc.go` (231 lines)

**Dependencies:** `Arc<dyn InvestmentDocumentRepo>`, `Arc<dyn CapitalIncomeRepo>`, `Arc<dyn SecurityTransactionRepo>`, `PathBuf`, `Arc<AuditService>`

**Methods:** `upload`, `get_by_id`, `list_by_year`, `delete`, `get_file_path`

**Extra behavior:**
- `upload` validates `platform` against allowlist: `portu`, `zonky`, `trading212`, `revolut`, `other`
- `upload` validates `year` range (2000-2100)
- `delete` cascades: deletes linked capital income entries AND security transactions before deleting document

**Storage path:** `{data_dir}/investment-documents/{year}/{uuid}_{filename}`

---

### 3.4 Core Business Services (Tier 2)

#### 3.4.1 InvoiceService -- `invoice_svc.rs`

**Go source:** `internal/service/invoice_svc.go` (476 lines)

**Dependencies:** `Arc<dyn InvoiceRepo>`, `Arc<ContactService>`, `Arc<SequenceService>`, `Arc<AuditService>`

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `create` | `(&self, invoice: &mut Invoice) -> Result<()>` | Auto-assigns number, calculates totals |
| `update` | `(&self, invoice: &mut Invoice) -> Result<()>` | Rejects paid invoices |
| `delete` | `(&self, id: i64) -> Result<()>` | Rejects paid invoices |
| `get_by_id` | `(&self, id: i64) -> Result<Invoice>` | |
| `get_related_invoices` | `(&self, invoice_id: i64) -> Result<Vec<Invoice>>` | |
| `list` | `(&self, filter: InvoiceFilter) -> Result<(Vec<Invoice>, usize)>` | Clamps limit 1..=100 |
| `mark_as_sent` | `(&self, id: i64) -> Result<()>` | Only from Draft |
| `mark_as_paid` | `(&self, id: i64, amount: Amount, paid_at: NaiveDate) -> Result<()>` | Rejects already-paid/cancelled |
| `settle_proforma` | `(&self, proforma_id: i64) -> Result<Invoice>` | Idempotent; creates bidirectional link |
| `create_credit_note` | `(&self, original_id: i64, items: Option<Vec<InvoiceItem>>, reason: &str) -> Result<Invoice>` | Full or partial credit note |
| `duplicate` | `(&self, id: i64) -> Result<Invoice>` | Copy with reset status/dates |

**Invoice state machine:**
```
Draft --mark_as_sent--> Sent --mark_as_paid--> Paid
Draft -----------------> Cancelled
Sent --overdue_check--> Overdue --mark_as_paid--> Paid
```

**Create flow:**
1. Validate: customer_id required, items non-empty, due_date required
2. Verify customer exists via `contact_svc.get_by_id()`
3. Set defaults: status=Draft, type=Regular, currency=CZK, issue_date=now, delivery_date=issue_date
4. Auto-assign sequence: prefix based on type (FV/ZF/DN), get_or_create_for_year
5. Get next number from repo
6. Set variable_symbol = invoice_number
7. `calculate_totals()` on domain struct
8. Persist via repo
9. Audit log

**Update restrictions:**
- Paid invoices cannot be updated (`ErrPaidInvoice`)
- Preserves status, invoice_number, sequence_id, variable_symbol, type if not set in update

**settle_proforma:**
- Proforma must be type=Proforma and status=Paid
- Idempotent: checks for existing settlement via `FindByRelatedInvoice`
- Creates regular invoice with copied items and bidirectional link
- Calls `self.create()` internally for number assignment

**create_credit_note:**
- Original must be type=Regular and status in {Sent, Paid}
- If items empty: full credit (all items with negated unit_price)
- If items provided: partial credit (provided items with negated unit_price)
- Calls `self.create()` for number assignment from "DN" sequence

#### 3.4.2 ExpenseService -- `expense_svc.rs`

**Go source:** `internal/service/expense_svc.go` (183 lines)

**Dependencies:** `Arc<dyn ExpenseRepo>`, `Arc<AuditService>`

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `create` | `(&self, expense: &mut Expense) -> Result<()>` | |
| `update` | `(&self, expense: &mut Expense) -> Result<()>` | |
| `delete` | `(&self, id: i64) -> Result<()>` | |
| `get_by_id` | `(&self, id: i64) -> Result<Expense>` | |
| `list` | `(&self, filter: ExpenseFilter) -> Result<(Vec<Expense>, usize)>` | |
| `mark_tax_reviewed` | `(&self, ids: Vec<i64>) -> Result<()>` | Max 500, deduplicates |
| `unmark_tax_reviewed` | `(&self, ids: Vec<i64>) -> Result<()>` | Max 500, deduplicates |

**Validation:**
- `description` required
- `amount > 0` or items non-empty
- `issue_date` required
- `business_percent` defaults to 100, must be 0..=100
- `currency_code` defaults to CZK
- If items present: `calculate_totals()` from items
- If no items but VAT rate set: compute VAT from rate

**Bulk operations:**
- `mark_tax_reviewed`/`unmark_tax_reviewed` max 500 IDs
- IDs deduplicated before passing to repo

---

### 3.5 Recurring Services (Tier 3 -- depend on InvoiceService/ExpenseService)

#### 3.5.1 RecurringInvoiceService -- `recurring_invoice_svc.rs`

**Go source:** `internal/service/recurring_invoice_svc.go` (221 lines)

**Dependencies:** `Arc<dyn RecurringInvoiceRepo>`, `Arc<InvoiceService>`, `Arc<AuditService>`

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `create` | `(&self, ri: &mut RecurringInvoice) -> Result<()>` | |
| `update` | `(&self, ri: &RecurringInvoice) -> Result<()>` | |
| `delete` | `(&self, id: i64) -> Result<()>` | |
| `get_by_id` | `(&self, id: i64) -> Result<RecurringInvoice>` | |
| `list` | `(&self) -> Result<Vec<RecurringInvoice>>` | |
| `generate_invoice` | `(&self, id: i64) -> Result<Invoice>` | Single generation |
| `process_due` | `(&self) -> Result<usize>` | Batch: find due, generate, advance, deactivate |

**process_due flow:**
1. Find all active recurring invoices where `next_issue_date <= today`
2. For each:
   - If past `end_date`: deactivate, skip
   - Create invoice from template via `create_invoice_from_template()`
   - Advance `next_issue_date` by frequency
   - If new next date past `end_date`: set `is_active = false`
   - Update recurring invoice
3. Return count of generated invoices

**create_invoice_from_template:**
- Copies: customer_id, currency, exchange_rate, payment method, bank details, notes
- Sets: type=Regular, status=Draft, issue_date=param, due_date=issue_date+14d, delivery_date=issue_date
- Copies items without IDs
- Delegates to `invoice_svc.create()` for number assignment and totals

#### 3.5.2 RecurringExpenseService -- `recurring_expense_svc.rs`

**Go source:** `internal/service/recurring_expense_svc.go`

**Dependencies:** `Arc<dyn RecurringExpenseRepo>`, `Arc<ExpenseService>`, `Arc<AuditService>`

**Methods:** `create`, `update`, `delete`, `get_by_id`, `list` (paginated), `activate`, `deactivate`, `generate_pending`

Same pattern as RecurringInvoiceService but for expenses.

---

### 3.6 Tax Year & Credits Services (Tier 2)

#### 3.6.1 TaxYearSettingsService -- `tax_year_settings_svc.rs`

**Go source:** `internal/service/tax_year_settings_svc.go`

**Dependencies:** `Arc<dyn TaxYearSettingsRepo>`, `Arc<dyn TaxPrepaymentRepo>`

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `get_by_year` | `(&self, year: i32) -> Result<TaxYearSettings>` | |
| `save` | `(&self, settings: &TaxYearSettings) -> Result<()>` | Upsert |
| `get_prepayments` | `(&self, year: i32) -> Result<Vec<TaxPrepayment>>` | |
| `get_prepayment_sums` | `(&self, year: i32) -> Result<(Amount, Amount, Amount)>` | (tax, social, health) |

#### 3.6.2 TaxCreditsService -- `tax_credits_svc.rs`

**Go source:** `internal/service/tax_credits_svc.go` (509 lines)

**Dependencies:** `Arc<dyn TaxSpouseCreditRepo>`, `Arc<dyn TaxChildCreditRepo>`, `Arc<dyn TaxPersonalCreditsRepo>`, `Arc<dyn TaxDeductionRepo>`, `Arc<AuditService>`

**CRUD Methods (18 total):**

*Spouse credit:*
- `upsert_spouse`, `get_spouse`, `delete_spouse`

*Child credits:*
- `create_child`, `update_child`, `delete_child`, `list_children`

*Personal credits:*
- `upsert_personal`, `get_personal`

*Deductions:*
- `create_deduction`, `update_deduction`, `delete_deduction`, `get_deduction`, `list_deductions`

**Computation Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `compute_credits` | `(&self, year: i32) -> Result<(Amount, Amount, Amount)>` | Returns (spouse, disability, student) |
| `compute_child_benefit` | `(&self, year: i32) -> Result<Amount>` | |
| `compute_deductions` | `(&self, year: i32, tax_base: Amount) -> Result<Amount>` | Applies statutory caps, persists allowed amounts |
| `copy_from_year` | `(&self, source: i32, target: i32) -> Result<()>` | Skips entities that already exist in target |

**Validation:**
- Year range: 2000-2100
- Spouse: name required, income 0..100M, months 1-12
- Child: order 1-3, months 1-12
- Personal: student_months 0-12, disability_level 0-3
- Deduction categories: `mortgage`, `life_insurance`, `pension`, `donation`, `union_dues`
- Deduction claimed_amount: 0..100M

**compute_deductions side effects:**
- Fetches deductions from repo
- Calls `calc::compute_deductions()` (pure function)
- Persists `max_amount` and `allowed_amount` back to each deduction via repo update
- Returns total allowed

**copy_from_year:**
- Copies each entity type independently
- Skips if target year already has data for that entity type
- Copied entries have zeroed computed amounts, months_claimed reset to 12

---

### 3.7 Investment Services (Tier 2)

#### 3.7.1 InvestmentIncomeService -- `investment_income_svc.rs`

**Go source:** `internal/service/investment_income_svc.go` (379 lines)

**Dependencies:** `Arc<dyn CapitalIncomeRepo>`, `Arc<dyn SecurityTransactionRepo>`, `Arc<AuditService>`

**Capital Income CRUD:**
- `create_capital_entry`, `update_capital_entry`, `delete_capital_entry`, `get_capital_entry`, `list_capital_entries`
- Auto-computes `net_amount = gross - withheld_tax_cz - withheld_tax_foreign`
- Validates category: `dividend_cz`, `dividend_foreign`, `interest`, `coupon`, `fund_distribution`, `other`

**Security Transaction CRUD:**
- `create_security_transaction`, `update_security_transaction`, `delete_security_transaction`, `get_security_transaction`, `list_security_transactions`
- Validates asset_type: `stock`, `etf`, `bond`, `fund`, `crypto`, `other`
- Validates transaction_type: `buy`, `sell`

**FIFO Calculation:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `recalculate_fifo` | `(&self, year: i32) -> Result<()>` | Full FIFO recompute for all sells in year |

**FIFO algorithm:**
1. Get tax constants for year (time_test_years, security_exemption_limit)
2. List all sells for year
3. Group by (asset_name, asset_type), sort keys deterministically
4. For each group, sort sells by transaction_date
5. Load all buys for group, track consumed quantity per buy ID
6. For each sell:
   - Match to buys FIFO (oldest first)
   - Calculate proportional cost basis from matched buys
   - Check time test: all matched buys must be before `sell_date - time_test_years`
   - Compute gain = sell_total - fees - cost_basis
   - Apply exemption limit (cumulative across all groups in year)
   - Persist via `update_fifo_results()`

**Year Summary:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `get_year_summary` | `(&self, year: i32) -> Result<InvestmentYearSummary>` | Aggregates capital + security income |
| `compute_capital_income_totals` | `(&self, year: i32) -> Result<(Amount, Amount, Amount)>` | (gross, tax, net) |

#### 3.7.2 InvestmentExtractionService -- `investment_extraction_svc.rs`

**Go source:** `internal/service/investment_extraction_svc.go`

**Dependencies:** `Arc<dyn OcrProvider>`, `Arc<InvestmentDocumentService>`, `Arc<dyn CapitalIncomeRepo>`, `Arc<dyn SecurityTransactionRepo>`, `Arc<dyn InvestmentDocumentRepo>`

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `extract_from_document` | `(&self, document_id: i64) -> Result<()>` | OCR -> parse -> create entries |

---

### 3.8 Tax Filing Services (Tier 3+)

All filing services share a common pattern. Filing state machine:
```
Draft --recalculate--> Ready/Draft --generate_xml--> Ready --mark_filed--> Filed
```

**Common validation for all filings:**
- Filed filings cannot be modified, deleted, or recalculated (`ErrFilingAlreadyFiled`)
- Regular filings: duplicate for same period rejected (`ErrFilingAlreadyExists`)
- Corrective/supplementary filings allowed for same period
- Filing type must be `regular`, `corrective`, or `supplementary`

#### 3.8.1 VATReturnService -- `vat_return_svc.rs`

**Go source:** `internal/service/vat_return_svc.go` (386 lines)

**Dependencies:** `Arc<dyn VATReturnRepo>`, `Arc<dyn InvoiceRepo>`, `Arc<dyn ExpenseRepo>`, `Arc<dyn SettingsRepo>`, `Arc<AuditService>`

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `create` | `(&self, vr: &mut VATReturn) -> Result<()>` | Period validation, duplicate check |
| `get_by_id` | `(&self, id: i64) -> Result<VATReturn>` | |
| `list` | `(&self, year: i32) -> Result<Vec<VATReturn>>` | |
| `delete` | `(&self, id: i64) -> Result<()>` | Blocked if filed |
| `recalculate` | `(&self, id: i64) -> Result<VATReturn>` | Core calculation flow |
| `generate_xml` | `(&self, id: i64) -> Result<VATReturn>` | EPO XML via vatxml module |
| `get_xml_data` | `(&self, id: i64) -> Result<Vec<u8>>` | |
| `mark_filed` | `(&self, id: i64) -> Result<VATReturn>` | |

**Recalculate flow:**
1. Fetch VAT return, verify not filed
2. Determine date range from period (monthly or quarterly)
3. Query invoices in period (sent/paid/overdue, excluding proformas)
4. For each invoice: load items, build `VATInvoiceInput` with per-item base/vat/rate
5. Query expenses in period (tax deductible only)
6. Build `VATExpenseInput` with amount/vat/rate/business_percent
7. Call `calc::calculate_vat_return()` (pure function)
8. Map result to entity fields (output/input VAT at 21%/12%, totals, net)
9. Persist updated values
10. Link invoice IDs and expense IDs to VAT return
11. Audit log

**generate_xml flow:**
1. Fetch VAT return
2. Build `TaxpayerInfo` from settings (DIC required, strip "CZ" prefix)
3. Call `vatxml::VATReturnGenerator::generate()`
4. Store XML data in entity, persist
5. Audit log

**Period helpers:**
- `period_date_range(period) -> (NaiveDate, NaiveDate)`: monthly (1st to last day) or quarterly

#### 3.8.2 VATControlStatementService -- `vat_control_svc.rs`

**Go source:** `internal/service/vat_control_svc.go` (439 lines)

**Dependencies:** `Arc<dyn VATControlStatementRepo>`, `Arc<dyn InvoiceRepo>`, `Arc<dyn ExpenseRepo>`, `Arc<dyn ContactRepo>`, `Arc<AuditService>`

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `create` | `(&self, cs: &mut VATControlStatement) -> Result<()>` | Monthly only (month 1-12) |
| `get_by_id` | `(&self, id: i64) -> Result<VATControlStatement>` | |
| `list` | `(&self, year: i32) -> Result<Vec<VATControlStatement>>` | |
| `delete` | `(&self, id: i64) -> Result<()>` | |
| `get_lines` | `(&self, id: i64) -> Result<Vec<VATControlStatementLine>>` | |
| `recalculate` | `(&self, id: i64) -> Result<()>` | |
| `generate_xml` | `(&self, id: i64, dic: &str) -> Result<Vec<u8>>` | |
| `mark_filed` | `(&self, id: i64) -> Result<()>` | |

**Recalculate -- Line classification (Czech KH DPH rules):**

*Output (invoices):*
- **A4**: Individual line per rate for invoices above threshold (10,000 CZK). Requires partner DIC, document number, DPPD.
- **A5**: Aggregated by rate for invoices at/below threshold.
- Only CZ DIC partners included. Non-CZ partners skipped.

*Input (expenses):*
- **B2**: Individual line for expenses above threshold. Requires vendor DIC.
- **B3**: Aggregated by rate for expenses at/below threshold.
- Only expenses with CZ DIC vendor included.

**Threshold:** `domain::CONTROL_STATEMENT_THRESHOLD` (10,000 CZK = 1,000,000 halere)

**Recalculate flow:**
1. Delete old lines
2. Build invoice lines (A4/A5) from delivery_date in month
3. Build expense lines (B2/B3) from issue_date in month
4. Create new lines
5. Update status to Ready

#### 3.8.3 VIESSummaryService -- `vies_svc.rs`

**Go source:** `internal/service/vies_svc.go`

**Dependencies:** `Arc<dyn VIESSummaryRepo>`, `Arc<dyn InvoiceRepo>`, `Arc<dyn ContactRepo>`, `Arc<AuditService>`

**Methods:** `create`, `get_by_id`, `get_lines`, `list`, `delete`, `recalculate`, `generate_xml`, `mark_filed`

**Recalculate:**
- Find EU partner invoices (quarterly period)
- Group by partner DIC
- Sum amounts per partner
- Create lines

#### 3.8.4 IncomeTaxReturnService -- `income_tax_return_svc.rs`

**Go source:** `internal/service/income_tax_return_svc.go` (356 lines)

**Dependencies:** `Arc<dyn IncomeTaxReturnRepo>`, `Arc<dyn InvoiceRepo>`, `Arc<dyn ExpenseRepo>`, `Arc<dyn SettingsRepo>`, `Arc<dyn TaxYearSettingsRepo>`, `Arc<dyn TaxPrepaymentRepo>`, `Arc<TaxCreditsService>`, `Option<Arc<InvestmentIncomeService>>`, `Arc<AuditService>`

**Circular dependency:** `InvestmentIncomeService` is set via a setter method after construction, not in constructor. In Rust, use `RwLock<Option<Arc<InvestmentIncomeService>>>` or `OnceLock`.

```rust
pub struct IncomeTaxReturnService {
    // ...
    investment_svc: RwLock<Option<Arc<InvestmentIncomeService>>>,
    // ...
}

impl IncomeTaxReturnService {
    pub fn set_investment_service(&self, svc: Arc<InvestmentIncomeService>) {
        *self.investment_svc.write().unwrap() = Some(svc);
    }
}
```

**Methods:** `create`, `get_by_id`, `list`, `delete`, `recalculate`, `generate_xml`, `get_xml_data`, `mark_filed`

**Recalculate flow (most complex service method):**
1. Calculate annual base from invoices/expenses via `calculate_annual_base()`
2. Read flat_rate_percent from tax year settings
3. Get tax constants for year
4. If investment service set: get year summary (capital + security income)
5. Compute credits: spouse, disability, student via `tax_credits_svc.compute_credits()`
6. Compute child benefit via `tax_credits_svc.compute_child_benefit()`
7. Compute raw tax base for deduction cap calculation
8. Compute deductions via `tax_credits_svc.compute_deductions()`
9. Sum tax prepayments
10. Call `calc::calculate_income_tax()` with all inputs
11. Map all result fields back to entity (20+ fields)
12. Persist, link invoices/expenses, audit log

#### 3.8.5 SocialInsuranceService -- `social_insurance_svc.rs`

**Go source:** `internal/service/social_insurance_svc.go` (294 lines)

**Dependencies:** `Arc<dyn SocialInsuranceOverviewRepo>`, `Arc<dyn InvoiceRepo>`, `Arc<dyn ExpenseRepo>`, `Arc<dyn SettingsRepo>`, `Arc<dyn TaxYearSettingsRepo>`, `Arc<dyn TaxPrepaymentRepo>`, `Arc<AuditService>`

**Methods:** `create`, `get_by_id`, `list`, `delete`, `recalculate`, `generate_xml`, `get_xml_data`, `mark_filed`

**Recalculate flow:**
1. Calculate annual base
2. Get flat_rate_percent from tax year settings
3. Get tax constants
4. Resolve used expenses (flat-rate vs actual via `calc::resolve_used_expenses()`)
5. Sum social prepayments
6. Call `calc::calculate_insurance()` with social rate from constants
7. Map result: tax_base, assessment_base, min_assessment_base, final_assessment_base, total_insurance, difference, new_monthly_prepay
8. Persist, audit log

#### 3.8.6 HealthInsuranceService -- `health_insurance_svc.rs`

**Go source:** `internal/service/health_insurance_svc.go`

Same structure as SocialInsuranceService but with health insurance rate (13.5%) and health prepayments.

---

### 3.9 Utility Services (Tier 3+)

#### 3.9.1 OCRService -- `ocr_svc.rs`

**Go source:** `internal/service/ocr_svc.go` (66 lines)

**Dependencies:** `Arc<dyn OcrProvider>`, `Arc<DocumentService>`

**External trait:**
```rust
pub trait OcrProvider: Send + Sync {
    fn process_image(&self, data: &[u8], content_type: &str) -> Result<OcrResult>;
}
```

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `process_document` | `(&self, document_id: i64) -> Result<OcrResult>` | Validates content type, reads file, calls provider |

**Supported content types:** `image/jpeg`, `image/png`, `application/pdf`

#### 3.9.2 ImportService -- `import_svc.rs`

**Go source:** `internal/service/import_svc.go` (73 lines)

**Dependencies:** `Arc<ExpenseService>`, `Arc<DocumentService>`, `Option<Arc<OCRService>>`

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `import_from_document` | `(&self, filename: &str, content_type: &str, data: &[u8]) -> Result<ImportResult>` | Creates skeleton expense + uploads doc + optional OCR |

**Flow:**
1. Create skeleton expense (description=filename, amount=1 CZK minimum, date=today)
2. Upload document linked to new expense
3. On upload failure: rollback (delete skeleton expense)
4. Run OCR if configured (non-fatal on failure, logged as warning)
5. Return `ImportResult { expense, document, ocr: Option<OcrResult> }`

#### 3.9.3 TaxDocumentExtractionService -- `tax_document_extraction_svc.rs`

**Go source:** `internal/service/tax_document_extraction_svc.go`

**Dependencies:** `Arc<dyn OcrProvider>`, `Arc<TaxDeductionDocumentService>`, `Arc<dyn TaxDeductionRepo>`, `Arc<dyn TaxDeductionDocumentRepo>`

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `extract_from_document` | `(&self, document_id: i64) -> Result<()>` | OCR -> parse -> update extracted_amount + confidence |

#### 3.9.4 OverdueService -- `overdue_svc.rs`

**Go source:** `internal/service/overdue_svc.go` (84 lines)

**Dependencies:** Uses local trait definitions (not the full repo interfaces):

```rust
trait OverdueInvoiceRepo: Send + Sync {
    fn list_overdue_candidate_ids(&self, before_date: NaiveDateTime) -> Result<Vec<i64>>;
    fn update_status(&self, id: i64, status: &str) -> Result<()>;
}

trait StatusHistoryRepo: Send + Sync {
    fn create(&self, change: &InvoiceStatusChange) -> Result<()>;
    fn list_by_invoice_id(&self, invoice_id: i64) -> Result<Vec<InvoiceStatusChange>>;
}
```

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `check_overdue` | `(&self) -> Result<usize>` | Batch update sent->overdue for past-due invoices |
| `get_history` | `(&self, invoice_id: i64) -> Result<Vec<InvoiceStatusChange>>` | |

**check_overdue:**
- Finds sent invoices where due_date < now
- Updates each to status=Overdue
- Records status change in history table
- Errors per-invoice are logged but don't stop the batch
- Returns count of successfully updated invoices

#### 3.9.5 ReminderService -- `reminder_svc.rs`

**Go source:** `internal/service/reminder_svc.go` (215 lines)

**Dependencies:** Uses local trait definitions for testability:
- `reminderRepository` (create, list, count)
- `reminderInvoiceRepo` (get_by_id)
- `reminderEmailSender` (send)
- `reminderSettingsReader` (get)

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `send_reminder` | `(&self, invoice_id: i64) -> Result<PaymentReminder>` | Validates, sends email, records |
| `get_reminders` | `(&self, invoice_id: i64) -> Result<Vec<PaymentReminder>>` | |

**send_reminder flow:**
1. Get invoice, verify overdue (status=Overdue or status=Sent with past due_date)
2. Verify customer has email address
3. Determine reminder level (1-3, capped at 3)
4. Calculate days overdue
5. Read company_name from settings
6. Build template data, generate email (subject + HTML + text)
7. Send email
8. Record reminder in DB
9. Return reminder

**Email formatting helpers:**
- `format_amount_czk`: Czech currency format with thousands separator ("12 345,67 Kc")
- `format_date_czech`: Czech date format ("10. 3. 2026")
- `format_bank_account`: "account/code" format

#### 3.9.6 ReportService -- `report_svc.rs`

**Go source:** `internal/service/report_svc.go` (116 lines)

**Dependencies:** `Arc<dyn ReportRepo>`

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `revenue_report` | `(&self, year: i32) -> Result<RevenueReport>` | Monthly + quarterly + yearly total |
| `expense_report` | `(&self, year: i32) -> Result<ExpenseReport>` | Monthly + quarterly + by category |
| `top_customers` | `(&self, year: i32) -> Result<Vec<CustomerRevenue>>` | Top 10 |
| `profit_loss` | `(&self, year: i32) -> Result<ProfitLossReport>` | Monthly revenue vs expenses |

Pure delegation to report repo with error wrapping. No business logic.

#### 3.9.7 DashboardService -- `dashboard_svc.rs`

**Go source:** `internal/service/dashboard_svc.go` (95 lines)

**Dependencies:** `Arc<dyn DashboardRepo>`

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `get_dashboard` | `(&self) -> Result<DashboardData>` | Aggregates all dashboard metrics |

**DashboardData:**
- Revenue/expenses current month
- Unpaid invoice count + total
- Overdue invoice count + total
- Monthly revenue/expenses for current year (12 months)
- 5 most recent invoices and expenses

#### 3.9.8 FakturoidImportService -- `fakturoid_import_svc.rs`

**Go source:** `internal/service/fakturoid_import_svc.go` (639 lines)

**Dependencies:** `Arc<dyn FakturoidImportLogRepo>`, `Arc<dyn ContactRepo>`, `Arc<dyn InvoiceRepo>`, `Arc<dyn ExpenseRepo>`, `Arc<ContactService>`, `Arc<InvoiceService>`, `Arc<ExpenseService>`, `Arc<DocumentService>`, `Arc<InvoiceDocumentService>`

**External trait:**
```rust
pub trait FakturoidClient: Send + Sync {
    fn list_subjects(&self) -> Result<Vec<fakturoid::Subject>>;
    fn list_invoices(&self) -> Result<Vec<fakturoid::Invoice>>;
    fn list_expenses(&self) -> Result<Vec<fakturoid::Expense>>;
    fn download_attachment(&self, url: &str) -> Result<(Vec<u8>, String)>;
}
```

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `import_all` | `(&self, client: &dyn FakturoidClient, download_attachments: bool) -> Result<FakturoidImportResult>` | Full import pipeline |

**Import pipeline:**
1. Fetch all subjects, invoices, expenses from Fakturoid API
2. Build preview (detect duplicates by import log, ICO, name, invoice number, expense number)
3. Import contacts first (invoices depend on them)
4. Build subject_map: fakturoid_id -> local_id
5. Import invoices (resolve customer_id via subject_map)
6. Import expenses (resolve vendor_id via subject_map)
7. For new entities: download attachments with rate limiting (700ms delay)
8. Log each import in fakturoid_import_log table

**Duplicate detection:**
- Contacts: check import log, then ICO, then exact name match
- Invoices: check import log, then invoice number
- Expenses: check import log, then expense_number + vendor + date

**Entity mapping:**
- `map_subject_to_contact`: name, ICO, DIC, address, bank account (parse "cislo/kod" format)
- `map_fakturoid_invoice`: number, variable symbol, currency, items, status mapping, date parsing
- `map_fakturoid_expense`: amount, items, description fallback chain

#### 3.9.9 BackupService -- `backup_svc.rs`

**Go source:** `internal/service/backup_svc.go` (482 lines)

**Dependencies:** `Arc<dyn BackupHistoryRepo>`, DB connection, `BackupConfig`, `PathBuf` (data_dir), `Arc<dyn BackupStorage>`

**Concurrency guard:** Uses `AtomicBool` to prevent concurrent backups.

```rust
pub struct BackupService {
    repo: Arc<dyn BackupHistoryRepo + Send + Sync>,
    db_path: PathBuf,
    config: BackupConfig,
    data_dir: PathBuf,
    storage: Arc<dyn BackupStorage + Send + Sync>,
    is_s3: bool,
    running: AtomicBool,
}
```

**BackupStorage trait:**
```rust
pub trait BackupStorage: Send + Sync {
    fn upload(&self, local_path: &Path, filename: &str) -> Result<()>;
    fn download(&self, filename: &str) -> Result<(Box<dyn Read>, i64)>;
    fn delete(&self, filename: &str) -> Result<()>;
}
```

**Methods:**
| Method | Signature | Notes |
|--------|-----------|-------|
| `create_backup` | `(&self, trigger: &str) -> Result<BackupRecord>` | Full backup pipeline |
| `list_backups` | `(&self) -> Result<Vec<BackupRecord>>` | |
| `get_backup` | `(&self, id: i64) -> Result<BackupRecord>` | |
| `delete_backup` | `(&self, id: i64) -> Result<()>` | Removes archive + record |
| `get_backup_reader` | `(&self, id: i64) -> Result<(impl Read, i64, String)>` | For download |
| `get_status` | `(&self) -> Result<BackupStatus>` | |
| `is_running` | `(&self) -> bool` | |

**create_backup flow:**
1. `compare_and_swap` running flag (prevents concurrent backups)
2. Create initial DB record (status=Running)
3. `VACUUM INTO` temp file (SQLite full copy)
4. Create tar.gz archive containing:
   - `database.db` (vacuumed copy)
   - `documents/` directory (if exists)
   - `tax-documents/` directory (if exists)
   - `backup-meta.json` (app version, migration version, timestamps, file count, DB size)
5. Upload to storage backend
6. For S3: remove local temp archive
7. Update record (status=Completed, size, file_count, duration)
8. Apply retention policy (delete oldest beyond `retention_count`)
9. On failure: update record with status=Failed and error message

**Implementations:**
- `BackupStorageLocal` -- `backup_storage_local.rs`: local filesystem operations
- `BackupStorageS3` -- `backup_storage_s3.rs`: S3-compatible storage using `rust-s3` crate

---

### 3.10 Shared Helper: AnnualTaxBase

**Go source:** `internal/service/annual_tax_base.go` (90 lines)

Not a service but a shared function used by IncomeTaxReturnService, SocialInsuranceService, and HealthInsuranceService.

```rust
pub struct AnnualTaxBase {
    pub revenue: Amount,
    pub expenses: Amount,
    pub invoice_ids: Vec<i64>,
    pub expense_ids: Vec<i64>,
}

/// Calculate annual revenue and expenses for tax purposes.
/// Revenue: sum of SubtotalAmount from invoices where status IN (sent, paid, overdue),
/// delivery_date in year, type is regular or credit_note.
/// Expenses: sum of (amount - vat_amount) * business_percent/100 from tax-reviewed expenses.
pub fn calculate_annual_base(
    invoice_repo: &dyn InvoiceRepo,
    expense_repo: &dyn ExpenseRepo,
    year: i32,
) -> Result<AnnualTaxBase> {
    // ...
}
```

## 4. Dependency Wiring

### 4.1 Construction Order

The dependency graph is a DAG. Construction follows topological order:

```
Level 0: All repositories (from DB connection)
Level 1: AuditService, TaxCalendarService
Level 2: ContactService, SequenceService, CategoryService, SettingsService
         TaxYearSettingsService, TaxCreditsService
         DocumentService, InvoiceDocumentService, TaxDeductionDocumentService
Level 3: InvoiceService (needs ContactService, SequenceService)
         ExpenseService
         InvestmentIncomeService
         InvestmentDocumentService
Level 4: RecurringInvoiceService (needs InvoiceService)
         RecurringExpenseService (needs ExpenseService)
         OCRService (needs DocumentService)
Level 5: ImportService (needs ExpenseService, DocumentService, OCRService)
         OverdueService, ReminderService
         VATReturnService, VATControlStatementService, VIESSummaryService
Level 6: IncomeTaxReturnService (needs TaxCreditsService)
         -> then call set_investment_service(InvestmentIncomeService)
         SocialInsuranceService, HealthInsuranceService
Level 7: InvestmentExtractionService, TaxDocumentExtractionService
         ReportService, DashboardService
         BackupService
Level 8: FakturoidImportService (needs ContactService, InvoiceService,
         ExpenseService, DocumentService, InvoiceDocumentService)
```

### 4.2 App State Container

In the GPUI application, all services are held in a shared container:

```rust
pub struct AppServices {
    // Foundation
    pub audit: Arc<AuditService>,
    pub tax_calendar: Arc<TaxCalendarService>,

    // Settings & entities
    pub settings: Arc<SettingsService>,
    pub contacts: Arc<ContactService>,
    pub sequences: Arc<SequenceService>,
    pub categories: Arc<CategoryService>,

    // Documents
    pub documents: Arc<DocumentService>,
    pub invoice_documents: Arc<InvoiceDocumentService>,
    pub tax_deduction_documents: Arc<TaxDeductionDocumentService>,
    pub investment_documents: Arc<InvestmentDocumentService>,

    // Core business
    pub invoices: Arc<InvoiceService>,
    pub expenses: Arc<ExpenseService>,

    // Recurring
    pub recurring_invoices: Arc<RecurringInvoiceService>,
    pub recurring_expenses: Arc<RecurringExpenseService>,

    // Tax
    pub tax_year_settings: Arc<TaxYearSettingsService>,
    pub tax_credits: Arc<TaxCreditsService>,
    pub vat_returns: Arc<VATReturnService>,
    pub vat_control: Arc<VATControlStatementService>,
    pub vies: Arc<VIESSummaryService>,
    pub income_tax: Arc<IncomeTaxReturnService>,
    pub social_insurance: Arc<SocialInsuranceService>,
    pub health_insurance: Arc<HealthInsuranceService>,

    // Investments
    pub investment_income: Arc<InvestmentIncomeService>,
    pub investment_extraction: Option<Arc<InvestmentExtractionService>>,

    // Utility
    pub ocr: Option<Arc<OCRService>>,
    pub import: Arc<ImportService>,
    pub overdue: Arc<OverdueService>,
    pub reminders: Arc<ReminderService>,
    pub reports: Arc<ReportService>,
    pub dashboard: Arc<DashboardService>,
    pub backup: Arc<BackupService>,
    pub fakturoid_import: Arc<FakturoidImportService>,

    // Extraction
    pub tax_doc_extraction: Option<Arc<TaxDocumentExtractionService>>,
}

impl AppServices {
    pub fn new(db: &Connection, config: &AppConfig) -> Result<Self> {
        // Construct repos...
        // Construct services in dependency order...
        // Wire circular deps via setters...
    }
}
```

**Optional services:** `ocr`, `investment_extraction`, `tax_doc_extraction` are `Option<Arc<T>>` because they depend on optional external providers (OCR API key).

## 5. Testing Strategy

### 5.1 Mock Framework

Use `mockall` crate for generating mock implementations of repository traits:

```rust
use mockall::automock;

#[automock]
pub trait InvoiceRepo: Send + Sync {
    fn create(&self, invoice: &mut Invoice) -> Result<()>;
    fn update(&self, invoice: &Invoice) -> Result<()>;
    fn delete(&self, id: i64) -> Result<()>;
    fn get_by_id(&self, id: i64) -> Result<Invoice>;
    fn list(&self, filter: InvoiceFilter) -> Result<(Vec<Invoice>, usize)>;
    // ...
}
```

### 5.2 Test Helper: Service Builder

Each service gets a builder for tests that pre-configures mocks:

```rust
#[cfg(test)]
mod tests {
    use super::*;

    fn mock_audit() -> Arc<AuditService> {
        let mut repo = MockAuditLogRepo::new();
        repo.expect_create().returning(|_| Ok(()));
        Arc::new(AuditService::new(Arc::new(repo)))
    }

    fn test_invoice_service() -> (InvoiceService, MockInvoiceRepo) {
        let repo = MockInvoiceRepo::new();
        let contact_svc = Arc::new(/* mock contact service */);
        let sequence_svc = Arc::new(/* mock sequence service */);
        let audit = mock_audit();
        let svc = InvoiceService::new(
            Arc::new(repo.clone()),
            contact_svc,
            sequence_svc,
            audit,
        );
        (svc, repo)
    }
}
```

### 5.3 Required Test Scenarios Per Service

**For every service, test at minimum:**

1. **Happy path CRUD** -- Create, read, update, delete succeed with valid input
2. **Validation failures** -- Each required field missing, invalid values, out-of-range
3. **State machine enforcement** -- Illegal transitions (e.g., edit paid invoice, delete filed return)
4. **Business rules** -- Domain-specific logic (e.g., sequence uniqueness, duplicate filing rejection)
5. **Error wrapping** -- Verify error messages contain context ("creating invoice: ...")
6. **Audit logging** -- Verify `audit_svc.log()` called on every mutation with correct entity_type/action

**Service-specific critical tests:**

| Service | Critical Test Scenarios |
|---------|------------------------|
| InvoiceService | create calculates_totals; create auto-assigns number; update rejects paid; settle_proforma idempotent; credit_note negates prices |
| ExpenseService | create with items recalculates; VAT computed from rate when no items; bulk mark max 500 |
| SequenceService | get_or_create race condition; update rejects lowering next_number |
| VATReturnService | recalculate filters by delivery_date and status; excludes proformas |
| VATControlStatementService | A4 vs A5 threshold classification; B2 vs B3; CZ DIC filter |
| IncomeTaxReturnService | full recalculate pipeline; circular dep with InvestmentIncomeService |
| InvestmentIncomeService | FIFO matching; time test exemption; exemption limit cap |
| BackupService | concurrent backup prevention via AtomicBool; retention policy |
| FakturoidImportService | duplicate detection by ICO/name/number; subject_map resolution |
| CategoryService | key format validation; default category protection |
| DocumentService | MIME spoofing detection; path traversal prevention; size limit |

### 5.4 Coverage Requirements

| Scope | Target |
|-------|--------|
| All services combined | 90%+ line coverage |
| Business validation paths | 100% |
| State transition logic | 100% |
| Error paths | 90%+ |
| Audit logging calls | 100% of mutations verified |

## 6. File Layout

```
zfaktury-core/src/service/
    mod.rs                          -- Module declarations, re-exports, AppServices container
    error.rs                        -- ServiceError type
    audit_svc.rs                    -- AuditService
    settings_svc.rs                 -- SettingsService + PdfSettings + KNOWN_KEYS
    contact_svc.rs                  -- ContactService + AresClient trait
    sequence_svc.rs                 -- SequenceService + format_preview()
    category_svc.rs                 -- CategoryService
    invoice_svc.rs                  -- InvoiceService
    expense_svc.rs                  -- ExpenseService + deduplicate_ids()
    document_svc.rs                 -- DocumentService + upload validation
    invoice_document_svc.rs         -- InvoiceDocumentService
    tax_deduction_document_svc.rs   -- TaxDeductionDocumentService
    investment_document_svc.rs      -- InvestmentDocumentService
    recurring_invoice_svc.rs        -- RecurringInvoiceService
    recurring_expense_svc.rs        -- RecurringExpenseService
    tax_year_settings_svc.rs        -- TaxYearSettingsService
    tax_credits_svc.rs              -- TaxCreditsService (18 CRUD + 4 compute methods)
    tax_calendar_svc.rs             -- TaxCalendarService + Czech holidays
    vat_return_svc.rs               -- VATReturnService + period_date_range()
    vat_control_svc.rs              -- VATControlStatementService
    vies_svc.rs                     -- VIESSummaryService
    income_tax_return_svc.rs        -- IncomeTaxReturnService
    social_insurance_svc.rs         -- SocialInsuranceService
    health_insurance_svc.rs         -- HealthInsuranceService
    investment_income_svc.rs        -- InvestmentIncomeService + FIFO
    investment_extraction_svc.rs    -- InvestmentExtractionService
    tax_document_extraction_svc.rs  -- TaxDocumentExtractionService
    ocr_svc.rs                      -- OCRService + OcrProvider trait
    import_svc.rs                   -- ImportService
    overdue_svc.rs                  -- OverdueService
    reminder_svc.rs                 -- ReminderService + formatting helpers
    report_svc.rs                   -- ReportService
    dashboard_svc.rs                -- DashboardService
    backup_svc.rs                   -- BackupService + BackupStorage trait
    backup_storage_local.rs         -- BackupStorageLocal
    backup_storage_s3.rs            -- BackupStorageS3
    fakturoid_import_svc.rs         -- FakturoidImportService + FakturoidClient trait + mappers
    annual_tax_base.rs              -- calculate_annual_base() shared helper
```

## 7. Implementation Plan

### 7.1 Week 1: Foundation + Core Entities

**Day 1-2:** Error types, AuditService, SettingsService, TaxCalendarService
**Day 3-4:** ContactService, SequenceService, CategoryService
**Day 5:** DocumentService (all 4 variants: expense, invoice, tax deduction, investment)

### 7.2 Week 2: Business Services + Tax

**Day 1-2:** InvoiceService (largest single service), ExpenseService
**Day 3:** RecurringInvoiceService, RecurringExpenseService
**Day 4:** TaxYearSettingsService, TaxCreditsService
**Day 5:** VATReturnService, VATControlStatementService, VIESSummaryService

### 7.3 Week 3: Complex Tax + Utility + Wiring

**Day 1:** IncomeTaxReturnService, SocialInsuranceService, HealthInsuranceService
**Day 2:** InvestmentIncomeService (FIFO), InvestmentExtractionService
**Day 3:** OverdueService, ReminderService, ReportService, DashboardService
**Day 4:** OCRService, ImportService, BackupService (+ storage implementations), FakturoidImportService
**Day 5:** AppServices container, dependency wiring, integration smoke tests

### 7.4 Parallelization Strategy

Services with no cross-dependencies can be implemented in parallel by separate agents:

**Parallel batch 1** (no service-to-service deps):
- AuditService, SettingsService, TaxCalendarService, CategoryService

**Parallel batch 2** (depend on AuditService only):
- ContactService, SequenceService, all 4 DocumentServices

**Parallel batch 3** (depend on batch 2):
- InvoiceService, ExpenseService, TaxYearSettingsService, TaxCreditsService

**Parallel batch 4** (depend on batch 3):
- RecurringInvoice/ExpenseService, all filing services, InvestmentIncomeService

**Sequential** (complex deps):
- IncomeTaxReturnService (depends on TaxCreditsService + InvestmentIncomeService)
- FakturoidImportService (depends on Contact/Invoice/Expense/Document services)
- AppServices wiring (depends on all)

## 8. Acceptance Criteria

1. All 35 services implemented with complete method signatures matching Go source
2. Dependency wiring compiles successfully with all `Arc<T>` types
3. Every service method wraps errors with descriptive context (pattern: "verb-ing entity: {source}")
4. Audit logging called on every create/update/delete mutation across all services
5. Business validation matches Go behavior exactly (same error conditions, same defaults)
6. Invoice state machine enforced: Draft->Sent->Paid, Draft->Cancelled, Sent->Overdue->Paid
7. Paid invoices reject edit and delete operations
8. Filed filings reject modification, deletion, and recalculation
9. Duplicate regular filings for same period rejected
10. `calculate_totals()` called on create/update for invoices and expenses
11. All service types are `Send + Sync` (verified by compiler)
12. 90%+ test coverage across all services
13. All tests pass: `cargo test -p zfaktury-core`

## 9. Review Checklist

- [ ] All services use repository traits (not concrete types)
- [ ] Error wrapping follows pattern: "verb-ing entity: {source}"
- [ ] Audit log called on every mutation (create/update/delete)
- [ ] Filed filings cannot be modified (rejected with `FilingAlreadyFiled`)
- [ ] Duplicate filing for same period rejected (rejected with `FilingAlreadyExists`)
- [ ] Invoice state machine: Draft->Sent->Paid, Draft->Cancelled, Sent->Overdue->Paid
- [ ] Paid invoices cannot be edited or deleted (rejected with `PaidInvoice`)
- [ ] `calculate_totals()` called on create/update for invoices and expenses
- [ ] All services are `Send + Sync` (for `Arc` sharing across GPUI threads)
- [ ] List methods clamp limit to 1..=100, offset >= 0
- [ ] Document upload validates: content type, size, MIME detection, path traversal
- [ ] Settings validates keys against known set
- [ ] Category key format validated (lowercase alphanumeric + underscores)
- [ ] Sequence update prevents lowering next_number below used numbers
- [ ] FIFO respects time test and exemption limit
- [ ] BackupService prevents concurrent backups via AtomicBool
- [ ] FakturoidImportService detects duplicates before importing
- [ ] Circular dependency (IncomeTaxReturn <-> InvestmentIncome) handled via setter
- [ ] Optional external services (OCR, ARES, email) handled as `Option<Arc<T>>`
