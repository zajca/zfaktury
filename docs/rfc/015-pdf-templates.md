# RFC-015: Customizable PDF Templates

**Status:** Draft
**Date:** 2026-03-15

## Summary

Allow users to customize invoice PDF appearance: upload company logo, set accent color, customize footer text, and choose between compact/detailed layout variants. Configuration stored in the settings table, logo file in the documents directory.

## Background

The current PDF generator (`internal/pdf/invoice_pdf.go`) uses a hardcoded layout with no customization. Users cannot add their company logo, change colors, or modify the footer. This is one of the most requested features for a professional invoicing tool — clients receiving invoices expect branded documents.

## Design

### Settings

New settings keys (stored in `settings` table):

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `pdf.logo_path` | string | `""` | Relative path to logo file in documents dir |
| `pdf.accent_color` | string | `"#2563eb"` | Hex color for header bar and accents |
| `pdf.footer_text` | string | `""` | Custom footer text (max 200 chars) |
| `pdf.show_qr` | bool | `"true"` | Show QR payment code on PDF |
| `pdf.show_bank_details` | bool | `"true"` | Show bank account details |
| `pdf.paper_size` | string | `"A4"` | Paper size (A4 only for now) |
| `pdf.font_size` | string | `"normal"` | `"small"` (9pt), `"normal"` (10pt), `"large"` (11pt) |

### Logo Upload

New API endpoint for logo management:

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/settings/logo` | Upload logo (multipart/form-data) |
| `GET` | `/api/v1/settings/logo` | Download current logo |
| `DELETE` | `/api/v1/settings/logo` | Remove logo |

Logo constraints:
- Max file size: 2 MB
- Accepted formats: PNG, JPEG, SVG
- Stored in `{DataDir}/documents/logo.{ext}`
- Setting `pdf.logo_path` updated automatically on upload

### PDF Generator Changes

Modify `internal/pdf/invoice_pdf.go`:

1. **Logo placement** — Top-left of header, max height 40pt, proportional width. If no logo, supplier name takes full width.
2. **Accent color** — Applied to: header background bar, table header row, horizontal rules. Parsed from hex string to RGB.
3. **Footer text** — Centered at bottom of last page, below payment info. Rendered in smaller font (8pt), muted color.
4. **QR code toggle** — Skip QR generation when `pdf.show_qr` is `"false"`.
5. **Font size** — Base font size affects all text proportionally.

### PDF Settings Service

```go
type PDFSettings struct {
    LogoPath       string
    AccentColor    string // hex, e.g. "#2563eb"
    FooterText     string
    ShowQR         bool
    ShowBankDetails bool
    FontSize       string // "small", "normal", "large"
}

func (s *SettingsService) GetPDFSettings(ctx context.Context) (*PDFSettings, error) {
    // Reads from settings table, applies defaults
}
```

### Frontend: Settings Page

Add new section to `/settings` page: "PDF sablona" (PDF Template).

Layout:
1. **Logo upload** — File input with preview, delete button
2. **Accent color** — Color picker input (HTML5 `<input type="color">`)
3. **Footer text** — Text input with character counter (max 200)
4. **Toggles** — QR code, bank details
5. **Font size** — Radio buttons (Maly / Normalni / Velky)
6. **Preview button** — Opens PDF preview in new tab using a sample/last invoice

### Preview Endpoint

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/settings/pdf-preview` | Generate preview PDF with current settings using sample data |

The preview uses either the most recent invoice or generates sample data if no invoices exist.

## Implementation Order

1. Add settings keys + defaults to settings service
2. Logo upload/download/delete handler
3. Modify PDF generator to accept `PDFSettings` parameter
4. Apply logo, accent color, footer, font size, QR toggle in PDF generation
5. Add preview endpoint
6. Frontend settings section with logo upload, color picker, preview
7. Wire into router.go, client.ts
8. Tests: PDF generation with various settings combinations

## Files to Modify

| File | Change |
|------|--------|
| `internal/pdf/invoice_pdf.go` | Accept `PDFSettings`, apply logo/color/footer/font |
| `internal/service/settings_svc.go` | Add `GetPDFSettings()` method |
| `internal/handler/settings_handler.go` | Add logo upload/download/delete, preview endpoint |
| `internal/handler/router.go` | Mount new endpoints |
| `frontend/src/routes/settings/+page.svelte` | Add PDF template section |
| `frontend/src/lib/api/client.ts` | Add logo upload/preview API methods |

## Dependencies

No new Go dependencies. The `maroto/v2` library already supports images, colors, and custom fonts.

## Out of Scope

- HTML/CSS template engine (too complex, maroto layout is sufficient)
- Multiple template variants (one layout with customization covers 90% of needs)
- Per-invoice template selection (all invoices use the same template)
- Custom fonts (system default is fine for Czech diacritics)
