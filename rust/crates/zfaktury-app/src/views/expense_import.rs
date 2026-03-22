use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::document_svc::DocumentService;
use zfaktury_core::service::import_svc::ImportService;

use crate::components::button::{ButtonVariant, render_button};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;

/// View for importing expense documents (receipts, invoices).
/// Provides file selection, upload, and optional OCR processing.
pub struct ExpenseImportView {
    import_service: Arc<ImportService>,
    document_service: Arc<DocumentService>,

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
        _cx: &mut Context<Self>,
    ) -> Self {
        Self {
            import_service,
            document_service,
            processing: false,
            error: None,
            success_message: None,
        }
    }

    /// Trigger file selection dialog.
    /// Currently a placeholder -- requires `rfd` crate to be added to Cargo.toml.
    fn select_file(&mut self, cx: &mut Context<Self>) {
        if self.processing {
            return;
        }

        // rfd is not yet in Cargo.toml, so we cannot open a native file dialog.
        // Once rfd is added, this will use rfd::AsyncFileDialog to pick a file,
        // then call process_file() with the selected path.
        self.error =
            Some("Vyber souboru bude dostupny po pridani rfd zavislosti do Cargo.toml".to_string());
        cx.notify();
    }

    /// Process a selected file: create skeleton expense, store document, optionally run OCR.
    #[allow(dead_code)]
    fn process_file(&mut self, filename: String, _data: Vec<u8>, cx: &mut Context<Self>) {
        self.processing = true;
        self.error = None;
        self.success_message = None;
        cx.notify();

        let import_svc = self.import_service.clone();
        let _doc_svc = self.document_service.clone();

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { import_svc.create_skeleton_expense(&filename) })
                .await;

            this.update(cx, |this, cx| {
                this.processing = false;
                match result {
                    Ok(expense) => {
                        this.success_message =
                            Some(format!("Naklad #{} vytvoren", expense.expense_number));
                        // Navigate to expense detail for manual editing
                        cx.emit(NavigateEvent(Route::ExpenseDetail(expense.id)));
                    }
                    Err(e) => {
                        this.error = Some(format!("Chyba pri importu: {e}"));
                    }
                }
                cx.notify();
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
                .child("Nahrani dokladu"),
        );

        // Description
        area = area.child(
            div()
                .text_sm()
                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                .text_center()
                .child("Vyberte soubor pro import. Podporovane formaty: PDF, JPG, PNG (max 20 MB)"),
        );

        // Upload button
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
                    .child("Zpracovani..."),
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
        // OCR is not configured yet (no OCR service wired in AppServices)
        div()
            .p_3()
            .bg(rgb(ZfColors::STATUS_YELLOW_BG))
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
                    .bg(rgb(ZfColors::STATUS_YELLOW)),
            )
            .child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child("OCR neni nastaveno - doklad bude importovan bez rozpoznani"),
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
                    .child("1. Vyberte soubor s dokladem (uctenka, faktura)"),
            )
            .child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child("2. System vytvori naklad a ulozi dokument"),
            )
            .child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child("3. Pokud je OCR aktivni, data budou automaticky rozpoznana"),
            )
            .child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child("4. Zkontrolujte a upravte udaje nakladu"),
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
            "Zpet na naklady",
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
