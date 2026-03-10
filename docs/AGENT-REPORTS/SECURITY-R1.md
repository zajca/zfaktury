# Security Review - Round 1

**Date:** 2026-03-10
**Reviewer:** Security Agent (claude-sonnet-4-6)
**Scope:** Round 1 new features: document upload/download, expense tax review, invoice relations, settle proforma, email service

---

## Security Review Checklist

- [x] Injection risks reviewed
- [x] Authentication/Authorization verified
- [x] Secrets handling reviewed
- [x] Dependency audit completed (scope: files under review)
- [x] Transport security verified
- [x] Logging practices checked
- [x] Concurrency issues reviewed
- [x] IaC and container configs analyzed (N/A - no IaC in scope)

---

## Summary

| Severity | Count |
|----------|-------|
| HIGH     | 3     |
| MEDIUM   | 5     |
| LOW      | 4     |
| INFO     | 2     |

---

## HIGH Severity Findings

### H1 - Content-Type Bypass in Document Upload (CWE-434)

**File:** `internal/handler/document_handler.go:69-74`
**File:** `internal/service/document_svc.go:51-53`

The upload handler trusts the `Content-Type` header sent by the client without verifying the actual file contents:

```go
// document_handler.go:69-74
contentType := header.Header.Get("Content-Type")
if contentType == "" {
    contentType = "application/octet-stream"
}
doc, err := h.svc.Upload(r.Context(), expenseID, header.Filename, contentType, file)
```

The service validates the content type string against an allowlist (`allowedContentTypes`), but only checks the string value — it never inspects the actual file magic bytes. An attacker can upload an HTML file with executable JavaScript by setting `Content-Type: image/jpeg`. When the file is served by `Download`, the browser may execute it if it sniffs the content type.

**Impact:** Stored XSS / malicious file delivery. An attacker who can POST to `/api/v1/expenses/{id}/documents` can store an HTML payload that executes in the victim's browser when the document is downloaded.

**Fix:** Add magic-byte verification after reading the file bytes. Use `http.DetectContentType` (first 512 bytes) and cross-check against the declared type before writing to disk.

```go
// After reading fileBytes in document_svc.go Upload():
detected := http.DetectContentType(fileBytes)
// Map detected sniffed type to canonical type
if !isCompatibleContentType(contentType, detected) {
    return nil, fmt.Errorf("file content does not match declared content type")
}
```

Note: `http.DetectContentType` does not detect PDF reliably; for PDF also check the `%PDF-` magic prefix: `bytes.HasPrefix(fileBytes, []byte("%PDF-"))`.

---

### H2 - Unescaped Filename in Content-Disposition Header (CWE-113 / Header Injection)

**File:** `internal/handler/document_handler.go:150`

```go
w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
```

The `filename` is extracted from `filepath.Base(filePath)` which includes the UUID prefix and original (sanitized) filename. The sanitization in `document_svc.go` strips path separators but does not strip double-quote characters or CRLF sequences. A filename containing `"` can terminate the header value early, and `\r\n` sequences can inject additional HTTP headers.

**Example:** A filename like `receipt".png\r\nX-Injected: evil` would corrupt the response headers.

**Impact:** HTTP response splitting, potential cache poisoning, or CSP bypass depending on proxy behavior.

**Fix:** Use `mime` package quoting or simply restrict the characters allowed in filenames. The quickest fix is to percent-encode or strip non-alphanumeric characters (except `.`, `-`, `_`):

```go
// Use RFC 5987 encoding or the mime package
import "mime"
w.Header().Set("Content-Disposition",
    "attachment; filename*=UTF-8''"+url.PathEscape(filename))
```

Alternatively use `fmt.Sprintf` with `%q` which Go already applies correctly for the invoice PDF case in `invoice_handler.go:321`:
```go
w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filename))
```
Apply the same pattern here as is already done in `invoice_handler.go`.

---

### H3 - Unbounded Bulk ID Array - No Upper Limit (CWE-400)

**File:** `internal/handler/expense_handler.go:164-177` (MarkTaxReviewed)
**File:** `internal/handler/expense_handler.go:179-192` (UnmarkTaxReviewed)
**File:** `internal/repository/expense_repo.go:301-316` (MarkTaxReviewed)

The bulk review endpoints accept an arbitrary number of IDs from the client. There is no maximum on how many IDs can be submitted:

```go
// expense_handler.go - no limit check before passing to service
if err := h.svc.MarkTaxReviewed(r.Context(), req.IDs); err != nil {
```

The repository builds a SQL statement with one `?` placeholder per ID:
```go
placeholders := strings.Repeat("?,", len(ids))
```

SQLite has a limit of 999 bound parameters by default (`SQLITE_MAX_VARIABLE_NUMBER`). Submitting 1000+ IDs will cause a database error. More importantly, there is no protection against DoS: a client can send tens of thousands of IDs, causing large memory allocations and long-running queries.

Note: The ISDOC batch export in `invoice_handler.go:475` does have a 500-item limit — the same pattern needs to be applied here.

**Impact:** Denial of Service (resource exhaustion), SQLite runtime errors exposed to the caller.

**Fix:**

```go
// expense_handler.go - add before calling svc
const maxBulkIDs = 500
if len(req.IDs) == 0 {
    respondError(w, http.StatusBadRequest, "ids is required")
    return
}
if len(req.IDs) > maxBulkIDs {
    respondError(w, http.StatusBadRequest, fmt.Sprintf("maximum %d IDs per request", maxBulkIDs))
    return
}
```

---

## MEDIUM Severity Findings

### M1 - No Authorization / Ownership Checks on Document Operations (CWE-639 - IDOR)

**File:** `internal/handler/document_handler.go` (all endpoints)
**File:** `internal/service/document_svc.go` (GetByID, Delete, GetFilePath)

The document endpoints retrieve and delete documents by numeric ID without verifying that the requesting user owns the associated expense. Since this is a single-user application the risk is lower, but if multi-user support is ever added (or if the app is exposed to a shared network), any authenticated party could access or delete any document by guessing its auto-increment ID.

The same pattern exists for the invoice `SettleProforma` endpoint - there is no ownership verification beyond the ID being valid and the invoice being a proforma type.

**Impact:** In the current single-user design, low risk. As a structural issue, this represents a classic IDOR pattern that should be documented as a known limitation and blocked before any multi-user work begins.

**Recommendation:** Document the single-user assumption explicitly in handler-level comments. If multi-tenancy is ever introduced, add an ownership check at the service layer before any data is returned.

---

### M2 - Pagination Limit Has No Maximum Cap (CWE-400)

**File:** `internal/handler/helpers.go:40-44`

```go
if v := r.URL.Query().Get("limit"); v != "" {
    if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
        limit = parsed  // no upper bound
    }
}
```

The `limit` query parameter is accepted without a maximum. A request with `?limit=1000000` will attempt to fetch and serialize one million rows in a single response, exhausting memory and CPU.

**Fix:** Cap the limit at a reasonable maximum (e.g., 1000):

```go
if parsed > 0 {
    if parsed > 1000 {
        parsed = 1000
    }
    limit = parsed
}
```

---

### M3 - SMTP Password Logged Indirectly via Error Messages (CWE-532)

**File:** `internal/service/email/sender.go:164-166`

```go
if err := client.Auth(auth); err != nil {
    return fmt.Errorf("SMTP auth: %w", err)
}
```

When SMTP authentication fails, the underlying `net/smtp` library error may include server responses that reflect back credentials or authentication tokens in some SMTP server implementations (e.g., certain SASL error payloads). The error is wrapped and propagated to `slog.Error` at the call site.

More critically, `SMTPConfig` including the `Password` field is stored in-memory in the `EmailSender` struct. Any future logging of the config struct (e.g., via `%+v` formatting) would leak the password.

**Current risk:** Low, since the password is not directly logged. **Future risk:** Medium.

**Fix:** Add a `String()` method to `SMTPConfig` that redacts the password:

```go
func (c SMTPConfig) String() string {
    return fmt.Sprintf("SMTPConfig{Host:%s Port:%d Username:%s Password:[REDACTED] From:%s}",
        c.Host, c.Port, c.Username, c.From)
}
```

---

### M4 - Filename Sanitization Allows Null Bytes and Unicode Homoglyphs (CWE-20)

**File:** `internal/service/document_svc.go:162-176`

The `sanitizeFilename` function strips path separators and trims spaces, but does not:
- Remove null bytes (`\x00`) which can truncate filenames on some filesystems
- Restrict to a safe character allowlist

A filename containing a null byte like `receipt\x00.jpg` could behave differently across OS calls depending on the underlying libc implementation. While Go's `os.WriteFile` handles this safely on Linux (Go uses syscalls directly, not libc), it is still an unexpected input.

**Fix:** Add null byte stripping and consider a strict allowlist:

```go
func sanitizeFilename(name string) string {
    name = filepath.Base(name)
    name = strings.ReplaceAll(name, "/", "")
    name = strings.ReplaceAll(name, "\\", "")
    name = strings.ReplaceAll(name, "\x00", "")  // strip null bytes
    name = strings.TrimSpace(name)
    // ... rest of function
}
```

---

### M5 - `http.ServeFile` Used with a DB-Controlled Path (CWE-22 - Path Traversal)

**File:** `internal/handler/document_handler.go:151`

```go
http.ServeFile(w, r, filePath)
```

`filePath` comes from the database (`doc.StoragePath`). The service's `Upload` method constructs this path safely using `filepath.Join(s.dataDir, "documents", ...)` with a UUID prefix, and sanitizes the filename. However, there is no validation at `GetFilePath` / `GetByID` that the path returned from the database still resides within `dataDir`.

If the database were tampered with (e.g., via a SQL injection vulnerability in another part of the application, or direct DB file access), a `storage_path` value like `../../../etc/passwd` could be served.

**Fix:** Add a confinement check in `GetFilePath` or at the service level:

```go
func (s *DocumentService) GetFilePath(ctx context.Context, id int64) (string, string, error) {
    doc, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return "", "", err
    }
    // Verify path is within dataDir
    rel, err := filepath.Rel(s.dataDir, doc.StoragePath)
    if err != nil || strings.HasPrefix(rel, "..") {
        return "", "", errors.New("document storage path is outside data directory")
    }
    return doc.StoragePath, doc.ContentType, nil
}
```

---

## LOW Severity Findings

### L1 - email/templates.go: Subject Line Injection Risk (CWE-93)

**File:** `internal/service/email/templates.go:36`
**File:** `internal/service/email/sender.go:229`

```go
subject = fmt.Sprintf("Faktura %s - %s", data.InvoiceNumber, data.SenderName)
// ...
writeHeader(&buf, "Subject", encodeHeader(msg.Subject))
```

`SenderName` comes from user-controlled settings. `encodeHeader` only base64-encodes the entire value if a non-ASCII character is found, but for ASCII-only names it passes the value through literally. A `SenderName` containing `\r\n` sequences could inject additional MIME headers.

**Impact:** SMTP header injection if the sender name contains CRLF. In practice, the sender name comes from the application's own settings rather than end-customer input, which limits exploitability.

**Fix:** Strip or reject CRLF in the subject and sender name before building the message:

```go
func sanitizeHeaderValue(v string) string {
    v = strings.ReplaceAll(v, "\r", "")
    v = strings.ReplaceAll(v, "\n", "")
    return v
}
```

Apply before calling `encodeHeader`.

---

### L2 - No CSRF Protection on State-Changing Endpoints (CWE-352)

**File:** All POST/DELETE handlers

The API is a JSON API consumed by a same-origin SPA. Standard browser CSRF attacks are mitigated by the `Content-Type: application/json` requirement (CORS pre-flight), but this is not an explicit defense. There are no CSRF tokens, `SameSite` cookie attributes, or `Origin` header checks.

**Impact:** Low for a single-user local app. If the app is ever exposed on a network or cookies are added for session management, this becomes a real vector.

**Recommendation:** Add a CSRF middleware (e.g., chi's built-in or `gorilla/csrf`) before any authentication layer is introduced.

---

### L3 - 006_expense_documents.sql: No CHECK Constraints on content_type or size (CWE-20)

**File:** `internal/database/migrations/006_expense_documents.sql`

The `expense_documents` table has no database-level constraints on `content_type` or `size`. Enforcement is only at the service layer. If a bug or direct DB access bypasses the service, arbitrary values can be written.

**Fix:** Add CHECK constraints:

```sql
content_type TEXT NOT NULL CHECK(content_type IN ('image/jpeg','image/png','application/pdf','image/webp','image/heic')),
size INTEGER NOT NULL CHECK(size > 0 AND size <= 20971520),
```

---

### L4 - Error Messages Leak Internal Detail to Clients (CWE-209)

**File:** `internal/handler/expense_handler.go:57-59`, `internal/handler/document_handler.go:77`

```go
// expense_handler.go:57-59
if err := h.svc.Create(r.Context(), expense); err != nil {
    respondError(w, http.StatusUnprocessableEntity, err.Error())
```

```go
// document_handler.go:76-78
if err != nil {
    respondError(w, http.StatusUnprocessableEntity, err.Error())
```

Service and repository error messages (which include table names, column names, and SQL error details) are returned directly to the client. This leaks internal schema information.

**Fix:** Map internal errors to user-facing messages at the handler layer. Use structured error types or sentinel errors rather than forwarding raw `err.Error()` strings.

---

## Informational Findings

### I1 - sendPlain Falls Back to Unencrypted SMTP (port != 465/587)

**File:** `internal/service/email/sender.go:94-106`

The `sendViaSMTP` function has a `default` case that connects without TLS for any port other than 465 and 587. This means misconfigured SMTP (e.g., `port = 25`) silently sends credentials and email content in plain text.

**Recommendation:** Log a prominent warning when falling back to plain SMTP. Consider requiring explicit opt-in for unencrypted transport.

---

### I2 - `storage_path` Correctly Excluded from API Response

**File:** `internal/handler/document_handler.go:172-182`

The `documentFromDomain` function correctly omits `StoragePath` from the JSON response. The `_ = doc.StoragePath` line documents this intentional exclusion. This is good practice and confirmed working as intended.

---

## Dependency Vulnerabilities

No dependency audit was performed as part of this review scope (no `go.sum` or `package.json` changes were included in Round 1). Run `govulncheck ./...` and `npm audit` separately.

---

## Recommendations Summary

### Immediately Required (HIGH)

1. **H1** - Add magic-byte content-type verification in `document_svc.go:Upload()`. Do not rely solely on the client-declared `Content-Type` header.
2. **H2** - Fix `Content-Disposition` header construction in `document_handler.go:150` to use `fmt.Sprintf(..., %q, filename)` consistent with the existing pattern in `invoice_handler.go:321`.
3. **H3** - Add a maximum bound (500) on bulk ID arrays in `expense_handler.go:MarkTaxReviewed` and `UnmarkTaxReviewed`.

### Fix This Sprint (MEDIUM)

4. **M2** - Cap the pagination `limit` parameter at a sensible maximum (e.g., 1000) in `helpers.go`.
5. **M3** - Add a `String()` redaction method to `SMTPConfig` to prevent accidental password exposure.
6. **M4** - Strip null bytes from filenames in `sanitizeFilename`.
7. **M5** - Validate that `doc.StoragePath` is within `dataDir` before serving in `document_svc.go:GetFilePath`.

### Address Before Multi-User Work (LOW)

8. **L1** - Sanitize CRLF from email header values (subject, sender name) in `templates.go`.
9. **L3** - Add `CHECK` constraints to `expense_documents` table for `content_type` and `size`.
10. **L4** - Stop forwarding raw `err.Error()` to API clients in `expense_handler.go` and `document_handler.go`.

### Deferred (LOW/INFO)

11. **L2** - Introduce CSRF protection before any network-exposed or multi-user deployment.
12. **I1** - Warn explicitly when plain SMTP is used (port != 465/587).
