use std::sync::Arc;

use gpui::*;
use zfaktury_core::service::ExpenseService;
use zfaktury_domain::Expense;

use crate::components::button::{ButtonVariant, render_button};
use crate::components::confirm_dialog::{ConfirmDialog, ConfirmDialogResult};
use crate::navigation::{NavigateEvent, Route};
use crate::theme::ZfColors;
use crate::util::format::{format_amount, format_date};

/// Expense detail view displaying all expense data with action buttons.
pub struct ExpenseDetailView {
    service: Arc<ExpenseService>,
    expense_id: i64,
    loading: bool,
    error: Option<String>,
    expense: Option<Expense>,
    confirm_dialog: Option<Entity<ConfirmDialog>>,
    action_loading: bool,
}

impl ExpenseDetailView {
    pub fn new(service: Arc<ExpenseService>, expense_id: i64, cx: &mut Context<Self>) -> Self {
        let mut view = Self {
            service,
            expense_id,
            loading: true,
            error: None,
            expense: None,
            confirm_dialog: None,
            action_loading: false,
        };
        view.load_data(cx);
        view
    }

    fn load_data(&mut self, cx: &mut Context<Self>) {
        let service = self.service.clone();
        let id = self.expense_id;
        cx.spawn(async move |this, cx| {
            let result = cx
                .background_executor()
                .spawn(async move { service.get_by_id(id) })
                .await;

            this.update(cx, |this, cx| {
                this.loading = false;
                match result {
                    Ok(expense) => this.expense = Some(expense),
                    Err(e) => this.error = Some(format!("Chyba pri nacitani nakladu: {e}")),
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
                "Smazat naklad?",
                "Tato akce je nevratna. Naklad bude trvale smazan.",
                "Smazat",
            )
        });
        cx.subscribe(
            &dialog,
            |this: &mut Self, _, result: &ConfirmDialogResult, cx| match result {
                ConfirmDialogResult::Confirmed => {
                    let service = this.service.clone();
                    let id = this.expense_id;
                    this.confirm_dialog = None;
                    this.action_loading = true;
                    this.error = None;
                    cx.notify();
                    cx.spawn(async move |this, cx| {
                        let result = cx
                            .background_executor()
                            .spawn(async move { service.delete(id) })
                            .await;
                        this.update(cx, |this, cx| {
                            this.action_loading = false;
                            match result {
                                Ok(()) => cx.emit(NavigateEvent(Route::ExpenseList)),
                                Err(e) => {
                                    this.error = Some(format!("{e}"));
                                    cx.notify();
                                }
                            }
                        })
                        .ok();
                    })
                    .detach();
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
        let expense_id = self.expense_id;

        // Back button
        bar = bar.child(render_button(
            "btn-back",
            "Zpet na seznam",
            ButtonVariant::Secondary,
            disabled,
            false,
            cx.listener(|_this, _event: &ClickEvent, _window, cx| {
                cx.emit(NavigateEvent(Route::ExpenseList));
            }),
        ));

        // Edit button
        bar = bar.child(render_button(
            "btn-edit",
            "Upravit",
            ButtonVariant::Secondary,
            disabled,
            false,
            cx.listener(move |_this, _event: &ClickEvent, _window, cx| {
                cx.emit(NavigateEvent(Route::ExpenseEdit(expense_id)));
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

    fn render_expense_content(&self, exp: &Expense, cx: &mut Context<Self>) -> Div {
        let vendor_name = exp
            .vendor
            .as_ref()
            .map(|v| v.name.clone())
            .unwrap_or_else(|| {
                exp.vendor_id
                    .map(|id| format!("ID {}", id))
                    .unwrap_or_else(|| "-".to_string())
            });

        let mut content = div().flex().flex_col().gap_6();

        // Header
        content = content.child(
            div()
                .text_xl()
                .font_weight(FontWeight::SEMIBOLD)
                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                .child(format!("Naklad {}", exp.expense_number)),
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

        // Basic info
        content = content.child(self.render_section(
            "Zakladni udaje",
            vec![
                ("Cislo", exp.expense_number.clone()),
                ("Popis", exp.description.clone()),
                ("Kategorie", exp.category.clone()),
                ("Dodavatel", vendor_name),
                ("Datum", format_date(exp.issue_date)),
                ("Castka", format_amount(exp.amount)),
                ("Mena", exp.currency_code.clone()),
                ("Zpusob platby", exp.payment_method.clone()),
                (
                    "Danove uznatelny",
                    if exp.is_tax_deductible { "Ano" } else { "Ne" }.to_string(),
                ),
                ("Obchodni podil", format!("{}%", exp.business_percent)),
            ],
        ));

        // VAT section
        content = content.child(self.render_section(
            "DPH",
            vec![
                ("Sazba DPH", format!("{}%", exp.vat_rate_percent)),
                ("DPH", format_amount(exp.vat_amount)),
            ],
        ));

        // Items table
        if !exp.items.is_empty() {
            let mut items_section = div()
                .p_4()
                .bg(rgb(ZfColors::SURFACE))
                .rounded_md()
                .border_1()
                .border_color(rgb(ZfColors::BORDER))
                .flex()
                .flex_col()
                .gap_3();

            // Title
            items_section = items_section.child(
                div()
                    .text_sm()
                    .font_weight(FontWeight::SEMIBOLD)
                    .text_color(rgb(ZfColors::TEXT_PRIMARY))
                    .child("Polozky"),
            );

            // Column headers
            items_section = items_section.child(
                div()
                    .flex()
                    .gap_2()
                    .pb_2()
                    .border_b_1()
                    .border_color(rgb(ZfColors::BORDER))
                    .child(
                        div()
                            .flex_1()
                            .text_xs()
                            .font_weight(FontWeight::MEDIUM)
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .child("Popis"),
                    )
                    .child(
                        div()
                            .w(px(80.0))
                            .text_xs()
                            .font_weight(FontWeight::MEDIUM)
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .child("Mnozstvi"),
                    )
                    .child(
                        div()
                            .w(px(60.0))
                            .text_xs()
                            .font_weight(FontWeight::MEDIUM)
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .child("Jednotka"),
                    )
                    .child(
                        div()
                            .w(px(100.0))
                            .text_xs()
                            .font_weight(FontWeight::MEDIUM)
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .text_right()
                            .child("Cena/ks"),
                    )
                    .child(
                        div()
                            .w(px(60.0))
                            .text_xs()
                            .font_weight(FontWeight::MEDIUM)
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .text_right()
                            .child("DPH %"),
                    )
                    .child(
                        div()
                            .w(px(100.0))
                            .text_xs()
                            .font_weight(FontWeight::MEDIUM)
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .text_right()
                            .child("DPH"),
                    )
                    .child(
                        div()
                            .w(px(100.0))
                            .text_xs()
                            .font_weight(FontWeight::MEDIUM)
                            .text_color(rgb(ZfColors::TEXT_SECONDARY))
                            .text_right()
                            .child("Celkem"),
                    ),
            );

            // Item rows
            for item in &exp.items {
                let qty_display = format!("{}", item.quantity.to_czk());
                items_section = items_section.child(
                    div()
                        .flex()
                        .gap_2()
                        .py_1()
                        .child(
                            div()
                                .flex_1()
                                .text_sm()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(item.description.clone()),
                        )
                        .child(
                            div()
                                .w(px(80.0))
                                .text_sm()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .child(qty_display),
                        )
                        .child(
                            div()
                                .w(px(60.0))
                                .text_sm()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .child(item.unit.clone()),
                        )
                        .child(
                            div()
                                .w(px(100.0))
                                .text_sm()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .text_right()
                                .child(format_amount(item.unit_price)),
                        )
                        .child(
                            div()
                                .w(px(60.0))
                                .text_sm()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .text_right()
                                .child(format!("{}%", item.vat_rate_percent)),
                        )
                        .child(
                            div()
                                .w(px(100.0))
                                .text_sm()
                                .text_color(rgb(ZfColors::TEXT_SECONDARY))
                                .text_right()
                                .child(format_amount(item.vat_amount)),
                        )
                        .child(
                            div()
                                .w(px(100.0))
                                .text_sm()
                                .text_color(rgb(ZfColors::TEXT_PRIMARY))
                                .text_right()
                                .font_weight(FontWeight::MEDIUM)
                                .child(format_amount(item.total_amount)),
                        ),
                );
            }

            content = content.child(items_section);
        }

        // Notes
        if !exp.notes.is_empty() {
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
                            .child(exp.notes.clone()),
                    ),
            );
        }

        content
    }
}

impl EventEmitter<NavigateEvent> for ExpenseDetailView {}

impl Render for ExpenseDetailView {
    fn render(&mut self, _window: &mut Window, cx: &mut Context<Self>) -> impl IntoElement {
        let mut outer = div()
            .id("expense-detail-scroll")
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
                    .child("Nacitani nakladu..."),
            );
        }

        if self.expense.is_none()
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

        if let Some(ref expense) = self.expense.clone() {
            outer = outer.child(self.render_expense_content(expense, cx));
        }

        // Confirm dialog overlay
        if let Some(ref dialog) = self.confirm_dialog {
            outer = outer.child(dialog.clone());
        }

        outer
    }
}
