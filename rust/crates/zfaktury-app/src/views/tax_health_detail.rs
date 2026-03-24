use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::HealthInsuranceService;
use zfaktury_domain::{FilingStatus, HealthInsuranceOverview};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::confirm_dialog::{ConfirmDialog, ConfirmDialogResult};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;
use crate::util::format::format_amount;

/// Health insurance overview detail view.
pub struct TaxHealthDetailView {
    service: Arc<HealthInsuranceService>,
    overview_id: i64,
    loading: bool,
    error: Option<String>,
    overview: Option<HealthInsuranceOverview>,
    confirm_dialog: Option<Entity<ConfirmDialog>>,
    action_loading: bool,
}

impl TaxHealthDetailView {
    pub fn new(
        service: Arc<HealthInsuranceService>,
        overview_id: i64,
        cx: &mut Context<Self>,
    ) -> Self {
        let mut view = Self {
            service,
            overview_id,
            loading: true,
            error: None,
            overview: None,
            confirm_dialog: None,
            action_loading: false,
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        let id = self.overview_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.get_by_id(id) })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(hi) => this.overview = Some(hi),
                    Err(e) => {
                        this.error = Some(format!("Chyba při načítání přehledu ZP: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn handle_mark_filed(&mut self, cx: &mut Context<Self>) {
        self.action_loading = true;
        self.error = None;
        cx.notify();
        let service = self.service.clone();
        let id = self.overview_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.mark_filed(id) })
                .await;
            this.update(cx, |this, cx| {
                this.action_loading = false;
                match result {
                    Ok(_hi) => this.load_data(cx),
                    Err(e) => this.error = Some(format!("{e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn handle_delete_confirmed(&mut self, cx: &mut Context<Self>) {
        self.confirm_dialog = None;
        self.action_loading = true;
        self.error = None;
        cx.notify();
        let service = self.service.clone();
        let id = self.overview_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.delete(id) })
                .await;
            this.update(cx, |this, cx| {
                this.action_loading = false;
                match result {
                    Ok(()) => cx.emit(NavigateEvent(Route::TaxOverview)),
                    Err(e) => this.error = Some(format!("{e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn show_delete_dialog(&mut self, cx: &mut Context<Self>) {
        let dialog = cx.new(|_cx| {
            ConfirmDialog::new(
                "Smazat přehled ZP?",
                "Tato akce je nevratná. Přehled bude trvale smazán.",
                "Smazat",
            )
        });
        cx.subscribe(
            &dialog,
            |this: &mut Self, _, result: &ConfirmDialogResult, cx| match result {
                ConfirmDialogResult::Confirmed => {
                    this.handle_delete_confirmed(cx);
                }
                ConfirmDialogResult::Cancelled => {
                    this.confirm_dialog = None;
                    cx.notify();
                }
            },
        )
        .detach();
        self.confirm_dialog = Some(dialog);
        cx.notify();
    }

    fn render_action_buttons(&self, hi: &HealthInsuranceOverview, cx: &mut Context<Self>) -> Div {
        let mut bar = div().flex().items_center().gap_2().flex_wrap();
        let disabled = self.action_loading;

        bar = bar.child(render_button(
            "btn-back",
            "Zpět",
            ButtonVariant::Secondary,
            disabled,
            false,
            cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                cx.emit(NavigateEvent(Route::TaxOverview));
            }),
        ));

        if hi.status != FilingStatus::Filed {
            bar = bar.child(render_button(
                "btn-mark-filed",
                "Označit jako podané",
                ButtonVariant::Primary,
                disabled,
                self.action_loading,
                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                    this.handle_mark_filed(cx);
                }),
            ));

            bar = bar.child(render_button(
                "btn-delete",
                "Smazat",
                ButtonVariant::Danger,
                disabled,
                false,
                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                    this.show_delete_dialog(cx);
                }),
            ));
        }

        bar
    }

    fn render_amount_row(&self, label: &str, value: zfaktury_domain::Amount) -> Div {
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
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child(format_amount(value)),
            )
    }

    fn render_content(&self, hi: &HealthInsuranceOverview, cx: &mut Context<Self>) -> Div {
        let status_text = hi.status.to_string();
        let status_color = match hi.status {
            FilingStatus::Draft => ZfColors::STATUS_GRAY,
            FilingStatus::Ready => ZfColors::STATUS_YELLOW,
            FilingStatus::Filed => ZfColors::STATUS_GREEN,
        };

        // Insurance rate display: stored as permille*10, e.g. 135 = 13.5%
        let rate_display = format!("{},{} %", hi.insurance_rate / 10, hi.insurance_rate % 10);

        let mut content = div().flex().flex_col().gap_6();

        // Header
        content = content.child(
            div().flex().items_center().justify_between().child(
                div()
                    .flex()
                    .items_center()
                    .gap_3()
                    .child(
                        div()
                            .text_xl()
                            .font_weight(FontWeight::SEMIBOLD)
                            .text_color(rgb(ZfColors::TEXT_PRIMARY))
                            .child(format!("Zdravotní pojištění {}", hi.year)),
                    )
                    .child(
                        div()
                            .px_2()
                            .py(px(2.0))
                            .rounded(px(4.0))
                            .text_xs()
                            .text_color(rgb(status_color))
                            .child(status_text),
                    ),
            ),
        );

        // Action buttons
        content = content.child(self.render_action_buttons(hi, cx));

        // Error
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
        }

        // Income
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
                        .child("Příjmy a výdaje"),
                )
                .child(self.render_amount_row("Celkové příjmy", hi.total_revenue))
                .child(self.render_amount_row("Celkové výdaje", hi.total_expenses))
                .child(self.render_amount_row("Základ daně", hi.tax_base)),
        );

        // Assessment
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
                        .child("Vyměřovací základ"),
                )
                .child(self.render_amount_row("Vyměřovací základ", hi.assessment_base))
                .child(self.render_amount_row("Minimální vym. základ", hi.min_assessment_base))
                .child(self.render_amount_row("Konečný vym. základ", hi.final_assessment_base))
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .text_sm()
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child("Sazba pojistného"),
                        )
                        .child(
                            div()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(rate_display),
                        ),
                ),
        );

        // Result
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
                        .child("Výsledek"),
                )
                .child(self.render_amount_row("Pojistné celkem", hi.total_insurance))
                .child(self.render_amount_row("Zaplacené zálohy", hi.prepayments))
                .child(div().h(px(1.0)).bg(rgb(ZfColors::BORDER)))
                .child(
                    div()
                        .flex()
                        .justify_between()
                        .child(
                            div()
                                .text_sm()
                                .font_weight(FontWeight::SEMIBOLD)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child("Doplatek / Přeplatek"),
                        )
                        .child(
                            div()
                                .text_lg()
                                .font_weight(FontWeight::BOLD)
                                .text_color(rgb(ZfColors::ACCENT))
                                .child(format_amount(hi.difference)),
                        ),
                )
                .child(self.render_amount_row("Nová měsíční záloha", hi.new_monthly_prepay)),
        );

        content
    }
}

impl EventEmitter<NavigateEvent> for TaxHealthDetailView {}

impl Render for TaxHealthDetailView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("tax-health-detail-scroll")
            .size_full()
            .bg(rgb(ZfColors::BG))
            .p_6()
            .overflow_y_scroll()
            .relative();

        if self.loading {
            return outer.child(
                div()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Načítání přehledu ZP..."),
            );
        }

        if self.overview.is_none()
            && let Some(ref error) = self.error
        {
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

        if let Some(ref hi) = self.overview.clone() {
            outer = outer.child(self.render_content(hi, cx));
        }

        if let Some(ref dialog) = self.confirm_dialog {
            outer = outer.child(dialog.clone());
        }

        outer
    }
}
