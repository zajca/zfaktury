# RFC-004: VAT Filings (DPH, Kontrolni hlaseni, Souhrnne hlaseni)

**Status:** Done
**Date:** 2026-03-10

## Summary

Implements the three Czech VAT filing forms required of VAT-registered OSVC: DPH return (Priznani k DPH), Control Statement (Kontrolni hlaseni), and VIES Recapitulative Summary (Souhrnne hlaseni). Each form is calculated from existing invoices and expenses, generates EPO-compatible XML for electronic submission, and follows a draft -> filed lifecycle.

## Background

Czech VAT-registered sole proprietors must submit:

1. **DPH Return (Priznani k DPH)** — monthly or quarterly, summarises output and input VAT at 21% and 12% rates, reverse charge, and net VAT due/refund
2. **Control Statement (Kontrolni hlaseni)** — monthly, itemised transaction listing with partner DIC, document numbers, DPPD dates; transactions above 10,000 CZK threshold reported individually (sections A4/B2), below threshold aggregated (sections A5/B3)
3. **VIES Summary (Souhrnne hlaseni)** — quarterly, summarises supplies of goods and services to EU partners by country code

All three share the same filing type (regular/corrective/supplementary) and status (draft/ready/filed) workflow, and link back to source invoices and expenses for audit trail.

## Implementation

### Database (Migration 014)

Six tables:

| Table | Purpose |
|-------|---------|
| `vat_returns` | Main DPH return with output/input VAT at 21% & 12%, reverse charge, net VAT, XML storage |
| `vat_return_invoices` | Junction: VAT return -> invoices (audit trail) |
| `vat_return_expenses` | Junction: VAT return -> expenses (audit trail) |
| `vat_control_statements` | Monthly control statements with XML storage |
| `vat_control_statement_lines` | Per-transaction lines: partner DIC, doc number, DPPD, amount, rate, section |
| `vies_summaries` | Quarterly EU summaries with XML storage |
| `vies_summary_lines` | Per-EU-partner lines: country code, DIC, service code, amounts |

### Domain Types

All in `internal/domain/tax.go`:

- `VATReturn` — output/input bases and amounts at 21%/12%, reverse charge, net VAT calculation
- `VATControlStatement` + `VATControlStatementLine` — monthly control statement with section classification (A4, A5, B2, B3)
- `VIESSummary` + `VIESSummaryLine` — quarterly EU partner summaries
- `TaxPeriod` — unified period struct supporting monthly (month 1-12) and quarterly (quarter 1-4) filings
- Constants: `FilingTypeRegular`/`Corrective`/`Supplementary`, `FilingStatusDraft`/`Ready`/`Filed`, `ControlStatementThreshold` (10,000 CZK = 1,000,000 halere)

### Repository Layer

Three repositories following the standard pattern (scan helpers, parameterised SQL, soft-delete awareness):

- `VATReturnRepo` — CRUD + `GetByPeriod`, `LinkInvoices`, `LinkExpenses`
- `VATControlStatementRepo` — CRUD + lines management, `GetByPeriod`
- `VIESSummaryRepo` — CRUD + lines management, `GetByPeriod`

### Service Layer

Three services, each with: `Create`, `GetByID`, `List`, `Delete`, `Recalculate`, `GenerateXML`, `GetXMLData`, `MarkFiled`.

**Recalculate logic:**
- DPH return: sums output VAT from invoices by rate (21%, 12%) and input VAT from tax-reviewed expenses, computes reverse charge from applicable invoices, calculates net VAT
- Control statement: classifies each transaction into A4/A5 (output) or B2/B3 (input) based on 10,000 CZK threshold, includes partner DIC and document numbers
- VIES summary: aggregates EU-destination invoices by partner country and DIC, classifies goods vs services

### XML Generators (`internal/vatxml/`)

EPO-compatible XML for all three forms:

| File | Form | Root Element |
|------|------|-------------|
| `vat_return_gen.go` + `vat_return_types.go` | DPH | `<Pisemnost><DPHDAP3>` with Veta1-6 |
| `control_statement_gen.go` + `control_statement_types.go` | KH | Control statement EPO format |
| `vies_gen.go` + `vies_types.go` | SH | VIES summary EPO format |

### API Endpoints

All under `/api/v1/`, each with identical endpoint pattern:

| Route group | Endpoints |
|-------------|-----------|
| `/vat-returns` | POST `/`, GET `/`, GET `/{id}`, DELETE `/{id}`, POST `/{id}/recalculate`, POST `/{id}/generate-xml`, GET `/{id}/xml`, POST `/{id}/mark-filed` |
| `/vat-control-statements` | Same pattern |
| `/vies-summaries` | Same pattern |

### Frontend

| Route | Purpose |
|-------|---------|
| `/vat` | Dashboard: year selector, monthly/quarterly grid, status badges for all three filing types |
| `/vat/returns/new` | Create DPH return (year, month/quarter, filing type) |
| `/vat/returns/[id]` | DPH return detail: all VAT fields, action buttons |
| `/vat/control/new` | Create control statement |
| `/vat/control/[id]` | Control statement detail with transaction lines |
| `/vat/vies/new` | Create VIES summary |
| `/vat/vies/[id]` | VIES summary detail with partner lines |

Shared utilities in `$lib/utils/vat.ts`: `vatStatusLabels`, `vatStatusColors`, `filingTypeLabels`.

### Tests

- Repository tests with in-memory SQLite
- Service tests for calculation correctness (VAT rates, thresholds, reverse charge)
- Handler tests for HTTP status codes and error mapping
- XML generator tests validating output structure
- Frontend vitest page tests for all routes

## Key Files

| Purpose | Path |
|---------|------|
| Domain types | `internal/domain/tax.go` |
| Migration | `internal/database/migrations/014_vat_filings.sql` |
| VAT return repo | `internal/repository/vat_return_repo.go` |
| Control statement repo | `internal/repository/vat_control_repo.go` |
| VIES repo | `internal/repository/vies_repo.go` |
| VAT return service | `internal/service/vat_return_svc.go` |
| Control statement service | `internal/service/vat_control_svc.go` |
| VIES service | `internal/service/vies_svc.go` |
| XML generators | `internal/vatxml/*.go` |
| Handlers | `internal/handler/vat_return_handler.go`, `vat_control_handler.go`, `vies_handler.go` |
| Frontend pages | `frontend/src/routes/vat/**` |
| Shared utils | `frontend/src/lib/utils/vat.ts` |

## Out of Scope

- Annual income tax (DPFO) — RFC-006
- Social and health insurance overviews — RFC-006
- Automatic filing submission to EPO portal (manual XML download only)
