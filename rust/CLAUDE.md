# ZFaktury Rust -- Claude Code Instructions

## Project Overview

ZFaktury is a Czech invoicing and tax management desktop app for sole proprietors (OSVC). Native GPUI desktop UI + SQLite database, compiled to a single binary.

- **GUI framework:** GPUI (from Zed editor) -- Rust-native GPU-accelerated UI
- **Database:** SQLite via rusqlite (bundled, no system dependency)
- **Config:** TOML at `~/.zfaktury/config.toml`, env override via `ZFAKTURY_DATA_DIR`

## Build & Run

All commands must run inside the nix devshell (provides Rust toolchain, system libs, headless testing tools):

```bash
cd rust && nix develop

# Build
cargo build --workspace

# Test
cargo test --workspace

# Clippy
cargo clippy --workspace -- -D warnings

# Format check
cargo fmt --check

# Run the app
cargo run -p zfaktury-app

# Run migrations only
cargo run -p zfaktury-app -- migrate

# Check migration status
cargo run -p zfaktury-app -- migrate --status

# Headless screenshot
./scripts/headless-screenshot.sh './target/debug/zfaktury-app --route /invoices --exit-after 3'
```

For CI/one-liners that avoid interactive shell:
```bash
nix develop --command bash -c "CARGO_NET_GIT_FETCH_WITH_CLI=true cargo build --workspace"
```

## Workspace Crates

| Crate | Purpose |
|-------|---------|
| `zfaktury-domain` | Pure domain types, `Amount(i64)`, `DomainError` |
| `zfaktury-core` | Repository traits, services, calc modules |
| `zfaktury-db` | SQLite repo implementations, migrations |
| `zfaktury-gen` | XML/ISDOC/CSV/QR generation |
| `zfaktury-api` | External API clients (ARES, FIO, Fakturoid, OCR, email) |
| `zfaktury-config` | TOML config loading |
| `zfaktury-testutil` | Test builders and helpers |
| `zfaktury-app` | GPUI desktop binary (main entry point) |

## Coding Standards

### Money

Always use `domain::Amount` (newtype over `i64`, halere/cents). Never use `f64` for money in domain or service code.

```rust
let price = Amount::new(100, 50);      // 100.50 CZK
let tax = price.multiply(0.21);         // VAT
let total = price + tax;
assert_eq!(total.to_czk(), 121.61);    // Only for display
```

### Domain Purity

Domain structs in `zfaktury-domain` must have NO `serde`, `rusqlite`, or framework-specific derives/attributes. Serialization is handled in the crate that needs it (db crate maps to SQL, gen crate uses serde for XML).

### Error Handling

- Use `DomainError` variants for business errors: `NotFound`, `InvalidInput`, `PaidInvoice`, etc.
- Services return `Result<T, DomainError>`
- Wrap errors with context in services: descriptive verb + entity name
- DB crate converts `rusqlite::Error` to `DomainError` via the `From` impl in `error.rs`
- App crate uses `anyhow::Result` at startup; views store `Option<String>` for error display

### Database

- SQLite with WAL mode, foreign keys ON, busy timeout 5000ms
- Dates as TEXT (`YYYY-MM-DD`), datetimes as TEXT (`YYYY-MM-DDTHH:MM:SSZ`)
- Amounts as INTEGER (halere)
- Soft deletes via `deleted_at` column
- Use helper functions from `zfaktury-db/src/helpers.rs` for date parsing/formatting
- One connection per repository (WAL allows concurrent readers)

### GPUI Views

- Views hold `Arc<Service>` and spawn background tasks for all data operations
- Always use `cx.background_executor()` for blocking repo calls
- Call `cx.notify()` after mutating view state
- Use `ZfColors` constants from `theme.rs` for all colors

## Test Conventions

- `rstest` for parameterized tests (in `zfaktury-core`)
- `proptest` for property-based tests (Amount arithmetic, calc modules)
- `#[tokio::test]` with `wiremock` for API client tests (in `zfaktury-api`)
- `tempfile` for tests that need a real SQLite database (in `zfaktury-db`)
- `zfaktury-testutil` provides builder functions for domain structs in tests

Run all tests:
```bash
cargo test --workspace
```

## File Naming

- Rust files: `snake_case.rs` (e.g., `contact_repo.rs`, `invoice_svc.rs`, `vat_return.rs`)
- Service files: `*_svc.rs`
- Repository files: `*_repo.rs`
- SQL migrations: `V{N}__{description}.sql` (double underscore)

## Key Dependencies

| Purpose | Crate |
|---------|-------|
| GUI framework | `gpui` + `gpui_platform` (from Zed git repo) |
| CLI args | `clap` (derive) |
| SQLite | `rusqlite` (bundled) |
| HTTP client | `reqwest` (blocking + json) |
| Email | `lettre` (SMTP + rustls) |
| XML | `quick-xml` (serialize) |
| QR codes | `qrcode` + `image` |
| CSV | `csv` |
| Config | `toml` + `serde` |
| Dates | `chrono` |
| Errors | `thiserror` |
| Logging | `log` |
| Testing | `rstest`, `proptest`, `wiremock`, `tempfile` |

## Adding a New Feature (Checklist)

1. **Domain** -- Add types to `zfaktury-domain/src/` (no serde, no framework tags)
2. **Repo trait** -- Add trait to `zfaktury-core/src/repository/traits.rs`
3. **Repo impl** -- Implement in `zfaktury-db/src/repos/` with `SqliteXxxRepo` struct
4. **Service** -- Create `zfaktury-core/src/service/xxx_svc.rs`, re-export in `mod.rs`
5. **App wiring** -- Add service to `AppServices` in `zfaktury-app/src/app.rs`
6. **View** -- Create view in `zfaktury-app/src/views/`, register in `root.rs` and `views/mod.rs`
7. **Route** -- Add variant to `Route` enum in `navigation.rs`, sidebar entry in `sidebar.rs`
8. **Migration** -- If new tables needed, add SQL file in `migrations/` and entry in `migrate.rs`
9. **Tests** -- Unit tests in the relevant crate, integration tests in `zfaktury-db` for repos
