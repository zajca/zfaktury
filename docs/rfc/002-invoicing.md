# RFC-002: Invoicing Phase

**Status:** Draft
**Date:** 2026-03-10

## Summary

Phase 2 builds the full invoice lifecycle on top of the foundation from RFC-001. This includes PDF generation with QR payment codes, email sending via SMTP, ISDOC XML export, invoice sequence management UI, proforma invoices with settlement, credit notes, recurring invoices, multi-currency support with CNB exchange rates, automatic overdue detection, status history/timeline, and payment reminders.

The scope is large, so it is split into three sub-phases:
- **2A** — PDF generation, QR codes, email sending, ISDOC export, sequence management UI
- **2B** — Proforma invoices, credit notes, recurring invoices
- **2C** — Multi-currency with CNB rates, automatic overdue detection, status history, payment reminders

## Existing Code Inventory

The following invoice-related code was built and validated in RFC-001.

### Backend (Go)

| Layer | Files | What exists |
|-------|-------|-------------|
| Domain | `internal/domain/invoice.go` | Invoice, InvoiceItem, InvoiceSequence structs. Type constants (regular, proforma, credit_note). Status constants (draft, sent, paid, overdue, cancelled). CalculateTotals(), IsOverdue(), IsPaid() methods |
| Domain | `internal/domain/money.go` | Amount (int64 halere), NewAmount, FromFloat, ToCZK, Add, Sub, Multiply, String |
| Repository | `internal/repository/invoice_repo.go` | Create (with items in tx), Update (delete+reinsert items), Delete (soft), GetByID (with customer join + items), List (with filters, pagination), UpdateStatus, GetNextNumber |
| Repository | `internal/repository/interfaces.go` | InvoiceRepo interface |
| Service | `internal/service/invoice_svc.go` | Create (validate, defaults, sequence number, totals), Update, Delete, GetByID, List, MarkAsSent, MarkAsPaid, Duplicate |
| Handler | `internal/handler/invoice_handler.go` | Routes (POST /, GET /, GET /{id}, PUT /{id}, DELETE /{id}, POST /{id}/send, POST /{id}/mark-paid, POST /{id}/duplicate) |
| Handler | `internal/handler/helpers.go` | invoiceRequest, invoiceResponse, invoiceItemRequest, invoiceItemResponse DTOs, toDomain/fromDomain mappers, markPaidRequest |
| Config | `internal/config/config.go` | SMTPConfig (Host, Port, Username, Password, From) |

### Frontend (SvelteKit)

| Page | Route | What exists |
|------|-------|-------------|
| Invoice list | `invoices/+page.svelte` | API calls, status filter, search, pagination |
| Invoice create | `invoices/new/+page.svelte` | Line items, VAT calc, customer select, submit |
| Invoice detail | `invoices/[id]/+page.svelte` | View, edit draft, send, pay, duplicate, delete |
| API client | `src/lib/api/client.ts` | invoicesApi with list, getById, create, update, delete, send, markPaid, duplicate |

### Database Schema

| Table | Relevant columns |
|-------|------------------|
| `invoices` | type CHECK (issued, received, proforma, credit_note), status CHECK (draft, sent, paid, partially_paid, overdue, cancelled) |
| `invoice_items` | quantity_cents, unit_price_amount, vat_rate_percent, vat_amount, total_amount |
| `invoice_sequences` | prefix, next_number, year, format_pattern, UNIQUE(prefix, year) |

### Dependencies Already Installed (Go)

| Package | Purpose | Status |
|---------|---------|--------|
| `github.com/johnfercher/maroto/v2` | PDF generation | Installed, not used |
| `github.com/dundee/qrpay` | QR Platba SPD format | Installed, not used |

---

## Sub-Phase 2A: PDF, Email, ISDOC, Sequence UI

### Task 1: PDF Generation Service

#### Background

Invoice PDFs must follow Czech invoicing conventions: supplier/customer blocks, line items table, VAT summary, payment info with QR code. The `maroto/v2` library is already a dependency.

#### Implementation Tasks

1. **Create `internal/pdf/invoice_pdf.go`**
   - `type InvoicePDFGenerator struct` with constructor accepting supplier settings
   - `Generate(ctx context.Context, invoice *domain.Invoice, supplier SupplierInfo) ([]byte, error)` — returns PDF bytes
   - `SupplierInfo` struct: Name, ICO, DIC, VATRegistered, Street, City, ZIP, Email, Phone, BankAccount, BankCode, IBAN, SWIFT, LogoPath
   - PDF layout:
     - Header: logo (optional), invoice number, type label, issue/due/delivery dates
     - Two-column block: Supplier info (left) | Customer info (right)
     - Line items table: #, Description, Qty, Unit, Unit Price, VAT %, VAT Amount, Total
     - VAT summary table: grouped by VAT rate (base, VAT amount, total per rate)
     - Totals: subtotal, total VAT, total amount (bold)
     - Payment info: bank account, IBAN, variable symbol, constant symbol, due date
     - QR code: bottom-right of payment section (see Task 2)
     - Footer: legal text ("Subjekt neni platce DPH" if not VAT registered, or reverse charge note)

2. **Create `internal/pdf/supplier.go`**
   - `LoadSupplierFromSettings(ctx context.Context, settingsSvc *service.SettingsService) (SupplierInfo, error)`
   - Maps setting keys (`user_name`, `user_ico`, `user_dic`, etc.) to SupplierInfo struct

3. **Create `internal/handler/invoice_pdf_handler.go`**
   - `GET /api/v1/invoices/{id}/pdf` — returns `application/pdf` with `Content-Disposition: inline; filename="FV20260001.pdf"`
   - Loads invoice via InvoiceService, supplier via SettingsService
   - Calls PDFGenerator.Generate(), writes bytes to response
   - Returns 404 if invoice not found, 500 on generation error

4. **Register route in `internal/handler/invoice_handler.go`**
   - Add `r.Get("/{id}/pdf", h.GeneratePDF)` to Routes()
   - InvoiceHandler needs access to PDFGenerator and SettingsService (add to constructor or create separate handler)

5. **Frontend PDF button**
   - Add "Download PDF" / "View PDF" button to invoice detail page (`invoices/[id]/+page.svelte`)
   - Opens `/api/v1/invoices/{id}/pdf` in new tab or triggers download
   - Add `getPdfUrl(id: number): string` helper to `client.ts`

6. **Tests**
   - `internal/pdf/invoice_pdf_test.go` — test PDF generation with sample invoice data, verify non-empty bytes returned
   - `internal/pdf/supplier_test.go` — test SupplierInfo loading from mock settings
   - `internal/handler/invoice_pdf_handler_test.go` — HTTP test: valid invoice returns 200 + PDF content type, missing invoice returns 404

#### Acceptance Criteria

- [ ] `GET /api/v1/invoices/{id}/pdf` returns a valid PDF
- [ ] PDF contains all required Czech invoice fields (supplier, customer, items, totals, payment info)
- [ ] PDF includes QR payment code (see Task 2)
- [ ] Non-VAT-registered supplier shows "Subjekt neni platce DPH" text
- [ ] VAT-registered supplier shows VAT summary breakdown by rate
- [ ] Frontend "Download PDF" button works from invoice detail page
- [ ] Tests pass for PDF generation and HTTP endpoint

---

### Task 2: QR Payment Code

#### Background

Czech QR Platba uses the SPD (Short Payment Descriptor) format. The `qrpay` library generates these. The QR code must be embedded in the PDF and optionally displayed in the frontend.

#### Implementation Tasks

1. **Create `internal/pdf/qr_payment.go`**
   - `GenerateQRPayment(invoice *domain.Invoice, iban string, swift string) ([]byte, error)` — returns QR code PNG bytes
   - Uses `qrpay` library to create SPD string:
     - `ACC` — IBAN (or bank account + bank code converted to IBAN)
     - `AM` — total amount in CZK (from `TotalAmount.ToCZK()`)
     - `CC` — currency code (CZK)
     - `X-VS` — variable symbol
     - `X-KS` — constant symbol (if set)
     - `DT` — due date (YYYYMMDD format)
     - `MSG` — invoice number
   - Generates QR code image from SPD string

2. **Integrate QR in PDF generator**
   - In `invoice_pdf.go`, call `GenerateQRPayment()` and embed the PNG in the PDF payment section
   - QR code size: approximately 3x3 cm

3. **QR code HTTP endpoint**
   - `GET /api/v1/invoices/{id}/qr` — returns QR code as `image/png`
   - Useful for frontend display and standalone QR code download

4. **Frontend QR display**
   - Show QR code image on invoice detail page in the payment section
   - Load from `/api/v1/invoices/{id}/qr`

5. **Tests**
   - `internal/pdf/qr_payment_test.go` — test SPD string generation, QR image output (non-empty PNG bytes)

#### Acceptance Criteria

- [ ] QR code contains valid SPD format parseable by Czech banking apps
- [ ] QR code is embedded in invoice PDF
- [ ] `GET /api/v1/invoices/{id}/qr` returns a valid PNG image
- [ ] Frontend shows QR code on invoice detail page
- [ ] Tests cover SPD string format and image generation

---

### Task 3: Email Sending

#### Background

Users need to send invoices by email with the PDF attached. SMTP config already exists in `internal/config/config.go` (SMTPConfig struct with Host, Port, Username, Password, From).

#### Implementation Tasks

1. **Create `internal/email/sender.go`**
   - `type EmailSender struct` with SMTP config
   - `NewEmailSender(cfg config.SMTPConfig) *EmailSender`
   - `Send(ctx context.Context, msg EmailMessage) error`
   - `EmailMessage` struct: To []string, Subject, Body (HTML), Attachments []Attachment
   - `Attachment` struct: Filename string, ContentType string, Data []byte
   - SMTP with TLS/STARTTLS support
   - Connection timeout 30s, send timeout 60s

2. **Create `internal/email/templates.go`**
   - `InvoiceEmailTemplate(invoice *domain.Invoice, supplier SupplierInfo) EmailMessage`
   - Default subject: "Faktura {invoice_number} od {supplier_name}"
   - HTML body: greeting, invoice summary (number, date, total, due date), payment instructions, supplier contact info
   - Czech text throughout

3. **Create `internal/handler/invoice_email_handler.go`**
   - `POST /api/v1/invoices/{id}/email` — sends invoice via email
   - Request body:
     ```json
     {
       "to": ["customer@example.com"],
       "subject": "optional custom subject",
       "message": "optional custom body text"
     }
     ```
   - If `to` is empty, use the customer's email from the contact record
   - If `subject` is empty, use the default template subject
   - Generates PDF, attaches to email, sends
   - On success, marks invoice as sent (calls MarkAsSent if status is draft)
   - Returns 200 with `{"status": "sent", "to": ["customer@example.com"]}`
   - Returns 422 if no email address available (customer has no email and none provided)
   - Returns 503 if SMTP is not configured

4. **Register route**
   - Add `r.Post("/{id}/email", h.SendEmail)` to invoice Routes()
   - Wire EmailSender in `serve.go` (only create if SMTP config is non-empty)

5. **Frontend email dialog**
   - Add "Send by Email" button to invoice detail page
   - Modal/dialog with: To (pre-filled from customer email), Subject, Message fields
   - Submit calls `invoicesApi.sendEmail(id, { to, subject, message })`
   - Show success/error feedback

6. **Add to `client.ts`**
   ```typescript
   sendEmail(id: number, data: { to?: string[]; subject?: string; message?: string }) {
     return post<{ status: string; to: string[] }>(`/invoices/${id}/email`, data);
   }
   ```

7. **Tests**
   - `internal/email/sender_test.go` — test with mock SMTP server (net/smtp test server)
   - `internal/email/templates_test.go` — test template rendering with sample data
   - `internal/handler/invoice_email_handler_test.go` — HTTP tests: success, missing email, SMTP not configured

#### Acceptance Criteria

- [ ] `POST /api/v1/invoices/{id}/email` sends email with PDF attachment
- [ ] Email uses Czech text template with invoice details
- [ ] Sending email marks invoice as "sent" if currently "draft"
- [ ] Customer email is used as default recipient
- [ ] Returns proper error when SMTP is not configured
- [ ] Frontend shows send dialog with editable fields
- [ ] Tests cover sending, templates, and error cases

---

### Task 4: ISDOC XML Export

#### Background

ISDOC (Information System Document) is the Czech national standard for electronic invoicing (based on UBL). Version 6.0.2 is current. Invoices exported as ISDOC XML can be imported into other Czech accounting systems.

#### Implementation Tasks

1. **Create `internal/isdoc/types.go`**
   - Go structs matching ISDOC 6.0.2 XML schema
   - Key elements: `Invoice` root, `AccountingSupplierParty`, `AccountingCustomerParty`, `InvoiceLines`, `TaxTotal`, `LegalMonetaryTotal`, `PaymentMeans`
   - XML namespace: `urn:isdoc:invoice:6.0.2`

2. **Create `internal/isdoc/generator.go`**
   - `type ISDOCGenerator struct`
   - `Generate(ctx context.Context, invoice *domain.Invoice, supplier SupplierInfo) ([]byte, error)` — returns ISDOC XML bytes
   - Maps domain.Invoice fields to ISDOC XML structure
   - Proper Czech tax categorization (VAT rates 0%, 12%, 21%)
   - Document type mapping: regular -> invoice (1), proforma -> proforma (4), credit_note -> credit note (2)
   - ISO 8601 date formatting
   - UTF-8 encoding with XML declaration

3. **Create `internal/handler/invoice_isdoc_handler.go`**
   - `GET /api/v1/invoices/{id}/isdoc` — returns `application/xml` with `Content-Disposition: attachment; filename="FV20260001.isdoc"`
   - `POST /api/v1/invoices/export/isdoc` — batch export
     - Request body: `{"invoice_ids": [1, 2, 3]}`
     - Returns ZIP file containing multiple ISDOC files
     - Content-Type: `application/zip`

4. **Register routes**
   - Add `r.Get("/{id}/isdoc", h.ExportISDOC)` to invoice Routes()
   - Add `r.Post("/export/isdoc", h.BatchExportISDOC)` to invoice Routes()

5. **Frontend export buttons**
   - Add "Export ISDOC" button to invoice detail page
   - Add bulk export option to invoice list page (select multiple, export)
   - Add to `client.ts`:
     ```typescript
     getIsdocUrl(id: number): string { return `${API_BASE}/invoices/${id}/isdoc`; }
     ```

6. **Tests**
   - `internal/isdoc/generator_test.go` — test XML output structure, namespace, required fields
   - Validate generated XML against basic structure expectations (root element, namespaces, required children)

#### Acceptance Criteria

- [ ] `GET /api/v1/invoices/{id}/isdoc` returns valid ISDOC 6.0.2 XML
- [ ] XML contains all required invoice fields (parties, items, tax, totals, payment)
- [ ] Batch export returns ZIP with individual ISDOC files
- [ ] Frontend has single and bulk export buttons
- [ ] Generated ISDOC is importable by common Czech accounting software (manual verification)
- [ ] Tests validate XML structure

---

### Task 5: Invoice Sequence Management UI

#### Background

Invoice sequences (`invoice_sequences` table) define numbering patterns like "FV20260001". The backend already supports sequences (GetNextNumber in repo, sequence assignment in service Create), but there is no UI to manage sequences, and no CRUD endpoints for sequences themselves.

#### Implementation Tasks

1. **Create `internal/handler/sequence_handler.go`**
   - `GET /api/v1/invoice-sequences` — list all sequences
   - `POST /api/v1/invoice-sequences` — create sequence
     ```json
     {
       "prefix": "FV",
       "year": 2026,
       "next_number": 1,
       "format_pattern": "{prefix}{year}{number:04d}"
     }
     ```
   - `PUT /api/v1/invoice-sequences/{id}` — update sequence (prefix, format_pattern, next_number)
   - `DELETE /api/v1/invoice-sequences/{id}` — delete sequence (only if no invoices reference it)
   - Response DTO:
     ```json
     {
       "id": 1,
       "prefix": "FV",
       "year": 2026,
       "next_number": 5,
       "format_pattern": "{prefix}{year}{number:04d}",
       "preview": "FV20260005"
     }
     ```

2. **Add sequence repository methods**
   - Add `InvoiceSequenceRepo` interface to `internal/repository/interfaces.go`:
     ```go
     type InvoiceSequenceRepo interface {
       Create(ctx context.Context, seq *domain.InvoiceSequence) error
       Update(ctx context.Context, seq *domain.InvoiceSequence) error
       Delete(ctx context.Context, id int64) error
       GetByID(ctx context.Context, id int64) (*domain.InvoiceSequence, error)
       List(ctx context.Context) ([]domain.InvoiceSequence, error)
       GetByPrefixAndYear(ctx context.Context, prefix string, year int) (*domain.InvoiceSequence, error)
     }
     ```
   - Implement in `internal/repository/sequence_repo.go`

3. **Create `internal/service/sequence_svc.go`**
   - `Create` — validate uniqueness (prefix+year), defaults
   - `Update` — prevent lowering next_number below already-used numbers
   - `Delete` — check no invoices reference this sequence
   - `List`, `GetByID`
   - `GetOrCreateForYear(ctx, prefix, year)` — auto-create sequence for new year

4. **Wire in `serve.go` and `router.go`**
   - Add SequenceHandler mount: `api.Mount("/invoice-sequences", sequenceHandler.Routes())`

5. **Frontend settings section**
   - Add "Invoice Sequences" section to settings page or create dedicated `/settings/sequences` page
   - List existing sequences with year, prefix, next number, format preview
   - Create new sequence form
   - Edit next number (with warning about gaps)
   - Add `sequencesApi` to `client.ts`:
     ```typescript
     export const sequencesApi = {
       list() { return get<InvoiceSequence[]>('/invoice-sequences'); },
       create(data: Partial<InvoiceSequence>) { return post<InvoiceSequence>('/invoice-sequences', data); },
       update(id: number, data: Partial<InvoiceSequence>) { return put<InvoiceSequence>(`/invoice-sequences/${id}`, data); },
       delete(id: number) { return del<void>(`/invoice-sequences/${id}`); },
     };
     ```

6. **Auto-create sequence on invoice create**
   - Modify `InvoiceService.Create()` to auto-create a sequence for the current year if none exists and no sequence_id is provided
   - Use default prefix "FV" for regular invoices, "ZF" for proformas, "DN" for credit notes

7. **Tests**
   - Repository, service, handler tests for sequence CRUD
   - Test auto-creation logic
   - Test prevention of next_number decrease

#### Acceptance Criteria

- [ ] CRUD API for invoice sequences works
- [ ] Frontend shows sequence management UI
- [ ] Auto-creates sequence for current year if none exists
- [ ] Cannot delete sequence referenced by invoices
- [ ] Cannot lower next_number below already-used values
- [ ] Format preview shows correct next invoice number
- [ ] Tests cover CRUD, validation, and auto-creation

---

## Sub-Phase 2B: Proforma, Credit Notes, Recurring

### Task 6: Proforma Invoices and Settlement

#### Background

Proforma invoices (zalohove faktury) are requests for advance payment. When paid, a regular "settlement" invoice (vyuctovaci faktura) is issued referencing the proforma. The domain already supports `type: "proforma"` but no settlement logic exists.

#### Implementation Tasks

1. **Schema migration `003_invoice_relations.sql`**
   - Add columns to `invoices`:
     ```sql
     ALTER TABLE invoices ADD COLUMN related_invoice_id INTEGER REFERENCES invoices(id);
     ALTER TABLE invoices ADD COLUMN relation_type TEXT CHECK (relation_type IN ('settlement', 'credit_note', 'recurring_source'));
     ```
   - Index: `CREATE INDEX idx_invoices_related_invoice_id ON invoices(related_invoice_id);`

2. **Update domain struct**
   - Add to `domain.Invoice`:
     ```go
     RelatedInvoiceID *int64 `json:"related_invoice_id,omitempty"`
     RelationType     string `json:"relation_type,omitempty"`
     ```

3. **Update repository**
   - Add `related_invoice_id` and `relation_type` to all INSERT/UPDATE/SELECT queries in `invoice_repo.go`
   - Add `GetRelatedInvoices(ctx, invoiceID) ([]Invoice, error)` — find all invoices related to a given invoice

4. **Create settlement service logic**
   - Add to `InvoiceService`:
     ```go
     func (s *InvoiceService) SettleProforma(ctx context.Context, proformaID int64) (*domain.Invoice, error)
     ```
   - Validates proforma is type "proforma" and status "paid"
   - Creates a new regular invoice with:
     - Same items, customer, payment info as proforma
     - `RelatedInvoiceID` pointing to proforma
     - `RelationType` = "settlement"
     - Status = "draft"
     - Fresh invoice number from regular sequence
   - Returns the new settlement invoice

5. **HTTP endpoint**
   - `POST /api/v1/invoices/{id}/settle` — settle a paid proforma
   - Returns 201 with the new settlement invoice
   - Returns 422 if proforma is not paid or not a proforma type

6. **Frontend**
   - On proforma detail page (when status is "paid"), show "Create Settlement Invoice" button
   - After settlement, navigate to the new invoice
   - Show "Related invoices" section on both proforma and settlement invoice detail pages
   - Add to `client.ts`:
     ```typescript
     settle(id: number) { return post<Invoice>(`/invoices/${id}/settle`, {}); }
     ```

7. **Update DTOs**
   - Add `related_invoice_id`, `relation_type`, `related_invoices` to invoiceResponse
   - Add `RelatedInvoice` TypeScript interface

8. **Tests**
   - Service test: settle proforma creates correct regular invoice
   - Service test: cannot settle non-proforma or unpaid proforma
   - Handler test: HTTP flow
   - Repository test: related invoice queries

#### Acceptance Criteria

- [ ] Proforma invoices can be created with type "proforma" and sequence prefix "ZF"
- [ ] Paid proforma can be settled, creating a regular invoice with correct relation
- [ ] Settlement invoice copies items and amounts from proforma
- [ ] Cannot settle unpaid proforma or non-proforma invoice
- [ ] Both invoices show their relation in the detail view
- [ ] Tests cover settlement flow and validation

---

### Task 7: Credit Notes (Dobropisy)

#### Background

Credit notes reverse all or part of a previously issued invoice. They have negative amounts and reference the original invoice.

#### Implementation Tasks

1. **Create credit note service logic**
   - Add to `InvoiceService`:
     ```go
     func (s *InvoiceService) CreateCreditNote(ctx context.Context, originalID int64, items []domain.InvoiceItem, reason string) (*domain.Invoice, error)
     ```
   - Validates original invoice exists and is sent or paid
   - Creates a new invoice with:
     - `Type` = "credit_note"
     - `RelatedInvoiceID` pointing to original
     - `RelationType` = "credit_note"
     - All amounts negated (negative values)
     - `Notes` = reason
     - Fresh number from "DN" sequence
   - If `items` is empty, creates full credit note (copies all items with negated amounts)
   - If `items` is provided, creates partial credit note with specified items

2. **HTTP endpoint**
   - `POST /api/v1/invoices/{id}/credit-note` — create credit note for invoice
   - Request body:
     ```json
     {
       "reason": "Goods returned",
       "items": [
         {
           "description": "Service X",
           "quantity": 100,
           "unit": "ks",
           "unit_price": 10000,
           "vat_rate_percent": 21
         }
       ]
     }
     ```
   - If `items` is omitted or empty, full credit note is created
   - Returns 201 with the new credit note

3. **Frontend**
   - Add "Create Credit Note" button to invoice detail page (visible for sent/paid invoices)
   - Dialog: select full or partial credit note
   - For partial: item selection with quantity adjustment
   - Reason text field (required)
   - Credit note detail page shows negative amounts in red
   - Related invoice link

4. **Update CalculateTotals()**
   - Ensure negative amounts work correctly through the calculation pipeline

5. **Tests**
   - Full credit note: all items negated
   - Partial credit note: only selected items
   - Cannot create credit note for draft or cancelled invoice
   - Negative amount arithmetic in CalculateTotals

#### Acceptance Criteria

- [ ] Full credit note copies all items with negated amounts
- [ ] Partial credit note allows selecting specific items and quantities
- [ ] Credit note has type "credit_note" and references original invoice
- [ ] Credit note uses "DN" sequence prefix
- [ ] Negative amounts display correctly in frontend (red text)
- [ ] Cannot create credit note for draft or cancelled invoices
- [ ] Tests cover full, partial, and validation scenarios

---

### Task 8: Recurring Invoices

#### Background

Recurring invoices auto-generate draft invoices at configured intervals (monthly, quarterly, yearly). A recurring invoice is a template, not a real invoice.

#### Implementation Tasks

1. **Schema migration `004_recurring_invoices.sql`**
   ```sql
   CREATE TABLE recurring_invoices (
       id INTEGER PRIMARY KEY AUTOINCREMENT,
       name TEXT NOT NULL,
       customer_id INTEGER NOT NULL REFERENCES contacts(id),
       sequence_id INTEGER REFERENCES invoice_sequences(id),
       currency_code TEXT NOT NULL DEFAULT 'CZK',
       payment_method TEXT NOT NULL DEFAULT 'bank_transfer',
       bank_account TEXT,
       bank_code TEXT,
       iban TEXT,
       swift TEXT,
       constant_symbol TEXT,
       notes TEXT,
       internal_notes TEXT,
       interval TEXT NOT NULL CHECK (interval IN ('weekly', 'monthly', 'quarterly', 'yearly')),
       day_of_month INTEGER DEFAULT 1,
       payment_terms_days INTEGER NOT NULL DEFAULT 14,
       next_issue_date TEXT NOT NULL,
       last_issued_at TEXT,
       is_active INTEGER NOT NULL DEFAULT 1,
       created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
       updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
       deleted_at TEXT
   );

   CREATE TABLE recurring_invoice_items (
       id INTEGER PRIMARY KEY AUTOINCREMENT,
       recurring_invoice_id INTEGER NOT NULL REFERENCES recurring_invoices(id) ON DELETE CASCADE,
       description TEXT NOT NULL,
       quantity INTEGER NOT NULL DEFAULT 100,
       unit TEXT NOT NULL DEFAULT 'ks',
       unit_price INTEGER NOT NULL DEFAULT 0,
       vat_rate_percent INTEGER NOT NULL DEFAULT 0,
       sort_order INTEGER NOT NULL DEFAULT 0
   );

   CREATE INDEX idx_recurring_invoices_next_issue ON recurring_invoices(next_issue_date);
   CREATE INDEX idx_recurring_invoices_active ON recurring_invoices(is_active);
   CREATE INDEX idx_recurring_invoice_items_rid ON recurring_invoice_items(recurring_invoice_id);
   ```

2. **Domain structs — `internal/domain/recurring.go`**
   ```go
   type RecurringInvoice struct {
       ID               int64
       Name             string
       CustomerID       int64
       Customer         *Contact
       SequenceID       int64
       CurrencyCode     string
       PaymentMethod    string
       BankAccount      string
       BankCode         string
       IBAN             string
       SWIFT            string
       ConstantSymbol   string
       Notes            string
       InternalNotes    string
       Interval         string // weekly, monthly, quarterly, yearly
       DayOfMonth       int
       PaymentTermsDays int
       NextIssueDate    time.Time
       LastIssuedAt     *time.Time
       IsActive         bool
       Items            []RecurringInvoiceItem
       CreatedAt        time.Time
       UpdatedAt        time.Time
       DeletedAt        *time.Time
   }

   type RecurringInvoiceItem struct {
       ID                  int64
       RecurringInvoiceID  int64
       Description         string
       Quantity            Amount
       Unit                string
       UnitPrice           Amount
       VATRatePercent      int
       SortOrder           int
   }
   ```

3. **Repository — `internal/repository/recurring_repo.go`**
   - CRUD for recurring invoices with items (same tx pattern as invoice_repo)
   - `ListDue(ctx, date) ([]RecurringInvoice, error)` — find active recurring invoices where `next_issue_date <= date`
   - Interface in `interfaces.go`:
     ```go
     type RecurringInvoiceRepo interface {
         Create(ctx context.Context, ri *domain.RecurringInvoice) error
         Update(ctx context.Context, ri *domain.RecurringInvoice) error
         Delete(ctx context.Context, id int64) error
         GetByID(ctx context.Context, id int64) (*domain.RecurringInvoice, error)
         List(ctx context.Context) ([]domain.RecurringInvoice, error)
         ListDue(ctx context.Context, date time.Time) ([]domain.RecurringInvoice, error)
     }
     ```

4. **Service — `internal/service/recurring_svc.go`**
   - CRUD with validation
   - `ProcessDue(ctx context.Context) ([]domain.Invoice, error)`:
     - Finds all due recurring invoices
     - For each: creates a draft invoice from the template
     - Updates `next_issue_date` based on interval
     - Updates `last_issued_at`
     - Returns list of created invoices
   - `CalculateNextIssueDate(current time.Time, interval string, dayOfMonth int) time.Time`

5. **Background processing**
   - Add daily check in `serve.go` startup: goroutine with ticker that calls `ProcessDue()` once per hour
   - Log created invoices via slog

6. **HTTP endpoints**
   - `GET /api/v1/recurring-invoices` — list all
   - `POST /api/v1/recurring-invoices` — create
   - `GET /api/v1/recurring-invoices/{id}` — get by ID
   - `PUT /api/v1/recurring-invoices/{id}` — update
   - `DELETE /api/v1/recurring-invoices/{id}` — delete (soft)
   - `POST /api/v1/recurring-invoices/{id}/generate` — manually trigger generation
   - `POST /api/v1/recurring-invoices/process-due` — manually trigger processing of all due

7. **Frontend**
   - New page: `frontend/src/routes/recurring/+page.svelte` — list recurring invoices
   - New page: `frontend/src/routes/recurring/new/+page.svelte` — create form
   - New page: `frontend/src/routes/recurring/[id]/+page.svelte` — detail/edit
   - Add to sidebar navigation
   - Add `recurringApi` to `client.ts`

8. **Tests**
   - Service: ProcessDue creates correct invoices, advances next_issue_date
   - Service: interval calculation (monthly on 31st, quarterly, yearly)
   - Repository: ListDue returns only active + due records

#### Acceptance Criteria

- [ ] CRUD for recurring invoice templates works
- [ ] Due recurring invoices automatically generate draft invoices
- [ ] Next issue date advances correctly for all intervals
- [ ] Edge cases handled: monthly on 31st (uses last day of month), leap years
- [ ] Manual trigger endpoint works
- [ ] Frontend shows recurring invoices list with status (active/paused)
- [ ] Tests cover processing, date calculation, and edge cases

---

## Sub-Phase 2C: Multi-Currency, Overdue Detection, History, Reminders

### Task 9: Multi-Currency with CNB Exchange Rates

#### Background

Invoices can be issued in EUR, USD, or other currencies. For tax purposes, the CZK equivalent must be calculated using the CNB (Czech National Bank) daily exchange rate. The exchange rate on the delivery date (datum uskutecneni zdanitelneho plneni) is used.

CNB API: `https://www.cnb.cz/cs/financni-trhy/devizovy-trh/kurzy-devizoveho-trhu/kurzy-devizoveho-trhu/denni_kurz.txt?date=DD.MM.YYYY`

#### Implementation Tasks

1. **Create `internal/cnb/client.go`**
   - `type CNBClient struct` with HTTP client
   - `GetExchangeRate(ctx context.Context, currency string, date time.Time) (float64, error)` — returns rate (e.g., 25.34 CZK per 1 EUR)
   - Parse CNB text format (pipe-delimited, skip header lines)
   - Cache rates in memory (map[date+currency]rate) to avoid repeated API calls
   - Handle weekends/holidays: if rate not available for exact date, use the most recent previous business day

2. **Create `internal/cnb/types.go`**
   - CNB response parsing structs/helpers

3. **Schema migration `005_currency_support.sql`**
   - Replace the two exchange_rate columns with a single one if not already aligned:
     ```sql
     -- If schema mismatch exists, normalize to single exchange_rate INTEGER column
     -- Stores rate in halere precision (e.g., 2534 = 25.34 CZK per 1 foreign unit)
     ```
   - Add `exchange_rate_date TEXT` column to invoices (date the rate was fetched for)

4. **Update InvoiceService.Create()**
   - If currency is not CZK and exchange rate is not manually set:
     - Fetch rate from CNB for the delivery date
     - Store in `exchange_rate` field
   - Calculate CZK equivalent amounts for tax reporting

5. **Update invoice detail frontend**
   - Show currency selector (CZK, EUR, USD, GBP + custom)
   - When non-CZK currency selected:
     - Fetch exchange rate from backend: `GET /api/v1/exchange-rate?currency=EUR&date=2026-03-10`
     - Show fetched rate with option to override manually
     - Display both foreign currency and CZK equivalent amounts

6. **HTTP endpoint for rate lookup**
   - `GET /api/v1/exchange-rate?currency=EUR&date=2026-03-10` — returns `{ "rate": 2534, "date": "2026-03-10", "source": "cnb" }`

7. **Tests**
   - `internal/cnb/client_test.go` — test parsing CNB format, rate lookup, caching, weekend fallback
   - Mock HTTP server with sample CNB response data

#### Acceptance Criteria

- [ ] CNB client fetches and parses daily exchange rates
- [ ] Invoice in EUR/USD automatically gets CNB rate for delivery date
- [ ] Exchange rate can be manually overridden
- [ ] CZK equivalent amounts are calculated and stored
- [ ] Weekend/holiday dates fall back to previous business day
- [ ] Frontend shows currency selector and rate lookup
- [ ] Tests cover parsing, caching, and fallback logic

---

### Task 10: Automatic Overdue Detection

#### Background

Invoices past their due date should be automatically marked as "overdue". The domain already has `IsOverdue()` method but nothing triggers the status change.

#### Implementation Tasks

1. **Add overdue check to InvoiceService**
   - `func (s *InvoiceService) CheckOverdue(ctx context.Context) (int, error)`:
     - Query all invoices with status "sent" and `due_date < now`
     - Update their status to "overdue"
     - Return count of updated invoices

2. **Add repository method**
   - `MarkOverdue(ctx context.Context, beforeDate time.Time) (int, error)` — batch update in single query:
     ```sql
     UPDATE invoices SET status = 'overdue', updated_at = ?
     WHERE status = 'sent' AND due_date < ? AND deleted_at IS NULL
     ```
   - Add to InvoiceRepo interface

3. **Background job**
   - Run `CheckOverdue()` in the hourly ticker alongside recurring invoice processing (from Task 8)
   - Log count of newly overdue invoices

4. **Invoice list visual indicator**
   - Frontend already has status badges; ensure "overdue" status displays with red/warning styling
   - Sort overdue invoices prominently in the list (already sorted by issue_date DESC, which is fine)

5. **Tests**
   - Repository: MarkOverdue updates correct invoices, ignores paid/draft/cancelled
   - Service: CheckOverdue integration test

#### Acceptance Criteria

- [ ] Sent invoices past due date are automatically marked as "overdue"
- [ ] Overdue check runs periodically (hourly)
- [ ] Only "sent" status invoices are affected (not draft, paid, cancelled)
- [ ] Frontend displays overdue status with appropriate visual styling
- [ ] Tests verify correct invoices are updated

---

### Task 11: Invoice Status History / Timeline

#### Background

Track every status change on an invoice with timestamp and optional note. Useful for audit trail and displaying a timeline on the invoice detail page.

#### Implementation Tasks

1. **Schema migration `006_invoice_status_history.sql`**
   ```sql
   CREATE TABLE invoice_status_history (
       id INTEGER PRIMARY KEY AUTOINCREMENT,
       invoice_id INTEGER NOT NULL REFERENCES invoices(id),
       from_status TEXT NOT NULL,
       to_status TEXT NOT NULL,
       note TEXT,
       created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
   );

   CREATE INDEX idx_invoice_status_history_invoice ON invoice_status_history(invoice_id);
   ```

2. **Domain struct — add to `internal/domain/invoice.go`**
   ```go
   type InvoiceStatusChange struct {
       ID         int64
       InvoiceID  int64
       FromStatus string
       ToStatus   string
       Note       string
       CreatedAt  time.Time
   }
   ```

3. **Repository**
   - Add to `InvoiceRepo` interface:
     ```go
     RecordStatusChange(ctx context.Context, change *domain.InvoiceStatusChange) error
     GetStatusHistory(ctx context.Context, invoiceID int64) ([]domain.InvoiceStatusChange, error)
     ```
   - Implement in `invoice_repo.go`

4. **Service integration**
   - Modify `MarkAsSent()`, `MarkAsPaid()`, `CheckOverdue()`, and any other status-changing methods to record status changes
   - Include contextual notes: "Marked as sent", "Payment received: 10,000.00 CZK", "Automatically detected as overdue"

5. **HTTP endpoint**
   - `GET /api/v1/invoices/{id}/history` — returns status change timeline
   - Response:
     ```json
     [
       {
         "id": 1,
         "from_status": "draft",
         "to_status": "sent",
         "note": "Marked as sent",
         "created_at": "2026-03-10T14:30:00Z"
       }
     ]
     ```

6. **Frontend timeline component**
   - Add timeline/history section to invoice detail page
   - Vertical timeline with status change entries, timestamps, and notes
   - Color-coded by status type

7. **Add to `client.ts`**
   ```typescript
   getHistory(id: number) {
     return get<InvoiceStatusChange[]>(`/invoices/${id}/history`);
   }
   ```

8. **Tests**
   - Repository: insert and query status changes
   - Service: verify status changes are recorded on MarkAsSent, MarkAsPaid
   - Handler: HTTP endpoint returns correct timeline

#### Acceptance Criteria

- [ ] Every status change is recorded with from/to status, note, and timestamp
- [ ] `GET /api/v1/invoices/{id}/history` returns ordered timeline
- [ ] Frontend shows visual timeline on invoice detail page
- [ ] Status changes from MarkAsSent, MarkAsPaid, CheckOverdue are all recorded
- [ ] Tests cover recording and retrieval

---

### Task 12: Payment Reminders

#### Background

When an invoice is overdue, the system should support sending payment reminder emails. Reminders can be triggered manually or automatically at configurable intervals.

#### Implementation Tasks

1. **Schema migration `007_payment_reminders.sql`**
   ```sql
   CREATE TABLE payment_reminders (
       id INTEGER PRIMARY KEY AUTOINCREMENT,
       invoice_id INTEGER NOT NULL REFERENCES invoices(id),
       reminder_number INTEGER NOT NULL DEFAULT 1,
       sent_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
       email_to TEXT NOT NULL,
       subject TEXT,
       body TEXT
   );

   CREATE INDEX idx_payment_reminders_invoice ON payment_reminders(invoice_id);
   ```

2. **Domain struct — `internal/domain/reminder.go`**
   ```go
   type PaymentReminder struct {
       ID             int64
       InvoiceID      int64
       ReminderNumber int
       SentAt         time.Time
       EmailTo        string
       Subject        string
       Body           string
   }
   ```

3. **Repository — `internal/repository/reminder_repo.go`**
   - `Create(ctx, reminder) error`
   - `ListByInvoiceID(ctx, invoiceID) ([]PaymentReminder, error)`
   - `CountByInvoiceID(ctx, invoiceID) (int, error)`
   - Interface in `interfaces.go`

4. **Email template — `internal/email/reminder_template.go`**
   - 1st reminder: polite ("We would like to remind you...")
   - 2nd reminder: firmer ("This is a second reminder...")
   - 3rd+ reminder: urgent ("This is an urgent reminder...")
   - All in Czech
   - Include: invoice number, original amount, due date, days overdue, payment instructions

5. **Service — `internal/service/reminder_svc.go`**
   - `SendReminder(ctx, invoiceID) error`:
     - Validates invoice is overdue
     - Determines reminder number (count existing + 1)
     - Generates appropriate email template
     - Sends via EmailSender
     - Records in payment_reminders table
   - `GetRemindersForInvoice(ctx, invoiceID) ([]PaymentReminder, error)`

6. **HTTP endpoints**
   - `POST /api/v1/invoices/{id}/remind` — send payment reminder
   - `GET /api/v1/invoices/{id}/reminders` — list reminders sent for this invoice

7. **Frontend**
   - Add "Send Reminder" button to invoice detail page (visible when status is "overdue")
   - Show reminder history in the timeline section (integrated with Task 11)
   - Badge on invoice list showing reminder count

8. **Tests**
   - Service: sends correct template based on reminder number
   - Service: cannot send reminder for non-overdue invoice
   - Repository: CRUD tests

#### Acceptance Criteria

- [ ] Payment reminders can be sent for overdue invoices
- [ ] Reminder email escalates tone based on reminder number (1st, 2nd, 3rd+)
- [ ] Reminder history is recorded and viewable
- [ ] Cannot send reminder for non-overdue invoices
- [ ] Frontend shows reminder button and history
- [ ] Tests cover email generation, validation, and persistence

---

## Implementation Order

```
Sub-Phase 2A (can be partially parallelized):
  1. Task 2: QR Payment Code (no dependencies)
  2. Task 1: PDF Generation (depends on Task 2 for QR embedding)
  3. Task 5: Invoice Sequence UI (no dependencies)
  4. Task 3: Email Sending (depends on Task 1 for PDF attachment)
  5. Task 4: ISDOC Export (no dependencies, can parallel with Tasks 1-3)

Sub-Phase 2B (sequential, depends on 2A):
  6. Task 6: Proforma Invoices (needs migration, depends on stable invoice CRUD)
  7. Task 7: Credit Notes (same migration as Task 6)
  8. Task 8: Recurring Invoices (independent of 6-7, can parallel)

Sub-Phase 2C (depends on 2B for migrations):
  9. Task 9: Multi-Currency (independent)
  10. Task 10: Overdue Detection (independent)
  11. Task 11: Status History (should come before Task 12)
  12. Task 12: Payment Reminders (depends on Task 3 for email, Task 10 for overdue, Task 11 for timeline)
```

---

## Migration Summary

| Migration | Task | Description |
|-----------|------|-------------|
| `003_invoice_relations.sql` | Tasks 6, 7 | Add related_invoice_id and relation_type to invoices |
| `004_recurring_invoices.sql` | Task 8 | Create recurring_invoices and recurring_invoice_items tables |
| `005_currency_support.sql` | Task 9 | Add exchange_rate_date, normalize exchange_rate columns |
| `006_invoice_status_history.sql` | Task 11 | Create invoice_status_history table |
| `007_payment_reminders.sql` | Task 12 | Create payment_reminders table |

---

## New Files Summary

### Backend

| File | Task | Purpose |
|------|------|---------|
| `internal/pdf/invoice_pdf.go` | 1 | PDF generation with maroto/v2 |
| `internal/pdf/supplier.go` | 1 | Load supplier info from settings |
| `internal/pdf/qr_payment.go` | 2 | QR Platba SPD generation |
| `internal/email/sender.go` | 3 | SMTP email sending |
| `internal/email/templates.go` | 3 | Invoice email templates |
| `internal/email/reminder_template.go` | 12 | Payment reminder email templates |
| `internal/isdoc/types.go` | 4 | ISDOC XML type definitions |
| `internal/isdoc/generator.go` | 4 | ISDOC XML generation |
| `internal/handler/sequence_handler.go` | 5 | Invoice sequence CRUD handler |
| `internal/repository/sequence_repo.go` | 5 | Invoice sequence repository |
| `internal/service/sequence_svc.go` | 5 | Invoice sequence service |
| `internal/domain/recurring.go` | 8 | RecurringInvoice domain struct |
| `internal/repository/recurring_repo.go` | 8 | Recurring invoice repository |
| `internal/service/recurring_svc.go` | 8 | Recurring invoice service |
| `internal/handler/recurring_handler.go` | 8 | Recurring invoice HTTP handler |
| `internal/cnb/client.go` | 9 | CNB exchange rate client |
| `internal/cnb/types.go` | 9 | CNB response parsing |
| `internal/domain/reminder.go` | 12 | PaymentReminder domain struct |
| `internal/repository/reminder_repo.go` | 12 | Payment reminder repository |
| `internal/service/reminder_svc.go` | 12 | Payment reminder service |

### Frontend

| File | Task | Purpose |
|------|------|---------|
| `frontend/src/routes/recurring/+page.svelte` | 8 | Recurring invoices list |
| `frontend/src/routes/recurring/new/+page.svelte` | 8 | Create recurring invoice |
| `frontend/src/routes/recurring/[id]/+page.svelte` | 8 | Recurring invoice detail/edit |

### Modified Shared Files

| File | Tasks | Changes |
|------|-------|---------|
| `internal/handler/router.go` | 5, 8 | Mount sequence and recurring handlers |
| `internal/handler/invoice_handler.go` | 1-4, 6, 7, 10, 12 | Add PDF, QR, email, ISDOC, settle, credit-note, remind routes |
| `internal/handler/helpers.go` | 6, 7, 8, 11, 12 | New DTOs for sequences, recurring, history, reminders |
| `internal/repository/interfaces.go` | 5, 8, 10, 11, 12 | New interfaces for sequences, recurring, reminders; updated InvoiceRepo |
| `internal/domain/invoice.go` | 6, 11 | Add RelatedInvoiceID, RelationType, InvoiceStatusChange |
| `internal/service/invoice_svc.go` | 6, 7, 10, 11 | Add SettleProforma, CreateCreditNote, CheckOverdue; record status changes |
| `internal/repository/invoice_repo.go` | 6, 10, 11 | Add related invoice columns, MarkOverdue, status history methods |
| `internal/cli/serve.go` | All | Wire new services, handlers, background jobs |
| `frontend/src/lib/api/client.ts` | All | Add new API methods and types |

---

## Out of Scope

- Invoice customization themes/branding beyond logo + basic colors (future polish)
- Digital signature / electronic seal on PDFs
- Automatic email reminders on schedule (only manual trigger in this RFC; automatic schedule is Phase 8)
- ISDOC 7.x support (only 6.0.2)
- Received invoices workflow (this RFC covers only issued invoices)
- OCR/AI for invoice data extraction (Phase 3 — expenses)
- Multi-language invoice templates (Czech only for now)
