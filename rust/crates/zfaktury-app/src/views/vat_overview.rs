use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::{VATControlStatementService, VATReturnService, VIESSummaryService};
use zfaktury_domain::{FilingStatus, VATControlStatement, VATReturn, VIESSummary};

use crate::components::button::{ButtonVariant, render_button};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;
use crate::util::format::format_amount;

/// VAT overview view with year selector and returns list.
pub struct VatOverviewView {
    vat_service: Arc<VATReturnService>,
    control_service: Arc<VATControlStatementService>,
    vies_service: Arc<VIESSummaryService>,
    loading: bool,
    error: Option<String>,
    year: i32,
    returns: Vec<VATReturn>,
    control_statements: Vec<VATControlStatement>,
    vies_summaries: Vec<VIESSummary>,
}

impl VatOverviewView {
    pub fn new(
        vat_service: Arc<VATReturnService>,
        control_service: Arc<VATControlStatementService>,
        vies_service: Arc<VIESSummaryService>,
        cx: &mut Context<Self>,
    ) -> Self {
        let year = chrono::Local::now().date_naive().year();
        let mut view = Self {
            vat_service,
            control_service,
            vies_service,
            loading: true,
            error: None,
            year,
            returns: Vec::new(),
            control_statements: Vec::new(),
            vies_summaries: Vec::new(),
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let vat_svc = self.vat_service.clone();
        let ctrl_svc = self.control_service.clone();
        let vies_svc = self.vies_service.clone();
        let year = self.year;

        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    let returns = vat_svc.list(year)?;
                    let controls = ctrl_svc.list(year)?;
                    let vies = vies_svc.list(year)?;
                    Ok::<
                        (Vec<VATReturn>, Vec<VATControlStatement>, Vec<VIESSummary>),
                        zfaktury_domain::DomainError,
                    >((returns, controls, vies))
                })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok((returns, controls, vies)) => {
                        this.returns = returns;
                        this.control_statements = controls;
                        this.vies_summaries = vies;
                    }
                    Err(e) => {
                        this.error = Some(format!("Chyba pri nacitani DPH: {e}"));
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

    fn render_quarter_card(&self, quarter: i32) -> Div {
        let months: Vec<&str> = match quarter {
            1 => vec!["Leden", "Unor", "Brezen"],
            2 => vec!["Duben", "Kveten", "Cerven"],
            3 => vec!["Cervenec", "Srpen", "Zari"],
            _ => vec!["Rijen", "Listopad", "Prosinec"],
        };

        let has_return = self.returns.iter().any(|r| r.period.quarter == quarter);
        let border_color = if has_return {
            ZfColors::STATUS_GREEN
        } else {
            ZfColors::BORDER
        };

        let mut card = div()
            .flex_1()
            .p_4()
            .bg(rgb(ZfColors::SURFACE))
            .rounded_md()
            .border_1()
            .border_color(rgb(border_color))
            .flex()
            .flex_col()
            .gap_2()
            .child(
                div()
                    .text_sm()
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child(format!("Q{}", quarter)),
            );

        for month in months {
            card = card.child(
                div()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child(month.to_string()),
            );
        }

        card
    }

    fn render_status_badge(&self, status: &FilingStatus) -> Div {
        let status_text = status.to_string();
        let status_color = match status {
            FilingStatus::Draft => ZfColors::STATUS_GRAY,
            FilingStatus::Ready => ZfColors::STATUS_YELLOW,
            FilingStatus::Filed => ZfColors::STATUS_GREEN,
        };

        div()
            .px_2()
            .py(px(2.0))
            .rounded(px(4.0))
            .text_xs()
            .text_color(rgb(status_color))
            .child(status_text)
    }
}

use chrono::Datelike;

impl EventEmitter<NavigateEvent> for VatOverviewView {}

impl Render for VatOverviewView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut content = div()
            .id("vat-overview-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .flex()
            .flex_col()
            .gap_6()
            .overflow_y_scroll();

        // Header with year selector and buttons
        content = content.child(
            div()
                .flex()
                .items_center()
                .justify_between()
                .child(
                    div()
                        .flex()
                        .items_center()
                        .gap_3()
                        .child(
                            div()
                                .text_xl()
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child("DPH"),
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
                )
                .child(
                    div()
                        .flex()
                        .gap_2()
                        .child(render_button(
                            "btn-new-vat",
                            "Nove DPH priznani",
                            ButtonVariant::Primary,
                            false,
                            false,
                            cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                                cx.emit(NavigateEvent(Route::VATReturnNew));
                            }),
                        ))
                        .child(render_button(
                            "btn-new-control",
                            "Kontrolni hlaseni",
                            ButtonVariant::Secondary,
                            false,
                            false,
                            cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                                cx.emit(NavigateEvent(Route::VATControlNew));
                            }),
                        ))
                        .child(render_button(
                            "btn-new-vies",
                            "Souhrnne hlaseni",
                            ButtonVariant::Secondary,
                            false,
                            false,
                            cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                                cx.emit(NavigateEvent(Route::VIESNew));
                            }),
                        )),
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

        // Quarter cards
        content = content.child(
            div()
                .flex()
                .gap_4()
                .child(self.render_quarter_card(1))
                .child(self.render_quarter_card(2))
                .child(self.render_quarter_card(3))
                .child(self.render_quarter_card(4)),
        );

        // VAT Returns table
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
                    .child("DPH priznani"),
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
                    .child(div().w(px(80.0)).child("Obdobi"))
                    .child(div().w(px(112.0)).child("Typ"))
                    .child(div().w(px(112.0)).text_right().child("DPH na vystupu"))
                    .child(div().w(px(112.0)).text_right().child("DPH na vstupu"))
                    .child(div().w(px(112.0)).text_right().child("Vysledek"))
                    .child(div().w_20().text_right().child("Stav")),
            );

            if self.returns.is_empty() {
                table = table.child(
                    div()
                        .px_4()
                        .py_8()
                        .text_sm()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child("Zadna DPH priznani pro tento rok."),
                );
            } else {
                for vr in &self.returns {
                    let period_label = if vr.period.month > 0 {
                        format!("{}/{}", vr.period.month, vr.period.year)
                    } else {
                        format!("Q{}/{}", vr.period.quarter, vr.period.year)
                    };

                    let vr_id = vr.id;
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
                            .on_mouse_down(
                                MouseButton::Left,
                                cx.listener(move |_this, _event: &MouseDownEvent, _window, cx| {
                                    cx.emit(NavigateEvent(Route::VATReturnDetail(vr_id)));
                                }),
                            )
                            .child(
                                div()
                                    .w(px(80.0))
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                    .child(period_label),
                            )
                            .child(
                                div()
                                    .w(px(112.0))
                                    .text_color(rgb(ZfColors::TEXT_MUTED))
                                    .child(vr.filing_type.to_string()),
                            )
                            .child(
                                div()
                                    .w(px(112.0))
                                    .text_right()
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child(format_amount(vr.total_output_vat)),
                            )
                            .child(
                                div()
                                    .w(px(112.0))
                                    .text_right()
                                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                    .child(format_amount(vr.total_input_vat)),
                            )
                            .child(
                                div()
                                    .w(px(112.0))
                                    .text_right()
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                    .child(format_amount(vr.net_vat)),
                            )
                            .child(
                                div()
                                    .w_20()
                                    .flex()
                                    .justify_end()
                                    .child(self.render_status_badge(&vr.status)),
                            ),
                    );
                }
            }

            content = content.child(table);
        }

        // Control statements table
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
                    .child("Kontrolni hlaseni"),
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
                    .child(div().w(px(80.0)).child("Obdobi"))
                    .child(div().flex_1().child("Typ"))
                    .child(div().w_20().text_right().child("Stav")),
            );

            if self.control_statements.is_empty() {
                table = table.child(
                    div()
                        .px_4()
                        .py_4()
                        .text_sm()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child("Zadna kontrolni hlaseni pro tento rok."),
                );
            } else {
                for cs in &self.control_statements {
                    let period_label = format!("{}/{}", cs.period.month, cs.period.year);
                    let cs_id = cs.id;

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
                            .on_mouse_down(
                                MouseButton::Left,
                                cx.listener(move |_this, _event: &MouseDownEvent, _window, cx| {
                                    cx.emit(NavigateEvent(Route::VATControlDetail(cs_id)));
                                }),
                            )
                            .child(
                                div()
                                    .w(px(80.0))
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                    .child(period_label),
                            )
                            .child(
                                div()
                                    .flex_1()
                                    .text_color(rgb(ZfColors::TEXT_MUTED))
                                    .child(cs.filing_type.to_string()),
                            )
                            .child(
                                div()
                                    .w_20()
                                    .flex()
                                    .justify_end()
                                    .child(self.render_status_badge(&cs.status)),
                            ),
                    );
                }
            }

            content = content.child(table);
        }

        // VIES summaries table
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
                    .child("Souhrnna hlaseni (VIES)"),
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
                    .child(div().w(px(80.0)).child("Obdobi"))
                    .child(div().flex_1().child("Typ"))
                    .child(div().w_20().text_right().child("Stav")),
            );

            if self.vies_summaries.is_empty() {
                table = table.child(
                    div()
                        .px_4()
                        .py_4()
                        .text_sm()
                        .text_color(rgb(ZfColors::TEXT_MUTED))
                        .child("Zadna souhrnna hlaseni pro tento rok."),
                );
            } else {
                for vs in &self.vies_summaries {
                    let period_label = format!("Q{}/{}", vs.period.quarter, vs.period.year);
                    let vs_id = vs.id;

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
                            .on_mouse_down(
                                MouseButton::Left,
                                cx.listener(move |_this, _event: &MouseDownEvent, _window, cx| {
                                    cx.emit(NavigateEvent(Route::VIESDetail(vs_id)));
                                }),
                            )
                            .child(
                                div()
                                    .w(px(80.0))
                                    .font_weight(FontWeight::MEDIUM)
                                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                    .child(period_label),
                            )
                            .child(
                                div()
                                    .flex_1()
                                    .text_color(rgb(ZfColors::TEXT_MUTED))
                                    .child(vs.filing_type.to_string()),
                            )
                            .child(
                                div()
                                    .w_20()
                                    .flex()
                                    .justify_end()
                                    .child(self.render_status_badge(&vs.status)),
                            ),
                    );
                }
            }

            content = content.child(table);
        }

        content
    }
}
