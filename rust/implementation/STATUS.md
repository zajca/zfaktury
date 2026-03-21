# ZFaktury Rust Rewrite - Implementation Status

**Last updated:** 2026-03-21
**Total tests:** 444 passing
**Crates:** 7 (domain, config, core, db, gen, api, app) + testutil
**Commits:** 14 (from POC through Phase 10)

---

## Phase Status Overview

| Phase | Status | Tests | Key Gaps |
|-------|--------|-------|----------|
| P0: POC | DONE | 0 | - |
| P1: Foundation | DONE | 221 | - |
| P2: Persistence | DONE | 61 | - |
| P3: Generation | DONE | 73 | - |
| P4: External APIs | DONE | 50 | - |
| P5: Services | DONE | 32 | - |
| P6: GPUI App | PARTIAL | 3 | 17 read-only views, no forms/actions work |
| P7: Polish | DONE | 0 | CLI + docs done |
| P8: PDF Generation | DONE | 15 | typst-bake with Czech template |
| P9: View Completion | PARTIAL | 0 | Views exist but all read-only, 8 stubs, 19 StubView routes |
| P10: Quality Gates | PARTIAL | 0 | Clippy fixed, reviews run, but UI not usable |
| P11: UI Infrastructure | DONE | 0 | 9 form components, 4 new routes, 18 services wired, NavigateEvent for all views |
| P12: Invoice CRUD | NOT STARTED | - | Forms, actions, items editor, PDF/ISDOC/email |
| P13: Expense+Contact CRUD | NOT STARTED | - | Forms, ARES lookup, documents, tax review |
| P14: Settings CRUD | NOT STARTED | - | Editable settings, PDF config, backup |
| P15: Recurring Templates | NOT STARTED | - | CRUD for recurring invoices/expenses |
| P16: VAT Management | NOT STARTED | - | Returns, control statements, VIES |
| P17: Tax Filing | NOT STARTED | - | Income tax, social/health insurance, credits |
| P18: Reports+Dashboard | NOT STARTED | - | Charts, CSV export, quick actions |
| P19: Import+UX Polish | NOT STARTED | - | Fakturoid, OCR, toasts, keyboard shortcuts |

---

## COMPLETE - No Further Work Needed

### zfaktury-domain (18 files, 23 tests)
- All 34+ domain types ported from Go `internal/domain/`
- Amount(i64) with all ops (Add, Sub, Neg, multiply, Display, Ord)
- All enums with Display impl
- calculate_totals() on Invoice and Expense
- next_date() on RecurringInvoice/Expense with month-end clamping
- DomainError enum (10 variants)
- NO serde derives -- domain purity maintained

### zfaktury-config (1 file, 14 tests)
- All config structs match Go internal/config/config.go
- ZFAKTURY_DATA_DIR env override
- TOML parsing, path expansion, defaults
- Fail-fast on missing file

### zfaktury-core/calc (9 modules, 178 tests)
- constants.rs: 2024/2025/2026 tax constants
- vat.rs: VAT return calculation (credit notes, business_percent)
- income_tax.rs: Progressive 15%/23%, rounding to 100 CZK
- insurance.rs: Social 29.2%, Health 13.5%, min base enforcement
- credits.rs: Spouse (68k limit, ZTP), personal, child benefit
- deductions.rs: Category caps (mortgage 150k, pension 24k, etc.)
- fifo.rs: Full FIFO algorithm with time test + exemption limit
- annual_base.rs: Year filtering (status, type, delivery_date)
- recurring.rs: next_occurrence with month-end clamping
- proptest strategies for Amount, tax invariants

### zfaktury-core/repository (3 files)
- traits.rs: 33 repository traits matching Go interfaces.go
- types.rs: MonthlyAmount, QuarterlyAmount, CategoryAmount, CustomerRevenue, RecentInvoice, RecentExpense
- All method signatures match Go originals

### zfaktury-db (32 repo implementations + 21 migrations, 61 tests)
- All repos fully implemented with rusqlite (not stubs)
- Connection setup: WAL, foreign_keys ON, busy_timeout 5000
- Date/datetime parsing helpers
- Goose-to-refinery migration bridge
- Soft delete pattern consistently applied
- Parameterized queries (no SQL injection)

### zfaktury-api (6 clients, 50 tests)
- ARES: ICO lookup with 8-digit validation
- CNB: Exchange rates with weekend fallback (5 days)
- Email: lettre with TLS/STARTTLS, multipart MIME, attachments
- OCR: Anthropic + OpenAI providers with Czech prompt
- Fakturoid: OAuth2, pagination, FlexFloat64, SSRF protection
- FIO: Bank transaction stub with trait (intentionally not implemented)

### zfaktury-core/service (31 files, 32 tests)
- All 31 services implemented
- Arc<dyn Repo + Send + Sync> dependency injection
- Audit logging on mutations
- Error wrapping with DomainError

### zfaktury-gen (PDF, XML, ISDOC, QR, CSV)
- PDF: typst-bake with Czech template, Liberation Sans fonts, QR embedding
- XML: DPHDP3, DPHKH1, DPHSHV, DPFDP5, CSSZ, health insurance
- ISDOC 6.0.2 with DocumentType 1/2/4
- QR/SPAYD payment string + PNG generation
- CSV export with UTF-8 BOM, semicolons

---

## PARTIAL - Detailed UI Gap Analysis

The backend (domain, config, core, db, gen, api) is production-ready. The gap is entirely in `zfaktury-app` -- the GPUI desktop UI layer.

### AppServices: 18 Services NOT Wired

`app.rs` currently has 12 services. These 18 are implemented in `zfaktury-core` but NOT instantiated in `AppServices`:

| Missing Service | Required By |
|---|---|
| BackupService | Settings Backup view |
| DocumentService | Expense document upload |
| HealthInsuranceService | Tax health insurance views |
| IncomeTaxReturnService | Tax income views |
| ImportService | Expense OCR import |
| InvestmentDocumentService | Tax investments view |
| InvestmentIncomeService | Tax investments view |
| InvoiceDocumentService | Invoice document attachments |
| OCRService | Expense import, tax document extraction |
| OverdueService | Invoice list (check overdue) |
| ReminderService | Invoice detail (send reminder) |
| SocialInsuranceService | Tax social insurance views |
| TaxCalendarService | Reports (tax calendar tab) |
| TaxCreditsService | Tax credits view |
| TaxDeductionService | Tax credits/deductions view |
| TaxYearSettingsService | Tax overview |
| VATControlStatementService | VAT control statement views |
| VIESSummaryService | VIES summary views |

### Missing Routes in navigation.rs

These routes do NOT exist in the `Route` enum:

| Missing Route | Purpose |
|---|---|
| `ContactNew` | Create new contact |
| `ContactEdit(i64)` | Edit existing contact |
| `InvoiceEdit(i64)` | Edit existing invoice |
| `ExpenseEdit(i64)` | Edit existing expense |

### Components: Only 1 Exists

`zfaktury-app/src/components/` contains only `status_badge.rs`. Missing:

- Text input, number input, text area
- Select/dropdown
- Date picker
- Checkbox/toggle
- Button (with loading/disabled states)
- Confirm dialog (modal)
- Toast notification system
- Invoice items editor
- Contact picker
- Category picker
- Document upload widget
- Bar/doughnut chart

### View Status Detail

#### Read-Only Views (17) -- Display data but NO create/edit/delete actions

| View | File | What Works | What's Missing |
|---|---|---|---|
| Dashboard | `dashboard.rs` | Stats, recent tables | Quick action buttons, charts, click navigation |
| Invoice List | `invoice_list.rs` | Table with badges | Row click, search, filters, pagination, "New" button |
| Invoice Detail | `invoice_detail.rs` | All fields + items | Edit, Delete, Send, Mark Paid, Duplicate, Credit Note, PDF, QR, ISDOC, Email |
| Expense List | `expense_list.rs` | Table | Row click, search, filters, pagination, "New" button |
| Expense Detail | `expense_detail.rs` | Basic fields | Edit, Delete, Tax review, Document upload |
| Contact List | `contact_list.rs` | Table | Row click, search, "New" button |
| Contact Detail | `contact_detail.rs` | All fields | Edit, Delete, Favorite toggle, related invoices |
| Recurring Invoice List | `recurring_invoice_list.rs` | Table | Row click, "New", Generate, Activate/Deactivate |
| Recurring Expense List | `recurring_expense_list.rs` | Table | Row click, "New", Generate |
| VAT Overview | `vat_overview.rs` | Year selector, quarter cards | Real data loading, status, "New" buttons |
| VAT Return Detail | `vat_return_detail.rs` | Period, status, amounts | Recalculate, Generate XML, Mark Filed, Delete |
| Settings Firma | `settings_firma.rs` | Read-only display | Edit mode with save/cancel |
| Settings Email | `settings_email.rs` | Read-only display | Edit mode, test email button |
| Settings Categories | `settings_categories.rs` | List display | CRUD (create, edit, delete), color picker |
| Settings Sequences | `settings_sequences.rs` | List display | CRUD (create, edit, delete) |
| Settings Audit | `settings_audit.rs` | Paginated log | Filters (entity type, action, date range) |
| Reports | `reports.rs` | P&L tab | Revenue, Expenses, Top Customers, Tax Calendar tabs, charts |

#### Stub/Placeholder Views (8) -- Layout only, no data, no functionality

| View | File | What Exists | What's Needed |
|---|---|---|---|
| Invoice Form | `invoice_form.rs` | Title + placeholder sections | Full form with items editor, customer select, validation, save |
| Expense Form | `expense_form.rs` | Title + placeholder sections | Full form with category/vendor select, document upload, save |
| Settings Backup | `settings_backup.rs` | Non-functional button, empty table | Wire BackupService, create/list/download/delete |
| Import Fakturoid | `import_fakturoid.rs` | Title + "no import" card | OAuth input, preview, import with progress, summary |
| Tax Overview | `tax_overview.rs` | Year selector + static cards | Wire services, load data, create/calculate/file buttons |
| Tax Credits | `tax_credits.rs` | Year selector + empty message | Spouse/children/personal credits CRUD, deductions CRUD |
| Tax Prepayments | `tax_prepayments.rs` | Year selector + empty table | Load monthly data, edit per month |
| Tax Investments | `tax_investments.rs` | Year selector + empty summary | Capital income CRUD, securities CRUD, FIFO, documents |

#### StubView Routes (19) -- Fall through to generic "Coming soon" page

```
RecurringInvoiceNew          RecurringInvoiceDetail(id)
RecurringExpenseNew          RecurringExpenseDetail(id)
ExpenseImport                ExpenseReview
VATReturnNew                 VATControlNew
VATControlDetail(id)         VIESNew
VIESDetail(id)               TaxIncomeNew
TaxIncomeDetail(id)          TaxSocialNew
TaxSocialDetail(id)          TaxHealthNew
TaxHealthDetail(id)          SettingsPdf
ContactNew (ROUTE MISSING)
```

### Navigation: No View-Level Navigation

- Only sidebar emits `NavigateEvent`
- List views have hover styling but NO `on_click` handlers on rows
- No "New" buttons on list views
- No "Back" button on detail/form views
- `NavigationState::go_back()` exists but is never called

---

## SvelteKit Feature Parity Reference

The SvelteKit frontend has these capabilities that the Rust GPUI app must match:

### Invoice Actions (SvelteKit)
- Create/edit with items editor (InvoiceItemsEditor component)
- Mark sent, mark paid (with amount/date dialog)
- Duplicate, create credit note (CreditNoteDialog), settle proforma
- Download PDF, QR code, ISDOC XML
- Send via email (SendEmailDialog with attachments)
- Status timeline, payment history
- Search by number/customer, filter by status/type, pagination

### Expense Actions (SvelteKit)
- Create/edit with items, category/vendor select
- Document upload (drag & drop, PDF/JPG/PNG/WebP, 20MB max)
- OCR import with review dialog (OCRReviewDialog)
- Bulk tax review (mark/unmark)
- Search, date range filter, pagination

### Contact Actions (SvelteKit)
- Create/edit with ARES lookup (auto-populate from ICO)
- Favorite toggle, VAT reliability display
- Search by name/ICO/email, pagination

### Tax Management (SvelteKit)
- Income Tax: Create, view detail, recalculate, generate XML (DPFDP5), mark filed
- Social Insurance: Same workflow
- Health Insurance: Same workflow
- Tax Credits: Spouse, children, personal (student/disability) -- full CRUD
- Tax Deductions: CRUD with document upload + OCR extraction
- Tax Prepayments: Monthly display, edit per month
- Investment Income: Capital income CRUD, security transactions CRUD, FIFO calc

### VAT Management (SvelteKit)
- VAT Returns: Create, view detail, recalculate, generate XML, mark filed
- Control Statements: Same workflow
- VIES Summaries: Same workflow

### Reports (SvelteKit)
- Revenue (monthly/quarterly bars), Expenses (monthly + category doughnut)
- Profit & Loss (combined chart), Top Customers, Tax Calendar
- Year selector, CSV export links

### Settings (SvelteKit)
- Company Info: Editable form
- Email: Editable SMTP + test email
- Sequences: CRUD, Categories: CRUD with color
- PDF: Logo upload/delete, accent color, footer, QR/bank toggles, preview
- Backup: Create, history, download/delete
- Audit Log: Filters (entity type, action, date range)

---

## Recommended Next Steps

See `rust/implementation/AGENT-PROMPT.md` for Phases 11-19 with detailed task lists, team structures, and acceptance criteria.

**Execution order:**
1. **Phase 11: UI Infrastructure** -- MUST be first (blocks all other phases)
2. **Phase 12: Invoice CRUD** -- highest business value
3. **Phase 13: Expense+Contact CRUD** -- core business features
4. **Phase 14: Settings CRUD** -- needed for configuration
5. **Phase 15: Recurring Templates** -- automation feature
6. **Phase 16: VAT Management** -- tax compliance
7. **Phase 17: Tax Filing** -- tax compliance
8. **Phase 18: Reports+Dashboard** -- analytics/visibility
9. **Phase 19: Import+UX Polish** -- final polish
