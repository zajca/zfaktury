# Multi-Company Support — Design Spec

| Field | Value |
|---|---|
| Date | 2026-05-24 |
| Version | v2 — incorporates review feedback from Gemini CLI (Architect Role); see "Review notes" at end |
| Status | Approved (pending implementation plan) |
| Branch | `feature/multi-company` |
| Author | Jiri Manas (with brainstorming via Claude) |
| Audience | Upstream maintainer (zajca), AI design reviewer, implementer |

## TL;DR

Add support for one user managing multiple legal entities (e.g., an OSVČ plus an s.r.o.) inside a single zfaktury install. Every business-domain entity becomes per-company; the current "company info" stored as ad-hoc keys in the `settings` table is promoted to a first-class `companies` table. The active company is selectable via a header dropdown and persisted in localStorage; the backend uses a URL prefix `/api/v1/companies/{companyID}/…` for all per-company routes. Existing single-company data is auto-migrated into a default company on first launch. No authentication or multi-user changes — the human at the keyboard is still one person.

## Goals

- A single zfaktury install supports N companies (typical N: 2–5).
- Each company has its own contacts, invoices, expenses, sequences, tax filings, and settings — strictly partitioned with no cross-company leakage.
- Switching is instant: pick a different company from a header dropdown, every page re-loads its data scoped to the new company.
- Existing single-company users upgrade transparently — their data lands in a pre-created default company on first launch.
- Backups remain full-database snapshots (cover all companies).

## Non-Goals

- **Multi-user / authentication.** zfaktury today has no auth (single-binary local app); this change does not introduce users, roles, or sessions. Adding auth is a separate, larger initiative.
- **Per-company backup files.** A single nightly snapshot still covers everything; per-company export (ISDOC bundle, PDF archive) may come later as a separate feature.
- **Cross-company data sharing.** Contacts, expense categories, and templates are strictly per-company. If the same human client invoices through two of the user's companies, the contact is duplicated.
- **Schema-per-company or database-per-company isolation.** All companies share one SQLite file and one schema; partitioning is column-based (`company_id` FK).
- **URL-based frontend routing per company.** The browser URL stays `/invoices`, `/expenses`, etc. The current company is reflected in API URLs and in the localStorage-backed store, not in the frontend route.

## Context

zfaktury is a self-contained invoicing and tax-management app for Czech sole proprietors (OSVČ). Architecture: Go backend (chi HTTP, cobra CLI, SQLite via `modernc.org/sqlite`) + SvelteKit frontend embedded as a static SPA in the Go binary. Three-layer backend: pure domain structs → repositories on `*sql.DB` → services → HTTP handlers with JSON DTOs.

Today the app assumes exactly one business: 17 keys in the `settings` key-value table store the company identity (`company_name`, `ico`, `dic`, `vat_registered`, address, bank details, personal-name fields for tax filings, presentation defaults). Every other table — contacts, invoices, invoice_items, invoice_sequences, expenses, expense_categories, recurring_*, VAT filings, income-tax returns, social/health insurance overviews, investments, payment reminders, etc. — has no notion of which business it belongs to.

There are 24 goose migrations as of `024_backup_history.sql`. Database mode: WAL, foreign keys ON, busy timeout 5000ms. Migrations are embedded SQL files in `internal/database/migrations/`.

## Decisions (locked in during brainstorming)

| Decision | Choice | Reasoning |
|---|---|---|
| Use case | Multiple legal entities owned by the same person | Drives full feature parity per company, not a sandbox/test split |
| Data sharing | Strictly per-company | Cleanest partitioning, mirrors Pohoda / Money S3 conventions; small N makes occasional duplicate-contact entry acceptable |
| Switcher UX | Header dropdown above existing sidebar | Familiar (Notion / Linear / Fakturoid), persists via localStorage, minimal layout change |
| Existing-data migration | Auto-migrate seamlessly | Goose migration creates a `companies` row from the existing 17 settings keys; no wizard / no manual step |
| API transport | URL path prefix `/api/v1/companies/{companyID}/…` | Explicit, RESTful, every URL self-documents its company scope; worth the per-handler churn |
| Backups | Single full-DB snapshot (unchanged behavior) | Same human owns all companies — one restore restores everything |

## Data model

### New `companies` table

```sql
CREATE TABLE companies (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT    NOT NULL,                  -- short name shown in switcher dropdown
    legal_name      TEXT    NOT NULL,                  -- full legal name printed on invoices
    ico             TEXT    NOT NULL,                  -- IČO (Czech business ID)
    dic             TEXT,                              -- DIČ (VAT ID), NULL for non-VAT-payers
    vat_registered  INTEGER NOT NULL DEFAULT 0,        -- 0/1 boolean
    -- address
    street          TEXT, house_number TEXT, city TEXT, zip TEXT,
    -- contact
    email           TEXT, phone TEXT,
    -- personal name (used for OSVČ tax filings — the human behind the business)
    first_name      TEXT, last_name TEXT,
    -- bank
    bank_account    TEXT, bank_code TEXT, iban TEXT, swift TEXT,
    -- presentation
    logo_path       TEXT, accent_color TEXT,
    -- audit
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    deleted_at      TEXT                                -- soft delete; mirrors the rest of the app
);
CREATE UNIQUE INDEX idx_companies_ico_active ON companies(ico) WHERE deleted_at IS NULL;
```

The 17 company-identity keys leave the `settings` key-value table and become real columns here. Validation (matches the app's existing rules):

- `ico` is required and unique among non-deleted companies.
- `dic` must follow Czech format `CZ\d{8,10}` if `vat_registered = 1`; otherwise free-form / NULL.
- `name` must be unique (not enforced at DB level, surfaced as a friendly UI warning — two companies legitimately can share a display name).

### Per-company tables (gain `company_id INTEGER NOT NULL REFERENCES companies(id)`)

contacts, invoices, invoice_items, invoice_sequences, invoice_status_history, invoice_documents, recurring_invoices, recurring_invoice_items, recurring_expenses, expenses, expense_items, expense_documents, expense_categories, payment_reminders, tax_year_settings, tax_prepayments, vat_returns (+ vat_return_invoices, vat_return_expenses), vat_control_statements (+ lines), vies_summaries (+ lines), income_tax_returns, social_insurance_overviews, health_insurance_overviews, tax_spouse_credits, tax_child_credits, tax_personal_credits, tax_deductions, tax_deduction_documents, investment_documents, capital_income_entries, security_transactions, fakturoid_import_log, settings.

Each gets a matching `CREATE INDEX idx_<table>_company ON <table>(company_id)` for query performance — every list query becomes `WHERE company_id = ? AND …`.

### Tables that stay global

- `audit_log` — cross-company auditing trail for the human user. Records actions across all companies; the `company_id` (nullable) is added as a *value*, not as a partition key, so admin queries can filter or aggregate across companies.
- `backup_history` — full-DB snapshots cover all companies in one file. Already global by nature.
- `schema_migrations` (goose's own metadata table).

### Constraint changes

- `invoice_sequences`: `UNIQUE(prefix, year)` → `UNIQUE(company_id, prefix, year)`. Two companies can both have `FV2026-0001` independently. Requires the SQLite "rename → recreate → copy → drop" rebuild because SQLite cannot add or replace a UNIQUE constraint via `ALTER TABLE`.
- `settings`: gains `company_id`, and the unique index becomes `(company_id, key)` so each company keeps its own templates, defaults, and Czech-specific office codes.

### Soft-delete semantics for companies

- Companies have a nullable `deleted_at`. Soft-deleted companies are hidden from the switcher dropdown and from the global `GET /companies` list.
- A company with **any non-deleted** invoice, expense, or tax filing **cannot be soft-deleted** — the service returns `domain.ErrInUse` (new sentinel). The user must soft-delete the contained records first.
- The switcher never shows zero companies. If only one company exists, soft-delete is blocked at the service layer regardless of other state.
- Soft-deleted companies retain their FK relationships for audit-log integrity. Hard delete is not exposed in the UI.

## API surface

### Two route tiers

**Global tier** (no company prefix):

| Path | Notes |
|---|---|
| `GET /health` | unchanged |
| `GET, POST /api/v1/companies` | list / create |
| `GET, PUT, DELETE /api/v1/companies/{id}` | the company resource itself |
| `GET /api/v1/ares/*` | ARES sync lookup by IČO — no company context needed |
| `GET /api/v1/vies-check/*` | VIES VAT ID validation (sync lookup, distinct from the `vies-summary` filing) |
| `GET /api/v1/cnb/*` | CNB exchange-rate sync lookup |
| `GET /api/v1/audit-log` | cross-company trail; accepts optional `?company_id=<id>` query filter to scope the result to a single company |
| `GET, POST /api/v1/backups` | full-DB snapshots |

**Per-company tier** under `/api/v1/companies/{companyID}/`:

contacts, invoices (+ status, send, pdf, isdoc, qr, documents), recurring-invoices, expenses, expense-categories, recurring-expenses, sequences, payment-reminders, settings, imports (fakturoid, ocr), email (send), tax/* (year-settings, prepayments, credits, deductions, income-return, social, health), vat-return, vat-control-statement, vies-summary, investments/* (documents, capital-income, security-transactions), reports/* (dashboard, revenue, expenses, profit, cash-flow, tax-calendar).

### Routing (chi)

```go
r.Route("/api/v1", func(r chi.Router) {
    // Global tier
    r.Get("/health", health)
    r.Route("/companies", companyHandler.GlobalRoutes)       // list, create, {id} CRUD
    r.Route("/ares", aresHandler.Routes)
    r.Route("/cnb", cnbHandler.Routes)
    r.Route("/vies-check", viesCheckHandler.Routes)
    r.Get("/audit-log", auditHandler.List)
    r.Route("/backups", backupHandler.Routes)

    // Per-company tier
    r.Route("/companies/{companyID}", func(r chi.Router) {
        r.Use(companyHandler.WithCompany)                    // resolves & validates {companyID}
        r.Mount("/contacts", contactHandler.Routes())
        r.Mount("/invoices", invoiceHandler.Routes())
        r.Mount("/expenses", expenseHandler.Routes())
        // ...one Mount call per resource...
    })
})
```

### `WithCompany` middleware

Reads `chi.URLParam(r, "companyID")`, parses it as `int64`, looks up the company via `CompanyService.Get(ctx, id)` (which excludes soft-deleted rows), and stores the loaded `*domain.Company` in the request context under a typed key (no string-keyed map). Returns:

- `400 Bad Request` if the path segment isn't a positive integer.
- `404 Not Found` if no company has that ID (or it's soft-deleted).
- On success, downstream handlers retrieve via `company.FromContext(r.Context())` and **also** pass `company.ID` explicitly into service and repository calls.
- On success, the middleware also writes `X-Company-Id: <id>` to the response so the frontend can verify the response matches the still-active company (race-condition detection — see Edge Cases).

### Service & repository signatures

Every per-company service / repository method that today takes `ctx context.Context` gains an explicit `companyID int64` parameter as the next argument:

```go
// before
func (r *ContactRepository) List(ctx context.Context, filter ContactFilter) ([]Contact, error)
// after
func (r *ContactRepository) List(ctx context.Context, companyID int64, filter ContactFilter) ([]Contact, error)
```

Rationale: explicit args win on testability and follow the existing convention. Context is used elsewhere in this codebase for cancellation only, not for ambient state — passing the company in context would be inconsistent and harder to grep for missed call sites.

Repository SQL changes are mechanical: every query gains `AND company_id = ?` in its `WHERE`, every `INSERT` adds `company_id` to the column list. Tests catch any miss because they always seed two companies and verify cross-company isolation.

### Settings handler

`/api/v1/companies/{companyID}/settings` returns and updates only the keys that *remain* per-company in the `settings` table (email templates, PDF defaults, Czech office codes). The 17 identity keys are gone and live on the company resource itself, accessed via `PUT /api/v1/companies/{id}`.

## Frontend

### Current-company store

`frontend/src/lib/stores/currentCompany.svelte.ts`:

```typescript
import { browser } from '$app/environment';

let current = $state<Company | null>(null);
let companies = $state<Company[]>([]);

export const currentCompany = {
    get current() { return current; },
    get companies() { return companies; },
    setCompanies(list: Company[]) { companies = list; },
    select(id: number) {
        current = companies.find(c => c.id === id) ?? null;
        if (browser && current) localStorage.setItem('zfaktury.company', String(id));
    },
};
```

Reactive (`$state`) so any component using `currentCompany.current` re-renders on switch. Persists across page reloads via `localStorage` key `zfaktury.company`.

### Bootstrap flow (root `+layout.svelte`)

1. `onMount`: `GET /api/v1/companies` → populate the store's list.
2. If list is empty → redirect to `/companies/new` (one-time onboarding for fresh installs).
3. Read `localStorage.getItem('zfaktury.company')`. If valid (still in the list) → select it. Else select the first.
4. Render children.

### Header component

`frontend/src/lib/components/CompanyHeader.svelte` mounts in the existing layout above the sidebar/content split. Shows the current company name with a chevron; on click, opens a dropdown listing all (non-deleted) companies with a checkmark next to the active one, plus a "Manage companies →" link to `/companies` and an "+ Add company" link to `/companies/new`. Closes on outside click (existing pattern).

### Typed API client

Per-company methods read `currentCompany.current.id` internally and build the full path. Callers do not pass company IDs explicitly. **The captured id and the response's `X-Company-Id` header are returned to the caller** so write actions can detect when the user has switched companies mid-flight (see Edge Cases):

```typescript
// Caller (read):
await client.invoices.list({ status: 'sent' });

// Caller (write — returns capture so the caller can verify):
const { data, submittedFor, respondedFor } = await client.invoices.create(payload);
if (submittedFor !== respondedFor) {
    // Should never happen — middleware echoes the URL's company id.
}
if (submittedFor !== currentCompany.current?.id) {
    toast(`Saved to ${nameOf(submittedFor)} — you've since switched to ${nameOf(currentCompany.current?.id)}`);
    return; // skip the redirect-to-list path
}

// Inside client.ts:
create(input: NewInvoice) {
    const cid = currentCompany.current?.id;
    if (!cid) throw new NoCompanyError();
    return fetch(`/api/v1/companies/${cid}/invoices`, { method: 'POST', body: ... })
        .then(async (r) => ({
            data: await r.json(),
            submittedFor: cid,
            respondedFor: Number(r.headers.get('X-Company-Id')),
        }));
}
```

Global methods (`client.companies.*`, `client.ares.*`, `client.cnb.*`, `client.backups.*`, `client.audit.*`, `client.viesCheck.*`) stay as plain paths.

### Re-fetching on switch

Each page already follows the CLAUDE.md `onMount` + `$effect`-guarded-with-`mounted` pattern for filter changes. Adding `currentCompany.current?.id` to that effect's reactive deps makes a switch re-fire the data load:

```typescript
let mounted = false;
onMount(() => { loadData(); mounted = true; });
$effect(() => {
    currentCompany.current?.id;
    filterStatus;
    if (!mounted) return;
    loadData();
});
```

### New / changed routes

| Route | Purpose |
|---|---|
| `/companies` | list + manage (CRUD, soft-delete with confirmation) |
| `/companies/new` | create flow with optional ARES auto-fill by IČO; also serves as the empty-state onboarding screen |
| `/companies/[id]` | edit a single company |

All existing routes (`/invoices`, `/expenses`, `/tax`, `/contacts`, etc.) keep their URLs — the active company is implicit (sourced from the store).

## Migration

Single goose migration `025_multi_company.sql`, applied in one transaction. Behavior:

1. **Create `companies` table.**
2. **Seed default company** from existing `settings` keys (only if any settings exist — fresh installs get an empty `companies` table and rely on the onboarding screen).
3. **Partition every per-company table** via `ALTER TABLE … ADD COLUMN company_id INTEGER NOT NULL DEFAULT 1 REFERENCES companies(id)`. SQLite supports inline `REFERENCES` since 3.6.19; the `DEFAULT 1` populates existing rows.
4. **Add `idx_<table>_company` index** for each.
5. **Rebuild `invoice_sequences`** with `UNIQUE(company_id, prefix, year)` via SQLite's rename / create-new / copy / drop dance.
6. **Rebuild parent-child tables for composite-FK targets.** `invoices`, `recurring_invoices`, `expenses` each need `UNIQUE(company_id, id)`. Their children — `invoice_items`, `recurring_invoice_items`, `expense_items`, `vat_return_invoices`, `vat_return_expenses` — get their FK redefined as composite `FOREIGN KEY (company_id, <fk>) REFERENCES <parent>(company_id, id)`. Five composite-FK child rebuilds + three parent unique-index adds. Order matters: parents before children so the FK target exists.
7. **Strip the 17 identity keys from `settings`.**
8. **Gain `company_id` on `settings`**, replace `UNIQUE(key)` with `UNIQUE(company_id, key)`.

Inside goose's BEGIN/COMMIT envelope, a failure rolls back cleanly.

### Down migration

Best-effort reverse: drop the `company_id` columns via SQLite rebuild, drop `companies`, re-insert the 17 keys into `settings` from `companies WHERE id = 1`. **Destructive for users who created more than one company** — only the first company's data survives a downgrade. Documented in the migration's `-- +goose Down` comment and called out in README under "Upgrading".

### App-level invariant check

After `database.Migrate(db)` succeeds at startup, run `company.ValidateInvariants(db)`:

- Every per-company table has zero rows with `NULL` `company_id`.
- All `company_id` values reference an existing (non-deleted-OK-here) row in `companies`.
- `invoice_sequences` has at most one row per `(company_id, prefix, year)`.

Costs three `COUNT(*)`-style queries per table; only runs once per process start; aborts startup with a clear error if any invariant fails.

### Fresh install vs upgrade

- **Fresh install**: empty `settings`, so step 2 inserts nothing. Migration leaves `companies` empty. First API call returns `[]`; the frontend's bootstrap routes to `/companies/new`.
- **Upgrade**: existing settings → default company auto-created with `id = 1` named from `settings.company_name` (fallback `"My Company"`). User can rename via `/companies/[id]` post-upgrade.

## Edge cases & invariants

- **Cross-company FK leak — defense in depth.** An invoice in company A cannot reference a contact / sequence / category from company B. Primary enforcement is the **service layer**: when a handler receives a request scoped to company A, all lookups (`contactRepo.GetByID`, `sequenceRepo.GetActive`, etc.) pass `companyID = A` and the repository's `WHERE company_id = ?` filter naturally rejects company-B IDs as `NotFound`. **Additionally**, for the parent-child relationships that drive financial aggregation (where a leak would silently corrupt totals or VAT filings), the migration adds **composite SQL-level foreign keys** that make a cross-company link physically impossible: `invoice_items.invoice_id`, `recurring_invoice_items.recurring_invoice_id`, `expense_items.expense_id`, `vat_return_invoices.invoice_id`, `vat_return_expenses.expense_id`. The parents (`invoices`, `expenses`, etc.) gain a `UNIQUE(company_id, id)` index to serve as the composite-FK target, and the child tables get composite `FOREIGN KEY (company_id, invoice_id) REFERENCES invoices(company_id, id)`. Lookup-style references (`invoices.contact_id`, `invoices.sequence_id`, `expenses.category_id`) stay as single-column FKs — a bug there surfaces immediately as a wrong-contact/wrong-category in the UI and is caught by the exhaustive cross-company leak-detector test suite (see Testing).
- **Sequence collision.** Two companies legitimately can run `FV2026-0001` — the unique constraint `(company_id, prefix, year)` permits it. Tested in the integration suite.
- **Soft-delete of the last company.** Blocked at the service layer with `domain.ErrLastCompany`.
- **Soft-delete of a non-empty company.** Blocked with `domain.ErrInUse`. User must clear contained records (or soft-delete them) first.
- **Stale localStorage company id.** Bootstrap validates against the fetched company list; if the stored ID is no longer present (or is soft-deleted), it falls back to the first available and overwrites localStorage.
- **Concurrent switching while a request is in flight (read path).** Acceptable: a list response lands while the user is now on another company; the active page's `$effect` is already re-firing for the new company id, so the stale response is harmless — by the time it's rendered, a newer fetch has been initiated. No data corruption — the response was correct for company X; the user is now on company Y; the next render will be correct.
- **Concurrent switching while a write is in flight (write path).** This is the dangerous case: user POSTs an invoice to company A, switches to company B before the response, and would normally land on B's invoice list with no indication that anything was saved to A. The frontend mitigates this by **capturing `companyId` at submit time**, comparing it against `currentCompany.current?.id` when the response arrives, and showing a context-explaining toast ("Saved to *Manas OSVC* — you've since switched to *Manas s.r.o.*") instead of redirecting to the now-wrong list. The captured-vs-active comparison is what the typed API client returns as `submittedFor` (see Frontend). The `X-Company-Id` response header is an additional belt-and-suspenders check that the server actually processed the request for the company the URL named — should never disagree, but the assertion is cheap.
- **ARES auto-fill.** `/companies/new?ico=…` calls the existing `/api/v1/ares` global endpoint and pre-fills the create form. Same flow as today's contact create.

## Testing

| Layer | New tests |
|---|---|
| **Migration — correctness** | `migrations_test.go` exercises (a) a v024 fixture with realistic seed data → applies 025 → asserts exactly one company, all rows backfilled, 17 keys gone, sequences intact, composite FKs in place; (b) an empty v024 → applies 025 → zero companies, zero rows |
| **Migration — production-sized** | A separate slow test (gated behind `-tags integration` so it doesn't run on every CI build) seeds a synthetic v024 DB with ~5,000 invoices, ~10,000 invoice items, ~2,500 contacts, ~5,000 expenses across one default company, then runs migration 025 end-to-end. Asserts: completes under 30 s on commodity hardware, exact row counts unchanged, all composite FK constraints satisfied, no orphan rows |
| **Repository — leak detector (table-driven)** | A single exhaustive test in `internal/repository/leak_detector_test.go` that, for every per-company repo's Get/List/Update/Delete (~80 cases total), seeds the entity in company A and asserts that the same operation invoked with `companyID = B` returns `ErrNotFound` (or zero rows for List). Generated from a table so adding a new entity to the suite is one line. Complements (does not replace) the per-repo `Test*_companyIsolation` tests, which cover the happy path |
| **Service** | `CompanyService.Delete` blocks on non-empty / last-company; companyID threaded through every existing service test |
| **Handler / middleware** | `WithCompany`: 400 on non-numeric, 404 on missing / soft-deleted, success populates context; sample handler test for a per-company route |
| **Frontend component (Vitest)** | `currentCompany` store (persist, restore, clear-stale); `CompanyHeader.svelte` (render, dropdown, select, manage link); empty-state redirect; representative page refetches on `currentCompany` change |
| **Integration** | One end-to-end flow in `tests/integration/`: two companies, contacts, invoices, cross-company rejection, sequence collision, delete protection, soft-delete success after cleanup |
| **Coverage gate** | Unchanged at 80% backend minimum; pre-commit hook blocks regressions |
| **Manual** | One-page upgrade walkthrough in the PR description (upgrade from v0.0.50 on a real-data DB → verify default company → add second company → switch → totals change → cross-company invoice rejected) |

## PR scope & sequencing

**One bundled PR** rather than phased. The migration is atomic; any partial intermediate state ("invoices know their company, expenses don't") would be more confusing to reason about than no state. Roughly ~30 tables and ~60 handler files are touched, plus the frontend; the branch will be long-lived (~3–5 weeks of focused work). Inside the branch, commits stay surgical and well-scoped (typically one entity or one concern per commit) so the diff is navigable when the upstream maintainer reviews.

## Review notes (v2)

The v1 of this spec was reviewed by Gemini CLI (Architect role) on 2026-05-24. Ten items raised; six incorporated, four pushed back on with technical reasoning. Recorded here so the next reviewer can see the trade-offs rather than re-litigate them.

### Incorporated

| # | Item | Where it landed |
|---|---|---|
| 2.1 | Composite SQL-level FKs for cross-company safety, scoped to parent-child aggregation paths (not every relationship) | Edge Cases → "Cross-company FK leak — defense in depth"; Migration → step 6 |
| 2.2 | Audit log `?company_id=` filter | API surface → global tier table |
| 2.3 | Capture `companyId` at submit-time and surface a context-explaining toast when the user has switched companies before the response lands | Edge Cases → "Concurrent switching while a write is in flight"; Frontend → typed client snippet |
| 3.1b | Exhaustive leak-detector test that table-driven-checks every per-company Get/List/Update/Delete with a wrong `companyID` | Testing table |
| 4.1 | Production-sized migration test on a synthetic ~5k-invoice DB | Testing table |
| 4.3 | `X-Company-Id` response header set by the `WithCompany` middleware on every per-company response | API surface → middleware; Frontend → typed client snippet |

### Pushed back on (with reasoning)

| # | Item | Reasoning |
|---|---|---|
| 2.1 (broad form) | Composite FKs on *every* per-company relationship, not just parent-child aggregation paths | Adding composite FKs to existing tables requires the full rename → recreate → copy → drop dance because composite FKs are table-level (no `ALTER TABLE ADD CONSTRAINT` in SQLite). Doing this for ~25 child tables means ~25 table rebuilds in the migration, each a non-trivial risk on production DBs. Service-layer enforcement plus the leak-detector test catches the same bugs at much lower cost; composite FKs are reserved for the relationships where a silent leak would corrupt financial aggregations |
| 2.4 | "Email templates may contain stale hardcoded company name after migration" | Verified false premise: `internal/service/email/templates.go` defines email bodies as Go `text/html/template` constants in code, not strings in the settings table. They already render with placeholders like `{{.SenderName}}`, `{{.BankAccount}}`, `{{.CustomerName}}` populated at send time. No risk of stale baked-in company data |
| 2.5 | Add a CLI `purge` / `HardDelete` for cleaning up old companies | YAGNI. zfaktury is a local single-user binary on a SQLite file the user owns. If true deletion is needed, `sqlite3 zfaktury.db` is one command. Adding a destructive CLI surface for a niche case at v1 is feature creep; can be added later if real demand emerges |
| 3.1a | Adopt a `BaseRepository` pattern to centralize the `company_id = ?` clause | Against existing code style — each repo in this codebase is its own concrete struct with explicit SQL, sharing only narrow helpers (`scanInvoiceRow`). Introducing a base class would be a stylistic refactor out of scope for this PR. The repetition is mechanical and the leak detector catches misses |
| 3.2 | Return `412 Precondition Failed` / `428 Precondition Required` for missing-company so the frontend knows to show onboarding | Unnecessary: the frontend bootstrap calls `GET /api/v1/companies` (global) before any per-company URL is built, and routes to `/companies/new` on empty result. No per-company URL is ever requested without a known-valid id, so the missing-company case at the per-company tier is genuinely "resource not found" — 404 is RFC-correct here |

## Open questions / future work

- **Per-company export.** Out of scope for this PR. A future feature could add `POST /api/v1/companies/{id}/export` producing a per-company ISDOC bundle + PDF archive + JSON dump.
- **Switching to multi-user.** Out of scope. If/when introduced, "user" sits above "company" with a many-to-many user↔company relationship; this design's per-company partitioning is the right base for that.
- **Importing from a different multi-company tool.** Out of scope. The existing Fakturoid importer maps to one company; an upgrade could expose a company selector during import.
- **Performance.** With small expected N (2–5 companies, thousands of invoices each), the new `company_id` indexes are sufficient. No additional sharding considered.

## Glossary

| Term | Meaning |
|---|---|
| OSVČ | "Osoba samostatně výdělečně činná" — Czech sole proprietor / self-employed person |
| s.r.o. | "Společnost s ručením omezeným" — Czech LLC equivalent |
| IČO | "Identifikační číslo osoby" — 8-digit Czech business ID |
| DIČ | "Daňové identifikační číslo" — Czech VAT ID, format `CZ` + 8–10 digits |
| ISDOC | Czech XML invoice exchange format |
| ARES | "Administrativní registr ekonomických subjektů" — Czech business registry; public sync API for looking up company info by IČO |
| VIES | EU VAT validation service; this codebase uses it in two distinct senses — `vies-check` (sync lookup) and `vies-summary` (the souhrnné hlášení tax filing) |
| CNB | Česká národní banka — Czech central bank, source of daily exchange rate fixings |
| Goose | Go database migration tool used throughout the codebase |
