# ZFaktury - Claude Code Instructions

## Project Overview

ZFaktury is a self-contained invoicing and tax management app for Czech sole proprietors (OSVC). Go backend + SvelteKit frontend, single binary with embedded static files, SQLite database.

## Architecture

3-layer architecture: Handler (HTTP/CLI) -> Service (business logic) -> Repository (SQL)

- **Domain structs** (`internal/domain/`) are pure -- no DB tags, no JSON tags
- **Repositories** (`internal/repository/`) map between domain and SQL, use `*sql.DB` directly
- **Services** (`internal/service/`) contain business logic, depend on repository interfaces
- **Handlers** (`internal/handler/`) use JSON DTOs (defined in handler files), map to/from domain structs
- **CLI commands** (`internal/cli/`) share the same service layer as HTTP handlers
- Repository interfaces are defined in `internal/repository/interfaces.go`

## Key Conventions

### Money

All monetary amounts use `domain.Amount` (int64, halere/cents). Never use float for money. Database stores INTEGER columns for all amounts.

### Database

- SQLite with WAL mode, foreign keys ON, busy timeout 5000ms
- Pure Go driver: `modernc.org/sqlite` (no CGO required)
- Migrations via goose, embedded SQL files in `internal/database/migrations/`
- Soft deletes via `deleted_at` column (nullable timestamp)
- Dates stored as TEXT in ISO 8601 format

### API

- All routes under `/api/v1/`
- JSON request/response with proper HTTP status codes
- Handler DTOs are separate from domain structs (defined in `internal/handler/helpers.go`)
- Use `chi.URLParam` for path parameters
- Pagination via `limit` and `offset` query params

### Frontend

- SvelteKit with adapter-static (SPA mode, fallback to index.html)
- Svelte 5 runes (`$state`, `$derived`, `$effect`) -- NOT old reactive syntax
- Tailwind CSS v4 with vite plugin (NOT PostCSS)
- TypeScript throughout
- API client in `frontend/src/lib/api/client.ts`
- Czech locale for UI text, dates, currency formatting

### Config

- TOML config at `~/.zfaktury/config.toml`
- `ZFAKTURY_DATA_DIR` env var overrides data directory
- Config struct in `internal/config/config.go`

## Build & Run

```bash
# Development (2 processes: Vite HMR + Go server)
make dev

# Production build (frontend + single Go binary)
make build

# Tests
make test

# Build without CGO (required on systems without gcc)
CGO_ENABLED=0 go build -o zfaktury ./cmd/zfaktury
```

## File Naming

- Go: `snake_case.go` (e.g., `contact_repo.go`, `invoice_svc.go`, `contact_handler.go`)
- Svelte: `PascalCase.svelte` for components, SvelteKit file conventions for routes
- SQL migrations: `NNN_description.sql` (goose format)

## Dependencies

| Purpose | Package |
|---------|---------|
| HTTP router | `github.com/go-chi/chi/v5` |
| CLI | `github.com/spf13/cobra` |
| SQLite | `modernc.org/sqlite` |
| Migrations | `github.com/pressly/goose/v3` |
| Config | `github.com/BurntSushi/toml` |
| PDF | `github.com/johnfercher/maroto/v2` |
| QR codes | `github.com/dundee/qrpay` |
| Logging | `log/slog` (stdlib) |
| XML | `encoding/xml` (stdlib) |

## Adding New Features

1. Define domain structs in `internal/domain/`
2. Add repository interface to `internal/repository/interfaces.go`
3. Implement repository in `internal/repository/`
4. Create service in `internal/service/`
5. Add HTTP handler with DTOs in `internal/handler/`
6. Mount routes in `internal/handler/router.go`
7. Wire dependencies in `internal/cli/serve.go`
8. Add frontend page in `frontend/src/routes/`
9. Add API types to `frontend/src/lib/api/client.ts`

## LSP Setup (Claude Code)

Three LSP servers provide diagnostics and code navigation in Claude Code:

| Language | Plugin | Binary |
|----------|--------|--------|
| Go | `gopls-lsp@claude-plugins-official` | `gopls` |
| TypeScript | `typescript-lsp@claude-plugins-official` | `typescript-language-server` |
| Svelte | `svelte-lsp@local` | `svelteserver` |

### 1. Install language servers

**Ubuntu/Debian:**

```bash
# gopls
go install golang.org/x/tools/gopls@latest

# svelte-language-server
npm install -g svelte-language-server
```

**NixOS:**

```nix
home.packages = [ pkgs.gopls pkgs.nodePackages.svelte-language-server ];
```

### 2. Install Claude Code plugins

```bash
# Go and TypeScript (from marketplace)
claude plugin install gopls-lsp@claude-plugins-official
claude plugin install typescript-lsp@claude-plugins-official
```

Svelte has no marketplace plugin -- create a local one:

```bash
mkdir -p ~/.claude/plugins/svelte-lsp
```

`~/.claude/plugins/svelte-lsp/manifest.json`:
```json
{
  "name": "svelte-lsp",
  "version": "0.1.0",
  "description": "Svelte Language Server integration for Claude Code"
}
```

`~/.claude/plugins/svelte-lsp/lsp.json`:
```json
{
  "svelte": {
    "command": "svelteserver",
    "args": ["--stdio"],
    "extensionToLanguage": {
      ".svelte": "svelte"
    }
  }
}
```

Then enable it in `~/.claude/settings.json`:
```json
{
  "enabledPlugins": {
    "svelte-lsp@local": true
  }
}
```

### 3. Restart Claude Code

LSP plugins load on startup. After setup, restart Claude Code and verify with `LSP documentSymbol` on any `.go`, `.ts`, or `.svelte` file.

## Testing

- Use `go test ./...` for Go tests
- Integration tests with real SQLite in `tests/integration/`
- Frontend: `cd frontend && npm run check` for type checking
