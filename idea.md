# ZFaktury - Complete Solo Entrepreneur Management System

## Vision

A self-contained, single-binary application for Czech sole proprietors (OSVČ) without inventory needs. Covers the entire financial lifecycle: invoicing, expense tracking, contact management, tax filings, and government communication. All data stored locally in SQLite — no cloud dependency, full data ownership.

## Target User

- Small Czech OSVČ (freelancer, consultant, contractor)
- No employees, no warehouse/inventory
- Flat-rate expenses (paušální výdaje) or real expenses (skutečné výdaje)
- May or may not be VAT registered (plátce/neplátce DPH)
- Wants one tool to replace scattered spreadsheets and online services

---

## Core Modules

### 1. Invoice Management (Správa faktur)

#### Features
- Create, edit, duplicate, and delete invoices
- Invoice numbering with configurable sequences (per year, continuous)
- Support for both VAT and non-VAT invoices (automatic based on user's VAT status)
- Multiple VAT rates (21%, 12%, 0%) with correct line-item calculation
- Reverse charge mechanism for EU B2B services
- Proforma invoices (zálohové faktury) and their settlement
- Credit notes (dobropisy)
- Recurring invoices with configurable intervals
- Invoice templates (customizable HTML/CSS layout, logo, colors, footer text)
- PDF generation from templates
- Send invoice via email (SMTP configuration)
- Invoice status tracking: draft → sent → viewed → paid → overdue
- Payment reminders (automatic or manual)
- QR code for bank payment (Czech QR Platba standard - SPD format)
- Due date tracking and overdue notifications
- Multi-currency support with manual or fetched exchange rates (CNB daily rates)
- FIO Bank API integration for automatic payment matching
- Export to ISDOC format (Czech invoicing standard)

#### Data Model (key fields)
- Invoice number, issue date, due date, delivery date (DUZP)
- Supplier info, customer info (linked to contacts)
- Line items: description, quantity, unit, unit price, VAT rate, total
- Currency, exchange rate, payment method, bank account
- Variable symbol (variabilní symbol), constant symbol
- Notes, internal notes
- Status, sent date, paid date, paid amount

### 2. Expense Management (Správa nákladů)

#### Features
- Manual expense entry with categorization
- Upload expense documents: PDF, JPG, PNG
- AI-powered OCR for automatic data extraction from uploaded documents
  - Vendor name, ICO, DIC
  - Date, amount, VAT breakdown
  - Invoice/receipt number
- Expense categories aligned with Czech tax categories
- Link expenses to contacts (vendors)
- Recurring expenses
- Expense approval workflow (mark as tax-deductible or not)
- Split expenses between business and personal use (percentage)
- Support for flat-rate expenses (paušální výdaje) — 60%, 40%, 30%, 80% based on activity type
- Fuel log and vehicle expense tracking (if using car for business)
- Cash vs. bank payment tracking
- Receipt/document archive with full-text search

#### OCR Implementation
- Use AI vision models (local or API-based) for document text extraction
- Structured data extraction pipeline: raw text → parsed fields → user confirmation
- Learning from corrections to improve future extractions

### 3. Contact Management (Správa kontaktů)

#### Features
- Company and individual contact records
- ARES integration: fetch company details by ICO (name, address, DIC, legal form)
  - API endpoint: `https://ares.gov.cz/ekonomicke-subjekty-v-be/rest/ekonomicke-subjekty/{ICO}`
- VIES VAT validation for EU contacts
- Contact linked to invoices (as customer) and expenses (as vendor)
- Contact tags/groups for organization
- Contact notes and communication history
- Favorite/frequent contacts for quick access
- Bank account details per contact
- Default payment terms per contact
- Automatic unreliable payer check (nespolehlivý plátce DPH)

### 4. Government Communication — VAT (Komunikace s finanční správou)

#### 4a. VAT Return (Přiznání k DPH)
- Monthly or quarterly filing (configurable)
- Automatic calculation from invoices and expenses
- Generate XML in EPO format (XSD schemas from MOJE daně portal)
- All standard rows of the DPH form (current version 25)
- Support for regular, corrective, and supplementary filings
- Preview form before XML generation
- XML validation against official XSD schema
- Deadline tracking (25th of following month)

#### 4b. VAT Control Statement (Kontrolní hlášení)
- Automatic categorization into sections A1-A5, B1-B3, C
- Threshold-based row assignment (over/under 10,000 CZK)
- Cross-reference with VAT return totals
- Generate XML in official format
- Monthly filing for all VAT payers
- Support for regular, corrective, follow-up, and supplementary filings

#### 4c. EU Sales List / VIES (Souhrnné hlášení)
- Track intra-EU B2B transactions
- Automatic detection from invoices with EU customers
- Generate XML for submission
- Monthly filing deadline (25th of following month)
- Transaction codes: goods (0), services (3), triangulation (2)

### 5. Annual Tax Filing (Daňové přiznání)

#### 5a. Income Tax Return (Přiznání k dani z příjmů FO)
- Income from self-employment (§7) — main income
- Support for additional income types:
  - Employment income (§6) — from potvrzení o příjmech
  - Capital income (§8) — dividends, interest
  - Rental income (§9)
  - Other income (§10) — investment trading (from brokerage statements)
- Expense method selection: flat-rate vs. actual expenses
- Tax deductions (nezdanitelné části základu daně):
  - Spouse deduction (sleva na manželku/manžela)
  - Mortgage interest (úroky z hypotéky) — from annual bank statement
  - Pension savings (penzijní spoření) — from annual statement
  - Life insurance (životní pojištění)
  - Union fees, donations, blood donation credits
- Tax credits (slevy na dani):
  - Basic taxpayer credit
  - Student credit
  - Disability credits
  - Child tax credit (daňové zvýhodnění na děti) — with child details
- Health and social insurance payment tracking
- Advance tax payments tracking (zálohy na daň)
- Generate XML for EPO submission
- Calculate final tax liability or overpayment

#### 5b. Social Insurance Overview (Přehled OSSZ/ČSSZ)
- Calculate assessment base from income
- Monthly advance payments tracking
- Generate XML for ČSSZ ePortal submission
- Overpayment/underpayment calculation

#### 5c. Health Insurance Overview (Přehled ZP)
- Support for major health insurance companies (VZP, ČPZP, OZP, VoZP, etc.)
- Calculate assessment base
- Monthly advance payments tracking
- Generate XML/form for submission
- Minimum assessment base enforcement

### 6. Dashboard & Reporting

#### Features
- Financial overview: revenue, expenses, profit (monthly/quarterly/yearly)
- Cash flow chart
- Outstanding invoices summary
- Upcoming tax deadlines calendar
- VAT liability overview (for VAT payers)
- Income vs. expenses comparison charts
- Top customers by revenue
- Expense breakdown by category
- Year-over-year comparison
- Export reports to PDF/CSV

---

## Recommended Additional Features

### 7. Bank Integration
- FIO Bank API: automatic transaction import and invoice matching
- Manual bank statement import (CSV/GPC format) for other banks
- Automatic payment detection by variable symbol
- Unmatched transaction review queue

### 8. Asset Register (Evidence majetku)
- Track business assets (computer, phone, furniture, etc.)
- Depreciation calculation (rovnoměrné/zrychlené odpisy)
- Depreciation groups and schedules per Czech tax law
- Auto-generate depreciation entries for tax return

### 9. Vehicle Log (Kniha jízd)
- Trip records: date, from, to, distance, purpose
- Automatic distance calculation (optional map integration)
- Monthly summaries for tax deduction
- Support for flat-rate vehicle deduction (paušál na auto — 5,000 CZK/month)

### 10. Document Archive
- Centralized storage for all business documents
- Automatic linking to invoices/expenses/contacts
- Full-text search across all documents
- Legal retention period tracking (10 years for tax documents)
- Backup and export functionality

### 11. Data Box Integration (Datová schránka)
- Send XML submissions directly via ISDS (Informační systém datových schránek)
- Receive and display messages from government
- Automatic filing deadline reminders

### 12. Multi-year History & Migration
- Import data from CSV/JSON for initial migration
- Year closing workflow
- Historical data comparison
- Archive old years while keeping them searchable

### 13. Notification System
- Invoice due date reminders
- Tax filing deadline alerts
- Overdue payment warnings
- Upcoming recurring invoice/expense notifications
- Configurable: in-app, system notification, email

### 14. Backup & Sync
- Automatic local backups of SQLite database
- Manual export/import of full database
- Optional encrypted backup to user-specified location
- Database integrity checks

---

## Architecture

### Technology Stack
- **Language:** Go or Rust (single binary, cross-platform)
- **Database:** SQLite with WAL mode (embedded, zero-config)
- **UI:** Web-based UI served from embedded assets (HTML/CSS/JS)
  - Framework options: htmx + Go templates, or SvelteKit/React embedded
  - Accessible via browser at `localhost:PORT`
- **CLI:** Full CLI interface for all operations (scriptable, pipeable)
- **PDF Generation:** Embedded engine (e.g., chromedp, wkhtmltopdf, or native)
- **OCR:** Integration with AI vision API (OpenAI, Anthropic) or local model

### Single Binary Design
```
zfaktury
├── serve          # Start web UI server
├── invoice        # Invoice CRUD and PDF generation
│   ├── create
│   ├── list
│   ├── send
│   ├── pdf
│   └── ...
├── expense        # Expense management
│   ├── add
│   ├── scan       # OCR from file
│   └── ...
├── contact        # Contact management
│   ├── add
│   ├── ares       # Fetch from ARES
│   └── ...
├── tax            # Tax filing
│   ├── vat        # DPH přiznání XML
│   ├── control    # Kontrolní hlášení XML
│   ├── vies       # Souhrnné hlášení XML
│   ├── income     # DPFO XML
│   ├── social     # ČSSZ přehled XML
│   ├── health     # ZP přehled XML
│   └── ...
├── bank           # Bank integration
│   ├── import
│   ├── match
│   └── ...
├── report         # Reports and exports
├── backup         # Backup management
└── config         # Application settings
```

### Data Storage
- Single SQLite file: `~/.zfaktury/data.db` (configurable)
- Document attachments stored in: `~/.zfaktury/documents/`
- Configuration in: `~/.zfaktury/config.toml`
- Generated XMLs in: `~/.zfaktury/exports/`

### Key Design Principles
- **Offline-first:** Everything works without internet (except ARES lookups, OCR API, bank sync)
- **Data portability:** SQLite is universally readable, easy to backup
- **Privacy:** All data stays local, no telemetry
- **Idempotent XML generation:** Same data always produces identical XML
- **Audit trail:** All changes logged with timestamps

---

## XML Schema Sources

| Filing | Schema Source |
|--------|-------------|
| DPH přiznání | https://adisspr.mfcr.cz/dpr/adis/idpr_pub/epo2_info/popis_struktury_seznam.faces |
| Kontrolní hlášení | https://financnisprava.gov.cz/cs/dane/dane/dan-z-pridane-hodnoty/kontrolni-hlaseni-dph/struktura-pro-podani-kontrolniho-hlaseni-xml |
| Souhrnné hlášení | EPO portal — XSD section |
| DPFO | EPO portal — DPFO section (yearly versioned) |
| ČSSZ přehled | https://eportal.cssz.cz/web/portal/-/tiskopisy/osvc-2024 |
| ZP přehled | Individual health insurance company portals |

---

## Implementation Phases

### Phase 1 — Foundation
- Project setup, SQLite schema, CLI framework
- User settings (ICO, DIC, VAT status, bank accounts)
- Contact management with ARES integration

### Phase 2 — Invoicing
- Invoice CRUD with all fields
- PDF generation with customizable templates
- QR payment code generation
- Email sending
- ISDOC export

### Phase 3 — Expenses
- Expense CRUD with document upload
- AI OCR integration
- Category management

### Phase 4 — VAT Filings
- DPH přiznání XML generation
- Kontrolní hlášení XML generation
- Souhrnné hlášení XML generation
- XML validation against XSD

### Phase 5 — Annual Tax
- Income tax return calculation and XML
- ČSSZ overview calculation and XML
- Health insurance overview and XML
- Deductions, credits, spouse/children support

### Phase 6 — Banking & Automation
- FIO Bank API integration
- Automatic payment matching
- Bank statement import

### Phase 7 — Web UI
- Dashboard with financial overview
- Full CRUD for all entities
- Reports and charts
- Document viewer

### Phase 8 — Polish
- Asset register and depreciation
- Vehicle log
- Notification system
- Backup management
- Data box integration (ISDS)

---

## Out of Scope (by design)
- Double-entry bookkeeping (podvojné účetnictví)
- Payroll / employee management
- Inventory / warehouse management
- E-shop integration
- Multi-user / team features
- Cloud hosting / SaaS model
