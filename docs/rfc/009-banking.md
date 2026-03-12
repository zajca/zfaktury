# RFC-009: Banking Integration

**Status:** Draft
**Date:** 2026-03-12

## Summary

Generic bank transaction import and automatic invoice payment matching. The architecture uses a provider interface so that multiple banks and import methods can coexist. The first provider is CSV import (universal, works with any Czech bank). Future providers include FIO Bank API (no PSD2 license required) and COBS/PSD2 API (AirBank, CSOB, KB, Moneta — requires PSD2 license or third-party aggregator).

## Background

Czech OSVC typically receive payments via bank transfer. The most common workflow is:
1. Issue invoice with variable symbol (= invoice number)
2. Customer pays via bank transfer, entering the variable symbol
3. OSVC manually checks bank statements and marks invoices as paid

ZFaktury currently supports only manual "Mark as paid" on invoices. This RFC automates step 3 by importing bank transactions and matching them to invoices by variable symbol and amount.

### Czech Banking Landscape

| Bank | API | Auth | License needed |
|------|-----|------|----------------|
| FIO banka | Proprietary REST | API token | No (free for clients) |
| Air Bank | COBS 3.1 | OAuth2 | PSD2 license from CNB |
| Ceska sporitelna | Erste Group API | OAuth2 | PSD2 license |
| CSOB | COBS | OAuth2 | PSD2 license |
| Komercni banka | KB API | OAuth2 | PSD2 license |
| MONETA | COBS | OAuth2 | PSD2 license |

**Practical implication:** For a self-hosted OSVC app, only FIO offers direct API access without a license. All other banks require either a PSD2 license, a third-party aggregator (Salt Edge, Enable Banking), or manual CSV/statement export from internet banking.

## Design

### Provider Interface

```go
// BankProvider fetches transactions from a bank.
type BankProvider interface {
    // Name returns the provider identifier (e.g., "csv", "fio", "airbank").
    Name() string
    // FetchTransactions returns transactions for the given date range.
    // For file-based providers (CSV), dateFrom/dateTo may be ignored
    // and the file content determines the range.
    FetchTransactions(ctx context.Context, params FetchParams) ([]BankTransaction, error)
}

type FetchParams struct {
    DateFrom time.Time
    DateTo   time.Time
    // FileData is set for file-based providers (CSV upload).
    FileData []byte
    // FileName helps detect bank format (e.g., "airbank_export.csv").
    FileName string
}
```

### Provider Implementations

#### Phase A — CSV Import (universal)

Parses CSV exports from Czech banks. Each bank has a different CSV format, so the provider auto-detects the bank by header row or filename pattern.

Supported formats (initial):
- **AirBank** — CSV export from internet banking
- **FIO banka** — CSV export (pohyby na uctu)
- **Generic** — fallback with configurable column mapping

CSV auto-detection logic:
1. Read header row
2. Match against known bank signatures (AirBank uses specific Czech column names, FIO uses different ones)
3. If no match, try generic parser with user-configured column mapping

#### Phase B — FIO API (future)

Direct API integration using FIO's REST API with token auth. No PSD2 license needed. Fetches JSON transactions for a date range.

#### Phase C — COBS/PSD2 (future)

OAuth2-based integration for banks supporting the Czech Open Banking Standard. Requires PSD2 license or third-party aggregator.

### Transaction Matching

Automatic matching runs on every import. The algorithm:

1. **Exact match** — Variable symbol matches invoice number AND amount matches invoice total. Confidence: high. Auto-matched.
2. **Partial match** — Variable symbol matches but amount differs (partial payment or overpayment). Confidence: medium. Suggested to user for manual confirmation.
3. **Amount-only match** — No variable symbol, but amount matches exactly one unpaid invoice. Confidence: low. Suggested to user.
4. **No match** — Transaction cannot be matched to any invoice. Shown in "unmatched" list.

When a transaction is matched to an invoice:
- `BankTransaction.InvoiceID` is set
- `BankTransaction.MatchType` records how it was matched (auto/manual)
- Invoice `PaidAmount` is updated (accumulated for partial payments)
- If `PaidAmount >= TotalAmount`, invoice status changes to "paid"

### Domain Changes

Update existing `internal/domain/bank.go`:

```go
type BankTransaction struct {
    ID                  int64
    ProviderName        string    // "csv", "fio", "airbank"
    BankAccount         string    // account the transaction belongs to
    ExternalID          string    // provider-specific unique ID (for dedup)
    TransactionDate     time.Time
    Amount              Amount
    Currency            string
    CounterpartyAccount string
    CounterpartyName    string
    VariableSymbol      string
    ConstantSymbol      string
    SpecificSymbol      string
    Message             string
    InvoiceID           *int64
    MatchType           string    // "auto", "manual", "" (unmatched)
    ImportedAt          time.Time
    CreatedAt           time.Time
}

type BankImport struct {
    ID           int64
    ProviderName string
    FileName     string    // original filename for CSV imports
    DateFrom     time.Time
    DateTo       time.Time
    TotalCount   int
    MatchedCount int
    ImportedAt   time.Time
}
```

### Database (Migration 020)

```sql
CREATE TABLE bank_imports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_name TEXT NOT NULL,
    file_name TEXT NOT NULL DEFAULT '',
    date_from TEXT NOT NULL,
    date_to TEXT NOT NULL,
    total_count INTEGER NOT NULL DEFAULT 0,
    matched_count INTEGER NOT NULL DEFAULT 0,
    imported_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE bank_transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    import_id INTEGER NOT NULL REFERENCES bank_imports(id),
    provider_name TEXT NOT NULL,
    bank_account TEXT NOT NULL DEFAULT '',
    external_id TEXT NOT NULL DEFAULT '',
    transaction_date TEXT NOT NULL,
    amount INTEGER NOT NULL,
    currency TEXT NOT NULL DEFAULT 'CZK',
    counterparty_account TEXT NOT NULL DEFAULT '',
    counterparty_name TEXT NOT NULL DEFAULT '',
    variable_symbol TEXT NOT NULL DEFAULT '',
    constant_symbol TEXT NOT NULL DEFAULT '',
    specific_symbol TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT '',
    invoice_id INTEGER REFERENCES invoices(id) ON DELETE SET NULL,
    match_type TEXT NOT NULL DEFAULT '',
    imported_at TEXT NOT NULL DEFAULT (datetime('now')),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(provider_name, external_id)
);

CREATE INDEX idx_bank_transactions_variable_symbol ON bank_transactions(variable_symbol);
CREATE INDEX idx_bank_transactions_invoice_id ON bank_transactions(invoice_id);
CREATE INDEX idx_bank_transactions_transaction_date ON bank_transactions(transaction_date);
CREATE INDEX idx_bank_transactions_import_id ON bank_transactions(import_id);
```

### Config Changes

Replace `FIOConfig` with generic `BankConfig`:

```toml
[bank]
# Default account identifier (shown in UI)
account = "1234567890/3030"

[bank.fio]
api_token = "xxx"  # Only needed for FIO API provider

[bank.csv]
# Column mapping for generic CSV import (optional, auto-detection preferred)
# date_column = "Datum"
# amount_column = "Castka"
# variable_symbol_column = "VS"
```

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/bank/import` | Upload CSV file, import + auto-match |
| GET | `/api/v1/bank/imports` | List past imports |
| GET | `/api/v1/bank/imports/{id}` | Import detail with transaction summary |
| GET | `/api/v1/bank/transactions` | List transactions (filterable: matched/unmatched, date range) |
| GET | `/api/v1/bank/transactions/{id}` | Transaction detail |
| POST | `/api/v1/bank/transactions/{id}/match` | Manually match transaction to invoice |
| POST | `/api/v1/bank/transactions/{id}/unmatch` | Remove match |
| GET | `/api/v1/bank/suggestions` | Get match suggestions for unmatched transactions |

### Frontend Pages

**`/bank/` — Transaction list**
- Table: date, amount, counterparty, variable symbol, matched invoice (link), match type badge
- Filters: date range, matched/unmatched, search
- "Import CSV" button -> file upload dialog

**`/bank/import/` — Import page**
- Drag & drop CSV upload
- Auto-detection result (detected bank name)
- Preview of transactions before import
- Import summary after processing (X imported, Y auto-matched, Z need review)

**`/bank/match/` — Matching review**
- List of unmatched transactions with suggested invoice matches
- One-click match/dismiss for suggestions
- Manual search to find invoice by number/customer

### Deduplication

On import, transactions are deduplicated by `(provider_name, external_id)`:
- For CSV imports, `external_id` is computed as a hash of (date + amount + counterparty_account + variable_symbol + message) since CSV rows have no unique ID
- For API providers, `external_id` is the bank's transaction ID
- Duplicate transactions are silently skipped

## Implementation Order

1. Migration 020 (bank_imports + bank_transactions tables)
2. Updated domain structs
3. Repository (BankTransactionRepo, BankImportRepo)
4. CSV parser — AirBank format first, then FIO, then generic
5. Matching service
6. Bank service (orchestrates import + matching)
7. HTTP handler
8. Frontend: transaction list, CSV import, matching review
9. Wire into router.go, serve.go, interfaces.go, client.ts

## Out of Scope

- FIO API provider (Phase B — separate RFC when needed)
- COBS/PSD2 providers (Phase C — requires license discussion)
- Third-party aggregator integration (Salt Edge, Enable Banking)
- Automatic periodic import (cron/scheduler)
- Bank statement PDF parsing
- Multi-currency transaction matching (CZK only for now)
- Expense matching (only invoice matching in this RFC)

## Open Questions

1. **AirBank CSV format** — Need a sample CSV export from AirBank internet banking to implement the parser. The exact column names and format need to be verified against a real export.
2. **Partial payments** — Should multiple transactions be matchable to one invoice (accumulating `PaidAmount`)? Current design says yes.
3. **Config migration** — Should `FIOConfig` be kept for backward compatibility, or replaced with `BankConfig` immediately?
