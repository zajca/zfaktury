use gpui::*;

use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;

/// Invoice creation/edit form view.
/// Currently a placeholder with the form layout structure.
pub struct InvoiceFormView {
    is_edit: bool,
    invoice_id: Option<i64>,
}

impl InvoiceFormView {
    pub fn new_create() -> Self {
        Self {
            is_edit: false,
            invoice_id: None,
        }
    }

    pub fn new_edit(id: i64) -> Self {
        Self {
            is_edit: true,
            invoice_id: Some(id),
        }
    }

    fn render_form_field(&self, label: &str, placeholder: &str) -> Div {
        div()
            .flex()
            .flex_col()
            .gap_1()
            .child(
                div()
                    .text_xs()
                    .font_weight(FontWeight::MEDIUM)
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child(label.to_string()),
            )
            .child(
                div()
                    .px_3()
                    .py_2()
                    .bg(rgb(ZfColors::SURFACE))
                    .border_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .rounded_md()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child(placeholder.to_string()),
            )
    }
}

impl EventEmitter<NavigateEvent> for InvoiceFormView {}

impl Render for InvoiceFormView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let title = if self.is_edit {
            format!("Upravit fakturu #{}", self.invoice_id.unwrap_or_default())
        } else {
            "Nova faktura".to_string()
        };

        div()
            .id("invoice-form-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll()
            // Title
            .child(
                div()
                    .text_xl()
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child(title),
            )
            // Form section: Customer & dates
            .child(
                div()
                    .p_4()
                    .bg(rgb(ZfColors::SURFACE))
                    .rounded_md()
                    .border_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .flex()
                    .flex_col()
                    .gap_4()
                    .child(
                        div()
                            .text_sm()
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                            .child("Zakladni udaje"),
                    )
                    .child(
                        div()
                            .flex()
                            .gap_4()
                            .child(
                                div().flex_1().child(
                                    self.render_form_field("Zakaznik", "Vyberte zakaznika..."),
                                ),
                            )
                            .child(self.render_form_field("Typ", "Faktura"))
                            .child(self.render_form_field("Mena", "CZK")),
                    )
                    .child(
                        div()
                            .flex()
                            .gap_4()
                            .child(self.render_form_field("Datum vystaveni", "dd.mm.yyyy"))
                            .child(self.render_form_field("Datum splatnosti", "dd.mm.yyyy"))
                            .child(
                                self.render_form_field("Datum zdanitelneho plneni", "dd.mm.yyyy"),
                            ),
                    ),
            )
            // Items section placeholder
            .child(
                div()
                    .p_4()
                    .bg(rgb(ZfColors::SURFACE))
                    .rounded_md()
                    .border_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .flex()
                    .flex_col()
                    .gap_4()
                    .child(
                        div()
                            .text_sm()
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                            .child("Polozky faktury"),
                    )
                    .child(
                        div()
                            .text_sm()
                            .text_color(rgb(ZfColors::TEXT_MUTED))
                            .child("Editor polozek bude implementovan v dalsi fazi."),
                    ),
            )
            // Notes
            .child(
                div()
                    .p_4()
                    .bg(rgb(ZfColors::SURFACE))
                    .rounded_md()
                    .border_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .flex()
                    .flex_col()
                    .gap_4()
                    .child(self.render_form_field("Poznamky", "Volitelne poznamky..."))
                    .child(self.render_form_field("Interni poznamky", "Pouze pro vas...")),
            )
            // Save button
            .child(
                div().flex().justify_end().child(
                    div()
                        .px_4()
                        .py_2()
                        .bg(rgb(ZfColors::ACCENT))
                        .rounded_md()
                        .text_sm()
                        .font_weight(FontWeight::MEDIUM)
                        .text_color(rgb(0xffffff))
                        .cursor_pointer()
                        .hover(|s| s.bg(rgb(ZfColors::ACCENT_HOVER)))
                        .child("Ulozit fakturu"),
                ),
            )
    }
}
