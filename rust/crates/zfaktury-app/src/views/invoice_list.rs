use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::InvoiceService;
use zfaktury_domain::{Invoice, InvoiceFilter};

use crate::components::status_badge::render_status_badge;
use crate::theme::ZfColors;
use crate::util::format::{format_amount, format_date};

/// Invoice list view with table.
pub struct InvoiceListView {
    service: Arc<InvoiceService>,
    loading: bool,
    error: Option<String>,
    invoices: Vec<Invoice>,
    total: i64,
}

impl InvoiceListView {
    pub fn new(service: Arc<InvoiceService>, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            service,
            loading: true,
            error: None,
            invoices: Vec::new(),
            total: 0,
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    service.list(InvoiceFilter {
                        limit: 50,
                        ..Default::default()
                    })
                })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok((invoices, total)) => {
                        this.invoices = invoices;
                        this.total = total;
                    }
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani faktur: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }
}

impl Render for InvoiceListView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("invoice-list-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_4()
            .overflow_y_scroll();

        // Header
        content = content.child(
            div().flex().items_center().justify_between().child(
                div()
                    .flex()
                    .items_center()
                    .gap_3()
                    .child(
                        div()
                            .text_xl()
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                            .child("Faktury"),
                    )
                    .child(
                        div()
                            .text_sm()
                            .text_color(rgb(ZfColors::TEXT_MUTED))
                            .child(format!("({} celkem)", self.total)),
                    ),
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
                .child(div().w(px(112.0)).child("Cislo"))
                .child(div().flex_1().child("Zakaznik"))
                .child(div().w(px(112.0)).child("Datum vystaveni"))
                .child(div().w(px(112.0)).child("Splatnost"))
                .child(div().w(px(112.0)).text_right().child("Castka"))
                .child(div().w(px(112.0)).text_right().child("Stav")),
        );

        if self.invoices.is_empty() {
            table = table.child(
                div()
                    .px_4()
                    .py_8()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Zadne faktury. Vytvorte novou fakturu."),
            );
        } else {
            for inv in &self.invoices {
                let customer_name = inv
                    .customer
                    .as_ref()
                    .map(|c| c.name.clone())
                    .unwrap_or_else(|| format!("ID {}", inv.customer_id));

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
                                .w(px(112.0))
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(inv.invoice_number.clone()),
                        )
                        .child(
                            div()
                                .flex_1()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(customer_name),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format_date(inv.issue_date)),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format_date(inv.due_date)),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_right()
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(inv.total_amount)),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .flex()
                                .justify_end()
                                .child(render_status_badge(&inv.status)),
                        ),
                );
            }
        }

        content.child(table)
    }
}
