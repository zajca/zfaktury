# ZFaktury

Self-contained invoicing and tax management application for Czech sole proprietors (OSVC). Single binary, local SQLite database, full data ownership.

## Features

- **Invoicing** -- create, send, track invoices with PDF generation and QR payment codes
- **Expenses** -- manual entry, document upload, AI OCR extraction
- **Contacts** -- company/individual records with ARES and VIES integration
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
make build

# Run
./zfaktury serve
# Open http://localhost:8080
```

## Development

Two processes run in parallel -- Vite HMR for frontend and Go API server:

```bash
make dev
```

Or manually in two terminals:

```bash
# Terminal 1: Frontend (Vite HMR on :5173)
cd frontend && npm run dev

# Terminal 2: Go API (on :8080, proxies non-API to Vite)
go run ./cmd/zfaktury serve --dev
```

## Build

```bash
make build    # Frontend + Go binary -> ./zfaktury
make test     # Run all tests
make clean    # Remove build artifacts
```

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
[user]
name = "Jan Novak"
ico = "12345678"
dic = "CZ12345678"
vat_registered = true
street = "Hlavni 1"
city = "Praha"
zip = "11000"
email = "jan@example.com"
bank_account = "1234567890"
bank_code = "0800"

[smtp]
host = "smtp.example.com"
port = 587
username = "user"
password = "pass"
from = "jan@example.com"

[fio]
api_token = "your-fio-api-token"
```

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

## API Endpoints

All endpoints under `/api/v1`:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET/POST | `/api/v1/contacts` | List / create contacts |
| GET/PUT/DELETE | `/api/v1/contacts/{id}` | Get / update / delete contact |
| GET | `/api/v1/contacts/ares/{ico}` | ARES lookup by ICO |
| GET/POST | `/api/v1/invoices` | List / create invoices |
| GET/PUT/DELETE | `/api/v1/invoices/{id}` | Get / update / delete invoice |
| POST | `/api/v1/invoices/{id}/send` | Mark invoice as sent |
| POST | `/api/v1/invoices/{id}/pay` | Mark invoice as paid |
| POST | `/api/v1/invoices/{id}/duplicate` | Duplicate invoice |
| GET/POST | `/api/v1/expenses` | List / create expenses |
| GET/PUT/DELETE | `/api/v1/expenses/{id}` | Get / update / delete expense |

## Claude Code Agent Teams

Agent teams allow parallel development using multiple coordinated Claude Code sessions. One session acts as team lead, others work as teammates on independent tasks.

### Prerequisites

- tmux (installed via NixOS home-manager `programs.tmux`)
- `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1` in `~/.claude/settings.json` env

### Setup

The env var is already configured in `~/.claude/settings.json`:

```json
{
  "env": {
    "CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS": "1"
  }
}
```

### Usage with tmux

Start a tmux session first, then launch Claude Code inside it. The `auto` display mode (default) detects tmux and uses split panes automatically:

```bash
# Start tmux session
tmux new -s zfaktury

# Inside tmux, start Claude Code
claude

# Ask Claude to create an agent team:
# "Create an agent team for implementing PDF invoice generation:
#  - Backend agent: service + handler
#  - Frontend agent: download button
#  - Test agent: integration tests"
```

Each teammate gets its own tmux pane. Click into a pane to interact with that teammate directly.

To force a specific display mode:

```bash
claude --teammate-mode in-process  # all in one terminal, Shift+Down to cycle
```

### Common scenarios

**Full-stack feature** (3 agents: backend + frontend + tests):
```
Create an agent team for [feature]:
- Backend agent: implement service and handler
- Frontend agent: add UI components
- Test agent: write integration tests
```

**Parallel services** (2-4 agents, each owns a directory):
```
Create an agent team for implementing independent services:
- Agent ARES: internal/service/ares/
- Agent QR: internal/service/qr/
- Agent CNB: internal/service/cnb/
```

**Code review** (3 agents with different focus):
```
Create an agent team to review the project:
- Security reviewer
- Performance reviewer
- Architecture reviewer
```

### Conventions

See `CLAUDE.md` section "Agent Teams Conventions" for rules on file ownership, shared file coordination, and quality gates.

### Key commands

| Action | Command |
|--------|---------|
| Cycle teammates (in-process) | Shift+Down |
| Toggle task list | Ctrl+T |
| List tmux sessions | `tmux ls` |
| Kill orphaned session | `tmux kill-session -t <name>` |

## License

Private project.
