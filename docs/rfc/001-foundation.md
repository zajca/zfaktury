# RFC-001: Foundation Phase

**Status:** Draft
**Date:** 2026-03-10

## Summary

Phase 1 establishes the project foundation: database schema, CLI framework, user settings, contact management with ARES integration, and test infrastructure. Significant prototype code already exists in the repository but has not been formally reviewed or tested.

## Existing Code Inventory

The following code exists as unreviewed prototype. It must be validated, tested, and fixed as part of this RFC's implementation.

### Backend (Go)

| Layer | Files | What exists |
|-------|-------|-------------|
| Domain | `internal/domain/*.go` | Contact, Invoice, InvoiceItem, InvoiceSequence, Expense, Amount (int64 halere), BankTransaction, VATReturn, VATControlStatement, VIESSummary, filter structs |
| Repository | `internal/repository/*.go` | ContactRepo, InvoiceRepo, ExpenseRepo with SQL CRUD, filtering, pagination. Interfaces in `interfaces.go` |
| Service | `internal/service/*.go` | ContactService (with ARESClient interface, wired as nil), InvoiceService (CRUD, status transitions, duplication), ExpenseService (CRUD, VAT calc) |
| Handler | `internal/handler/*.go` | Contact, Invoice, Expense handlers with REST endpoints. DTOs in `helpers.go`. chi router in `router.go` |
| CLI | `internal/cli/*.go` | Root command + `serve` command with dev proxy |
| Config | `internal/config/config.go` | TOML loading, UserConfig, ServerConfig, SMTP, FIO, OCR |
| Database | `internal/database/*.go` | SQLite connection (WAL, FK, busy timeout), goose migrations |
| Schema | `migrations/001_initial_schema.sql` | contacts, invoices, invoice_items, invoice_sequences, expenses, audit_log, settings tables |

### Frontend (SvelteKit)

| Page | Route | What exists |
|------|-------|-------------|
| Layout | `+layout.svelte` | Responsive sidebar nav, mobile menu |
| Dashboard | `+page.svelte` | Hardcoded zeros, no API calls |
| Contacts list | `contacts/+page.svelte` | Real API calls, search, pagination |
| Contact detail/edit | `contacts/[id]/+page.svelte` | Full form (20+ fields), ARES lookup button, create/edit/delete |
| Invoices list | `invoices/+page.svelte` | Real API calls, status filter, search, pagination |
| Invoice create | `invoices/new/+page.svelte` | Line items, VAT calc, customer select, submit |
| Expenses list | `expenses/+page.svelte` | Real API calls, search, pagination |
| Settings | `settings/+page.svelte` | Placeholder — "Pripravuje se..." |
| API client | `src/lib/api/client.ts` | contactsApi, invoicesApi, expensesApi with real fetch calls |
| Utils | `src/lib/utils/money.ts`, `date.ts` | CZK formatting, Czech locale dates |

### Known Issues in Existing Code

1. **ARES client wired as nil** — `serve.go` passes nil for ARESClient, so ARES lookup endpoint returns nothing
2. **API endpoint mismatch** — TypeScript calls `/mark-paid`, Go registers `/pay`
3. **No tests** — Zero `*_test.go` or `*.test.ts` files
4. **No settings backend** — Table exists, no service/handler/frontend
5. **Missing frontend pages** — Invoice edit/detail, expense create/edit/detail
6. **Dashboard empty** — Hardcoded zeros
7. **Audit log unused** — Table exists, nothing writes to it
8. **bank_transactions** — Domain struct exists, no DB table

---

## Task 1: Review & Validate Existing Code

Before building new features, the prototype code must be validated.

### Implementation Tasks

1. **Schema review**
   - Verify all monetary columns are INTEGER (halere, not float)
   - Verify domain structs match schema columns
   - Check foreign key constraints and indexes are correct
   - Verify soft-delete pattern is consistent (deleted_at on all relevant tables)

2. **Fix API endpoint mismatch**
   - Align TypeScript client and Go handler for mark-paid/pay endpoint
   - Audit all other endpoints for mismatches between client.ts and router.go

3. **Review domain logic**
   - Validate `CalculateTotals()` arithmetic (VAT rounding, halere precision)
   - Validate `Amount` type operations (overflow, negative values)
   - Validate invoice status transitions (which transitions are allowed?)

4. **Review repository SQL**
   - Check for SQL injection risks (parameterized queries)
   - Verify transaction handling (Create invoice with items)
   - Verify soft-delete is respected in all List/Get queries
   - Verify pagination (off-by-one errors, total count accuracy)

5. **Review handler layer**
   - Validate request parsing and error responses
   - Check DTO <-> domain mapping completeness
   - Verify HTTP status codes follow REST conventions

6. **Review frontend**
   - Verify all forms submit correct data shapes to API
   - Check money conversion (halere <-> CZK) in all places
   - Verify error handling in API calls
   - Run `npm run check` and fix TypeScript errors

### Acceptance Criteria

- [ ] All monetary values use INTEGER in DB and int64/Amount in Go — no floats
- [ ] API endpoints match between Go handlers and TypeScript client
- [ ] All repository queries use parameterized SQL
- [ ] Soft-delete is consistent across all entities
- [ ] `npm run check` passes without errors

---

## Task 2: Test Infrastructure

### Implementation Tasks

1. **Go test helpers (`internal/testutil/` or `tests/helpers/`)**
   - `NewTestDB()` — creates in-memory SQLite with migrations applied
   - `SeedContacts()`, `SeedInvoices()`, `SeedExpenses()` — test data factories

2. **Domain tests**
   - `internal/domain/money_test.go` — Amount arithmetic, overflow, conversions, String()
   - `internal/domain/invoice_test.go` — CalculateTotals(), IsOverdue(), IsPaid()

3. **Repository tests**
   - `internal/repository/contact_repo_test.go` — CRUD, FindByICO, List with filters, soft-delete
   - `internal/repository/invoice_repo_test.go` — CRUD with items (transaction), List with filters, UpdateStatus, GetNextNumber
   - `internal/repository/expense_repo_test.go` — CRUD, List with filters
   - All tests use real SQLite (in-memory), no mocks

4. **Service tests**
   - `internal/service/contact_svc_test.go` — validation, ARES integration (mock ARESClient interface)
   - `internal/service/invoice_svc_test.go` — validation, status transitions, duplication logic
   - `internal/service/expense_svc_test.go` — VAT calculations, business share

5. **Handler tests**
   - `internal/handler/contact_handler_test.go` — HTTP request/response with httptest
   - `internal/handler/invoice_handler_test.go`
   - `internal/handler/expense_handler_test.go`

6. **Makefile integration**
   - `make test` runs Go tests + frontend type check

### Acceptance Criteria

- [ ] `go test ./...` passes
- [ ] Coverage > 80% on domain, repository, and service layers
- [ ] `cd frontend && npm run check` passes
- [ ] No external dependencies or network calls in tests

---

## Task 3: ARES Integration

### Background

ARES (Administrativni registr ekonomickych subjektu) is the Czech business registry API.

- **Endpoint:** `https://ares.gov.cz/ekonomicke-subjekty-v-be/rest/ekonomicke-subjekty/{ICO}`
- **Response:** JSON with company name, address, DIC (VAT ID), legal form

Existing code: ARESClient interface in service, handler endpoint, frontend lookup button — all wired but returning nothing because the client is nil.

### Implementation Tasks

1. **Create `internal/ares/client.go`**
   - HTTP client with configurable timeout (10s default)
   - `LookupByICO(ctx, ico) (*domain.Contact, error)` method
   - ICO validation (8 digits, checksum)
   - Handle: 404 (not found), rate limiting, network errors

2. **Create `internal/ares/types.go`**
   - Response structs matching ARES API JSON schema
   - Mapping function: ARES response -> domain.Contact

3. **Wire ARES client in `internal/cli/serve.go`**
   - Replace `nil` with actual ARES client instance

4. **Create `internal/ares/client_test.go`**
   - Unit tests with httptest (recorded responses)
   - Test: valid ICO, invalid ICO, not found, timeout, malformed response

### Acceptance Criteria

- [ ] `GET /api/v1/contacts/ares/12345678` returns parsed company data
- [ ] Invalid ICO returns 400 with descriptive error
- [ ] Network errors return 502
- [ ] Unit tests cover happy path and error cases
- [ ] Frontend ARES lookup works end-to-end (enter ICO, fields auto-fill)

---

## Task 4: User Settings

### Background

User settings store the OSVC's identity (ICO, DIC, name, address, bank accounts) and app preferences. The `settings` key-value table exists in the DB, config structs exist, but there's no way to manage settings through the API or UI.

### Implementation Tasks

1. **Settings repository (`internal/repository/settings_repo.go`)**
   - Implement against `settings` table (key-value store)
   - Add SettingsRepository interface to `interfaces.go`

2. **Settings service (`internal/service/settings_svc.go`)**
   - `GetAll(ctx) (map[string]string, error)`
   - `Get(ctx, key) (string, error)`
   - `Set(ctx, key, value) error`
   - `SetBulk(ctx, map[string]string) error`
   - Known setting keys as constants

3. **Settings handler (`internal/handler/settings_handler.go`)**
   - `GET /api/v1/settings` — returns all settings
   - `PUT /api/v1/settings` — bulk update
   - Mount in router.go, wire in serve.go

4. **Frontend settings page**
   - Replace placeholder in `settings/+page.svelte`
   - Form sections: Identity (name, ICO, DIC), Address, Bank accounts, VAT status
   - Add `settingsApi` to `client.ts`

5. **Tests**
   - Repository, service, handler tests

### Acceptance Criteria

- [ ] Settings persist to SQLite and survive restarts
- [ ] Frontend loads current values and saves changes
- [ ] Required fields validated
- [ ] Other services can read settings (e.g., for invoice PDF supplier info)

---

## Task 5: Missing Frontend Pages

### Background

Several CRUD pages are missing. Contact pages are complete but invoice edit/detail and all expense management pages beyond the list are absent.

### Implementation Tasks

1. **Invoice detail/edit page (`frontend/src/routes/invoices/[id]/+page.svelte`)**
   - View invoice with all fields, items, totals
   - Edit mode for draft invoices
   - Action buttons: send, mark paid, duplicate, delete

2. **Expense create form (`frontend/src/routes/expenses/new/+page.svelte`)**
   - Form: description, amount, date, category, vendor, VAT rate, business share
   - Submit to `expensesApi.create()`

3. **Expense detail/edit page (`frontend/src/routes/expenses/[id]/+page.svelte`)**
   - View/edit expense fields, delete with confirmation

4. **Fix expense list add button** — link to `/expenses/new`

### Acceptance Criteria

- [ ] User can view and edit existing invoices
- [ ] User can create, view, and edit expenses
- [ ] All list pages link correctly to detail/create pages

---

## Implementation Order

```
1. Review & Validate Existing Code (Task 1)
   - Must happen first — establishes what needs fixing

2. Test Infrastructure (Task 2)
   - Tests validate the fixes from Task 1

3. ARES Integration (Task 3) + User Settings (Task 4)
   - Independent of each other, can be parallelized

4. Missing Frontend Pages (Task 5)
   - Depends on backend being validated and tested
```

---

## Out of Scope

- Invoice PDF generation (Phase 2 — RFC-002)
- Expense document upload / OCR (Phase 3 — RFC-003)
- Tax filing XML generation (Phase 4-5)
- Bank integration (Phase 6)
- Dashboard with real data (Phase 7)
- VIES validation, unreliable payer check (future contact enhancement)
