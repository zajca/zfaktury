use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::report_svc::{ProfitLossReport, ReportService};
use zfaktury_core::service::{ExpenseService, InvoiceService};
use zfaktury_domain::{ExpenseFilter, InvoiceFilter};

use crate::components::button::{ButtonVariant, render_button};
use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;
use crate::util::format::format_amount;

/// Reports view with tabs for Revenue, Expenses, and Profit/Loss.
pub struct ReportsView {
    service: Arc<ReportService>,
    /// Invoice service for CSV export.
    /// NOTE for lead: wire this from `services.invoices.clone()` in root.rs when creating ReportsView.
    invoice_service: Arc<InvoiceService>,
    /// Expense service for CSV export.
    /// NOTE for lead: wire this from `services.expenses.clone()` in root.rs when creating ReportsView.
    expense_service: Arc<ExpenseService>,
    loading: bool,
    error: Option<String>,
    success: Option<String>,
    year: i32,
    active_tab: ReportTab,
    report: Option<ProfitLossReport>,
    csv_exporting: bool,
}

#[derive(Clone, PartialEq)]
enum ReportTab {
    Revenue,
    Expenses,
    ProfitLoss,
}

impl ReportsView {
    pub fn new(
        service: Arc<ReportService>,
        invoice_service: Arc<InvoiceService>,
        expense_service: Arc<ExpenseService>,
        cx: &mut Context<Self>,
    ) -> Self {
        let year = chrono::Local::now().date_naive().year();
        let mut view = Self {
            service,
            invoice_service,
            expense_service,
            loading: true,
            error: None,
            success: None,
            year,
            active_tab: ReportTab::ProfitLoss,
            report: None,
            csv_exporting: false,
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

    fn render_tab_button(
        &self,
        label: &str,
        tab: ReportTab,
        cx: &mut Context<Self>,
    ) -> Stateful<Div> {
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
        let tab_id = format!(
            "tab-{}",
            match tab {
                ReportTab::Revenue => "revenue",
                ReportTab::Expenses => "expenses",
                ReportTab::ProfitLoss => "profit-loss",
            }
        );

        div()
            .id(ElementId::Name(tab_id.into()))
            .px_4()
            .py_2()
            .bg(rgb(bg))
            .rounded_md()
            .text_sm()
            .font_weight(FontWeight::MEDIUM)
            .text_color(rgb(text_color))
            .cursor_pointer()
            .hover(|s| {
                s.bg(rgb(if is_active {
                    ZfColors::ACCENT_HOVER
                } else {
                    ZfColors::SURFACE_HOVER
                }))
            })
            .on_click(cx.listener(move |this, _ev: &ClickEvent, _w, cx| {
                this.active_tab = tab.clone();
                cx.notify();
            }))
            .child(label.to_string())
    }

    fn render_year_button(&self, year: i32, cx: &mut Context<Self>) -> Stateful<Div> {
        let is_active = self.year == year;
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
            .id(ElementId::Name(format!("year-{year}").into()))
            .px_3()
            .py_1()
            .rounded_md()
            .text_sm()
            .bg(rgb(bg))
            .text_color(rgb(text_color))
            .cursor_pointer()
            .hover(|s| {
                s.bg(rgb(if is_active {
                    ZfColors::ACCENT_HOVER
                } else {
                    ZfColors::SURFACE_HOVER
                }))
            })
            .on_click(cx.listener(move |this, _ev: &ClickEvent, _w, cx| {
                this.year = year;
                this.load_data(cx);
            }))
            .child(year.to_string())
    }

    fn export_invoices_csv(&mut self, cx: &mut Context<Self>) {
        self.csv_exporting = true;
        self.error = None;
        self.success = None;
        cx.notify();

        let invoice_service = self.invoice_service.clone();
        let year = self.year;

        cx.spawn(async move |this, cx| {
            let result: Result<usize, String> = cx
                .background_executor()
                .spawn(async move {
                    // Load all invoices for the selected year using date filter.
                    let date_from = chrono::NaiveDate::from_ymd_opt(year, 1, 1).unwrap_or_default();
                    let date_to = chrono::NaiveDate::from_ymd_opt(year, 12, 31).unwrap_or_default();
                    let filter = InvoiceFilter {
                        date_from: Some(date_from),
                        date_to: Some(date_to),
                        limit: 10_000,
                        offset: 0,
                        ..InvoiceFilter::default()
                    };
                    let (invoices, _count) = invoice_service
                        .list(filter)
                        .map_err(|e| format!("Chyba pri nacitani faktur: {e}"))?;

                    // Generate CSV.
                    let csv_bytes = zfaktury_gen::csv::export_invoices_csv(&invoices)
                        .map_err(|e| format!("Chyba pri generovani CSV: {e}"))?;

                    // Save to temp file.
                    let tmp_path = std::env::temp_dir().join(format!("faktury-{}.csv", year));
                    std::fs::write(&tmp_path, &csv_bytes)
                        .map_err(|e| format!("Chyba pri zapisu CSV: {e}"))?;

                    // Open with system viewer.
                    let _ = std::process::Command::new("xdg-open")
                        .arg(&tmp_path)
                        .spawn();

                    Ok(invoices.len())
                })
                .await;

            this.update(cx, |this, cx| {
                this.csv_exporting = false;
                match result {
                    Ok(count) => {
                        this.success = Some(format!("Export {} faktur do CSV dokoncen", count));
                    }
                    Err(e) => {
                        this.error = Some(e);
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn export_expenses_csv(&mut self, cx: &mut Context<Self>) {
        self.csv_exporting = true;
        self.error = None;
        self.success = None;
        cx.notify();

        let expense_service = self.expense_service.clone();
        let year = self.year;

        cx.spawn(async move |this, cx| {
            let result: Result<usize, String> = cx
                .background_executor()
                .spawn(async move {
                    // Load all expenses for the selected year using date filter.
                    let date_from = chrono::NaiveDate::from_ymd_opt(year, 1, 1).unwrap_or_default();
                    let date_to = chrono::NaiveDate::from_ymd_opt(year, 12, 31).unwrap_or_default();
                    let filter = ExpenseFilter {
                        date_from: Some(date_from),
                        date_to: Some(date_to),
                        limit: 10_000,
                        offset: 0,
                        ..ExpenseFilter::default()
                    };
                    let (expenses, _count) = expense_service
                        .list(filter)
                        .map_err(|e| format!("Chyba pri nacitani nakladu: {e}"))?;

                    // Generate CSV.
                    let csv_bytes = zfaktury_gen::csv::export_expenses_csv(&expenses)
                        .map_err(|e| format!("Chyba pri generovani CSV: {e}"))?;

                    // Save to temp file.
                    let tmp_path = std::env::temp_dir().join(format!("naklady-{}.csv", year));
                    std::fs::write(&tmp_path, &csv_bytes)
                        .map_err(|e| format!("Chyba pri zapisu CSV: {e}"))?;

                    // Open with system viewer.
                    let _ = std::process::Command::new("xdg-open")
                        .arg(&tmp_path)
                        .spawn();

                    Ok(expenses.len())
                })
                .await;

            this.update(cx, |this, cx| {
                this.csv_exporting = false;
                match result {
                    Ok(count) => {
                        this.success = Some(format!("Export {} nakladu do CSV dokoncen", count));
                    }
                    Err(e) => {
                        this.error = Some(e);
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
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
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("reports-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll();

        // Header with year selector
        let current_year = chrono::Local::now().date_naive().year();
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
                        .child("Prehled"),
                )
                .child(
                    div()
                        .flex()
                        .items_center()
                        .gap_2()
                        .child(self.render_year_button(current_year - 1, cx))
                        .child(self.render_year_button(current_year, cx))
                        .child(div().w(px(1.0)).h_6().bg(rgb(ZfColors::BORDER)).mx_1())
                        .child(render_button(
                            "btn-export-invoices-csv",
                            "Export faktur (CSV)",
                            ButtonVariant::Secondary,
                            self.csv_exporting,
                            self.csv_exporting,
                            cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                this.export_invoices_csv(cx);
                            }),
                        ))
                        .child(render_button(
                            "btn-export-expenses-csv",
                            "Export nakladu (CSV)",
                            ButtonVariant::Secondary,
                            self.csv_exporting,
                            self.csv_exporting,
                            cx.listener(|this, _event: &ClickEvent, _window, cx| {
                                this.export_expenses_csv(cx);
                            }),
                        )),
                ),
        );

        // Tabs
        content = content.child(
            div()
                .flex()
                .gap_2()
                .child(self.render_tab_button("Prijmy", ReportTab::Revenue, cx))
                .child(self.render_tab_button("Naklady", ReportTab::Expenses, cx))
                .child(self.render_tab_button("Zisk a ztrata", ReportTab::ProfitLoss, cx)),
        );

        // Success message
        if let Some(ref success) = self.success {
            content = content.child(
                div()
                    .px_4()
                    .py_3()
                    .bg(rgb(ZfColors::STATUS_GREEN_BG))
                    .rounded_md()
                    .text_sm()
                    .text_color(rgb(ZfColors::STATUS_GREEN))
                    .child(success.clone()),
            );
        }

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
