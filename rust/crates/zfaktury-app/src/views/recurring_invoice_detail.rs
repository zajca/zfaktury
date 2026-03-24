use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::RecurringInvoiceService;
use zfaktury_domain::{Amount, Frequency, RecurringInvoice};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::confirm_dialog::{ConfirmDialog, ConfirmDialogResult};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;
use crate::util::format::{format_amount, format_date, format_number};

/// Recurring invoice detail view displaying all template data with action buttons.
pub struct RecurringInvoiceDetailView {
    service: Arc<RecurringInvoiceService>,
    template_id: i64,
    loading: bool,
    error: Option<String>,
    template: Option<RecurringInvoice>,
    confirm_dialog: Option<Entity<ConfirmDialog>>,
    action_loading: bool,
}

impl RecurringInvoiceDetailView {
    pub fn new(
        service: Arc<RecurringInvoiceService>,
        template_id: i64,
        cx: &mut Context<Self>,
    ) -> Self {
        let mut view = Self {
            service,
            template_id,
            loading: true,
            error: None,
            template: None,
            confirm_dialog: None,
            action_loading: false,
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        let id = self.template_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.get_by_id(id) })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(template) => this.template = Some(template),
                    Err(e) => this.error = Some(format!("Chyba při načítání šablony faktury: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn handle_generate_invoice(&mut self, cx: &mut Context<Self>) {
        self.action_loading = true;
        self.error = None;
        cx.notify();
        let service = self.service.clone();
        let id = self.template_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.generate_invoice(id) })
                .await;
            this.update(cx, |this, cx| {
                this.action_loading = false;
                match result {
                    Ok(invoice) => cx.emit(NavigateEvent(Route::InvoiceDetail(invoice.id))),
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
        let id = self.template_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.delete(id) })
                .await;
            this.update(cx, |this, cx| {
                this.action_loading = false;
                match result {
                    Ok(()) => cx.emit(NavigateEvent(Route::RecurringInvoiceList)),
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
                "Smazat šablonu faktury?",
                "Tato akce je nevratná. Šablona bude trvale smazána.",
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

    fn render_action_buttons(&self, cx: &mut Context<Self>) -> Div {
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
                cx.emit(NavigateEvent(Route::RecurringInvoiceList));
            }),
        ));

        // Generate invoice button
        bar = bar.child(render_button(
            "btn-generate",
            "Generovat fakturu",
            ButtonVariant::Primary,
            disabled,
            self.action_loading,
            cx.listener(|this, _event: &ClickEvent, _window, cx| {
                this.handle_generate_invoice(cx);
            }),
        ));

        // Delete button
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

        bar
    }

    fn render_field(&self, label: &str, value: String) -> Div {
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
                    .child(value),
            )
    }

    fn frequency_label(freq: &Frequency) -> &'static str {
        match freq {
            Frequency::Weekly => "Týdenní",
            Frequency::Monthly => "Měsíční",
            Frequency::Quarterly => "Čtvrtletní",
            Frequency::Yearly => "Roční",
        }
    }

    fn total_amount(ri: &RecurringInvoice) -> Amount {
        let mut total = Amount::ZERO;
        for item in &ri.items {
            let base = Amount::from_halere(item.quantity.halere() * item.unit_price.halere() / 100);
            let vat = base.multiply(item.vat_rate_percent as f64 / 100.0);
            total += base + vat;
        }
        total
    }

    fn render_template_content(&self, ri: &RecurringInvoice, cx: &mut Context<Self>) -> Div {
        let customer_name = ri
            .customer
            .as_ref()
            .map(|c| c.name.clone())
            .unwrap_or_else(|| format!("ID {}", ri.customer_id));

        let status_text = if ri.is_active {
            "Aktivní"
        } else {
            "Neaktivní"
        };
        let status_color = if ri.is_active {
            ZfColors::STATUS_GREEN
        } else {
            ZfColors::STATUS_GRAY
        };

        let mut content = div().flex().flex_col().gap_6();

        // Header with name and status badge
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
                        .child(format!("Šablona: {}", ri.name)),
                )
                .child(
                    div()
                        .px_3()
                        .py(px(4.0))
                        .rounded_md()
                        .text_sm()
                        .font_weight(FontWeight::MEDIUM)
                        .text_color(rgb(status_color))
                        .child(status_text),
                ),
        );

        // Action buttons
        content = content.child(self.render_action_buttons(cx));

        // Error message (if action failed)
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

        // Info grid
        let end_date_str = ri
            .end_date
            .map(format_date)
            .unwrap_or_else(|| "-".to_string());

        content = content.child(
            div()
                .p_4()
                .bg(rgb(ZfColors::SURFACE))
                .rounded_md()
                .border_1()
                .border_color(rgb(ZfColors::BORDER))
                .flex()
                .flex_col()
                .gap_4()
                .child(
                    div()
                        .flex()
                        .gap_8()
                        .child(self.render_field("Zákazník", customer_name))
                        .child(self.render_field(
                            "Frekvence",
                            Self::frequency_label(&ri.frequency).to_string(),
                        ))
                        .child(self.render_field("Měna", ri.currency_code.clone())),
                )
                .child(
                    div()
                        .flex()
                        .gap_8()
                        .child(
                            self.render_field("Další vystavení", format_date(ri.next_issue_date)),
                        )
                        .child(self.render_field("Konec", end_date_str))
                        .child(self.render_field("Způsob platby", ri.payment_method.clone())),
                )
                .child(
                    div()
                        .flex()
                        .gap_8()
                        .child(self.render_field("Účet", ri.bank_account.clone()))
                        .child(self.render_field("Kód banky", ri.bank_code.clone()))
                        .child(self.render_field("IBAN", ri.iban.clone())),
                ),
        );

        // Items table
        if !ri.items.is_empty() {
            let mut items_table = div()
                .flex()
                .flex_col()
                .bg(rgb(ZfColors::SURFACE))
                .rounded_md()
                .border_1()
                .border_color(rgb(ZfColors::BORDER))
                .overflow_hidden();

            items_table = items_table.child(
                div()
                    .px_4()
                    .py_3()
                    .border_b_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .text_sm()
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child("Položky"),
            );

            // Column headers
            items_table = items_table.child(
                div()
                    .flex()
                    .px_4()
                    .py_2()
                    .text_xs()
                    .text_color(rgb(ZfColors::TEXT_MUTED))
                    .border_b_1()
                    .border_color(rgb(ZfColors::BORDER_SUBTLE))
                    .child(div().flex_1().child("Popis"))
                    .child(div().w_20().text_right().child("Množství"))
                    .child(div().w_16().text_right().child("Jednotka"))
                    .child(div().w_24().text_right().child("Cena/ks"))
                    .child(div().w_16().text_right().child("DPH %")),
            );

            for item in &ri.items {
                items_table = items_table.child(
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
                                .child(item.description.clone()),
                        )
                        .child(
                            div()
                                .w_20()
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(format_number(item.quantity)),
                        )
                        .child(
                            div()
                                .w_16()
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(item.unit.clone()),
                        )
                        .child(
                            div()
                                .w_24()
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(format_amount(item.unit_price)),
                        )
                        .child(
                            div()
                                .w_16()
                                .text_right()
                                .text_color(rgb(ZfColors::TEXT_MUTED))
                                .child(format!("{}%", item.vat_rate_percent)),
                        ),
                );
            }

            content = content.child(items_table);
        }

        // Total
        let total = Self::total_amount(ri);
        content = content.child(
            div()
                .p_4()
                .bg(rgb(ZfColors::SURFACE))
                .rounded_md()
                .border_1()
                .border_color(rgb(ZfColors::BORDER))
                .flex()
                .justify_between()
                .child(
                    div()
                        .text_sm()
                        .font_weight(FontWeight::SEMIBOLD)
                        .text_color(rgb(ZfColors::TEXT_PRIMARY))
                        .child("Odhadovaná celková částka"),
                )
                .child(
                    div()
                        .text_lg()
                        .font_weight(FontWeight::BOLD)
                        .text_color(rgb(ZfColors::ACCENT))
                        .child(format_amount(total)),
                ),
        );

        // Notes
        if !ri.notes.is_empty() {
            content = content.child(
                div()
                    .p_4()
                    .bg(rgb(ZfColors::SURFACE))
                    .rounded_md()
                    .border_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .flex()
                    .flex_col()
                    .gap_1()
                    .child(
                        div()
                            .text_xs()
                            .text_color(rgb(ZfColors::TEXT_MUTED))
                            .child("Poznámky"),
                    )
                    .child(
                        div()
                            .text_sm()
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .child(ri.notes.clone()),
                    ),
            );
        }

        content
    }
}

impl EventEmitter<NavigateEvent> for RecurringInvoiceDetailView {}

impl Render for RecurringInvoiceDetailView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("recurring-invoice-detail-scroll")
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
                    .child("Načítání šablony faktury..."),
            );
        }

        if self.template.is_none()
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

        if let Some(ref ri) = self.template.clone() {
            outer = outer.child(self.render_template_content(ri, cx));
        }

        // Confirm dialog overlay
        if let Some(ref dialog) = self.confirm_dialog {
            outer = outer.child(dialog.clone());
        }

        outer
    }
}
