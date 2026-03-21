use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::RecurringExpenseService;
use zfaktury_domain::{Frequency, RecurringExpense};

use crate::components::button::{ButtonVariant, render_button};
use crate::components::confirm_dialog::{ConfirmDialog, ConfirmDialogResult};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;
use crate::util::format::{format_amount, format_date};

/// Recurring expense detail view displaying all template data with action buttons.
pub struct RecurringExpenseDetailView {
    service: Arc<RecurringExpenseService>,
    template_id: i64,
    loading: bool,
    error: Option<String>,
    template: Option<RecurringExpense>,
    confirm_dialog: Option<Entity<ConfirmDialog>>,
    action_loading: bool,
}

impl RecurringExpenseDetailView {
    pub fn new(
        service: Arc<RecurringExpenseService>,
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
                    Err(e) => this.error = Some(format!("Chyba pri nacitani opak. nakladu: {e}")),
                }
                cx.notify();
            })
            .ok();
        })
        .detach();
    }

    fn handle_activate(&mut self, cx: &mut Context<Self>) {
        self.action_loading = true;
        self.error = None;
        cx.notify();
        let service = self.service.clone();
        let id = self.template_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.activate(id) })
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

    fn handle_deactivate(&mut self, cx: &mut Context<Self>) {
        self.action_loading = true;
        self.error = None;
        cx.notify();
        let service = self.service.clone();
        let id = self.template_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.deactivate(id) })
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
        let id = self.template_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.delete(id) })
                .await;
            this.update(cx, |this, cx| {
                this.action_loading = false;
                match result {
                    Ok(()) => cx.emit(NavigateEvent(Route::RecurringExpenseList)),
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
                "Smazat opakovany naklad?",
                "Tato akce je nevratna. Sablona bude trvale smazana.",
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

    fn render_action_buttons(&self, re: &RecurringExpense, cx: &mut Context<Self>) -> Div {
        let mut bar = div().flex().items_center().gap_2().flex_wrap();
        let disabled = self.action_loading;

        // Back button
        bar = bar.child(render_button(
            "btn-back",
            "Zpet",
            ButtonVariant::Secondary,
            disabled,
            false,
            cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                cx.emit(NavigateEvent(Route::RecurringExpenseList));
            }),
        ));

        // Activate / Deactivate toggle
        if re.is_active {
            bar = bar.child(render_button(
                "btn-deactivate",
                "Deaktivovat",
                ButtonVariant::Secondary,
                disabled,
                self.action_loading,
                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                    this.handle_deactivate(cx);
                }),
            ));
        } else {
            bar = bar.child(render_button(
                "btn-activate",
                "Aktivovat",
                ButtonVariant::Primary,
                disabled,
                self.action_loading,
                cx.listener(|this, _event: &ClickEvent, _window, cx| {
                    this.handle_activate(cx);
                }),
            ));
        }

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
            .items_center()
            .gap_4()
            .child(
                div()
                    .w_40()
                    .text_sm()
                    .text_color(rgb(ZfColors::TEXT_SECONDARY))
                    .child(label.to_string()),
            )
            .child(
                div()
                    .flex_1()
                    .text_sm()
                    .text_color(if value.is_empty() {
                        rgb(ZfColors::TEXT_MUTED)
                    } else {
                        rgb(ZfColors::TEXT_PRIMARY)
                    })
                    .child(if value.is_empty() {
                        "-".to_string()
                    } else {
                        value
                    }),
            )
    }

    fn render_section(&self, title: &str, fields: Vec<(&str, String)>) -> Div {
        let mut section = div()
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
                    .child(title.to_string()),
            );

        for (label, value) in fields {
            section = section.child(self.render_field(label, value));
        }

        section
    }

    fn frequency_label(freq: &Frequency) -> &'static str {
        match freq {
            Frequency::Weekly => "Tydenni",
            Frequency::Monthly => "Mesicni",
            Frequency::Quarterly => "Ctvrtletni",
            Frequency::Yearly => "Rocni",
        }
    }

    fn render_template_content(&self, re: &RecurringExpense, cx: &mut Context<Self>) -> Div {
        let vendor_name = re
            .vendor
            .as_ref()
            .map(|v| v.name.clone())
            .unwrap_or_else(|| {
                re.vendor_id
                    .map(|id| format!("ID {}", id))
                    .unwrap_or_else(|| "-".to_string())
            });

        let status_text = if re.is_active { "Aktivni" } else { "Neaktivni" };
        let status_color = if re.is_active {
            ZfColors::STATUS_GREEN
        } else {
            ZfColors::STATUS_GRAY
        };

        let end_date_str = re
            .end_date
            .map(format_date)
            .unwrap_or_else(|| "-".to_string());

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
                        .child(format!("Opakovany naklad: {}", re.name)),
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
        content = content.child(self.render_action_buttons(re, cx));

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

        // Basic info
        content = content.child(self.render_section(
            "Zakladni udaje",
            vec![
                ("Popis", re.description.clone()),
                ("Kategorie", re.category.clone()),
                ("Dodavatel", vendor_name),
                ("Castka", format_amount(re.amount)),
                ("Mena", re.currency_code.clone()),
                ("Zpusob platby", re.payment_method.clone()),
                (
                    "Danove uznatelny",
                    if re.is_tax_deductible { "Ano" } else { "Ne" }.to_string(),
                ),
                ("Obchodni podil", format!("{}%", re.business_percent)),
            ],
        ));

        // VAT section
        content = content.child(self.render_section(
            "DPH",
            vec![
                ("Sazba DPH", format!("{}%", re.vat_rate_percent)),
                ("DPH", format_amount(re.vat_amount)),
            ],
        ));

        // Schedule section
        content = content.child(self.render_section(
            "Opakovani",
            vec![
                (
                    "Frekvence",
                    Self::frequency_label(&re.frequency).to_string(),
                ),
                ("Dalsi vystaveni", format_date(re.next_issue_date)),
                ("Konec", end_date_str),
            ],
        ));

        // Notes
        if !re.notes.is_empty() {
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
                            .child("Poznamky"),
                    )
                    .child(
                        div()
                            .text_sm()
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .child(re.notes.clone()),
                    ),
            );
        }

        content
    }
}

impl EventEmitter<NavigateEvent> for RecurringExpenseDetailView {}

impl Render for RecurringExpenseDetailView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("recurring-expense-detail-scroll")
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
                    .child("Nacitani opak. nakladu..."),
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

        if let Some(ref re) = self.template.clone() {
            outer = outer.child(self.render_template_content(re, cx));
        }

        // Confirm dialog overlay
        if let Some(ref dialog) = self.confirm_dialog {
            outer = outer.child(dialog.clone());
        }

        outer
    }
}
