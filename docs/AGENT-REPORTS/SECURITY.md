# Security Review - RFC-001 Foundation Phase

**Date:** 2026-03-10
**Scope:** All new code in commits a3142a3..83fde30
**Model:** claude-sonnet-4-6

---

## Security Review Checklist

- [x] Injection risks reviewed
- [x] Authentication/Authorization verified
- [x] Secrets handling reviewed
- [x] Dependency audit completed (manual, no known CVEs in go.mod)
- [x] Transport security verified
- [x] Logging practices checked
- [x] Concurrency issues reviewed
- [x] IaC and container configs analyzed (N/A - single binary)

---

## Findings Summary

| ID | Severity | File | Line(s) | Title |
|----|----------|------|---------|-------|
| SEC-001 | HIGH | `internal/ares/client.go` | 97 | Unbounded ARES response body read |
| SEC-002 | HIGH | `internal/handler/router.go` | 36-39 | No authentication on any API route |
| SEC-003 | MEDIUM | `internal/repository/invoice_repo.go` | 343 | LIMIT/OFFSET interpolated via fmt.Sprintf |
| SEC-004 | MEDIUM | `internal/repository/contact_repo.go` | 159 | LIMIT/OFFSET interpolated via fmt.Sprintf |
| SEC-005 | MEDIUM | `internal/repository/expense_repo.go` | 208 | LIMIT/OFFSET interpolated via fmt.Sprintf |
| SEC-006 | MEDIUM | `internal/handler/helpers.go` | 430-440 | document_path stored/returned without validation |
| SEC-007 | MEDIUM | `internal/service/settings_svc.go` | 69-76 | Settings key allowlist not enforced |
| SEC-008 | LOW | `internal/handler/router.go` | 92-105 | CORS wildcard active in dev, no guard against accidental production use |
| SEC-009 | LOW | `internal/handler/router.go` | (all) | No HTTP security headers |
| SEC-010 | LOW | `internal/handler/invoice_handler.go` | 52 | Service errors forwarded verbatim to client (information leak) |
| SEC-011 | INFO | `internal/handler/helpers.go` | 37-51 | No upper bound on `limit` in contact/expense handlers |

---

## Detailed Findings

### SEC-001 - HIGH - Unbounded ARES response body read
**CWE-400** (Uncontrolled Resource Consumption)

**File:** `/home/zajca/Code/Me/ZFaktury/internal/ares/client.go` line 97

```go
body, err := io.ReadAll(resp.Body)
```

`io.ReadAll` will buffer the entire response into memory. A malicious or misconfigured upstream server (or a SSRF redirect — see note below) could return a multi-gigabyte body and exhaust memory. The ARES base URL is configurable via `WithBaseURL`, which also means a compromised config could point this client at an attacker-controlled host.

**Fix — add a size cap before reading:**
```go
const maxAresResponseBytes = 1 << 20 // 1 MB
body, err := io.ReadAll(io.LimitReader(resp.Body, maxAresResponseBytes+1))
if err != nil {
    return nil, fmt.Errorf("reading ARES response: %w", err)
}
if int64(len(body)) > maxAresResponseBytes {
    return nil, errors.New("ARES response too large")
}
```

**SSRF note:** The ICO input is validated with `icoRegexp` (8 digits) before use in the URL path, so direct SSRF via user-controlled ICO is not possible. The risk is limited to the `WithBaseURL` option, which is only reachable at startup from trusted configuration.

---

### SEC-002 - HIGH - No authentication on any API route
**CWE-306** (Missing Authentication for Critical Function)

**File:** `/home/zajca/Code/Me/ZFaktury/internal/handler/router.go` lines 42-54

The entire `/api/v1` surface (contacts, invoices, expenses, settings — including financial data and IBANs) is accessible without any authentication token or session cookie. The application is described as a single-user local tool, but the server listens on a network socket. Any process or browser tab on the same machine (or LAN if the bind address is not localhost) can read and write all data.

**Recommended mitigations:**
1. Bind the HTTP server to `127.0.0.1` only (document and enforce in config).
2. If multi-user or network access is ever desired, add a middleware that checks a shared secret token passed in the `Authorization` header or via a session cookie with `HttpOnly; SameSite=Strict`.

---

### SEC-003 / SEC-004 / SEC-005 - MEDIUM - LIMIT/OFFSET injected via fmt.Sprintf
**CWE-89** (Improper Neutralization of Special Elements in SQL Command)

**Files:**
- `/home/zajca/Code/Me/ZFaktury/internal/repository/invoice_repo.go` line 343
- `/home/zajca/Code/Me/ZFaktury/internal/repository/contact_repo.go` line 159
- `/home/zajca/Code/Me/ZFaktury/internal/repository/expense_repo.go` line 208

```go
query += fmt.Sprintf(" LIMIT %d OFFSET %d", filter.Limit, filter.Offset)
```

`filter.Limit` and `filter.Offset` are `int` values parsed from query parameters in `parsePagination`. Because they are typed integers (not strings), injection is not directly possible here. However:

1. The InvoiceService caps `Limit` to 100 (`invoice_svc.go` line 142), but the ContactService and ExpenseService do NOT enforce an upper cap.
2. A caller can send `limit=2147483647` and get a full table dump in one query — a denial-of-service / data exfiltration risk.

The deeper concern is that the `where` string itself is built by concatenation of hardcoded fragments plus `?` placeholders. The current code is correct, but this pattern is fragile: future contributors may accidentally concatenate a user-supplied value instead of using a placeholder.

**Fix:**
- Use parameterized placeholders for LIMIT/OFFSET (SQLite supports `LIMIT ? OFFSET ?`).
- Add a maximum-limit cap (e.g. 500) in all three services, mirroring the invoice service pattern.

```go
// Safe alternative: pass limit/offset as bind params
query += " LIMIT ? OFFSET ?"
args = append(args, filter.Limit, filter.Offset)
rows, err := r.db.QueryContext(ctx, query, args...)
```

---

### SEC-006 - MEDIUM - document_path stored without path validation
**CWE-22** (Path Traversal)

**File:** `/home/zajca/Code/Me/ZFaktury/internal/handler/helpers.go` lines 437, 456

`document_path` is accepted from the client as a free-form string:
```go
DocumentPath string `json:"document_path"`
```

It is stored in the database and later returned in API responses. Currently no code reads the file at this path, so there is no immediate path-traversal exploit. However:

1. If any future feature (PDF preview, file serve endpoint) reads `DocumentPath` from the database and opens the file, a stored value of `../../etc/passwd` or similar would be exploitable.
2. The path is returned verbatim to API consumers, potentially leaking server filesystem layout.

**Fix:** Either strip `document_path` to a filename-only value (no directory separators) when persisting, or validate at the service layer using `filepath.Clean` and confirming the cleaned path remains within an allowed base directory.

```go
// In expense service Create/Update:
if expense.DocumentPath != "" {
    base := "/home/user/.zfaktury/documents" // from config
    clean := filepath.Clean(filepath.Join(base, filepath.Base(expense.DocumentPath)))
    if !strings.HasPrefix(clean, base+string(os.PathSeparator)) {
        return errors.New("invalid document path")
    }
    expense.DocumentPath = clean
}
```

---

### SEC-007 - MEDIUM - Settings key allowlist not enforced
**CWE-915** (Improperly Controlled Modification of Dynamically-Determined Object Attributes)

**File:** `/home/zajca/Code/Me/ZFaktury/internal/service/settings_svc.go` lines 59-66

`SetBulk` validates only that keys are non-empty and under 255 characters. The API endpoint at `PUT /api/v1/settings` accepts an arbitrary `map[string]string` body and writes every key directly to the database. An attacker (or confused frontend) can write arbitrary keys such as `__proto__`, `admin`, or future internal keys that may be checked elsewhere in the application.

The service already defines constants (`SettingCompanyName`, `SettingICO`, etc.). The allowlist should be enforced.

**Fix:**
```go
var validKeys = map[string]bool{
    SettingCompanyName: true, SettingICO: true, SettingDIC: true,
    SettingVATRegistered: true, SettingStreet: true, SettingCity: true,
    SettingZIP: true, SettingEmail: true, SettingPhone: true,
    SettingBankAccount: true, SettingBankCode: true,
    SettingIBAN: true, SettingSWIFT: true,
}

func validateKey(key string) error {
    if !validKeys[key] {
        return fmt.Errorf("unknown setting key %q", key)
    }
    return nil
}
```

---

### SEC-008 - LOW - CORS wildcard origin with no production guard
**CWE-942** (Permissive Cross-domain Policy)

**File:** `/home/zajca/Code/Me/ZFaktury/internal/handler/router.go` lines 92-105

The `corsMiddleware` sets `Access-Control-Allow-Origin: *` and is gated behind `cfg.DevMode`. The risk is that `DevMode` is set by the caller (`cli/serve.go`) via a flag and could accidentally be enabled in a deployed instance. With a wildcard CORS policy and no authentication (SEC-002), any malicious website visited by the user could issue cross-origin API calls and exfiltrate all invoice/contact data.

**Fix:** Even in dev mode, restrict the allowed origin to `http://localhost:5173` (Vite's default) rather than `*`. In production (non-dev) mode, set a restrictive `Access-Control-Allow-Origin` or omit the header entirely.

---

### SEC-009 - LOW - No HTTP security response headers
**CWE-693** (Protection Mechanism Failure)

**File:** `/home/zajca/Code/Me/ZFaktury/internal/handler/router.go`

No security headers are set on responses:
- `X-Content-Type-Options: nosniff` — missing (MIME sniffing attacks)
- `X-Frame-Options: DENY` — missing (clickjacking)
- `Referrer-Policy: no-referrer` — missing
- `Content-Security-Policy` — missing

While risk is lower for a local-only app, these headers are cheap to add via a middleware and follow defense-in-depth.

**Fix (add a middleware):**
```go
func securityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("Referrer-Policy", "no-referrer")
        next.ServeHTTP(w, r)
    })
}
```

---

### SEC-010 - LOW - Service error messages forwarded verbatim to HTTP clients
**CWE-209** (Generation of Error Message Containing Sensitive Information)

**File:** `/home/zajca/Code/Me/ZFaktury/internal/handler/invoice_handler.go` line 52, and similar in other handlers

```go
respondError(w, http.StatusUnprocessableEntity, err.Error())
```

Internal error strings from services and repositories (e.g., `"invoice 42 not found: sql: no rows in result set"`) are sent directly to the client. These can leak internal IDs, SQL driver names, and schema details to an attacker.

**Fix:** For 4xx validation errors (known, expected errors from the service layer), forwarding is acceptable. For unexpected errors (5xx), log the detail and return only a generic message:
```go
// Only forward known validation errors
if isValidationError(err) {
    respondError(w, http.StatusUnprocessableEntity, err.Error())
} else {
    slog.Error("unexpected error", "error", err)
    respondError(w, http.StatusInternalServerError, "internal error")
}
```

---

### SEC-011 - INFO - No upper bound on limit in contact/expense handlers
**CWE-400** (Uncontrolled Resource Consumption)

**File:** `/home/zajca/Code/Me/ZFaktury/internal/handler/helpers.go` lines 37-51

`parsePagination` accepts any positive integer for `limit`. The InvoiceService caps it to 100, but `ContactService` and `ExpenseService` do not apply the same cap. A request with `limit=1000000` will attempt to load the entire table.

**Fix:** Apply the same cap (e.g., 500) in both services, or enforce it centrally in `parsePagination`.

---

## Dependency Vulnerabilities

No CVEs identified via manual review of `go.mod` dependencies. All pinned versions appear current as of 2026-03. A full automated scan with `govulncheck ./...` is recommended as part of CI.

| Package | Current | Notes |
|---------|---------|-------|
| `github.com/go-chi/chi/v5` | v5.2.5 | No known CVEs |
| `modernc.org/sqlite` | v1.46.1 | No known CVEs |
| `github.com/pressly/goose/v3` | v3.27.0 | No known CVEs |
| `github.com/johnfercher/maroto/v2` | v2.3.3 | No known CVEs |

---

## XSS Assessment

All Svelte templates reviewed (`contacts/+page.svelte`, `contacts/[id]/+page.svelte`, `invoices/+page.svelte`, `invoices/[id]/+page.svelte`, `expenses/+page.svelte`, `expenses/[id]/+page.svelte`) use standard Svelte interpolation (`{value}`) exclusively. No `{@html ...}` directives are present in any source file. Svelte's compiler escapes all interpolated values by default.

**Result: No XSS risk found in Svelte templates.**

---

## Recommendations

1. **Immediately (before any network exposure):** Fix SEC-001 (body size limit) and SEC-002 (bind to 127.0.0.1 or add auth token).
2. **This sprint:** Fix SEC-007 (settings allowlist), SEC-003/004/005 (LIMIT/OFFSET parameterization), SEC-006 (document_path validation).
3. **Low-priority:** Add security headers (SEC-009), restrict CORS (SEC-008), refine error messages (SEC-010).
4. Add `govulncheck ./...` to CI pipeline.
5. Add `npm audit --production` to frontend CI pipeline.

---

## Next Steps

1. Apply fixes for SEC-001 and SEC-002 immediately.
2. Open tickets for SEC-003 through SEC-007.
3. Configure `govulncheck` and `npm audit` in CI.
4. Re-run this audit after implementing authentication.

---

---

# Security Review - Commit 1987968 (RFC-002 Sub-Phase 2A + RFC-003 Task 3)

**Date:** 2026-03-10
**Commit:** `19879687db7bead2ca055f7d56658c4e4f489d34`
**Scope:** PDF generation, QR payment, ISDOC export, invoice sequences, expense categories
**Model:** claude-sonnet-4-6

---

## Security Review Checklist

- [x] Injection risks reviewed
- [x] Authentication/Authorization verified
- [x] Secrets handling reviewed
- [x] Dependency audit completed (manual)
- [x] Transport security verified
- [x] Logging practices checked
- [x] Concurrency issues reviewed
- [x] IaC and container configs analyzed (N/A)

---

## Findings Summary

| ID | Severity | File | Line(s) | Title |
|----|----------|------|---------|-------|
| SEC-012 | MEDIUM | `internal/handler/invoice_handler.go` | 398-400 | Header injection via InvoiceNumber in ISDOC Content-Disposition |
| SEC-013 | MEDIUM | `internal/handler/invoice_handler.go` | 411-458 | No upper bound on batch ISDOC export invoice count |
| SEC-014 | LOW | `frontend/src/routes/settings/categories/+page.svelte` | 236 | Unvalidated color value injected into inline CSS style attribute |
| SEC-015 | LOW | `internal/handler/invoice_handler.go` | 450-451 | InvoiceNumber used unsanitized as zip entry filename |
| SEC-016 | LOW | `internal/handler/sequence_handler.go` | 64, 108 | Service errors forwarded verbatim in 422 response (extends SEC-010) |
| SEC-017 | INFO | `internal/pdf/qr_payment.go` | 41-47 | SPD extended attributes not length-validated per SPAYD spec |

---

## Detailed Findings

### SEC-012 — MEDIUM: Header Injection via InvoiceNumber in ISDOC Content-Disposition

**File:** `internal/handler/invoice_handler.go:398-400`
**CWE:** CWE-113 (Improper Neutralization of CRLF Sequences in HTTP Headers)

```go
filename := fmt.Sprintf("%s.isdoc", invoice.InvoiceNumber)
w.Header().Set("Content-Type", "application/xml")
w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
```

`invoice.InvoiceNumber` is stored in the database and rendered directly into the `Content-Disposition` header value without stripping double-quotes, backslashes, or CRLF characters. Go's `net/http` rejects bare `\r\n` in `Header.Set`, which limits full header splitting exploitability, but embedded double-quotes break RFC 6266 filename token parsing (e.g. a number containing `"` produces `filename="FV2026".isdoc"` which some HTTP clients misparse).

This is inconsistent with the PDF endpoint at line 291-293 which correctly uses `%q` (Go's quoted-string format, which escapes `"` and `\`):

```go
// PDF endpoint (line 293) - correctly uses %q:
w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

// ISDOC endpoint (line 400) - raw interpolation into quoted string:
w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
```

**Fix:** Apply the same `%q` approach already used for PDF:

```go
filename := fmt.Sprintf("%s.isdoc", invoice.InvoiceNumber)
w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
```

---

### SEC-013 — MEDIUM: No Upper Bound on Batch ISDOC Export Size

**File:** `internal/handler/invoice_handler.go:411-458`
**CWE:** CWE-400 (Uncontrolled Resource Consumption)

```go
func (h *InvoiceHandler) ExportISDOCBatch(w http.ResponseWriter, r *http.Request) {
    var req isdocBatchRequest
    ...
    if len(req.InvoiceIDs) == 0 {
        respondError(w, http.StatusBadRequest, "invoice_ids is required")
        return
    }
    // No maximum length check on req.InvoiceIDs
    for _, id := range req.InvoiceIDs {
        invoice, _ := h.svc.GetByID(...)     // 1 DB query per ID
        xmlData, _ := h.isdocGen.Generate(...)  // 1 XML marshal per ID
        f, _ := zipWriter.Create(filename)
        f.Write(xmlData)
    }
}
```

An unbounded `invoice_ids` slice means the loop runs for however many IDs are submitted. The response is streamed so memory is bounded by individual XML documents, but each ID requires a DB round-trip and XML marshal. Submitting 100,000 IDs will keep the goroutine and a DB connection occupied for the full processing time with no timeout beyond Go's default HTTP server timeout.

**Fix:** Add a maximum cap immediately after the empty check:

```go
const maxBatchISDOC = 500

if len(req.InvoiceIDs) > maxBatchISDOC {
    respondError(w, http.StatusBadRequest,
        fmt.Sprintf("too many invoice IDs: maximum is %d", maxBatchISDOC))
    return
}
```

---

### SEC-014 — LOW: Unvalidated Color Value in Inline CSS Style Attribute

**File:** `frontend/src/routes/settings/categories/+page.svelte:234-237`
**CWE:** CWE-79 (XSS via CSS Injection) — limited to CSS property context

```svelte
<div
    class="h-5 w-5 rounded-full border border-gray-200"
    style="background-color: {cat.color}"
></div>
```

Svelte escapes `{}` interpolations in text nodes and regular HTML attribute values, but interpolation into a `style` attribute that is itself constructed as a string passes the value through the CSS property context. A `cat.color` value of `red; } body { display:none` would break the page layout. Modern browsers block `javascript:` in CSS values, so script execution is not realistically achievable.

The backend service (`category_svc.go:validateCategory`) does not validate the `Color` field format at all, so any string can be stored.

**Fix (backend):** Add color format validation to `internal/service/category_svc.go`:

```go
var colorHexPattern = regexp.MustCompile(`^#[0-9a-fA-F]{3}(?:[0-9a-fA-F]{3})?$`)

// In validateCategory:
if cat.Color != "" && !colorHexPattern.MatchString(cat.Color) {
    return errors.New("color must be a valid hex color (e.g. #6B7280)")
}
```

**Fix (frontend, defense-in-depth):** Use Svelte's `style:` directive which applies the value to a specific property without allowing further CSS structure:

```svelte
<div
    class="h-5 w-5 rounded-full border border-gray-200"
    style:background-color={cat.color}
></div>
```

The `style:property={value}` directive sets a single CSS property and does not allow injecting additional rules.

---

### SEC-015 — LOW: InvoiceNumber Used as Zip Entry Filename Without Sanitization

**File:** `internal/handler/invoice_handler.go:450-451`
**CWE:** CWE-22 (Path Traversal — limited to zip entry paths)

```go
filename := fmt.Sprintf("%s.isdoc", invoice.InvoiceNumber)
f, err := zipWriter.Create(filename)
```

`archive/zip.Create` does not validate or sanitize the entry name. If `InvoiceNumber` contains `../` (e.g. `../../etc/passwd`), the resulting zip archive will contain an entry with a path traversal name. This does not affect the server itself but may be exploitable by a client's zip extraction tool if it auto-extracts without path sanitization (a known class of vulnerability in many CLI zip tools, e.g. zip slip).

Since invoice numbers are generated by the sequence formatter (`{prefix}{year}{number:04d}`) and consist only of alphanumeric characters under normal operation, this is only reachable if the `invoice_number` stored in the database has been manipulated directly or via a bug in sequence formatting.

**Fix:** Sanitize the filename before using it as a zip entry name:

```go
import "path/filepath"

safeName := filepath.Base(invoice.InvoiceNumber)
filename := safeName + ".isdoc"
f, err := zipWriter.Create(filename)
```

Additionally, add validation to sequence prefix creation to reject any prefix containing `/`, `\`, or `.`.

---

### SEC-016 — LOW: Service Errors Forwarded Verbatim in 422 Responses (extends SEC-010)

**Files:**
- `internal/handler/sequence_handler.go:64` (Create)
- `internal/handler/sequence_handler.go:108` (Update)
- `internal/handler/category_handler.go:61` (Create)
- `internal/handler/category_handler.go:85` (Update)

Same pattern as SEC-010 from the previous review. For the specific validation errors generated in the new code this is intentional behavior, but it also means any unexpected repository error that bubbles up (e.g. a DB constraint error) will surface with its full message to the client.

This finding extends the prior SEC-010 recommendation to cover the new handlers. Consolidate by implementing a `ValidationError` sentinel type to distinguish expected from unexpected errors at the handler boundary.

---

### SEC-017 — INFO: SPD Extended Attributes Not Length-Validated per SPAYD Spec

**File:** `internal/pdf/qr_payment.go:41-47`
**CWE:** CWE-20 (Improper Input Validation) — data integrity, not security

```go
p.SetExtendedAttribute("VS", invoice.VariableSymbol)
p.SetExtendedAttribute("KS", invoice.ConstantSymbol)
```

The Czech SPAYD specification limits `VS` (variable symbol) to 10 digits and `KS` (constant symbol) to 4 digits. Passing values outside these constraints produces a QR code that is technically non-compliant and may be rejected by banking apps. The `qrpay` library does not validate these constraints internally.

**Fix:** Validate or truncate at the service layer when creating/updating invoices, or add guards in `GenerateQRPayment`:

```go
if len(invoice.VariableSymbol) > 10 {
    return nil, fmt.Errorf("variable symbol exceeds 10 characters")
}
if len(invoice.ConstantSymbol) > 4 {
    return nil, fmt.Errorf("constant symbol exceeds 4 characters")
}
```

---

## What Was Reviewed and Found Clear

### SQL Injection — CLEAR

All three new repositories (`sequence_repo.go`, `category_repo.go`) and the modified `invoice_repo.go` use exclusively parameterized queries. Dynamic `WHERE` construction in `invoice_repo.go:296-343` concatenates only hardcoded SQL fragments; user inputs are bound as `?` parameters. `LIMIT`/`OFFSET` are integer-typed values parsed by `strconv.Atoi`, not string-interpolated.

### XML Injection — CLEAR

ISDOC generation in `internal/isdoc/generator.go` uses `encoding/xml.MarshalIndent` exclusively. All user-supplied strings are assigned to Go struct fields and encoded by the standard library marshaller, which escapes `<`, `>`, `&`, `"`, and `'` automatically. No manual XML string building occurs anywhere.

### Path Traversal in PDF — CLEAR

`internal/pdf/supplier.go` loads all data from the settings service (database). It does not construct file paths from user input. The `LogoPath` field in `SupplierInfo` exists in the struct definition but is never populated by `LoadSupplierFromSettings`; no file read occurs. If logo loading is added in future, path validation will be mandatory.

### Frontend XSS — CLEAR (with caveat at SEC-014)

Svelte 5 escapes all `{}` interpolations in text nodes and standard HTML attributes. All new pages (`sequences/+page.svelte`, `categories/+page.svelte`, `CategoryPicker.svelte`, `invoices/[id]/+page.svelte` additions) render user data via text interpolation only. The `{@html ...}` directive is absent from all new code. The sole exception is the inline `style` attribute covered by SEC-014.

### QR Payment SPD Injection — CLEAR

The `qrpay` library receives user data through typed setters (`SetIBAN`, `SetBIC`, `SetAmount`, `SetMessage`). IBAN and SWIFT/BIC inputs are validated by the library's setters with errors checked and returned. No raw SPD string concatenation occurs in application code.

### Concurrency in Sequence Counter — CLEAR

`GetNextNumber` in `invoice_repo.go` uses a proper database transaction (`BeginTx`/`Commit`/`defer Rollback`) to atomically read and increment `next_number`. The `defer tx.Rollback()` after a successful `Commit` is a harmless no-op per Go's `database/sql` contract.

### Hardcoded Secrets — CLEAR

No credentials, API keys, tokens, or private keys present in any new file.

---

## New Dependency Audit

| Package | Purpose | Known CVEs at review date |
|---------|---------|--------------------------|
| `github.com/johnfercher/maroto/v2` | PDF generation | None known |
| `github.com/dundee/qrpay` | Czech SPAYD QR codes | None known |

Automated scan via `govulncheck ./...` remains blocked by missing GCC in the build environment. Recommend enabling as part of CI with `CGO_ENABLED=0`.

---

## Recommendations for Commit 1987968

1. **Before next release:** Apply the one-line fix for SEC-012 (change `%s` to `%q` in ISDOC Content-Disposition header) — identical to the existing pattern on the PDF endpoint.
2. **Before next release:** Add the 500-item cap for SEC-013 (batch ISDOC export).
3. **This sprint:** Add `colorHexPattern` validation to `category_svc.go` for SEC-014 and switch the Svelte template to `style:background-color` directive.
4. **This sprint:** Add `filepath.Base` sanitization to batch export zip entry names for SEC-015.
5. **Backlog:** Implement a `ValidationError` sentinel type to address SEC-010/SEC-016 across all handlers.

---

## Cumulative Open Findings (All Commits)

| ID | Severity | Status | Title |
|----|----------|--------|-------|
| SEC-001 | HIGH | Open | Unbounded ARES response body |
| SEC-002 | HIGH | Open | No authentication on API routes |
| SEC-003 | MEDIUM | Open | LIMIT/OFFSET via fmt.Sprintf |
| SEC-004 | MEDIUM | Open | LIMIT/OFFSET via fmt.Sprintf (contacts) |
| SEC-005 | MEDIUM | Open | LIMIT/OFFSET via fmt.Sprintf (expenses) |
| SEC-006 | MEDIUM | Open | document_path path traversal risk |
| SEC-007 | MEDIUM | Open | Settings key allowlist not enforced |
| SEC-012 | MEDIUM | Open | ISDOC Content-Disposition header injection |
| SEC-013 | MEDIUM | Open | Batch ISDOC export unbounded |
| SEC-008 | LOW | Open | CORS wildcard |
| SEC-009 | LOW | Open | No security headers |
| SEC-010 | LOW | Open | Verbatim error forwarding |
| SEC-014 | LOW | Open | CSS color injection |
| SEC-015 | LOW | Open | Zip entry path traversal |
| SEC-016 | LOW | Open | Verbatim error forwarding (new handlers) |
| SEC-011 | INFO | Open | Unbounded limit in contact/expense |
| SEC-017 | INFO | Open | SPD attribute length not validated |
