use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::VATControlStatementService;
use zfaktury_domain::{FilingStatus, VATControlStatement, VATControlStatementLine};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::confirm_dialog::{ConfirmDialog, ConfirmDialogResult};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;
use crate::util::format::format_amount;

/// VAT control statement detail view.
pub struct VatControlDetailView {
    service: Arc<VATControlStatementService>,
    statement_id: i64,
    loading: bool,
    error: Option<String>,
    statement: Option<VATControlStatement>,
    lines: Vec<VATControlStatementLine>,
    confirm_dialog: Option<Entity<ConfirmDialog>>,
    action_loading: bool,
}

impl VatControlDetailView {
    pub fn new(
        service: Arc<VATControlStatementService>,
        statement_id: i64,
        cx: &mut Context<Self>,
    ) -> Self {
        let mut view = Self {
            service,
            statement_id,
            loading: true,
            error: None,
            statement: None,
            lines: Vec::new(),
            confirm_dialog: None,
            action_loading: false,
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        let id = self.statement_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move {
                    let cs = service.get_by_id(id)?;
                    let lines = service.get_lines(id)?;
                    Ok::<
                        (VATControlStatement, Vec<VATControlStatementLine>),
                        zfaktury_domain::DomainError,
                    >((cs, lines))
                })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok((cs, lines)) => {
                        this.statement = Some(cs);
                        this.lines = lines;
                    }
                    Err(e) => {
                        this.error = Some(format!("Chyba při načítání kontrolního hlášení: {e}"));
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
        let id = self.statement_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.mark_filed(id) })
                .await;
            this.update(cx, |this, cx| {
                this.action_loading = false;
                match result {
                    Ok(()) => this.load_data(cx),
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
        let id = self.statement_id;
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
                "Smazat kontrolní hlášení?",
                "Tato akce je nevratná. Kontrolní hlášení bude trvale smazáno.",
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

    fn render_action_buttons(&self, cs: &VATControlStatement, cx: &mut Context<Self>) -> Div {
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

        if cs.status != FilingStatus::Filed {
            // Mark filed
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

    fn render_content(&self, cs: &VATControlStatement, cx: &mut Context<Self>) -> Div {
        let period_label = format!("{}/{}", cs.period.month, cs.period.year);
        let status_text = cs.status.to_string();
        let status_color = match cs.status {
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
                            .child(format!("Kontrolní hlášení {}", period_label)),
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
        content = content.child(self.render_action_buttons(cs, cx));

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
                                .child(cs.filing_type.to_string()),
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
                                .child("Počet řádků"),
                        )
                        .child(
                            div()
                                .text_sm()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(self.lines.len().to_string()),
                        ),
                ),
        );

        // Lines table
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
                .child("Řádky kontrolního hlášení"),
        );

        // Column headers
        table = table.child(
            div()
                .flex()
                .px_4()
                .py_2()
                .text_xs()
                .text_color(rgb(ZfColors::TEXT_MUTED))
                .border_b_1()
                .border_color(rgb(ZfColors::BORDER_SUBTLE))
                .child(div().w(px(60.0)).child("Sekce"))
                .child(div().w(px(120.0)).child("DIC partnera"))
                .child(div().flex_1().child("Číslo dokladu"))
                .child(div().w(px(100.0)).child("DPPD"))
                .child(div().w(px(112.0)).text_right().child("Základ"))
                .child(div().w(px(112.0)).text_right().child("DPH"))
                .child(div().w(px(60.0)).text_right().child("Sazba")),
        );

        if self.lines.is_empty() {
            table = table.child(
                div()
                    .px_4()
                    .py_8()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .child("Žádné řádky."),
            );
        } else {
            for line in &self.lines {
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
                                .w(px(60.0))
                                .font_weight(FontWeight::MEDIUM)
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(line.section.to_string()),
                        )
                        .child(
                            div()
                                .w(px(120.0))
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(line.partner_dic.clone()),
                        )
                        .child(
                            div()
                                .flex_1()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(line.document_number.clone()),
                        )
                        .child(
                            div()
                                .w(px(100.0))
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(line.dppd.clone()),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(format_amount(line.base)),
                        )
                        .child(
                            div()
                                .w(px(112.0))
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(format_amount(line.vat)),
                        )
                        .child(
                            div()
                                .w(px(60.0))
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format!("{}%", line.vat_rate_percent)),
                        ),
                );
            }
        }

        content.child(table)
    }
}

impl EventEmitter<NavigateEvent> for VatControlDetailView {}

impl Render for VatControlDetailView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("vat-control-detail-scroll")
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
                    .child("Načítání kontrolního hlášení..."),
            );
        }

        if self.statement.is_none()
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

        if let Some(ref cs) = self.statement.clone() {
            outer = outer.child(self.render_content(cs, cx));
        }

        // Confirm dialog overlay
        if let Some(ref dialog) = self.confirm_dialog {
            outer = outer.child(dialog.clone());
        }

        outer
    }
}
