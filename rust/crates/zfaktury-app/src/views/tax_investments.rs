use std::path::PathBuf;
use std::sync::Arc;

use chrono::{Datelike, Local, NaiveDate};
use gpui::*;
use zfaktury_core::service::{InvestmentDocumentService, InvestmentIncomeService, OCRService};
use zfaktury_domain::{
    Amount, AssetType, CapitalCategory, CapitalIncomeEntry, ExtractionStatus, InvestmentDocument,
    InvestmentYearSummary, Platform, SecurityTransaction, TransactionType,
};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::checkbox::Checkbox;
use crate::components::confirm_dialog::{ConfirmDialog, ConfirmDialogResult};
use crate::components::date_input::DateInput;
use crate::components::number_input::NumberInput;
use crate::components::select::{Select, SelectOption};
use crate::components::text_input::TextInput;
use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;
use crate::util::format::{format_amount, format_date};

// ---------------------------------------------------------------------------
// Tabs
// ---------------------------------------------------------------------------

#[derive(Clone, PartialEq)]
enum InvestmentTab {
    Documents,
    CapitalIncome,
    SecurityTransactions,
}

// ---------------------------------------------------------------------------
// Editing state for Capital Income inline form
// ---------------------------------------------------------------------------

struct EditingCapitalEntry {
    id: i64, // 0 = new
    category: Entity<Select>,
    description: Entity<TextInput>,
    income_date: Entity<DateInput>,
    gross_amount: Entity<NumberInput>,
    withheld_tax_cz: Entity<NumberInput>,
    withheld_tax_foreign: Entity<NumberInput>,
    country_code: Entity<TextInput>,
    needs_declaring: Entity<Checkbox>,
}

// ---------------------------------------------------------------------------
// Editing state for Security Transaction inline form
// ---------------------------------------------------------------------------

struct EditingSecurityTx {
    id: i64, // 0 = new
    asset_type: Entity<Select>,
    asset_name: Entity<TextInput>,
    isin: Entity<TextInput>,
    transaction_type: Entity<Select>,
    transaction_date: Entity<DateInput>,
    quantity: Entity<NumberInput>,
    unit_price: Entity<NumberInput>,
    fees: Entity<NumberInput>,
    currency_code: Entity<TextInput>,
    exchange_rate: Entity<NumberInput>,
}

// ---------------------------------------------------------------------------
// Delete target discriminator
// ---------------------------------------------------------------------------

#[derive(Clone)]
enum DeleteTarget {
    CapitalEntry(i64),
    SecurityTx(i64),
    Document(i64),
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

/// Tax investments view with tabbed CRUD for capital income, security transactions, and documents.
pub struct TaxInvestmentsView {
    service: Arc<InvestmentIncomeService>,
    document_service: Arc<InvestmentDocumentService>,
    ocr_service: Option<Arc<OCRService>>,
    year: i32,
    loading: bool,
    saving: bool,
    error: Option<String>,
    active_tab: InvestmentTab,
    summary: Option<InvestmentYearSummary>,
    capital_entries: Vec<CapitalIncomeEntry>,
    security_transactions: Vec<SecurityTransaction>,

    // Documents tab state
    documents: Vec<InvestmentDocument>,
    uploading: bool,
    extracting_id: Option<i64>,
    platform_select: Option<Entity<Select>>,

    // Inline editing state
    editing_capital: Option<EditingCapitalEntry>,
    editing_security: Option<EditingSecurityTx>,

    // Delete confirmation
    confirm_dialog: Option<Entity<ConfirmDialog>>,
    delete_target: Option<DeleteTarget>,
}

impl TaxInvestmentsView {
    pub fn new(
        service: Arc<InvestmentIncomeService>,
        document_service: Arc<InvestmentDocumentService>,
        ocr_service: Option<Arc<OCRService>>,
        cx: &mut Context<Self>,
    ) -> Self {
        let year = Local::now().date_naive().year();
        let platform_select = Some(cx.new(|_cx| {
            Select::new(
                "doc-platform-select",
                "Platforma...",
                Self::platform_options(),
            )
        }));
        let mut view = Self {
            service,
            document_service,
            ocr_service,
            year,
            loading: true,
            saving: false,
            error: None,
            active_tab: InvestmentTab::Documents,
            summary: None,
            capital_entries: Vec::new(),
            security_transactions: Vec::new(),
            documents: Vec::new(),
            uploading: false,
            extracting_id: None,
            platform_select,
            editing_capital: None,
            editing_security: None,
            confirm_dialog: None,
            delete_target: None,
        };
        view.load_data(cx);
        view
    }

    // ------------------------------------------------------------------
    // Data loading
    // ------------------------------------------------------------------

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        let doc_service = self.document_service.clone();
        let year = self.year;

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    let summary = service.get_year_summary(year)?;
                    let capital = service.list_capital_entries(year)?;
                    let securities = service.list_security_transactions(year)?;
                    let documents = doc_service.list_by_year(year)?;
                    Ok::<
                        (
                            InvestmentYearSummary,
                            Vec<CapitalIncomeEntry>,
                            Vec<SecurityTransaction>,
                            Vec<InvestmentDocument>,
                        ),
                        zfaktury_domain::DomainError,
                    >((summary, capital, securities, documents))
                })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok((summary, capital, securities, documents)) => {
                        this.summary = Some(summary);
                        this.capital_entries = capital;
                        this.security_transactions = securities;
                        this.documents = documents;
                    }
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani investic: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn change_year(&mut self, delta: i32, cx: &mut Context<Self>) {
        self.year += delta;
        self.loading = true;
        self.error = None;
        self.editing_capital = None;
        self.editing_security = None;
        self.extracting_id = None;
        cx.notify();
        self.load_data(cx);
    }

    // ------------------------------------------------------------------
    // Platform helpers
    // ------------------------------------------------------------------

    fn platform_options() -> Vec<SelectOption> {
        vec![
            SelectOption {
                value: "portu".into(),
                label: "Portu".into(),
            },
            SelectOption {
                value: "zonky".into(),
                label: "Zonky".into(),
            },
            SelectOption {
                value: "trading212".into(),
                label: "Trading 212".into(),
            },
            SelectOption {
                value: "revolut".into(),
                label: "Revolut".into(),
            },
            SelectOption {
                value: "other".into(),
                label: "Ostatni".into(),
            },
        ]
    }

    fn parse_platform(value: &str) -> Platform {
        match value {
            "portu" => Platform::Portu,
            "zonky" => Platform::Zonky,
            "trading212" => Platform::Trading212,
            "revolut" => Platform::Revolut,
            _ => Platform::Other,
        }
    }

    fn platform_label(platform: &Platform) -> &'static str {
        match platform {
            Platform::Portu => "Portu",
            Platform::Zonky => "Zonky",
            Platform::Trading212 => "Trading 212",
            Platform::Revolut => "Revolut",
            Platform::Other => "Ostatni",
        }
    }

    fn extraction_status_label(status: &ExtractionStatus) -> &'static str {
        match status {
            ExtractionStatus::Pending => "Ceka na zpracovani",
            ExtractionStatus::Extracted => "Zpracovano",
            ExtractionStatus::Failed => "Chyba",
        }
    }

    fn extraction_status_color(status: &ExtractionStatus) -> u32 {
        match status {
            ExtractionStatus::Pending => ZfColors::STATUS_YELLOW,
            ExtractionStatus::Extracted => ZfColors::STATUS_GREEN,
            ExtractionStatus::Failed => ZfColors::STATUS_RED,
        }
    }

    fn extraction_status_bg(status: &ExtractionStatus) -> u32 {
        match status {
            ExtractionStatus::Pending => ZfColors::STATUS_YELLOW_BG,
            ExtractionStatus::Extracted => ZfColors::STATUS_GREEN_BG,
            ExtractionStatus::Failed => ZfColors::STATUS_RED_BG,
        }
    }

    fn content_type_from_filename(filename: &str) -> String {
        let lower = filename.to_lowercase();
        if lower.ends_with(".pdf") {
            "application/pdf".to_string()
        } else if lower.ends_with(".png") {
            "image/png".to_string()
        } else if lower.ends_with(".jpg") || lower.ends_with(".jpeg") {
            "image/jpeg".to_string()
        } else if lower.ends_with(".csv") {
            "text/csv".to_string()
        } else {
            "application/octet-stream".to_string()
        }
    }

    // ------------------------------------------------------------------
    // Document upload flow
    // ------------------------------------------------------------------

    fn upload_document(&mut self, cx: &mut Context<Self>) {
        if self.uploading {
            return;
        }

        // Read selected platform
        let platform_value = self
            .platform_select
            .as_ref()
            .and_then(|s| s.read(cx).selected_value().map(|v| v.to_string()))
            .unwrap_or_else(|| "other".to_string());

        self.uploading = true;
        self.error = None;
        cx.notify();

        let doc_service = self.document_service.clone();
        let year = self.year;
        let platform = Self::parse_platform(&platform_value);

        cx.spawn(async move |this, cx| {
            // Open file dialog via async API (xdg-desktop-portal)
            let file_result = rfd::AsyncFileDialog::new()
                .set_title("Vyberte soubor")
                .add_filter("Dokumenty", &["pdf", "png", "jpg", "jpeg", "csv"])
                .pick_file()
                .await;

            let file_path = match file_result {
                Some(handle) => handle.path().to_path_buf(),
                None => {
                    // User cancelled
                    this.update(cx, |this, cx| {
                        this.uploading = false;
                        cx.notify();
                    })
                    .ok();
                    return;
                }
            };

            // Read file and create record in background
            let result = cx
                .background_executor()
                .spawn(async move {
                    let filename = file_path
                        .file_name()
                        .map(|n| n.to_string_lossy().to_string())
                        .unwrap_or_else(|| "unknown".to_string());

                    let data = std::fs::read(&file_path).map_err(|e| {
                        log::error!("reading file {}: {e}", file_path.display());
                        zfaktury_domain::DomainError::InvalidInput
                    })?;

                    let content_type = Self::content_type_from_filename(&filename);
                    let size = data.len() as i64;

                    // Generate unique storage path
                    let timestamp = Local::now().format("%Y%m%d%H%M%S%f");
                    let storage_dir = PathBuf::from(doc_service.data_dir())
                        .join("investment-documents")
                        .join(year.to_string());
                    std::fs::create_dir_all(&storage_dir).map_err(|e| {
                        log::error!("creating storage dir {}: {e}", storage_dir.display());
                        zfaktury_domain::DomainError::InvalidInput
                    })?;

                    let storage_filename = format!("{timestamp}_{filename}");
                    let storage_path = storage_dir.join(&storage_filename);

                    std::fs::write(&storage_path, &data).map_err(|e| {
                        log::error!("writing file {}: {e}", storage_path.display());
                        zfaktury_domain::DomainError::InvalidInput
                    })?;

                    let now = Local::now().naive_local();
                    let mut doc = InvestmentDocument {
                        id: 0,
                        year,
                        platform,
                        filename,
                        content_type,
                        storage_path: storage_path.to_string_lossy().to_string(),
                        size,
                        extraction_status: ExtractionStatus::Pending,
                        extraction_error: String::new(),
                        created_at: now,
                        updated_at: now,
                    };

                    doc_service.create_record(&mut doc)?;
                    doc_service.list_by_year(year)
                })
                .await;

            this.update(cx, |this, cx| {
                this.uploading = false;
                match result {
                    Ok(documents) => {
                        this.documents = documents;
                    }
                    Err(e) => this.error = Some(format!("Chyba pri nahravani souboru: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    // ------------------------------------------------------------------
    // Document extraction flow
    // ------------------------------------------------------------------

    fn extract_document(&mut self, doc_id: i64, cx: &mut Context<Self>) {
        let ocr = match &self.ocr_service {
            Some(ocr) => ocr.clone(),
            None => {
                self.error = Some("OCR sluzba neni nakonfigurovana.".to_string());
                cx.notify();
                return;
            }
        };

        // Find the document to get its storage path and content type
        let doc = match self.documents.iter().find(|d| d.id == doc_id) {
            Some(d) => d.clone(),
            None => return,
        };

        self.extracting_id = Some(doc_id);
        self.error = None;
        cx.notify();

        let doc_service = self.document_service.clone();
        let service = self.service.clone();
        let year = self.year;

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    // Read file bytes
                    let data = std::fs::read(&doc.storage_path).map_err(|e| {
                        log::error!("reading document file {}: {e}", doc.storage_path);
                        zfaktury_domain::DomainError::NotFound
                    })?;

                    // Call OCR
                    let ocr_result = match ocr.process_bytes(&data, &doc.content_type) {
                        Ok(result) => result,
                        Err(e) => {
                            let _ = doc_service.update_extraction_status(
                                doc_id,
                                "failed",
                                &format!("{e}"),
                            );
                            return Err(e);
                        }
                    };

                    // Create a single capital income entry from OCR result if we got meaningful data
                    if !ocr_result.description.is_empty() && ocr_result.total_amount != Amount::ZERO
                    {
                        let income_date = if !ocr_result.issue_date.is_empty() {
                            NaiveDate::parse_from_str(&ocr_result.issue_date, "%Y-%m-%d")
                                .unwrap_or_else(|_| Local::now().date_naive())
                        } else {
                            Local::now().date_naive()
                        };

                        let now = Local::now().naive_local();
                        let mut entry = CapitalIncomeEntry {
                            id: 0,
                            year,
                            document_id: Some(doc_id),
                            category: CapitalCategory::Other,
                            description: ocr_result.description.clone(),
                            income_date,
                            gross_amount: ocr_result.total_amount,
                            withheld_tax_cz: ocr_result.vat_amount,
                            withheld_tax_foreign: Amount::ZERO,
                            country_code: "CZ".to_string(),
                            needs_declaring: true,
                            net_amount: Amount::ZERO,
                            created_at: now,
                            updated_at: now,
                        };
                        service.create_capital_entry(&mut entry)?;
                    }

                    // Mark as extracted
                    doc_service.update_extraction_status(doc_id, "extracted", "")?;

                    // Reload all data
                    let summary = service.get_year_summary(year)?;
                    let capital = service.list_capital_entries(year)?;
                    let securities = service.list_security_transactions(year)?;
                    let documents = doc_service.list_by_year(year)?;
                    Ok::<_, zfaktury_domain::DomainError>((summary, capital, securities, documents))
                })
                .await;

            this.update(cx, |this, cx| {
                this.extracting_id = None;
                match result {
                    Ok((summary, capital, securities, documents)) => {
                        this.summary = Some(summary);
                        this.capital_entries = capital;
                        this.security_transactions = securities;
                        this.documents = documents;
                    }
                    Err(e) => this.error = Some(format!("Chyba pri extrakci: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    // ------------------------------------------------------------------
    // Document delete
    // ------------------------------------------------------------------

    fn request_delete_document(&mut self, id: i64, cx: &mut Context<Self>) {
        self.delete_target = Some(DeleteTarget::Document(id));
        let dialog = cx.new(|_cx| {
            ConfirmDialog::new(
                "Smazat dokument",
                "Opravdu chcete smazat tento dokument? Budou smazany i vsechny propojene zaznamy.",
                "Smazat",
            )
        });
        cx.subscribe(&dialog, Self::on_confirm_result).detach();
        self.confirm_dialog = Some(dialog);
        cx.notify();
    }

    fn do_delete_document(&mut self, id: i64, cx: &mut Context<Self>) {
        self.saving = true;
        self.error = None;
        cx.notify();

        let doc_service = self.document_service.clone();
        let service = self.service.clone();
        let year = self.year;

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    // Also delete the file from disk
                    if let Ok(doc) = doc_service.get_by_id(id) {
                        let _ = std::fs::remove_file(&doc.storage_path);
                    }
                    doc_service.delete(id)?;
                    let summary = service.get_year_summary(year)?;
                    let capital = service.list_capital_entries(year)?;
                    let securities = service.list_security_transactions(year)?;
                    let documents = doc_service.list_by_year(year)?;
                    Ok::<_, zfaktury_domain::DomainError>((summary, capital, securities, documents))
                })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok((summary, capital, securities, documents)) => {
                        this.summary = Some(summary);
                        this.capital_entries = capital;
                        this.security_transactions = securities;
                        this.documents = documents;
                    }
                    Err(e) => this.error = Some(format!("Chyba pri mazani dokumentu: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    // ------------------------------------------------------------------
    // Capital Income CRUD
    // ------------------------------------------------------------------

    fn capital_category_options() -> Vec<SelectOption> {
        vec![
            SelectOption {
                value: "dividend_cz".into(),
                label: "Dividendy (CZ)".into(),
            },
            SelectOption {
                value: "dividend_foreign".into(),
                label: "Dividendy (zahranicni)".into(),
            },
            SelectOption {
                value: "interest".into(),
                label: "Uroky".into(),
            },
            SelectOption {
                value: "coupon".into(),
                label: "Kupony".into(),
            },
            SelectOption {
                value: "fund_distribution".into(),
                label: "Vynosy fondu".into(),
            },
            SelectOption {
                value: "other".into(),
                label: "Ostatni".into(),
            },
        ]
    }

    fn parse_capital_category(value: &str) -> CapitalCategory {
        match value {
            "dividend_cz" => CapitalCategory::DividendCZ,
            "dividend_foreign" => CapitalCategory::DividendForeign,
            "interest" => CapitalCategory::Interest,
            "coupon" => CapitalCategory::Coupon,
            "fund_distribution" => CapitalCategory::FundDistribution,
            _ => CapitalCategory::Other,
        }
    }

    fn capital_category_value(cat: &CapitalCategory) -> &'static str {
        match cat {
            CapitalCategory::DividendCZ => "dividend_cz",
            CapitalCategory::DividendForeign => "dividend_foreign",
            CapitalCategory::Interest => "interest",
            CapitalCategory::Coupon => "coupon",
            CapitalCategory::FundDistribution => "fund_distribution",
            CapitalCategory::Other => "other",
        }
    }

    fn capital_category_label(cat: &CapitalCategory) -> &'static str {
        match cat {
            CapitalCategory::DividendCZ => "Dividendy (CZ)",
            CapitalCategory::DividendForeign => "Dividendy (zahr.)",
            CapitalCategory::Interest => "Uroky",
            CapitalCategory::Coupon => "Kupony",
            CapitalCategory::FundDistribution => "Vynosy fondu",
            CapitalCategory::Other => "Ostatni",
        }
    }

    fn start_new_capital(&mut self, cx: &mut Context<Self>) {
        let category = cx.new(|_cx| {
            Select::new(
                "cap-new-category",
                "Kategorie...",
                Self::capital_category_options(),
            )
        });
        let description = cx.new(|cx| TextInput::new("cap-new-desc", "Popis...", cx));
        let income_date = cx.new(|cx| DateInput::new("cap-new-date", cx));
        let gross_amount = cx.new(|cx| NumberInput::new("cap-new-gross", "0.00", cx));
        let withheld_tax_cz = cx.new(|cx| NumberInput::new("cap-new-tax-cz", "0.00", cx));
        let withheld_tax_foreign = cx.new(|cx| NumberInput::new("cap-new-tax-for", "0.00", cx));
        let country_code = cx.new(|cx| {
            let mut t = TextInput::new("cap-new-country", "CZ", cx);
            t.set_value("CZ", cx);
            t
        });
        let needs_declaring = cx.new(|_cx| Checkbox::new("cap-new-declare", "Nutne danit", true));

        self.editing_capital = Some(EditingCapitalEntry {
            id: 0,
            category,
            description,
            income_date,
            gross_amount,
            withheld_tax_cz,
            withheld_tax_foreign,
            country_code,
            needs_declaring,
        });
        self.error = None;
        cx.notify();
    }

    fn start_edit_capital(&mut self, entry: &CapitalIncomeEntry, cx: &mut Context<Self>) {
        let eid = entry.id;
        let category = cx.new(|cx| {
            let mut s = Select::new(
                SharedString::from(format!("cap-edit-cat-{eid}")),
                "Kategorie...",
                Self::capital_category_options(),
            );
            s.set_selected_value(Self::capital_category_value(&entry.category), cx);
            s
        });
        let description = cx.new(|cx| {
            let mut t = TextInput::new(
                SharedString::from(format!("cap-edit-desc-{eid}")),
                "Popis...",
                cx,
            );
            t.set_value(&entry.description, cx);
            t
        });
        let income_date = cx.new(|cx| {
            let mut d = DateInput::new(SharedString::from(format!("cap-edit-date-{eid}")), cx);
            d.set_iso_value(&entry.income_date.format("%Y-%m-%d").to_string(), cx);
            d
        });
        let gross_amount = cx.new(|cx| {
            let mut n = NumberInput::new(
                SharedString::from(format!("cap-edit-gross-{eid}")),
                "0.00",
                cx,
            );
            n.set_amount(entry.gross_amount, cx);
            n
        });
        let withheld_tax_cz = cx.new(|cx| {
            let mut n = NumberInput::new(
                SharedString::from(format!("cap-edit-taxcz-{eid}")),
                "0.00",
                cx,
            );
            n.set_amount(entry.withheld_tax_cz, cx);
            n
        });
        let withheld_tax_foreign = cx.new(|cx| {
            let mut n = NumberInput::new(
                SharedString::from(format!("cap-edit-taxfor-{eid}")),
                "0.00",
                cx,
            );
            n.set_amount(entry.withheld_tax_foreign, cx);
            n
        });
        let country_code = cx.new(|cx| {
            let mut t = TextInput::new(
                SharedString::from(format!("cap-edit-country-{eid}")),
                "CZ",
                cx,
            );
            t.set_value(&entry.country_code, cx);
            t
        });
        let needs_declaring = cx.new(|_cx| {
            Checkbox::new(
                SharedString::from(format!("cap-edit-declare-{eid}")),
                "Nutne danit",
                entry.needs_declaring,
            )
        });

        self.editing_capital = Some(EditingCapitalEntry {
            id: eid,
            category,
            description,
            income_date,
            gross_amount,
            withheld_tax_cz,
            withheld_tax_foreign,
            country_code,
            needs_declaring,
        });
        self.error = None;
        cx.notify();
    }

    fn cancel_capital_edit(&mut self, cx: &mut Context<Self>) {
        self.editing_capital = None;
        self.error = None;
        cx.notify();
    }

    fn save_capital(&mut self, cx: &mut Context<Self>) {
        if self.saving {
            return;
        }
        let editing = match &self.editing_capital {
            Some(e) => e,
            None => return,
        };

        let cat_value = editing
            .category
            .read(cx)
            .selected_value()
            .unwrap_or("other")
            .to_string();
        let desc = editing.description.read(cx).value().to_string();
        let date_iso = editing.income_date.read(cx).iso_value().to_string();
        let gross = editing
            .gross_amount
            .read(cx)
            .to_amount()
            .unwrap_or(Amount::ZERO);
        let tax_cz = editing
            .withheld_tax_cz
            .read(cx)
            .to_amount()
            .unwrap_or(Amount::ZERO);
        let tax_foreign = editing
            .withheld_tax_foreign
            .read(cx)
            .to_amount()
            .unwrap_or(Amount::ZERO);
        let country = editing.country_code.read(cx).value().to_string();
        let declaring = editing.needs_declaring.read(cx).is_checked();
        let edit_id = editing.id;

        if desc.trim().is_empty() {
            self.error = Some("Popis je povinny.".into());
            cx.notify();
            return;
        }
        if date_iso.is_empty() {
            self.error = Some("Datum je povinne.".into());
            cx.notify();
            return;
        }

        let income_date = match NaiveDate::parse_from_str(&date_iso, "%Y-%m-%d") {
            Ok(d) => d,
            Err(_) => {
                self.error = Some("Neplatny format data.".into());
                cx.notify();
                return;
            }
        };

        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.service.clone();
        let doc_service = self.document_service.clone();
        let year = self.year;
        let now = Local::now().naive_local();
        let category = Self::parse_capital_category(&cat_value);

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    let mut entry = CapitalIncomeEntry {
                        id: edit_id,
                        year,
                        document_id: None,
                        category,
                        description: desc,
                        income_date,
                        gross_amount: gross,
                        withheld_tax_cz: tax_cz,
                        withheld_tax_foreign: tax_foreign,
                        country_code: country,
                        needs_declaring: declaring,
                        net_amount: Amount::ZERO, // computed by service
                        created_at: now,
                        updated_at: now,
                    };
                    if edit_id == 0 {
                        service.create_capital_entry(&mut entry)?;
                    } else {
                        service.update_capital_entry(&mut entry)?;
                    }
                    // Reload all data after save
                    let summary = service.get_year_summary(year)?;
                    let capital = service.list_capital_entries(year)?;
                    let securities = service.list_security_transactions(year)?;
                    let documents = doc_service.list_by_year(year)?;
                    Ok::<_, zfaktury_domain::DomainError>((summary, capital, securities, documents))
                })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok((summary, capital, securities, documents)) => {
                        this.summary = Some(summary);
                        this.capital_entries = capital;
                        this.security_transactions = securities;
                        this.documents = documents;
                        this.editing_capital = None;
                    }
                    Err(e) => this.error = Some(format!("Chyba pri ukladani: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    // ------------------------------------------------------------------
    // Security Transaction CRUD
    // ------------------------------------------------------------------

    fn asset_type_options() -> Vec<SelectOption> {
        vec![
            SelectOption {
                value: "stock".into(),
                label: "Akcie".into(),
            },
            SelectOption {
                value: "etf".into(),
                label: "ETF".into(),
            },
            SelectOption {
                value: "bond".into(),
                label: "Dluhopisy".into(),
            },
            SelectOption {
                value: "fund".into(),
                label: "Fondy".into(),
            },
            SelectOption {
                value: "crypto".into(),
                label: "Kryptomeny".into(),
            },
            SelectOption {
                value: "other".into(),
                label: "Ostatni".into(),
            },
        ]
    }

    fn tx_type_options() -> Vec<SelectOption> {
        vec![
            SelectOption {
                value: "buy".into(),
                label: "Nakup".into(),
            },
            SelectOption {
                value: "sell".into(),
                label: "Prodej".into(),
            },
        ]
    }

    fn parse_asset_type(value: &str) -> AssetType {
        match value {
            "stock" => AssetType::Stock,
            "etf" => AssetType::ETF,
            "bond" => AssetType::Bond,
            "fund" => AssetType::Fund,
            "crypto" => AssetType::Crypto,
            _ => AssetType::Other,
        }
    }

    fn asset_type_value(at: &AssetType) -> &'static str {
        match at {
            AssetType::Stock => "stock",
            AssetType::ETF => "etf",
            AssetType::Bond => "bond",
            AssetType::Fund => "fund",
            AssetType::Crypto => "crypto",
            AssetType::Other => "other",
        }
    }

    fn asset_type_label(at: &AssetType) -> &'static str {
        match at {
            AssetType::Stock => "Akcie",
            AssetType::ETF => "ETF",
            AssetType::Bond => "Dluhopisy",
            AssetType::Fund => "Fondy",
            AssetType::Crypto => "Kryptomeny",
            AssetType::Other => "Ostatni",
        }
    }

    fn parse_tx_type(value: &str) -> TransactionType {
        match value {
            "buy" => TransactionType::Buy,
            _ => TransactionType::Sell,
        }
    }

    fn tx_type_value(tt: &TransactionType) -> &'static str {
        match tt {
            TransactionType::Buy => "buy",
            TransactionType::Sell => "sell",
        }
    }

    fn tx_type_label(tt: &TransactionType) -> &'static str {
        match tt {
            TransactionType::Buy => "Nakup",
            TransactionType::Sell => "Prodej",
        }
    }

    /// Convert display quantity (e.g. "1.5") to internal representation (15000).
    fn quantity_display_to_internal(display: &str) -> i64 {
        if display.is_empty() {
            return 0;
        }
        let normalized = display.replace(',', ".");
        let f: f64 = normalized.parse().unwrap_or(0.0);
        (f * 10000.0).round() as i64
    }

    /// Convert internal quantity (e.g. 15000) to display string ("1.5000").
    fn quantity_internal_to_display(internal: i64) -> String {
        let whole = internal / 10000;
        let frac = (internal % 10000).unsigned_abs();
        if frac == 0 {
            format!("{whole}")
        } else {
            // Trim trailing zeros
            let frac_str = format!("{frac:04}");
            let trimmed = frac_str.trim_end_matches('0');
            format!("{whole}.{trimmed}")
        }
    }

    /// Convert exchange rate display to internal (10000 = 1.0).
    fn exchange_rate_display_to_internal(display: &str) -> i64 {
        if display.is_empty() {
            return 10000; // default 1.0
        }
        let normalized = display.replace(',', ".");
        let f: f64 = normalized.parse().unwrap_or(1.0);
        (f * 10000.0).round() as i64
    }

    fn exchange_rate_internal_to_display(internal: i64) -> String {
        let f = internal as f64 / 10000.0;
        format!("{f:.4}")
    }

    fn start_new_security(&mut self, cx: &mut Context<Self>) {
        let asset_type =
            cx.new(|_cx| Select::new("sec-new-asset-type", "Typ...", Self::asset_type_options()));
        let asset_name = cx.new(|cx| TextInput::new("sec-new-name", "Nazev...", cx));
        let isin = cx.new(|cx| TextInput::new("sec-new-isin", "ISIN...", cx));
        let transaction_type =
            cx.new(|_cx| Select::new("sec-new-tx-type", "Operace...", Self::tx_type_options()));
        let transaction_date = cx.new(|cx| DateInput::new("sec-new-date", cx));
        let quantity = cx.new(|cx| NumberInput::new("sec-new-qty", "Pocet...", cx));
        let unit_price = cx.new(|cx| NumberInput::new("sec-new-price", "0.00", cx));
        let fees = cx.new(|cx| NumberInput::new("sec-new-fees", "0.00", cx));
        let currency_code = cx.new(|cx| {
            let mut t = TextInput::new("sec-new-currency", "CZK", cx);
            t.set_value("CZK", cx);
            t
        });
        let exchange_rate =
            cx.new(|cx| NumberInput::new("sec-new-rate", "1.0000", cx).with_value("1.0000"));

        self.editing_security = Some(EditingSecurityTx {
            id: 0,
            asset_type,
            asset_name,
            isin,
            transaction_type,
            transaction_date,
            quantity,
            unit_price,
            fees,
            currency_code,
            exchange_rate,
        });
        self.error = None;
        cx.notify();
    }

    fn start_edit_security(&mut self, tx: &SecurityTransaction, cx: &mut Context<Self>) {
        let tid = tx.id;
        let asset_type = cx.new(|cx| {
            let mut s = Select::new(
                SharedString::from(format!("sec-edit-atype-{tid}")),
                "Typ...",
                Self::asset_type_options(),
            );
            s.set_selected_value(Self::asset_type_value(&tx.asset_type), cx);
            s
        });
        let asset_name = cx.new(|cx| {
            let mut t = TextInput::new(
                SharedString::from(format!("sec-edit-name-{tid}")),
                "Nazev...",
                cx,
            );
            t.set_value(&tx.asset_name, cx);
            t
        });
        let isin = cx.new(|cx| {
            let mut t = TextInput::new(
                SharedString::from(format!("sec-edit-isin-{tid}")),
                "ISIN...",
                cx,
            );
            t.set_value(&tx.isin, cx);
            t
        });
        let transaction_type = cx.new(|cx| {
            let mut s = Select::new(
                SharedString::from(format!("sec-edit-txtype-{tid}")),
                "Operace...",
                Self::tx_type_options(),
            );
            s.set_selected_value(Self::tx_type_value(&tx.transaction_type), cx);
            s
        });
        let transaction_date = cx.new(|cx| {
            let mut d = DateInput::new(SharedString::from(format!("sec-edit-date-{tid}")), cx);
            d.set_iso_value(&tx.transaction_date.format("%Y-%m-%d").to_string(), cx);
            d
        });
        let quantity = cx.new(|cx| {
            NumberInput::new(
                SharedString::from(format!("sec-edit-qty-{tid}")),
                "Pocet...",
                cx,
            )
            .with_value(Self::quantity_internal_to_display(tx.quantity))
        });
        let unit_price = cx.new(|cx| {
            let mut n = NumberInput::new(
                SharedString::from(format!("sec-edit-price-{tid}")),
                "0.00",
                cx,
            );
            n.set_amount(tx.unit_price, cx);
            n
        });
        let fees = cx.new(|cx| {
            let mut n = NumberInput::new(
                SharedString::from(format!("sec-edit-fees-{tid}")),
                "0.00",
                cx,
            );
            n.set_amount(tx.fees, cx);
            n
        });
        let currency_code = cx.new(|cx| {
            let mut t = TextInput::new(
                SharedString::from(format!("sec-edit-curr-{tid}")),
                "CZK",
                cx,
            );
            t.set_value(&tx.currency_code, cx);
            t
        });
        let exchange_rate = cx.new(|cx| {
            NumberInput::new(
                SharedString::from(format!("sec-edit-rate-{tid}")),
                "1.0000",
                cx,
            )
            .with_value(Self::exchange_rate_internal_to_display(tx.exchange_rate))
        });

        self.editing_security = Some(EditingSecurityTx {
            id: tid,
            asset_type,
            asset_name,
            isin,
            transaction_type,
            transaction_date,
            quantity,
            unit_price,
            fees,
            currency_code,
            exchange_rate,
        });
        self.error = None;
        cx.notify();
    }

    fn cancel_security_edit(&mut self, cx: &mut Context<Self>) {
        self.editing_security = None;
        self.error = None;
        cx.notify();
    }

    fn save_security(&mut self, cx: &mut Context<Self>) {
        if self.saving {
            return;
        }
        let editing = match &self.editing_security {
            Some(e) => e,
            None => return,
        };

        let asset_type_val = editing
            .asset_type
            .read(cx)
            .selected_value()
            .unwrap_or("stock")
            .to_string();
        let name = editing.asset_name.read(cx).value().to_string();
        let isin_val = editing.isin.read(cx).value().to_string();
        let tx_type_val = editing
            .transaction_type
            .read(cx)
            .selected_value()
            .unwrap_or("buy")
            .to_string();
        let date_iso = editing.transaction_date.read(cx).iso_value().to_string();
        let qty_str = editing.quantity.read(cx).value().to_string();
        let price = editing
            .unit_price
            .read(cx)
            .to_amount()
            .unwrap_or(Amount::ZERO);
        let fees_amt = editing.fees.read(cx).to_amount().unwrap_or(Amount::ZERO);
        let currency = editing.currency_code.read(cx).value().to_string();
        let rate_str = editing.exchange_rate.read(cx).value().to_string();
        let edit_id = editing.id;

        if name.trim().is_empty() {
            self.error = Some("Nazev aktiva je povinny.".into());
            cx.notify();
            return;
        }
        if date_iso.is_empty() {
            self.error = Some("Datum je povinne.".into());
            cx.notify();
            return;
        }

        let tx_date = match NaiveDate::parse_from_str(&date_iso, "%Y-%m-%d") {
            Ok(d) => d,
            Err(_) => {
                self.error = Some("Neplatny format data.".into());
                cx.notify();
                return;
            }
        };

        let quantity = Self::quantity_display_to_internal(&qty_str);
        let exchange_rate = Self::exchange_rate_display_to_internal(&rate_str);

        // Compute total_amount = quantity * unit_price / 10000
        let total_halere = (quantity as i128 * price.halere() as i128 / 10000) as i64;
        let total_amount = Amount::from_halere(total_halere);

        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.service.clone();
        let doc_service = self.document_service.clone();
        let year = self.year;
        let now = Local::now().naive_local();
        let asset_type = Self::parse_asset_type(&asset_type_val);
        let transaction_type = Self::parse_tx_type(&tx_type_val);

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    let mut tx = SecurityTransaction {
                        id: edit_id,
                        year,
                        document_id: None,
                        asset_type,
                        asset_name: name,
                        isin: isin_val,
                        transaction_type,
                        transaction_date: tx_date,
                        quantity,
                        unit_price: price,
                        total_amount,
                        fees: fees_amt,
                        currency_code: currency,
                        exchange_rate,
                        cost_basis: Amount::ZERO,
                        computed_gain: Amount::ZERO,
                        time_test_exempt: false,
                        exempt_amount: Amount::ZERO,
                        created_at: now,
                        updated_at: now,
                    };
                    if edit_id == 0 {
                        service.create_security_transaction(&mut tx)?;
                    } else {
                        service.update_security_transaction(&mut tx)?;
                    }
                    let summary = service.get_year_summary(year)?;
                    let capital = service.list_capital_entries(year)?;
                    let securities = service.list_security_transactions(year)?;
                    let documents = doc_service.list_by_year(year)?;
                    Ok::<_, zfaktury_domain::DomainError>((summary, capital, securities, documents))
                })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok((summary, capital, securities, documents)) => {
                        this.summary = Some(summary);
                        this.capital_entries = capital;
                        this.security_transactions = securities;
                        this.documents = documents;
                        this.editing_security = None;
                    }
                    Err(e) => this.error = Some(format!("Chyba pri ukladani: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    // ------------------------------------------------------------------
    // Delete
    // ------------------------------------------------------------------

    fn request_delete_capital(&mut self, id: i64, cx: &mut Context<Self>) {
        self.delete_target = Some(DeleteTarget::CapitalEntry(id));
        let dialog = cx.new(|_cx| {
            ConfirmDialog::new(
                "Smazat zaznam",
                "Opravdu chcete smazat tento kapitalovy prijem?",
                "Smazat",
            )
        });
        cx.subscribe(&dialog, Self::on_confirm_result).detach();
        self.confirm_dialog = Some(dialog);
        cx.notify();
    }

    fn request_delete_security(&mut self, id: i64, cx: &mut Context<Self>) {
        self.delete_target = Some(DeleteTarget::SecurityTx(id));
        let dialog = cx.new(|_cx| {
            ConfirmDialog::new(
                "Smazat transakci",
                "Opravdu chcete smazat tuto transakci?",
                "Smazat",
            )
        });
        cx.subscribe(&dialog, Self::on_confirm_result).detach();
        self.confirm_dialog = Some(dialog);
        cx.notify();
    }

    fn on_confirm_result(
        &mut self,
        _dialog: Entity<ConfirmDialog>,
        event: &ConfirmDialogResult,
        cx: &mut Context<Self>,
    ) {
        match event {
            ConfirmDialogResult::Confirmed => {
                if let Some(target) = self.delete_target.take() {
                    self.confirm_dialog = None;
                    match target {
                        DeleteTarget::CapitalEntry(id) => self.do_delete_capital(id, cx),
                        DeleteTarget::SecurityTx(id) => self.do_delete_security(id, cx),
                        DeleteTarget::Document(id) => self.do_delete_document(id, cx),
                    }
                }
            }
            ConfirmDialogResult::Cancelled => {
                self.delete_target = None;
                self.confirm_dialog = None;
                cx.notify();
            }
        }
    }

    fn do_delete_capital(&mut self, id: i64, cx: &mut Context<Self>) {
        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.service.clone();
        let doc_service = self.document_service.clone();
        let year = self.year;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    service.delete_capital_entry(id)?;
                    let summary = service.get_year_summary(year)?;
                    let capital = service.list_capital_entries(year)?;
                    let securities = service.list_security_transactions(year)?;
                    let documents = doc_service.list_by_year(year)?;
                    Ok::<_, zfaktury_domain::DomainError>((summary, capital, securities, documents))
                })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok((summary, capital, securities, documents)) => {
                        this.summary = Some(summary);
                        this.capital_entries = capital;
                        this.security_transactions = securities;
                        this.documents = documents;
                        if let Some(ref editing) = this.editing_capital
                            && editing.id == id
                        {
                            this.editing_capital = None;
                        }
                    }
                    Err(e) => this.error = Some(format!("Chyba pri mazani: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn do_delete_security(&mut self, id: i64, cx: &mut Context<Self>) {
        self.saving = true;
        self.error = None;
        cx.notify();

        let service = self.service.clone();
        let doc_service = self.document_service.clone();
        let year = self.year;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    service.delete_security_transaction(id)?;
                    let summary = service.get_year_summary(year)?;
                    let capital = service.list_capital_entries(year)?;
                    let securities = service.list_security_transactions(year)?;
                    let documents = doc_service.list_by_year(year)?;
                    Ok::<_, zfaktury_domain::DomainError>((summary, capital, securities, documents))
                })
                .await;

            this.update(cx, |this, cx| {
                this.saving = false;
                match result {
                    Ok((summary, capital, securities, documents)) => {
                        this.summary = Some(summary);
                        this.capital_entries = capital;
                        this.security_transactions = securities;
                        this.documents = documents;
                        if let Some(ref editing) = this.editing_security
                            && editing.id == id
                        {
                            this.editing_security = None;
                        }
                    }
                    Err(e) => this.error = Some(format!("Chyba pri mazani: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    // ------------------------------------------------------------------
    // Summary helpers
    // ------------------------------------------------------------------

    fn render_summary_field(&self, label: &str, value: Amount) -> Div {
        div()
            .flex()
            .flex_col()
            .gap(px(2.0))
            .child(
                div()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child(label.to_string()),
            )
            .child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child(format_amount(value)),
            )
    }

    fn render_summary_field_bold(&self, label: &str, value: Amount) -> Div {
        div()
            .flex()
            .flex_col()
            .gap(px(2.0))
            .child(
                div()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child(label.to_string()),
            )
            .child(
                div()
                    .text_sm()
                    .font_weight(FontWeight::BOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child(format_amount(value)),
            )
    }

    // ------------------------------------------------------------------
    // Tab bar rendering
    // ------------------------------------------------------------------

    fn render_tab_button(
        &self,
        label: &str,
        tab: InvestmentTab,
        cx: &mut Context<Self>,
    ) -> Stateful<Div> {
        let is_active = self.active_tab == tab;
        let bg = if is_active {
            ZfColors::ACCENT
        } else {
            ZfColors::SURFACE
        };
        let text_color = if is_active {
            0xffffff
        } else {
            ZfColors::TEXT_SECONDARY
        };
        let tab_id = format!(
            "inv-tab-{}",
            match tab {
                InvestmentTab::Documents => "documents",
                InvestmentTab::CapitalIncome => "capital",
                InvestmentTab::SecurityTransactions => "securities",
            }
        );

        div()
            .id(ElementId::Name(tab_id.into()))
            .px_4()
            .py_2()
            .bg(rgb(bg))
            .rounded_md()
            .text_sm()
            .font_weight(FontWeight::MEDIUM)
            .text_color(rgb(text_color))
            .cursor_pointer()
            .hover(|s| {
                s.bg(rgb(if is_active {
                    ZfColors::ACCENT_HOVER
                } else {
                    ZfColors::SURFACE_HOVER
                }))
            })
            .on_click(cx.listener(move |this, _ev: &ClickEvent, _w, cx| {
                this.active_tab = tab.clone();
                this.editing_capital = None;
                this.editing_security = None;
                this.error = None;
                cx.notify();
            }))
            .child(label.to_string())
    }

    // ------------------------------------------------------------------
    // Capital Income form row rendering
    // ------------------------------------------------------------------

    fn render_capital_form(&self, editing: &EditingCapitalEntry) -> Div {
        div()
            .flex()
            .flex_col()
            .gap_3()
            .p_4()
            .bg(rgb(ZfColors::SURFACE_HOVER))
            .border_t_1()
            .border_color(rgb(ZfColors::BORDER_SUBTLE))
            // Row 1: category, description
            .child(
                div()
                    .flex()
                    .gap_3()
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap_1()
                            .w(px(200.0))
                            .child(
                                div()
                                    .text_xs()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child("Kategorie"),
                            )
                            .child(editing.category.clone()),
                    )
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap_1()
                            .flex_1()
                            .child(
                                div()
                                    .text_xs()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child("Popis"),
                            )
                            .child(editing.description.clone()),
                    ),
            )
            // Row 2: date, gross, tax CZ, tax foreign
            .child(
                div()
                    .flex()
                    .gap_3()
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap_1()
                            .w(px(140.0))
                            .child(
                                div()
                                    .text_xs()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child("Datum"),
                            )
                            .child(editing.income_date.clone()),
                    )
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap_1()
                            .w(px(140.0))
                            .child(
                                div()
                                    .text_xs()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child("Brutto castka"),
                            )
                            .child(editing.gross_amount.clone()),
                    )
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap_1()
                            .w(px(140.0))
                            .child(
                                div()
                                    .text_xs()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child("Dan CZ"),
                            )
                            .child(editing.withheld_tax_cz.clone()),
                    )
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap_1()
                            .w(px(140.0))
                            .child(
                                div()
                                    .text_xs()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child("Dan zahranicni"),
                            )
                            .child(editing.withheld_tax_foreign.clone()),
                    ),
            )
            // Row 3: country code, needs_declaring
            .child(
                div()
                    .flex()
                    .gap_3()
                    .items_center()
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap_1()
                            .w(px(100.0))
                            .child(
                                div()
                                    .text_xs()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child("Zeme"),
                            )
                            .child(editing.country_code.clone()),
                    )
                    .child(editing.needs_declaring.clone()),
            )
    }

    // ------------------------------------------------------------------
    // Security Transaction form row rendering
    // ------------------------------------------------------------------

    fn render_security_form(&self, editing: &EditingSecurityTx) -> Div {
        div()
            .flex()
            .flex_col()
            .gap_3()
            .p_4()
            .bg(rgb(ZfColors::SURFACE_HOVER))
            .border_t_1()
            .border_color(rgb(ZfColors::BORDER_SUBTLE))
            // Row 1: asset type, name, ISIN
            .child(
                div()
                    .flex()
                    .gap_3()
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap_1()
                            .w(px(160.0))
                            .child(
                                div()
                                    .text_xs()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child("Typ aktiva"),
                            )
                            .child(editing.asset_type.clone()),
                    )
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap_1()
                            .flex_1()
                            .child(
                                div()
                                    .text_xs()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child("Nazev"),
                            )
                            .child(editing.asset_name.clone()),
                    )
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap_1()
                            .w(px(160.0))
                            .child(
                                div()
                                    .text_xs()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child("ISIN"),
                            )
                            .child(editing.isin.clone()),
                    ),
            )
            // Row 2: tx type, date, quantity, unit price
            .child(
                div()
                    .flex()
                    .gap_3()
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap_1()
                            .w(px(130.0))
                            .child(
                                div()
                                    .text_xs()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child("Operace"),
                            )
                            .child(editing.transaction_type.clone()),
                    )
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap_1()
                            .w(px(140.0))
                            .child(
                                div()
                                    .text_xs()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child("Datum"),
                            )
                            .child(editing.transaction_date.clone()),
                    )
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap_1()
                            .w(px(120.0))
                            .child(
                                div()
                                    .text_xs()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child("Pocet"),
                            )
                            .child(editing.quantity.clone()),
                    )
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap_1()
                            .w(px(140.0))
                            .child(
                                div()
                                    .text_xs()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child("Cena za kus"),
                            )
                            .child(editing.unit_price.clone()),
                    ),
            )
            // Row 3: fees, currency, exchange rate
            .child(
                div()
                    .flex()
                    .gap_3()
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap_1()
                            .w(px(120.0))
                            .child(
                                div()
                                    .text_xs()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child("Poplatky"),
                            )
                            .child(editing.fees.clone()),
                    )
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap_1()
                            .w(px(100.0))
                            .child(
                                div()
                                    .text_xs()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child("Mena"),
                            )
                            .child(editing.currency_code.clone()),
                    )
                    .child(
                        div()
                            .flex()
                            .flex_col()
                            .gap_1()
                            .w(px(140.0))
                            .child(
                                div()
                                    .text_xs()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child("Kurz"),
                            )
                            .child(editing.exchange_rate.clone()),
                    ),
            )
    }

    // ------------------------------------------------------------------
    // Render: Capital Income tab content
    // ------------------------------------------------------------------

    fn render_capital_tab(&self, cx: &mut Context<Self>) -> Div {
        let mut table = div()
            .flex()
            .flex_col()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .overflow_hidden();

        // Header with add button
        let is_new = self
            .editing_capital
            .as_ref()
            .map(|e| e.id == 0)
            .unwrap_or(false);
        let has_editing = self.editing_capital.is_some();

        table = table.child(
            div()
                .flex()
                .items_center()
                .justify_between()
                .px_4()
                .py_3()
                .border_b_1()
                .border_color(rgb(ZfColors::BORDER))
                .child(
                    div()
                        .text_sm()
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child("Kapitalove prijmy (paragraf 8)"),
                )
                .child(render_button(
                    "btn-add-capital",
                    "Pridat zaznam",
                    ButtonVariant::Primary,
                    is_new || self.saving,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.start_new_capital(cx);
                    }),
                )),
        );

        // Column headers
        table = table.child(
            div()
                .flex()
                .px_4()
                .py_2()
                .text_xs()
                .text_color(rgb(ZfColors::TEXT_MUTED))
                .border_b_1()
                .border_color(rgb(ZfColors::BORDER_SUBTLE))
                .child(div().flex_1().child("Popis"))
                .child(div().w(px(120.0)).child("Kategorie"))
                .child(div().w_24().child("Datum"))
                .child(div().w(px(100.0)).text_right().child("Brutto"))
                .child(div().w(px(100.0)).text_right().child("Dan"))
                .child(div().w(px(100.0)).text_right().child("Netto"))
                .child(div().w(px(120.0)).text_right().child("Akce")),
        );

        // New row form at top
        if let Some(ref editing) = self.editing_capital
            && editing.id == 0
        {
            table = table.child(
                div()
                    .flex()
                    .flex_col()
                    .child(self.render_capital_form(editing))
                    .child(
                        div()
                            .flex()
                            .justify_end()
                            .gap_2()
                            .px_4()
                            .py_2()
                            .bg(rgb(ZfColors::SURFACE_HOVER))
                            .border_t_1()
                            .border_color(rgb(ZfColors::BORDER_SUBTLE))
                            .child(render_button(
                                "cap-new-cancel",
                                "Zrusit",
                                ButtonVariant::Secondary,
                                self.saving,
                                false,
                                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                    this.cancel_capital_edit(cx);
                                }),
                            ))
                            .child(render_button(
                                "cap-new-save",
                                "Ulozit",
                                ButtonVariant::Primary,
                                false,
                                self.saving,
                                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                    this.save_capital(cx);
                                }),
                            )),
                    ),
            );
        }

        // Data rows
        if self.capital_entries.is_empty() && !is_new {
            table = table.child(
                div()
                    .px_4()
                    .py_4()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Zadne kapitalove prijmy pro tento rok."),
            );
        } else {
            for entry in &self.capital_entries {
                let entry_id = entry.id;
                let is_editing_this = self
                    .editing_capital
                    .as_ref()
                    .map(|e| e.id == entry_id)
                    .unwrap_or(false);

                if is_editing_this {
                    if let Some(ref editing) = self.editing_capital {
                        table = table.child(
                            div()
                                .flex()
                                .flex_col()
                                .child(self.render_capital_form(editing))
                                .child(
                                    div()
                                        .flex()
                                        .justify_end()
                                        .gap_2()
                                        .px_4()
                                        .py_2()
                                        .bg(rgb(ZfColors::SURFACE_HOVER))
                                        .border_t_1()
                                        .border_color(rgb(ZfColors::BORDER_SUBTLE))
                                        .child(render_button(
                                            SharedString::from(format!(
                                                "cap-edit-cancel-{entry_id}"
                                            )),
                                            "Zrusit",
                                            ButtonVariant::Secondary,
                                            self.saving,
                                            false,
                                            cx.listener(
                                                |this, _event: &ClickEvent, _window, cx| {
                                                    this.cancel_capital_edit(cx);
                                                },
                                            ),
                                        ))
                                        .child(render_button(
                                            SharedString::from(format!("cap-edit-save-{entry_id}")),
                                            "Ulozit",
                                            ButtonVariant::Primary,
                                            false,
                                            self.saving,
                                            cx.listener(
                                                |this, _event: &ClickEvent, _window, cx| {
                                                    this.save_capital(cx);
                                                },
                                            ),
                                        )),
                                ),
                        );
                    }
                } else {
                    let cat_label = Self::capital_category_label(&entry.category);

                    table = table.child(
                        div()
                            .flex()
                            .items_center()
                            .px_4()
                            .py_2()
                            .text_sm()
                            .border_t_1()
                            .border_color(rgb(ZfColors::BORDER_SUBTLE))
                            .hover(|s| s.bg(rgb(ZfColors::SURFACE_HOVER)))
                            .child(
                                div()
                                    .flex_1()
                                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                    .child(entry.description.clone()),
                            )
                            .child(
                                div()
                                    .w(px(120.0))
                                    .text_color(rgb(ZfColors::TEXT_MUTED))
                                    .child(cat_label),
                            )
                            .child(
                                div()
                                    .w_24()
                                    .text_color(rgb(ZfColors::TEXT_MUTED))
                                    .child(format_date(entry.income_date)),
                            )
                            .child(
                                div()
                                    .w(px(100.0))
                                    .text_right()
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child(format_amount(entry.gross_amount)),
                            )
                            .child(
                                div()
                                    .w(px(100.0))
                                    .text_right()
                                    .text_color(rgb(ZfColors::TEXT_MUTED))
                                    .child(format_amount(
                                        entry.withheld_tax_cz + entry.withheld_tax_foreign,
                                    )),
                            )
                            .child(
                                div()
                                    .w(px(100.0))
                                    .text_right()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                    .child(format_amount(entry.net_amount)),
                            )
                            .child(
                                div()
                                    .w(px(120.0))
                                    .flex()
                                    .justify_end()
                                    .gap_1()
                                    .child(render_button(
                                        SharedString::from(format!("cap-edit-{entry_id}")),
                                        "Upravit",
                                        ButtonVariant::Secondary,
                                        has_editing || self.saving,
                                        false,
                                        cx.listener(
                                            move |this, _event: &ClickEvent, _window, cx| {
                                                if let Some(entry) = this
                                                    .capital_entries
                                                    .iter()
                                                    .find(|e| e.id == entry_id)
                                                {
                                                    let entry = entry.clone();
                                                    this.start_edit_capital(&entry, cx);
                                                }
                                            },
                                        ),
                                    ))
                                    .child(render_button(
                                        SharedString::from(format!("cap-del-{entry_id}")),
                                        "Smazat",
                                        ButtonVariant::Danger,
                                        has_editing || self.saving,
                                        false,
                                        cx.listener(
                                            move |this, _event: &ClickEvent, _window, cx| {
                                                this.request_delete_capital(entry_id, cx);
                                            },
                                        ),
                                    )),
                            ),
                    );
                }
            }
        }

        table
    }

    // ------------------------------------------------------------------
    // Render: Security Transactions tab content
    // ------------------------------------------------------------------

    fn render_security_tab(&self, cx: &mut Context<Self>) -> Div {
        let mut table = div()
            .flex()
            .flex_col()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .overflow_hidden();

        // Header with add button
        let is_new = self
            .editing_security
            .as_ref()
            .map(|e| e.id == 0)
            .unwrap_or(false);
        let has_editing = self.editing_security.is_some();

        table = table.child(
            div()
                .flex()
                .items_center()
                .justify_between()
                .px_4()
                .py_3()
                .border_b_1()
                .border_color(rgb(ZfColors::BORDER))
                .child(
                    div()
                        .text_sm()
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child("Transakce s cennymi papiry (paragraf 10)"),
                )
                .child(render_button(
                    "btn-add-security",
                    "Pridat transakci",
                    ButtonVariant::Primary,
                    is_new || self.saving,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.start_new_security(cx);
                    }),
                )),
        );

        // Column headers
        table = table.child(
            div()
                .flex()
                .px_4()
                .py_2()
                .text_xs()
                .text_color(rgb(ZfColors::TEXT_MUTED))
                .border_b_1()
                .border_color(rgb(ZfColors::BORDER_SUBTLE))
                .child(div().flex_1().child("Nazev"))
                .child(div().w_20().child("Typ"))
                .child(div().w_20().child("Operace"))
                .child(div().w_24().child("Datum"))
                .child(div().w(px(80.0)).text_right().child("Pocet"))
                .child(div().w(px(100.0)).text_right().child("Cena/ks"))
                .child(div().w(px(100.0)).text_right().child("Celkem"))
                .child(div().w(px(100.0)).text_right().child("Zisk/Ztrata"))
                .child(div().w(px(120.0)).text_right().child("Akce")),
        );

        // New row form at top
        if let Some(ref editing) = self.editing_security
            && editing.id == 0
        {
            table = table.child(
                div()
                    .flex()
                    .flex_col()
                    .child(self.render_security_form(editing))
                    .child(
                        div()
                            .flex()
                            .justify_end()
                            .gap_2()
                            .px_4()
                            .py_2()
                            .bg(rgb(ZfColors::SURFACE_HOVER))
                            .border_t_1()
                            .border_color(rgb(ZfColors::BORDER_SUBTLE))
                            .child(render_button(
                                "sec-new-cancel",
                                "Zrusit",
                                ButtonVariant::Secondary,
                                self.saving,
                                false,
                                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                    this.cancel_security_edit(cx);
                                }),
                            ))
                            .child(render_button(
                                "sec-new-save",
                                "Ulozit",
                                ButtonVariant::Primary,
                                false,
                                self.saving,
                                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                    this.save_security(cx);
                                }),
                            )),
                    ),
            );
        }

        // Data rows
        if self.security_transactions.is_empty() && !is_new {
            table = table.child(
                div()
                    .px_4()
                    .py_4()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Zadne transakce pro tento rok."),
            );
        } else {
            for tx in &self.security_transactions {
                let tx_id = tx.id;
                let is_editing_this = self
                    .editing_security
                    .as_ref()
                    .map(|e| e.id == tx_id)
                    .unwrap_or(false);

                if is_editing_this {
                    if let Some(ref editing) = self.editing_security {
                        table = table.child(
                            div()
                                .flex()
                                .flex_col()
                                .child(self.render_security_form(editing))
                                .child(
                                    div()
                                        .flex()
                                        .justify_end()
                                        .gap_2()
                                        .px_4()
                                        .py_2()
                                        .bg(rgb(ZfColors::SURFACE_HOVER))
                                        .border_t_1()
                                        .border_color(rgb(ZfColors::BORDER_SUBTLE))
                                        .child(render_button(
                                            SharedString::from(format!("sec-edit-cancel-{tx_id}")),
                                            "Zrusit",
                                            ButtonVariant::Secondary,
                                            self.saving,
                                            false,
                                            cx.listener(
                                                |this, _event: &ClickEvent, _window, cx| {
                                                    this.cancel_security_edit(cx);
                                                },
                                            ),
                                        ))
                                        .child(render_button(
                                            SharedString::from(format!("sec-edit-save-{tx_id}")),
                                            "Ulozit",
                                            ButtonVariant::Primary,
                                            false,
                                            self.saving,
                                            cx.listener(
                                                |this, _event: &ClickEvent, _window, cx| {
                                                    this.save_security(cx);
                                                },
                                            ),
                                        )),
                                ),
                        );
                    }
                } else {
                    let asset_label = Self::asset_type_label(&tx.asset_type);
                    let tx_type_label = Self::tx_type_label(&tx.transaction_type);
                    let qty_display = Self::quantity_internal_to_display(tx.quantity);

                    let gain_color = if tx.computed_gain.halere() >= 0 {
                        ZfColors::STATUS_GREEN
                    } else {
                        ZfColors::STATUS_RED
                    };

                    table = table.child(
                        div()
                            .flex()
                            .items_center()
                            .px_4()
                            .py_2()
                            .text_sm()
                            .border_t_1()
                            .border_color(rgb(ZfColors::BORDER_SUBTLE))
                            .hover(|s| s.bg(rgb(ZfColors::SURFACE_HOVER)))
                            .child(
                                div()
                                    .flex_1()
                                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                    .child(tx.asset_name.clone()),
                            )
                            .child(
                                div()
                                    .w_20()
                                    .text_color(rgb(ZfColors::TEXT_MUTED))
                                    .child(asset_label),
                            )
                            .child(
                                div()
                                    .w_20()
                                    .text_color(rgb(ZfColors::TEXT_MUTED))
                                    .child(tx_type_label),
                            )
                            .child(
                                div()
                                    .w_24()
                                    .text_color(rgb(ZfColors::TEXT_MUTED))
                                    .child(format_date(tx.transaction_date)),
                            )
                            .child(
                                div()
                                    .w(px(80.0))
                                    .text_right()
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child(qty_display),
                            )
                            .child(
                                div()
                                    .w(px(100.0))
                                    .text_right()
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child(format_amount(tx.unit_price)),
                            )
                            .child(
                                div()
                                    .w(px(100.0))
                                    .text_right()
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child(format_amount(tx.total_amount)),
                            )
                            .child(
                                div()
                                    .w(px(100.0))
                                    .text_right()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(gain_color))
                                    .child(format_amount(tx.computed_gain)),
                            )
                            .child(
                                div()
                                    .w(px(120.0))
                                    .flex()
                                    .justify_end()
                                    .gap_1()
                                    .child(render_button(
                                        SharedString::from(format!("sec-edit-{tx_id}")),
                                        "Upravit",
                                        ButtonVariant::Secondary,
                                        has_editing || self.saving,
                                        false,
                                        cx.listener(
                                            move |this, _event: &ClickEvent, _window, cx| {
                                                if let Some(tx) = this
                                                    .security_transactions
                                                    .iter()
                                                    .find(|t| t.id == tx_id)
                                                {
                                                    let tx = tx.clone();
                                                    this.start_edit_security(&tx, cx);
                                                }
                                            },
                                        ),
                                    ))
                                    .child(render_button(
                                        SharedString::from(format!("sec-del-{tx_id}")),
                                        "Smazat",
                                        ButtonVariant::Danger,
                                        has_editing || self.saving,
                                        false,
                                        cx.listener(
                                            move |this, _event: &ClickEvent, _window, cx| {
                                                this.request_delete_security(tx_id, cx);
                                            },
                                        ),
                                    )),
                            ),
                    );
                }
            }
        }

        table
    }

    // ------------------------------------------------------------------
    // Render: Documents tab content
    // ------------------------------------------------------------------

    fn render_documents_tab(&self, cx: &mut Context<Self>) -> Div {
        let mut container = div().flex().flex_col().gap_4();

        // Upload section
        let upload_disabled = self.uploading || self.saving;
        container = container.child(
            div()
                .p_4()
                .bg(rgb(ZfColors::SURFACE))
                .rounded_md()
                .border_1()
                .border_color(rgb(ZfColors::BORDER))
                .flex()
                .flex_col()
                .gap_3()
                .child(
                    div()
                        .text_sm()
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child("Nahrat dokument"),
                )
                .child(
                    div()
                        .flex()
                        .items_center()
                        .gap_3()
                        .child(
                            div()
                                .flex()
                                .flex_col()
                                .gap_1()
                                .w(px(200.0))
                                .child(
                                    div()
                                        .text_xs()
                                        .font_weight(FontWeight::MEDIUM)
                                        .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                        .child("Platforma"),
                                )
                                .child(
                                    self.platform_select
                                        .as_ref()
                                        .map(|s| s.clone().into_any_element())
                                        .unwrap_or_else(|| div().into_any_element()),
                                ),
                        )
                        .child(div().pt(px(18.0)).child(render_button(
                            "btn-upload-doc",
                            if self.uploading {
                                "Nahravani..."
                            } else {
                                "Nahrat soubor"
                            },
                            ButtonVariant::Primary,
                            upload_disabled,
                            self.uploading,
                            cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                this.upload_document(cx);
                            }),
                        ))),
                ),
        );

        // Documents table
        let mut table = div()
            .flex()
            .flex_col()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .overflow_hidden();

        // Header
        table = table.child(
            div()
                .flex()
                .items_center()
                .justify_between()
                .px_4()
                .py_3()
                .border_b_1()
                .border_color(rgb(ZfColors::BORDER))
                .child(
                    div()
                        .text_sm()
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child(format!("Dokumenty ({})", self.documents.len())),
                ),
        );

        // Column headers
        table = table.child(
            div()
                .flex()
                .px_4()
                .py_2()
                .text_xs()
                .text_color(rgb(ZfColors::TEXT_MUTED))
                .border_b_1()
                .border_color(rgb(ZfColors::BORDER_SUBTLE))
                .child(div().flex_1().child("Nazev souboru"))
                .child(div().w(px(120.0)).child("Platforma"))
                .child(div().w(px(140.0)).child("Stav"))
                .child(div().w(px(120.0)).child("Datum nahrani"))
                .child(div().w(px(180.0)).text_right().child("Akce")),
        );

        // Document rows
        if self.documents.is_empty() {
            table = table.child(
                div()
                    .px_4()
                    .py_4()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Zadne dokumenty pro tento rok."),
            );
        } else {
            for doc in &self.documents {
                let doc_id = doc.id;
                let is_extracting = self.extracting_id == Some(doc_id);
                let can_extract = matches!(
                    doc.extraction_status,
                    ExtractionStatus::Pending | ExtractionStatus::Failed
                );
                let has_ocr = self.ocr_service.is_some();

                let status_label = Self::extraction_status_label(&doc.extraction_status);
                let status_color = Self::extraction_status_color(&doc.extraction_status);
                let status_bg = Self::extraction_status_bg(&doc.extraction_status);
                let platform_label = Self::platform_label(&doc.platform);

                let created = doc.created_at.format("%d.%m.%Y %H:%M").to_string();

                let mut row = div()
                    .flex()
                    .items_center()
                    .px_4()
                    .py_2()
                    .text_sm()
                    .border_t_1()
                    .border_color(rgb(ZfColors::BORDER_SUBTLE))
                    .hover(|s| s.bg(rgb(ZfColors::SURFACE_HOVER)));

                // Filename
                row = row.child(
                    div()
                        .flex_1()
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child(doc.filename.clone()),
                );

                // Platform
                row = row.child(
                    div()
                        .w(px(120.0))
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child(platform_label),
                );

                // Status badge
                row = row.child(
                    div().w(px(140.0)).child(
                        div()
                            .px_2()
                            .py(px(2.0))
                            .bg(rgb(status_bg))
                            .rounded(px(4.0))
                            .text_xs()
                            .text_color(rgb(status_color))
                            .child(status_label),
                    ),
                );

                // Created date
                row = row.child(
                    div()
                        .w(px(120.0))
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .text_xs()
                        .child(created),
                );

                // Actions
                let mut actions = div().w(px(180.0)).flex().justify_end().gap_1();

                if can_extract && has_ocr {
                    actions = actions.child(render_button(
                        SharedString::from(format!("doc-extract-{doc_id}")),
                        if is_extracting {
                            "Zpracovavam..."
                        } else {
                            "Extrahovat"
                        },
                        ButtonVariant::Secondary,
                        is_extracting || self.saving,
                        is_extracting,
                        cx.listener(move |this, _event: &ClickEvent, _window, cx| {
                            this.extract_document(doc_id, cx);
                        }),
                    ));
                }

                actions = actions.child(render_button(
                    SharedString::from(format!("doc-del-{doc_id}")),
                    "Smazat",
                    ButtonVariant::Danger,
                    is_extracting || self.saving,
                    false,
                    cx.listener(move |this, _event: &ClickEvent, _window, cx| {
                        this.request_delete_document(doc_id, cx);
                    }),
                ));

                row = row.child(actions);

                table = table.child(row);

                // Show extraction error if any
                if !doc.extraction_error.is_empty() {
                    table = table.child(
                        div()
                            .px_4()
                            .py_1()
                            .text_xs()
                            .text_color(rgb(ZfColors::STATUS_RED))
                            .bg(rgb(ZfColors::STATUS_RED_BG))
                            .child(format!("Chyba: {}", doc.extraction_error)),
                    );
                }
            }
        }

        container = container.child(table);
        container
    }
}

impl EventEmitter<NavigateEvent> for TaxInvestmentsView {}

impl Render for TaxInvestmentsView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("tax-investments-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll();

        // Header with year selector
        content = content.child(
            div()
                .flex()
                .items_center()
                .gap_3()
                .child(
                    div()
                        .text_xl()
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child("Investice"),
                )
                .child(render_button(
                    "btn-year-prev",
                    "<",
                    ButtonVariant::Secondary,
                    false,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.change_year(-1, cx);
                    }),
                ))
                .child(
                    div()
                        .px_3()
                        .py_1()
                        .bg(rgb(ZfColors::SURFACE))
                        .border_1()
                        .border_color(rgb(ZfColors::BORDER))
                        .rounded_md()
                        .text_sm()
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child(self.year.to_string()),
                )
                .child(render_button(
                    "btn-year-next",
                    ">",
                    ButtonVariant::Secondary,
                    false,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.change_year(1, cx);
                    }),
                )),
        );

        if self.loading {
            return content.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Nacitani..."),
            );
        }

        if let Some(ref error) = self.error {
            content = content.child(
                div()
                    .px_4()
                    .py_3()
                    .bg(rgb(ZfColors::STATUS_RED_BG))
                    .rounded_md()
                    .text_sm()
                    .text_color(rgb(ZfColors::STATUS_RED))
                    .child(error.clone()),
            );
        }

        // Summary card (always visible)
        if let Some(ref summary) = self.summary {
            content = content.child(
                div()
                    .p_4()
                    .bg(rgb(ZfColors::SURFACE))
                    .rounded_md()
                    .border_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .flex()
                    .flex_col()
                    .gap_3()
                    .child(
                        div()
                            .text_sm()
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                            .child("Souhrn investicnich prijmu"),
                    )
                    .child(
                        div()
                            .flex()
                            .gap_8()
                            .child(self.render_summary_field(
                                "Kapitalove prijmy (brutto)",
                                summary.capital_income_gross,
                            ))
                            .child(
                                self.render_summary_field(
                                    "Srazena dan",
                                    summary.capital_income_tax,
                                ),
                            )
                            .child(self.render_summary_field_bold(
                                "Kapitalovy prijem (netto)",
                                summary.capital_income_net,
                            )),
                    )
                    .child(div().h(px(1.0)).bg(rgb(ZfColors::BORDER)))
                    .child(
                        div()
                            .flex()
                            .gap_8()
                            .child(self.render_summary_field(
                                "Ostatni prijmy (brutto)",
                                summary.other_income_gross,
                            ))
                            .child(
                                self.render_summary_field("Naklady", summary.other_income_expenses),
                            )
                            .child(
                                self.render_summary_field(
                                    "Osvobozeno",
                                    summary.other_income_exempt,
                                ),
                            )
                            .child(self.render_summary_field_bold(
                                "Zaklad dane (p.10)",
                                summary.other_income_net,
                            )),
                    ),
            );
        }

        // Tab bar
        content = content.child(
            div()
                .flex()
                .gap_2()
                .child(self.render_tab_button("Dokumenty", InvestmentTab::Documents, cx))
                .child(self.render_tab_button(
                    "Kapitalove prijmy (p.8)",
                    InvestmentTab::CapitalIncome,
                    cx,
                ))
                .child(self.render_tab_button(
                    "Obchody s CP (p.10)",
                    InvestmentTab::SecurityTransactions,
                    cx,
                )),
        );

        // Tab content
        match self.active_tab {
            InvestmentTab::Documents => {
                content = content.child(self.render_documents_tab(cx));
            }
            InvestmentTab::CapitalIncome => {
                content = content.child(self.render_capital_tab(cx));
            }
            InvestmentTab::SecurityTransactions => {
                content = content.child(self.render_security_tab(cx));
            }
        }

        // Confirm dialog overlay
        if let Some(ref dialog) = self.confirm_dialog {
            content = content.child(dialog.clone());
        }

        content
    }
}
