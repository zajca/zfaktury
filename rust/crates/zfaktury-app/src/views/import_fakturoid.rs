use gpui::*;

use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;

/// Import from Fakturoid view.
pub struct ImportFakturoidView;

impl ImportFakturoidView {
    pub fn new() -> Self {
        Self
    }
}

impl EventEmitter<NavigateEvent> for ImportFakturoidView {}

impl Render for ImportFakturoidView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        div()
            .id("import-fakturoid-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll()
            // Header
            .child(
                div()
                    .text_xl()
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child("Import z Fakturoid"),
            )
            // Status card
            .child(
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
                            .child("Stav importu"),
                    )
                    .child(
                        div()
                            .flex()
                            .gap_8()
                            .child(
                                div()
                                    .flex()
                                    .flex_col()
                                    .gap(px(2.0))
                                    .child(
                                        div()
                                            .text_xs()
                                            .text_color(rgb(ZfColors::TEXT_MUTED))
                                            .child("Posledni import"),
                                    )
                                    .child(
                                        div()
                                            .text_sm()
                                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                            .child("Zadny import nebyl proveden"),
                                    ),
                            )
                            .child(
                                div()
                                    .flex()
                                    .flex_col()
                                    .gap(px(2.0))
                                    .child(
                                        div()
                                            .text_xs()
                                            .text_color(rgb(ZfColors::TEXT_MUTED))
                                            .child("Importovane faktury"),
                                    )
                                    .child(
                                        div()
                                            .text_sm()
                                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                            .child("0"),
                                    ),
                            )
                            .child(
                                div()
                                    .flex()
                                    .flex_col()
                                    .gap(px(2.0))
                                    .child(
                                        div()
                                            .text_xs()
                                            .text_color(rgb(ZfColors::TEXT_MUTED))
                                            .child("Importovane kontakty"),
                                    )
                                    .child(
                                        div()
                                            .text_sm()
                                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                            .child("0"),
                                    ),
                            ),
                    ),
            )
            // Action
            .child(
                div().flex().child(
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
                        .child("Spustit import"),
                ),
            )
            // Info note
            .child(
                div()
                    .p_4()
                    .bg(rgb(ZfColors::SURFACE))
                    .rounded_md()
                    .border_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Pro import z Fakturoid je nutne nastavit API klic v config.toml."),
            )
    }
}
