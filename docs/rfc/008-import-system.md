# RFC-008: Import System & Fakturoid Integration

**Status:** Done
**Date:** 2026-03-12

## Summary

Extensible data import system for migrating data from external invoicing services into ZFaktury. First provider: Fakturoid. Architecture designed for easy addition of future providers (Pohoda, iDoklad, etc.).

## Background

ZFaktury targets Czech sole proprietors who are often migrating from existing invoicing tools. The most common migration path is from Fakturoid, a popular Czech invoicing SaaS.

Key requirements:
- **One-time migration** — import is not a sync, it runs once to bring historical data in
- **Credentials in UI** — users should not need to edit config files for a one-time operation
- **Duplicate detection** — avoid creating duplicate contacts/invoices/expenses on re-import
- **Extensible** — adding a new import source should be isolated and follow a clear pattern

## Design

### API Structure

Each import provider gets its own route namespace under `/api/v1/import/`:

```
POST /api/v1/import/fakturoid/import   { slug, email, api_token }  →  ImportResult
POST /api/v1/import/pohoda/import      { ... }                     →  ImportResult  (future)
```

Credentials are passed per-request in the POST body — never stored in config or database.

### Shared Import Result

All providers return the same result shape:

```go
type FakturoidImportResult struct {
    ContactsCreated int
    ContactsSkipped int
    InvoicesCreated int
    InvoicesSkipped int
    ExpensesCreated int
    ExpensesSkipped int
    Errors          []string
}
```

### Duplicate Detection

Before importing, each entity is checked against:

1. **Import log** — `fakturoid_import_log` table tracks previously imported entities by `(entity_type, fakturoid_id)` pair
2. **Business key matching** — contacts by ICO or exact name, invoices by invoice number, expenses by expense number + vendor + date

Duplicates are silently skipped (counted in `*Skipped` fields). Import log entries are created for each successfully imported entity.

### Import Order

Entities are imported in dependency order:
1. **Contacts** first — creates the subject-to-contact ID mapping
2. **Invoices** second — resolves `customer_id` via the mapping
3. **Expenses** third — resolves `vendor_id` via the mapping

### Fakturoid API Client

`internal/fakturoid/client.go` — HTTP client for Fakturoid API v3:
- Basic auth with email + API token
- Automatic pagination (all pages fetched)
- Rate limiting: 700ms delay between pages (~85 req/min, under Fakturoid's 100/min limit)
- Context-aware (cancellation propagates)

### Frontend Flow

Three states: `idle` → `importing` → `done`

- **idle**: Form with 3 credential fields (slug, email, API token) + "Importovat" button
- **importing**: Spinner with "Stahování a import dat z Fakturoidu..." message
- **done**: Result cards showing created/skipped counts + error list

No preview step, no entity selection — imports everything, skips duplicates automatically.

## Architecture — Adding a New Provider

Adding a new import source (e.g., Pohoda):

1. `internal/pohoda/client.go` — API client for the provider
2. `internal/service/pohoda_import_svc.go` — service with `ImportAll(ctx, client)` method
3. `internal/handler/pohoda_handler.go` — handler accepting provider-specific credentials in request body
4. Mount in router: `api.Mount("/import/pohoda", pohodaHandler.Routes())`
5. Frontend page: `frontend/src/routes/import/pohoda/+page.svelte`

Each provider is fully isolated — shares only the route prefix `/import/` and the import result pattern.

## Data Model

### Tables

**`fakturoid_import_log`** — tracks imported entities for duplicate detection:

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| fakturoid_entity_type | TEXT | `"subject"`, `"invoice"`, `"expense"` |
| fakturoid_id | INTEGER | Fakturoid's entity ID |
| local_entity_type | TEXT | `"contact"`, `"invoice"`, `"expense"` |
| local_id | INTEGER | ZFaktury's entity ID |
| imported_at | DATETIME | Timestamp of import |

### Mapping Rules

**Fakturoid Subject → Contact:**
- `registration_no` → `ico`, `vat_no` → `dic`
- Bank account parsed from `"číslo/kód"` format into separate fields
- All subjects mapped as `company` type
- Payment terms preserved from `due` field

**Fakturoid Invoice → Invoice:**
- Document type mapped: `"proforma"` → proforma, `"credit_note"` → credit note, default → regular
- Status mapped: `"paid"`, `"overdue"`, `"cancelled"` → direct mapping, others → `"sent"`
- Line items preserved with quantity, unit price, VAT rate
- Payment info extracted from first payment entry
- `CalculateTotals()` called to ensure VAT amounts are consistent

**Fakturoid Expense → Expense:**
- Dominant VAT rate calculated from line items (highest total amount wins)
- Description falls back: `description` → first line name → `"Import z Fakturoidu"`
- Payment method: `"cash"` preserved, everything else → `"bank_transfer"`

## Files

| File | Purpose |
|------|---------|
| `internal/fakturoid/client.go` | Fakturoid API v3 HTTP client |
| `internal/fakturoid/types.go` | Fakturoid API response types |
| `internal/domain/fakturoid_import.go` | Import domain types (result, preview item, log) |
| `internal/repository/fakturoid_import_repo.go` | Import log repository |
| `internal/service/fakturoid_import_svc.go` | Import service with `ImportAll()` |
| `internal/handler/fakturoid_handler.go` | HTTP handler (single POST endpoint) |
| `internal/handler/router.go` | Routes mounted at `/import/fakturoid` |
| `internal/cli/serve.go` | Wiring (no config dependency) |
| `frontend/src/routes/import/fakturoid/+page.svelte` | Import UI (form + results) |
| `frontend/src/lib/api/client.ts` | API types and client methods |

## Testing

- **Service tests**: Mapping function tests (subject→contact, invoice→invoice, expense→expense), `ImportAll` integration test with mock repos verifying created/skipped counts
- **Handler tests**: Request validation (missing credentials, invalid body, partial credentials)
- **Frontend tests**: Form rendering, successful import flow, error handling, result display, field disabling during import
