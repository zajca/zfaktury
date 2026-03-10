# RFC-003: Expenses Phase

**Status:** Draft
**Date:** 2026-03-10

## Summary

Phase 3 extends the existing expense management with document upload and storage, AI-powered OCR for automatic data extraction from receipts and invoices, expense categories management, recurring expenses, and a tax-deductible marking workflow. The basic expense CRUD (domain, repository, service, handler, frontend list/create/edit pages) was completed in Phase 1.

## Existing Code Inventory

The following code was built and validated in RFC-001.

### Backend (Go)

| Layer | Files | What exists |
|-------|-------|-------------|
| Domain | `internal/domain/expense.go` | Expense struct with DocumentPath field (string), Amount (int64 halere), VATRatePercent, VATAmount, IsTaxDeductible, BusinessPercent |
| Domain | `internal/domain/filters.go` | ExpenseFilter with Category, VendorID, DateFrom, DateTo, Search, Limit, Offset |
| Repository | `internal/repository/expense_repo.go` | CRUD, List with filters, soft-delete, vendor JOIN |
| Repository | `internal/repository/interfaces.go` | ExpenseRepo interface (Create, Update, Delete, GetByID, List) |
| Service | `internal/service/expense_svc.go` | Validation, VAT calculation from rate, business percent defaults |
| Handler | `internal/handler/expense_handler.go` | REST endpoints (POST, GET, PUT, DELETE) |
| Handler | `internal/handler/helpers.go` | expenseRequest/expenseResponse DTOs with document_path field |
| Config | `internal/config/config.go` | OCRConfig struct with Provider and APIKey fields |
| Schema | `migrations/001_initial_schema.sql` | expenses table with document_path TEXT column |

### Frontend (SvelteKit)

| Page | Route | What exists |
|------|-------|-------------|
| Expenses list | `expenses/+page.svelte` | Real API calls, search, pagination |
| Expense create | `expenses/new/+page.svelte` | Form with all fields, submit to API |
| Expense detail/edit | `expenses/[id]/+page.svelte` | View/edit expense, delete with confirmation |
| API client | `src/lib/api/client.ts` | expensesApi with list, getById, create, update, delete |

### What Needs Building

1. **Document upload** — No file upload endpoint, no document storage, no document viewer
2. **AI OCR** — OCRConfig exists but no OCR service, no data extraction, no confirmation UI
3. **Expense categories** — Category is a free-text string field, no management UI or predefined list
4. **Recurring expenses** — No recurring expense concept exists
5. **Tax-deductible workflow** — IsTaxDeductible is a simple boolean, no workflow for bulk review

---

## Task 1: Document Storage & Upload

### Background

Expenses need attached documents (scanned receipts, PDF invoices, photos). Documents are stored in the local data directory under `documents/expenses/`. Each expense can have multiple documents (the existing single `document_path` field is replaced by a documents table).

### Storage Design

- **Location:** `{ZFAKTURY_DATA_DIR}/documents/expenses/{expense_id}/`
- **File naming:** `{uuid}_{original_filename}` (UUID prefix prevents collisions)
- **Supported formats:** PDF, PNG, JPG/JPEG, WEBP
- **Max file size:** 20 MB per file
- **Max files per expense:** 10

### Implementation Tasks

1. **Database migration (`internal/database/migrations/003_expense_documents.sql`)**

   ```sql
   -- +goose Up
   CREATE TABLE expense_documents (
       id INTEGER PRIMARY KEY AUTOINCREMENT,
       expense_id INTEGER NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
       filename TEXT NOT NULL,
       original_filename TEXT NOT NULL,
       content_type TEXT NOT NULL,
       file_size INTEGER NOT NULL,
       storage_path TEXT NOT NULL,
       created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
   );
   CREATE INDEX idx_expense_documents_expense_id ON expense_documents(expense_id);

   -- Migrate existing document_path data if any
   INSERT INTO expense_documents (expense_id, filename, original_filename, content_type, file_size, storage_path, created_at)
   SELECT id, document_path, document_path, 'application/octet-stream', 0, document_path, created_at
   FROM expenses WHERE document_path IS NOT NULL AND document_path != '';

   -- +goose Down
   DROP TABLE IF EXISTS expense_documents;
   ```

2. **Domain struct (`internal/domain/document.go`)**

   ```go
   type ExpenseDocument struct {
       ID               int64     `json:"id"`
       ExpenseID        int64     `json:"expense_id"`
       Filename         string    `json:"filename"`
       OriginalFilename string    `json:"original_filename"`
       ContentType      string    `json:"content_type"`
       FileSize         int64     `json:"file_size"`
       StoragePath      string    `json:"storage_path"`
       CreatedAt        time.Time `json:"created_at"`
   }
   ```

3. **Document repository (`internal/repository/document_repo.go`)**

   Add `DocumentRepo` interface to `interfaces.go`:

   ```go
   type DocumentRepo interface {
       Create(ctx context.Context, doc *ExpenseDocument) error
       Delete(ctx context.Context, id int64) error
       GetByID(ctx context.Context, id int64) (*ExpenseDocument, error)
       ListByExpenseID(ctx context.Context, expenseID int64) ([]ExpenseDocument, error)
       DeleteByExpenseID(ctx context.Context, expenseID int64) error
   }
   ```

   Implementation using `*sql.DB` with standard CRUD.

4. **Document storage service (`internal/service/document_svc.go`)**

   ```go
   type DocumentService struct {
       repo    repository.DocumentRepo
       dataDir string
   }

   func NewDocumentService(repo repository.DocumentRepo, dataDir string) *DocumentService

   func (s *DocumentService) Upload(ctx context.Context, expenseID int64, filename string, contentType string, reader io.Reader) (*domain.ExpenseDocument, error)
   func (s *DocumentService) Delete(ctx context.Context, id int64) error
   func (s *DocumentService) GetFile(ctx context.Context, id int64) (io.ReadCloser, *domain.ExpenseDocument, error)
   func (s *DocumentService) ListByExpense(ctx context.Context, expenseID int64) ([]domain.ExpenseDocument, error)
   ```

   Upload validates:
   - Content type is in allowed list (PDF, PNG, JPG, WEBP)
   - File size does not exceed 20 MB (validated via `io.LimitReader`)
   - Expense exists (calls ExpenseRepo.GetByID)
   - Document count per expense does not exceed 10
   - Generates UUID filename, creates directory, writes file to disk
   - Creates database record

5. **Document handler (`internal/handler/document_handler.go`)**

   Routes mounted on expense handler:

   ```
   POST   /api/v1/expenses/{id}/documents     Upload document (multipart/form-data)
   GET    /api/v1/expenses/{id}/documents      List documents for expense
   GET    /api/v1/documents/{id}               Get document metadata
   GET    /api/v1/documents/{id}/download      Download document file
   DELETE /api/v1/documents/{id}               Delete document
   ```

   Upload handler:
   - Parses multipart form with `r.ParseMultipartForm(20 << 20)` (20 MB max memory)
   - Reads file from `file` form field
   - Calls `DocumentService.Upload()`
   - Returns 201 with document metadata

   Download handler:
   - Sets `Content-Type` and `Content-Disposition` headers
   - Streams file via `io.Copy`

6. **Update expense response to include documents**

   Add `Documents []documentResponse` to `expenseResponse` in `helpers.go`.

   Add document DTOs:

   ```go
   type documentResponse struct {
       ID               int64  `json:"id"`
       ExpenseID        int64  `json:"expense_id"`
       OriginalFilename string `json:"original_filename"`
       ContentType      string `json:"content_type"`
       FileSize         int64  `json:"file_size"`
       DownloadURL      string `json:"download_url"`
       CreatedAt        string `json:"created_at"`
   }
   ```

7. **Mount routes in `router.go`**

   Add `DocumentService` to `NewRouter` parameters. Mount document routes under expenses and at `/api/v1/documents/`.

8. **Wire in `serve.go`**

   Create `DocumentRepository`, `DocumentService` (with `cfg.DataDir`), wire to handler.

9. **Frontend: document upload component**

   - `frontend/src/lib/components/DocumentUpload.svelte`
   - Drag-and-drop zone + file input button
   - Progress indicator during upload
   - Preview for images, icon for PDFs
   - Delete button per document

10. **Frontend: document viewer**

    - `frontend/src/lib/components/DocumentViewer.svelte`
    - Modal/lightbox for images
    - Embedded PDF viewer (iframe with `application/pdf`)
    - Download button

11. **Frontend: integrate into expense pages**

    - `expenses/[id]/+page.svelte` — show document list, upload area, viewer
    - `expenses/new/+page.svelte` — upload area (upload after expense is created, two-step: create expense, then upload)

12. **Add API methods to `client.ts`**

    ```typescript
    export interface ExpenseDocument {
        id: number;
        expense_id: number;
        original_filename: string;
        content_type: string;
        file_size: number;
        download_url: string;
        created_at: string;
    }

    export const documentsApi = {
        listByExpense(expenseId: number): Promise<ExpenseDocument[]>,
        upload(expenseId: number, file: File): Promise<ExpenseDocument>,
        delete(id: number): Promise<void>,
        downloadUrl(id: number): string,
    };
    ```

    The `upload` method uses `FormData` with `fetch` (not JSON).

### Tests

- `internal/repository/document_repo_test.go` — CRUD, ListByExpenseID, cascade delete
- `internal/service/document_svc_test.go` — upload validation (size, type, count limit), delete with file cleanup
- `internal/handler/document_handler_test.go` — multipart upload, download, list, delete via httptest

### Acceptance Criteria

- [ ] User can upload PDF/image documents to an expense via the UI
- [ ] Documents are stored at `{ZFAKTURY_DATA_DIR}/documents/expenses/{expense_id}/`
- [ ] User can view uploaded images in a lightbox and PDFs in an embedded viewer
- [ ] User can download documents
- [ ] User can delete documents (file removed from disk + DB record deleted)
- [ ] File size limit (20 MB) and type validation enforced
- [ ] Max 10 documents per expense enforced
- [ ] Existing `document_path` data migrated to new table
- [ ] All tests pass

---

## Task 2: AI OCR Integration

### Background

OCR extracts structured data from uploaded expense documents (receipts, invoices) to pre-fill expense fields. The OCR provider is configurable — the system ships with an OpenAI Vision API implementation but uses a pluggable interface so other providers (Google Document AI, local Tesseract, etc.) can be added.

### OCR Interface Design

```go
// OCRProvider extracts expense data from a document image/PDF.
type OCRProvider interface {
    ExtractExpenseData(ctx context.Context, reader io.Reader, contentType string) (*OCRResult, error)
    Name() string
}

// OCRResult holds the extracted data from OCR processing.
type OCRResult struct {
    VendorName     string  `json:"vendor_name"`
    VendorICO      string  `json:"vendor_ico"`
    ExpenseNumber  string  `json:"expense_number"`
    IssueDate      string  `json:"issue_date"`       // YYYY-MM-DD
    TotalAmount    int64   `json:"total_amount"`      // halere
    VATAmount      int64   `json:"vat_amount"`        // halere
    VATRatePercent int     `json:"vat_rate_percent"`
    CurrencyCode   string  `json:"currency_code"`
    Description    string  `json:"description"`
    Confidence     float64 `json:"confidence"`        // 0.0-1.0
    RawText        string  `json:"raw_text"`
}
```

### Implementation Tasks

1. **OCR domain types (`internal/domain/ocr.go`)**

   Define `OCRResult` struct (as above, without methods — pure data).

2. **OCR provider interface (`internal/ocr/provider.go`)**

   ```go
   package ocr

   type Provider interface {
       ExtractExpenseData(ctx context.Context, reader io.Reader, contentType string) (*domain.OCRResult, error)
       Name() string
   }
   ```

3. **OpenAI Vision provider (`internal/ocr/openai.go`)**

   ```go
   type OpenAIProvider struct {
       apiKey string
       model  string // default: "gpt-4o"
       client *http.Client
   }

   func NewOpenAIProvider(apiKey string) *OpenAIProvider
   ```

   - Sends image/PDF as base64 in a chat completion request
   - System prompt instructs the model to extract Czech invoice/receipt fields
   - Structured output via JSON mode
   - Maps response to `domain.OCRResult`
   - Timeout: 60 seconds
   - Max input: 20 MB (same as document upload limit)

4. **OCR service (`internal/service/ocr_svc.go`)**

   ```go
   type OCRService struct {
       provider    ocr.Provider
       documentSvc *DocumentService
   }

   func NewOCRService(provider ocr.Provider, documentSvc *DocumentService) *OCRService

   func (s *OCRService) ProcessDocument(ctx context.Context, documentID int64) (*domain.OCRResult, error)
   ```

   - Retrieves document file via DocumentService
   - Passes to OCR provider
   - Returns extracted data for user confirmation (does NOT auto-save)

5. **OCR handler (`internal/handler/ocr_handler.go`)**

   ```
   POST /api/v1/documents/{id}/ocr    Process document with OCR
   ```

   Request: empty body (document ID is in URL)

   Response (200):
   ```json
   {
       "vendor_name": "Alza.cz a.s.",
       "vendor_ico": "27082440",
       "expense_number": "FA-2026-001234",
       "issue_date": "2026-03-01",
       "total_amount": 121000,
       "vat_amount": 21000,
       "vat_rate_percent": 21,
       "currency_code": "CZK",
       "description": "USB-C kabel 2m",
       "confidence": 0.85,
       "raw_text": "..."
   }
   ```

   Error responses:
   - 404: document not found
   - 422: unsupported content type for OCR
   - 502: OCR provider error
   - 503: OCR not configured (no API key)

6. **Wire OCR in `serve.go`**

   ```go
   var ocrProvider ocr.Provider
   if cfg.OCR.APIKey != "" {
       ocrProvider = ocr.NewOpenAIProvider(cfg.OCR.APIKey)
   }
   ocrSvc := service.NewOCRService(ocrProvider, documentSvc)
   ```

   When no API key is configured, `OCRService.ProcessDocument()` returns a clear error: "OCR is not configured. Set ocr.api_key in config.toml".

7. **Mount OCR routes in `router.go`**

   Add `ocrSvc` to `NewRouter`, mount under `/api/v1/documents/{id}/ocr`.

8. **Frontend: OCR confirmation UI**

   - Add "Scan with OCR" button to document viewer (only for uploaded documents)
   - On click: call OCR endpoint, show loading spinner
   - Show extracted data in a confirmation dialog:
     - Side-by-side: document preview | extracted fields
     - Each field editable before applying
     - Confidence indicator (color-coded: green > 0.8, yellow > 0.5, red < 0.5)
     - "Apply" button fills expense form fields
     - "Cancel" button discards OCR results
   - Add to `expenses/new/+page.svelte` and `expenses/[id]/+page.svelte`

9. **Add API methods to `client.ts`**

   ```typescript
   export interface OCRResult {
       vendor_name: string;
       vendor_ico: string;
       expense_number: string;
       issue_date: string;
       total_amount: number;
       vat_amount: number;
       vat_rate_percent: number;
       currency_code: string;
       description: string;
       confidence: number;
       raw_text: string;
   }

   // Add to documentsApi:
   processOCR(documentId: number): Promise<OCRResult>
   ```

### Tests

- `internal/ocr/openai_test.go` — unit tests with httptest (recorded API responses), test prompt construction, response parsing, error handling
- `internal/service/ocr_svc_test.go` — test with mock Provider interface, document not found, provider error, no provider configured
- `internal/handler/ocr_handler_test.go` — HTTP tests for success, not found, provider error

### Acceptance Criteria

- [ ] User can trigger OCR on any uploaded document
- [ ] OCR results displayed in confirmation dialog with editable fields
- [ ] User can review, modify, and apply extracted data to expense form
- [ ] OpenAI Vision provider works for Czech receipts and invoices
- [ ] OCR gracefully degrades when not configured (clear error message)
- [ ] Provider interface allows adding alternative OCR backends
- [ ] Confidence score displayed for transparency
- [ ] All tests pass (with mocked HTTP for OpenAI API)

---

## Task 3: Expense Categories Management

### Background

Currently `category` is a free-text string field on expenses. This task adds a managed categories table with predefined defaults for Czech OSVC, a CRUD API, and a category picker in the UI. Free-text input remains possible (user can type a custom category).

### Default Categories (Czech OSVC)

| Key | Czech label | English label |
|-----|-------------|---------------|
| `office_supplies` | Kancelarske potreby | Office supplies |
| `software` | Software a licence | Software & licenses |
| `hardware` | Hardware a technika | Hardware & equipment |
| `telecom` | Telefon a internet | Telecom & internet |
| `travel` | Cestovni naklady | Travel expenses |
| `fuel` | Pohonne hmoty | Fuel |
| `vehicle` | Provoz vozidla | Vehicle operation |
| `rent` | Najem | Rent |
| `utilities` | Energie | Utilities |
| `education` | Vzdelavani | Education & training |
| `marketing` | Marketing a reklama | Marketing & advertising |
| `insurance` | Pojisteni | Insurance |
| `accounting` | Uctovnictvi a dane | Accounting & taxes |
| `postage` | Postovne | Postage & shipping |
| `services` | Sluzby | Services |
| `other` | Ostatni | Other |

### Implementation Tasks

1. **Database migration (`internal/database/migrations/004_expense_categories.sql`)**

   ```sql
   -- +goose Up
   CREATE TABLE expense_categories (
       id INTEGER PRIMARY KEY AUTOINCREMENT,
       key TEXT NOT NULL UNIQUE,
       label_cs TEXT NOT NULL,
       label_en TEXT NOT NULL,
       color TEXT NOT NULL DEFAULT '#6B7280',
       sort_order INTEGER NOT NULL DEFAULT 0,
       is_default INTEGER NOT NULL DEFAULT 0,
       created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
       deleted_at TEXT
   );
   CREATE INDEX idx_expense_categories_key ON expense_categories(key);
   CREATE INDEX idx_expense_categories_deleted_at ON expense_categories(deleted_at);

   -- Insert default categories
   INSERT INTO expense_categories (key, label_cs, label_en, sort_order, is_default) VALUES
   ('office_supplies', 'Kancelarske potreby', 'Office supplies', 1, 1),
   ('software', 'Software a licence', 'Software & licenses', 2, 1),
   ('hardware', 'Hardware a technika', 'Hardware & equipment', 3, 1),
   ('telecom', 'Telefon a internet', 'Telecom & internet', 4, 1),
   ('travel', 'Cestovni naklady', 'Travel expenses', 5, 1),
   ('fuel', 'Pohonne hmoty', 'Fuel', 6, 1),
   ('vehicle', 'Provoz vozidla', 'Vehicle operation', 7, 1),
   ('rent', 'Najem', 'Rent', 8, 1),
   ('utilities', 'Energie', 'Utilities', 9, 1),
   ('education', 'Vzdelavani', 'Education & training', 10, 1),
   ('marketing', 'Marketing a reklama', 'Marketing & advertising', 11, 1),
   ('insurance', 'Pojisteni', 'Insurance', 12, 1),
   ('accounting', 'Uctovnictvi a dane', 'Accounting & taxes', 13, 1),
   ('postage', 'Postovne', 'Postage & shipping', 14, 1),
   ('services', 'Sluzby', 'Services', 15, 1),
   ('other', 'Ostatni', 'Other', 99, 1);

   -- +goose Down
   DROP TABLE IF EXISTS expense_categories;
   ```

2. **Domain struct (`internal/domain/category.go`)**

   ```go
   type ExpenseCategory struct {
       ID        int64      `json:"id"`
       Key       string     `json:"key"`
       LabelCS   string     `json:"label_cs"`
       LabelEN   string     `json:"label_en"`
       Color     string     `json:"color"`
       SortOrder int        `json:"sort_order"`
       IsDefault bool       `json:"is_default"`
       CreatedAt time.Time  `json:"created_at"`
       DeletedAt *time.Time `json:"deleted_at,omitempty"`
   }
   ```

3. **Category repository (`internal/repository/category_repo.go`)**

   Add `CategoryRepo` interface to `interfaces.go`:

   ```go
   type CategoryRepo interface {
       Create(ctx context.Context, cat *domain.ExpenseCategory) error
       Update(ctx context.Context, cat *domain.ExpenseCategory) error
       Delete(ctx context.Context, id int64) error
       GetByID(ctx context.Context, id int64) (*domain.ExpenseCategory, error)
       GetByKey(ctx context.Context, key string) (*domain.ExpenseCategory, error)
       List(ctx context.Context) ([]domain.ExpenseCategory, error)
   }
   ```

4. **Category service (`internal/service/category_svc.go`)**

   ```go
   type CategoryService struct {
       repo repository.CategoryRepo
   }

   func NewCategoryService(repo repository.CategoryRepo) *CategoryService

   func (s *CategoryService) Create(ctx context.Context, cat *domain.ExpenseCategory) error
   func (s *CategoryService) Update(ctx context.Context, cat *domain.ExpenseCategory) error
   func (s *CategoryService) Delete(ctx context.Context, id int64) error
   func (s *CategoryService) List(ctx context.Context) ([]domain.ExpenseCategory, error)
   ```

   Validation:
   - Key is required, unique, lowercase alphanumeric + underscores
   - LabelCS is required
   - Default categories cannot be deleted (soft-delete disabled for `is_default=1`)

5. **Category handler (`internal/handler/category_handler.go`)**

   ```
   GET    /api/v1/expense-categories         List all categories
   POST   /api/v1/expense-categories         Create custom category
   PUT    /api/v1/expense-categories/{id}    Update category
   DELETE /api/v1/expense-categories/{id}    Delete custom category
   ```

6. **Mount in `router.go`, wire in `serve.go`**

7. **Frontend: category management**

   - `frontend/src/routes/settings/categories/+page.svelte` — list, create, edit, delete categories
   - Link from settings page

8. **Frontend: category picker in expense forms**

   - `frontend/src/lib/components/CategoryPicker.svelte`
   - Dropdown with category list (fetched from API)
   - Color indicator per category
   - Option to type custom value if category not in list
   - Replace free-text category input in `expenses/new/+page.svelte` and `expenses/[id]/+page.svelte`

9. **Add API types and methods to `client.ts`**

   ```typescript
   export interface ExpenseCategory {
       id: number;
       key: string;
       label_cs: string;
       label_en: string;
       color: string;
       sort_order: number;
       is_default: boolean;
       created_at: string;
   }

   export const categoriesApi = {
       list(): Promise<ExpenseCategory[]>,
       create(data: Partial<ExpenseCategory>): Promise<ExpenseCategory>,
       update(id: number, data: Partial<ExpenseCategory>): Promise<ExpenseCategory>,
       delete(id: number): Promise<void>,
   };
   ```

### Tests

- `internal/repository/category_repo_test.go` — CRUD, GetByKey, List order, soft-delete
- `internal/service/category_svc_test.go` — validation (key format, required fields, default protection)
- `internal/handler/category_handler_test.go` — HTTP tests

### Acceptance Criteria

- [ ] Default categories seeded on first migration
- [ ] User can create, edit, and delete custom categories
- [ ] Default categories cannot be deleted
- [ ] Category picker in expense form shows all categories with colors
- [ ] User can still type a custom category not in the list
- [ ] Category filter on expense list works with category keys
- [ ] All tests pass

---

## Task 4: Recurring Expenses

### Background

Many business expenses recur on a regular schedule (monthly software subscriptions, rent, internet, insurance). Recurring expenses create a template from which new expense records are automatically generated.

### Design

- A recurring expense is a template — it does not appear in the main expense list
- The system generates actual expense records from templates based on schedule
- Generation happens on server startup and can be triggered via API
- Each generated expense links back to its recurring template

### Implementation Tasks

1. **Database migration (`internal/database/migrations/005_recurring_expenses.sql`)**

   ```sql
   -- +goose Up
   CREATE TABLE recurring_expenses (
       id INTEGER PRIMARY KEY AUTOINCREMENT,
       vendor_id INTEGER REFERENCES contacts(id),
       category TEXT,
       description TEXT NOT NULL,
       amount INTEGER NOT NULL DEFAULT 0,
       currency_code TEXT NOT NULL DEFAULT 'CZK',
       vat_rate_percent INTEGER NOT NULL DEFAULT 0,
       is_tax_deductible INTEGER NOT NULL DEFAULT 1,
       business_percent INTEGER NOT NULL DEFAULT 100,
       payment_method TEXT NOT NULL DEFAULT 'bank_transfer',
       interval TEXT NOT NULL CHECK (interval IN ('monthly', 'quarterly', 'yearly')),
       day_of_month INTEGER NOT NULL DEFAULT 1,
       start_date TEXT NOT NULL,
       end_date TEXT,
       last_generated_date TEXT,
       is_active INTEGER NOT NULL DEFAULT 1,
       notes TEXT,
       created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
       updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
       deleted_at TEXT
   );
   CREATE INDEX idx_recurring_expenses_is_active ON recurring_expenses(is_active);
   CREATE INDEX idx_recurring_expenses_deleted_at ON recurring_expenses(deleted_at);

   -- Link generated expenses back to their template
   ALTER TABLE expenses ADD COLUMN recurring_expense_id INTEGER REFERENCES recurring_expenses(id);
   CREATE INDEX idx_expenses_recurring_expense_id ON expenses(recurring_expense_id);

   -- +goose Down
   -- Remove the column (SQLite doesn't support DROP COLUMN before 3.35.0, so recreate)
   CREATE TABLE expenses_backup AS SELECT
       id, vendor_id, expense_number, category, description,
       issue_date, amount, currency_code, exchange_rate_amount,
       vat_rate_percent, vat_amount,
       is_tax_deductible, business_percent, payment_method,
       document_path, notes, created_at, updated_at, deleted_at
   FROM expenses;
   DROP TABLE expenses;
   ALTER TABLE expenses_backup RENAME TO expenses;
   DROP TABLE IF EXISTS recurring_expenses;
   ```

2. **Domain struct (`internal/domain/recurring_expense.go`)**

   ```go
   type RecurringExpense struct {
       ID                int64      `json:"id"`
       VendorID          *int64     `json:"vendor_id,omitempty"`
       Vendor            *Contact   `json:"vendor,omitempty"`
       Category          string     `json:"category"`
       Description       string     `json:"description"`
       Amount            Amount     `json:"amount"`
       CurrencyCode      string     `json:"currency_code"`
       VATRatePercent    int        `json:"vat_rate_percent"`
       IsTaxDeductible   bool       `json:"is_tax_deductible"`
       BusinessPercent   int        `json:"business_percent"`
       PaymentMethod     string     `json:"payment_method"`
       Interval          string     `json:"interval"` // monthly, quarterly, yearly
       DayOfMonth        int        `json:"day_of_month"`
       StartDate         time.Time  `json:"start_date"`
       EndDate           *time.Time `json:"end_date,omitempty"`
       LastGeneratedDate *time.Time `json:"last_generated_date,omitempty"`
       IsActive          bool       `json:"is_active"`
       Notes             string     `json:"notes"`
       CreatedAt         time.Time  `json:"created_at"`
       UpdatedAt         time.Time  `json:"updated_at"`
       DeletedAt         *time.Time `json:"deleted_at,omitempty"`
   }
   ```

3. **Recurring expense repository (`internal/repository/recurring_expense_repo.go`)**

   Add `RecurringExpenseRepo` interface to `interfaces.go`:

   ```go
   type RecurringExpenseRepo interface {
       Create(ctx context.Context, re *domain.RecurringExpense) error
       Update(ctx context.Context, re *domain.RecurringExpense) error
       Delete(ctx context.Context, id int64) error
       GetByID(ctx context.Context, id int64) (*domain.RecurringExpense, error)
       List(ctx context.Context, activeOnly bool) ([]domain.RecurringExpense, error)
       UpdateLastGenerated(ctx context.Context, id int64, date time.Time) error
   }
   ```

4. **Recurring expense service (`internal/service/recurring_expense_svc.go`)**

   ```go
   type RecurringExpenseService struct {
       repo       repository.RecurringExpenseRepo
       expenseRepo repository.ExpenseRepo
   }

   func NewRecurringExpenseService(repo repository.RecurringExpenseRepo, expenseRepo repository.ExpenseRepo) *RecurringExpenseService

   func (s *RecurringExpenseService) Create(ctx context.Context, re *domain.RecurringExpense) error
   func (s *RecurringExpenseService) Update(ctx context.Context, re *domain.RecurringExpense) error
   func (s *RecurringExpenseService) Delete(ctx context.Context, id int64) error
   func (s *RecurringExpenseService) GetByID(ctx context.Context, id int64) (*domain.RecurringExpense, error)
   func (s *RecurringExpenseService) List(ctx context.Context, activeOnly bool) ([]domain.RecurringExpense, error)
   func (s *RecurringExpenseService) Activate(ctx context.Context, id int64) error
   func (s *RecurringExpenseService) Deactivate(ctx context.Context, id int64) error
   func (s *RecurringExpenseService) GeneratePending(ctx context.Context, asOfDate time.Time) ([]domain.Expense, error)
   ```

   `GeneratePending()` logic:
   - Fetch all active recurring expenses where `end_date` is null or >= `asOfDate`
   - For each: calculate which periods are due based on `interval`, `start_date`, `day_of_month`, and `last_generated_date`
   - Create expense records for each missing period
   - Set `recurring_expense_id` on generated expenses
   - Update `last_generated_date` on the template
   - Return all newly created expenses

5. **Recurring expense handler (`internal/handler/recurring_expense_handler.go`)**

   ```
   GET    /api/v1/recurring-expenses              List all
   POST   /api/v1/recurring-expenses              Create
   GET    /api/v1/recurring-expenses/{id}          Get by ID
   PUT    /api/v1/recurring-expenses/{id}          Update
   DELETE /api/v1/recurring-expenses/{id}          Delete
   POST   /api/v1/recurring-expenses/{id}/activate    Activate
   POST   /api/v1/recurring-expenses/{id}/deactivate  Deactivate
   POST   /api/v1/recurring-expenses/generate      Generate pending expenses
   ```

   DTOs in `helpers.go` or in `recurring_expense_handler.go`.

6. **Auto-generate on server start**

   In `serve.go`, after migrations, call:
   ```go
   generated, err := recurringExpenseSvc.GeneratePending(context.Background(), time.Now())
   if err != nil {
       slog.Error("failed to generate recurring expenses", "error", err)
   } else if len(generated) > 0 {
       slog.Info("generated recurring expenses", "count", len(generated))
   }
   ```

7. **Mount routes in `router.go`, wire in `serve.go`**

8. **Update Expense domain to include RecurringExpenseID**

   Add `RecurringExpenseID *int64` field to `domain.Expense`.
   Update expense repository SQL to include the new column.
   Update expense DTOs to include `recurring_expense_id` (read-only in response).

9. **Frontend: recurring expenses page**

   - `frontend/src/routes/expenses/recurring/+page.svelte` — list recurring expenses, active/inactive toggle
   - `frontend/src/routes/expenses/recurring/new/+page.svelte` — create form (same fields as expense + interval, day_of_month, start_date, end_date)
   - `frontend/src/routes/expenses/recurring/[id]/+page.svelte` — detail/edit, activate/deactivate, view generated expenses
   - "Generate Now" button that calls the generate endpoint
   - Navigation link in sidebar or expenses sub-nav

10. **Add API types and methods to `client.ts`**

    ```typescript
    export interface RecurringExpense {
        id: number;
        vendor_id?: number;
        category: string;
        description: string;
        amount: number;
        currency_code: string;
        vat_rate_percent: number;
        is_tax_deductible: boolean;
        business_percent: number;
        payment_method: string;
        interval: 'monthly' | 'quarterly' | 'yearly';
        day_of_month: number;
        start_date: string;
        end_date?: string;
        last_generated_date?: string;
        is_active: boolean;
        notes: string;
        created_at: string;
        updated_at: string;
    }

    export const recurringExpensesApi = {
        list(activeOnly?: boolean): Promise<RecurringExpense[]>,
        getById(id: number): Promise<RecurringExpense>,
        create(data: Partial<RecurringExpense>): Promise<RecurringExpense>,
        update(id: number, data: Partial<RecurringExpense>): Promise<RecurringExpense>,
        delete(id: number): Promise<void>,
        activate(id: number): Promise<RecurringExpense>,
        deactivate(id: number): Promise<RecurringExpense>,
        generate(): Promise<Expense[]>,
    };
    ```

### Tests

- `internal/repository/recurring_expense_repo_test.go` — CRUD, List active only, UpdateLastGenerated
- `internal/service/recurring_expense_svc_test.go` — generation logic (monthly/quarterly/yearly, boundary dates, end date respected, idempotent generation), validation
- `internal/handler/recurring_expense_handler_test.go` — HTTP tests

### Acceptance Criteria

- [ ] User can create recurring expense templates with monthly/quarterly/yearly intervals
- [ ] Pending expenses auto-generate on server startup
- [ ] User can manually trigger generation via API/UI
- [ ] Generated expenses link back to their template (recurring_expense_id)
- [ ] User can activate/deactivate recurring templates
- [ ] End date is respected (no generation past end date)
- [ ] Generation is idempotent (running twice does not create duplicates)
- [ ] All tests pass

---

## Task 5: Tax-Deductible Review Workflow

### Background

Czech OSVC need to review expenses for tax deductibility before filing. Currently `is_tax_deductible` is a simple boolean set at creation time. This task adds a review workflow: expenses start as "unreviewed", and the user can bulk-review them for tax periods.

### Design

The review is lightweight — no new states, no approval chain. It adds a `tax_reviewed_at` timestamp to expenses. Unreviewed expenses have `tax_reviewed_at IS NULL`. The UI provides a bulk review screen filtered by date range (typically a tax period: month for VAT, year for income tax).

### Implementation Tasks

1. **Database migration (`internal/database/migrations/006_tax_review.sql`)**

   ```sql
   -- +goose Up
   ALTER TABLE expenses ADD COLUMN tax_reviewed_at TEXT;
   CREATE INDEX idx_expenses_tax_reviewed_at ON expenses(tax_reviewed_at);

   -- +goose Down
   -- SQLite: recreate table without the column if needed
   ```

2. **Update domain struct**

   Add to `domain.Expense`:
   ```go
   TaxReviewedAt *time.Time `json:"tax_reviewed_at,omitempty"`
   ```

3. **Update expense repository**

   Add method to `ExpenseRepo` interface:
   ```go
   MarkTaxReviewed(ctx context.Context, ids []int64, reviewedAt time.Time) error
   UnmarkTaxReviewed(ctx context.Context, ids []int64) error
   ```

   Update existing SQL queries to include `tax_reviewed_at` in SELECT/INSERT/UPDATE.

4. **Update expense filter**

   Add to `ExpenseFilter`:
   ```go
   TaxReviewed *bool `json:"tax_reviewed"` // nil=all, true=reviewed, false=unreviewed
   ```

   Repository adds `AND tax_reviewed_at IS NOT NULL` or `AND tax_reviewed_at IS NULL` to WHERE clause.

5. **Add review endpoints to expense handler**

   ```
   POST /api/v1/expenses/review      Bulk mark as tax-reviewed
   POST /api/v1/expenses/unreview    Bulk unmark
   ```

   Request body:
   ```json
   {
       "expense_ids": [1, 2, 3]
   }
   ```

   Response: 200 with count of updated records.

6. **Add list filter parameter**

   `GET /api/v1/expenses?tax_reviewed=false` — filter by review status.

7. **Update DTOs**

   Add `tax_reviewed_at` to `expenseResponse`.
   Add `tax_reviewed` filter param parsing in List handler.

8. **Frontend: tax review page**

   - `frontend/src/routes/expenses/review/+page.svelte`
   - Date range picker (defaults to current month for VAT, current year for income tax)
   - Table of unreviewed expenses for the period
   - Checkbox per row + "Select All" toggle
   - Summary at top: total count, total amount, total VAT
   - "Mark as Reviewed" bulk action button
   - "Mark as Not Tax-Deductible" button (sets `is_tax_deductible=false` + marks reviewed)
   - Filter toggle: show reviewed / unreviewed / all

9. **Update expense list page**

   - Add visual indicator (icon/badge) for reviewed vs. unreviewed expenses
   - Add filter option for review status

10. **Add API methods to `client.ts`**

    Update `Expense` interface:
    ```typescript
    tax_reviewed_at?: string;
    ```

    Add to `expensesApi`:
    ```typescript
    markReviewed(ids: number[]): Promise<{ count: number }>,
    unmarkReviewed(ids: number[]): Promise<{ count: number }>,
    ```

    Update list params:
    ```typescript
    list(params?: { ..., tax_reviewed?: boolean }): Promise<ListResponse<Expense>>,
    ```

### Tests

- `internal/repository/expense_repo_test.go` — add tests for MarkTaxReviewed, UnmarkTaxReviewed, List with tax_reviewed filter
- `internal/service/expense_svc_test.go` — add tests for review operations
- `internal/handler/expense_handler_test.go` — HTTP tests for review/unreview endpoints, filter

### Acceptance Criteria

- [ ] Expenses have a `tax_reviewed_at` timestamp (nullable)
- [ ] User can bulk-mark expenses as tax-reviewed
- [ ] User can bulk-unmark reviewed expenses
- [ ] Expense list can be filtered by review status
- [ ] Review page shows unreviewed expenses for a date range with totals
- [ ] Visual indicator distinguishes reviewed from unreviewed expenses
- [ ] All tests pass

---

## Implementation Order

```
1. Document Storage & Upload (Task 1)
   - Foundation for OCR (Task 2 depends on this)
   - Most impactful standalone feature

2. AI OCR Integration (Task 2)
   - Depends on Task 1 (needs document files to process)
   - High value feature that differentiates the app

3. Expense Categories Management (Task 3)
   - Independent of Tasks 1-2
   - Can be parallelized with Task 1 or Task 2

4. Tax-Deductible Review Workflow (Task 5)
   - Independent of Tasks 1-3
   - Can be parallelized with Task 3

5. Recurring Expenses (Task 4)
   - Independent of other tasks
   - Can be parallelized with Tasks 3/5
   - Listed last because it adds most new complexity (generation logic)
```

Tasks 3, 4, and 5 are independent and can be built in parallel by different agents. Tasks 1 and 2 must be sequential.

---

## Shared File Edits

The following shared files require coordinated edits across tasks:

| File | Tasks | Changes |
|------|-------|---------|
| `internal/repository/interfaces.go` | 1, 3, 4 | Add DocumentRepo, CategoryRepo, RecurringExpenseRepo |
| `internal/handler/router.go` | 1, 2, 3, 4, 5 | Mount new handlers, add new service params |
| `internal/handler/helpers.go` | 1, 2, 3, 4, 5 | Add DTOs for documents, categories, recurring expenses, OCR |
| `internal/cli/serve.go` | 1, 2, 3, 4 | Wire new repos, services, handlers |
| `frontend/src/lib/api/client.ts` | 1, 2, 3, 4, 5 | Add types and API methods |
| `internal/domain/expense.go` | 4, 5 | Add RecurringExpenseID, TaxReviewedAt fields |
| `internal/domain/filters.go` | 5 | Add TaxReviewed filter |
| `internal/repository/expense_repo.go` | 4, 5 | Add recurring_expense_id column, tax_reviewed_at column, new methods |

---

## Out of Scope

- Automatic expense recognition from bank statements (Phase 6 — Banking)
- Expense reporting and charts (Phase 7 — Web UI)
- Asset register / depreciation (Phase 8 — Polish)
- Vehicle logbook integration (Phase 8 — Polish)
- Multi-currency expense conversion with CNB rates (future enhancement)
- Expense approval workflow (single-user app, not needed)
- Batch import from CSV/Excel (future enhancement)
