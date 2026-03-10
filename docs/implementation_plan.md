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
- Exists: Go module with dependencies (chi, cobra, sqlite, goose, toml, maroto, qrpay)
- Exists: Makefile with `dev`, `build`, `test` targets
- Exists: SvelteKit frontend with Tailwind v4, TypeScript, adapter-static
- Exists: Dev mode with Vite HMR proxy
- Exists: 3-layer architecture (handler -> service -> repository)
- Missing: Test infrastructure (zero test files)
- Missing: CI pipeline (linting, testing, building)

### SQLite Schema & Database
- Exists: SQLite connection with WAL mode, foreign keys, busy timeout
- Exists: Goose migration framework with embedded SQL
- Exists: Initial schema (contacts, invoices, invoice_items, invoice_sequences, expenses, audit_log, settings)
- Missing: bank_transactions table (domain struct exists, no migration)
- Needs review: Amount (int64 halere) consistency across all columns
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
- Exists: Config struct (UserConfig, ServerConfig, SMTPConfig, FIOConfig, OCRConfig)
- Exists: TOML config file loading from `~/.zfaktury/config.toml`
- Exists: `ZFAKTURY_DATA_DIR` env var override
- Exists: Settings key-value table in database
- Missing: Settings HTTP handler (GET/PUT /api/v1/settings)
- Missing: Settings service layer
- Missing: Frontend settings page (currently shows "Pripravuje se...")

### Contact Management
- Exists: Contact domain struct, repository, service, HTTP handler, frontend list + detail/edit pages
- Exists: ARES client interface in service layer, ARES lookup endpoint in handler, ARES lookup button in frontend
- Missing: ARES HTTP client implementation (interface defined, wired as nil — lookup does nothing)
- Missing: VIES VAT validation for EU contacts
- Missing: Contact tags/groups
- Missing: Unreliable VAT payer check (nespolehlivy platce)

### Domain Model
- Exists: Contact, Invoice, InvoiceItem, InvoiceSequence, Expense, Amount, BankTransaction, VATReturn, VATControlStatement, VIESSummary, filter structs
- Needs review: Validate all structs match idea.md requirements and schema

**Dependencies:** None (this is the foundation)

---

## Phase 2 — Invoicing

**RFC:** `docs/rfc/002-invoicing.md` (not yet written)
**Goal:** Full invoice lifecycle — CRUD, PDF, QR codes, email, ISDOC export.

### Invoice CRUD
- Exists: Invoice domain struct, repository (with transactions), service (with status transitions, duplication), HTTP handler
- Exists: Frontend invoice list page (filtering, pagination)
- Exists: Frontend invoice create form (line items, VAT calculations, customer selection)
- Missing: Frontend invoice edit form (no `/invoices/[id]/edit` route)
- Missing: Frontend invoice detail page
- Missing: Invoice numbering sequence management UI
- Missing: Proforma invoices (zalohove faktury) and settlement
- Missing: Credit notes (dobropisy)
- Missing: Recurring invoices with configurable intervals
- Missing: Multi-currency with CNB exchange rates

### PDF Generation
- Missing: PDF service using maroto/v2 (dependency installed, not used)
- Missing: Invoice PDF template (header, items table, totals, bank info, QR code)
- Missing: Customizable templates (logo, colors, footer)
- Missing: PDF HTTP endpoint (GET /api/v1/invoices/{id}/pdf)
- Missing: PDF CLI command

### QR Payment Code
- Missing: QR code generation using qrpay (dependency installed, not used)
- Missing: Czech QR Platba SPD format
- Missing: Embed QR in PDF and display in frontend

### Email Sending
- Exists: SMTP config in config struct
- Missing: Email service, templates, HTTP endpoint

### ISDOC Export
- Missing: ISDOC XML generation, HTTP endpoint, batch export

### Invoice Status Tracking
- Exists: Status field, MarkAsSent/MarkAsPaid service methods
- Missing: Automatic overdue detection
- Missing: Status history / timeline
- Missing: Payment reminders

**Dependencies:** Phase 1

---

## Phase 3 — Expenses

**RFC:** `docs/rfc/003-expenses.md` (not yet written)
**Goal:** Expense management with document upload and AI OCR.

### Expense CRUD
- Exists: Expense domain struct, repository, service (with VAT calculations), HTTP handler
- Exists: Frontend expense list page (search, pagination)
- Missing: Frontend expense create/edit form (add button exists but no route)
- Missing: Frontend expense detail page
- Missing: Expense categories management
- Missing: Recurring expenses
- Missing: Tax-deductible marking workflow

### Document Upload
- Missing: File upload endpoint, document storage, linking, viewer

### AI OCR
- Exists: OCR config in config struct
- Missing: OCR service, data extraction, user confirmation UI

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
- Exists: Contact list + detail/edit with ARES lookup UI
- Exists: Invoice list + create form
- Exists: Expense list
- Missing: Invoice edit/detail pages
- Missing: Expense create/edit/detail pages
- Missing: Settings page (placeholder only)
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
| Unit tests | Missing | Zero test files exist |
| Integration tests | Missing | Need test SQLite setup |
| Structured logging | Exists (partial) | slog middleware logs requests; needs expansion |
| Audit trail | Missing | Table exists, no write logic |
| Error handling | Exists (partial) | Handlers return errors; no global strategy |
| Input validation | Exists (partial) | Basic checks in services; no systematic approach |
| API documentation | Missing | No OpenAPI/Swagger |
| Frontend error states | Exists (partial) | List pages have loading/error/empty states; no toast system |
| API consistency | Needs review | Known mismatch: TS client `/mark-paid` vs Go handler `/pay` |

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

1. **RFC-001** (Foundation) — Written, pending review
2. **RFC-002** (Invoicing) — Write after RFC-001 is validated
3. **RFC-003** (Expenses) — Write after RFC-001 is validated
4. **RFC-004** (VAT Filings) — Write after RFC-002 + RFC-003
5. **RFC-005** (Annual Tax) — Write after RFC-004
6. **RFC-006** (Banking) — Write after RFC-002
7. **RFC-007** (Web UI) — Write incrementally as backend phases complete
8. **RFC-008** (Polish) — Write last
