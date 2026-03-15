# RFC-013: Code Quality & Robustness Hardening

**Status:** Draft
**Date:** 2026-03-15

## Summary

Systematic cleanup of error handling patterns across the codebase: replace ad-hoc `errors.New()` with domain sentinel errors, introduce error translation middleware to eliminate duplicate error-to-HTTP mapping in handlers, add pagination limit caps, and add config validation at startup.

## Background

The codebase has accumulated ~127 violations of the sentinel error convention defined in CLAUDE.md. Services use bare `errors.New("description is required")` instead of `domain.ErrInvalidInput`. This makes error handling inconsistent — some handlers check for sentinel errors, others pass raw strings to the HTTP response.

Additionally, 10+ handler files duplicate the same error-to-HTTP-status mapping logic via `mapXxxError()` functions that all do the same `errors.Is(err, domain.ErrNotFound) → 404` pattern.

### Problems

1. **~127 `errors.New()` violations** across 15 service/handler files instead of sentinel errors
2. **10+ duplicate `mapXxxError()` functions** in handlers doing identical error translation
3. **No pagination cap** — any client can request `?limit=999999999` and OOM the server
4. **No config validation at startup** — missing required settings cause cryptic errors at runtime
5. **2 sentinel errors defined in service layer** (`reminder_svc.go`) instead of `domain/errors.go`

## Design

### Task 1: Extend Domain Sentinel Errors

Add missing sentinel errors to `internal/domain/errors.go`:

```go
// New sentinel errors
var ErrInvoiceNotOverdue = errors.New("invoice is not overdue")
var ErrNoCustomerEmail   = errors.New("customer has no email address")
```

Remove the duplicates from `internal/service/reminder_svc.go`.

### Task 2: Replace `errors.New()` with Sentinel Errors in Services

For each violation, wrap the sentinel error with context using `fmt.Errorf`:

**Before:**
```go
if inv.CustomerID == 0 {
    return errors.New("customer is required")
}
```

**After:**
```go
if inv.CustomerID == 0 {
    return fmt.Errorf("customer is required: %w", domain.ErrInvalidInput)
}
```

This preserves the descriptive message while allowing `errors.Is(err, domain.ErrInvalidInput)` checks.

**Special mappings:**
- `"at least one line item is required"` → wrap `domain.ErrNoItems`
- `"cannot update a paid invoice"` / `"invoice is already paid"` → wrap `domain.ErrPaidInvoice`
- `"category with this key already exists"` → wrap `domain.ErrDuplicateNumber`
- All other validation messages → wrap `domain.ErrInvalidInput`

#### Files to modify (127 changes across):

| File | Violations |
|------|-----------|
| `service/invoice_svc.go` | 21 |
| `service/expense_svc.go` | 14 |
| `service/recurring_expense_svc.go` | 14 |
| `service/category_svc.go` | 11 |
| `service/sequence_svc.go` | 11 |
| `service/recurring_invoice_svc.go` | 9 |
| `service/contact_svc.go` | 8 |
| `service/document_svc.go` | 7 |
| `service/invoice_document_svc.go` | 7 |
| `service/tax_deduction_document_svc.go` | 7 |
| `handler/helpers.go` | 6 |
| `handler/recurring_expense_handler.go` | 2 |
| `handler/recurring_invoice_handler.go` | 2 |
| `service/settings_svc.go` | 1 |
| `service/reminder_svc.go` | 2 (move to domain) |

### Task 3: Error Translation Middleware

Create `internal/handler/error_middleware.go`:

```go
// mapDomainError translates domain errors to HTTP status codes.
// Used by handlers to avoid duplicating error-to-HTTP mapping.
func mapDomainError(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, domain.ErrNotFound):
        respondError(w, http.StatusNotFound, err.Error())
    case errors.Is(err, domain.ErrInvalidInput):
        respondError(w, http.StatusBadRequest, err.Error())
    case errors.Is(err, domain.ErrNoItems):
        respondError(w, http.StatusBadRequest, err.Error())
    case errors.Is(err, domain.ErrPaidInvoice):
        respondError(w, http.StatusConflict, err.Error())
    case errors.Is(err, domain.ErrDuplicateNumber):
        respondError(w, http.StatusConflict, err.Error())
    case errors.Is(err, domain.ErrFilingAlreadyExists):
        respondError(w, http.StatusConflict, err.Error())
    case errors.Is(err, domain.ErrFilingAlreadyFiled):
        respondError(w, http.StatusConflict, err.Error())
    case errors.Is(err, domain.ErrMissingSetting):
        respondError(w, http.StatusUnprocessableEntity, err.Error())
    case errors.Is(err, domain.ErrInvoiceNotOverdue):
        respondError(w, http.StatusBadRequest, err.Error())
    case errors.Is(err, domain.ErrNoCustomerEmail):
        respondError(w, http.StatusUnprocessableEntity, err.Error())
    default:
        respondError(w, http.StatusInternalServerError, "internal server error")
    }
}
```

Then replace all per-handler `mapXxxError()` functions with calls to `mapDomainError()`.

**Functions to delete:**
- `mapVATReturnError()` in `vat_return_handler.go`
- `mapIncomeTaxError()` in `income_tax_handler.go`
- `mapHealthInsuranceError()` in `health_insurance_handler.go`
- `mapSocialInsuranceError()` in `social_insurance_handler.go`
- `mapTaxCreditsError()` in `tax_credits_handler.go`
- `mapTaxDeductionError()` in `tax_deductions_handler.go`
- `mapInvestmentError()` in `investment_income_handler.go`
- Inline error blocks in `vies_handler.go`, `vat_control_handler.go`, `backup_handler.go`

### Task 4: Pagination Limit Cap

Add to `internal/handler/helpers.go`:

```go
const maxPaginationLimit = 500

func parsePagination(r *http.Request) (limit, offset int) {
    limit = parseQueryInt(r, "limit", 20)
    offset = parseQueryInt(r, "offset", 0)
    if limit > maxPaginationLimit {
        limit = maxPaginationLimit
    }
    if limit < 1 {
        limit = 1
    }
    if offset < 0 {
        offset = 0
    }
    return limit, offset
}
```

Replace all inline `parseQueryInt(r, "limit", ...)` / `parseQueryInt(r, "offset", ...)` calls in handlers with `parsePagination(r)`.

### Task 5: Config Validation at Startup

Add `internal/config/validate.go`:

```go
func (c *Config) Validate() error {
    var errs []string
    if c.Server.Port < 1 || c.Server.Port > 65535 {
        errs = append(errs, fmt.Sprintf("server.port must be 1-65535, got %d", c.Server.Port))
    }
    if c.SMTP.Host != "" && c.SMTP.Port == 0 {
        errs = append(errs, "smtp.port is required when smtp.host is set")
    }
    if c.OCR.Provider != "" && c.OCR.APIKey == "" {
        errs = append(errs, "ocr.api_key is required when ocr.provider is set")
    }
    if len(errs) > 0 {
        return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
    }
    return nil
}
```

Call from `internal/cli/serve.go` at startup, before starting the HTTP server.

## Implementation Order

1. Task 1: Extend domain sentinel errors (5 min)
2. Task 2: Replace `errors.New()` in services — 15 files, ~127 changes (parallelizable with worktree agents per file group)
3. Task 3: Error translation middleware + delete duplicate mappers (after Task 2)
4. Task 4: Pagination limit cap (independent)
5. Task 5: Config validation (independent)
6. Run `CGO_ENABLED=0 go build ./...` and `CGO_ENABLED=0 go test ./...`
7. Fix any test failures (error message assertions may need updating)

## Out of Scope

- Context propagation (`context.Background()` → proper `ctx`) — separate, much larger effort
- Slow query logging — requires database wrapper, separate RFC
- CLI command tests — separate effort
