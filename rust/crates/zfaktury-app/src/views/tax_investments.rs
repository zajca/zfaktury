use gpui::*;

use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;

/// Tax investments view showing investment income summary.
pub struct TaxInvestmentsView {
    year: i32,
}

impl TaxInvestmentsView {
    pub fn new() -> Self {
        let year = chrono::Local::now().date_naive().year();
        Self { year }
    }
}

use chrono::Datelike;

impl EventEmitter<NavigateEvent> for TaxInvestmentsView {}

impl Render for TaxInvestmentsView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
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
        );

        // Summary card
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
                        .child(
                            div()
                                .flex()
                                .flex_col()
                                .gap(px(2.0))
                                .child(
                                    div()
                                        .text_xs()
                                        .text_color(rgb(ZfColors::TEXT_MUTED))
                                        .child("Celkovy prijem"),
                                )
                                .child(
                                    div()
                                        .text_sm()
                                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                        .child("-"),
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
                                        .child("Naklady"),
                                )
                                .child(
                                    div()
                                        .text_sm()
                                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                        .child("-"),
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
                                        .child("Zaklad dane"),
                                )
                                .child(
                                    div()
                                        .text_sm()
                                        .font_weight(FontWeight::BOLD)
                                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                        .child("-"),
                                ),
                        ),
                ),
        );

        // Transactions table
        let mut table = div()
            .flex()
            .flex_col()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .overflow_hidden();

        table = table.child(
            div()
                .px_4()
                .py_3()
                .border_b_1()
                .border_color(rgb(ZfColors::BORDER))
                .text_sm()
                .font_weight(FontWeight::SEMIBOLD)
                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                .child("Transakce s cennymi papiry"),
        );

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
                .child(div().w(px(112.0)).text_right().child("Castka"))
                .child(div().w(px(112.0)).text_right().child("Zisk/Ztrata")),
        );

        table = table.child(
            div()
                .px_4()
                .py_8()
                .text_sm()
                .text_color(rgb(ZfColors::TEXT_MUTED))
                .child("Zadne transakce pro tento rok."),
        );

        content.child(table)
    }
}
