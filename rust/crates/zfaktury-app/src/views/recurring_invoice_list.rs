use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::RecurringInvoiceService;
use zfaktury_domain::RecurringInvoice;

use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;
use crate::util::format::{format_amount, format_date};

/// Recurring invoice list view.
pub struct RecurringInvoiceListView {
    service: Arc<RecurringInvoiceService>,
    loading: bool,
    error: Option<String>,
    items: Vec<RecurringInvoice>,
}

impl RecurringInvoiceListView {
    pub fn new(service: Arc<RecurringInvoiceService>, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            service,
            loading: true,
            error: None,
            items: Vec::new(),
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.list() })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(items) => this.items = items,
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani sablon faktur: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn total_amount(ri: &RecurringInvoice) -> zfaktury_domain::Amount {
        use zfaktury_domain::Amount;
        let mut total = Amount::ZERO;
        for item in &ri.items {
            let base = Amount::from_halere(item.quantity.halere() * item.unit_price.halere() / 100);
            let vat = base.multiply(item.vat_rate_percent as f64 / 100.0);
            total += base + vat;
        }
        total
    }
}

impl EventEmitter<NavigateEvent> for RecurringInvoiceListView {}

impl Render for RecurringInvoiceListView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("recurring-invoice-list-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_4()
            .overflow_y_scroll();

        // Header
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
                        .child("Sablony faktur"),
                )
                .child(
                    div()
                        .text_sm()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child(format!("({} celkem)", self.items.len())),
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
                .child(div().flex_1().child("Nazev"))
                .child(div().w(px(150.0)).child("Zakaznik"))
                .child(div().w(px(112.0)).text_right().child("Castka"))
                .child(div().w_24().child("Frekvence"))
                .child(div().w(px(112.0)).child("Dalsi vystaveni"))
                .child(div().w_20().text_right().child("Stav")),
        );

        if self.items.is_empty() {
            table = table.child(
                div()
                    .px_4()
                    .py_8()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Zadne sablony faktur."),
            );
        } else {
            for ri in &self.items {
                let customer_name = ri
                    .customer
                    .as_ref()
                    .map(|c| c.name.clone())
                    .unwrap_or_else(|| format!("ID {}", ri.customer_id));

                let total = Self::total_amount(ri);

                let status_text = if ri.is_active { "Aktivni" } else { "Neaktivni" };
                let status_color = if ri.is_active {
                    ZfColors::STATUS_GREEN
                } else {
                    ZfColors::STATUS_GRAY
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
                                .flex_1()
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(ri.name.clone()),
                        )
                        .child(
                            div()
                                .w(px(150.0))
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(customer_name),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_right()
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(total)),
                        )
                        .child(
                            div()
                                .w_24()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(ri.frequency.to_string()),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format_date(ri.next_issue_date)),
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
