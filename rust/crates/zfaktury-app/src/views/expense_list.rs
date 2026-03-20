use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::ExpenseService;
use zfaktury_domain::{Expense, ExpenseFilter};

use crate::theme::ZfColors;
use crate::util::format::{format_amount, format_date};

/// Expense list view with table.
pub struct ExpenseListView {
    service: Arc<ExpenseService>,
    loading: bool,
    error: Option<String>,
    expenses: Vec<Expense>,
    total: i64,
}

impl ExpenseListView {
    pub fn new(service: Arc<ExpenseService>, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            service,
            loading: true,
            error: None,
            expenses: Vec::new(),
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
                    service.list(ExpenseFilter {
                        limit: 50,
                        ..Default::default()
                    })
                })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok((expenses, total)) => {
                        this.expenses = expenses;
                        this.total = total;
                    }
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani nakladu: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }
}

impl Render for ExpenseListView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("expense-list-scroll")
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
                        .child("Naklady"),
                )
                .child(
                    div()
                        .text_sm()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child(format!("({} celkem)", self.total)),
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
                .child(div().w_24().child("Cislo"))
                .child(div().flex_1().child("Popis"))
                .child(div().w(px(112.0)).child("Kategorie"))
                .child(div().w(px(112.0)).child("Datum"))
                .child(div().w_16().text_right().child("DPH %"))
                .child(div().w(px(112.0)).text_right().child("Castka")),
        );

        if self.expenses.is_empty() {
            table = table.child(
                div()
                    .px_4()
                    .py_8()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Zadne naklady."),
            );
        } else {
            for exp in &self.expenses {
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
                                .w_24()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(exp.expense_number.clone()),
                        )
                        .child(
                            div()
                                .flex_1()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(exp.description.clone()),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(exp.category.clone()),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format_date(exp.issue_date)),
                        )
                        .child(
                            div()
                                .w_16()
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format!("{}%", exp.vat_rate_percent)),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_right()
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(exp.amount)),
                        ),
                );
            }
        }

        content.child(table)
    }
}
