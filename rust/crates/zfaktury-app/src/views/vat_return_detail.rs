use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::VATReturnService;
use zfaktury_domain::VATReturn;

use crate::theme::ZfColors;
use crate::util::format::format_amount;

/// VAT return detail view.
pub struct VatReturnDetailView {
    service: Arc<VATReturnService>,
    return_id: i64,
    loading: bool,
    error: Option<String>,
    vat_return: Option<VATReturn>,
}

impl VatReturnDetailView {
    pub fn new(service: Arc<VATReturnService>, return_id: i64, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            service,
            return_id,
            loading: true,
            error: None,
            vat_return: None,
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        let id = self.return_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.get_by_id(id) })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(vr) => this.vat_return = Some(vr),
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

    fn render_amount_row(&self, label: &str, value: zfaktury_domain::Amount) -> Div {
        div()
            .flex()
            .justify_between()
            .text_sm()
            .child(
                div()
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child(label.to_string()),
            )
            .child(
                div()
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child(format_amount(value)),
            )
    }

    fn render_vat_content(&self, vr: &VATReturn) -> Div {
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

        let mut content = div().flex().flex_col().gap_6();

        // Header
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
                                .child(format!("DPH priznani {}", period_label)),
                        )
                        .child(
                            div()
                                .px_2()
                                .py(px(2.0))
                                .rounded(px(4.0))
                                .text_xs()
                                .text_color(rgb(status_color))
                                .child(status_text),
                        ),
                )
                .child(
                    div()
                        .flex()
                        .gap_2()
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
                                .child("Prepocitat"),
                        )
                        .child(
                            div()
                                .px_4()
                                .py_2()
                                .bg(rgb(ZfColors::SURFACE))
                                .border_1()
                                .border_color(rgb(ZfColors::BORDER))
                                .rounded_md()
                                .text_sm()
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .cursor_pointer()
                                .hover(|s| s.bg(rgb(ZfColors::SURFACE_HOVER)))
                                .child("Generovat XML"),
                        )
                        .child(
                            div()
                                .px_4()
                                .py_2()
                                .bg(rgb(ZfColors::STATUS_GREEN_BG))
                                .rounded_md()
                                .text_sm()
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::STATUS_GREEN))
                                .cursor_pointer()
                                .child("Oznacit jako podane"),
                        ),
                ),
        );

        // Info row
        content = content.child(
            div()
                .p_4()
                .bg(rgb(ZfColors::SURFACE))
                .rounded_md()
                .border_1()
                .border_color(rgb(ZfColors::BORDER))
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
                                .child("Obdobi"),
                        )
                        .child(
                            div()
                                .text_sm()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(period_label.clone()),
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
                                .child("Typ podani"),
                        )
                        .child(
                            div()
                                .text_sm()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(vr.filing_type.to_string()),
                        ),
                ),
        );

        // Output VAT section
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
                        .child("Vystupni DPH (dan na vystupu)"),
                )
                .child(self.render_amount_row("Zaklad 21%", vr.output_vat_base_21))
                .child(self.render_amount_row("DPH 21%", vr.output_vat_amount_21))
                .child(self.render_amount_row("Zaklad 12%", vr.output_vat_base_12))
                .child(self.render_amount_row("DPH 12%", vr.output_vat_amount_12))
                .child(self.render_amount_row("Zaklad 0%", vr.output_vat_base_0))
                .child(div().h(px(1.0)).bg(rgb(ZfColors::BORDER)))
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .child(
                            div()
                                .text_sm()
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child("Celkem vystupni DPH"),
                        )
                        .child(
                            div()
                                .text_sm()
                                .font_weight(FontWeight::BOLD)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(vr.total_output_vat)),
                        ),
                ),
        );

        // Input VAT section
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
                        .child("Vstupni DPH (dan na vstupu)"),
                )
                .child(self.render_amount_row("Zaklad 21%", vr.input_vat_base_21))
                .child(self.render_amount_row("DPH 21%", vr.input_vat_amount_21))
                .child(self.render_amount_row("Zaklad 12%", vr.input_vat_base_12))
                .child(self.render_amount_row("DPH 12%", vr.input_vat_amount_12))
                .child(div().h(px(1.0)).bg(rgb(ZfColors::BORDER)))
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .child(
                            div()
                                .text_sm()
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child("Celkem vstupni DPH"),
                        )
                        .child(
                            div()
                                .text_sm()
                                .font_weight(FontWeight::BOLD)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(vr.total_input_vat)),
                        ),
                ),
        );

        // Summary
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
                        .child("Souhrn"),
                )
                .child(self.render_amount_row("Vystupni DPH", vr.total_output_vat))
                .child(self.render_amount_row("Vstupni DPH", vr.total_input_vat))
                .child(div().h(px(1.0)).bg(rgb(ZfColors::BORDER)))
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .child(
                            div()
                                .text_sm()
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child("Vysledna danova povinnost"),
                        )
                        .child(
                            div()
                                .text_lg()
                                .font_weight(FontWeight::BOLD)
                                .text_color(rgb(ZfColors::ACCENT))
                                .child(format_amount(vr.net_vat)),
                        ),
                ),
        );

        content
    }
}

impl Render for VatReturnDetailView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("vat-return-detail-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .overflow_y_scroll();

        if self.loading {
            return outer.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Nacitani DPH priznani..."),
            );
        }

        if let Some(ref error) = self.error {
            return outer.child(
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

        if let Some(ref vr) = self.vat_return {
            outer = outer.child(self.render_vat_content(vr));
        }

        outer
    }
}
