# ZFaktury Implementation Plan

This document maps the phases from `idea.md` to concrete implementation tasks. Each phase must go through an RFC before implementation begins.

## Process

1. **RFC** — Write detailed RFC for the phase (design, tasks, acceptance criteria)
2. **Review** — Review RFC, validate against idea.md and existing code
3. **Implement** — Build according to RFC
4. **Verify** — Run tests, check acceptance criteria, validate existing code if any

Nothing is considered "done" until it passes through this process. Existing code in the repository is a draft/prototype that must be reviewed and validated as part of the relevant RFC.

## Legend

- **Exists** — Code exists in the repo but has NOT been reviewed/validated through RFC
- **RFC** — RFC written, pending implementation
- **Done** — Implemented, tested, and validated through RFC process
- **Missing** — No implementation exists

---

## Phase 1 — Foundation

**RFC:** `docs/rfc/001-foundation.md`
**Goal:** Project setup, SQLite schema, CLI framework, user settings, contact management with ARES integration.

### Project Setup & Infrastructure
- Done: Go module with dependencies (chi, cobra, sqlite, goose, toml, maroto, qrpay)
- Done: Makefile with `dev`, `build`, `test` targets (includes frontend vitest)
- Done: SvelteKit frontend with Tailwind v4, TypeScript, adapter-static
- Done: Dev mode with Vite HMR proxy
- Done: 3-layer architecture (handler -> service -> repository)
- Done: Test infrastructure (testutil with in-memory SQLite, seed factories; ~1500 Go test files, ~57 vitest test files)
- Done: CI pipeline (`.github/workflows/ci.yml` — linting, testing, building)

### SQLite Schema & Database
- Done: SQLite connection with WAL mode, foreign keys, busy timeout
- Done: Goose migration framework with embedded SQL (14 migrations: 001-014)
- Done: Initial schema + migration 002 (schema alignment fixes)
- Done: Amount (int64 halere) consistency validated across all columns
- Exists: audit_log table (in migration 001, but no repository/service writes to it)
- Missing: bank_transactions table (domain struct exists in `internal/domain/bank.go`, no migration)

### CLI Framework
- Exists: Cobra root command + `serve` command + `--config` flag
- Missing: `config init` / `config show` commands
- Missing: `invoice` subcommands (create, list, pdf, send)
- Missing: `expense` subcommands (add, list, scan)
- Missing: `contact` subcommands (add, list, ares)
- Missing: `tax` subcommands (placeholder for Phase 4-5)
- Missing: `backup` subcommands (placeholder for Phase 8)

### User Settings
- Done: Config struct (UserConfig, ServerConfig, SMTPConfig, FIOConfig, OCRConfig)
- Done: TOML config file loading from `~/.zfaktury/config.toml`
- Done: `ZFAKTURY_DATA_DIR` env var override
- Done: Settings key-value table in database
- Done: Settings HTTP handler (GET/PUT /api/v1/settings) with key allowlist validation
- Done: Settings service layer (GetAll, Get, Set, SetBulk)
- Done: Frontend settings page (Identity/Address/Bank/VAT sections)

### Contact Management
- Done: Contact domain struct, repository, service, HTTP handler, frontend list + detail/edit pages
- Done: ARES HTTP client implementation (ICO validation, response parsing, LimitReader, error handling)
- Done: ARES lookup wired end-to-end (enter ICO -> auto-fill contact fields)
- Done: Unreliable VAT payer check (`vat_unreliable_at` timestamp, migration 005)
- Missing: VIES VAT validation for EU contacts
- Missing: Contact tags/groups

### Domain Model
- Done: Contact, Invoice, InvoiceItem, InvoiceSequence, Expense, Amount, BankTransaction, VATReturn, VATControlStatement, VIESSummary, filter structs
- Done: RecurringInvoice, RecurringExpense, ExpenseDocument, OCRResult, PaymentReminder, InvoiceStatusChange
- Done: Validated structs match schema; fixed Amount.String() for negative sub-unit values

**Dependencies:** None (this is the foundation)

---

## Phase 2 — Invoicing

**RFC:** `docs/rfc/002-invoicing.md`
**Goal:** Full invoice lifecycle — CRUD, PDF, QR codes, email, ISDOC export, proforma, credit notes, recurring.

### Sub-Phase 2A (Done)

#### Invoice CRUD
- Done: Invoice domain struct, repository (with transactions), service (with status transitions, duplication), HTTP handler
- Done: Frontend invoice list page (filtering, pagination)
- Done: Frontend invoice create form (line items, VAT calculations, customer selection)
- Done: Frontend invoice detail/edit page (`/invoices/[id]`) — view, edit draft, send, pay, duplicate, delete with PDF/QR/ISDOC buttons

#### PDF Generation
- Done: PDF generator using maroto/v2 (`internal/pdf/invoice_pdf.go`)
- Done: Full Czech invoice layout (header, supplier/customer, items, VAT summary, payment info, QR code)
- Done: PDF HTTP endpoint (GET `/api/v1/invoices/{id}/pdf`)

#### QR Payment Code
- Done: QR code generation using qrpay (`internal/pdf/qr_payment.go`)
- Done: Czech QR Platba SPD format
- Done: QR HTTP endpoint (GET `/api/v1/invoices/{id}/qr`)
- Done: QR display in frontend invoice detail

#### ISDOC Export
- Done: ISDOC 6.0.2 XML generation (`internal/isdoc/generator.go`, `internal/isdoc/types.go`)
- Done: Single invoice ISDOC endpoint (GET `/api/v1/invoices/{id}/isdoc`)
- Done: Batch ISDOC export as ZIP (POST `/api/v1/invoices/export/isdoc`)

#### Invoice Sequence Management
- Done: Sequence repository, service, handler (`internal/repository/sequence_repo.go`, etc.)
- Done: Frontend sequence management UI (`/settings/sequences`)
- Done: Auto-create sequence for current year on invoice create

### Sub-Phase 2B (Done)
- Done: Proforma invoices (`InvoiceTypeProforma`, full CRUD, frontend support)
- Done: Credit notes (`InvoiceTypeCreditNote`, `RelatedInvoiceID`, frontend support)
- Done: Settlement/linking (`RelationType`, migration 007, related invoice tracking)
- Done: Recurring invoices (domain, migration 009, repo, service with `GenerateInvoice()`, handler, frontend `/recurring/`)

### Sub-Phase 2C (Done)
- Done: Multi-currency with CNB exchange rates (`internal/service/cnb/client.go`, daily rate fetching, 1h cache, weekend fallback)
- Done: Exchange rate handler (GET `/api/v1/exchange-rates`)
- Done: `CurrencyCode` and `ExchangeRate` fields on Invoice, Expense, RecurringInvoice, RecurringExpense
- Done: Automatic overdue detection (`internal/service/overdue_svc.go`, marks sent invoices past due date)
- Done: Status history / timeline (migration 012, `InvoiceStatusChange` domain, repo, handler, frontend display)
- Done: Payment reminders (migration 013, `PaymentReminder` domain, repo, service, handler with send endpoint, frontend reminder history)

### Email Service (Done)
- Done: SMTP email sender with TLS (`internal/service/email/sender.go`)
- Done: Email template system (`internal/service/email/templates.go`)
- Done: Payment reminder templates (`internal/service/email/reminder_template.go`)
- Done: Attachment support, HTML/text bodies, TO/CC/BCC
- Done: Mailpit integration for dev email testing (`config.dev.toml`, `scripts/dev.sh`)

### Still Missing
- Missing: Customizable PDF templates (logo, colors, footer — currently hardcoded layout)
- Missing: PDF CLI command (only HTTP endpoint exists)

**Dependencies:** Phase 1

---

## Phase 3 — Expenses

**RFC:** `docs/rfc/003-expenses.md`
**Goal:** Expense management with document upload, AI OCR, recurring expenses, and tax review.

### Task 3 — Expense Categories (Done)
- Done: ExpenseCategory domain struct (`internal/domain/category.go`)
- Done: Category repository with CRUD (`internal/repository/category_repo.go`)
- Done: Category service with validation (`internal/service/category_svc.go`)
- Done: Category HTTP handler (`internal/handler/category_handler.go`)
- Done: Frontend category management page (`/settings/categories`)
- Done: CategoryPicker component for expense forms
- Done: Migration 004 with 16 default Czech OSVC categories
- Done: Integrated into expense create/edit pages

### Expense CRUD (Done)
- Done: Expense domain struct, repository, service (with VAT calculations), HTTP handler
- Done: Frontend expense list page (search, pagination)
- Done: Frontend expense create form (`/expenses/new`) with CategoryPicker
- Done: Frontend expense detail/edit page (`/expenses/[id]`) with CategoryPicker

### Task 1 — Document Upload (Done)
- Done: `ExpenseDocument` domain struct (`internal/domain/document.go`)
- Done: Migration 006 (`expense_documents` table)
- Done: Document repository (`internal/repository/document_repo.go`)
- Done: Document service (`internal/service/document_svc.go`)
- Done: Document handler with upload/download (`internal/handler/document_handler.go`)
- Done: Frontend document management in expense detail pages

### Task 2 — AI OCR (Done)
- Done: `OCRResult` domain struct (`internal/domain/ocr.go`)
- Done: OpenAI OCR provider (`internal/service/ocr/openai.go`)
- Done: OCR service (`internal/service/ocr_svc.go`)
- Done: OCR handler (`internal/handler/ocr_handler.go`)
- Done: Frontend OCR file upload in `/expenses/new/`
- Done: Supported formats: JPEG, PNG, PDF
- Done: Config: `cfg.OCR.APIKey` and `cfg.OCR.Provider`

### Task 4 — Recurring Expenses (Done)
- Done: `RecurringExpense` domain struct (`internal/domain/recurring_expense.go`)
- Done: Migration 010 (`recurring_expenses` table)
- Done: Repository with full CRUD (`internal/repository/recurring_expense_repo.go`)
- Done: Service with `GeneratePending()` logic (`internal/service/recurring_expense_svc.go`)
- Done: Handler (`internal/handler/recurring_expense_handler.go`)
- Done: Frontend pages (`/expenses/recurring/`)

### Task 5 — Tax Review (Done)
- Done: `TaxReviewedAt` field on Expense (nullable timestamp)
- Done: Migration 008 (`tax_reviewed_at` column)
- Done: Review/unreview endpoints (POST `/api/v1/expenses/review`, `/unreview`)
- Done: Frontend tax review page (`/expenses/review/`)
- Done: Filter by `tax_reviewed` status in expense list

**Dependencies:** Phase 1

---

## Phase 4 — VAT Filings

**RFC:** `docs/rfc/004-vat-filings.md`
**Goal:** Generate XML submissions for DPH, kontrolni hlaseni, souhrnne hlaseni.

### VAT Returns — Prizani k DPH (Done)
- Done: `VATReturn` domain struct (`internal/domain/tax.go`)
- Done: Migration 014 (`vat_returns` table)
- Done: Repository (`internal/repository/vat_return_repo.go`)
- Done: Service (`internal/service/vat_return_svc.go`)
- Done: Handler (`internal/handler/vat_return_handler.go`)
- Done: XML generation (`internal/vatxml/vat_return_gen.go`)
- Done: Frontend pages (`/vat/returns/`)

### VAT Control Statement — Kontrolni hlaseni (Done)
- Done: `VATControlStatement` + `VATControlStatementLine` domain structs
- Done: Migration 014 (`vat_control_statements`, `vat_control_statement_lines` tables)
- Done: Repository (`internal/repository/vat_control_repo.go`)
- Done: Service (`internal/service/vat_control_svc.go`)
- Done: Handler (`internal/handler/vat_control_handler.go`)
- Done: XML generation (`internal/vatxml/control_statement_gen.go`)
- Done: Frontend pages (`/vat/control/`)

### VIES Summary — Souhrnne hlaseni (Done)
- Done: `VIESSummary` + `VIESSummaryLine` domain structs
- Done: Migration 014 (`vies_summaries`, `vies_summary_lines` tables)
- Done: Repository (`internal/repository/vies_repo.go`)
- Done: Service (`internal/service/vies_svc.go`)
- Done: Handler (`internal/handler/vies_handler.go`)
- Done: XML generation (`internal/vatxml/vies_gen.go`)
- Done: Frontend pages (`/vat/vies/`)

### RFC-005 VAT Polish (Done)
- Done: All 7 items already implemented during RFC-004
- Done: API clients consolidated in `client.ts`, shared utils in `$lib/utils/vat.ts`
- Done: Czech diacritics, month/quarter validation, input validation hardening
- Done: Content-Disposition safety with `mime.FormatMediaType()`
- Done: Dashboard uses calendar grid with per-year caching

**Dependencies:** Phase 2, Phase 3

---

## Phase 5 — Annual Tax

**RFC:** `docs/rfc/005-annual-tax.md` (not yet written)
**Goal:** Income tax return, social insurance overview, health insurance overview.

- Missing: Everything (no existing code beyond domain structs)

**Dependencies:** Phase 2, Phase 3, Phase 4

---

## Phase 6 — Banking & Automation

**RFC:** `docs/rfc/006-banking.md` (not yet written)
**Goal:** FIO Bank integration, automatic payment matching, bank statement import.

- Exists: FIO config in config struct, BankTransaction domain struct
- Missing: FIO API client, transaction import, matching, bank_transactions table, repository, frontend

**Dependencies:** Phase 2

---

## Phase 7 — Web UI

**RFC:** `docs/rfc/007-web-ui.md` (not yet written)
**Goal:** Full-featured dashboard, complete CRUD UIs, reports and charts.

### Dashboard
- Exists: Dashboard page with hardcoded zeros (no real data)
- Missing: Financial overview, charts, outstanding invoices, tax deadlines, API endpoints

### CRUD UIs
- Done: Contact list + detail/edit with ARES lookup UI
- Done: Invoice list + create form + detail/edit page (incl. proforma, credit notes)
- Done: Expense list + create form + detail/edit page (incl. documents, OCR)
- Done: Settings page (Identity/Address/Bank/VAT sections)
- Done: Recurring invoice management (`/recurring/`)
- Done: Recurring expense management (`/expenses/recurring/`)
- Done: Tax review page (`/expenses/review/`)
- Done: VAT filing pages (`/vat/returns/`, `/vat/control/`, `/vat/vies/`)

### Reports
- Missing: All reporting, charts, CSV/PDF export

**Dependencies:** Phase 1-6

---

## Phase 8 — Polish

**RFC:** `docs/rfc/008-polish.md` (not yet written)
**Goal:** Asset register, vehicle log, notifications, backup, data box integration.

- Missing: Everything

**Dependencies:** Phase 1-7

---

## Cross-cutting Concerns (All Phases)

| Concern | Status | Notes |
|---------|--------|-------|
| Unit tests | Done | ~1500 Go test files, ~57 vitest test files |
| Integration tests | Done | testutil with in-memory SQLite, seed factories, FK handling |
| CI pipeline | Done | `.github/workflows/ci.yml` |
| Structured logging | Done | `log/slog` throughout (middleware, services, email, OCR, overdue) |
| Audit trail | Exists (partial) | Table exists in migration 001, no repository/service writes to it |
| Error handling | Exists (partial) | Handlers return errors; no global strategy |
| Input validation | Done | Settings key allowlist, ICO validation, service-layer checks |
| API documentation | Missing | No OpenAPI/Swagger |
| Frontend error states | Exists (partial) | List pages have loading/error/empty states; no toast system |
| API consistency | Done | Fixed `/mark-paid` mismatch, SequenceID NULL handling |
| Security hardening | Done | LimitReader, localhost binding, security headers, settings allowlist |

---

## Dependency Graph

```
Phase 1 (Foundation)
  |
  +---> Phase 2 (Invoicing) --+
  |                            |
  +---> Phase 3 (Expenses) ----+---> Phase 4 (VAT) ---> Phase 5 (Tax)
                               |
                               +---> Phase 6 (Banking)
                               |
                               +---> Phase 7 (Web UI)
                                          |
                                          +---> Phase 8 (Polish)
```

## RFC Schedule

1. **RFC-001** (Foundation) — Done (implemented, tested, reviewed)
2. **RFC-002** (Invoicing) — Done (Sub-Phase 2A-2C, email service)
3. **RFC-003** (Expenses) — Done (categories, documents, OCR, recurring, tax review)
4. **RFC-004** (VAT Filings) — Done (VAT returns, kontrolni hlaseni, souhrnne hlaseni)
5. **RFC-005** (VAT Polish) — Done (all items implemented during RFC-004)
6. **RFC-006** (Banking) — Write after RFC-005
7. **RFC-007** (Web UI) — Write incrementally as backend phases complete
8. **RFC-008** (Polish) — Write last
