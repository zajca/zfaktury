use std::path::PathBuf;
use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::document_svc::DocumentService;
use zfaktury_core::service::import_svc::ImportService;
use zfaktury_core::service::ocr_svc::OCRService;
use zfaktury_domain::{ExpenseDocument, OCRResult};

use crate::components::button::{ButtonVariant, render_button};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;

/// Maximum file size for import: 20 MB.
const MAX_FILE_SIZE: u64 = 20 * 1024 * 1024;

/// Supported file extensions for import.
const SUPPORTED_EXTENSIONS: &[&str] = &["pdf", "jpg", "jpeg", "png", "webp"];

/// View for importing expense documents (receipts, invoices).
/// Provides file selection, upload, and optional OCR processing.
pub struct ExpenseImportView {
    import_service: Arc<ImportService>,
    document_service: Arc<DocumentService>,
    ocr_service: Option<Arc<OCRService>>,
    pending_ocr_cache: Arc<std::sync::Mutex<Option<(i64, OCRResult)>>>,

    // State
    processing: bool,
    error: Option<String>,
    success_message: Option<String>,
}

impl EventEmitter<NavigateEvent> for ExpenseImportView {}

impl ExpenseImportView {
    pub fn new(
        import_service: Arc<ImportService>,
        document_service: Arc<DocumentService>,
        ocr_service: Option<Arc<OCRService>>,
        pending_ocr_cache: Arc<std::sync::Mutex<Option<(i64, OCRResult)>>>,
        _cx: &mut Context<Self>,
    ) -> Self {
        Self {
            import_service,
            document_service,
            ocr_service,
            pending_ocr_cache,
            processing: false,
            error: None,
            success_message: None,
        }
    }

    /// Trigger file selection dialog using rfd async API (xdg-desktop-portal).
    fn select_file(&mut self, cx: &mut Context<Self>) {
        if self.processing {
            return;
        }

        self.processing = true;
        self.error = None;
        self.success_message = None;
        cx.notify();

        cx.spawn(async move |this, cx| {
            let file_result = rfd::AsyncFileDialog::new()
                .add_filter("Doklady", SUPPORTED_EXTENSIONS)
                .set_title("Vyberte doklad")
                .pick_file()
                .await;

            match file_result {
                Some(handle) => {
                    let path = handle.path().to_path_buf();
                    this.update(cx, |this, cx| {
                        this.process_file(path, cx);
                    })
                    .ok();
                }
                None => {
                    // User cancelled the dialog
                    this.update(cx, |this, cx| {
                        this.processing = false;
                        cx.notify();
                    })
                    .ok();
                }
            }
        })
        .detach();
    }

    /// Process a selected file: validate, create skeleton expense, store document,
    /// optionally run OCR, then navigate to detail.
    fn process_file(&mut self, path: PathBuf, cx: &mut Context<Self>) {
        // Validate extension
        let extension = path
            .extension()
            .and_then(|e| e.to_str())
            .map(|e| e.to_lowercase())
            .unwrap_or_default();

        if !SUPPORTED_EXTENSIONS.contains(&extension.as_str()) {
            self.processing = false;
            self.error = Some(format!(
                "Nepodporovaný formát souboru: .{}. Podporované: PDF, JPG, PNG, WebP",
                extension
            ));
            cx.notify();
            return;
        }

        // Detect content type from extension
        let content_type = match extension.as_str() {
            "pdf" => "application/pdf",
            "jpg" | "jpeg" => "image/jpeg",
            "png" => "image/png",
            "webp" => "image/webp",
            _ => {
                self.processing = false;
                self.error = Some("Nepodporovaný formát souboru".to_string());
                cx.notify();
                return;
            }
        }
        .to_string();

        let filename = path
            .file_name()
            .and_then(|n| n.to_str())
            .unwrap_or("dokument")
            .to_string();

        let import_svc = self.import_service.clone();
        let doc_svc = self.document_service.clone();
        let ocr_svc = self.ocr_service.clone();

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    // 1. Read file bytes from disk
                    let data =
                        std::fs::read(&path).map_err(|e| format!("Nelze přečíst soubor: {e}"))?;

                    // 2. Validate file size
                    if data.len() as u64 > MAX_FILE_SIZE {
                        return Err(format!(
                            "Soubor je příliš velký ({:.1} MB). Maximum je 20 MB.",
                            data.len() as f64 / (1024.0 * 1024.0)
                        ));
                    }

                    // 3. Create skeleton expense
                    let expense = import_svc
                        .create_skeleton_expense(&filename)
                        .map_err(|e| format!("Chyba při vytváření nákladu: {e}"))?;

                    // 4. Store document record
                    let data_dir = doc_svc.data_dir();
                    let docs_dir = std::path::Path::new(data_dir).join("documents");
                    std::fs::create_dir_all(&docs_dir)
                        .map_err(|e| format!("Nelze vytvořit adresář pro dokumenty: {e}"))?;

                    let storage_filename = format!("{}_{}", expense.id, filename);
                    let storage_path = docs_dir.join(&storage_filename);
                    std::fs::write(&storage_path, &data)
                        .map_err(|e| format!("Nelze uložit soubor: {e}"))?;

                    let now = chrono::Local::now().naive_local();
                    let mut doc = ExpenseDocument {
                        id: 0,
                        expense_id: expense.id,
                        filename: filename.clone(),
                        content_type: content_type.clone(),
                        storage_path: storage_path.to_string_lossy().to_string(),
                        size: data.len() as i64,
                        created_at: now,
                        deleted_at: None,
                    };

                    doc_svc
                        .create_record(&mut doc)
                        .map_err(|e| format!("Chyba při ukládání dokumentu: {e}"))?;

                    // 5. Optionally run OCR
                    let mut ocr_result_out = None;
                    if let Some(ref ocr) = ocr_svc {
                        match ocr.process_bytes(&data, &content_type) {
                            Ok(result) => {
                                log::info!("OCR processing successful for expense {}", expense.id);
                                ocr_result_out = Some(result);
                            }
                            Err(e) => {
                                log::warn!("OCR processing failed for expense {}: {e}", expense.id);
                                // Continue without OCR -- not a fatal error
                            }
                        }
                    }

                    Ok::<(i64, Option<OCRResult>), String>((expense.id, ocr_result_out))
                })
                .await;

            this.update(cx, |this, cx| {
                this.processing = false;
                match result {
                    Ok((expense_id, ocr_result)) => {
                        if let Some(result) = ocr_result {
                            // Store OCR result for review view
                            if let Ok(mut lock) = this.pending_ocr_cache.lock() {
                                *lock = Some((expense_id, result));
                            }
                            // Navigate to review (OCR data available)
                            cx.emit(NavigateEvent(Route::ExpenseReview(expense_id)));
                        } else {
                            // No OCR data -- go directly to detail
                            cx.emit(NavigateEvent(Route::ExpenseDetail(expense_id)));
                        }
                    }
                    Err(e) => {
                        this.error = Some(e);
                        cx.notify();
                    }
                }
            })
            .ok();
        })
        .detach();
    }

    fn render_upload_area(&self, cx: &mut Context<Self>) -> Div {
        let mut area = div()
            .p_8()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .flex()
            .flex_col()
            .items_center()
            .gap_4();

        // Upload icon placeholder
        area = area.child(
            div()
                .text_color(rgb(ZfColors::TEXT_MUTED))
                .text_xl()
                .child("Nahrání dokladu"),
        );

        // Description
        area = area.child(
            div()
                .text_sm()
                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                .text_center()
                .child(
                    "Vyberte soubor pro import. Podporované formáty: PDF, JPG, PNG, WebP (max 20 MB)",
                ),
        );

        // Upload button or processing spinner
        if self.processing {
            area = area.child(
                div()
                    .px_4()
                    .py_2()
                    .bg(rgb(ZfColors::ACCENT))
                    .rounded_md()
                    .text_sm()
                    .font_weight(FontWeight::MEDIUM)
                    .text_color(rgb(0xffffff))
                    .opacity(0.5)
                    .child("Zpracování..."),
            );
        } else {
            area = area.child(render_button(
                "select-file-btn",
                "Vybrat soubor",
                ButtonVariant::Primary,
                false,
                false,
                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                    this.select_file(cx);
                }),
            ));
        }

        area
    }

    fn render_ocr_status(&self) -> Div {
        let (dot_color, message) = if self.ocr_service.is_some() {
            (
                ZfColors::STATUS_GREEN,
                "OCR je aktivní - data budou automaticky rozpoznána",
            )
        } else {
            (
                ZfColors::STATUS_YELLOW,
                "OCR není nastaveno - doklad bude importován bez rozpoznání",
            )
        };

        let bg_color = if self.ocr_service.is_some() {
            ZfColors::STATUS_GREEN_BG
        } else {
            ZfColors::STATUS_YELLOW_BG
        };

        div()
            .p_3()
            .bg(rgb(bg_color))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .flex()
            .items_center()
            .gap_2()
            .child(
                div()
                    .w(px(8.0))
                    .h(px(8.0))
                    .rounded_full()
                    .bg(rgb(dot_color)),
            )
            .child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child(message),
            )
    }

    fn render_info_note(&self) -> Div {
        div()
            .p_4()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .flex()
            .flex_col()
            .gap_2()
            .child(
                div()
                    .text_sm()
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child("Jak to funguje"),
            )
            .child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child("1. Vyberte soubor s dokladem (účtenka, faktura)"),
            )
            .child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child("2. Systém vytvoří náklad a uloží dokument"),
            )
            .child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child("3. Pokud je OCR aktivní, data budou automaticky rozpoznána"),
            )
            .child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child("4. Zkontrolujte a upravte údaje nákladu"),
            )
    }
}

impl Render for ExpenseImportView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("expense-import-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll();

        // Header
        outer = outer.child(
            div()
                .text_xl()
                .font_weight(FontWeight::SEMIBOLD)
                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                .child("Import dokladu"),
        );

        // Error message
        if let Some(ref error) = self.error {
            outer = outer.child(
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

        // Success message
        if let Some(ref msg) = self.success_message {
            outer = outer.child(
                div()
                    .px_4()
                    .py_3()
                    .bg(rgb(ZfColors::STATUS_GREEN_BG))
                    .rounded_md()
                    .text_sm()
                    .text_color(rgb(ZfColors::STATUS_GREEN))
                    .child(msg.clone()),
            );
        }

        // OCR status badge
        outer = outer.child(self.render_ocr_status());

        // Upload area
        outer = outer.child(self.render_upload_area(cx));

        // Info note
        outer = outer.child(self.render_info_note());

        // Back button
        outer = outer.child(div().flex().child(render_button(
            "back-btn",
            "Zpět na náklady",
            ButtonVariant::Secondary,
            self.processing,
            false,
            cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                cx.emit(NavigateEvent(Route::ExpenseList));
            }),
        )));

        outer
    }
}
