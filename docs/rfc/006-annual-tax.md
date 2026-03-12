# RFC-006: Annual Tax (DPFO + CSSZ + ZP)

**Status:** Done
**Date:** 2026-03-11

## Summary

Implements the three annual tax filings required of Czech sole proprietors (OSVC): income tax return (DPFO), social insurance overview (Prehled OSVC pro CSSZ), and health insurance overview (Prehled OSVC pro ZP). All three share a common annual base (revenue from invoices minus expenses) and follow the same draft -> filed lifecycle. DPFO and CSSZ include XML generation for electronic submission; ZP defers XML until the unified SZP-VZP format is published (expected July 2026).

## Background

After each tax year, Czech OSVC must file:

1. **DPFO (Danove priznani fyzickych osob)** — income tax return to Financni sprava via EPO portal. Progressive tax rates: 15% up to threshold, 23% above. Tax credits (sleva na poplatnika, manzela, invalidita, student) and child benefit (danove zvyhodneni) reduce final tax.
2. **Prehled OSVC pro CSSZ** — social insurance overview to local OSSZ. Assessment base = 50% of tax base, floored to annual minimum. Rate: 29.2%.
3. **Prehled OSVC pro ZP** — health insurance overview to health insurance company. Assessment base = 50% of tax base, floored to annual minimum. Rate: 13.5%.

All three forms use the same revenue and expense base. Expenses can be actual or flat-rate (pausalni vydaje) at 30/40/60/80% of revenue with per-percent annual caps.

## Implementation

### Database (Migration 015)

Three main tables plus two junction tables:

| Table | Purpose |
|-------|---------|
| `income_tax_returns` | DPFO with progressive tax fields, credits, child benefit, prepayments |
| `income_tax_return_invoices` | Junction: DPFO -> invoices (audit trail) |
| `income_tax_return_expenses` | Junction: DPFO -> expenses (audit trail) |
| `social_insurance_overviews` | CSSZ overview with assessment base, insurance rate, prepayments |
| `health_insurance_overviews` | ZP overview with same structure as CSSZ |

All amount columns INTEGER (halere). UNIQUE constraint on (year, filing_type) per table.

### Domain Types (`internal/domain/tax.go`)

- `IncomeTaxReturn` — year, filing type, section 7 fields (revenue, actual expenses, flat-rate percent/amount, used expenses), tax base (raw + rounded to 100 CZK), progressive tax (15%/23%), credits (basic, spouse, disability, student), child benefit, prepayments, tax due
- `SocialInsuranceOverview` — year, filing type, revenue, expenses, tax base, assessment base (50%), min/final assessment base, insurance rate (permille), total insurance, prepayments, difference, new monthly prepayment
- `HealthInsuranceOverview` — same structure as social insurance

### Shared Calculation Helpers

**`internal/service/annual_tax_base.go`:**
- `CalculateAnnualBase(ctx, invoiceRepo, expenseRepo, year)` — computes annual revenue (from sent/paid/overdue invoices with delivery date in year) and expenses (from tax-reviewed expenses with issue date in year), returns `AnnualTaxBase{Revenue, Expenses, InvoiceIDs, ExpenseIDs}`

**`internal/service/annual_tax_constants.go`:**
- `GetTaxConstants(year)` — returns `TaxYearConstants` with year-specific values: progressive threshold, basic credit, social/health minimum monthly bases, flat-rate caps, insurance rates
- Hardcoded for years 2024, 2025, 2026; returns error for unknown years

### Tax Calculation Logic

**DPFO (`income_tax_return_svc.go` Recalculate):**
1. Get annual base (revenue + expenses from invoices/expenses)
2. Read `flat_rate_percent` from settings; if > 0, compute flat-rate amount with cap
3. `taxBase = revenue - usedExpenses` (clamped to 0)
4. `taxBaseRounded = roundDown100CZK(taxBase)`
5. Progressive tax: 15% up to threshold, 23% above
6. Apply credits: basic (30,840 CZK for 2024) + spouse + disability + student
7. Apply child benefit (can make result negative = tax bonus)
8. Subtract prepayments from settings -> tax due
9. Link source invoices and expenses

**CSSZ (`social_insurance_svc.go` Recalculate):**
1. Same annual base
2. `assessmentBase = taxBase / 2`
3. `minBase = constants.SocialMinMonthly * 12`
4. `finalBase = max(assessmentBase, minBase)`
5. `totalInsurance = finalBase * 292 / 1000`
6. `difference = totalInsurance - prepayments`
7. `newMonthlyPrepay = ceil(totalInsurance / 12)` rounded up to whole CZK

**ZP (`health_insurance_svc.go` Recalculate):**
Same pattern as CSSZ with 13.5% rate and different minimum base.

### Settings Keys

New keys added to `knownKeys` in settings service:

| Key | Purpose |
|-----|---------|
| `flat_rate_percent` | 0/30/40/60/80 — flat-rate expense percentage |
| `tax_prepayments` | Annual income tax prepayments (halere) |
| `social_prepayments` | Annual social insurance prepayments (halere) |
| `health_prepayments` | Annual health insurance prepayments (halere) |
| `financni_urad_code` | Tax office code (for DPFO XML) |
| `cssz_code` | Local OSSZ code (for CSSZ XML) |
| `health_insurance_code` | Health insurance company code |

### XML Generators (`internal/annualtaxxml/`)

**DPFO XML (`income_tax_gen.go` + `income_tax_types.go`):**
- Root: `<Pisemnost><DPFDP5>` with `VetaD` (all tax computation attributes) and `VetaP` (taxpayer identification)
- VetaD attributes: `rok`, `dap_typ`, `c_ufo_cil`, `kc_zd7` (section 7 base), `pr_zd7` (revenue), `vy_zd7` (expenses), `kc_zakldan23` (23% base), `kc_zdzaokr` (rounded base), `da_slezap` (total tax), credits, prepayments, final due
- VetaP attributes: taxpayer name, birth number, DIC, address

**CSSZ XML (`social_insurance_gen.go` + `social_insurance_types.go`):**
- Root: `<OSVC xmlns="http://schemas.cssz.cz/OSVC2025">` with version="1.0"
- Sections: VENDOR, SENDER, PREHLEDOSVC (main), CLIENT (taxpayer info), PVV (income/expense overview with HV pairs), ZAL (advance payments), PRE (overpayment return), PRIZN (declaration flags)
- PVV fields: `rdza` (revenue/expenses), `vvz`/`dvz` (assessment base), `poj` (insurance), `zal` (prepayments), `ned` (difference)

**ZP XML:** Deferred — no stable public XSD. Health insurance overview has full calculation and CRUD but XML generation returns "not implemented" error.

**Shared helpers (`common.go`):**
- `ToWholeCZK(Amount) int64` — converts halere to whole CZK
- `DPFOFilingTypeCode(string) string` — maps filing type to DPFO codes (B/O/D)
- `CSSZFilingTypeCode(string) string` — maps filing type to CSSZ codes (N/O/Z)

### API Endpoints

All under `/api/v1/`, each with identical endpoint pattern:

| Route group | Endpoints |
|-------------|-----------|
| `/income-tax-returns` | POST `/`, GET `/`, GET `/{id}`, DELETE `/{id}`, POST `/{id}/recalculate`, POST `/{id}/generate-xml`, GET `/{id}/xml`, POST `/{id}/mark-filed` |
| `/social-insurance` | Same pattern |
| `/health-insurance` | Same pattern (generate-xml returns error until ZP XML is implemented) |

### Frontend

**Dashboard (`/tax`):**
- Year selector (defaults to previous year) with arrow navigation
- Three cards: DPFO, CSSZ, ZP — each shows status badge, key figures, or "Create" button if empty

**Create pages:**
- `/tax/income/new`, `/tax/social/new`, `/tax/health/new` — year + filing type form

**Detail pages:**
- `/tax/income/[id]` — full DPFO detail: revenue/expenses section, flat-rate info, tax base, progressive tax breakdown, credits, child benefit, prepayments, final due. Action buttons: Recalculate, Generate XML, Download XML, Mark Filed, Delete
- `/tax/social/[id]` — CSSZ detail: revenue/expenses, assessment base (50%, minimum, final), insurance calculation, prepayments, difference, new monthly prepayment
- `/tax/health/[id]` — same as CSSZ, XML buttons disabled with explanatory note

**Settings page additions:**
- New "Danove nastaveni" card with flat-rate percent dropdown, prepayment inputs, office codes

**Navigation:**
- "Dan z prijmu" nav item added under "Ucetnictvi" section

**Help content:**
- New `rocni-dane` help topic with simple and legal descriptions

### Tests

- Repository tests with in-memory SQLite for all three entities
- Service tests for calculation correctness (flat-rate caps, progressive tax, min bases, credits, insurance rates)
- Handler tests for HTTP status codes and error mapping
- Frontend vitest: help-content test updated for new topic

## Key Files

| Purpose | Path |
|---------|------|
| Domain types | `internal/domain/tax.go` |
| Migration | `internal/database/migrations/015_annual_tax.sql` |
| Shared base calc | `internal/service/annual_tax_base.go` |
| Tax constants | `internal/service/annual_tax_constants.go` |
| DPFO service | `internal/service/income_tax_return_svc.go` |
| CSSZ service | `internal/service/social_insurance_svc.go` |
| ZP service | `internal/service/health_insurance_svc.go` |
| DPFO repo | `internal/repository/income_tax_return_repo.go` |
| CSSZ repo | `internal/repository/social_insurance_repo.go` |
| ZP repo | `internal/repository/health_insurance_repo.go` |
| XML generators | `internal/annualtaxxml/*.go` |
| Handlers | `internal/handler/income_tax_handler.go`, `social_insurance_handler.go`, `health_insurance_handler.go` |
| Frontend pages | `frontend/src/routes/tax/**` |
| API client types | `frontend/src/lib/api/client.ts` |
| Settings page | `frontend/src/routes/settings/+page.svelte` |

## Out of Scope

- Automatic filing submission to EPO/ePortal (manual XML download only)
- ZP XML generation (deferred to post-July 2026 when SZP-VZP format is published)
- Multiple children configuration UI (child benefit is a manual input for now)
- Spouse income/disability details (credits are manual inputs)
- Data box (datova schranka) integration
