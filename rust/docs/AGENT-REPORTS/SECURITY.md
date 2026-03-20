# Security Review Report - ZFaktury Rust Codebase

**Date:** 2026-03-20
**Scope:** `/home/zajca/Code/Me/ZFaktury/rust/crates/`
**Reviewer:** Security Agent (automated)

---

## Security Review Checklist

- [x] Injection risks reviewed
- [x] Authentication/Authorization verified
- [x] Secrets handling reviewed
- [x] Dependency audit (static — no CVE scanner available in this environment)
- [x] Transport security verified
- [x] Logging practices checked
- [x] Concurrency issues reviewed
- [x] IaC and container configs analyzed (N/A — desktop app)

---

## Findings

### CRITICAL

None.

---

### HIGH

#### H1 — SQL Injection via Integer Array Interpolation in `mark_tax_reviewed` / `unmark_tax_reviewed`

**File:** `/home/zajca/Code/Me/ZFaktury/rust/crates/zfaktury-db/src/repos/expense_repo.rs`
**Lines:** 326–353
**CWE:** CWE-89

The `ids` slice is serialized to a comma-separated string and embedded directly into the SQL via `format!`. Although the values come from an `i64` slice (and are therefore numeric), the construction pattern is incorrect by design: if the caller contract ever changes, or if the type widens to `String`, there is no barrier.

More concretely, the `unmark_tax_reviewed` path (line 349) already passes `[]` (no bound parameters) even though the placeholders string is embedded in the query — the params do not match the positional `?N` pattern at all for the IDs. This means the code is relying entirely on `i64::to_string()` being safe, rather than using the database driver's parameterization for the list.

```rust
// Current (risky pattern — correct only because i64 cannot contain SQL metacharacters):
let placeholders: String = ids.iter().map(|id| id.to_string()).collect::<Vec<_>>().join(",");
conn.execute(
    &format!("UPDATE expenses SET tax_reviewed_at = ?1 WHERE id IN ({placeholders}) AND deleted_at IS NULL"),
    params![now_str],
)
```

**Recommended fix** — use `?`-parameterized placeholders and pass each id as a bound parameter:

```rust
let placeholders: String = ids.iter().map(|_| "?").collect::<Vec<_>>().join(",");
// Build params tuple dynamically via rusqlite's params_from_iter:
conn.execute(
    &format!("UPDATE expenses SET tax_reviewed_at = ?1 WHERE id IN ({placeholders})"),
    rusqlite::params_from_iter(
        std::iter::once(now_str as &dyn rusqlite::types::ToSql)
            .chain(ids.iter().map(|id| id as &dyn rusqlite::types::ToSql))
    ),
)
```

Note: `rusqlite` does not support binding a `Vec<i64>` directly as `?1`; the correct approach is building a `?,...,?` placeholder string and using `params_from_iter`. The `unmark_tax_reviewed` variant has the same issue and must also be fixed.

---

#### H2 — Plaintext SMTP Fallback (`builder_dangerous`)

**File:** `/home/zajca/Code/Me/ZFaktury/rust/crates/zfaktury-api/src/email.rs`
**Lines:** 184–191
**CWE:** CWE-319

Any port that is not 465 or 587 falls into the `builder_dangerous` branch, which uses `lettre`'s unencrypted transport. Port 25, port 2525, and custom ports all end up here. The function name `builder_dangerous` is itself lettre's signal that this path sends credentials in cleartext.

```rust
port => {
    // Plain SMTP (e.g., localhost relay).
    let mut builder = SmtpTransport::builder_dangerous(&self.config.host).port(port);
    ...
}
```

This is acceptable for localhost relay (port 25, loopback only), but the code does not enforce that the host is `127.0.0.1` or `localhost` before allowing the plaintext path. A misconfigured `config.toml` with `port = 25` and a remote SMTP host will silently send passwords over the network without encryption.

**Recommended fix:**

```rust
port => {
    // Only allow plaintext for loopback hosts.
    let is_local = matches!(
        self.config.host.as_str(),
        "127.0.0.1" | "::1" | "localhost"
    );
    if !is_local {
        return Err(ApiError::SmtpError(
            "plaintext SMTP (non-465/587 port) is only allowed for loopback hosts".to_string(),
        ));
    }
    let mut builder = SmtpTransport::builder_dangerous(&self.config.host).port(port);
    ...
}
```

---

### MEDIUM

#### M1 — LIMIT/OFFSET Injected into SQL String from `i64` Filter Fields

**Files:**
- `/home/zajca/Code/Me/ZFaktury/rust/crates/zfaktury-db/src/repos/contact_repo.rs` line 242
- `/home/zajca/Code/Me/ZFaktury/rust/crates/zfaktury-db/src/repos/invoice_repo.rs` line 400
- `/home/zajca/Code/Me/ZFaktury/rust/crates/zfaktury-db/src/repos/expense_repo.rs` line 296
- `/home/zajca/Code/Me/ZFaktury/rust/crates/zfaktury-db/src/repos/audit_log_repo.rs` line 91
- `/home/zajca/Code/Me/ZFaktury/rust/crates/zfaktury-db/src/repos/recurring_expense_repo.rs` line 122
**CWE:** CWE-89

All list queries embed `filter.limit` and `filter.offset` as raw integer strings:

```rust
query.push_str(&format!(" LIMIT {} OFFSET {}", filter.limit, filter.offset));
```

Because both fields are typed `i64`, they cannot currently carry SQL metacharacters. However, this practice is fragile — the safety is implicit, not structural. If filter types are ever loosened, or if the pattern is copied with a string field, the injection barrier disappears.

SQLite supports `LIMIT ?1 OFFSET ?2` with bound parameters. Using parameterized LIMIT/OFFSET is cheap, more idiomatic, and eliminates the category entirely.

**Recommended fix (example for contact_repo):**

```rust
if filter.limit > 0 {
    param_values.push(Box::new(filter.limit));
    let lim_idx = param_values.len();
    param_values.push(Box::new(filter.offset));
    let off_idx = param_values.len();
    query.push_str(&format!(" LIMIT ?{lim_idx} OFFSET ?{off_idx}"));
}
```

---

#### M2 — OAuth2 Error Body Forwarded Verbatim to Caller

**File:** `/home/zajca/Code/Me/ZFaktury/rust/crates/zfaktury-api/src/fakturoid.rs`
**Lines:** 284–288
**CWE:** CWE-209

When Fakturoid token exchange fails, the raw HTTP response body is embedded in the error:

```rust
let body = resp.text().unwrap_or_default();
return Err(ApiError::OAuthError(format!("HTTP {}: {}", status, body)));
```

Depending on the OAuth server, the error body may contain hints about the client secret or account identifiers that aid an attacker who can observe error messages. The same pattern exists in `list_paginated` (line 401) and `AnthropicProvider::process_image` (lines 127–133).

**Recommended fix:** Truncate or sanitize external error bodies before including them in errors that may surface in logs or UI. The `truncate(&body, 500)` helper already exists in `ares.rs` and `ocr.rs` — use it consistently for all external API error bodies.

```rust
return Err(ApiError::OAuthError(format!("HTTP {}: {}", status, truncate(&body, 200))));
```

---

#### M3 — `download_attachment` Accepts Arbitrary URLs

**File:** `/home/zajca/Code/Me/ZFaktury/rust/crates/zfaktury-api/src/fakturoid.rs`
**Lines:** 340–377
**CWE:** CWE-918 (SSRF)

`download_attachment` takes a `download_url: &str` and issues a GET with the Bearer token attached, without verifying that the URL belongs to Fakturoid's domain:

```rust
pub fn download_attachment(&self, download_url: &str) -> Result<(Vec<u8>, String)> {
    ...
    let resp = self.client.get(download_url)
        .header("Authorization", format!("Bearer {}", token))
        ...
```

If an attacker can influence the `download_url` field in a Fakturoid API response (or in imported data), the Fakturoid Bearer token is forwarded to an arbitrary server. This is a Server-Side Request Forgery / credential-forwarding risk.

In the current architecture, the URL comes from Fakturoid's own API response, which limits exploitability. However, the defence-in-depth fix is simple:

**Recommended fix:**

```rust
const FAKTUROID_HOST: &str = "app.fakturoid.cz";

pub fn download_attachment(&self, download_url: &str) -> Result<(Vec<u8>, String)> {
    let parsed = url::Url::parse(download_url)
        .map_err(|_| ApiError::InvalidInput("invalid attachment URL".to_string()))?;
    if parsed.host_str() != Some(FAKTUROID_HOST) {
        return Err(ApiError::InvalidInput(
            "attachment URL must be on app.fakturoid.cz".to_string(),
        ));
    }
    ...
}
```

---

#### M4 — Settings Key Logged in Error Messages

**File:** `/home/zajca/Code/Me/ZFaktury/rust/crates/zfaktury-db/src/repos/settings_repo.rs`
**Lines:** 65, 80, 108
**CWE:** CWE-532

Settings keys such as `smtp_password`, `api_key`, `ocr_api_key`, etc. are interpolated directly into log error messages:

```rust
log::error!("querying setting {key}: {e}");
log::error!("upserting setting {key}: {e}");
log::error!("upserting setting {key} in bulk: {e}");
```

If sensitive values are stored under keys with descriptive names (e.g., `smtp_password`, `fio_token`), the key name appears in log files. This is low-severity on its own, but combined with a log-exfiltration scenario it confirms the existence of stored credentials.

**Recommended fix:** Replace the key with a fixed placeholder in error logs, or redact sensitive-sounding keys:

```rust
log::error!("querying setting (key redacted): {e}");
```

---

### LOW

#### L1 — Mutex Poison on Concurrent Panics in Repositories

**Files:** All `SqliteXxxRepo` implementations
**CWE:** CWE-667

Every repository holds a `Mutex<Connection>` and calls `.lock().unwrap()`. If a thread panics while holding the lock, the mutex becomes poisoned and all subsequent `.unwrap()` calls will panic — effectively hanging or crashing the application. The GPUI desktop architecture currently runs a single UI thread, so this is low risk now, but as background tasks grow it becomes a latent reliability issue.

**Recommended fix:**

```rust
// Replace:
let conn = self.conn.lock().unwrap();
// With:
let conn = self.conn.lock().unwrap_or_else(|e| e.into_inner());
```

---

#### L2 — `format!`-Constructed WHERE Clauses with Parameterized Values Are Safe But Brittle

**Files:** `contact_repo.rs`, `invoice_repo.rs`, `expense_repo.rs`, `audit_log_repo.rs`
**CWE:** CWE-89 (potential)

All filter-driven `WHERE` clauses build the predicate string with `format!` but correctly use `?{idx}` positional placeholders for every user-supplied value. This is structurally sound for the current code. The risk is that the pattern is easy to misuse — a future contributor adding a string filter could accidentally interpolate the value directly instead of using a parameter index.

This is a code-quality concern rather than an active vulnerability. Consider adding a comment at each `where_clause` construction site noting that all user-supplied values must be bound via `?{idx}`.

---

#### L3 — ARES URL Contains ICO in Path Without Further Escaping

**File:** `/home/zajca/Code/Me/ZFaktury/rust/crates/zfaktury-api/src/ares.rs`
**Lines:** 48–56
**CWE:** CWE-20

```rust
let url = format!("{}/ekonomicke-subjekty/{}", self.base_url, ico);
```

The ICO is validated to be exactly 8 ASCII digits (line 50) before the URL is constructed. This validation is sufficient — 8 decimal digits cannot introduce path traversal or URL injection. No change required; noting it for completeness.

---

#### L4 — `SmtpTransport::send` Error May Include SMTP Banner/Hostname

**File:** `/home/zajca/Code/Me/ZFaktury/rust/crates/zfaktury-api/src/email.rs`
**Line:** 68
**CWE:** CWE-209

```rust
.map_err(|e| ApiError::SmtpError(format!("sending email: {}", e)))?;
```

Lettre's transport errors can include the remote SMTP server's greeting banner, IP address, or TLS negotiation details. These details are benign in a single-user desktop app but would constitute information leakage in a multi-user context. Acceptable for now; document the risk for any future web-API layer.

---

#### L5 — OCR API Key Stored as Plain String in `AnthropicProvider` / `OpenAIProvider`

**File:** `/home/zajca/Code/Me/ZFaktury/rust/crates/zfaktury-api/src/ocr.rs`
**Lines:** 60–63, 163–166
**CWE:** CWE-312

API keys are stored as `String` fields in the provider structs. In a desktop app loaded from config, this is acceptable. The risk is that any future serialization (e.g., state dump, crash report) could expose the key. Mitigation: ensure `Debug` is not derived for structs containing API keys, or implement a redacting `Debug`.

Currently `AnthropicProvider` and `OpenAIProvider` do not derive `Debug`, which is correct. No change required.

---

## Positive Security Observations

These items represent good practice observed during the review:

1. **All SQL queries are fully parameterized** for user-supplied values — every filter value uses `?{idx}` binding, not string interpolation. The H1 finding is the only exception, and it is limited to numeric types.

2. **ICO validation is strict** — exactly 8 ASCII digits, validated before the URL is constructed and before any database lookup.

3. **SMTP port 465 uses implicit TLS, port 587 uses STARTTLS with `Tls::Required`** — neither allows downgrade to cleartext. The only cleartext path requires the dangerous builder, which is clearly named.

4. **OAuth2 tokens are never logged** — the `authenticate()` method stores the token in memory only; no `log::debug!` or `println!` exposes it.

5. **`download_attachment` checks for an empty/missing token** before making the request (line 341–343).

6. **Foreign keys are enabled and WAL mode is set** in `connection.rs` — proper SQLite configuration.

7. **OCR content-type is validated against an allowlist** before sending to the external API.

8. **S3 config defaults `use_ssl = true`** — secure by default.

9. **Error messages use `DomainError` variants without leaking internal SQL detail** to callers — repo errors are mapped to opaque variants (`NotFound`, `InvalidInput`).

---

## Recommendations

1. **Fix H1 immediately** — replace the `ids.to_string()` SQL interpolation pattern with `params_from_iter` in `mark_tax_reviewed` and `unmark_tax_reviewed` in `expense_repo.rs`.

2. **Fix H2** — add a host allowlist check before the `builder_dangerous` SMTP path. A one-line guard (`localhost`/`127.0.0.1`) is sufficient.

3. **Address M1 in the next sprint** — parameterize LIMIT/OFFSET across all list queries. The change is mechanical and low-risk.

4. **Address M3** — add a URL host validation guard in `download_attachment` to prevent Bearer token forwarding to non-Fakturoid hosts.

5. **Address M2 and M4** — apply the existing `truncate()` helper to all external HTTP error bodies, and remove settings key names from error log messages.

6. **Document the `where_clause` pattern (L2)** with a comment in the repos that use it, to prevent future contributors from accidentally bypassing parameterization.
