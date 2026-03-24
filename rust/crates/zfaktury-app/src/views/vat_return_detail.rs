use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::VATReturnService;
use zfaktury_domain::{FilingStatus, VATReturn};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::confirm_dialog::{ConfirmDialog, ConfirmDialogResult};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;
use crate::util::format::format_amount;

/// VAT return detail view.
pub struct VatReturnDetailView {
    service: Arc<VATReturnService>,
    return_id: i64,
    loading: bool,
    error: Option<String>,
    vat_return: Option<VATReturn>,
    confirm_dialog: Option<Entity<ConfirmDialog>>,
    action_loading: bool,
}

impl VatReturnDetailView {
    pub fn new(service: Arc<VATReturnService>, return_id: i64, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            service,
            return_id,
            loading: true,
            error: None,
            vat_return: None,
            confirm_dialog: None,
            action_loading: false,
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        let id = self.return_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.get_by_id(id) })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(vr) => this.vat_return = Some(vr),
                    Err(e) => {
                        this.error = Some(format!("Chyba při načítání DPH přiznání: {e}"));
                    }
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn handle_recalculate(&mut self, cx: &mut Context<Self>) {
        self.action_loading = true;
        self.error = None;
        cx.notify();
        let service = self.service.clone();
        let id = self.return_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.recalculate(id) })
                .await;
            this.update(cx, |this, cx| {
                this.action_loading = false;
                match result {
                    Ok(vr) => this.vat_return = Some(vr),
                    Err(e) => this.error = Some(format!("{e}")),
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
        let id = self.return_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.mark_filed(id) })
                .await;
            this.update(cx, |this, cx| {
                this.action_loading = false;
                match result {
                    Ok(vr) => this.vat_return = Some(vr),
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
        let id = self.return_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.delete(id) })
                .await;
            this.update(cx, |this, cx| {
                this.action_loading = false;
                match result {
                    Ok(()) => cx.emit(NavigateEvent(Route::VATOverview)),
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
                "Smazat DPH přiznání?",
                "Tato akce je nevratná. DPH přiznání bude trvale smazáno.",
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

    fn render_action_buttons(&self, vr: &VATReturn, cx: &mut Context<Self>) -> Div {
        let mut bar = div().flex().items_center().gap_2().flex_wrap();
        let disabled = self.action_loading;

        // Back button
        bar = bar.child(render_button(
            "btn-back",
            "Zpět",
            ButtonVariant::Secondary,
            disabled,
            false,
            cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                cx.emit(NavigateEvent(Route::VATOverview));
            }),
        ));

        if vr.status != FilingStatus::Filed {
            // Recalculate
            bar = bar.child(render_button(
                "btn-recalculate",
                "Přepočítat",
                ButtonVariant::Primary,
                disabled,
                self.action_loading,
                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                    this.handle_recalculate(cx);
                }),
            ));

            // Mark filed
            bar = bar.child(render_button(
                "btn-mark-filed",
                "Označit jako podané",
                ButtonVariant::Secondary,
                disabled,
                false,
                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                    this.handle_mark_filed(cx);
                }),
            ));

            // Delete
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

    fn render_vat_content(&self, vr: &VATReturn, cx: &mut Context<Self>) -> Div {
        let period_label = if vr.period.month > 0 {
            format!("{}/{}", vr.period.month, vr.period.year)
        } else {
            format!("Q{}/{}", vr.period.quarter, vr.period.year)
        };

        let status_text = vr.status.to_string();
        let status_color = match vr.status {
            FilingStatus::Draft => ZfColors::STATUS_GRAY,
            FilingStatus::Ready => ZfColors::STATUS_YELLOW,
            FilingStatus::Filed => ZfColors::STATUS_GREEN,
        };

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
                            .child(format!("DPH přiznání {}", period_label)),
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

        // Action buttons (wired)
        content = content.child(self.render_action_buttons(vr, cx));

        // Error message
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

        // Info row
        content = content.child(
            div()
                .p_4()
                .bg(rgb(ZfColors::SURFACE))
                .rounded_md()
                .border_1()
                .border_color(rgb(ZfColors::BORDER))
                .flex()
                .gap_8()
                .child(
                    div()
                        .flex()
                        .flex_col()
                        .gap(px(2.0))
                        .child(
                            div()
                                .text_xs()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child("Období"),
                        )
                        .child(
                            div()
                                .text_sm()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(period_label.clone()),
                        ),
                )
                .child(
                    div()
                        .flex()
                        .flex_col()
                        .gap(px(2.0))
                        .child(
                            div()
                                .text_xs()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child("Typ podání"),
                        )
                        .child(
                            div()
                                .text_sm()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(vr.filing_type.to_string()),
                        ),
                ),
        );

        // Output VAT section
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
                        .child("Výstupní DPH (daň na výstupu)"),
                )
                .child(self.render_amount_row("Základ 21%", vr.output_vat_base_21))
                .child(self.render_amount_row("DPH 21%", vr.output_vat_amount_21))
                .child(self.render_amount_row("Základ 12%", vr.output_vat_base_12))
                .child(self.render_amount_row("DPH 12%", vr.output_vat_amount_12))
                .child(self.render_amount_row("Základ 0%", vr.output_vat_base_0))
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
                                .child("Celkem výstupní DPH"),
                        )
                        .child(
                            div()
                                .text_sm()
                                .font_weight(FontWeight::BOLD)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(vr.total_output_vat)),
                        ),
                ),
        );

        // Input VAT section
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
                        .child("Vstupní DPH (daň na vstupu)"),
                )
                .child(self.render_amount_row("Základ 21%", vr.input_vat_base_21))
                .child(self.render_amount_row("DPH 21%", vr.input_vat_amount_21))
                .child(self.render_amount_row("Základ 12%", vr.input_vat_base_12))
                .child(self.render_amount_row("DPH 12%", vr.input_vat_amount_12))
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
                                .child("Celkem vstupní DPH"),
                        )
                        .child(
                            div()
                                .text_sm()
                                .font_weight(FontWeight::BOLD)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(format_amount(vr.total_input_vat)),
                        ),
                ),
        );

        // Summary
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
                        .child("Souhrn"),
                )
                .child(self.render_amount_row("Výstupní DPH", vr.total_output_vat))
                .child(self.render_amount_row("Vstupní DPH", vr.total_input_vat))
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
                                .child("Výsledná daňová povinnost"),
                        )
                        .child(
                            div()
                                .text_lg()
                                .font_weight(FontWeight::BOLD)
                                .text_color(rgb(ZfColors::ACCENT))
                                .child(format_amount(vr.net_vat)),
                        ),
                ),
        );

        content
    }
}

impl EventEmitter<NavigateEvent> for VatReturnDetailView {}

impl Render for VatReturnDetailView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("vat-return-detail-scroll")
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
                    .child("Načítání DPH přiznání..."),
            );
        }

        if self.vat_return.is_none()
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

        if let Some(ref vr) = self.vat_return.clone() {
            outer = outer.child(self.render_vat_content(vr, cx));
        }

        // Confirm dialog overlay
        if let Some(ref dialog) = self.confirm_dialog {
            outer = outer.child(dialog.clone());
        }

        outer
    }
}
