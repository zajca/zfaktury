# Security Review - Round 3

**Date:** 2026-03-10
**Scope:** New files from Round 3 — CNB exchange rates, overdue detection, status history, payment reminders, and email templating
**Files reviewed:**
- `internal/service/cnb/client.go`
- `internal/handler/exchange_handler.go`
- `internal/repository/invoice_repo_overdue.go`
- `internal/repository/status_history_repo.go`
- `internal/service/overdue_svc.go`
- `internal/handler/status_history_handler.go`
- `internal/repository/reminder_repo.go`
- `internal/service/reminder_svc.go`
- `internal/service/email/reminder_template.go`
- `internal/service/email/sender.go`
- `internal/handler/reminder_handler.go`
- `internal/handler/router.go`
- `internal/cli/serve.go`
- `internal/database/migrations/012_invoice_status_history.sql`
- `internal/database/migrations/013_payment_reminders.sql`
**Model:** claude-sonnet-4-6

---

## Security Review Checklist

- [x] Injection risks reviewed
- [x] Authentication/Authorization verified
- [x] Secrets handling reviewed
- [x] Dependency audit reviewed
- [x] Transport security verified (SMTP TLS)
- [x] Logging practices checked
- [x] Concurrency issues reviewed (CNB cache)
- [x] IaC and container configs analyzed (N/A)

---

## Previous Findings Status Update

| ID | Title | Status in R3 code |
|----|-------|-------------------|
| SEC-001 | Unbounded ARES response body | Still open |
| SEC-002 | No authentication on API routes | Still open — new endpoints extend the surface |
| SEC-018 | Unbounded OpenAI response body | Still open |
| SEC-019 | OCR timeout exceeds WriteTimeout | Still open |
| SEC-003/004/005/020 | LIMIT/OFFSET via fmt.Sprintf | Still open — not repeated in R3 files |
| SEC-021 | OCR errors expose file paths | Still open |
| SEC-022 | Arbitrary as_of_date in generate endpoint | Still open |
| SEC-025 | ProcessDue/GeneratePending unauthenticated | Still open — new CheckOverdue extends the pattern |

---

## New Findings Summary

| ID | Severity | File | Line(s) | Title |
|----|----------|------|---------|-------|
| SEC-026 | HIGH | `internal/service/cnb/client.go` | 124 | Unbounded CNB response body read |
| SEC-027 | HIGH | `internal/handler/reminder_handler.go` | 74 | SendReminder errors forwarded verbatim including SMTP/email internals |
| SEC-028 | HIGH | `internal/service/email/sender.go` | 75-80 | Customer email address logged in plain text |
| SEC-029 | MEDIUM | `internal/handler/exchange_handler.go` | 66 | CNB upstream error forwarded verbatim to client |
| SEC-030 | MEDIUM | `internal/service/email/reminder_template.go` | 42-58 | Email header/body injection via unstripped CRLF in user-controlled fields |
| SEC-031 | MEDIUM | `internal/handler/status_history_handler.go` | 70 | CheckOverdue is an unauthenticated administrative trigger |
| SEC-032 | LOW | `internal/database/migrations/012_invoice_status_history.sql` | 4 | Missing CASCADE on FK — orphaned history rows on hard delete |
| SEC-033 | INFO | `internal/cli/serve.go` | 118 | CNB client always wired regardless of configuration |

---

## Detailed Findings

### SEC-026 — HIGH: Unbounded CNB Response Body Read

**File:** `internal/service/cnb/client.go:124`
**CWE:** CWE-400 (Uncontrolled Resource Consumption)

```go
return parseRates(resp.Body)
```

`parseRates` receives `resp.Body` directly and wraps it in a `bufio.Scanner`. The default `bufio.Scanner` token size is 64 KB per line, but there is no cap on the total number of lines or total bytes read. A rogue or compromised CNB endpoint (or a DNS-hijacked response in a hostile network) could stream an arbitrarily large body, consuming unbounded memory.

This is the third instance of the same pattern (SEC-001 for ARES, SEC-018 for OpenAI). The `http.Client` timeout of 10 seconds provides a time-based bound but not a memory bound — a 10-second stream at modest bandwidth can still deliver tens of megabytes.

**Fix — wrap the body before passing to the scanner:**
```go
const maxCNBResponseBytes = 256 * 1024 // 256 KB is generous for ~35 currency lines

limited := io.LimitReader(resp.Body, maxCNBResponseBytes+1)
rates, err := parseRates(limited)
if err != nil {
    return nil, err
}
// Optionally: detect truncation by checking len(rates) == 0 after limiting
```

---

### SEC-027 — HIGH: SendReminder Errors Forwarded Verbatim

**File:** `internal/handler/reminder_handler.go:74`
**CWE:** CWE-209 (Generation of Error Message Containing Sensitive Information)

```go
respondError(w, http.StatusUnprocessableEntity, err.Error())
```

The error chain from `ReminderService.SendReminder` can expose several categories of sensitive information to the caller:

1. **SMTP internals** — when email delivery fails, `email/sender.go` wraps errors as `"email sender: SMTP auth: ..."`, `"email sender: SMTP MAIL FROM: ..."`, or `"email sender: TLS dial smtp.example.com:465: ..."`. These expose the SMTP host, port, and authentication state to the API caller.

2. **Customer email address confirmation** — the success path returns `"invoice is not overdue"` or `"customer has no email address"`, confirming to an unauthenticated caller that a given invoice ID's customer has or does not have an email address on file.

3. **Internal invoice state** — `"getting invoice: invoice 42 not found or already deleted"` leaks whether invoice IDs exist (resource enumeration, same pattern as SEC-010/024).

**Fix — classify errors at the handler boundary:**
```go
reminder, err := h.svc.SendReminder(r.Context(), invoiceID)
if err != nil {
    slog.Error("failed to send payment reminder", "invoice_id", invoiceID, "error", err)
    // Return specific user-facing messages for known validation errors.
    switch {
    case errors.Is(err, service.ErrInvoiceNotFound):
        respondError(w, http.StatusNotFound, "invoice not found")
    case errors.Is(err, service.ErrNotOverdue):
        respondError(w, http.StatusUnprocessableEntity, "invoice is not overdue")
    case errors.Is(err, service.ErrNoCustomerEmail):
        respondError(w, http.StatusUnprocessableEntity, "customer has no email address")
    default:
        respondError(w, http.StatusInternalServerError, "failed to send reminder")
    }
    return
}
```

This requires defining sentinel errors in `reminder_svc.go` for the known business-logic rejection cases and returning them from `SendReminder`.

---

### SEC-028 — HIGH: Customer Email Address Logged in Plain Text

**File:** `internal/service/email/sender.go:75-80`
**CWE:** CWE-532 (Insertion of Sensitive Information into Log File)

```go
slog.Info("sending email",
    "to", msg.To,
    "subject", msg.Subject,
    "smtp_host", s.cfg.Host,
    "smtp_port", s.cfg.Port,
)
```

The full `To` address list (including the customer's email address) is written to the application log on every outgoing email. Payment reminder emails sent to overdue customers therefore cause their email addresses to appear in the server log, which persists on disk in `logs/`.

For a sole-proprietor invoicing tool, the number of unique email addresses is small and this is not immediately critical. However, if logs are ever rotated to a less-restricted location, shared with support services, or shipped to a log aggregator (even accidentally via a CI/CD artifact), this constitutes a personal data leak under GDPR Article 5(1)(f) (integrity and confidentiality).

**Fix — hash or redact the recipient list in logs:**
```go
slog.Info("sending email",
    "recipient_count", len(msg.To),
    "subject", msg.Subject,
    "smtp_host", s.cfg.Host,
    "smtp_port", s.cfg.Port,
)
```

If the exact recipient is needed for debugging, log only the domain part:
```go
func redactEmail(addr string) string {
    at := strings.LastIndex(addr, "@")
    if at < 0 {
        return "[invalid]"
    }
    return "***@" + addr[at+1:]
}
```

---

### SEC-029 — MEDIUM: CNB Upstream Error Forwarded Verbatim to Client

**File:** `internal/handler/exchange_handler.go:66`
**CWE:** CWE-209 (Generation of Error Message Containing Sensitive Information)

```go
respondError(w, http.StatusBadGateway, "failed to fetch exchange rate: "+err.Error())
```

The error from `cnbClient.GetRate` is concatenated directly into the response. The error chain in `cnb/client.go` wraps the raw `net/http` error, which can include:

- The full CNB URL including the `date` query parameter that the client constructed: `"fetching CNB rates: Get \"https://www.cnb.cz/cs/...?date=02.01.2006\": dial tcp: ..."`
- Internal DNS resolution failures that expose the server's network configuration
- TLS certificate errors that reveal the configured CA trust store

The currency code and date are user-supplied but already validated, so the injected values are benign. The risk is the wrapping of raw network errors.

**Fix — return a fixed upstream error message:**
```go
rate, err := h.cnbClient.GetRate(r.Context(), currency, date)
if err != nil {
    slog.Warn("failed to fetch CNB exchange rate", "currency", currency, "date", date, "error", err)
    respondError(w, http.StatusBadGateway, "exchange rate unavailable, try again later")
    return
}
```

---

### SEC-030 — MEDIUM: Email Header Injection via Unstripped CRLF in User-Controlled Fields

**File:** `internal/service/email/reminder_template.go:42-58` and `internal/service/email/sender.go:225`
**CWE:** CWE-93 (Improper Neutralization of CRLF Sequences — 'CRLF Injection')

The reminder email Subject header is constructed by embedding `d.InvoiceNumber` directly:

```go
// reminder_template.go:42
subject := fmt.Sprintf("Pripomenuti splatnosti faktury %s", d.InvoiceNumber)
```

This `subject` string is written verbatim into the MIME `Subject:` header in `sender.go:229`:

```go
writeHeader(&buf, "Subject", encodeHeader(msg.Subject))
```

`encodeHeader` only base64-encodes the value when it contains non-ASCII characters. For pure ASCII input it returns the string unchanged. If `InvoiceNumber` contains a CRLF sequence (e.g., `"\r\nBcc: attacker@evil.com"`), the resulting raw SMTP message gains an injected header line.

Invoice numbers are generated by the server's own sequence system and should be safe in practice. However, the fields `d.InvoiceNumber`, `d.BankAccount`, `d.VariableSymbol`, and `d.UserName` all originate from user-editable database fields (the user configures their name, bank account, and variable symbol). If any of these fields are set to a value containing CRLF, the injection is possible.

The same fields also flow into the plain-text and HTML body via `fmt.Sprintf`, but `escapeHTML` handles the HTML path for HTML-specific characters — it does not strip CRLF.

**Fix — strip CRLF from all user-controlled values before use in headers:**
```go
// In sender.go or a shared utility:
func sanitizeHeaderValue(s string) string {
    s = strings.ReplaceAll(s, "\r", "")
    s = strings.ReplaceAll(s, "\n", "")
    return s
}

// Applied in writeHeader:
func writeHeader(buf *bytes.Buffer, key, value string) {
    buf.WriteString(key)
    buf.WriteString(": ")
    buf.WriteString(sanitizeHeaderValue(value))
    buf.WriteString("\r\n")
}
```

For the template body, CRLF in user data becomes a visual line break in the plain-text email — unlikely to cause harm but worth stripping from sensitive fields like `UserName` and `BankAccount` at the template boundary.

---

### SEC-031 — MEDIUM: CheckOverdue Is an Unauthenticated Administrative Trigger

**File:** `internal/handler/status_history_handler.go:70` and `internal/handler/router.go:93`
**CWE:** CWE-306 (Missing Authentication for Critical Function — extends SEC-002/SEC-025)

```go
// router.go:93
api.Post("/invoices/check-overdue", statusHistoryHandler.CheckOverdue)
```

`POST /api/v1/invoices/check-overdue` triggers a bulk status transition for all `sent` invoices past their due date, writing status history records for each. Like `ProcessDue` (SEC-025), this endpoint has no authentication guard, no rate limiting, and no idempotency protection.

Repeated invocation by any process on the local machine:
- Creates duplicate `invoice_status_history` rows for the same invoice if it is repeatedly transitioned (the service calls `UpdateStatus` then `Create` in a loop — a second invocation between the two steps would re-create history records)
- Fills the `invoice_status_history` table with noise, degrading the usefulness of the audit trail

**Note:** This is informational in the same way as SEC-025 — fixing SEC-002 (authentication) resolves it. The specific additional risk here is the corrupted status history, which SEC-025 (recurring invoice) does not have, so the severity is escalated to MEDIUM.

**Immediate mitigation (without full auth):** Add a configurable secret token check or restrict the endpoint to `127.0.0.1` only at the HTTP server level until proper authentication is implemented.

---

### SEC-032 — LOW: Missing CASCADE on Foreign Keys in New Migrations

**File:** `internal/database/migrations/012_invoice_status_history.sql:4` and `013_payment_reminders.sql:4`
**CWE:** CWE-459 (Incomplete Cleanup)

```sql
-- 012:
invoice_id INTEGER NOT NULL REFERENCES invoices(id),

-- 013:
invoice_id INTEGER NOT NULL REFERENCES invoices(id),
```

Neither table specifies `ON DELETE CASCADE`. The application uses soft deletes (`deleted_at` column) for invoices, so hard deletes are not part of the normal workflow. However, if any administrative cleanup ever hard-deletes an invoice row, SQLite with `PRAGMA foreign_keys = ON` will reject the delete, and the status history and reminder records for that invoice will become unreachable orphans if `foreign_keys` is later disabled.

More practically, if a future migration or CLI command hard-deletes invoices without disabling foreign key enforcement first, those operations will fail silently or with a constraint error, potentially leaving the database in a partially cleaned state.

**Fix:**
```sql
invoice_id INTEGER NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
```

Alternatively, `ON DELETE RESTRICT` makes the constraint explicit rather than accidental. Either is preferable to the implicit default behavior.

---

### SEC-033 — INFO: CNB Client Always Wired Unconditionally

**File:** `internal/cli/serve.go:118`
**CWE:** N/A (operational concern)

```go
// Wire CNB client.
cnbClient := cnb.NewClient()
```

Unlike the OCR service (which is wired conditionally on `cfg.OCR.APIKey != ""`), the CNB client is always instantiated and always passed to the router. The CNB exchange rate endpoint is therefore always active, making outbound HTTP connections to `cnb.cz` on every request to `GET /api/v1/exchange-rate`.

This is not directly a vulnerability, but:
1. Users who do not need exchange rate functionality get an always-on outbound HTTP endpoint with no way to disable it via config.
2. If the CNB URL were ever blocked by a corporate firewall or changed to require credentials, the error would surface at runtime through user-facing API responses rather than at startup with a meaningful configuration error.

**Fix (optional):** Add a config flag to disable the exchange rate endpoint, consistent with how OCR is handled:
```go
var cnbClient *cnb.Client
if cfg.CNB.Enabled { // or check a feature flag
    cnbClient = cnb.NewClient()
}
```

The router already handles `cnbClient == nil` correctly (line 85-88 of `router.go`), so this change would be backward-compatible.

---

## What Was Reviewed and Found Clear

### SQL Injection — CLEAR

All SQL in `invoice_repo_overdue.go`, `status_history_repo.go`, and `reminder_repo.go` uses parameterized `?` placeholders exclusively. No dynamic SQL construction was observed in R3 files. The `LIMIT`/`OFFSET` fmt.Sprintf pattern from prior rounds is not repeated here.

### XSS in Email HTML — CLEAR

`reminder_template.go` uses a hand-rolled `escapeHTML` function that correctly escapes `&`, `<`, `>`, and `"`. All user-controlled fields (`CustomerName`, `InvoiceNumber`, `TotalAmount`, `DueDate`, `DaysOverdue`, `BankAccount`, `VariableSymbol`, `UserName`) pass through `escapeHTML` before being embedded in the `<body>` HTML via the pattern:

```go
html := wrapHTML(subject, strings.ReplaceAll(escapeHTML(text), "\n", "<br>\n"))
```

The text body is first built as a plain string, then HTML-escaped as a whole, so field values are escaped in context. Note that `wrapHTML` applies `escapeHTML` again to the `title` parameter (subject line in the `<title>` tag), so the `<title>` element is also safe. No raw HTML injection is possible through the template data fields.

### SSRF via CNB Client — CLEAR

`cnb/client.go` constructs the request URL as `fmt.Sprintf("%s?date=%s", c.baseURL, dateKey)` where `baseURL` is a package-level constant and `dateKey` is a server-formatted date string (`date.Format("02.01.2006")`), never user input. The currency code from the query parameter is validated by `currencyCodeRegex` to be exactly three uppercase ASCII letters before reaching the CNB client. There is no mechanism for user input to influence the target URL. No SSRF risk is present.

### CNB Cache Concurrency — CLEAR

`getRatesForDate` in `cnb/client.go` uses a `sync.RWMutex` correctly: read lock for cache lookup, write lock for cache update. The check-then-fetch pattern (read unlock then write lock) has a benign TOCTOU window where two concurrent requests for the same date may both fetch from CNB — this is a performance redundancy rather than a security issue, and the final cache state is correct either way.

### SMTP TLS Configuration — CLEAR

`sender.go` enforces `tls.VersionTLS12` as the minimum TLS version for both implicit TLS (port 465) and STARTTLS (port 587) connections. The plain SMTP path is only used for port configurations other than 465 and 587, which is appropriate for local relay setups. No credentials are accepted over plaintext SMTP by default.

### Email Validation — ACCEPTABLE

The customer email address stored in `domain.Contact.Email` is not re-validated before use as an SMTP `RCPT TO` recipient in the reminder flow. The SMTP server itself performs recipient validation and will reject malformed addresses. Since the email comes from the user's own contact database (entered by the sole proprietor, not by a third party), this is an acceptable risk for this application's threat model.

### Overdue Detection Logic — CLEAR

`overdue_svc.go` retrieves candidate IDs first, then updates status and records history per-ID in a loop. Errors per-ID are logged and skipped without aborting the batch, which is correct for a background sweep. The `isOverdue` helper in `reminder_svc.go` correctly handles both `overdue` status and the `sent`-past-due-date edge case.

### Migration Integrity — CLEAR (except SEC-032)

Both migrations use correct goose format with Up/Down sections. Column types match the domain: `TEXT` for timestamps (consistent with project convention), `INTEGER` for monetary amounts (not used here), `INTEGER` for IDs. Indexes are created on the foreign key columns for join performance. The only concern is the missing CASCADE noted in SEC-032.

---

## Dependency Vulnerabilities

No new Go module dependencies were introduced in R3. The CNB client uses only standard library packages (`bufio`, `net/http`, `io`, `strconv`, `strings`, `sync`, `time`). The reminder service and email sender use only standard library packages.

| Package | Status |
|---------|--------|
| All existing Go dependencies | No new CVEs identified in R3 |
| No new dependencies added | N/A |

---

## Recommendations

1. **Immediate (before any network exposure):**
   - Fix SEC-026: add `io.LimitReader` in `cnb/client.go:fetchRates` before passing `resp.Body` to `parseRates`. Suggested cap: 256 KB.
   - Fix SEC-027: define sentinel errors in `reminder_svc.go` and classify them at the handler boundary instead of forwarding raw error strings.
   - Fix SEC-030: define email header sanitization (CRLF stripping) in `sender.go:writeHeader` to block header injection from user-controlled fields.

2. **This sprint:**
   - Fix SEC-028: remove `msg.To` from the `slog.Info` call in `sender.go`. Log only recipient count and domain.
   - Fix SEC-029: replace verbatim CNB error forwarding in `exchange_handler.go:66` with a fixed "exchange rate unavailable" message. Log the raw error with `slog.Warn`.
   - Fix SEC-031: add the same idempotency or rate-limiting protection to `CheckOverdue` as recommended for `ProcessDue` in SEC-025. At minimum, add a note that this must be addressed alongside SEC-002.
   - Fix SEC-032: apply `ON DELETE CASCADE` or `ON DELETE RESTRICT` to `invoice_id` in both new migration tables. This requires new migration files — do not modify existing migrations.

3. **Backlog (open from prior rounds):**
   - SEC-001: ARES response body size cap
   - SEC-002: Authentication on API routes (also resolves SEC-025, SEC-031)
   - SEC-003/004/005/020: Parameterized LIMIT/OFFSET
   - SEC-018: OpenAI response body size cap
   - SEC-019: OCR timeout alignment
   - SEC-021/022/023/024: Error message leakage (OCR and recurring handlers)
   - SEC-006/007/008/010/011/012/014/016/017: See R1 report

---

## Cumulative Open Findings (All Rounds)

| ID | Severity | Round | Status | Title |
|----|----------|-------|--------|-------|
| SEC-001 | HIGH | R1 | Open | Unbounded ARES response body |
| SEC-002 | HIGH | R1 | Open | No authentication on API routes |
| SEC-018 | HIGH | R2 | Open | Unbounded OpenAI response body |
| SEC-019 | HIGH | R2 | Open | OCR timeout exceeds server WriteTimeout |
| SEC-026 | HIGH | R3 | Open | Unbounded CNB response body read |
| SEC-027 | HIGH | R3 | Open | SendReminder errors forwarded verbatim (SMTP/email internals) |
| SEC-028 | HIGH | R3 | Open | Customer email address written to logs in plain text |
| SEC-003 | MEDIUM | R1 | Open | LIMIT/OFFSET via fmt.Sprintf (invoices) |
| SEC-004 | MEDIUM | R1 | Open | LIMIT/OFFSET via fmt.Sprintf (contacts) |
| SEC-005 | MEDIUM | R1 | Open | LIMIT/OFFSET via fmt.Sprintf (expenses) |
| SEC-006 | MEDIUM | R1 | Open | document_path path traversal risk |
| SEC-007 | MEDIUM | R1 | Open | Settings key allowlist not enforced |
| SEC-012 | MEDIUM | R1 | Open | ISDOC Content-Disposition header injection |
| SEC-020 | MEDIUM | R2 | Open | LIMIT/OFFSET via fmt.Sprintf (recurring expenses) |
| SEC-021 | MEDIUM | R2 | Open | OCR errors expose file system paths |
| SEC-022 | MEDIUM | R2 | Open | Arbitrary as_of_date in generate endpoint |
| SEC-029 | MEDIUM | R3 | Open | CNB upstream error forwarded verbatim to client |
| SEC-030 | MEDIUM | R3 | Open | Email header injection via CRLF in user-controlled fields |
| SEC-031 | MEDIUM | R3 | Open | CheckOverdue unauthenticated trigger (extends SEC-002/025) |
| SEC-008 | LOW | R1 | Open | CORS wildcard |
| SEC-010 | LOW | R1 | Open | Verbatim error forwarding |
| SEC-014 | LOW | R1 | Open | CSS color injection |
| SEC-016 | LOW | R1 | Open | Verbatim error forwarding (new handlers) |
| SEC-023 | LOW | R2 | Open | OCR model content in error message |
| SEC-024 | LOW | R2 | Open | Recurring invoice delete error leaks ID |
| SEC-032 | LOW | R3 | Open | Missing CASCADE on FK in migrations 012/013 |
| SEC-009 | LOW | R1 | **FIXED** | No HTTP security headers |
| SEC-013 | MEDIUM | R1 | **FIXED** | Batch ISDOC export unbounded |
| SEC-015 | LOW | R1 | **FIXED** | Zip entry path traversal |
| SEC-011 | INFO | R1 | Open | No upper bound on limit (contacts/expenses) |
| SEC-017 | INFO | R1 | Open | SPD attribute length not validated |
| SEC-025 | INFO | R2 | Open | ProcessDue/GeneratePending unauthenticated (extends SEC-002) |
| SEC-033 | INFO | R3 | Open | CNB client always wired unconditionally |
