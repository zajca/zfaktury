# RFC-016: §6 Employment Income (DPC/DPP/HPP) with OCR

**Status:** Proposed
**Date:** 2026-05-02

## Summary

Adds full support for Czech §6 income (závislá činnost — DPČ, DPP, HPP employment) into the DPFO income tax return. Users upload a PDF/image of "Potvrzení o zdanitelných příjmech ze závislé činnosti" issued by their employer; an AI vision model extracts the structured fields, the user confirms, and the data flows into rows 31, 33, 34, 36, 42, 76, 84, 87 of DPFO and into the EPO XML attributes that already exist in the schema. Two Potvrzení variants are supported: zálohové (form 25 5460 vzor 33) and srážkové (form 25 5460/A vzor 12 — optional inclusion in DAP per § 36 odst. 6/7 ZDP).

## Background

The `IncomeTaxReturn` domain currently only models §7 (business), §8 (capital), and §10 (other income). The XML generator at `internal/annualtaxxml/income_tax_gen.go:210` explicitly hardcodes `// ř.42 -- assumes §6 employment base = 0`. OSVČ who have side employment (typically DPČ for occasional work) cannot file DPFO from this app — they must transcribe rows 31/33/34/84 manually elsewhere.

The DPFO schema (`docs/xml-schemas/epo/dpfdp7_epo2.xsd`) already exposes every attribute we need; this RFC fills them in instead of inventing new structure. Empty employer Potvrzení handling is the ergonomic win — most users have one or two papers per year, and OCR removes the manual transcription bottleneck (the same pattern used for §15 deductions in RFC-007 and investment docs in RFC-012).

### Verified facts (form 25 5405/1 vzor 33, year 2025)

Source: [Pokyny k vyplnění DPFO 2025, financnisprava.gov.cz](https://formulare.kurzy.cz/formulare/financni-sprava/2025/5405-1_33.pdf), §1–§7 oddíly.

| Row | Meaning | Source |
|----|---------|--------|
| ř.31 | Úhrn příjmů §6 od všech zaměstnavatelů (vč. zahraničí) | součet ř.2 + ř.4 z Potvrzení vzor 33; ř.2 z Potvrzení 25 5460/A vzor 12 (jen pokud uživatel zahrne srážkovou daň do DAP) |
| ř.32 | **Neobsazeno** v aktuálním vzoru. Superhrubá mzda zrušena od 2021 (zákon 609/2020 Sb.). | — |
| ř.33 | Daň zaplacená v zahraničí podle §6 odst. 13 (jen daň. rezident ČR) | doložené potvrzení o dani zaplacené v zahraničí |
| ř.34 | Dílčí ZD §6 = ř.31 − ř.33 | computed |
| ř.35 | Část ř.31 = příjmy bez záloh dle §38h (zahraniční zaměstnavatelé v ČR atd.) | informativní, neovlivňuje ZD |
| ř.36 | Dílčí ZD §6 (přenos ř.34) | computed |
| ř.42 | Základ daně = ř.36 + max(0, ř.41) — pokud je ř.41 záporné, jen ř.36 | computed |
| ř.76 | Vyplacené měsíční bonusy zaměstnavatelem | ř.13 Potvrzení vzor 33 |
| ř.84 | Úhrn sražených záloh §6 (po slevách na dani; po RZ snížený o vrácený přeplatek) | ř.8 Potvrzení vzor 33 |
| ř.87 | Sražená daň §36 odst. 6/7 zařazená do DAP (rezident ČR) | Potvrzení 25 5460/A vzor 12; volitelné |
| ř.87a | Sražená daň §36 odst. 7 (nerezident, daň. rezident EU/EHP) | nerelevantní pro tuzemské OSVČ |

### Verified XSD attributes (already present in `dpfdp7_epo2.xsd`)

| Element | Attribute | Row | Comment |
|---------|-----------|-----|---------|
| `VetaO` | `kc_prij6` | ř.31 | "Vyplňte údaje, které zjistíte … z Potvrzení o zdanitelných příjmech ze závislé činnosti …" |
| `VetaO` | `kc_dan_zah` | ř.33 | jen rezidenti ČR se zahraničním příjmem |
| `VetaO` | `kc_zd6` | ř.34/36 | "Přeneste údaj z ř. 34" |
| `VetaO` | `kc_prij6zahr` | ř.35 | "Část příjmů z ř. 31, u kterých neměl plátce povinnost srazit zálohy" |
| `VetaO` | `kc_zd6p` | §16a | "Vypočtená částka tvoří dílčí základ daně připadající na příjmy ze závislé činnosti" — pro samostatný základ při překročení 36× průměrné mzdy |
| `VetaO` | `kc_zakldan23` | ř.42 | XSD doc: "Pokud je ř. 41 záporný, uveďte pouze hodnotu z ř. 36" |
| `VetaD` | `kc_zalzavc` | ř.84 | "ve vzoru Potvrzení č. 33 se jedná o údaj uvedený na řádku 8" |
| `VetaD` | `kc_sraz_6_4` | ř.87 | sražená daň §36 odst. 6 (rezident ČR) |
| `VetaD` | `kc_sraz_rezehp` | ř.87a | nerezident EU/EHP |
| `VetaB` | `potv_zam` | příloha count | počet Potvrzení vzor 33 |
| `VetaB` | `potv_36` | příloha count | počet Potvrzení vzor 12 |
| `VetaB` | `potv_dazvyh` | příloha count | počet Potvrzení o vyplaceném daňovém bonusu |

### Legislative notes

- **Superhrubá mzda zrušena od 1. 1. 2021** (zákon 609/2020 Sb., § 6 odst. 12 ZDP). Pojistné odvedené zaměstnavatelem se nepřičítá ke ZD. Pole `kc_uhrn_pov` v původním návrhu této RFC bylo proto chybné.
- **Sleva na poplatníka 30 840 Kč ročně (§ 35ba odst. 1 písm. a)** se v DAP uplatňuje **vždy celá**. Pokud zaměstnavatel uplatňoval měsíčně 1/12, kompenzuje se to už ve výši sražených záloh na ř.84 — žádná korekce v `IncomeTaxReturn.CreditBasic` není potřeba.
- **§36 odst. 6 a 7 ZDP** umožňuje poplatníku zahrnout do DAP příjmy zdaněné srážkovou daní zvláštní sazbou (typicky DPP do limitu, DPČ bez prohlášení k dani). Pak musí podle § 38g odst. 6 ZDP do ř.31 zahrnout **veškeré** takové příjmy, ne jen vybrané — sledováno v UI varování.
- **Daňový bonus na děti** vyplacený zaměstnavatelem (ř.13 Potvrzení vzor 33) musí v DAP odečíst od nárokovaného ročního bonusu (ř.76 vs ř.72), jinak by se duplikoval.
- **§16a samostatný základ daně** (progresivní 23 % nad 36× průměrná mzda — pro 2025 limit 1 676 052 Kč součtu všech dílčích základů) **mimo MVP**. UI zobrazí varování, pokud `Section6Gross + §7+§8+§10 > limit`, a navrhne ruční úpravu v EPO portálu.

## Implementation

### Database (Migration 027)

```sql
-- +goose Up

-- Naskenovaná Potvrzení (PDF/JPG/PNG/WEBP)
CREATE TABLE employment_documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL,
    document_kind TEXT NOT NULL DEFAULT 'advance', -- advance | withholding | bonus
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    storage_path TEXT NOT NULL,
    size INTEGER NOT NULL DEFAULT 0,
    extraction_status TEXT NOT NULL DEFAULT 'pending', -- pending | extracted | failed
    extraction_error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX idx_employment_docs_year ON employment_documents(year);

-- Vyextrahovaný / ručně zadaný certifikát (1 plátce, 1 typ Potvrzení, 1 období)
CREATE TABLE employment_income_certificates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL,
    document_id INTEGER REFERENCES employment_documents(id) ON DELETE SET NULL,
    certificate_type TEXT NOT NULL DEFAULT 'advance', -- advance | withholding
    employer_name TEXT NOT NULL DEFAULT '',
    employer_ico TEXT NOT NULL DEFAULT '',
    employer_address TEXT NOT NULL DEFAULT '',
    contract_type TEXT NOT NULL DEFAULT 'dpc', -- dpc | dpp | hpp | other
    period_from TEXT NOT NULL,
    period_to TEXT NOT NULL,
    -- Z Potvrzení 25 5460 vzor 33 (advance)
    gross_income INTEGER NOT NULL DEFAULT 0,            -- ř.2 + ř.4 -> ř.31 DAP
    foreign_income INTEGER NOT NULL DEFAULT 0,          -- ř.35 DAP (informativní)
    foreign_tax_paid INTEGER NOT NULL DEFAULT 0,        -- §6 odst.13 -> ř.33 DAP
    advance_tax_withheld INTEGER NOT NULL DEFAULT 0,    -- ř.8 Potvrzení -> ř.84 DAP
    annual_settlement_refund INTEGER NOT NULL DEFAULT 0, -- vrácený přeplatek z RZ (snižuje ř.84)
    monthly_bonus_paid INTEGER NOT NULL DEFAULT 0,      -- ř.13 Potvrzení -> ř.76 DAP
    -- Z Potvrzení 25 5460/A vzor 12 (withholding)
    withheld_final_tax INTEGER NOT NULL DEFAULT 0,      -- §36/6/7 -> ř.87 DAP
    include_withholding_in_dap INTEGER NOT NULL DEFAULT 0, -- 1 = zahrnout do ř.31 a ř.87
    notes TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft',               -- draft | confirmed
    deleted_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (year, employer_ico, certificate_type, period_from, period_to)
        ON CONFLICT REPLACE
);
CREATE INDEX idx_employment_certs_year ON employment_income_certificates(year);

-- §6 agregáty na income_tax_returns
ALTER TABLE income_tax_returns ADD COLUMN section6_gross_income INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN section6_foreign_income INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN section6_foreign_tax INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN section6_tax_base INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN section6_advance_withheld INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN section6_withholding_credited INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN section6_monthly_bonus_paid INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN section6_certs_advance INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN section6_certs_withholding INTEGER NOT NULL DEFAULT 0;
ALTER TABLE income_tax_returns ADD COLUMN section6_certs_bonus INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE income_tax_returns DROP COLUMN section6_gross_income;
ALTER TABLE income_tax_returns DROP COLUMN section6_foreign_income;
ALTER TABLE income_tax_returns DROP COLUMN section6_foreign_tax;
ALTER TABLE income_tax_returns DROP COLUMN section6_tax_base;
ALTER TABLE income_tax_returns DROP COLUMN section6_advance_withheld;
ALTER TABLE income_tax_returns DROP COLUMN section6_withholding_credited;
ALTER TABLE income_tax_returns DROP COLUMN section6_monthly_bonus_paid;
ALTER TABLE income_tax_returns DROP COLUMN section6_certs_advance;
ALTER TABLE income_tax_returns DROP COLUMN section6_certs_withholding;
ALTER TABLE income_tax_returns DROP COLUMN section6_certs_bonus;
DROP TABLE IF EXISTS employment_income_certificates;
DROP TABLE IF EXISTS employment_documents;
```

### Domain Types

`internal/domain/employment_income.go`:

```go
package domain

import "time"

type EmploymentDocumentKind string

const (
    EmploymentDocAdvance     EmploymentDocumentKind = "advance"
    EmploymentDocWithholding EmploymentDocumentKind = "withholding"
    EmploymentDocBonus       EmploymentDocumentKind = "bonus"
)

type CertificateType string

const (
    CertificateAdvance     CertificateType = "advance"
    CertificateWithholding CertificateType = "withholding"
)

type ContractType string

const (
    ContractDPC   ContractType = "dpc"
    ContractDPP   ContractType = "dpp"
    ContractHPP   ContractType = "hpp"
    ContractOther ContractType = "other"
)

type EmploymentDocument struct {
    ID               int64
    Year             int
    Kind             EmploymentDocumentKind
    Filename         string
    ContentType      string
    StoragePath      string
    Size             int64
    ExtractionStatus string
    ExtractionError  string
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

type EmploymentCertificate struct {
    ID                       int64
    Year                     int
    DocumentID               *int64
    CertificateType          CertificateType
    EmployerName             string
    EmployerICO              string
    EmployerAddress          string
    ContractType             ContractType
    PeriodFrom               time.Time
    PeriodTo                 time.Time
    GrossIncome              Amount
    ForeignIncome            Amount
    ForeignTaxPaid           Amount
    AdvanceTaxWithheld       Amount
    AnnualSettlementRefund   Amount
    MonthlyBonusPaid         Amount
    WithheldFinalTax         Amount
    IncludeWithholdingInDAP  bool
    Notes                    string
    Status                   string
    DeletedAt                *time.Time
    CreatedAt                time.Time
    UpdatedAt                time.Time
}
```

Extend `IncomeTaxReturn` in `internal/domain/annual_tax.go`:

```go
// §6 employment income aggregates (DPC/DPP/HPP)
Section6GrossIncome         Amount  // ř.31
Section6ForeignIncome       Amount  // ř.35 (informativní)
Section6ForeignTax          Amount  // ř.33
Section6TaxBase             Amount  // ř.34/36 = ř.31 - ř.33
Section6AdvanceWithheld     Amount  // ř.84 (po vrácení přeplatku z RZ)
Section6WithholdingCredited Amount  // ř.87 (jen pokud uživatel zahrnul §36/6 do DAP)
Section6MonthlyBonusPaid    Amount  // ř.76 (vyplacené zaměstnavatelem)
Section6CertsAdvance        int     // count -> potv_zam
Section6CertsWithholding    int     // count -> potv_36
Section6CertsBonus          int     // count -> potv_dazvyh
```

### Repository

| File | Purpose |
|------|---------|
| `internal/repository/employment_document_repo.go` | CRUD analogicky `investment_document_repo.go`, scan helper `scanEmploymentDocument` |
| `internal/repository/employment_certificate_repo.go` | CRUD + `ListByYear`, soft delete via `deleted_at`, scan helper `scanEmploymentCertificate` |

Add interfaces to `internal/repository/interfaces.go` (lead merge):

```go
type EmploymentDocumentRepo interface {
    Create(ctx context.Context, doc *EmploymentDocument) error
    GetByID(ctx context.Context, id int64) (*EmploymentDocument, error)
    ListByYear(ctx context.Context, year int) ([]*EmploymentDocument, error)
    Delete(ctx context.Context, id int64) error
    UpdateExtraction(ctx context.Context, id int64, status, errMsg string) error
}

type EmploymentCertificateRepo interface {
    Create(ctx context.Context, cert *EmploymentCertificate) error
    GetByID(ctx context.Context, id int64) (*EmploymentCertificate, error)
    Update(ctx context.Context, cert *EmploymentCertificate) error
    Delete(ctx context.Context, id int64) error
    ListByYear(ctx context.Context, year int) ([]*EmploymentCertificate, error)
    ListConfirmedByYear(ctx context.Context, year int) ([]*EmploymentCertificate, error)
}
```

### OCR

`internal/service/ocr/employment_prompt.go` — Czech system prompt instructing the model to:

1. Identify form variant by header text:
   - "25 5460 MFin 5460 - vzor č. 33" or "Potvrzení o zdanitelných příjmech ze závislé činnosti" → `certificate_type: advance`
   - "25 5460/A MFin 5460/A - vzor č. 12" or "Potvrzení o vyplacených příjmech … srážkou" → `certificate_type: withholding`
2. Extract employer block (název, IČO, adresa) and zdaňovací období.
3. Detect contract type from textual hints:
   - "Dohoda o pracovní činnosti" / "DPČ" → `dpc`
   - "Dohoda o provedení práce" / "DPP" → `dpp`
   - "Pracovní poměr" / "HPP" → `hpp`
4. Extract amounts row-by-row per Potvrzení layout:
   - Vzor 33: ř.2 (úhrn zúčtovaných příjmů), ř.4 (další příjmy), ř.8 (sražené zálohy po slevách), ř.13 (vyplacené měsíční bonusy), případná položka "vrácený přeplatek z ročního zúčtování"
   - Vzor 12: ř.2 (úhrn vyplacených příjmů), položka "sražená daň zvláštní sazbou" (typicky ř.4 nebo 5)
5. Output `confidence` per [0.0–1.0] and `raw_text` (max 2000 znaků) for audit.

JSON response (`EmploymentExtractionResponse`):

```json
{
  "certificate_type": "advance|withholding",
  "employer_name": "...",
  "employer_ico": "...",
  "employer_address": "...",
  "contract_type": "dpc|dpp|hpp|other",
  "period_from": "YYYY-MM-DD",
  "period_to": "YYYY-MM-DD",
  "gross_income_czk": 0.0,
  "foreign_income_czk": 0.0,
  "foreign_tax_paid_czk": 0.0,
  "advance_tax_withheld_czk": 0.0,
  "annual_settlement_refund_czk": 0.0,
  "monthly_bonus_paid_czk": 0.0,
  "withheld_final_tax_czk": 0.0,
  "confidence": 0.0,
  "raw_text": "..."
}
```

Tests: `employment_prompt_test.go` covering both variants, missing fields → 0, malformed JSON, OCR confidence threshold.

### Service

`internal/service/employment_certificate_svc.go`:

```go
type EmploymentCertificateService struct {
    docs    repository.EmploymentDocumentRepo
    certs   repository.EmploymentCertificateRepo
    ocr     ocr.Provider
    audit   AuditLogger
    dataDir string
}

func (s *EmploymentCertificateService) UploadDocument(ctx, year, kind, filename, contentType, content) (*EmploymentDocument, error)
func (s *EmploymentCertificateService) ExtractDocument(ctx, docID) (*EmploymentCertificate, error)
func (s *EmploymentCertificateService) Create(ctx, cert) error
func (s *EmploymentCertificateService) Update(ctx, cert) error
func (s *EmploymentCertificateService) Confirm(ctx, certID) error
func (s *EmploymentCertificateService) ListByYear(ctx, year) ([]*EmploymentCertificate, error)
func (s *EmploymentCertificateService) Delete(ctx, certID) error
```

Validation rules:
- `period_from <= period_to`, both within `year`
- `gross_income >= 0`, all amounts non-negative
- `withheld_final_tax > 0` only if `certificate_type = withholding`
- `include_withholding_in_dap = true` only if `certificate_type = withholding`
- `annual_settlement_refund <= advance_tax_withheld` (cannot refund more than withheld)
- IČO format: 8 digits, validated via existing `domain.ValidateICO`

Storage: `DataDir/employment_docs/{year}/{uuid}_{filename}` with content-type whitelist `application/pdf`, `image/jpeg`, `image/png`, `image/webp` and 10 MB max size.

Audit categories: `employment_document`, `employment_certificate` — added to `audit_log_handler.go:55`.

### Tax Calculation

Extend `internal/service/income_tax_return_svc.go` `Recalculate`:

```go
// New: load §6 certificates and aggregate
certs, err := s.employmentCerts.ListConfirmedByYear(ctx, year)
if err != nil { return fmt.Errorf("listing employment certificates: %w", err) }
itr.Section6CertsAdvance = 0
itr.Section6CertsWithholding = 0
itr.Section6CertsBonus = 0
itr.Section6GrossIncome = 0
itr.Section6ForeignIncome = 0
itr.Section6ForeignTax = 0
itr.Section6AdvanceWithheld = 0
itr.Section6WithholdingCredited = 0
itr.Section6MonthlyBonusPaid = 0
for _, c := range certs {
    switch c.CertificateType {
    case domain.CertificateAdvance:
        itr.Section6GrossIncome += c.GrossIncome
        itr.Section6ForeignIncome += c.ForeignIncome
        itr.Section6ForeignTax += c.ForeignTaxPaid
        itr.Section6AdvanceWithheld += c.AdvanceTaxWithheld - c.AnnualSettlementRefund
        itr.Section6MonthlyBonusPaid += c.MonthlyBonusPaid
        itr.Section6CertsAdvance++
        if c.MonthlyBonusPaid > 0 { itr.Section6CertsBonus++ }
    case domain.CertificateWithholding:
        if c.IncludeWithholdingInDAP {
            itr.Section6GrossIncome += c.GrossIncome
            itr.Section6WithholdingCredited += c.WithheldFinalTax
            itr.Section6CertsWithholding++
        }
    }
}
itr.Section6TaxBase = itr.Section6GrossIncome - itr.Section6ForeignTax  // ř.34

// Existing §7 + new §6 in tax base calculation
totalBase := itr.Section6TaxBase
positiveSum := zd7 + zd8 + zd10  // §7+§8+§10
if positiveSum > 0 { totalBase += positiveSum }
// Use totalBase as the new TaxBase input to progressive 15/23 calc

// Child benefit: subtract bonuses already paid by employer (ř.76)
if itr.Section6MonthlyBonusPaid > 0 {
    itr.ChildBenefit -= itr.Section6MonthlyBonusPaid
    if itr.ChildBenefit < 0 { itr.ChildBenefit = 0 }
}

// §16a guard (out of scope for MVP)
limit := taxConstants.AvgWageMultiplier36x  // 1 676 052 CZK pro 2025
if itr.Section6GrossIncome+positiveSum > limit {
    // Surface warning to UI but emit XML without §16a split
    itr.Warnings = append(itr.Warnings, domain.WarningSection16aManual)
}
```

`CreditBasic` stays at full annual amount (30 840 Kč 2025) regardless of monthly application by employer — the difference flows back through ř.84 vs the recalculated total tax.

### XML Generator

`internal/annualtaxxml/income_tax_types.go` — extend `DPFOVetaO`, `DPFOVetaD`, `DPFOVetaB`:

```go
type DPFOVetaO struct {
    KcPrij6      int64 `xml:"kc_prij6,attr,omitempty"`      // ř.31
    KcPrij6zahr  int64 `xml:"kc_prij6zahr,attr,omitempty"`  // ř.35
    KcDanZah     int64 `xml:"kc_dan_zah,attr,omitempty"`    // ř.33
    KcZd6        int64 `xml:"kc_zd6,attr,omitempty"`        // ř.34/36
    KcZd6p       int64 `xml:"kc_zd6p,attr,omitempty"`       // §16a část §6 (MVP: 0)
    KcZd7        int64 `xml:"kc_zd7,attr"`
    KcZakldan8   int64 `xml:"kc_zakldan8,attr,omitempty"`
    KcZd9        int64 `xml:"kc_zd9,attr,omitempty"`
    KcZd10       int64 `xml:"kc_zd10,attr,omitempty"`
    KcUhrn       int64 `xml:"kc_uhrn,attr"`                 // ř.41 = ř.37+38+39+40
    KcZakldan23  int64 `xml:"kc_zakldan23,attr"`            // ř.42 = ř.36 + max(0, ř.41)
    KcZakldan    int64 `xml:"kc_zakldan,attr"`              // ř.45
}

type DPFOVetaD struct {
    // existing fields...
    KcZalzavc    int64 `xml:"kc_zalzavc,attr,omitempty"`    // ř.84
    KcSraz64     int64 `xml:"kc_sraz_6_4,attr,omitempty"`   // ř.87
    KcSrazRezEHP int64 `xml:"kc_sraz_rezehp,attr,omitempty"` // ř.87a (MVP: 0)
}

type DPFOVetaB struct {
    Priloha1   string `xml:"priloha1,attr,omitempty"`
    Priloha2   string `xml:"priloha2,attr,omitempty"`
    PotvZam    int    `xml:"potv_zam,attr,omitempty"`
    Potv36     int    `xml:"potv_36,attr,omitempty"`
    PotvDazvyh int    `xml:"potv_dazvyh,attr,omitempty"`
}
```

`income_tax_gen.go` — replace lines 209-211:

```go
// §6 inputs
prij6 := ToWholeCZK(itr.Section6GrossIncome)
prij6zahr := ToWholeCZK(itr.Section6ForeignIncome)
danZah := ToWholeCZK(itr.Section6ForeignTax)
zd6 := ToWholeCZK(itr.Section6TaxBase)  // = prij6 - danZah

// Existing §7-§10
zd7 := ...
zd8 := ToWholeCZK(itr.CapitalIncomeNet)
zd10 := ToWholeCZK(itr.OtherIncomeNet)
uhrn := zd7 + zd8 + zd10  // ř.41

// XSD critical: "Pokud je ř.41 záporný, uveďte pouze hodnotu z ř.36"
zakldan23 := zd6
if uhrn > 0 {
    zakldan23 += uhrn
}
```

Set `VetaO.KcPrij6/KcPrij6zahr/KcDanZah/KcZd6`, `VetaD.KcZalzavc/KcSraz64`, `VetaB.PotvZam/Potv36/PotvDazvyh`.

### HTTP Handler

`internal/handler/employment_handler.go` — new endpoints:

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/api/v1/tax/employment/documents?year=&kind=` | multipart upload (max 10 MB; MIME allowlist) |
| POST | `/api/v1/tax/employment/documents/{id}/extract` | run OCR, return draft certificate |
| GET | `/api/v1/tax/employment/documents?year=` | list |
| DELETE | `/api/v1/tax/employment/documents/{id}` | delete file + DB row (cascade SET NULL on certs) |
| GET | `/api/v1/tax/employment/certificates?year=` | list |
| GET | `/api/v1/tax/employment/certificates/{id}` | detail |
| POST | `/api/v1/tax/employment/certificates` | create (manual) |
| PUT | `/api/v1/tax/employment/certificates/{id}` | update draft |
| POST | `/api/v1/tax/employment/certificates/{id}/confirm` | confirm + trigger ITR recompute if exists |
| DELETE | `/api/v1/tax/employment/certificates/{id}` | soft delete |

DTOs in `helpers.go` (lead merges). Mount in `router.go` with `r.Route("/tax/employment", ...)`. Wire `EmploymentDocumentRepo`, `EmploymentCertificateRepo`, `EmploymentCertificateService` in `serve.go`.

### Frontend

#### New page `frontend/src/routes/tax/employment/+page.svelte`

- Year selector (mirroring `tax/+page.svelte`)
- Two upload tiles: "Nahrát zálohové Potvrzení (vzor 33)", "Nahrát srážkové Potvrzení (vzor 12)" — file input → POST upload → POST extract → open editor with extracted draft + confidence badge
- "Zadat ručně" button — empty editor
- Table of certificates with employer, period, type, gross, ř.84, ř.87, status, action menu
- After confirm/edit: auto-call `incomeTaxApi.recompute(returnId)` if return exists for year

#### Editor (modal or full-page)

Sections:
1. Identifikace plátce: name (required), IČO (8 digits via `validateICO`), address
2. Období: `period_from`, `period_to` (within `year`), contract type (DPČ/DPP/HPP/jiné), `notes`
3. Pro `certificate_type=advance` (vzor 33):
   - Úhrn zúčtovaných příjmů (ř.2 + ř.4) → `gross_income`
   - Z toho zahraniční bez záloh (ř.35 DAP) → `foreign_income`
   - Daň zaplacená v zahraničí (§6 odst.13) → `foreign_tax_paid`
   - Sražené zálohy po slevách (ř.8 Potvrzení) → `advance_tax_withheld`
   - Vrácený přeplatek z ročního zúčtování → `annual_settlement_refund`
   - Vyplacené měsíční bonusy (ř.13 Potvrzení) → `monthly_bonus_paid`
4. Pro `certificate_type=withholding` (vzor 12):
   - Úhrn vyplacených příjmů (ř.2 Potvrzení) → `gross_income`
   - Sražená daň zvláštní sazbou → `withheld_final_tax`
   - Checkbox "Zahrnout do daňového přiznání (§36 odst.7 ZDP)" → `include_withholding_in_dap` + warning "Pokud zaškrtnete, musíte zahrnout veškeré srážkově zdaněné příjmy z daného typu (§38g odst.6)"

#### Card on `tax/+page.svelte`

Add 4th card (after DPFO/CSSZ/ZP) "Závislá činnost (§6)":
- count of certificates (advance + withholding combined)
- ř.31 `Section6GrossIncome`, ř.84 `Section6AdvanceWithheld`, ř.87 `Section6WithholdingCredited`
- "Spravovat" → `/tax/employment`

#### Section on `tax/income/[id]/+page.svelte`

Read-only "§6 závislá činnost" panel above existing §7 panel:
- ř.31, ř.33, ř.34/36, ř.84, ř.87, ř.76 with HelpTip on each
- "Upravit certifikáty" link to `/tax/employment?year={year}`

#### API Client

Extend `frontend/src/lib/api/client.ts` (lead merges):

```typescript
export interface EmploymentDocument { ... }
export interface EmploymentCertificate { ... }
export interface EmploymentExtractionResult { /* OCR response */ }
export const employmentApi = {
  uploadDocument(year, kind, file): Promise<EmploymentDocument>,
  extractDocument(id): Promise<EmploymentCertificate>,
  listDocuments(year): Promise<EmploymentDocument[]>,
  deleteDocument(id): Promise<void>,
  listCertificates(year): Promise<EmploymentCertificate[]>,
  getCertificate(id): Promise<EmploymentCertificate>,
  createCertificate(cert): Promise<EmploymentCertificate>,
  updateCertificate(id, cert): Promise<EmploymentCertificate>,
  confirmCertificate(id): Promise<void>,
  deleteCertificate(id): Promise<void>,
};
```

### Help Content

Extend `HelpTopicId` union and topics in `frontend/src/lib/data/help-content.ts`:

| ID | Title | Where used |
|----|-------|-----------|
| `zavisla-cinnost-s6` | Závislá činnost (§6) | header on `/tax/employment`, card on `/tax`, section on income return detail |
| `dpc-dpp-hpp` | Typy pracovních smluv | contract type selector in editor |
| `potvrzeni-zalohove` | Potvrzení o zdanitelných příjmech (vzor 33) | upload tile, gross/withheld fields |
| `potvrzeni-srazkove` | Potvrzení o vyplacených příjmech a sražené dani (vzor 12) | upload tile, withholding fields |
| `srazkova-do-dap` | Zahrnutí srážkové daně do přiznání | `include_withholding_in_dap` checkbox |
| `radek-31-prijmy-s6` | ř.31 Úhrn příjmů §6 | display on income return detail |
| `radek-33-zahranicni-dan` | ř.33 Daň zaplacená v zahraničí | foreign tax field |
| `radek-34-dilci-zaklad-s6` | ř.34/36 Dílčí základ daně §6 | computed display |
| `radek-84-srazene-zalohy` | ř.84 Sražené zálohy zaměstnavateli | display on income return detail |
| `radek-87-srazena-dan` | ř.87 Sražená daň §36 odst.6 | display when withholding used |
| `radek-76-mesicni-bonus` | ř.76 Vyplacený měsíční daňový bonus | display when bonus paid |
| `rocni-zuctovani-rz` | Roční zúčtování (přeplatek/nedoplatek) | annual_settlement_refund field |
| `superhruba-mzda-zrusena` | Proč není pole na povinné pojistné | shown if user asks "kde je ř.32" |
| `samostatny-zaklad-16a` | Samostatný základ daně §16a | warning when limit exceeded |

Each topic has `simple` (for OSVČ without tax background) and `legal` (citing § ZDP and pokyny). Sample structure:

```typescript
'zavisla-cinnost-s6': {
  title: 'Závislá činnost (§6)',
  simple:
    'Zde nahrajte Potvrzení o zdanitelných příjmech, které vám vystavil zaměstnavatel za DPČ, DPP nebo hlavní pracovní poměr. Aplikace z něj vyextrahuje údaje a propíše je do řádků 31, 33, 34, 84 a 87 vašeho daňového přiznání.\n\nKaždé Potvrzení od jiného zaměstnavatele uložte zvlášť. Aplikace pozná dvě varianty -- "zálohové" (formulář 25 5460 vzor 33) a "srážkové" (25 5460/A vzor 12). U srážkového se rozhodnete, jestli ho chcete zahrnout do přiznání nebo ne.',
  legal:
    'Příjmy ze závislé činnosti definuje § 6 zákona č. 586/1992 Sb. o daních z příjmů. Zaměstnavatel je povinen vystavit Potvrzení do 1. března následujícího roku (§ 38j odst. 3 ZDP).\n\nDílčí základ daně podle § 6 = úhrn příjmů snížený o daň zaplacenou v zahraničí (§ 6 odst. 13). Od 1. 1. 2021 se nepřičítá pojistné odvedené zaměstnavatelem (zrušení superhrubé mzdy zákonem 609/2020 Sb.).'
},
'potvrzeni-zalohove': {
  title: 'Potvrzení o zdanitelných příjmech (vzor 33)',
  simple:
    'Tento formulář dostáváte od zaměstnavatele, pokud vám sráží zálohy na daň (typicky DPČ s podepsaným prohlášením, HPP). Najdete na něm úhrn vašich příjmů (ř. 2 + 4), sražené zálohy po slevách (ř. 8) a vyplacené daňové bonusy (ř. 13).\n\nDo aplikace stačí nahrát PDF nebo fotku -- AI to přečte za vás. Vždycky si ale zkontrolujte vyextrahované hodnoty, OCR může udělat chybu.',
  legal:
    'Formulář MFin 25 5460 vzor č. 33 vydává Ministerstvo financí ČR pro zdaňovací období 2025 podle § 38j odst. 3 ZDP. Údaje z něj se přenášejí do oddílu 1 Přiznání k DPFO (řádky 31, 33, 34, 36) a oddílu 7 (řádek 84). Roční zúčtování provádí zaměstnavatel podle § 38ch -- pokud bylo provedeno, snížíte ř. 84 o vrácený přeplatek.'
},
// ...etc.
```

### Audit Log

Extend `internal/handler/audit_log_handler.go:55`:

```go
"document": true, "tax_deduction_document": true, "investment_document": true,
"employment_document": true, "employment_certificate": true,
```

### Tests

| Test | What it covers |
|------|----------------|
| `repository/employment_document_repo_test.go` | CRUD, ListByYear, UpdateExtraction |
| `repository/employment_certificate_repo_test.go` | CRUD, soft delete, ListByYear, ListConfirmedByYear, UNIQUE clause REPLACE |
| `service/ocr/employment_prompt_test.go` | parse JSON for advance + withholding, malformed input, missing fields → 0, code-fence stripping |
| `service/employment_certificate_svc_test.go` | upload + MIME guard, extract via mock OCR provider, validation rules, audit emits, RZ refund subtraction |
| `service/income_tax_return_svc_test.go` (extension) | recompute aggregates §6, monthly bonus subtraction from child benefit, §16a limit warning |
| `annualtaxxml/income_tax_gen_test.go` (extensions) | `TestIncomeTaxXML_Section6Advance` (2 advance certs → kc_prij6/kc_zd6/kc_zalzavc/potv_zam=2), `TestIncomeTaxXML_Section6Withholding` (1 withholding included → kc_sraz_6_4/potv_36=1), `TestIncomeTaxXML_Section6OnlyNegativeSection7` (§6 + §7 ztráta → kc_zakldan23 = kc_zd6, no positive uhrn applied) |
| `handler/employment_handler_test.go` | multipart upload happy path, MIME rejection, oversize rejection, extract endpoint, confirm triggers recompute |
| `routes/tax/employment/page.test.ts` (Vitest) | upload flow with mocked employmentApi, OCR confidence rendering, advance vs withholding form switching, IČO validation, period range guard |
| `routes/tax/income/[id]/page.test.ts` (extension) | §6 panel renders only when `Section6GrossIncome > 0` |
| `tests/integration/employment_flow_test.go` | upload PDF testdata → mock OCR returns vzor 33 JSON → confirm 2 certs → generate DPFO XML → assert `kc_prij6=240000`, `kc_zalzavc=36000`, `potv_zam=2` |

## Migration Plan

1. Migration 027 ships dormant — empty tables, agregát columns default 0. Existing DPFO returns regenerate XML identically (new attrs use `omitempty`).
2. UI tile gates feature behind OCR provider config — if no `[ocr]` section in `config.toml`, show "Zadat ručně" button only and link to docs.
3. No data migration. Users with existing DPFO drafts re-open them and § 6 fields are 0 — they upload Potvrzení and re-confirm.

## Out of Scope

- §16a samostatný základ daně 23 % (limit 36× průměrná mzda) — emits warning, user must adjust in EPO portal.
- ř.87a (nerezident EU/EHP) — most users are tuzemští rezidenti; field exists in struct but UI does not expose it.
- Automatic linking of `Potvrzení o vyplaceném daňovém bonusu` (separate `bonus` document kind) to children rows in VetaA — out of scope; user updates child rows manually if needed.
- §38g odst. 6 enforcement — UI shows warning for `include_withholding_in_dap`, but does not block partial inclusion.
- Attaching the scanned PDF to the EPO submission XML — EPO accepts XML attachment via separate upload step in the portal; we generate only the DAP XML.

## Open Questions

1. **UNIQUE conflict on (year, employer_ico, type, period_from, period_to):** corrective re-issued Potvrzení currently overwrites via `ON CONFLICT REPLACE`. Alternative: append numeric suffix to `notes` and keep both. Decision: REPLACE (simpler, matches typical user intent of "got new corrected version").
2. **DPP threshold tracking:** for 2025 the DPP without prohlášení threshold is 11 500 Kč/month (rozhodný příjem pro účast na nemocenském). Should the editor warn when user marks `dpp` and gross/months > limit suggests withholding tax, hinting "ověřte typ Potvrzení"? Decision: yes, advisory warning; do not block.
3. **OCR vendor neutrality:** prompt is Czech-language and tested against `claude` + `openai` providers. Other vendors (`gemini`, `mistral`) have not been validated for this form layout — surface vendor in extraction error if confidence < 0.5.

## References

- Pokyny k vyplnění DPFO 2025, vzor č. 33 (financnisprava.gov.cz): https://formulare.kurzy.cz/formulare/financni-sprava/2025/5405-1_33.pdf
- Vyhláška 386/2025 Sb. o formulářových podáních pro daně z příjmů: https://www.zakonyprolidi.cz/cs/2025-386
- Aktuálně k DPFO 2025, Finanční správa: https://financnisprava.gov.cz/cs/dane/dane/dan-z-prijmu/dotazy-a-odpovedi/dan-z-prijmu-fyzickych-osob/aktualne-k-dani-z-prijmu-fyzickych-osob-2025
- Zákon č. 586/1992 Sb. o daních z příjmů, § 6, § 16, § 16a, § 35ba, § 36, § 38g, § 38h, § 38j
- Zákon č. 609/2020 Sb. (zrušení superhrubé mzdy od 1. 1. 2021)
- XSD: `docs/xml-schemas/epo/dpfdp7_epo2.xsd` (lokálně)
- Související RFC: 006-annual-tax (DPFO base), 007-tax-credits-deductions (§15 OCR), 012-calc-extraction (calc helpers), 015-pdf-templates
