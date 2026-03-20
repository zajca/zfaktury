use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::InvoiceService;
use zfaktury_domain::Invoice;

use crate::components::status_badge::render_status_badge;
use crate::theme::ZfColors;
use crate::util::format::{format_amount, format_date, format_number};

/// Invoice detail view displaying all invoice data.
pub struct InvoiceDetailView {
    service: Arc<InvoiceService>,
    invoice_id: i64,
    loading: bool,
    error: Option<String>,
    invoice: Option<Invoice>,
}

impl InvoiceDetailView {
    pub fn new(service: Arc<InvoiceService>, invoice_id: i64, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            service,
            invoice_id,
            loading: true,
            error: None,
            invoice: None,
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        let id = self.invoice_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.get_by_id(id) })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(invoice) => this.invoice = Some(invoice),
                    Err(e) => this.error = Some(format!("Chyba pri nacitani faktury: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn render_field(&self, label: &str, value: String) -> Div {
        div()
            .flex()
            .flex_col()
            .gap(px(2.0))
            .child(
                div()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child(label.to_string()),
            )
            .child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child(value),
            )
    }

    fn render_invoice_content(&self, inv: &Invoice) -> Div {
        let customer_name = inv
            .customer
            .as_ref()
            .map(|c| c.name.clone())
            .unwrap_or_else(|| format!("ID {}", inv.customer_id));

        let mut content = div().flex().flex_col().gap_6();

        // Header with number and status
        content = content.child(
            div()
                .flex()
                .items_center()
                .justify_between()
                .child(
                    div()
                        .text_xl()
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child(format!("Faktura {}", inv.invoice_number)),
                )
                .child(render_status_badge(&inv.status)),
        );

        // Info grid
        content = content.child(
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
                        .flex()
                        .gap_8()
                        .child(self.render_field("Zakaznik", customer_name))
                        .child(self.render_field("Typ", inv.invoice_type.to_string()))
                        .child(self.render_field("Mena", inv.currency_code.clone())),
                )
                .child(
                    div()
                        .flex()
                        .gap_8()
                        .child(self.render_field("Datum vystaveni", format_date(inv.issue_date)))
                        .child(self.render_field("Datum splatnosti", format_date(inv.due_date)))
                        .child(self.render_field(
                            "Datum zdanitelneho plneni",
                            format_date(inv.delivery_date),
                        )),
                )
                .child(
                    div()
                        .flex()
                        .gap_8()
                        .child(self.render_field("Variabilni symbol", inv.variable_symbol.clone()))
                        .child(self.render_field("Zpusob platby", inv.payment_method.clone())),
                ),
        );

        // Items table
        if !inv.items.is_empty() {
            let mut items_table = div()
                .flex()
                .flex_col()
                .bg(rgb(ZfColors::SURFACE))
                .rounded_md()
                .border_1()
                .border_color(rgb(ZfColors::BORDER))
                .overflow_hidden();

            items_table = items_table.child(
                div()
                    .px_4()
                    .py_3()
                    .border_b_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .text_sm()
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child("Polozky"),
            );

            // Column headers
            items_table = items_table.child(
                div()
                    .flex()
                    .px_4()
                    .py_2()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .border_b_1()
                    .border_color(rgb(ZfColors::BORDER_SUBTLE))
                    .child(div().flex_1().child("Popis"))
                    .child(div().w_20().text_right().child("Mnozstvi"))
                    .child(div().w_16().text_right().child("Jednotka"))
                    .child(div().w_24().text_right().child("Cena/ks"))
                    .child(div().w_16().text_right().child("DPH %"))
                    .child(div().w_24().text_right().child("DPH"))
                    .child(div().w(px(112.0)).text_right().child("Celkem")),
            );

            for item in &inv.items {
                items_table = items_table.child(
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
                                .flex_1()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(item.description.clone()),
                        )
                        .child(
                            div()
                                .w_20()
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(format_number(item.quantity)),
                        )
                        .child(
                            div()
                                .w_16()
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(item.unit.clone()),
                        )
                        .child(
                            div()
                                .w_24()
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(format_amount(item.unit_price)),
                        )
                        .child(
                            div()
                                .w_16()
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format!("{}%", item.vat_rate_percent)),
                        )
                        .child(
                            div()
                                .w_24()
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(format_amount(item.vat_amount)),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_right()
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(item.total_amount)),
                        ),
                );
            }

            content = content.child(items_table);
        }

        // Totals
        content = content.child(
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
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child("Zaklad dane"),
                        )
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(inv.subtotal_amount)),
                        ),
                )
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(div().text_color(rgb(ZfColors::TEXT_SECONDARY)).child("DPH"))
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(inv.vat_amount)),
                        ),
                )
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
                                .child("Celkem"),
                        )
                        .child(
                            div()
                                .text_lg()
                                .font_weight(FontWeight::BOLD)
                                .text_color(rgb(ZfColors::ACCENT))
                                .child(format_amount(inv.total_amount)),
                        ),
                ),
        );

        // Notes
        if !inv.notes.is_empty() {
            content = content.child(
                div()
                    .p_4()
                    .bg(rgb(ZfColors::SURFACE))
                    .rounded_md()
                    .border_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .flex()
                    .flex_col()
                    .gap_1()
                    .child(
                        div()
                            .text_xs()
                            .text_color(rgb(ZfColors::TEXT_MUTED))
                            .child("Poznamky"),
                    )
                    .child(
                        div()
                            .text_sm()
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .child(inv.notes.clone()),
                    ),
            );
        }

        content
    }
}

impl Render for InvoiceDetailView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("invoice-detail-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .overflow_y_scroll();

        if self.loading {
            return outer.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Nacitani faktury..."),
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

        if let Some(ref inv) = self.invoice {
            outer = outer.child(self.render_invoice_content(inv));
        }

        outer
    }
}
