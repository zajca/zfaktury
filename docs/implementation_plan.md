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
- Done: Test infrastructure (testutil with in-memory SQLite, seed factories; 96 Go + 53 vitest tests)
- Missing: CI pipeline (linting, testing, building)

### SQLite Schema & Database
- Done: SQLite connection with WAL mode, foreign keys, busy timeout
- Done: Goose migration framework with embedded SQL
- Done: Initial schema + migration 002 (schema alignment fixes)
- Done: Amount (int64 halere) consistency validated across all columns
- Missing: bank_transactions table (domain struct exists, no migration)
- Missing: Audit log write logic (table exists, nothing writes to it)

### CLI Framework
- Exists: Cobra root command + `serve` command
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
- Missing: VIES VAT validation for EU contacts
- Missing: Contact tags/groups
- Missing: Unreliable VAT payer check (nespolehlivy platce)

### Domain Model
- Done: Contact, Invoice, InvoiceItem, InvoiceSequence, Expense, Amount, BankTransaction, VATReturn, VATControlStatement, VIESSummary, filter structs
- Done: Validated structs match schema; fixed Amount.String() for negative sub-unit values

**Dependencies:** None (this is the foundation)

---

## Phase 2 — Invoicing

**RFC:** `docs/rfc/002-invoicing.md`
**Goal:** Full invoice lifecycle — CRUD, PDF, QR codes, email, ISDOC export.

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

### Sub-Phase 2B (Missing)
- Missing: Proforma invoices (zalohove faktury) and settlement
- Missing: Credit notes (dobropisy)
- Missing: Recurring invoices with configurable intervals

### Sub-Phase 2C (Missing)
- Missing: Multi-currency with CNB exchange rates
- Missing: Automatic overdue detection
- Missing: Status history / timeline
- Missing: Payment reminders

### Not Yet Started
- Missing: Email service, templates, HTTP endpoint (SMTP config exists)
- Missing: Frontend invoice edit form for non-draft invoices
- Missing: Customizable PDF templates (logo, colors, footer)
- Missing: PDF CLI command

**Dependencies:** Phase 1

---

## Phase 3 — Expenses

**RFC:** `docs/rfc/003-expenses.md`
**Goal:** Expense management with document upload and AI OCR.

### Task 3 — Expense Categories (Done)
- Done: ExpenseCategory domain struct (`internal/domain/category.go`)
- Done: Category repository with CRUD (`internal/repository/category_repo.go`)
- Done: Category service with validation (`internal/service/category_svc.go`)
- Done: Category HTTP handler (`internal/handler/category_handler.go`)
- Done: Frontend category management page (`/settings/categories`)
- Done: CategoryPicker component for expense forms
- Done: Migration 004 with 16 default Czech OSVC categories
- Done: Integrated into expense create/edit pages

### Expense CRUD
- Done: Expense domain struct, repository, service (with VAT calculations), HTTP handler
- Done: Frontend expense list page (search, pagination)
- Done: Frontend expense create form (`/expenses/new`) with CategoryPicker
- Done: Frontend expense detail/edit page (`/expenses/[id]`) with CategoryPicker

### Remaining Tasks (Missing)
- Missing: Task 1 — Document upload (file upload endpoint, document storage, linking, viewer)
- Missing: Task 2 — AI OCR (OCR service, data extraction, user confirmation UI; OCR config exists)
- Missing: Task 4 — Recurring expenses
- Missing: Task 5 — Tax-deductible marking workflow

**Dependencies:** Phase 1

---

## Phase 4 — VAT Filings

**RFC:** `docs/rfc/004-vat-filings.md` (not yet written)
**Goal:** Generate XML submissions for DPH, kontrolni hlaseni, souhrnne hlaseni.

- Exists: VATReturn, VATControlStatement, VIESSummary domain structs
- Missing: All calculation services, XML generation, validation, handlers, frontend pages

**Dependencies:** Phase 2, Phase 3

---

## Phase 5 — Annual Tax

**RFC:** `docs/rfc/005-annual-tax.md` (not yet written)
**Goal:** Income tax return, social insurance overview, health insurance overview.

- Missing: Everything (no existing code beyond domain structs in Phase 4)

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
- Done: Invoice list + create form + detail/edit page
- Done: Expense list + create form + detail/edit page
- Done: Settings page (Identity/Address/Bank/VAT sections)
- Missing: Tax filing pages

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
| Unit tests | Done (Phase 1) | 96 Go tests (domain, repo, service, handler) + 53 vitest (API client, money, date) |
| Integration tests | Done (Phase 1) | testutil with in-memory SQLite, seed factories, FK handling |
| Structured logging | Exists (partial) | slog middleware logs requests; needs expansion |
| Audit trail | Missing | Table exists, no write logic |
| Error handling | Exists (partial) | Handlers return errors; no global strategy |
| Input validation | Done (Phase 1) | Settings key allowlist, ICO validation, service-layer checks |
| API documentation | Missing | No OpenAPI/Swagger |
| Frontend error states | Exists (partial) | List pages have loading/error/empty states; no toast system |
| API consistency | Done (Phase 1) | Fixed `/mark-paid` mismatch, SequenceID NULL handling |
| Security hardening | Done (Phase 1) | LimitReader, localhost binding, security headers, settings allowlist |

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
2. **RFC-002** (Invoicing) — Written, Sub-Phase 2A done (PDF, QR, ISDOC, sequences)
3. **RFC-003** (Expenses) — Written, Task 3 done (expense categories)
4. **RFC-004** (VAT Filings) — Write after RFC-002 + RFC-003
5. **RFC-005** (Annual Tax) — Write after RFC-004
6. **RFC-006** (Banking) — Write after RFC-002
7. **RFC-007** (Web UI) — Write incrementally as backend phases complete
8. **RFC-008** (Polish) — Write last
