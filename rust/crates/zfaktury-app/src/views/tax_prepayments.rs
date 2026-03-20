use gpui::*;

use crate::theme::ZfColors;

/// Tax prepayments view showing monthly prepayment schedule.
pub struct TaxPrepaymentsView {
    year: i32,
}

impl TaxPrepaymentsView {
    pub fn new() -> Self {
        let year = chrono::Local::now().date_naive().year();
        Self { year }
    }
}

use chrono::Datelike;

impl Render for TaxPrepaymentsView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let months = [
            "Leden", "Unor", "Brezen", "Duben", "Kveten", "Cerven", "Cervenec", "Srpen", "Zari",
            "Rijen", "Listopad", "Prosinec",
        ];

        let mut content = div()
            .id("tax-prepayments-scroll")
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
                        .child("Zalohy"),
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

        // Table
        let mut table = div()
            .flex()
            .flex_col()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .overflow_hidden();

        // Column headers
        table = table.child(
            div()
                .flex()
                .px_4()
                .py_3()
                .text_xs()
                .font_weight(FontWeight::MEDIUM)
                .text_color(rgb(ZfColors::TEXT_MUTED))
                .border_b_1()
                .border_color(rgb(ZfColors::BORDER))
                .child(div().w(px(120.0)).child("Mesic"))
                .child(div().flex_1().text_right().child("Dan z prijmu"))
                .child(div().flex_1().text_right().child("Socialni pojisteni"))
                .child(div().flex_1().text_right().child("Zdravotni pojisteni"))
                .child(div().flex_1().text_right().child("Celkem")),
        );

        for (i, month) in months.iter().enumerate() {
            table = table.child(
                div()
                    .flex()
                    .items_center()
                    .px_4()
                    .py_2()
                    .text_sm()
                    .border_t_1()
                    .border_color(rgb(ZfColors::BORDER_SUBTLE))
                    .child(
                        div()
                            .w(px(120.0))
                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                            .child(format!("{}. {}", i + 1, month)),
                    )
                    .child(
                        div()
                            .flex_1()
                            .text_right()
                            .text_color(rgb(ZfColors::TEXT_MUTED))
                            .child("-"),
                    )
                    .child(
                        div()
                            .flex_1()
                            .text_right()
                            .text_color(rgb(ZfColors::TEXT_MUTED))
                            .child("-"),
                    )
                    .child(
                        div()
                            .flex_1()
                            .text_right()
                            .text_color(rgb(ZfColors::TEXT_MUTED))
                            .child("-"),
                    )
                    .child(
                        div()
                            .flex_1()
                            .text_right()
                            .text_color(rgb(ZfColors::TEXT_MUTED))
                            .child("-"),
                    ),
            );
        }

        // Totals row
        table = table.child(
            div()
                .flex()
                .items_center()
                .px_4()
                .py_3()
                .text_sm()
                .font_weight(FontWeight::SEMIBOLD)
                .border_t_1()
                .border_color(rgb(ZfColors::BORDER))
                .bg(rgb(ZfColors::SURFACE_HOVER))
                .child(
                    div()
                        .w(px(120.0))
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child("Celkem"),
                )
                .child(
                    div()
                        .flex_1()
                        .text_right()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child("-"),
                )
                .child(
                    div()
                        .flex_1()
                        .text_right()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child("-"),
                )
                .child(
                    div()
                        .flex_1()
                        .text_right()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child("-"),
                )
                .child(
                    div()
                        .flex_1()
                        .text_right()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child("-"),
                ),
        );

        content.child(table)
    }
}
