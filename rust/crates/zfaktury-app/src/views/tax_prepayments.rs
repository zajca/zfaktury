use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::TaxYearSettingsService;
use zfaktury_domain::TaxPrepayment;

use crate::components::button::{ButtonVariant, render_button};
use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;
use crate::util::format::format_amount;

/// Tax prepayments view showing monthly prepayment schedule.
pub struct TaxPrepaymentsView {
    service: Arc<TaxYearSettingsService>,
    year: i32,
    loading: bool,
    error: Option<String>,
    prepayments: Vec<TaxPrepayment>,
}

impl TaxPrepaymentsView {
    pub fn new(service: Arc<TaxYearSettingsService>, cx: &mut Context<Self>) -> Self {
        let year = chrono::Local::now().date_naive().year();
        let mut view = Self {
            service,
            year,
            loading: true,
            error: None,
            prepayments: Vec::new(),
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
                .spawn(async move { service.get_prepayments(year) })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(prepayments) => this.prepayments = prepayments,
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani zaloh: {e}"));
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
}

use chrono::Datelike;

impl EventEmitter<NavigateEvent> for TaxPrepaymentsView {}

impl Render for TaxPrepaymentsView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let months = [
            "Leden", "Unor", "Brezen", "Duben", "Kveten", "Cerven", "Cervenec", "Srpen", "Zari",
            "Rijen", "Listopad", "Prosinec",
        ];

        let mut content = div()
            .id("tax-prepayments-scroll")
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
                        .child("Zalohy"),
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
                .child(div().w(px(120.0)).child("Mesic"))
                .child(div().flex_1().text_right().child("Dan z prijmu"))
                .child(div().flex_1().text_right().child("Socialni pojisteni"))
                .child(div().flex_1().text_right().child("Zdravotni pojisteni"))
                .child(div().flex_1().text_right().child("Celkem")),
        );

        let mut total_tax = zfaktury_domain::Amount::ZERO;
        let mut total_social = zfaktury_domain::Amount::ZERO;
        let mut total_health = zfaktury_domain::Amount::ZERO;
        let mut total_all = zfaktury_domain::Amount::ZERO;

        for (i, month) in months.iter().enumerate() {
            let prepayment = self.prepayments.get(i);
            let tax_amount = prepayment.map_or(zfaktury_domain::Amount::ZERO, |p| p.tax_amount);
            let social_amount =
                prepayment.map_or(zfaktury_domain::Amount::ZERO, |p| p.social_amount);
            let health_amount =
                prepayment.map_or(zfaktury_domain::Amount::ZERO, |p| p.health_amount);
            let row_total = tax_amount + social_amount + health_amount;

            total_tax += tax_amount;
            total_social += social_amount;
            total_health += health_amount;
            total_all += row_total;

            let has_data = tax_amount != zfaktury_domain::Amount::ZERO
                || social_amount != zfaktury_domain::Amount::ZERO
                || health_amount != zfaktury_domain::Amount::ZERO;

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
                            .child(format!("{}. {}", i + 1, month)),
                    )
                    .child(
                        div()
                            .flex_1()
                            .text_right()
                            .text_color(if has_data {
                                rgb(ZfColors::TEXT_PRIMARY)
                            } else {
                                rgb(ZfColors::TEXT_MUTED)
                            })
                            .child(if has_data {
                                format_amount(tax_amount)
                            } else {
                                "-".to_string()
                            }),
                    )
                    .child(
                        div()
                            .flex_1()
                            .text_right()
                            .text_color(if has_data {
                                rgb(ZfColors::TEXT_PRIMARY)
                            } else {
                                rgb(ZfColors::TEXT_MUTED)
                            })
                            .child(if has_data {
                                format_amount(social_amount)
                            } else {
                                "-".to_string()
                            }),
                    )
                    .child(
                        div()
                            .flex_1()
                            .text_right()
                            .text_color(if has_data {
                                rgb(ZfColors::TEXT_PRIMARY)
                            } else {
                                rgb(ZfColors::TEXT_MUTED)
                            })
                            .child(if has_data {
                                format_amount(health_amount)
                            } else {
                                "-".to_string()
                            }),
                    )
                    .child(
                        div()
                            .flex_1()
                            .text_right()
                            .font_weight(FontWeight::MEDIUM)
                            .text_color(if has_data {
                                rgb(ZfColors::TEXT_PRIMARY)
                            } else {
                                rgb(ZfColors::TEXT_MUTED)
                            })
                            .child(if has_data {
                                format_amount(row_total)
                            } else {
                                "-".to_string()
                            }),
                    ),
            );
        }

        let has_totals = total_all != zfaktury_domain::Amount::ZERO;

        // Totals row
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
                        .text_color(if has_totals {
                            rgb(ZfColors::TEXT_PRIMARY)
                        } else {
                            rgb(ZfColors::TEXT_MUTED)
                        })
                        .child(if has_totals {
                            format_amount(total_tax)
                        } else {
                            "-".to_string()
                        }),
                )
                .child(
                    div()
                        .flex_1()
                        .text_right()
                        .text_color(if has_totals {
                            rgb(ZfColors::TEXT_PRIMARY)
                        } else {
                            rgb(ZfColors::TEXT_MUTED)
                        })
                        .child(if has_totals {
                            format_amount(total_social)
                        } else {
                            "-".to_string()
                        }),
                )
                .child(
                    div()
                        .flex_1()
                        .text_right()
                        .text_color(if has_totals {
                            rgb(ZfColors::TEXT_PRIMARY)
                        } else {
                            rgb(ZfColors::TEXT_MUTED)
                        })
                        .child(if has_totals {
                            format_amount(total_health)
                        } else {
                            "-".to_string()
                        }),
                )
                .child(
                    div()
                        .flex_1()
                        .text_right()
                        .text_color(if has_totals {
                            rgb(ZfColors::ACCENT)
                        } else {
                            rgb(ZfColors::TEXT_MUTED)
                        })
                        .child(if has_totals {
                            format_amount(total_all)
                        } else {
                            "-".to_string()
                        }),
                ),
        );

        content.child(table)
    }
}
