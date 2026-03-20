# ZFaktury Rust -- Architecture

## Workspace Structure

The workspace contains 8 crates with a strict dependency hierarchy (leaf to root):

```
zfaktury-domain          Pure domain types, Amount, DomainError (no external deps except chrono, thiserror)
    |
zfaktury-testutil        Test helpers and builders (depends on domain)
    |
zfaktury-core            Repository traits, service layer, calc modules (depends on domain)
    |
    +--- zfaktury-db     SQLite implementations of repo traits (depends on domain, core; uses rusqlite)
    |
    +--- zfaktury-gen    XML/ISDOC/CSV/QR generation (depends on domain; uses quick-xml, qrcode, csv)
    |
    +--- zfaktury-api    External API clients: ARES, FIO, Fakturoid, OCR, email (depends on domain; uses reqwest, lettre)
    |
zfaktury-config          TOML config loading, path resolution (standalone; uses serde, toml, dirs)
    |
zfaktury-app             GPUI desktop binary (depends on all above; uses gpui, clap)
```

## 3-Layer Architecture

```
GPUI View (app crate)  -->  Arc<Service> (core crate)  -->  Arc<dyn Repo> (core trait, db impl)  -->  rusqlite
```

**Views** (in `zfaktury-app/src/views/`) implement the GPUI `Render` trait and hold an `Arc<SomeService>`. They use `cx.spawn()` with `background_executor` to call service methods off the main thread.

**Services** (in `zfaktury-core/src/service/`) contain business logic. Each service owns `Arc<dyn RepoTrait + Send + Sync>` and optionally depends on other services (also via `Arc`). All services are `Send + Sync`.

**Repositories** are defined as traits in `zfaktury-core/src/repository/traits.rs` and implemented in `zfaktury-db/src/repos/`. Each SQLite repo wraps a `rusqlite::Connection` and maps between domain structs and SQL.

## Data Flow

### Read path (e.g. loading invoice list)

1. `InvoiceListView::load_data()` clones `Arc<InvoiceService>` and calls `cx.spawn()`
2. Inside the spawn, `cx.background_executor().spawn()` moves the call off the UI thread
3. `InvoiceService::list()` calls `self.repo.list(filter)` on `Arc<dyn InvoiceRepo>`
4. `SqliteInvoiceRepo::list()` executes SQL, maps rows to `Invoice` domain structs
5. Result propagates back through the spawn chain
6. `this.update(cx, ...)` applies data to view state and calls `cx.notify()` to trigger re-render

### Write path (e.g. creating an invoice)

1. View collects form state, builds an `Invoice` domain struct
2. Spawns to background: `service.create(&mut invoice)`
3. Service validates (checks items, contact existence), computes totals, assigns number
4. Calls `self.repo.create(&mut invoice)` which INSERTs and sets `invoice.id`
5. Optionally logs to audit: `self.audit.log("invoice", id, "created", ...)`
6. View navigates to the detail screen on success

## Money: Amount(i64)

All monetary values use `domain::Amount` -- a newtype over `i64` representing the smallest currency unit (halere for CZK, cents for EUR/USD). 100 CZK = `Amount::from_halere(10000)`.

- Arithmetic: `Add`, `Sub`, `Neg`, `AddAssign`, `SubAssign`
- Conversion: `from_float()`, `to_czk()`, `multiply(factor)`
- Display: formats as `"123.45"` (whole.fraction)
- Database: stored as `INTEGER` columns
- Never use `f64` for monetary values in domain or service code

## Database

- **Engine:** SQLite via `rusqlite` with bundled SQLite (no system dependency)
- **Pragmas:** WAL mode, foreign keys ON, busy timeout 5000ms
- **Dates:** stored as `TEXT` in ISO 8601 format (`YYYY-MM-DD` for dates, `YYYY-MM-DDTHH:MM:SSZ` for datetimes)
- **Amounts:** stored as `INTEGER` (halere/cents)
- **Soft deletes:** `deleted_at` column (nullable TEXT timestamp)
- **Connection strategy:** one `Connection` per repository (WAL allows concurrent readers)

### Helper functions (`zfaktury-db/src/helpers.rs`)

- `parse_date(value)` / `parse_datetime(value)` -- parse TEXT to chrono types
- `parse_date_optional(value)` / `parse_datetime_optional(value)` -- handle nullable columns
- `format_date(date)` / `format_datetime(dt)` -- format for storage

## Migration Strategy

Migrations are embedded SQL files in `migrations/` compiled into the binary via `include_str!()`. The migrate module (`zfaktury-db/src/migrate.rs`):

1. Creates `_zfaktury_migrations` tracking table
2. Detects an existing `goose_db_version` table (bridge from Go codebase) and records already-applied versions
3. Applies pending migrations in version order
4. Currently 24 migrations (V1 through V24)

CLI: `zfaktury migrate` runs pending migrations; `zfaktury migrate --status` shows status without changes.

## Error Handling

- **Domain errors:** `DomainError` enum in `zfaktury-domain/src/errors.rs` with variants: `NotFound`, `InvalidInput`, `PaidInvoice`, `NoItems`, `DuplicateNumber`, `FilingAlreadyExists`, `FilingAlreadyFiled`, `MissingSetting`, `InvoiceNotOverdue`, `NoCustomerEmail`
- **DB errors:** `DbError` in `zfaktury-db/src/error.rs` wraps `rusqlite::Error` and converts to `DomainError` (e.g. `QueryReturnedNoRows` -> `NotFound`)
- **Config errors:** `ConfigError` in `zfaktury-config/src/lib.rs` with `HomeDir`, `ReadFile`, `ParseFile` variants
- **Services** return `Result<T, DomainError>` -- repository traits use the same error type
- **App crate** uses `anyhow::Result` at the top level for initialization; views store `Option<String>` for error display

All error types use `thiserror` for derive macros.

## Configuration

TOML config at `~/.zfaktury/config.toml` (or `$ZFAKTURY_DATA_DIR/config.toml`).

Resolution order:
1. `ZFAKTURY_DATA_DIR` env var sets data directory (overrides everything)
2. Config file at `{data_dir}/config.toml`
3. If file doesn't exist, defaults are used
4. `~` in paths is expanded to home directory

Key sections: `[database]`, `[log]`, `[server]`, `[backup]`, `[smtp]`, `[fio]`, `[ocr]`.

## Calculation Modules

`zfaktury-core/src/calc/` contains pure computation modules:

| Module | Purpose |
|--------|---------|
| `vat` | VAT return calculation, control statement line generation |
| `income_tax` | Income tax computation with brackets, solidarity surcharge |
| `insurance` | Social and health insurance premium calculations |
| `constants` | Czech tax constants per year (rates, limits, thresholds) |
| `credits` | Tax credit aggregation (personal, spouse, children) |
| `deductions` | Tax deduction summation and cap enforcement |
| `fifo` | FIFO cost basis matching for security transactions |
| `annual_base` | Annual tax base computation (revenue - expenses - deductions) |
| `recurring` | Next-due-date calculation for recurring invoices/expenses |
