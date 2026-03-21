use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::InvestmentIncomeService;
use zfaktury_domain::{CapitalIncomeEntry, InvestmentYearSummary, SecurityTransaction};

use crate::components::button::{ButtonVariant, render_button};
use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;
use crate::util::format::{format_amount, format_date};

/// Tax investments view showing investment income summary.
pub struct TaxInvestmentsView {
    service: Arc<InvestmentIncomeService>,
    year: i32,
    loading: bool,
    error: Option<String>,
    summary: Option<InvestmentYearSummary>,
    capital_entries: Vec<CapitalIncomeEntry>,
    security_transactions: Vec<SecurityTransaction>,
}

impl TaxInvestmentsView {
    pub fn new(service: Arc<InvestmentIncomeService>, cx: &mut Context<Self>) -> Self {
        let year = chrono::Local::now().date_naive().year();
        let mut view = Self {
            service,
            year,
            loading: true,
            error: None,
            summary: None,
            capital_entries: Vec::new(),
            security_transactions: Vec::new(),
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
                .spawn(async move {
                    let summary = service.get_year_summary(year)?;
                    let capital = service.list_capital_entries(year)?;
                    let securities = service.list_security_transactions(year)?;
                    Ok::<
                        (
                            InvestmentYearSummary,
                            Vec<CapitalIncomeEntry>,
                            Vec<SecurityTransaction>,
                        ),
                        zfaktury_domain::DomainError,
                    >((summary, capital, securities))
                })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok((summary, capital, securities)) => {
                        this.summary = Some(summary);
                        this.capital_entries = capital;
                        this.security_transactions = securities;
                    }
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani investic: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn change_year(&mut self, delta: i32, cx: &mut Context<Self>) {
        self.year += delta;
        self.loading = true;
        self.error = None;
        cx.notify();
        self.load_data(cx);
    }

    fn render_summary_field(&self, label: &str, value: zfaktury_domain::Amount) -> Div {
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
                    .child(format_amount(value)),
            )
    }

    fn render_summary_field_bold(&self, label: &str, value: zfaktury_domain::Amount) -> Div {
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
                    .font_weight(FontWeight::BOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child(format_amount(value)),
            )
    }
}

use chrono::Datelike;

impl EventEmitter<NavigateEvent> for TaxInvestmentsView {}

impl Render for TaxInvestmentsView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("tax-investments-scroll")
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
                .gap_3()
                .child(
                    div()
                        .text_xl()
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child("Investice"),
                )
                .child(render_button(
                    "btn-year-prev",
                    "<",
                    ButtonVariant::Secondary,
                    false,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.change_year(-1, cx);
                    }),
                ))
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
                )
                .child(render_button(
                    "btn-year-next",
                    ">",
                    ButtonVariant::Secondary,
                    false,
                    false,
                    cx.listener(|this, _event: &ClickEvent, _window, cx| {
                        this.change_year(1, cx);
                    }),
                )),
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

        // Summary card
        if let Some(ref summary) = self.summary {
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
                            .child("Souhrn investicnich prijmu"),
                    )
                    .child(
                        div()
                            .flex()
                            .gap_8()
                            .child(self.render_summary_field(
                                "Kapitalove prijmy (brutto)",
                                summary.capital_income_gross,
                            ))
                            .child(
                                self.render_summary_field(
                                    "Srazena dan",
                                    summary.capital_income_tax,
                                ),
                            )
                            .child(self.render_summary_field_bold(
                                "Kapitalovy prijem (netto)",
                                summary.capital_income_net,
                            )),
                    )
                    .child(div().h(px(1.0)).bg(rgb(ZfColors::BORDER)))
                    .child(
                        div()
                            .flex()
                            .gap_8()
                            .child(self.render_summary_field(
                                "Ostatni prijmy (brutto)",
                                summary.other_income_gross,
                            ))
                            .child(
                                self.render_summary_field("Naklady", summary.other_income_expenses),
                            )
                            .child(
                                self.render_summary_field(
                                    "Osvobozeno",
                                    summary.other_income_exempt,
                                ),
                            )
                            .child(self.render_summary_field_bold(
                                "Zaklad dane (p.10)",
                                summary.other_income_net,
                            )),
                    ),
            );
        }

        // Capital income entries table
        {
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
                    .child("Kapitalove prijmy (paragraf 8)"),
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
                    .child(div().flex_1().child("Popis"))
                    .child(div().w_20().child("Kategorie"))
                    .child(div().w_24().child("Datum"))
                    .child(div().w(px(112.0)).text_right().child("Brutto"))
                    .child(div().w(px(112.0)).text_right().child("Dan"))
                    .child(div().w(px(112.0)).text_right().child("Netto")),
            );

            if self.capital_entries.is_empty() {
                table = table.child(
                    div()
                        .px_4()
                        .py_4()
                        .text_sm()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child("Zadne kapitalove prijmy pro tento rok."),
                );
            } else {
                for entry in &self.capital_entries {
                    let cat_label = match entry.category {
                        zfaktury_domain::CapitalCategory::DividendCZ => "Dividenda CZ",
                        zfaktury_domain::CapitalCategory::DividendForeign => "Dividenda zahr.",
                        zfaktury_domain::CapitalCategory::Interest => "Urok",
                        zfaktury_domain::CapitalCategory::Coupon => "Kupon",
                        zfaktury_domain::CapitalCategory::FundDistribution => "Fond",
                        zfaktury_domain::CapitalCategory::Other => "Ostatni",
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
                                    .flex_1()
                                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                    .child(entry.description.clone()),
                            )
                            .child(
                                div()
                                    .w_20()
                                    .text_color(rgb(ZfColors::TEXT_MUTED))
                                    .child(cat_label),
                            )
                            .child(
                                div()
                                    .w_24()
                                    .text_color(rgb(ZfColors::TEXT_MUTED))
                                    .child(format_date(entry.income_date)),
                            )
                            .child(
                                div()
                                    .w(px(112.0))
                                    .text_right()
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child(format_amount(entry.gross_amount)),
                            )
                            .child(
                                div()
                                    .w(px(112.0))
                                    .text_right()
                                    .text_color(rgb(ZfColors::TEXT_MUTED))
                                    .child(format_amount(
                                        entry.withheld_tax_cz + entry.withheld_tax_foreign,
                                    )),
                            )
                            .child(
                                div()
                                    .w(px(112.0))
                                    .text_right()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                    .child(format_amount(entry.net_amount)),
                            ),
                    );
                }
            }

            content = content.child(table);
        }

        // Security transactions table
        {
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
                    .child("Transakce s cennymi papiry (paragraf 10)"),
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
                    .child(div().flex_1().child("Nazev"))
                    .child(div().w_20().child("Typ"))
                    .child(div().w_20().child("Operace"))
                    .child(div().w_24().child("Datum"))
                    .child(div().w(px(112.0)).text_right().child("Castka"))
                    .child(div().w(px(112.0)).text_right().child("Zisk/Ztrata")),
            );

            if self.security_transactions.is_empty() {
                table = table.child(
                    div()
                        .px_4()
                        .py_4()
                        .text_sm()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child("Zadne transakce pro tento rok."),
                );
            } else {
                for tx in &self.security_transactions {
                    let asset_label = match tx.asset_type {
                        zfaktury_domain::AssetType::Stock => "Akcie",
                        zfaktury_domain::AssetType::ETF => "ETF",
                        zfaktury_domain::AssetType::Bond => "Dluhopis",
                        zfaktury_domain::AssetType::Fund => "Fond",
                        zfaktury_domain::AssetType::Crypto => "Krypto",
                        zfaktury_domain::AssetType::Other => "Ostatni",
                    };

                    let tx_type_label = match tx.transaction_type {
                        zfaktury_domain::TransactionType::Buy => "Nakup",
                        zfaktury_domain::TransactionType::Sell => "Prodej",
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
                                    .flex_1()
                                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                    .child(tx.asset_name.clone()),
                            )
                            .child(
                                div()
                                    .w_20()
                                    .text_color(rgb(ZfColors::TEXT_MUTED))
                                    .child(asset_label),
                            )
                            .child(
                                div()
                                    .w_20()
                                    .text_color(rgb(ZfColors::TEXT_MUTED))
                                    .child(tx_type_label),
                            )
                            .child(
                                div()
                                    .w_24()
                                    .text_color(rgb(ZfColors::TEXT_MUTED))
                                    .child(format_date(tx.transaction_date)),
                            )
                            .child(
                                div()
                                    .w(px(112.0))
                                    .text_right()
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child(format_amount(tx.total_amount)),
                            )
                            .child(
                                div()
                                    .w(px(112.0))
                                    .text_right()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                    .child(format_amount(tx.computed_gain)),
                            ),
                    );
                }
            }

            content = content.child(table);
        }

        content
    }
}
