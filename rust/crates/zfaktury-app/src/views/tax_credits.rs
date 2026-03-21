use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::TaxCreditsService;
use zfaktury_domain::{TaxChildCredit, TaxDeduction, TaxPersonalCredits, TaxSpouseCredit};

use crate::components::button::{ButtonVariant, render_button};
use crate::navigation::NavigateEvent;
use crate::theme::ZfColors;
use crate::util::format::format_amount;

/// Tax credits and deductions view.
pub struct TaxCreditsView {
    service: Arc<TaxCreditsService>,
    year: i32,
    loading: bool,
    error: Option<String>,
    spouse_credit: Option<TaxSpouseCredit>,
    children: Vec<TaxChildCredit>,
    personal: Option<TaxPersonalCredits>,
    deductions: Vec<TaxDeduction>,
}

impl TaxCreditsView {
    pub fn new(service: Arc<TaxCreditsService>, cx: &mut Context<Self>) -> Self {
        let year = chrono::Local::now().date_naive().year();
        let mut view = Self {
            service,
            year,
            loading: true,
            error: None,
            spouse_credit: None,
            children: Vec::new(),
            personal: None,
            deductions: Vec::new(),
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
                    let spouse = service.get_spouse(year).ok();
                    let children = service.list_children(year)?;
                    let personal = service.get_personal(year).ok();
                    let deductions = service.list_deductions(year)?;
                    Ok::<
                        (
                            Option<TaxSpouseCredit>,
                            Vec<TaxChildCredit>,
                            Option<TaxPersonalCredits>,
                            Vec<TaxDeduction>,
                        ),
                        zfaktury_domain::DomainError,
                    >((spouse, children, personal, deductions))
                })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok((spouse, children, personal, deductions)) => {
                        this.spouse_credit = spouse;
                        this.children = children;
                        this.personal = personal;
                        this.deductions = deductions;
                    }
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani slev: {e}"));
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

    fn render_spouse_card(&self) -> Div {
        let mut card = div()
            .p_4()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .flex()
            .flex_col()
            .gap_2()
            .child(
                div()
                    .text_sm()
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child("Sleva na manzelku/manzela"),
            )
            .child(
                div()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Prijem manzelky/manzela do 68 000 Kc. ZTP/P = dvojnasobek."),
            );

        if let Some(ref sc) = self.spouse_credit {
            card = card
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child("Jmeno"),
                        )
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(sc.spouse_name.clone()),
                        ),
                )
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child("Prijem"),
                        )
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(sc.spouse_income)),
                        ),
                )
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child("ZTP/P"),
                        )
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(if sc.spouse_ztp { "Ano" } else { "Ne" }),
                        ),
                )
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child("Mesicu"),
                        )
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(sc.months_claimed.to_string()),
                        ),
                )
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child("Sleva"),
                        )
                        .child(
                            div()
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(rgb(ZfColors::ACCENT))
                                .child(format_amount(sc.credit_amount)),
                        ),
                );
        } else {
            card = card.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Zadna sleva na manzela/ku pro tento rok."),
            );
        }

        card
    }

    fn render_children_card(&self) -> Div {
        let mut card = div()
            .p_4()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .flex()
            .flex_col()
            .gap_2()
            .child(
                div()
                    .text_sm()
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child("Deti"),
            )
            .child(
                div()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Danove zvyhodneni na vyzivanou deti. 1./2./3.+ dite."),
            );

        if self.children.is_empty() {
            card = card.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Zadne deti pro tento rok."),
            );
        } else {
            for child in &self.children {
                card = card.child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .py_1()
                        .border_t_1()
                        .border_color(rgb(ZfColors::BORDER_SUBTLE))
                        .child(div().text_color(rgb(ZfColors::TEXT_PRIMARY)).child(format!(
                            "{}. dite: {} ({}m{})",
                            child.child_order,
                            child.child_name,
                            child.months_claimed,
                            if child.ztp { " ZTP/P" } else { "" }
                        )))
                        .child(
                            div()
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::ACCENT))
                                .child(format_amount(child.credit_amount)),
                        ),
                );
            }
        }

        card
    }

    fn render_personal_card(&self) -> Div {
        let mut card = div()
            .p_4()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .flex()
            .flex_col()
            .gap_2()
            .child(
                div()
                    .text_sm()
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child("Osobni slevy"),
            )
            .child(
                div()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Sleva na studenta, invalidni duchod."),
            );

        if let Some(ref pc) = self.personal {
            if pc.is_student {
                card = card.child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(format!("Student ({}m)", pc.student_months)),
                        )
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(pc.credit_student)),
                        ),
                );
            }
            if pc.disability_level > 0 {
                let level_text = match pc.disability_level {
                    1 => "I./II. stupen",
                    2 => "III. stupen",
                    3 => "ZTP/P",
                    _ => "Nezname",
                };
                card = card.child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(format!("Invalidita ({})", level_text)),
                        )
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(pc.credit_disability)),
                        ),
                );
            }
            if !pc.is_student && pc.disability_level == 0 {
                card = card.child(
                    div()
                        .text_sm()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child("Zadne osobni slevy."),
                );
            }
        } else {
            card = card.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Zadne osobni slevy pro tento rok."),
            );
        }

        card
    }

    fn render_deductions_card(&self) -> Div {
        let mut card = div()
            .p_4()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(ZfColors::BORDER))
            .flex()
            .flex_col()
            .gap_2()
            .child(
                div()
                    .text_sm()
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child("Odpocty"),
            )
            .child(
                div()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Hypoteka, zivotni pojisteni, penzijni pripojisteni, dary, odbory."),
            );

        if self.deductions.is_empty() {
            card = card.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Zadne odpocty pro tento rok."),
            );
        } else {
            for ded in &self.deductions {
                let category_label = match ded.category {
                    zfaktury_domain::DeductionCategory::Mortgage => "Hypoteka",
                    zfaktury_domain::DeductionCategory::LifeInsurance => "Zivotni pojisteni",
                    zfaktury_domain::DeductionCategory::Pension => "Penzijni pripojisteni",
                    zfaktury_domain::DeductionCategory::Donation => "Dar",
                    zfaktury_domain::DeductionCategory::UnionDues => "Odbory",
                };
                card = card.child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .py_1()
                        .border_t_1()
                        .border_color(rgb(ZfColors::BORDER_SUBTLE))
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format!("{}: {}", category_label, ded.description)),
                        )
                        .child(
                            div()
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(format_amount(ded.allowed_amount)),
                        ),
                );
            }
        }

        card
    }
}

use chrono::Datelike;

impl EventEmitter<NavigateEvent> for TaxCreditsView {}

impl Render for TaxCreditsView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("tax-credits-scroll")
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
                        .child("Slevy a odpocty"),
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

        // Credit cards in a 2-column grid
        outer = outer.child(
            div()
                .flex()
                .gap_4()
                .child(div().flex_1().child(self.render_spouse_card()))
                .child(div().flex_1().child(self.render_children_card())),
        );

        outer = outer.child(
            div()
                .flex()
                .gap_4()
                .child(div().flex_1().child(self.render_personal_card()))
                .child(div().flex_1().child(self.render_deductions_card())),
        );

        outer
    }
}
