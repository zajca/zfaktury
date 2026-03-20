use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::VATReturnService;
use zfaktury_domain::VATReturn;

use crate::theme::ZfColors;
use crate::util::format::format_amount;

/// VAT overview view with year selector and returns list.
pub struct VatOverviewView {
    service: Arc<VATReturnService>,
    loading: bool,
    error: Option<String>,
    year: i32,
    returns: Vec<VATReturn>,
}

impl VatOverviewView {
    pub fn new(service: Arc<VATReturnService>, cx: &mut Context<Self>) -> Self {
        let year = chrono::Local::now().date_naive().year();
        let mut view = Self {
            service,
            loading: true,
            error: None,
            year,
            returns: Vec::new(),
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        let year = self.year;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.list(year) })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(returns) => this.returns = returns,
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani DPH priznani: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn render_quarter_card(&self, quarter: i32) -> Div {
        let months: Vec<&str> = match quarter {
            1 => vec!["Leden", "Unor", "Brezen"],
            2 => vec!["Duben", "Kveten", "Cerven"],
            3 => vec!["Cervenec", "Srpen", "Zari"],
            _ => vec!["Rijen", "Listopad", "Prosinec"],
        };

        // Check if there is a return for this quarter
        let has_return = self.returns.iter().any(|r| r.period.quarter == quarter);
        let border_color = if has_return {
            ZfColors::STATUS_GREEN
        } else {
            ZfColors::BORDER
        };

        let mut card = div()
            .flex_1()
            .p_4()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(border_color))
            .flex()
            .flex_col()
            .gap_2()
            .child(
                div()
                    .text_sm()
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child(format!("Q{}", quarter)),
            );

        for month in months {
            card = card.child(
                div()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child(month.to_string()),
            );
        }

        card
    }
}

use chrono::Datelike;

impl Render for VatOverviewView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("vat-overview-scroll")
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
                .justify_between()
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
                                .child("DPH"),
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
                        .child("Nove DPH priznani"),
                ),
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
            return content.child(
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

        // Quarter cards
        content = content.child(
            div()
                .flex()
                .gap_4()
                .child(self.render_quarter_card(1))
                .child(self.render_quarter_card(2))
                .child(self.render_quarter_card(3))
                .child(self.render_quarter_card(4)),
        );

        // Returns table
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
                .child("DPH priznani"),
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
                .child(div().w(px(80.0)).child("Obdobi"))
                .child(div().w(px(112.0)).child("Typ"))
                .child(div().w(px(112.0)).text_right().child("DPH na vystupu"))
                .child(div().w(px(112.0)).text_right().child("DPH na vstupu"))
                .child(div().w(px(112.0)).text_right().child("Vysledek"))
                .child(div().w_20().text_right().child("Stav")),
        );

        if self.returns.is_empty() {
            table = table.child(
                div()
                    .px_4()
                    .py_8()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Zadna DPH priznani pro tento rok."),
            );
        } else {
            for vr in &self.returns {
                let period_label = if vr.period.month > 0 {
                    format!("{}/{}", vr.period.month, vr.period.year)
                } else {
                    format!("Q{}/{}", vr.period.quarter, vr.period.year)
                };

                let status_text = vr.status.to_string();
                let status_color = match vr.status {
                    zfaktury_domain::FilingStatus::Draft => ZfColors::STATUS_GRAY,
                    zfaktury_domain::FilingStatus::Ready => ZfColors::STATUS_YELLOW,
                    zfaktury_domain::FilingStatus::Filed => ZfColors::STATUS_GREEN,
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
                        .cursor_pointer()
                        .hover(|s| s.bg(rgb(ZfColors::SURFACE_HOVER)))
                        .child(
                            div()
                                .w(px(80.0))
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(period_label),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(vr.filing_type.to_string()),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(format_amount(vr.total_output_vat)),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(format_amount(vr.total_input_vat)),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_right()
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(vr.net_vat)),
                        )
                        .child(
                            div().w_20().flex().justify_end().child(
                                div()
                                    .px_2()
                                    .py(px(2.0))
                                    .rounded(px(4.0))
                                    .text_xs()
                                    .text_color(rgb(status_color))
                                    .child(status_text),
                            ),
                        ),
                );
            }
        }

        content.child(table)
    }
}
