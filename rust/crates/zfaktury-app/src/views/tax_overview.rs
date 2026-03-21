use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::{
    HealthInsuranceService, IncomeTaxReturnService, SocialInsuranceService,
};
use zfaktury_domain::{
    FilingStatus, HealthInsuranceOverview, IncomeTaxReturn, SocialInsuranceOverview,
};

use crate::components::button::{ButtonVariant, render_button};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;
use crate::util::format::format_amount;

/// Tax overview view showing income tax, social insurance, health insurance cards.
pub struct TaxOverviewView {
    income_tax_service: Arc<IncomeTaxReturnService>,
    social_service: Arc<SocialInsuranceService>,
    health_service: Arc<HealthInsuranceService>,
    year: i32,
    loading: bool,
    error: Option<String>,
    income_returns: Vec<IncomeTaxReturn>,
    social_overviews: Vec<SocialInsuranceOverview>,
    health_overviews: Vec<HealthInsuranceOverview>,
}

impl TaxOverviewView {
    pub fn new(
        income_tax_service: Arc<IncomeTaxReturnService>,
        social_service: Arc<SocialInsuranceService>,
        health_service: Arc<HealthInsuranceService>,
        cx: &mut Context<Self>,
    ) -> Self {
        let year = chrono::Local::now().date_naive().year();
        let mut view = Self {
            income_tax_service,
            social_service,
            health_service,
            year,
            loading: true,
            error: None,
            income_returns: Vec::new(),
            social_overviews: Vec::new(),
            health_overviews: Vec::new(),
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let it_svc = self.income_tax_service.clone();
        let si_svc = self.social_service.clone();
        let hi_svc = self.health_service.clone();
        let year = self.year;

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    let income = it_svc.list(year)?;
                    let social = si_svc.list(year)?;
                    let health = hi_svc.list(year)?;
                    Ok::<
                        (
                            Vec<IncomeTaxReturn>,
                            Vec<SocialInsuranceOverview>,
                            Vec<HealthInsuranceOverview>,
                        ),
                        zfaktury_domain::DomainError,
                    >((income, social, health))
                })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok((income, social, health)) => {
                        this.income_returns = income;
                        this.social_overviews = social;
                        this.health_overviews = health;
                    }
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani dani: {e}"));
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

    #[allow(clippy::too_many_arguments)]
    fn render_tax_card(
        &self,
        title: &str,
        subtitle: &str,
        status: &str,
        status_color: u32,
        amount_label: Option<(&str, zfaktury_domain::Amount)>,
        create_route: Route,
        detail_route: Option<Route>,
        cx: &mut Context<Self>,
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
            .gap_3()
            .child(
                div()
                    .flex()
                    .items_center()
                    .justify_between()
                    .child(
                        div()
                            .text_sm()
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                            .child(title.to_string()),
                    )
                    .child(
                        div()
                            .px_2()
                            .py(px(2.0))
                            .rounded(px(4.0))
                            .text_xs()
                            .text_color(rgb(status_color))
                            .child(status.to_string()),
                    ),
            )
            .child(
                div()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child(subtitle.to_string()),
            );

        // Show amount if available
        if let Some((label, amount)) = amount_label {
            card = card.child(
                div()
                    .flex()
                    .justify_between()
                    .text_sm()
                    .child(
                        div()
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .child(label.to_string()),
                    )
                    .child(
                        div()
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                            .child(format_amount(amount)),
                    ),
            );
        }

        // Button: navigate to detail if exists, or create new
        if let Some(route) = detail_route {
            card = card.child(render_button(
                ElementId::Name(format!("btn-detail-{}", title).into()),
                "Zobrazit detail",
                ButtonVariant::Secondary,
                false,
                false,
                cx.listener(move |_this, _event: &ClickEvent, _window, cx| {
                    cx.emit(NavigateEvent(route.clone()));
                }),
            ));
        } else {
            card = card.child(render_button(
                ElementId::Name(format!("btn-create-{}", title).into()),
                "Vytvorit",
                ButtonVariant::Primary,
                false,
                false,
                cx.listener(move |_this, _event: &ClickEvent, _window, cx| {
                    cx.emit(NavigateEvent(create_route.clone()));
                }),
            ));
        }

        card
    }
}

use chrono::Datelike;

impl EventEmitter<NavigateEvent> for TaxOverviewView {}

impl Render for TaxOverviewView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("tax-overview-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll();

        // Header with year selector
        outer = outer.child(
            div()
                .flex()
                .items_center()
                .gap_3()
                .child(
                    div()
                        .text_xl()
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child("Dan z prijmu"),
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
            return outer.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Nacitani..."),
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

        // Income tax card
        let income_status;
        let income_status_color;
        let income_amount;
        let income_detail;
        if let Some(itr) = self.income_returns.first() {
            income_status = itr.status.to_string();
            income_status_color = match itr.status {
                FilingStatus::Draft => ZfColors::STATUS_GRAY,
                FilingStatus::Ready => ZfColors::STATUS_YELLOW,
                FilingStatus::Filed => ZfColors::STATUS_GREEN,
            };
            income_amount = Some(("Doplatek/Preplatek", itr.tax_due));
            income_detail = Some(Route::TaxIncomeDetail(itr.id));
        } else {
            income_status = "Nevytvoreno".to_string();
            income_status_color = ZfColors::STATUS_GRAY;
            income_amount = None;
            income_detail = None;
        }

        // Social insurance card
        let social_status;
        let social_status_color;
        let social_amount;
        let social_detail;
        if let Some(sio) = self.social_overviews.first() {
            social_status = sio.status.to_string();
            social_status_color = match sio.status {
                FilingStatus::Draft => ZfColors::STATUS_GRAY,
                FilingStatus::Ready => ZfColors::STATUS_YELLOW,
                FilingStatus::Filed => ZfColors::STATUS_GREEN,
            };
            social_amount = Some(("Doplatek/Preplatek", sio.difference));
            social_detail = Some(Route::TaxSocialDetail(sio.id));
        } else {
            social_status = "Nevytvoreno".to_string();
            social_status_color = ZfColors::STATUS_GRAY;
            social_amount = None;
            social_detail = None;
        }

        // Health insurance card
        let health_status;
        let health_status_color;
        let health_amount;
        let health_detail;
        if let Some(hi) = self.health_overviews.first() {
            health_status = hi.status.to_string();
            health_status_color = match hi.status {
                FilingStatus::Draft => ZfColors::STATUS_GRAY,
                FilingStatus::Ready => ZfColors::STATUS_YELLOW,
                FilingStatus::Filed => ZfColors::STATUS_GREEN,
            };
            health_amount = Some(("Doplatek/Preplatek", hi.difference));
            health_detail = Some(Route::TaxHealthDetail(hi.id));
        } else {
            health_status = "Nevytvoreno".to_string();
            health_status_color = ZfColors::STATUS_GRAY;
            health_amount = None;
            health_detail = None;
        }

        // Tax cards
        outer = outer.child(
            div()
                .flex()
                .gap_4()
                .child(self.render_tax_card(
                    "Dan z prijmu",
                    "Danove priznani fyzickych osob (DPFO)",
                    &income_status,
                    income_status_color,
                    income_amount,
                    Route::TaxIncomeNew,
                    income_detail,
                    cx,
                ))
                .child(self.render_tax_card(
                    "Socialni pojisteni",
                    "Prehled OSVC pro CSSZ",
                    &social_status,
                    social_status_color,
                    social_amount,
                    Route::TaxSocialNew,
                    social_detail,
                    cx,
                ))
                .child(self.render_tax_card(
                    "Zdravotni pojisteni",
                    "Prehled OSVC pro ZP",
                    &health_status,
                    health_status_color,
                    health_amount,
                    Route::TaxHealthNew,
                    health_detail,
                    cx,
                )),
        );

        // Info note
        outer = outer.child(
            div()
                .p_4()
                .bg(rgb(ZfColors::SURFACE))
                .rounded_md()
                .border_1()
                .border_color(rgb(ZfColors::BORDER))
                .text_sm()
                .text_color(rgb(ZfColors::TEXT_MUTED))
                .child("Vytvorte danove priznani pro automaticky vypocet dane z prijmu, socialniho a zdravotniho pojisteni."),
        );

        outer
    }
}
