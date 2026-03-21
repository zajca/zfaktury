use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::report_svc::{ProfitLossReport, ReportService};

use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;
use crate::util::format::format_amount;

/// Reports view with tabs for Revenue, Expenses, and Profit/Loss.
pub struct ReportsView {
    service: Arc<ReportService>,
    loading: bool,
    error: Option<String>,
    year: i32,
    active_tab: ReportTab,
    report: Option<ProfitLossReport>,
}

#[derive(Clone, PartialEq)]
enum ReportTab {
    Revenue,
    Expenses,
    ProfitLoss,
}

impl ReportsView {
    pub fn new(service: Arc<ReportService>, cx: &mut Context<Self>) -> Self {
        let year = chrono::Local::now().date_naive().year();
        let mut view = Self {
            service,
            loading: true,
            error: None,
            year,
            active_tab: ReportTab::ProfitLoss,
            report: None,
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
                .spawn(async move { service.profit_loss(year) })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(report) => this.report = Some(report),
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani reportu: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn render_tab_button(&self, label: &str, tab: ReportTab) -> Div {
        let is_active = self.active_tab == tab;
        let bg = if is_active {
            ZfColors::ACCENT
        } else {
            ZfColors::SURFACE
        };
        let text_color = if is_active {
            0xffffff
        } else {
            ZfColors::TEXT_SECONDARY
        };

        div()
            .px_4()
            .py_2()
            .bg(rgb(bg))
            .rounded_md()
            .text_sm()
            .font_weight(FontWeight::MEDIUM)
            .text_color(rgb(text_color))
            .cursor_pointer()
            .child(label.to_string())
    }

    fn month_name(month: i32) -> &'static str {
        match month {
            1 => "Leden",
            2 => "Unor",
            3 => "Brezen",
            4 => "Duben",
            5 => "Kveten",
            6 => "Cerven",
            7 => "Cervenec",
            8 => "Srpen",
            9 => "Zari",
            10 => "Rijen",
            11 => "Listopad",
            12 => "Prosinec",
            _ => "-",
        }
    }
}

use chrono::Datelike;

impl EventEmitter<NavigateEvent> for ReportsView {}

impl Render for ReportsView {
    fn render(&mut self, _window: &mut Window, _cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("reports-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
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
                        .child("Prehled"),
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

        // Tabs
        content = content.child(
            div()
                .flex()
                .gap_2()
                .child(self.render_tab_button("Prijmy", ReportTab::Revenue))
                .child(self.render_tab_button("Naklady", ReportTab::Expenses))
                .child(self.render_tab_button("Zisk a ztrata", ReportTab::ProfitLoss)),
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

        // Data table
        if let Some(ref report) = self.report {
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
                    .child(div().flex_1().text_right().child("Prijmy"))
                    .child(div().flex_1().text_right().child("Naklady"))
                    .child(div().flex_1().text_right().child("Zisk/Ztrata")),
            );

            let mut total_rev = zfaktury_domain::Amount::ZERO;
            let mut total_exp = zfaktury_domain::Amount::ZERO;

            for month in 1..=12 {
                let rev = report
                    .monthly_revenue
                    .iter()
                    .find(|m| m.month == month)
                    .map(|m| m.amount)
                    .unwrap_or(zfaktury_domain::Amount::ZERO);
                let exp = report
                    .monthly_expenses
                    .iter()
                    .find(|m| m.month == month)
                    .map(|m| m.amount)
                    .unwrap_or(zfaktury_domain::Amount::ZERO);
                let profit = rev - exp;

                total_rev += rev;
                total_exp += exp;

                let profit_color = if profit.halere() >= 0 {
                    ZfColors::STATUS_GREEN
                } else {
                    ZfColors::STATUS_RED
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
                        .child(
                            div()
                                .w(px(120.0))
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(Self::month_name(month)),
                        )
                        .child(
                            div()
                                .flex_1()
                                .text_right()
                                .text_color(rgb(ZfColors::STATUS_GREEN))
                                .child(format_amount(rev)),
                        )
                        .child(
                            div()
                                .flex_1()
                                .text_right()
                                .text_color(rgb(ZfColors::STATUS_RED))
                                .child(format_amount(exp)),
                        )
                        .child(
                            div()
                                .flex_1()
                                .text_right()
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(profit_color))
                                .child(format_amount(profit)),
                        ),
                );
            }

            // Totals row
            let total_profit = total_rev - total_exp;
            let total_profit_color = if total_profit.halere() >= 0 {
                ZfColors::STATUS_GREEN
            } else {
                ZfColors::STATUS_RED
            };

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
                            .text_color(rgb(ZfColors::STATUS_GREEN))
                            .child(format_amount(total_rev)),
                    )
                    .child(
                        div()
                            .flex_1()
                            .text_right()
                            .text_color(rgb(ZfColors::STATUS_RED))
                            .child(format_amount(total_exp)),
                    )
                    .child(
                        div()
                            .flex_1()
                            .text_right()
                            .text_color(rgb(total_profit_color))
                            .child(format_amount(total_profit)),
                    ),
            );

            content = content.child(table);
        }

        content
    }
}
