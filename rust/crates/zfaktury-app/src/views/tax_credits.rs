use gpui::*;

use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;

/// Tax credits and deductions view.
pub struct TaxCreditsView {
    year: i32,
}

impl TaxCreditsView {
    pub fn new() -> Self {
        let year = chrono::Local::now().date_naive().year();
        Self { year }
    }

    fn render_credit_card(&self, title: &str, description: &str) -> Div {
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
                    .child(title.to_string()),
            )
            .child(
                div()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child(description.to_string()),
            )
            .child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child("Zadne zaznamy pro tento rok."),
            )
    }
}

use chrono::Datelike;

impl EventEmitter<NavigateEvent> for TaxCreditsView {}

impl Render for TaxCreditsView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        div()
            .id("tax-credits-scroll")
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
                            .child("Slevy a odpocty"),
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
            // Credit cards in a 2-column grid
            .child(
                div()
                    .flex()
                    .gap_4()
                    .child(div().flex_1().child(self.render_credit_card(
                        "Sleva na manzelku/manzelku",
                        "Prijem manzelky/manzela do 68 000 Kc. ZTP/P = dvojnasobek.",
                    )))
                    .child(div().flex_1().child(self.render_credit_card(
                        "Deti",
                        "Danove zvyhodneni na vyzivanou deti. 1./2./3.+ dite.",
                    ))),
            )
            .child(
                div()
                    .flex()
                    .gap_4()
                    .child(div().flex_1().child(self.render_credit_card(
                        "Osobni slevy",
                        "Sleva na studenta, invalidni duchod.",
                    )))
                    .child(div().flex_1().child(self.render_credit_card(
                        "Odpocty",
                        "Hypoteka, zivotni pojisteni, penzijni pripojisteni, dary, odbory.",
                    ))),
            )
    }
}
