use std::sync::Arc;

use gpui::*;
use zfaktury_core::repository::types::{RecentExpense, RecentInvoice};
use zfaktury_core::service::DashboardService;
use zfaktury_domain::Amount;

use crate::components::status_badge::render_status_badge;
use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;
use crate::util::format::{format_amount, format_date};

/// Dashboard view showing summary cards and recent activity.
pub struct DashboardView {
    service: Arc<DashboardService>,
    loading: bool,
    error: Option<String>,

    // Stats
    revenue: Amount,
    expenses_amount: Amount,
    unpaid_count: i64,
    unpaid_total: Amount,
    overdue_count: i64,
    overdue_total: Amount,

    // Recent items
    recent_invoices: Vec<RecentInvoice>,
    recent_expenses: Vec<RecentExpense>,
}

impl DashboardView {
    pub fn new(service: Arc<DashboardService>, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            service,
            loading: true,
            error: None,
            revenue: Amount::ZERO,
            expenses_amount: Amount::ZERO,
            unpaid_count: 0,
            unpaid_total: Amount::ZERO,
            overdue_count: 0,
            overdue_total: Amount::ZERO,
            recent_invoices: Vec::new(),
            recent_expenses: Vec::new(),
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.get_dashboard() })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(data) => {
                        this.revenue = data.revenue_current_month;
                        this.expenses_amount = data.expenses_current_month;
                        this.unpaid_count = data.unpaid_count;
                        this.unpaid_total = data.unpaid_total;
                        this.overdue_count = data.overdue_count;
                        this.overdue_total = data.overdue_total;
                        this.recent_invoices = data.recent_invoices;
                        this.recent_expenses = data.recent_expenses;
                    }
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn render_stat_card(
        &self,
        title: &str,
        value: &str,
        subtitle: Option<String>,
        color: u32,
    ) -> Div {
        let mut card = div()
            .flex_1()
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
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child(title.to_string()),
            )
            .child(
                div()
                    .text_xl()
                    .font_weight(FontWeight::BOLD)
                    .text_color(rgb(color))
                    .child(value.to_string()),
            );

        if let Some(sub) = subtitle {
            card = card.child(
                div()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child(sub),
            );
        }

        card
    }

    fn render_invoice_table(&self) -> Div {
        let mut table = div()
            .flex()
            .flex_col()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .overflow_hidden();

        // Header
        table = table.child(
            div()
                .flex()
                .items_center()
                .justify_between()
                .px_4()
                .py_3()
                .border_b_1()
                .border_color(rgb(ZfColors::BORDER))
                .child(
                    div()
                        .text_sm()
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child("Posledni faktury"),
                ),
        );

        if self.recent_invoices.is_empty() {
            table = table.child(
                div()
                    .px_4()
                    .py_6()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Zadne faktury"),
            );
        } else {
            // Column headers
            table = table.child(
                div()
                    .flex()
                    .px_4()
                    .py_2()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child(div().w_24().child("Cislo"))
                    .child(div().flex_1().child("Zakaznik"))
                    .child(div().w_24().child("Datum"))
                    .child(div().w(px(112.0)).text_right().child("Castka"))
                    .child(div().w_24().text_right().child("Stav")),
            );

            for inv in &self.recent_invoices {
                table = table.child(
                    div()
                        .flex()
                        .items_center()
                        .px_4()
                        .py_2()
                        .text_sm()
                        .border_t_1()
                        .border_color(rgb(ZfColors::BORDER_SUBTLE))
                        .hover(|s| s.bg(rgb(ZfColors::SURFACE_HOVER)))
                        .child(
                            div()
                                .w_24()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(inv.invoice_number.clone()),
                        )
                        .child(
                            div()
                                .flex_1()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(inv.customer_name.clone()),
                        )
                        .child(
                            div()
                                .w_24()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format_date(inv.issue_date)),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(inv.total_amount)),
                        )
                        .child(
                            div()
                                .w_24()
                                .flex()
                                .justify_end()
                                .child(render_status_badge(&inv.status)),
                        ),
                );
            }
        }

        table
    }

    fn render_expense_table(&self) -> Div {
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
                .flex()
                .items_center()
                .justify_between()
                .px_4()
                .py_3()
                .border_b_1()
                .border_color(rgb(ZfColors::BORDER))
                .child(
                    div()
                        .text_sm()
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child("Posledni naklady"),
                ),
        );

        if self.recent_expenses.is_empty() {
            table = table.child(
                div()
                    .px_4()
                    .py_6()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Zadne naklady"),
            );
        } else {
            table = table.child(
                div()
                    .flex()
                    .px_4()
                    .py_2()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child(div().flex_1().child("Popis"))
                    .child(div().w(px(112.0)).child("Kategorie"))
                    .child(div().w_24().child("Datum"))
                    .child(div().w(px(112.0)).text_right().child("Castka")),
            );

            for exp in &self.recent_expenses {
                table = table.child(
                    div()
                        .flex()
                        .items_center()
                        .px_4()
                        .py_2()
                        .text_sm()
                        .border_t_1()
                        .border_color(rgb(ZfColors::BORDER_SUBTLE))
                        .hover(|s| s.bg(rgb(ZfColors::SURFACE_HOVER)))
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
                                .w_24()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format_date(exp.issue_date)),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(exp.amount)),
                        ),
                );
            }
        }

        table
    }
}

impl EventEmitter<NavigateEvent> for DashboardView {}

impl Render for DashboardView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("dashboard-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll();

        // Title
        content = content.child(
            div()
                .text_xl()
                .font_weight(FontWeight::SEMIBOLD)
                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                .child("Dashboard"),
        );

        if self.loading {
            content = content.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Nacitani..."),
            );
            return content;
        }

        if let Some(ref error) = self.error {
            content = content.child(
                div()
                    .px_4()
                    .py_3()
                    .bg(rgb(ZfColors::STATUS_RED_BG))
                    .rounded_md()
                    .text_sm()
                    .text_color(rgb(ZfColors::STATUS_RED))
                    .child(error.clone()),
            );
            return content;
        }

        // Stats cards row
        content = content.child(
            div()
                .flex()
                .gap_4()
                .child(self.render_stat_card(
                    "Prijmy tento mesic",
                    &format_amount(self.revenue),
                    None,
                    ZfColors::STATUS_GREEN,
                ))
                .child(self.render_stat_card(
                    "Naklady tento mesic",
                    &format_amount(self.expenses_amount),
                    None,
                    ZfColors::STATUS_RED,
                ))
                .child(self.render_stat_card(
                    "Neuhrazene faktury",
                    &format_amount(self.unpaid_total),
                    Some(format!("{} faktur", self.unpaid_count)),
                    ZfColors::STATUS_YELLOW,
                ))
                .child(self.render_stat_card(
                    "Po splatnosti",
                    &format_amount(self.overdue_total),
                    Some(format!("{} faktur", self.overdue_count)),
                    ZfColors::STATUS_RED,
                )),
        );

        // Recent tables
        content = content
            .child(self.render_invoice_table())
            .child(self.render_expense_table());

        content
    }
}
