# Security Review - Round 2

**Date:** 2026-03-10
**Scope:** Commit `81f2468` (Round 1) + current unstaged changes — new files for Credit Notes, Recurring Invoices, Recurring Expenses, AI OCR, and shared wiring
**Files reviewed:**
- `internal/service/invoice_svc.go`
- `internal/handler/invoice_handler.go`
- `internal/repository/recurring_invoice_repo.go`
- `internal/service/recurring_invoice_svc.go`
- `internal/handler/recurring_invoice_handler.go`
- `internal/service/ocr/openai.go`
- `internal/service/ocr_svc.go`
- `internal/handler/ocr_handler.go`
- `internal/repository/recurring_expense_repo.go`
- `internal/service/recurring_expense_svc.go`
- `internal/handler/recurring_expense_handler.go`
- `internal/cli/serve.go`
- `internal/handler/router.go`
**Model:** claude-sonnet-4-6

---

## Security Review Checklist

- [x] Injection risks reviewed
- [x] Authentication/Authorization verified
- [x] Secrets handling reviewed (OCR API key)
- [x] Dependency audit reviewed
- [x] Transport security verified
- [x] Logging practices checked
- [x] Concurrency issues reviewed
- [x] IaC and container configs analyzed (N/A - single binary)

---

## Previous Findings Status Update

Before new findings, several items from prior rounds are worth noting as resolved or progressed:

| ID | Title | Status in R2 code |
|----|-------|-------------------|
| SEC-009 | No HTTP security headers | **FIXED** — `securityHeadersMiddleware` now sets `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy` |
| SEC-013 | Batch ISDOC export unbounded | **FIXED** — `ExportISDOCBatch` now enforces a `500` invoice cap at line 519 |
| SEC-015 | Zip entry path traversal | **FIXED** — `filepath.Base(invoice.InvoiceNumber)` applied before creating zip entry at line 549 |
| SEC-003/004/005 | LIMIT/OFFSET via fmt.Sprintf | **Still open** — pattern unchanged in `invoice_repo.go:382`, `contact_repo.go:169`, `expense_repo.go:240`; new `recurring_expense_repo.go:197` repeats the same pattern |

---

## New Findings Summary

| ID | Severity | File | Line(s) | Title |
|----|----------|------|---------|-------|
| SEC-018 | HIGH | `internal/service/ocr/openai.go` | 73 | Unbounded OpenAI response body read |
| SEC-019 | HIGH | `internal/cli/serve.go` | 134-135 | OCR HTTP client timeout (60s) exceeds server WriteTimeout (30s) — OCR requests always time out |
| SEC-020 | MEDIUM | `internal/repository/recurring_expense_repo.go` | 197 | LIMIT/OFFSET injected via fmt.Sprintf (extends SEC-003) |
| SEC-021 | MEDIUM | `internal/handler/ocr_handler.go` | 66 | OCR service errors forwarded verbatim including internal file paths |
| SEC-022 | MEDIUM | `internal/handler/recurring_expense_handler.go` | 321-348 | `as_of_date` in `/generate` accepts arbitrary past/future dates, no rate-limiting |
| SEC-023 | LOW | `internal/service/ocr/openai.go` | 248 | OCR model output content included in error message (information leakage) |
| SEC-024 | LOW | `internal/handler/recurring_invoice_handler.go` | 301 | Delete error forwarded verbatim including internal resource ID |
| SEC-025 | INFO | `internal/service/recurring_invoice_svc.go` | 110-149 | `ProcessDue` exposed as unauthenticated HTTP endpoint with no guard |

---

## Detailed Findings

### SEC-018 — HIGH: Unbounded OpenAI Response Body Read

**File:** `internal/service/ocr/openai.go:73`
**CWE:** CWE-400 (Uncontrolled Resource Consumption)

```go
body, err := io.ReadAll(resp.Body)
```

`io.ReadAll` reads the entire response into memory with no cap. A rogue or misconfigured OpenAI-compatible endpoint could return an arbitrarily large response body. The existing `openAITimeout` of 60 seconds limits exposure to time-based attacks but does not bound memory consumption: a 60-second stream of data at even a moderate rate could exhaust process memory.

This is the same pattern flagged as SEC-001 for the ARES client. The document upload path already demonstrates the correct approach.

**Fix — apply a size cap before reading:**
```go
const maxOpenAIResponseBytes = 2 << 20 // 2 MB is generous for a JSON chat response

limited := io.LimitReader(resp.Body, maxOpenAIResponseBytes+1)
body, err := io.ReadAll(limited)
if err != nil {
    return nil, fmt.Errorf("reading OpenAI response: %w", err)
}
if int64(len(body)) > maxOpenAIResponseBytes {
    return nil, fmt.Errorf("OpenAI response too large (> %d bytes)", maxOpenAIResponseBytes)
}
```

---

### SEC-019 — HIGH: OCR Client Timeout Exceeds Server WriteTimeout

**File:** `internal/cli/serve.go:134-135` and `internal/service/ocr/openai.go:19-20`
**CWE:** CWE-400 (Uncontrolled Resource Consumption)

```go
// serve.go
ReadTimeout:  15 * time.Second,
WriteTimeout: 30 * time.Second,

// openai.go
openAITimeout = 60 * time.Second
```

The `http.Server.WriteTimeout` of 30 seconds is the deadline for the entire response to be written back to the client. The OpenAI HTTP client has a 60-second timeout, meaning the OpenAI API call alone can take up to 60 seconds — which is 30 seconds longer than the server will hold the connection open.

The result is that **every OCR request that takes more than 30 seconds at the OpenAI tier will timeout at the Go server level first**, causing the client to receive a connection-reset/timeout error while the goroutine continues running and consuming resources until the 60-second OpenAI timeout fires. Under load, this creates goroutine leak pressure.

The chi `middleware.Timeout(30 * time.Second)` at the router level also cancels the request context after 30 seconds, which will propagate through `ctx` to the outgoing HTTP request — so the OpenAI call will be cancelled at ~30s anyway, making the 60s OCR timeout effectively dead code.

**Fix:** Either:
1. Raise `WriteTimeout` to 90 seconds for the server (and ensure clients use appropriate timeouts), **or**
2. Lower `openAITimeout` to 25 seconds so the OCR call completes (or fails cleanly) well within the server's WriteTimeout and chi context deadline, **or**
3. Handle OCR asynchronously (submit job, poll result) — most robust but a larger change.

The minimal fix is to align timeouts:
```go
// openai.go — keep well under server WriteTimeout and chi Timeout middleware
openAITimeout = 25 * time.Second

// serve.go — or raise server timeouts if OCR needs more time
WriteTimeout: 90 * time.Second,
```

---

### SEC-020 — MEDIUM: LIMIT/OFFSET Injected via fmt.Sprintf in recurring_expense_repo.go

**File:** `internal/repository/recurring_expense_repo.go:197`
**CWE:** CWE-89 (pattern; integer values, not directly injectable, but see below)

```go
if limit > 0 {
    query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
}
```

This repeats the pattern flagged as SEC-003/004/005 in the original review. While the `int` type prevents string injection, the limit value passed here is already capped by the service layer (`recurring_expense_svc.go:117-125`, max 100), so the immediate exploitation surface is low. The concern is code consistency: other repositories do not get this cap, and the pattern is fragile for future contributions.

**Fix:** Use parameterized LIMIT/OFFSET placeholders (SQLite supports `LIMIT ? OFFSET ?`) and pass the values as bind arguments to `scanList`:
```go
query += " LIMIT ? OFFSET ?"
// pass limit, offset to r.db.QueryContext
```

---

### SEC-021 — MEDIUM: OCR Service Errors Forwarded Verbatim Including File Paths

**File:** `internal/handler/ocr_handler.go:66`
**CWE:** CWE-209 (Generation of Error Message Containing Sensitive Information)

```go
respondError(w, http.StatusUnprocessableEntity, err.Error())
```

The error chain from `OCRService.ProcessDocument` can contain filesystem paths from `ocr_svc.go:54`:
```go
fileData, err := os.ReadFile(filePath)
if err != nil {
    return nil, fmt.Errorf("reading document file: %w", err)
}
```

If the file is missing or has bad permissions, the error message sent to the client will include the full absolute path of the document on the server, e.g.:
```
"reading document file: open /home/user/.zfaktury/documents/42/uuid_receipt.pdf: no such file or directory"
```

This leaks the server's home directory, data directory structure, and internal storage paths to any API caller. It also extends the existing SEC-010/SEC-016 pattern to this handler.

**Fix:** Wrap file I/O errors into a generic message before returning:
```go
// In ocr_svc.go
fileData, err := os.ReadFile(filePath)
if err != nil {
    return nil, fmt.Errorf("document file unavailable") // strip the path
}
```

Or, at the handler level, inspect the error type and return a generic message for unexpected errors (file system errors, connection errors) while passing through known validation errors.

---

### SEC-022 — MEDIUM: Arbitrary Date Accepted in Recurring Expense Generate Endpoint

**File:** `internal/handler/recurring_expense_handler.go:321-348`
**CWE:** CWE-20 (Improper Input Validation)

```go
// GeneratePending handles POST /api/v1/recurring-expenses/generate.
func (h *RecurringExpenseHandler) GeneratePending(w http.ResponseWriter, r *http.Request) {
    ...
    if req.AsOfDate != "" {
        asOfDate, err = time.Parse("2006-01-02", req.AsOfDate)
        ...
    } else {
        asOfDate = time.Now()
    }
    count, err := h.svc.GeneratePending(r.Context(), asOfDate)
    ...
```

The `as_of_date` parameter is accepted from any caller with no restriction on the date range. An attacker (or confused frontend) can:

1. **Backdate to a far past date** (e.g., `1970-01-01`) and generate thousands of recurring expense entries in a single request if any recurring expenses are configured, potentially spamming the expenses table.
2. **Forward-date** (e.g., `2099-01-01`) to speculatively generate future expenses, breaking the expected time-series integrity of financial records.

There is a symmetric risk in the `POST /api/v1/recurring-invoices/process-due` endpoint, although that one accepts no body parameter at all and always uses `time.Now()` — meaning it is safe from date manipulation but exposed as an unauthenticated trigger (see SEC-025).

**Fix — add a boundary check:**
```go
const maxPastDays = 7  // only allow generating for the past week
const maxFutureDays = 0 // no future generation via this endpoint

today := time.Now().Truncate(24 * time.Hour)
earliest := today.AddDate(0, 0, -maxPastDays)

if asOfDate.Before(earliest) {
    respondError(w, http.StatusBadRequest, "as_of_date cannot be more than 7 days in the past")
    return
}
if asOfDate.After(today) {
    respondError(w, http.StatusBadRequest, "as_of_date cannot be in the future")
    return
}
```

---

### SEC-023 — LOW: OCR Model Output Content Included in Parse Error Message

**File:** `internal/service/ocr/openai.go:248`
**CWE:** CWE-209 (Information Exposure Through an Error Message)

```go
return nil, fmt.Errorf("parsing OCR JSON from model output: %w (content: %s)", err, truncate(content, 200))
```

When the OCR model returns malformed JSON, the error message includes up to 200 characters of the model's raw output. This error propagates through `ocr_svc.go` to `ocr_handler.go:66` which forwards it verbatim to the HTTP response.

The `content` value is the raw model output from the OpenAI API. While it is unlikely to contain server credentials, it may contain fragments of the scanned invoice (vendor names, amounts, IBANs from the document image) in its raw text, leaking financial document data to the API client in an error context rather than the structured response.

**Fix:** Log the content for debugging but omit it from the returned error:
```go
slog.Debug("failed to parse OCR JSON", "content_preview", truncate(content, 200))
return nil, fmt.Errorf("parsing OCR JSON from model output: %w", err)
```

---

### SEC-024 — LOW: Recurring Invoice Delete Error Forwarded Verbatim

**File:** `internal/handler/recurring_invoice_handler.go:301`
**CWE:** CWE-209 (Information Exposure Through an Error Message)

```go
if err := h.svc.Delete(r.Context(), id); err != nil {
    slog.Error("failed to delete recurring invoice", "error", err, "id", id)
    respondError(w, http.StatusNotFound, err.Error())
    return
}
```

The error from `repo.Delete` contains the internal ID: `"recurring invoice 42 not found or already deleted"`. This confirms to an unauthenticated caller whether a given numeric ID exists in the database (resource enumeration). The same information-leaking pattern is present in the corresponding recurring expense Delete handler at line 279.

**Fix:** Return a fixed `"recurring invoice not found"` message, consistent with other delete handlers in the codebase:
```go
respondError(w, http.StatusNotFound, "recurring invoice not found")
```

---

### SEC-025 — INFO: ProcessDue and GeneratePending Are Unauthenticated Administrative Triggers

**File:** `internal/handler/recurring_invoice_handler.go:193, 332` and `internal/handler/recurring_expense_handler.go:29, 321`
**CWE:** CWE-306 (Missing Authentication for Critical Function — extends SEC-002)

Both `POST /api/v1/recurring-invoices/process-due` and `POST /api/v1/recurring-expenses/generate` are bulk side-effect endpoints that create financial records. Unlike CRUD endpoints, these trigger financial document generation for all due recurring templates in the database. They carry no authentication requirement (SEC-002 is the root cause) and are exposed on the same router as all other API routes.

Any process on the local machine can trigger these repeatedly, generating duplicate draft invoices and expense records. For the expense generate endpoint, the date-range issue noted in SEC-022 amplifies this risk.

**Note:** This is informational because SEC-002 (no authentication on any API route) is the primary finding. Fixing SEC-002 remedies this automatically.

---

## What Was Reviewed and Found Clear

### SQL Injection — CLEAR

All SQL in the reviewed files uses parameterized `?` placeholders exclusively. Dynamic WHERE clause construction in `recurring_expense_repo.go` concatenates only hardcoded SQL fragments with no user input. `LIMIT`/`OFFSET` use integer-typed values (see SEC-020 for the fmt.Sprintf pattern note, but injection is not possible with typed integers).

### Path Traversal — CLEAR

The OCR service reads files via `DocumentService.GetFilePath` (`ocr_svc.go:49`), which validates that the resolved absolute path begins with `{dataDir}/documents/` before returning it (`document_svc.go:178-188`). This correctly prevents reading arbitrary paths.

Credit note and recurring invoice creation in `invoice_svc.go` copies data fields from existing domain objects (loaded from the database) and does not construct any file paths.

### SSRF via OCR — CLEAR

The OpenAI provider calls a hardcoded constant URL `openAIAPIURL = "https://api.openai.com/v1/chat/completions"` (`openai.go:17`). There is no mechanism for user input to influence the target URL. Image data from the document is base64-encoded and sent as a `data:` URI in the JSON body, not as a URL the server fetches — OpenAI's API receives the bytes directly. No SSRF risk is present.

### API Key Logging — CLEAR

The OpenAI API key is stored in `OpenAIProvider.apiKey` (unexported field) and used only to set the `Authorization` header on outbound requests. It is never passed to `slog`, included in error messages, or returned in any response. The startup log line (`slog.Info("OCR service configured", "provider", provider.Name())`) logs only the provider name, not the key.

### Credit Note Logic — CLEAR

`CreateCreditNote` in `invoice_svc.go:276-345` correctly validates that:
- The original invoice exists (line 281)
- It is a regular invoice, not a proforma or credit note (line 286)
- It has been sent or paid before allowing a credit note (line 289)

Partial credit note items are negated at creation time (line 329: `UnitPrice: item.UnitPrice * -1`). The negation happens inside the service layer before reaching the repository — not at render time — so the stored data is correct.

### Recurring Invoice Frequency Validation — CLEAR

`RecurringExpenseService.Create` validates frequency against an allowlist (`validFrequencies` map, `recurring_expense_svc.go:25-30`). The recurring invoice service does not validate frequency but relies on the database `CHECK` constraint or defaults to `domain.FrequencyMonthly`. Consider adding an explicit allowlist check in `recurring_invoice_svc.go:Create` to match the expense service pattern.

### Hardcoded Secrets — CLEAR

No API keys, tokens, passwords, or private keys are present in any reviewed file. The OCR API key flows exclusively from config file to `NewOpenAIProvider` constructor.

### Concurrency — CLEAR

All repository operations use standard `database/sql` with transactions where needed. The recurring invoice and expense processing loops (`ProcessDue`, `GeneratePending`) are synchronous within a single goroutine. No shared mutable state outside the database.

---

## Dependency Vulnerabilities

No new Go dependencies introduced by the reviewed files beyond those already tracked in prior rounds. The OCR feature uses only standard library packages (`net/http`, `encoding/json`, `encoding/base64`, `io`, `bytes`, `fmt`, `time`).

| Package | Status |
|---------|--------|
| All existing Go dependencies | No new CVEs identified |
| No new dependencies added | N/A |

---

## Recommendations

1. **Immediate (before any network exposure):**
   - Fix SEC-019: align OCR client timeout with server WriteTimeout. Minimum fix: set `openAITimeout = 25 * time.Second` in `internal/service/ocr/openai.go`.
   - Fix SEC-018: add `io.LimitReader` wrapping before `io.ReadAll` in `internal/service/ocr/openai.go:73`.

2. **This sprint:**
   - Fix SEC-021: strip file system paths from OCR error messages in `internal/service/ocr_svc.go`.
   - Fix SEC-022: add date boundary validation in `internal/handler/recurring_expense_handler.go:GeneratePending`.
   - Fix SEC-023: move model content out of returned errors, log it with `slog.Debug` instead.
   - Fix SEC-024: return a fixed "not found" string in the recurring invoice Delete handler.

3. **Backlog (open from prior rounds):**
   - SEC-001: ARES response body size cap
   - SEC-002: Authentication on API routes (also resolves SEC-025)
   - SEC-003/004/005/020: Parameterized LIMIT/OFFSET
   - SEC-006/007/008/010/011/012/014/016/017: See prior reports

---

## Cumulative Open Findings (All Rounds)

| ID | Severity | Round | Status | Title |
|----|----------|-------|--------|-------|
| SEC-001 | HIGH | R1 | Open | Unbounded ARES response body |
| SEC-002 | HIGH | R1 | Open | No authentication on API routes |
| SEC-018 | HIGH | R2 | Open | Unbounded OpenAI response body |
| SEC-019 | HIGH | R2 | Open | OCR timeout exceeds server WriteTimeout |
| SEC-003 | MEDIUM | R1 | Open | LIMIT/OFFSET via fmt.Sprintf (invoices) |
| SEC-004 | MEDIUM | R1 | Open | LIMIT/OFFSET via fmt.Sprintf (contacts) |
| SEC-005 | MEDIUM | R1 | Open | LIMIT/OFFSET via fmt.Sprintf (expenses) |
| SEC-006 | MEDIUM | R1 | Open | document_path path traversal risk |
| SEC-007 | MEDIUM | R1 | Open | Settings key allowlist not enforced |
| SEC-012 | MEDIUM | R1 | Open | ISDOC Content-Disposition header injection |
| SEC-020 | MEDIUM | R2 | Open | LIMIT/OFFSET via fmt.Sprintf (recurring expenses) |
| SEC-021 | MEDIUM | R2 | Open | OCR errors expose file system paths |
| SEC-022 | MEDIUM | R2 | Open | Arbitrary as_of_date in generate endpoint |
| SEC-008 | LOW | R1 | Open | CORS wildcard |
| SEC-010 | LOW | R1 | Open | Verbatim error forwarding |
| SEC-014 | LOW | R1 | Open | CSS color injection |
| SEC-016 | LOW | R1 | Open | Verbatim error forwarding (new handlers) |
| SEC-023 | LOW | R2 | Open | OCR model content in error message |
| SEC-024 | LOW | R2 | Open | Recurring invoice delete error leaks ID |
| SEC-009 | LOW | R1 | **FIXED** | No HTTP security headers |
| SEC-013 | MEDIUM | R1 | **FIXED** | Batch ISDOC export unbounded |
| SEC-015 | LOW | R1 | **FIXED** | Zip entry path traversal |
| SEC-011 | INFO | R1 | Open | No upper bound on limit (contacts/expenses) |
| SEC-017 | INFO | R1 | Open | SPD attribute length not validated |
| SEC-025 | INFO | R2 | Open | ProcessDue/GeneratePending unauthenticated (extends SEC-002) |
