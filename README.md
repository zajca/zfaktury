# ZFaktury

Self-contained invoicing and tax management application for Czech sole proprietors (OSVC). Single binary, local SQLite database, full data ownership.

## Features

- **Invoicing** -- create, send, track invoices with PDF generation and QR payment codes
- **Credit notes & proforma invoices** -- settle proformas, issue credit notes
- **Recurring invoices & expenses** -- automatic generation on schedule
- **Email sending** -- send invoices with PDF and ISDOC attachments via SMTP
- **Payment reminders** -- overdue tracking and reminder emails
- **Expenses** -- manual entry, document upload, AI OCR extraction
- **ISDOC export** -- single invoice or batch export in Czech ISDOC format
- **Contacts** -- company/individual records with ARES and VIES integration
- **Exchange rates** -- CNB (Czech National Bank) daily rates
- **Invoice sequences & expense categories** -- customizable numbering and categorization
- **VAT filings** -- VAT return, control statement, EU sales list (EPO XML)
- **Income tax** -- annual tax return with deductions and credits
- **Bank integration** -- FIO API, CSV/GPC import, automatic payment matching
- **Reports** -- revenue, expenses, profit, cash flow, tax deadlines

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go, chi (HTTP), cobra (CLI), SQLite (modernc.org/sqlite) |
| Frontend | SvelteKit (adapter-static), Tailwind CSS v4, TypeScript |
| PDF | Maroto v2 |
| QR codes | dundee/qrpay (Czech SPD format) |
| Migrations | goose (embedded SQL) |
| Config | TOML |

## Prerequisites

- Go 1.25+
- Node.js 20+
- npm

## Quick Start

```bash
# Clone and build
git clone https://github.com/zajca/zfaktury.git
cd zfaktury
make install-hooks  # Set up pre-commit coverage check
make build

# Run
./zfaktury serve
# Open http://localhost:8080
```

## Demo Data

Populate the database with realistic demo data (Czech freelance web developer scenario -- 10 contacts, 18 invoices, 18 expenses, recurring templates):

```bash
./zfaktury seed --config config.dev.toml
```

Re-seed (clears all data first):

```bash
./zfaktury seed --force --config config.dev.toml
```

## Development

Two processes run in parallel -- Vite HMR for frontend and Go API server:

```bash
make dev
```

On first run, `config.dev.toml` is automatically created from `config.dev.dist.toml`. Edit the local copy to customize your dev environment (it is gitignored).

Or manually in two terminals:

```bash
# Terminal 1: Frontend (Vite HMR on :5173)
cd frontend && npm run dev

# Terminal 2: Go API (on :8080, proxies non-API to Vite)
go run ./cmd/zfaktury serve --dev
```

## Build

```bash
make build          # Frontend + Go binary -> ./zfaktury
make test           # Run all tests (backend + frontend)
make lint           # Lint all code (Go + frontend)
make clean          # Remove build artifacts
make install-hooks  # Install git pre-commit hook
```

## Testing

```bash
# Go tests
CGO_ENABLED=0 go test ./...

# Go tests with coverage report
make coverage-go

# Frontend tests
cd frontend && npm test

# Frontend tests with coverage
make coverage-frontend
```

### Coverage

Go backend test coverage is enforced at **80% minimum** via a pre-commit hook. Run `make install-hooks` after cloning to enable it.

Coverage by layer:

| Layer | Coverage |
|-------|----------|
| domain | 100% |
| service | ~82% |
| repository | ~80% |
| handler | ~78% |

Infrastructure packages (cli, config, database) are excluded from the coverage target as they require integration testing with the full application.

## Project Structure

```
cmd/zfaktury/         Entry point
internal/
  cli/                Cobra CLI commands
  config/             TOML config loading
  database/           SQLite connection, migrations
  domain/             Pure domain structs (no DB/JSON deps)
  repository/         SQL data access layer
  service/            Business logic
  handler/            HTTP JSON API handlers
frontend/             SvelteKit SPA
web/                  Embedded frontend (//go:embed)
scripts/              Build and dev helper scripts
```

## Configuration

Config file at `~/.zfaktury/config.toml` (created on first run):

```toml
[database]
# path = "/path/to/custom.db"  # Custom DB path (default: ~/.zfaktury/zfaktury.db)

[log]
# path = "logs/zfaktury.log"   # Log file path (default: stderr only)
# level = "info"               # debug, info, warn, error (default: info)

[smtp]
host = "smtp.example.com"
port = 587
username = "user"
password = "pass"
from = "jan@example.com"

[fio]
api_token = "your-fio-api-token"

[ocr]
provider = "openai"            # openai | openrouter | gemini | mistral | claude
api_key = "sk-your-api-key"
# model = ""                   # optional, each provider has a sensible default
# base_url = ""                # optional, for custom/self-hosted endpoints
```

| Setting | Description |
|---------|-------------|
| `database.path` | Custom SQLite database path. Useful for running multiple instances. Defaults to `~/.zfaktury/zfaktury.db`. |
| `log.path` | Log file path. When set, logs are written to both stderr and the file. |
| `log.level` | Log verbosity: `debug`, `info` (default), `warn`, `error`. |

Override data directory with `ZFAKTURY_DATA_DIR` environment variable.

## Data Storage

```
~/.zfaktury/
  zfaktury.db       SQLite database (WAL mode)
  documents/        Uploaded PDFs, images
  exports/          Generated XMLs, PDFs
  config.toml       Configuration
  logs/             Structured JSON logs
```

## Document Import & OCR

ZFaktury supports importing expenses directly from scanned documents (PDF, JPG, PNG, WebP). When OCR is configured, the system automatically extracts vendor name, invoice number, amounts, dates, and other fields.

### How it works

1. Go to **Naklady** (Expenses) and click **Import z dokladu**
2. Drag & drop a document or click to select a file
3. The system creates a skeleton expense and uploads the document
4. If OCR is configured, extracted data is shown for review -- edit any field and confirm
5. If OCR is not configured, you are redirected to the expense detail for manual entry

### OCR setup

OCR uses vision/multimodal LLM APIs to extract data from documents. Multiple providers are supported -- all use compatible Chat Completions APIs except Claude which uses Anthropic's Messages API.

Add the following to `~/.zfaktury/config.toml`:

```toml
[ocr]
provider = "openai"
api_key = "sk-your-api-key"
```

| Setting | Description |
|---------|-------------|
| `provider` | OCR provider: `openai`, `openrouter`, `gemini`, `mistral`, or `claude`. Defaults to `openai` if omitted. |
| `api_key` | Provider API key. OCR is disabled when this is empty. |
| `model` | Optional. Override the default model for the provider. |
| `base_url` | Optional. Override the API endpoint URL (useful for self-hosted or proxy setups). |

**Default models per provider:**

| Provider | Default model |
|----------|--------------|
| `openai` | `gpt-4o` |
| `openrouter` | `google/gemini-2.0-flash-001` |
| `gemini` | `gemini-2.0-flash` |
| `mistral` | `pixtral-large-latest` |
| `claude` | `claude-sonnet-4-20250514` |

Without the `[ocr]` section (or with an empty `api_key`), the import still works -- documents are uploaded and linked to expenses, but fields must be filled in manually.

### Supported file types

| Format | MIME type |
|--------|-----------|
| PDF | `application/pdf` |
| JPEG | `image/jpeg` |
| PNG | `image/png` |
| WebP | `image/webp` |
| HEIC | `image/heic` |

Maximum file size: 20 MB. Up to 10 documents per expense.

## License

Private project.
