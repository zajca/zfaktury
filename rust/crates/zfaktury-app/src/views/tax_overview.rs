use gpui::*;

use crate::theme::ZfColors;

/// Tax overview view showing income tax, social insurance, health insurance cards.
pub struct TaxOverviewView {
    year: i32,
}

impl TaxOverviewView {
    pub fn new() -> Self {
        let year = chrono::Local::now().date_naive().year();
        Self { year }
    }

    fn render_tax_card(&self, title: &str, subtitle: &str, status: &str, status_color: u32) -> Div {
        div()
            .flex_1()
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
                    .flex()
                    .items_center()
                    .justify_between()
                    .child(
                        div()
                            .text_sm()
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                            .child(title.to_string()),
                    )
                    .child(
                        div()
                            .px_2()
                            .py(px(2.0))
                            .rounded(px(4.0))
                            .text_xs()
                            .text_color(rgb(status_color))
                            .child(status.to_string()),
                    ),
            )
            .child(
                div()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child(subtitle.to_string()),
            )
            .child(
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
                    .child("Vytvorit"),
            )
    }
}

use chrono::Datelike;

impl Render for TaxOverviewView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        div()
            .id("tax-overview-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll()
            // Header with year selector
            .child(
                div()
                    .flex()
                    .items_center()
                    .gap_3()
                    .child(
                        div()
                            .text_xl()
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                            .child("Dan z prijmu"),
                    )
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
                    ),
            )
            // Tax cards
            .child(
                div()
                    .flex()
                    .gap_4()
                    .child(self.render_tax_card(
                        "Dan z prijmu",
                        "Danove priznani fyzickych osob (DPFO)",
                        "Koncept",
                        ZfColors::STATUS_GRAY,
                    ))
                    .child(self.render_tax_card(
                        "Socialni pojisteni",
                        "Prehled OSVC pro CSSZ",
                        "Koncept",
                        ZfColors::STATUS_GRAY,
                    ))
                    .child(self.render_tax_card(
                        "Zdravotni pojisteni",
                        "Prehled OSVC pro ZP",
                        "Koncept",
                        ZfColors::STATUS_GRAY,
                    )),
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
                    .child("Vytvorte danove priznani pro automaticky vypocet dane z prijmu, socialniho a zdravotniho pojisteni."),
            )
    }
}
