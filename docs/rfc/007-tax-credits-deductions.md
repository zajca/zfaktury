# RFC-007: Tax Credits, Deductions & AI Document Extraction

**Status:** Done
**Date:** 2026-03-12

## Summary

Extends the annual tax system (RFC-006) with user-manageable tax credits (slevy na dani), tax deductions (nezdanitelne casti zakladu dane), and AI-powered extraction from uploaded proof documents. Credits reduce the calculated tax amount; deductions reduce the tax base before rounding.

## Background

Czech sole proprietors can claim several categories of tax relief:

1. **Tax credits (slevy na dani)** reduce the final tax amount:
   - Basic taxpayer credit (sleva na poplatnika) — already in RFC-006
   - Spouse credit (sleva na manzela/ku) — if spouse income < 68,000 CZK
   - Disability credit — 1st/2nd degree, 3rd degree, ZTP/P holder
   - Student credit — proportional by months of study

2. **Child benefit (danove zvyhodneni)** reduces tax or generates a bonus:
   - Per-child amount depends on order (1st, 2nd, 3rd+)
   - Doubled for ZTP/P children
   - Proportional by months claimed

3. **Tax deductions (nezdanitelne casti)** reduce the tax base:
   - Mortgage interest (max 150,000 CZK/year for contracts from 2021+)
   - Life insurance (max 24,000 CZK)
   - Pension savings (max 24,000 CZK)
   - Donations (max 15% of tax base)
   - Union dues (max 3,000 CZK)

Each deduction category requires proof documents (bank statements, insurance confirmations). AI extraction from these documents speeds up data entry.

## Implementation

### Database (Migration 017)

Five new tables plus one ALTER:

| Table | Purpose |
|-------|---------|
| `tax_spouse_credits` | One per year, spouse info + income for credit eligibility |
| `tax_child_credits` | N per year, child info + order + months for benefit calculation |
| `tax_personal_credits` | One per year (year as PK), student/disability flags |
| `tax_deductions` | N per year per category, claimed/max/allowed amounts |
| `tax_deduction_documents` | Proof documents linked to deductions, with AI extraction fields |

ALTER `income_tax_returns` adds `total_deductions` column to store applied deductions.

### Domain Model

- `TaxSpouseCredit`, `TaxChildCredit`, `TaxPersonalCredits` — credit entities
- `TaxDeduction`, `TaxDeductionDocument` — deduction entities with proof docs
- `TaxExtractionResult` — AI extraction output (amount, year, confidence)

### Recalculate Flow Change

The DPFO recalculation (`IncomeTaxReturnService.Recalculate`) was modified:

```
Step 5:  taxBase = revenue - expenses
Step 5b: load deductions via TaxCreditsService.ComputeDeductions(year, taxBase)
Step 5c: taxBase -= totalDeductions (floor at 0)
Step 6:  round to 100 CZK
Step 7:  progressive tax (15% / 23%)
Step 8:  load credits via TaxCreditsService.ComputeCredits(year)
         -> spouse, disability, student from tax_credits tables
Step 8b: child benefit from TaxCreditsService.ComputeChildBenefit(year)
Step 9:  prepayments
```

Key: deductions reduce tax BASE (before rounding), credits reduce TAX (after calculation).

### Services

- **TaxCreditsService** — CRUD for all credit/deduction entities + computation methods:
  - `ComputeCredits` — spouse (income threshold), disability (3 levels), student (proportional)
  - `ComputeChildBenefit` — per-child by order, proportional by months, ZTP doubling
  - `ComputeDeductions` — applies statutory caps per category
  - `CopyFromYear` — copies credits/deductions structure to a new year
- **TaxDeductionDocumentService** — upload/download/delete proof documents
- **TaxDocumentExtractionService** — AI extraction via OCR provider

### API Endpoints

Credits:
- `GET /api/v1/tax-credits/{year}` — full summary with computed amounts
- `PUT/DELETE /api/v1/tax-credits/{year}/spouse` — manage spouse credit
- `GET/POST /api/v1/tax-credits/{year}/children` — list/add children
- `PUT/DELETE /api/v1/tax-credits/{year}/children/{id}` — manage child
- `PUT /api/v1/tax-credits/{year}/personal` — student/disability
- `POST /api/v1/tax-credits/{year}/copy-from/{sourceYear}` — copy from previous year

Deductions:
- `GET/POST /api/v1/tax-deductions/{year}` — list/create deductions
- `PUT/DELETE /api/v1/tax-deductions/{year}/{id}` — manage deduction
- `POST/GET /api/v1/tax-deductions/{year}/{id}/documents` — upload/list docs
- `DELETE/GET /api/v1/tax-deduction-documents/{id}` — delete/download doc
- `POST /api/v1/tax-deduction-documents/{id}/extract` — AI extract amount

### Frontend

New page at `/tax/credits` with four sections:
1. **Osobni slevy** — student checkbox + months, disability level select
2. **Manzel/ka** — name, birth number, income, ZTP checkbox
3. **Deti** — add/edit/remove children with order, months, ZTP
4. **Odpocty** — deductions by category with document upload and AI extraction

Year selector with copy-from-previous-year button. Sidebar nav item "Slevy a odpocty" added under Ucetnictvi section.

### Tax Constants

Added `DisabilityZTPP` constant (16,140 CZK) to `TaxYearConstants` for ZTP/P holders, alongside existing `DisabilityCredit1` (2,520 CZK) and `DisabilityCredit3` (5,040 CZK).

## Files Changed

### New files
- `internal/domain/tax_credits.go`, `tax_deductions.go`
- `internal/database/migrations/017_tax_credits_deductions.sql`
- `internal/repository/tax_spouse_credit_repo.go`, `tax_child_credit_repo.go`, `tax_personal_credits_repo.go`, `tax_deduction_repo.go`, `tax_deduction_document_repo.go`
- `internal/service/tax_credits_svc.go`, `tax_deduction_document_svc.go`, `tax_document_extraction_svc.go`
- `internal/handler/tax_credits_handler.go`, `tax_deductions_handler.go`
- `frontend/src/routes/tax/credits/+page.svelte`

### Modified files
- `internal/domain/annual_tax.go` — added TotalDeductions field
- `internal/repository/interfaces.go` — 5 new repo interfaces
- `internal/repository/income_tax_return_repo.go` — total_deductions column
- `internal/service/income_tax_return_svc.go` — modified Recalculate flow
- `internal/service/annual_tax_constants.go` — added DisabilityZTPP constant
- `internal/handler/router.go` — mounted new routes
- `internal/cli/serve.go` — wired new dependencies
- `frontend/src/lib/api/client.ts` — new types and API methods
- `frontend/src/lib/components/Layout.svelte` — sidebar nav item
