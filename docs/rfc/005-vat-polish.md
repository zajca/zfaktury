# RFC-005: VAT Filings Polish

**Status:** Done
**Date:** 2026-03-10

## Summary

RFC-004 delivered a working VAT filings implementation (DPH priznani, kontrolni hlaseni, souhrnne hlaseni). This RFC addresses technical debt, code quality, UX, and hardening issues identified during code review, security review, and UX review of the RFC-004 implementation.

## 1. Frontend: Consolidate API Clients

### Problem

`vat.ts`, `vat-control.ts`, and `vat-vies.ts` each define their own `fetch` wrappers, `ApiError` classes, and `TaxPeriod` interface. This violates the project convention where all other pages import from `$lib/api/client.ts`.

- `vat.ts` uses a simple `throw new Error(await res.text())` — raw HTML/JSON may leak to users on non-text responses
- `vat-control.ts` and `vat-vies.ts` have a more sophisticated `ApiError` class but duplicate it between files
- `TaxPeriod` is defined 3x

### Fix

Move VAT types and API methods into `$lib/api/client.ts` (or extend the shared `get`/`post`/`del` helpers). Remove the standalone files. Update all imports across VAT pages.

**Files:** `frontend/src/lib/api/vat.ts`, `vat-control.ts`, `vat-vies.ts`, `client.ts`

## 2. Frontend: Extract Shared VAT Status Utilities

### Problem

`vatStatusLabels`, `vatStatusColors`, and `filingTypeLabels` maps are copy-pasted across 4+ files:

- `routes/vat/+page.svelte`
- `routes/vat/returns/[id]/+page.svelte`
- `routes/vat/control/[id]/+page.svelte`
- `routes/vat/vies/[id]/+page.svelte`

This mirrors the pattern already solved for invoices via `$lib/utils/invoice.ts`.

### Fix

Create `$lib/utils/vat.ts` with shared `vatStatusLabels`, `vatStatusColors`, `filingTypeLabels` maps. Import everywhere.

**Files:** New `frontend/src/lib/utils/vat.ts`, update all VAT page imports.

## 3. Frontend: Czech Diacritics

### Problem

All Czech UI text lacks diacritics (hacky/carky). Examples:

- "Unor" → "Únor", "Brezen" → "Březen", "Duben" → "Duben", "Kveten" → "Květen"
- "Cerven" → "Červen", "Cervenec" → "Červenec", "Zari" → "Září", "Rijen" → "Říjen"
- "Radne" → "Řádné", "Nasledne" → "Následné", "Opravne" → "Opravné"
- "Nacitani" → "Načítání", "Nepodarilo se" → "Nepodařilo se"
- "Koncept" → "Koncept" (ok), "Pripraveno" → "Připraveno", "Podano" → "Podáno"
- "Zadna DPH priznani" → "Žádná DPH přiznání"

This is a systematic issue across all VAT pages. Other existing pages (invoices, expenses) may have the same issue — scope this to VAT pages first.

### Fix

Audit all Czech text strings in `frontend/src/routes/vat/**/*.svelte` and shared utils. Replace ASCII approximations with proper Unicode diacritics.

## 4. Frontend: VAT Return Create — Month vs Quarter Validation

### Problem

On `/vat/returns/new/+page.svelte`, the user can independently select both a month AND a quarter. There is no validation preventing conflicting combinations (e.g., month=3 and quarter=2).

### Fix

Add a radio toggle "Mesicni / Ctvrtletni" that shows only the relevant select. Or add validation that rejects selecting both month and quarter simultaneously. Consider pulling the `vat_filing_frequency` setting to auto-select the mode.

**File:** `frontend/src/routes/vat/returns/new/+page.svelte`

## 5. Backend: Input Validation Hardening

### Problem (LOW severity from security review)

- `VATReturnService.Create` checks `year != 0` but allows absurd values like 9999 or negative years
- No month bounds check (1-12) in VAT return creation
- `filing_type` is not validated against an allowlist — unknown values like `"hacked"` are stored verbatim

### Fix

Add bounds validation in all three services:

```go
// Year bounds
if vr.Period.Year < 2000 || vr.Period.Year > 2100 {
    return fmt.Errorf("year out of valid range: %w", domain.ErrInvalidInput)
}

// Month bounds (if provided)
if vr.Period.Month != 0 && (vr.Period.Month < 1 || vr.Period.Month > 12) {
    return fmt.Errorf("month must be 1-12: %w", domain.ErrInvalidInput)
}

// Filing type allowlist
switch vr.FilingType {
case domain.FilingTypeRegular, domain.FilingTypeCorrective, domain.FilingTypeSupplementary:
    // ok
default:
    return fmt.Errorf("invalid filing_type: %w", domain.ErrInvalidInput)
}
```

**Files:** `internal/service/vat_return_svc.go`, `vat_control_svc.go`, `vies_svc.go`

## 6. Backend: Content-Disposition Header Safety

### Problem (LOW severity from security review)

XML download handlers build Content-Disposition via string concatenation:

```go
w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
```

Currently safe because filenames are built from integers only. But the pattern is fragile — if any string field is ever used, it becomes header injection.

### Fix

Use `mime.FormatMediaType` for proper RFC 6266 encoding:

```go
import "mime"
params := map[string]string{"filename": filename}
w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", params))
```

**Files:** `internal/handler/vat_control_handler.go`, `vies_handler.go`, `vat_return_handler.go`

## 7. Frontend: Dashboard Loading Optimization

### Problem (LOW priority)

Changing the year on the VAT dashboard triggers `Promise.all` for all 3 endpoints (vat-returns, control-statements, vies-summaries), even though the user only sees one tab at a time.

### Fix

Lazy-load tab data: only fetch the active tab's data on year change. Fetch other tabs when the user switches to them. Cache fetched data per year to avoid redundant calls.

**File:** `frontend/src/routes/vat/+page.svelte`

## Priority

| # | Item | Effort | Impact |
|---|------|--------|--------|
| 1 | Consolidate API clients | Medium | High — reduces duplication, consistent error handling |
| 2 | Extract VAT status utils | Small | Medium — DRY, easier to update labels |
| 3 | Czech diacritics | Medium | High — visible to every user |
| 4 | Month/quarter validation | Small | Medium — prevents invalid data |
| 5 | Input validation hardening | Small | Low — defense in depth |
| 6 | Content-Disposition safety | Small | Low — future-proofing |
| 7 | Dashboard lazy loading | Medium | Low — performance optimization |
