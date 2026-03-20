# Phase 3: Document Generation

**Crate:** `zfaktury-gen`
**Estimated duration:** 2 weeks
**Depends on:** Phase 1 (domain types, `Amount`), Phase 2 (repository layer for loading data)

## Objectives

Port all document generation from the Go codebase to Rust with identical output:

- Invoice PDF generation via typst templates (replacing maroto)
- VAT return XML (DPHDP3), Control Statement XML (DPHKH1), VIES XML (DPHSHV)
- Income tax XML (DPFDP5), Social insurance XML (CSSZ/OSVC), Health insurance XML
- ISDOC 6.0.2 XML generation
- SPAYD QR code generation for invoice payments
- CSV export for invoices and expenses
- 100% E2E test coverage with golden files and XSD validation

## Crate Layout

```
zfaktury-gen/
  Cargo.toml
  src/
    lib.rs
    pdf/
      mod.rs
      invoice.rs          # PDF generation entry point
      template.typ        # Typst template (embedded at compile time)
      types.rs            # SupplierInfo, PDFSettings
    xml/
      mod.rs
      common.rs           # Shared helpers: to_whole_czk, filing type codes
      vat_return.rs        # DPHDP3 generator
      control_statement.rs # DPHKH1 generator
      vies.rs              # DPHSHV generator
      income_tax.rs        # DPFDP5 generator
      social_insurance.rs  # CSSZ/OSVC generator
      health_insurance.rs  # Health insurance generator
    isdoc/
      mod.rs
      generator.rs         # ISDOC 6.0.2 generator
      types.rs             # ISDOC XML structs
    qr/
      mod.rs
      spayd.rs             # SPAYD string + QR PNG
    csv/
      mod.rs
      invoices.rs          # Invoice CSV export
      expenses.rs          # Expense CSV export
  tests/
    golden/
      pdf/
      xml/
      isdoc/
      csv/
    schemas/               # XSD files for validation
    pdf_test.rs
    xml_test.rs
    isdoc_test.rs
    qr_test.rs
    csv_test.rs
```

## Dependencies

```toml
[dependencies]
typst = "0.13"              # PDF generation via typst as a library
typst-pdf = "0.13"          # PDF rendering backend
quick-xml = { version = "0.37", features = ["serialize"] }
serde = { version = "1", features = ["derive"] }
qrcode = "0.14"             # QR code PNG generation
image = { version = "0.25", default-features = false, features = ["png"] }
csv = "1"
thiserror = "2"

# From workspace (Phase 1)
zfaktury-domain = { path = "../zfaktury-domain" }

[dev-dependencies]
similar = "2"               # Readable golden file diffs
roxmltree = "0.20"          # XML parsing for XSD-like validation in tests
pdf-extract = "0.8"         # PDF text extraction for verification
insta = "1"                 # Snapshot testing (optional, complements golden files)
```

## Module Specifications

### 1. PDF Invoice Generation (`src/pdf/`)

#### `src/pdf/types.rs`

```rust
use zfaktury_domain::Amount;

/// Supplier (OSVC) details loaded from application settings.
/// Mirrors the Go `pdf.SupplierInfo` struct.
pub struct SupplierInfo {
    pub name: String,
    pub ico: String,
    pub dic: String,
    pub vat_registered: bool,
    pub street: String,
    pub city: String,
    pub zip: String,
    pub email: String,
    pub phone: String,
    pub bank_account: String,
    pub bank_code: String,
    pub iban: String,
    pub swift: String,
    pub logo_path: Option<String>,
}

/// PDF customization settings. Maps to Go `pdf.PDFSettings`.
pub struct PdfSettings {
    pub logo_path: Option<String>,
    pub accent_color: String,      // hex "#2563eb"
    pub footer_text: String,
    pub show_qr: bool,
    pub show_bank_details: bool,
    pub font_size: FontSize,
}

pub enum FontSize {
    Small,  // 9pt
    Normal, // 10pt (default)
    Large,  // 11pt
}

impl Default for PdfSettings {
    fn default() -> Self {
        Self {
            logo_path: None,
            accent_color: "#2563eb".to_string(),
            footer_text: String::new(),
            show_qr: true,
            show_bank_details: true,
            font_size: FontSize::Normal,
        }
    }
}
```

#### `src/pdf/invoice.rs`

```rust
use zfaktury_domain::Invoice;
use crate::pdf::types::{SupplierInfo, PdfSettings};
use crate::qr::spayd::generate_qr_png;

/// Generate a PDF invoice as raw bytes.
///
/// Uses typst as a library: builds a typst document from the embedded template,
/// injects invoice data as typst variables, and renders to PDF.
pub fn generate_invoice_pdf(
    invoice: &Invoice,
    supplier: &SupplierInfo,
    settings: &PdfSettings,
) -> Result<Vec<u8>, GenError> { ... }
```

**Template structure (A4 page):**

1. **Header:** Optional logo (PNG/JPG), invoice number, type label (Faktura/Dobropis/Proforma faktura), status label
2. **Dates row:** Datum vystaveni (DD.MM.YYYY), Datum splatnosti, DUZP
3. **Two-column parties:** Dodavatel (left) | Odberatel (right) -- name, address, ICO, DIC
4. **Items table:** #, Popis, Mn., Jedn., Cena/ks, DPH %, DPH, Celkem
5. **VAT summary by rate:** Sazba DPH, Zaklad, DPH (grouped by 21%, 12%)
6. **Totals:** Zaklad celkem, DPH celkem, Celkem k uhrade (bold, with CZK suffix)
7. **Payment section:** Cislo uctu, IBAN, Variabilni symbol, Konstantni symbol, Datum splatnosti, QR platba (embedded PNG)
8. **Footer:** "Subjekt neni platce DPH." if not VAT registered + custom footer text

**Font handling:** The Go version embeds Liberation Sans TTF files. The Rust version embeds fonts via `include_bytes!` and passes them to typst. Use Liberation Sans (or Inter if available as OTF) for text and a monospace font for amounts.

**Currency format:** Czech locale -- `1 234,56 CZK` (non-breaking space as thousands separator, comma as decimal). Reuse `Amount::display_czech()` from the domain crate.

**Logo support:** PNG and JPG only (same as Go). Read from `logo_path`, embed as typst image. Silently skip if file missing or unsupported format.

**QR code:** When `settings.show_qr` is true and IBAN is available, call `generate_qr_png()` from the `qr` module and embed the resulting PNG in the PDF.

### 2. VAT XML Generation (`src/xml/`)

#### `src/xml/common.rs`

Shared helpers ported from Go `vatxml/common.go` and `annualtaxxml/common.go`:

```rust
use zfaktury_domain::Amount;

/// Convert halere to whole CZK with standard math rounding.
/// Used for VAT returns and control statements.
/// Go equivalent: vatxml.ToWholeCZK (math.Round)
pub fn to_whole_czk_rounded(a: Amount) -> i64 {
    (a.as_i64() as f64 / 100.0).round() as i64
}

/// Convert halere to whole CZK with truncation toward zero.
/// Used for income tax and insurance calculations.
/// Go equivalent: annualtaxxml.ToWholeCZK (integer division)
pub fn to_whole_czk_truncated(a: Amount) -> i64 {
    a.as_i64() / 100
}

/// DPHDP3 filing type codes.
/// regular -> "B" (radne), corrective -> "O" (opravne), supplementary -> "D" (dodatecne)
pub fn dph_filing_type_code(filing_type: &str) -> &'static str { ... }

/// DPHKH1 control statement filing type codes.
/// regular -> "R", corrective -> "N", supplementary -> "O"
pub fn kh_filing_type_code(filing_type: &str) -> &'static str { ... }

/// VIES filing type codes.
/// regular -> "B", corrective -> "O", supplementary -> "N"
pub fn vies_filing_type_code(filing_type: &str) -> &'static str { ... }

/// DPFDP5 income tax filing type codes.
/// B = regular, O = corrective, D = supplementary
pub fn dpfo_filing_type_code(filing_type: &str) -> &'static str { ... }

/// CSSZ social insurance filing type codes.
/// N = regular (nova), O = corrective (opravna), Z = supplementary (zmena)
pub fn cssz_filing_type_code(filing_type: &str) -> &'static str { ... }
```

**Critical difference:** VAT XML uses `math.Round` for halere-to-CZK conversion, while income tax/insurance uses integer truncation. The Rust port must preserve this distinction exactly.

#### `src/xml/vat_return.rs` -- DPHDP3

```rust
use zfaktury_domain::{VATReturn, TaxpayerInfo};

/// Generate DPHDP3 VAT return XML.
/// Output matches the Czech EPO format: XSD adisspr.mfcr.cz/adis/jepo/schema/dphdp3_epo2.xsd
pub fn generate_vat_return_xml(
    vr: &VATReturn,
    info: &TaxpayerInfo,
) -> Result<Vec<u8>, GenError> { ... }
```

**XML structure (exact attribute names from Go):**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Pisemnost nazevSW="ZFaktury">
  <DPHDP3 verzePis="01.02.16">
    <VetaD dokument="DP3" k_uladis="DPH" dapdph_forma="{B|O|D}"
           typ_platce="P" trans="{A|N}" c_okec="{okec}"
           d_poddp="{DD.MM.YYYY}" rok="{year}" mesic="{month}" ctvrt="{quarter}"/>
    <VetaP c_pracufo="{prac_ufo}" c_ufo="{ufo_code}" dic="{dic}"
           email="{email}" c_telef="{phone}" ulice="{street}"
           naz_obce="{city}" psc="{zip}" stat="CESKA REPUBLIKA"
           c_pop="{house_num}" jmeno="{first_name}" prijmeni="{last_name}" typ_ds="F"/>
    <Veta1 obrat23="{}" dan23="{}" obrat5="{}" dan5="{}" .../>
    <Veta2 dod_zb="0" pln_sluzby="0" .../>
    <Veta4 pln23="{}" odp_tuz23_nar="{}" pln5="{}" odp_tuz5_nar="{}"
           odp_sum_kr="0" odp_sum_nar="{}"/>
    <Veta6 dano="0" dano_no="{}" dano_da="{}" dan_zocelk="{}" odp_zocelk="{}"/>
  </DPHDP3>
</Pisemnost>
```

**Key rules:**
- All amounts are whole CZK (rounded via `to_whole_czk_rounded`)
- `trans` = "A" if any amounts are non-zero, "N" otherwise
- `dano_da` (tax payable) set when net VAT > 0; `dano_no` (refund) set when net VAT < 0
- Submission date format: DD.MM.YYYY
- `verzePis` = "01.02.16" (current EPO schema version)

#### `src/xml/control_statement.rs` -- DPHKH1

```rust
use zfaktury_domain::{VATControlStatement, VATControlStatementLine};

/// Generate DPHKH1 control statement XML.
pub fn generate_control_statement_xml(
    cs: &VATControlStatement,
    lines: &[VATControlStatementLine],
    dic: &str,
) -> Result<Vec<u8>, GenError> { ... }
```

**XML structure:**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Pisemnost xmlns="http://adis.mfcr.cz/rozhranni/">
  <DPHKH1>
    <VetaD d_typ="{R|N|O}" rok="{}" mesic="{}" dokdphkh="KH" khdph_forma="{R|N|O}"/>
    <VetaP dic="{dic_without_CZ}" typ_ds="P"/>
    <VetaA4 c_evid_dd="{doc_num}" dppd="{DD.MM.YYYY}" dic_odb="{partner_dic}"
            kod_rezim_pln="0" zakl_dane1="{base}" dan1="{vat}"/>
    <VetaA5 kod_rezim_pln="0" zakl_dane1="{}" dan1="{}"/>
    <VetaB2 c_evid_dd="{}" dppd="{}" dic_dod="{}" kod_rezim_pln="0" .../>
    <VetaB3 kod_rezim_pln="0" .../>
  </DPHKH1>
</Pisemnost>
```

**Section routing rules (from Go):**
- **A4:** Output transactions > 10,000 CZK (individual, with partner DIC)
- **A5:** Output transactions <= 10,000 CZK (aggregated, no partner DIC)
- **B2:** Input transactions > 10,000 CZK (individual, with partner DIC)
- **B3:** Input transactions <= 10,000 CZK (aggregated)

Threshold constant: `ControlStatementThreshold = 1_000_000` halere (10,000 CZK).

VAT rate fields: `zakl_dane1`/`dan1` for 21%, `zakl_dane2`/`dan2` for 12%. Use `Option<i64>` (serialize as attribute only when `Some`).

Date format in XML: DD.MM.YYYY (converted from YYYY-MM-DD stored in domain).

#### `src/xml/vies.rs` -- DPHSHV

```rust
use zfaktury_domain::{VIESSummary, VIESSummaryLine};

/// Generate DPHSHV VIES recapitulative statement XML.
pub fn generate_vies_xml(
    vs: &VIESSummary,
    lines: &[VIESSummaryLine],
    dic: &str,
) -> Result<Vec<u8>, GenError> { ... }
```

**XML structure:**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Pisemnost xmlns="http://adis.mfcr.cz/rozhranni/">
  <DPHSHV>
    <VetaD k_daph="{B|O|N}" rok="{}" ctvrt="{}" dic_odb="{filer_dic}"/>
    <VetaP k_stat="{CC}" dic_odbe="{partner_dic_no_prefix}" k_plneni="3" obrat="{czk}"/>
    <!-- one VetaP per EU partner -->
  </DPHSHV>
</Pisemnost>
```

**Key rules:**
- Strip 2-letter country prefix from partner DIC (`DE123456789` -> `123456789`)
- Filing type codes differ from DPHDP3: B=regular, O=corrective, N=supplementary
- `k_plneni` = "3" for services (default for Czech OSVC)
- `obrat` = whole CZK (rounded)

#### `src/xml/income_tax.rs` -- DPFDP5

```rust
use std::collections::HashMap;
use zfaktury_domain::IncomeTaxReturn;

/// Generate DPFDP5 income tax return XML.
///
/// Required settings keys:
/// - financni_urad_code, dic
/// - taxpayer_first_name, taxpayer_last_name, taxpayer_birth_number
/// - taxpayer_street, taxpayer_house_number, taxpayer_city, taxpayer_postal_code
pub fn generate_income_tax_xml(
    itr: &IncomeTaxReturn,
    settings: &HashMap<String, String>,
) -> Result<Vec<u8>, GenError> { ... }
```

**XML structure:**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Pisemnost nazevSW="ZFaktury" verzeSW="1.0">
  <DPFDP5 verzePis="05.01">
    <VetaD dokument="DP5" k_uladis="DPF" rok="{}" dap_typ="{B|O|D}"
           c_ufo_cil="{fu_code}" pln_moc="A" audit="N"
           kc_zd7="{}" pr_zd7="{}" vy_zd7="{}"
           kc_zakldan23="{}" kc_zakldan="{}" kc_zdzaokr="{}"
           da_slezap="{}" sleva_rp="{}" uhrn_slevy35ba="{}"
           da_slevy35ba="{}" kc_dazvyhod="{}" da_slevy35c="{}"
           kc_zalpred="{}" kc_zbyvpred="{}"/>
    <VetaP jmeno="{}" prijmeni="{}" rod_c="{}" dic="{}"
           ulice="{}" c_pop="{}" naz_obce="{}" psc="{}"
           k_stat="CZ" stat="CESKA REPUBLIKA"/>
  </DPFDP5>
</Pisemnost>
```

**Key rules:**
- All amounts in whole CZK using **truncated** integer division (`to_whole_czk_truncated`)
- `verzePis` = "05.01"
- Section 7 fields: `kc_zd7` (tax base), `pr_zd7` (revenue), `vy_zd7` (expenses)
- Tax credits: `sleva_rp` (basic), `uhrn_slevy35ba` (total credits), `da_slevy35ba` (tax after credits)
- Child benefit: `kc_dazvyhod`, `da_slevy35c`
- `stat` = "CESKA REPUBLIKA" (uppercase Czech, with diacritics on C and A)
- Filing type codes: B=regular, O=corrective, D=supplementary

#### `src/xml/social_insurance.rs` -- CSSZ/OSVC

```rust
use std::collections::HashMap;
use zfaktury_domain::SocialInsuranceOverview;

/// Generate CSSZ OSVC annual overview XML.
///
/// Required settings keys:
/// - cssz_code, flat_rate_expenses
/// - taxpayer_first_name, taxpayer_last_name, taxpayer_birth_number, taxpayer_birth_date
/// - taxpayer_street, taxpayer_house_number, taxpayer_city, taxpayer_postal_code
pub fn generate_social_insurance_xml(
    sio: &SocialInsuranceOverview,
    settings: &HashMap<String, String>,
) -> Result<Vec<u8>, GenError> { ... }
```

**XML structure:**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<OSVC xmlns="http://schemas.cssz.cz/OSVC2025" version="1.0">
  <VENDOR productName="ZFaktury" productVersion="1.0"/>
  <SENDER EmailNotifikace="" ISDSreport="3" VerzeProtokolu="1"/>
  <prehledosvc for="{cssz_code}" dep="122" rok="{year}" typ="{N|O|Z}" vsdp="" dat="">
    <client>
      <name fir="{first}" sur="{last}" tit=""/>
      <birth bno="{birth_number}" den="{birth_date}"/>
      <adr str="{street}" num="{house}" pnu="{zip}" cit="{city}" cnt="CZ"/>
      <druc>H</druc>
      <hlavc><m1>A</m1><m2/><m3/>...<m13/></hlavc>
      <vedc><m1/><m2/>...<m13/><zam/><duchod/><pdite/><ppm/><pece/><ndite/></vedc>
      <sleva><m1>n</m1>...</sleva>
    </client>
    <pvv pri="1">
      <mesc h="1" v=""/><mesv h="1" v=""/><mesp>1</mesp>
      <rdza h="{revenue}" v="{expenses}"/>
      <vvz h="{assessment_base}" v="0"/>
      <dvz h="0" v="0"/>
      <mvz>{min_base}</mvz><uvz>{final_base}</uvz>
      <vzsu>{final_base}</vzsu><vzsvc>{final_base}</vzsvc>
      <poj>{total_insurance}</poj><zal>{prepayments}</zal><ned>{difference}</ned>
    </pvv>
    <zal pau="{A|N}" vz="{final_base}" dp="{monthly_prepay}" np="0" duch=""/>
    <pre vra="0"><bs pu="" cu="" kb="" ss="" vs=""/></pre>
    <prizn><pau>H</pau><pov>A</pov><elektr>N</elektr><por>N</por></prizn>
    <spo><name sur="" fir="" tit=""/></spo>
  </prehledosvc>
</OSVC>
```

**Key rules:**
- Namespace: `http://schemas.cssz.cz/OSVC2025`
- All amounts as whole CZK strings (truncated)
- Filing type codes: N=regular, O=corrective, Z=supplementary
- `flat_rate_expenses` setting: "true" -> `pau="A"`, else `pau="N"` in `zal` section
- Month flags: `hlavc.m1` = "A" (main activity), `sleva.m1` = "n" (no discount) as defaults
- `dep` = "122" (fixed department code)
- `druc` = "H" (main activity type)

#### `src/xml/health_insurance.rs`

```rust
use std::collections::HashMap;
use zfaktury_domain::HealthInsuranceOverview;

/// Generate health insurance annual overview XML.
///
/// The Go codebase does not yet have a health insurance XML generator.
/// This module implements it following the same pattern as social insurance
/// but adapted for the health insurance authority format (VZP/other ZP).
pub fn generate_health_insurance_xml(
    hio: &HealthInsuranceOverview,
    settings: &HashMap<String, String>,
) -> Result<Vec<u8>, GenError> { ... }
```

**Note:** The Go codebase has the `HealthInsuranceOverview` domain struct but no XML generator yet. The Rust implementation will be the first. The exact XML schema depends on the health insurance provider (VZP, OZP, etc.). The initial implementation will target the VZP format, which is the most common. If the XSD is not publicly available, implement a reasonable structure mirroring the social insurance pattern and validate against real submissions.

### 3. ISDOC Generation (`src/isdoc/`)

#### `src/isdoc/types.rs`

Serde-serializable structs matching the ISDOC 6.0.2 XSD. Ported 1:1 from Go `internal/isdoc/types.go`:

```rust
use serde::Serialize;

pub const ISDOC_NAMESPACE: &str = "urn:isdoc:invoice:6.0.2";

#[derive(Serialize)]
#[serde(rename = "Invoice")]
pub struct IsdocInvoice {
    #[serde(rename = "@xmlns")]
    pub xmlns: String,            // "urn:isdoc:invoice:6.0.2"
    #[serde(rename = "@version")]
    pub version: String,          // "6.0.2"
    #[serde(rename = "DocumentType")]
    pub document_type: i32,       // 1=invoice, 2=credit note, 4=proforma
    #[serde(rename = "ID")]
    pub id: String,
    #[serde(rename = "UUID")]
    pub uuid: String,             // "zfaktury-{id}"
    // ... remaining fields mirror Go types.go
}
```

#### `src/isdoc/generator.rs`

```rust
use zfaktury_domain::Invoice;
use crate::pdf::types::SupplierInfo;

/// Generate ISDOC 6.0.2 XML for the given invoice.
///
/// All monetary amounts are formatted as decimal strings ("1234.56"),
/// NOT whole CZK like the EPO tax XMLs.
pub fn generate_isdoc(
    invoice: &Invoice,
    supplier: &SupplierInfo,
) -> Result<Vec<u8>, GenError> { ... }
```

**Key rules (from Go implementation):**
- Document types: 1=invoice, 2=credit note, 4=proforma
- UUID format: `"zfaktury-{id}"`
- Amounts as decimal strings with 2 decimal places (e.g., `"1234.56"`)
- Foreign currency: set `ForeignCurrencyCode`, `CurrRate`, `RefCurrRate` = "1.00"
- PaymentMeansCode: 42=bank transfer, 10=cash
- Country defaults to "CZ" when empty
- VAT calculation method: 0 (from top)
- TaxSubTotal includes "already claimed" and "difference" fields (set to "0.00" for initial invoice)
- LegalMonetaryTotal: `PayableAmount` = `TotalAmount - PaidAmount`

### 4. QR Code Generation (`src/qr/`)

#### `src/qr/spayd.rs`

```rust
use zfaktury_domain::Invoice;

/// Generate a SPAYD/SPD payment string for Czech bank QR codes.
///
/// Format: SPD*1.0*ACC:{IBAN}+{SWIFT}*AM:{amount}*CC:CZK*X-VS:{vs}*...
pub fn generate_spayd_string(
    invoice: &Invoice,
    iban: &str,
    swift: &str,
) -> Result<String, GenError> { ... }

/// Generate a QR code PNG image from a SPAYD string.
///
/// Uses the `qrcode` crate to produce a PNG image suitable
/// for embedding in PDFs or serving via HTTP.
pub fn generate_qr_png(spayd: &str) -> Result<Vec<u8>, GenError> { ... }
```

**SPAYD format specification:**

```
SPD*1.0*ACC:{IBAN}+{SWIFT}*AM:{amount_czk}*CC:CZK*X-VS:{variable_symbol}*X-KS:{constant_symbol}*DT:{YYYYMMDD}*MSG:{invoice_number}
```

- IBAN is required; return error if empty
- SWIFT/BIC is optional
- Amount in CZK with 2 decimal places (e.g., "12345.67")
- Date format for `DT` field: YYYYMMDD (no separators)
- `X-VS` and `X-KS` are extended attributes, included only when non-empty
- `MSG` = invoice number, included only when non-empty

**Go compatibility:** The Go version uses `github.com/dundee/qrpay` which handles SPAYD formatting. The Rust version builds the string manually following the same SPD 1.0 specification, then uses the `qrcode` crate for PNG rendering.

### 5. CSV Export (`src/csv/`)

#### `src/csv/invoices.rs`

```rust
use zfaktury_domain::Invoice;

/// Export invoices to CSV bytes with Czech formatting.
///
/// Columns: Cislo, Typ, Stav, Odberatel, Datum vystaveni,
///          Datum splatnosti, DUZP, Castka bez DPH, DPH, Celkem, Mena
pub fn export_invoices_csv(invoices: &[Invoice]) -> Result<Vec<u8>, GenError> { ... }
```

#### `src/csv/expenses.rs`

```rust
use zfaktury_domain::Expense;

/// Export expenses to CSV bytes with Czech formatting.
///
/// Columns: Cislo, Popis, Kategorie, Dodavatel, Datum, Castka, DPH, Mena
pub fn export_expenses_csv(expenses: &[Expense]) -> Result<Vec<u8>, GenError> { ... }
```

**CSV formatting rules (from Go `export_handler.go`):**
- Delimiter: semicolon (`;`) -- Excel compatibility for Czech locale
- UTF-8 BOM prefix: `0xEF, 0xBB, 0xBF` (first 3 bytes)
- Amount format: comma as decimal separator (`"1234,56"` not `"1234.56"`)
- Date format: DD.MM.YYYY
- Customer/vendor name: empty string if null

### 6. Error Type

All generation modules return a unified error type:

```rust
use thiserror::Error;

#[derive(Error, Debug)]
pub enum GenError {
    #[error("invalid input: {0}")]
    InvalidInput(String),

    #[error("XML serialization failed: {0}")]
    XmlSerialization(#[from] quick_xml::Error),

    #[error("PDF generation failed: {0}")]
    PdfGeneration(String),

    #[error("QR code generation failed: {0}")]
    QrGeneration(String),

    #[error("CSV generation failed: {0}")]
    CsvGeneration(#[from] csv::Error),

    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),
}
```

## XML Serialization Strategy

### `quick-xml` with serde

All XML generation uses `quick-xml` with serde derive macros. The EPO format uses XML attributes heavily (not child elements), so serde field renaming with `@` prefix is essential:

```rust
use serde::Serialize;

#[derive(Serialize)]
#[serde(rename = "VetaD")]
struct VetaD {
    #[serde(rename = "@dokument")]
    dokument: String,
    #[serde(rename = "@k_uladis")]
    k_uladis: String,
    #[serde(rename = "@rok")]
    rok: i32,
    // ...
}
```

### XML Declaration

Every generated XML starts with `<?xml version="1.0" encoding="UTF-8"?>` followed by a newline. The Go version prepends `xml.Header` manually; the Rust version does the same:

```rust
fn serialize_with_header<T: Serialize>(doc: &T) -> Result<Vec<u8>, GenError> {
    let mut buf = Vec::from(b"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n");
    let writer = quick_xml::Writer::new_with_indent(&mut buf, b' ', 2);
    // ... serialize doc into writer
    Ok(buf)
}
```

### Attribute Ordering

XML attribute order should match the Go output for golden file compatibility. If `quick-xml` + serde doesn't preserve struct field order, implement manual serialization for the affected types using `quick_xml::Writer` directly.

## Golden File Testing Strategy

### File Organization

```
tests/golden/
  xml/
    vat_return_regular.golden.xml
    vat_return_corrective.golden.xml
    vat_return_negative.golden.xml        # refund scenario
    vat_return_zero.golden.xml            # no transactions
    control_statement_a4a5.golden.xml
    control_statement_b2b3.golden.xml
    control_statement_corrective.golden.xml
    vies_summary_regular.golden.xml
    vies_summary_empty.golden.xml
    income_tax_minimal.golden.xml
    income_tax_full.golden.xml
    income_tax_corrective.golden.xml
    income_tax_supplementary.golden.xml
    social_insurance_regular.golden.xml
    social_insurance_actual_expenses.golden.xml
    social_insurance_corrective.golden.xml
    social_insurance_supplementary.golden.xml
    health_insurance_regular.golden.xml
  isdoc/
    regular_invoice.golden.xml
    credit_note.golden.xml
    foreign_currency.golden.xml
    cash_payment.golden.xml
  pdf/
    regular_invoice.golden.pdf_meta       # extracted text + metadata
  csv/
    invoices_2025.golden.csv
    expenses_2025.golden.csv
```

### Golden File Workflow

```rust
/// Compare generated output against a golden file.
/// Set UPDATE_GOLDEN=1 to regenerate golden files.
fn assert_golden(name: &str, actual: &[u8]) {
    let golden_path = Path::new("tests/golden").join(name);
    if std::env::var("UPDATE_GOLDEN").is_ok() {
        std::fs::write(&golden_path, actual).unwrap();
        return;
    }
    let expected = std::fs::read(&golden_path)
        .unwrap_or_else(|_| panic!("golden file not found: {}", golden_path.display()));
    if actual != expected {
        // Use `similar` crate for readable diff
        let diff = similar::TextDiff::from_lines(
            &String::from_utf8_lossy(&expected),
            &String::from_utf8_lossy(actual),
        );
        panic!(
            "Golden file mismatch: {}\n{}",
            golden_path.display(),
            diff.unified_diff().header("expected", "actual")
        );
    }
}
```

**Workflow:**
1. `UPDATE_GOLDEN=1 cargo test` -- regenerate all golden files
2. `cargo test` -- strict comparison, fails on any diff
3. CI always runs without `UPDATE_GOLDEN`

### Porting Golden Files from Go

The initial golden files are ported from the Go testdata directories:
- `internal/vatxml/testdata/*.golden.xml` -> `tests/golden/xml/`
- `internal/annualtaxxml/testdata/*.golden.xml` -> `tests/golden/xml/`
- `internal/isdoc/testdata/*.golden.xml` -> `tests/golden/isdoc/`

The Rust output must match the Go output byte-for-byte (same attribute order, same indentation, same XML declaration). Any deviation indicates a bug in the port.

### XSD Validation in Tests

For XML modules, tests perform structural validation using `roxmltree`:

```rust
fn validate_xml_structure(xml: &[u8], expected_root: &str, required_children: &[&str]) {
    let doc = roxmltree::Document::parse(std::str::from_utf8(xml).unwrap()).unwrap();
    let root = doc.root_element();
    assert_eq!(root.tag_name().name(), expected_root);
    for child_name in required_children {
        assert!(
            root.descendants().any(|n| n.tag_name().name() == *child_name),
            "Missing required element: {child_name}"
        );
    }
}
```

Where official XSD files are available (DPHDP3, DPHKH, ISDOC 6.0.2), store them in `tests/schemas/` and validate against them. If a Rust XSD validator crate is not mature enough, fall back to structural checks + golden file comparison.

### PDF Testing

PDF testing uses text extraction (not pixel comparison):

```rust
#[test]
fn test_invoice_pdf_content() {
    let pdf_bytes = generate_invoice_pdf(&invoice, &supplier, &settings).unwrap();

    // Verify it's a valid PDF
    assert!(pdf_bytes.starts_with(b"%PDF"));

    // Extract text and verify key content
    let text = pdf_extract::extract_text_from_mem(&pdf_bytes).unwrap();
    assert!(text.contains("FV-2025-001"));
    assert!(text.contains("Faktura"));
    assert!(text.contains(&supplier.name));
    assert!(text.contains("Celkem"));
}
```

### CSV Testing

CSV roundtrip: generate -> parse back -> verify columns and values:

```rust
#[test]
fn test_invoice_csv_roundtrip() {
    let csv_bytes = export_invoices_csv(&invoices).unwrap();

    // Verify BOM
    assert_eq!(&csv_bytes[..3], &[0xEF, 0xBB, 0xBF]);

    // Parse back
    let mut reader = csv::ReaderBuilder::new()
        .delimiter(b';')
        .from_reader(&csv_bytes[3..]);  // skip BOM

    let headers = reader.headers().unwrap();
    assert_eq!(headers[0], "Cislo");
    assert_eq!(headers[10], "Mena");

    let records: Vec<_> = reader.records().collect::<Result<_, _>>().unwrap();
    assert_eq!(records.len(), invoices.len());
    assert_eq!(&records[0][0], "FV-2025-001");
}
```

## Test Coverage Requirements

| Module | Coverage Target | Test Types |
|--------|----------------|------------|
| `xml/vat_return` | 100% | Golden files (4 scenarios) + XSD structure |
| `xml/control_statement` | 100% | Golden files (3 scenarios) + section routing |
| `xml/vies` | 100% | Golden files (2 scenarios) + DIC stripping |
| `xml/income_tax` | 100% | Golden files (4 scenarios) + truncation verification |
| `xml/social_insurance` | 100% | Golden files (4 scenarios) + filing type codes |
| `xml/health_insurance` | 100% | Golden files (1+ scenarios) |
| `isdoc/generator` | 100% | Golden files (4 scenarios) + decimal amounts |
| `pdf/invoice` | 100% | Text extraction + metadata |
| `qr/spayd` | 100% | SPAYD string format + PNG validity |
| `csv/invoices` | 100% | Roundtrip + BOM + delimiter |
| `csv/expenses` | 100% | Roundtrip + BOM + delimiter |
| `xml/common` | 100% | Unit tests for rounding/truncation + filing codes |

**Total golden files:** ~25 XML + 4 ISDOC + 2 CSV + 1 PDF metadata = ~32 golden files.

## Implementation Order

### Week 1: XML + ISDOC + CSV

1. **Day 1-2:** `xml/common.rs` + `xml/vat_return.rs` with golden file tests
2. **Day 2-3:** `xml/control_statement.rs` + `xml/vies.rs` with golden file tests
3. **Day 3-4:** `xml/income_tax.rs` + `xml/social_insurance.rs` + `xml/health_insurance.rs`
4. **Day 4-5:** `isdoc/generator.rs` + `csv/invoices.rs` + `csv/expenses.rs`

### Week 2: PDF + QR + Integration

5. **Day 6-7:** `qr/spayd.rs` (SPAYD string + PNG)
6. **Day 7-9:** `pdf/invoice.rs` (typst template, font embedding, QR integration)
7. **Day 9-10:** Full integration tests, XSD validation, coverage verification

## Acceptance Criteria

1. All XML outputs validate against their respective XSD schemas (where available)
2. All golden file comparisons pass with byte-for-byte match to Go output
3. Generated PDF contains correct invoice data verified by text extraction
4. ISDOC output conforms to ISDOC 6.0.2 specification
5. QR code contains valid SPAYD string parseable by Czech banking apps
6. CSV roundtrip: generate -> parse -> verify all columns and values match
7. 100% line coverage for all modules in `zfaktury-gen`
8. No hardcoded test data in production code
9. Error handling: all public functions return `Result<_, GenError>` with descriptive messages

## Review Checklist

- [ ] XML namespaces and attributes match Czech EPO specifications exactly
- [ ] VAT XML amounts use `to_whole_czk_rounded` (math.Round equivalent)
- [ ] Income tax/insurance amounts use `to_whole_czk_truncated` (integer division)
- [ ] ISDOC amounts are decimal strings with 2 places ("1234.56")
- [ ] PDF uses Czech locale for all labels and number formatting
- [ ] QR code SPAYD format matches SPD 1.0 specification
- [ ] CSV uses semicolon delimiter, UTF-8 BOM, comma decimal separator
- [ ] Filing type code mappings match Go for all 5 form types
- [ ] No `unwrap()` in production code (only in tests)
- [ ] Golden files are committed and diffable in PRs
- [ ] XML attribute order matches Go output for golden file compatibility
- [ ] `SupplierInfo` struct used consistently across PDF and ISDOC (consider shared type)

## Open Questions

1. **typst as library:** The typst crate API has been evolving. Verify that `typst v0.13` supports programmatic document construction (without shelling out to the CLI). If the library API is too unstable, fall back to `printpdf` or `genpdf` with manual layout.

2. **Health insurance XSD:** The Go codebase has no health insurance XML generator yet. Confirm the target XSD schema (VZP format?) before implementing. If no official XSD is available, defer this sub-module to a later phase.

3. **Shared SupplierInfo:** Both `pdf` and `isdoc` modules need supplier info. Consider defining a single `SupplierInfo` in the crate root or domain crate rather than duplicating in both modules.

4. **Font licensing:** Liberation Sans is libre-licensed (SIL OFL). Confirm the embedded font approach works with typst and that the license is included in the binary distribution.
