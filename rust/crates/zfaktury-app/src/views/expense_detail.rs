use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::ExpenseService;
use zfaktury_domain::Expense;

use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;
use crate::util::format::{format_amount, format_date};

/// Expense detail view displaying all expense data.
pub struct ExpenseDetailView {
    service: Arc<ExpenseService>,
    expense_id: i64,
    loading: bool,
    error: Option<String>,
    expense: Option<Expense>,
}

impl ExpenseDetailView {
    pub fn new(service: Arc<ExpenseService>, expense_id: i64, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            service,
            expense_id,
            loading: true,
            error: None,
            expense: None,
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        let id = self.expense_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.get_by_id(id) })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(expense) => this.expense = Some(expense),
                    Err(e) => this.error = Some(format!("Chyba pri nacitani nakladu: {e}")),
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
            .items_center()
            .gap_4()
            .child(
                div()
                    .w_40()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child(label.to_string()),
            )
            .child(
                div()
                    .flex_1()
                    .text_sm()
                    .text_color(if value.is_empty() {
                        rgb(ZfColors::TEXT_MUTED)
                    } else {
                        rgb(ZfColors::TEXT_PRIMARY)
                    })
                    .child(if value.is_empty() {
                        "-".to_string()
                    } else {
                        value
                    }),
            )
    }

    fn render_section(&self, title: &str, fields: Vec<(&str, String)>) -> Div {
        let mut section = div()
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
                    .child(title.to_string()),
            );

        for (label, value) in fields {
            section = section.child(self.render_field(label, value));
        }

        section
    }

    fn render_expense_content(&self, exp: &Expense) -> Div {
        let vendor_name = exp
            .vendor
            .as_ref()
            .map(|v| v.name.clone())
            .unwrap_or_else(|| {
                exp.vendor_id
                    .map(|id| format!("ID {}", id))
                    .unwrap_or_else(|| "-".to_string())
            });

        let mut content = div().flex().flex_col().gap_6();

        // Header
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
                        .child(format!("Naklad {}", exp.expense_number)),
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
                                .child("Upravit"),
                        )
                        .child(
                            div()
                                .px_4()
                                .py_2()
                                .bg(rgb(ZfColors::STATUS_RED_BG))
                                .rounded_md()
                                .text_sm()
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::STATUS_RED))
                                .cursor_pointer()
                                .child("Smazat"),
                        ),
                ),
        );

        // Basic info
        content = content.child(self.render_section(
            "Zakladni udaje",
            vec![
                ("Cislo", exp.expense_number.clone()),
                ("Popis", exp.description.clone()),
                ("Kategorie", exp.category.clone()),
                ("Dodavatel", vendor_name),
                ("Datum", format_date(exp.issue_date)),
                ("Castka", format_amount(exp.amount)),
                ("Mena", exp.currency_code.clone()),
                ("Zpusob platby", exp.payment_method.clone()),
                (
                    "Danove uznatelny",
                    if exp.is_tax_deductible { "Ano" } else { "Ne" }.to_string(),
                ),
                ("Obchodni podil", format!("{}%", exp.business_percent)),
            ],
        ));

        // VAT section
        content = content.child(self.render_section(
            "DPH",
            vec![
                ("Sazba DPH", format!("{}%", exp.vat_rate_percent)),
                ("DPH", format_amount(exp.vat_amount)),
            ],
        ));

        // Notes
        if !exp.notes.is_empty() {
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
                            .child(exp.notes.clone()),
                    ),
            );
        }

        content
    }
}

impl EventEmitter<NavigateEvent> for ExpenseDetailView {}

impl Render for ExpenseDetailView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("expense-detail-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .overflow_y_scroll();

        if self.loading {
            return outer.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Nacitani nakladu..."),
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

        if let Some(ref expense) = self.expense {
            outer = outer.child(self.render_expense_content(expense));
        }

        outer
    }
}
