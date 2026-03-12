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

## Coding Standards (MANDATORY)

These rules are non-negotiable. Every new or modified code MUST follow them.

### Go Error Handling

- **Always use `errors.Is()` for sentinel error comparison** -- NEVER `err == sql.ErrNoRows` or direct equality. Use `errors.Is(err, sql.ErrNoRows)`.
- **Always handle `time.Parse` errors** -- NEVER ignore with `_, _ = time.Parse(...)`. Use the helpers in `internal/repository/helpers.go`: `parseDate(layout, value)`, `parseDateOptional(layout, ns)`, `parseDatePtr(layout, ns)`.
- **Wrap errors in services with context** -- Every service method that calls a repo must wrap errors: `fmt.Errorf("creating invoice: %w", err)`. Use descriptive verbs: "creating", "updating", "deleting", "listing", "fetching".
- **Use sentinel errors from `domain/errors.go`** for business errors -- `ErrNotFound`, `ErrInvalidInput`, `ErrPaidInvoice`, `ErrNoItems`, `ErrDuplicateNumber`. Do NOT use ad-hoc `errors.New("...")` strings.

### Go Domain Purity

- **No struct tags on domain types** -- Domain structs in `internal/domain/` must have NO `json:`, `db:`, or any other tags. All serialization is handled via handler DTOs.
- **No duplicate scan code in repositories** -- Use scan helper functions (e.g., `scanInvoiceRow`, `scanInvoiceItem` in `invoice_repo.go`). When adding new entities, extract scan logic into a reusable helper.

### Svelte Data Loading

- **Use `onMount` for initial data loading** -- NEVER `$effect(() => { loadData(); })` for one-time initialization.
- **Use `$effect` ONLY for reactive side effects** -- Filter/search changes that trigger reload. Always guard with `let mounted = false` pattern:
  ```typescript
  let mounted = false;
  onMount(() => { loadData(); mounted = true; });
  $effect(() => { filterVar; if (!mounted) return; loadData(); });
  ```

### Svelte API Usage

- **Always use the typed API client** (`$lib/api/client.ts`) -- NEVER raw `fetch()` for API calls. If a method is missing from the client, add it there first.
- **No local type re-declarations** -- Import types from `$lib/api/client.ts`, do NOT redeclare `interface Contact { ... }` locally.

### Svelte Shared Utilities

- **Invoice status labels/colors** -- Import from `$lib/utils/invoice.ts`, do NOT define inline.
- **Invoice items editor** -- Use `<InvoiceItemsEditor>` component from `$lib/components/InvoiceItemsEditor.svelte` for any invoice item editing UI. Do NOT duplicate item editor markup.

### Accessibility

- **Error messages** must have `role="alert"` on the container div.
- **Loading spinners** must have `role="status"` and `<span class="sr-only">Nacitani...</span>`.
- **Clickable table rows** must have `role="link"`, `tabindex="0"`, `onkeydown` Enter handler, and use `goto()` (not `window.location.href`).
- **Backdrop overlays** use `role="presentation"`, NOT `role="button"`.

### Config

- TOML config at `~/.zfaktury/config.toml`
- `ZFAKTURY_DATA_DIR` env var overrides data directory
- Config struct in `internal/config/config.go`
- Dev config: `config.dev.dist.toml` is the template (committed), `config.dev.toml` is local (gitignored). `scripts/dev.sh` auto-copies dist to local on first run.
- `[database] path` -- custom SQLite path (default: `DataDir/zfaktury.db`). Useful for running multiple instances.
- `[log] path` -- log file path (default: stderr only). When set, logs go to both stderr and the file.
- `[log] level` -- `debug`, `info` (default), `warn`, `error`

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

## Development Cycle (MANDATORY)

Every implementation batch MUST follow this cycle. No exceptions. Skipping steps leads to bugs, security issues, and inconsistent code.

### Phase 1: Plan

- Use `plan` mode (EnterPlanMode) to design the implementation before writing code
- Break work into independent tasks that can be parallelized
- Identify shared file conflicts upfront
- Get plan approval before proceeding

### Phase 2: Implement with Teammates

- **ALWAYS use Agent tool with worktree isolation** for parallel tasks
- Each agent gets its own worktree branch to avoid conflicts
- Agents MUST NOT edit shared files -- lead merges those after
- Each agent runs `go build` and `go test` on its own code before reporting done

### Phase 3: Merge & Build

- Lead copies new files from worktrees to main
- Lead manually integrates shared files (router.go, serve.go, interfaces.go, client.ts, helpers.go)
- Run full build: `CGO_ENABLED=0 go build ./...`
- Run full tests: `CGO_ENABLED=0 go test ./...`
- Fix any failures before proceeding

### Phase 4: Review (parallel agents)

All three reviews run as parallel background agents after merge:

1. **Code Review** (`developer:code-reviewer` agent)
   - Check for bugs, logic errors, code quality issues
   - Verify adherence to project conventions (3-layer arch, Amount for money, no JSON tags on domain structs)
   - Check test coverage adequacy

2. **Security Review** (`developer:code-security` agent)
   - SQL injection, path traversal, XSS, input validation
   - File upload safety (size limits, content type validation)
   - Sensitive data exposure, error message leakage
   - Output to `docs/AGENT-REPORTS/SECURITY.md`

3. **UX Review** (general agent reviewing frontend)
   - Check Svelte 5 runes usage (NOT old reactive syntax)
   - Verify Czech locale for UI text, dates, currency
   - Check error states, loading states, empty states
   - Verify responsive layout and accessibility basics

### Phase 5: Fix & Refactor

- Fix all critical and important findings from reviews
- Run `simplify` skill if needed for code cleanup
- Run build + tests again after fixes
- Commit fixes separately from implementation

### Phase 6: Commit

- Only commit after all reviews pass and fixes are applied
- Clean, descriptive commit message summarizing what was implemented

## Agent Teams Conventions

When working as part of an agent team:

- **ALWAYS use Agent tool with `isolation: "worktree"`** for parallel implementation tasks
- Each teammate owns its own files -- NEVER edit files owned by another agent
- Shared files (`router.go`, `interfaces.go`, `client.ts`, `serve.go`, `helpers.go`) are edited ONLY by the lead after all agents finish
- After completing work, run `go build ./...` and `go test ./...` to verify
- Coordinate via messages when you need API contracts or interface definitions from another agent

### Critical shared files (lead merges only)

- `internal/handler/router.go` - route registration
- `internal/repository/interfaces.go` - repository interfaces
- `frontend/src/lib/api/client.ts` - API types
- `internal/cli/serve.go` - dependency wiring
- `internal/handler/helpers.go` - shared DTOs

## Testing

### Go

- `CGO_ENABLED=0 go test ./...` for all Go tests
- Integration tests with real SQLite in `tests/integration/`

### Frontend (Svelte Component Tests)

**Run:** `cd frontend && npm test` (Vitest with jsdom)

**Infrastructure:** Vitest + jsdom + `@testing-library/svelte` + `@testing-library/jest-dom`
- Config: `frontend/vite.config.ts` (test section)
- Setup: `frontend/src/test-setup.ts` (jest-dom matchers)
- 276 tests across 24 files covering all route pages and shared components

**File naming:** `page.test.ts` (NOT `+page.test.ts` -- avoids SvelteKit routing conflicts)

**Test locations:**
- Shared components: `src/lib/components/*.test.ts`
- Route pages: `src/routes/**/page.test.ts`

**Standard test pattern:**
```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);
vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

function jsonResponse(data: unknown, status = 200) {
  return new Response(JSON.stringify(data), {
    status, statusText: status === 200 ? 'OK' : 'Error',
    headers: { 'Content-Type': 'application/json' }
  });
}

beforeEach(() => { mockFetch.mockReset(); });
afterEach(() => { cleanup(); });
```

**Critical pitfalls:**

1. **`vi.mock` hoisting** -- Factory functions are hoisted above all `const`/`let` declarations. Cannot reference variables defined in the same file. Use inline `vi.fn()` in factories, access mocks via dynamic import in test bodies:
   ```typescript
   // WRONG: vi.mock(() => ({ goto: mockGoto }))  -- mockGoto not yet defined
   // RIGHT:
   vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
   // In test:
   const { goto } = await import('$app/navigation');
   expect(goto).toHaveBeenCalledWith('/invoices');
   ```

2. **`$app/state` mock for detail pages** -- Pages using `page.params` need mutable mock:
   ```typescript
   vi.mock('$app/state', () => ({
     page: { params: { id: '1' }, url: { pathname: '/contacts/1', searchParams: new URLSearchParams() } }
   }));
   // Reset in beforeEach:
   beforeEach(async () => {
     const { page } = await import('$app/state');
     (page as any).params = { id: '1' };
   });
   ```

3. **HTML5 `required` bypass** -- `fireEvent.click(submitBtn)` triggers native validation, blocking custom validation tests. Remove required attrs first:
   ```typescript
   document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));
   ```

4. **Fake timers + waitFor** -- `waitFor` polling can freeze with fake timers. Use `await vi.advanceTimersByTimeAsync(10)` to flush Svelte state updates.

5. **Snippet children (Layout)** -- Layout uses `children: Snippet`. Cannot pass children directly in `render()`. Use `LayoutTestWrapper.svelte` wrapper component.

6. **Delete mock chaining** -- After delete (204 response), component may auto-reload data. Chain mocks: `mockFetch.mockResolvedValueOnce(deleteResponse).mockResolvedValueOnce(jsonResponse(reloadData))`

**Type checking:** `cd frontend && npm run check`
